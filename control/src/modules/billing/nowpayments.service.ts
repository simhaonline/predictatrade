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
import { CommissionsService } from '../commissions/commissions.service';
import * as crypto from 'crypto';
import Decimal from 'decimal.js';

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
    private commissionsService: CommissionsService,
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
    // Exact-decimal read of the plan price (audit 2.4). `price` is used as both
    // the gateway payload amount and the stored payment amount, so it is kept as a
    // Decimal and converted to a number/string only at the external boundary.
    const price = new Decimal(interval === 'ANNUAL' ? p.annual_price : p.monthly_price);
    if (!price.isFinite() || price.lte(0)) {
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
          price_amount: price.toNumber(),
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
      [userId, subscriptionId, invoiceId, providerInvoiceId, price.toString()],
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

    // ─── AMOUNT VERIFICATION (anti-underpaid-scam, 2026-08-29) ───
    // NOWPayments reports the crypto actually received in the IPN. A scammer
    // pattern is paying a tiny USDT amount on the invoice address and relying
    // on the gateway still reaching 'confirmed'. We only settle when the paid
    // amount covers the expected price (NOWPayments price_paid/actually_paid
    // in the invoice currency, USD) within a configured tolerance, or an
    // overpay. Underpaid marks the payment UNDERPAID and does NOT activate.
    const pay = await this.pool.query(
      `SELECT p.*, i.total AS invoice_total
       FROM billing.payments p
       LEFT JOIN billing.invoices i ON i.id = p.invoice_id
       WHERE p.provider = 'nowpayments' AND p.provider_payment_id = $1
       LIMIT 1`,
      [gatewayPaymentRef],
    );
    const paymentPrecheck = pay.rows[0];
    if (paymentPrecheck) {
      // Exact-decimal money math (audit 2.4): no floating-point arithmetic on
      // settlement amounts. `expected` falls back to the invoice total only when
      // the payment amount is absent/zero.
      const expected = new Decimal(paymentPrecheck.amount ?? 0).gt(0)
        ? new Decimal(paymentPrecheck.amount ?? 0)
        : new Decimal(paymentPrecheck.invoice_total ?? 0);
      const actualCandidates = [
        record.actually_paid, record.price_paid, record.pay_amount,
        record.actually_paid_in_usd, record.price_amount_paid,
      ]
        .map((v) => new Decimal(v == null ? 0 : (v as number | string)))
        .filter((d) => d.gt(0));
      const tolerancePct = new Decimal(
        Number.isFinite(Number(process.env.NOWPAYMENTS_UNDERPAY_TOLERANCE_PCT))
          ? process.env.NOWPAYMENTS_UNDERPAY_TOLERANCE_PCT
          : '2',
      );
      if (expected.gt(0) && actualCandidates.length > 0) {
        const actual = actualCandidates.reduce(
          (max, d) => (d.gt(max) ? d : max),
          new Decimal(0),
        );
        const minOk = expected.minus(expected.times(tolerancePct).div(100));
        if (actual.plus(new Decimal('1e-9')).lt(minOk)) {
          // Underpaid — record, notify-log, do NOT settle; idempotent via
          // payment_events dedupe above (the UNDERPAID transition is one-shot
          // guarded by the payments.status <> 'UNDERPAID' case below).
          await this.pool.query(
            `UPDATE billing.payments SET status = 'UNDERPAID', updated_at = now()
             WHERE provider = 'nowpayments' AND provider_payment_id = $1 AND status NOT IN ('COMPLETED','UNDERPAID')`,
            [gatewayPaymentRef],
          );
          await this.pool.query(
            `INSERT INTO audit.audit_events (actor_type, action, entity_type, entity_id, reason, new_value)
             VALUES ('system', 'billing.nowpayments.underpaid', 'payment', $1, $2, $3::jsonb)`,
            [paymentPrecheck.id, `Underpaid: got ${actual.toString()} USD, expected >= ${minOk.toString()} USD`,
             JSON.stringify({ gateway_payment_ref: gatewayPaymentRef, actual: actual.toString(), expected: expected.toString() })],
          );
          this.logger.warn(`IPN underpaid: ${actual.toString()} < ${minOk.toString()} USD for ${gatewayPaymentRef} — NOT settling`);
          return { received: true, status: 'underpaid' };
        }
        // Overpay tolerance: accepted (credit applied in full) — no upper bound.
      }
      // If the gateway omitted the paid amount entirely we keep the legacy
      // behavior (settle) — but this is logged for review; set
      // NOWPAYMENTS_REQUIRE_AMOUNT=strict to refuse settling without amounts.
      if (actualCandidates.length === 0 && process.env.NOWPAYMENTS_REQUIRE_AMOUNT === 'strict') {
        this.logger.warn(`IPN amount missing for ${gatewayPaymentRef} — strict mode NOT settling`);
        return { received: true, status: 'amount_unverified' };
      }
    }

    const client = await this.pool.connect();
    try {
      await client.query('BEGIN');
      const pay = await client.query(
        `SELECT p.*, s.plan_id AS sub_plan_id
         FROM billing.payments p
         LEFT JOIN billing.subscriptions s ON s.id = p.subscription_id
         WHERE p.provider = 'nowpayments' AND p.provider_payment_id = $1
         FOR UPDATE`,
        [gatewayPaymentRef],
      );
      const paymentRow = pay.rows[0];
      let commissionCredit:
        | { userId: string; planId: string; paymentId: string; amount: string; currency: string }
        | null = null;
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

        // Capture validated-revenue inputs so commission is credited ONLY after
        // the settlement transaction commits (audit 2.4). The ledger write is
        // itself transactional and idempotent per settled payment id.
        if (paymentRow.sub_plan_id) {
          commissionCredit = {
            userId: String(paymentRow.user_id),
            planId: String(paymentRow.sub_plan_id),
            paymentId: String(paymentRow.id),
            amount: String(paymentRow.amount),
            currency: paymentRow.currency || 'USD',
          };
        }
      }
      await client.query('COMMIT');

      // Credit referral commission from VALIDATED revenue only (audit 2.4).
      // Triggered here — the NOWPayments settlement webhook — never on license
      // assignment. Fail-safe: a commission failure does not roll back the
      // already-settled payment, and the ledger is idempotent per payment.
      if (commissionCredit) {
        try {
          await this.commissionsService.creditReferralForSettledRevenue(
            commissionCredit.userId,
            commissionCredit.planId,
            commissionCredit.paymentId,
            commissionCredit.amount,
            commissionCredit.currency,
          );
        } catch (e) {
          this.logger.error(
            `Referral commission credit failed for settled payment ${commissionCredit.paymentId}: ${e instanceof Error ? e.message : e}`,
          );
        }
      }

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
