import {
  Injectable,
  Inject,
  Logger,
  BadRequestException,
  NotFoundException,
  ServiceUnavailableException,
  UnauthorizedException,
} from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';
import { BillingService } from './billing.service';
import * as crypto from 'crypto';

const NOWPAYMENTS_API_BASE = 'https://api.nowpayments.io/v1';
const SETTLED_STATUSES = new Set(['finished', 'confirmed']);

interface NowPaymentsInvoiceResponse {
  id?: string | number;
  invoice_url?: string;
}

/**
 * NOWPayments USDT collection (control-plane only; never in the Go hot path).
 *
 * Security:
 * - create-invoice fails closed with 503 payment_gateway_not_configured when
 *   NOWPAYMENTS_API_KEY is empty.
 * - IPN callbacks are authenticated by HMAC-SHA512 over the NOWPayments-documented
 *   canonical string (keys sorted alphabetically, `key:value` pairs joined by `|`)
 *   compared timing-safe against NOWPAYMENTS_IPN_SECRET. Missing secret or any
 *   mismatch rejects the delivery before any state mutation.
 * Financial integrity:
 * - Idempotent per (provider, provider_event_id) via billing.payment_events.
 * - Settlement mutates payment/invoice/subscription inside one DB transaction.
 */
@Injectable()
export class NowPaymentsService {
  private logger = new Logger(NowPaymentsService.name);
  constructor(
    @Inject(DB_POOL) private pool: Pool,
    private billingService: BillingService,
  ) {}

  async createInvoice(
    userId: string,
    planId: string,
    billingInterval: 'MONTHLY' | 'ANNUAL' = 'MONTHLY',
  ): Promise<{ payment_url: string; invoice_id: string }> {
    const apiKey = (process.env.NOWPAYMENTS_API_KEY || '').trim();
    if (!apiKey) {
      this.logger.warn('USDT checkout rejected: NOWPAYMENTS_API_KEY not configured');
      throw new ServiceUnavailableException('payment_gateway_not_configured');
    }

    const interval = String(billingInterval || 'MONTHLY').toUpperCase();
    if (!['MONTHLY', 'ANNUAL'].includes(interval)) {
      throw new BadRequestException('billing_interval must be MONTHLY or ANNUAL');
    }

    const plan = await this.pool.query(
      `SELECT id, name, code, monthly_price, annual_price, allowed_strategies, currency
       FROM control.plans WHERE id = $1 AND status = 'ACTIVE'`,
      [planId],
    );
    if (!plan.rows[0]) throw new NotFoundException('Active plan not found');
    const p = plan.rows[0];
    if (interval === 'ANNUAL' && p.annual_price === null) {
      throw new BadRequestException('Annual billing is not available for this plan');
    }
    const price = Number(interval === 'ANNUAL' ? p.annual_price : p.monthly_price);
    if (!Number.isFinite(price) || price <= 0) {
      throw new BadRequestException('Plan does not require payment');
    }

    // Reuse an existing pending subscription for this plan/interval; otherwise
    // create one (mirrors SubscriptionsService.create semantics for paid plans).
    let sub = await this.pool.query(
      `SELECT s.id FROM billing.subscriptions s
       WHERE s.user_id = $1 AND s.plan_id = $2 AND s.billing_interval = $3 AND s.status = 'INCOMPLETE'
       ORDER BY s.created_at DESC LIMIT 1`,
      [userId, planId, interval],
    );
    let subscriptionId = sub.rows[0]?.id as string | undefined;
    if (!subscriptionId) {
      subscriptionId = crypto.randomUUID();
      await this.pool.query(
        `INSERT INTO billing.subscriptions
         (id, user_id, plan_id, status, billing_interval, billing_period_start, billing_period_end, selected_strategies)
         VALUES ($1,$2,$3,'INCOMPLETE',$4,now(),
                 now() + CASE WHEN $4 = 'ANNUAL' THEN interval '1 year' ELSE interval '1 month' END,
                 $5::jsonb)`,
        [subscriptionId, userId, planId, interval, JSON.stringify(p.allowed_strategies || [])],
      );
    }

    // Branded invoice for the period; idempotent per (subscription, period).
    const invoiceId = await this.billingService.generateInvoiceForSubscription(subscriptionId, userId);

    const orderId = `sub:${subscriptionId}:${crypto.randomUUID()}`;
    const successUrl = process.env.NOWPAYMENTS_SUCCESS_URL || 'https://platform.predictatrade.com/dashboard/billing?payment=success';
    const cancelUrl = process.env.NOWPAYMENTS_CANCEL_URL || 'https://platform.predictatrade.com/dashboard/billing?payment=cancelled';
    const ipnCallbackUrl =
      process.env.NOWPAYMENTS_IPN_CALLBACK_URL ||
      'https://api.predictatrade.com/api/v1/billing/webhook/nowpayments';

    let resp: Response;
    try {
      resp = await fetch(`${NOWPAYMENTS_API_BASE}/invoice`, {
        method: 'POST',
        headers: { 'x-api-key': apiKey, 'Content-Type': 'application/json' },
        body: JSON.stringify({
          price_amount: price,
          price_currency: 'usd',
          pay_currency: 'usdt',
          order_id: orderId,
          order_description: `${p.name} subscription (${interval})`,
          success_url: successUrl,
          cancel_url: cancelUrl,
          ipn_callback_url: ipnCallbackUrl,
        }),
      });
    } catch (e) {
      this.logger.error(`NOWPayments unreachable: ${e instanceof Error ? e.message : e}`);
      throw new ServiceUnavailableException('payment_gateway_unreachable');
    }
    if (!resp.ok) {
      this.logger.error(`NOWPayments create-invoice failed: HTTP ${resp.status}`);
      throw new ServiceUnavailableException('payment_gateway_error');
    }
    const data = (await resp.json()) as NowPaymentsInvoiceResponse;
    if (!data?.id || !data?.invoice_url) {
      this.logger.error('NOWPayments create-invoice returned an unexpected payload');
      throw new ServiceUnavailableException('payment_gateway_error');
    }

    const providerInvoiceId = String(data.id);
    await this.pool.query(
      `INSERT INTO billing.payments
       (user_id, subscription_id, invoice_id, provider, provider_payment_id, provider_event_id,
        amount, currency, payment_type, status)
       VALUES ($1,$2,$3,'nowpayments',$4,$4,$5,'USD','SUBSCRIPTION','PENDING')
       ON CONFLICT (provider, provider_payment_id) WHERE provider_payment_id IS NOT NULL
       DO NOTHING`,
      [userId, subscriptionId, invoiceId, providerInvoiceId, price],
    );
    await this.pool.query(
      `UPDATE billing.invoices SET provider_invoice_id = $2, provider_hosted_url = $3, updated_at = now()
       WHERE id = $1`,
      [invoiceId, providerInvoiceId, data.invoice_url],
    );

    return { payment_url: data.invoice_url, invoice_id: invoiceId };
  }

  /**
   * NOWPayments IPN handler. Signature IS the auth — endpoint must stay public.
   * NOWPayments signs the RAW request body with HMAC-SHA512 using the IPN
   * secret; the hex digest is sent in the `x-nowpayments-sig` header. We verify
   * over the raw body bytes (NOT a re-serialized/canonicalized object) so the
   * signature matches exactly what the gateway computed.
   */
  async handleIPN(rawBody: string, signatureHeader?: string | string[]) {
    const secret = (process.env.NOWPAYMENTS_IPN_SECRET || '').trim();
    if (!secret) {
      this.logger.error('IPN rejected: NOWPAYMENTS_IPN_SECRET not configured');
      throw new UnauthorizedException('ipn_not_configured');
    }
    const provided = Array.isArray(signatureHeader) ? signatureHeader[0] : signatureHeader;
    if (!provided || typeof rawBody !== 'string' || rawBody.length === 0) {
      this.logger.warn('IPN rejected: missing signature or payload');
      throw new UnauthorizedException('invalid_ipn_signature');
    }

    const expected = crypto.createHmac('sha512', secret).update(rawBody, 'utf8').digest('hex');
    const a = Buffer.from(String(provided), 'utf8');
    const b = Buffer.from(expected, 'utf8');
    if (a.length !== b.length || !crypto.timingSafeEqual(a, b)) {
      this.logger.warn('IPN rejected: HMAC-SHA512 signature mismatch');
      throw new UnauthorizedException('invalid_ipn_signature');
    }

    const record = JSON.parse(rawBody) as Record<string, unknown>;

    const status = String(record.payment_status ?? '');
    const gatewayPaymentRef = String(record.payment_id ?? record.invoice_id ?? '');
    if (!gatewayPaymentRef) {
      throw new UnauthorizedException('invalid_ipn_signature');
    }
    // Keyed by gateway reference + status so replayed deliveries dedupe while
    // the documented status progression (waiting -> confirmed -> finished)
    // still settles exactly once (transactional status guard below).
    const providerEventId = `${gatewayPaymentRef}:${status}`;
    const inserted = await this.pool.query(
      `INSERT INTO billing.payment_events (provider, provider_event_id, event_type, raw_payload, signature_verified)
       VALUES ('nowpayments', $1, $2, $3, true)
       ON CONFLICT (provider, provider_event_id) DO NOTHING
       RETURNING id`,
      [providerEventId, status || 'unknown', JSON.stringify(record)],
    );
    if (inserted.rowCount === 0) {
      this.logger.log(`IPN duplicate ignored: nowpayments/${providerEventId}`);
      return { received: true, duplicate: true };
    }

    if (!SETTLED_STATUSES.has(status)) {
      return { received: true, status };
    }

    const client = await this.pool.connect();
    try {
      await client.query('BEGIN');
      const pay = await client.query(
        `SELECT * FROM billing.payments
         WHERE provider = 'nowpayments' AND provider_payment_id = $1
         FOR UPDATE`,
        [gatewayPaymentRef],
      );
      const paymentRow = pay.rows[0];
      if (paymentRow && paymentRow.status !== 'COMPLETED') {
        await client.query(
          `UPDATE billing.payments SET status = 'COMPLETED', processed_at = now(), updated_at = now()
           WHERE id = $1`,
          [paymentRow.id],
        );
        if (paymentRow.invoice_id) {
          await client.query(
            `UPDATE billing.invoices SET status = 'PAID', paid_at = now(), updated_at = now()
             WHERE id = $1 AND status <> 'PAID'`,
            [paymentRow.invoice_id],
          );
        }
        if (paymentRow.subscription_id) {
          await client.query(
            `UPDATE billing.subscriptions
             SET status = 'ACTIVE',
                 billing_period_start = now(),
                 billing_period_end = now() + CASE WHEN billing_interval = 'ANNUAL'
                     THEN interval '1 year' ELSE interval '1 month' END,
                 updated_at = now()
             WHERE id = $1`,
            [paymentRow.subscription_id],
          );
        }
        await client.query(
          `INSERT INTO audit.audit_events (actor_type, action, entity_type, entity_id, reason, new_value)
           VALUES ('system', 'billing.nowpayments.payment_settled', 'payment', $1, $2, $3::jsonb)`,
          [
            paymentRow.id,
            `NOWPayments ${status} for gateway ref ${gatewayPaymentRef}`,
            JSON.stringify({ gateway_payment_ref: gatewayPaymentRef, status }),
          ],
        );
        this.logger.log(`IPN settled: payment ${paymentRow.id} (${status})`);
      }
      await client.query('COMMIT');
      return { received: true, status };
    } catch (e) {
      await client.query('ROLLBACK').catch(() => undefined);
      this.logger.error(`IPN settlement failed: ${e instanceof Error ? e.message : e}`);
      throw e;
    } finally {
      client.release();
    }
  }
}
