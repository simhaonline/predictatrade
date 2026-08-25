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
import { Decimal } from 'decimal.js';
import * as crypto from 'crypto';

const STRIPE_API_BASE = 'https://api.stripe.com/v1';

interface StripeCheckoutDto {
  planId: string;
  billingInterval: 'MONTHLY' | 'ANNUAL';
  userId: string;
  successUrl: string;
  cancelUrl: string;
}

/**
 * Stripe card checkout (control-plane only; never in the Go hot path).
 *
 * Security:
 * - create-checkout fails closed with 503 payment_gateway_not_configured when
 *   STRIPE_SECRET_KEY is empty.
 * - Webhooks are authenticated by HMAC-SHA256 over `${timestamp}.${rawBody}`
 *   using STRIPE_WEBHOOK_SECRET, timing-safe compared to the `v1` value in the
 *   `Stripe-Signature` header. Missing secret or any mismatch rejects the
 *   delivery before any state mutation.
 * Financial integrity:
 * - Idempotent per (provider, provider_event_id) via billing.payment_events.
 * - Settlement mutates payment/invoice/subscription inside one DB transaction.
 * - checkout.session.completed is ONLY treated as paid when its payment_status
 *   is 'paid' (the session event alone is not a charge — mirroring NOWPayments).
 */
@Injectable()
export class StripeService {
  private logger = new Logger(StripeService.name);
  constructor(
    @Inject(DB_POOL) private pool: Pool,
    private billingService: BillingService,
  ) {}

  async createCheckoutSession(dto: StripeCheckoutDto): Promise<{ url: string; sessionId: string }> {
    const secret = (process.env.STRIPE_SECRET_KEY || '').trim();
    if (!secret) {
      this.logger.warn('Stripe checkout rejected: STRIPE_SECRET_KEY not configured');
      throw new ServiceUnavailableException('payment_gateway_not_configured');
    }

    const interval = String(dto.billingInterval || 'MONTHLY').toUpperCase();
    if (!['MONTHLY', 'ANNUAL'].includes(interval)) {
      throw new BadRequestException('billing_interval must be MONTHLY or ANNUAL');
    }

    const plan = await this.pool.query(
      `SELECT id, name, code, monthly_price, annual_price, allowed_strategies, currency
       FROM control.plans WHERE id = $1 AND status = 'ACTIVE'`,
      [dto.planId],
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

    let sub = await this.pool.query(
      `SELECT s.id FROM billing.subscriptions s
       WHERE s.user_id = $1 AND s.plan_id = $2 AND s.billing_interval = $3 AND s.status = 'INCOMPLETE'
       ORDER BY s.created_at DESC LIMIT 1`,
      [dto.userId, dto.planId, interval],
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
        [subscriptionId, dto.userId, dto.planId, interval, JSON.stringify(p.allowed_strategies || [])],
      );
    }

    const invoiceId = await this.billingService.generateInvoiceForSubscription(subscriptionId, dto.userId);

    const unitAmount = new Decimal(price).times(100).round().toNumber();
    const successUrl = dto.successUrl || process.env.STRIPE_SUCCESS_URL || 'https://platform.predictatrade.com/dashboard/billing?payment=success';
    const cancelUrl = dto.cancelUrl || process.env.STRIPE_CANCEL_URL || 'https://platform.predictatrade.com/dashboard/billing?payment=cancelled';

    let resp: Response;
    try {
      resp = await fetch(`${STRIPE_API_BASE}/checkout/sessions`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${secret}`,
          'Content-Type': 'application/x-www-form-urlencoded',
        },
        body: new URLSearchParams({
          mode: 'payment',
          success_url: successUrl,
          cancel_url: cancelUrl,
          'client_reference_id': subscriptionId,
          'metadata[subscription_id]': subscriptionId,
          'metadata[invoice_id]': invoiceId,
          'payment_intent_data[metadata][subscription_id]': subscriptionId,
          'payment_intent_data[metadata][invoice_id]': invoiceId,
          'line_items[0][quantity]': '1',
          'line_items[0][price_data][currency]': 'usd',
          'line_items[0][price_data][product_data][name]': `${p.name} subscription (${interval})`,
          'line_items[0][price_data][unit_amount]': String(unitAmount),
        }).toString(),
      });
    } catch (e) {
      this.logger.error(`Stripe unreachable: ${e instanceof Error ? e.message : e}`);
      throw new ServiceUnavailableException('payment_gateway_unreachable');
    }
    if (!resp.ok) {
      this.logger.error(`Stripe create-session failed: HTTP ${resp.status}`);
      throw new ServiceUnavailableException('payment_gateway_error');
    }
    const data = (await resp.json()) as { id?: string; url?: string };
    if (!data?.id || !data?.url) {
      this.logger.error('Stripe create-session returned an unexpected payload');
      throw new ServiceUnavailableException('payment_gateway_error');
    }

    await this.pool.query(
      `INSERT INTO billing.payments
       (user_id, subscription_id, invoice_id, provider, provider_payment_id, provider_event_id,
        amount, currency, payment_type, status)
       VALUES ($1,$2,$3,'stripe',$4,$4,$5,'USD','SUBSCRIPTION','PENDING')
       ON CONFLICT (provider, provider_payment_id) WHERE provider_payment_id IS NOT NULL
       DO NOTHING`,
      [dto.userId, subscriptionId, invoiceId, data.id, price],
    );

    return { url: data.url, sessionId: data.id };
  }

  /**
   * Stripe webhook handler. Signature IS the auth — endpoint must stay public.
   * Stripe sends `Stripe-Signature: t=<ts>,v1=<hex HMAC-SHA256 over the raw
   * body using STRIPE_WEBHOOK_SECRET>` where the signed payload is
   * `${timestamp}.${rawBody}`. We verify over the raw body bytes so the
   * signature matches exactly what Stripe computed.
   */
  async handleWebhook(rawBody: string, signatureHeader?: string | string[]) {
    const secret = (process.env.STRIPE_WEBHOOK_SECRET || '').trim();
    if (!secret) {
      this.logger.error('Webhook rejected: STRIPE_WEBHOOK_SECRET not configured');
      throw new UnauthorizedException('webhook_not_configured');
    }
    const provided = Array.isArray(signatureHeader) ? signatureHeader[0] : signatureHeader;
    if (!provided || typeof rawBody !== 'string' || rawBody.length === 0) {
      this.logger.warn('Webhook rejected: missing signature or payload');
      throw new UnauthorizedException('invalid_webhook_signature');
    }

    let timestamp = '';
    let v1 = '';
    for (const part of provided.split(',')) {
      const [k, v] = part.split('=');
      if (k === 't') timestamp = v;
      else if (k === 'v1') v1 = v;
    }
    if (!timestamp || !v1) {
      this.logger.warn('Webhook rejected: malformed Stripe-Signature');
      throw new UnauthorizedException('invalid_webhook_signature');
    }

    const expected = crypto.createHmac('sha256', secret).update(`${timestamp}.${rawBody}`).digest('hex');
    const a = Buffer.from(v1, 'utf8');
    const b = Buffer.from(expected, 'utf8');
    if (a.length !== b.length || !crypto.timingSafeEqual(a, b)) {
      this.logger.warn('Webhook rejected: HMAC-SHA256 signature mismatch');
      throw new UnauthorizedException('invalid_webhook_signature');
    }

    const record = JSON.parse(rawBody) as Record<string, any>;
    const eventType = String(record?.type ?? '');
    const providerEventId = String(record?.id ?? `${eventType}:${timestamp}`);

    const inserted = await this.pool.query(
      `INSERT INTO billing.payment_events (provider, provider_event_id, event_type, raw_payload, signature_verified)
       VALUES ('stripe', $1, $2, $3, true)
       ON CONFLICT (provider, provider_event_id) DO NOTHING
       RETURNING id`,
      [providerEventId, eventType || 'unknown', rawBody],
    );
    if (inserted.rowCount === 0) {
      this.logger.log(`Webhook duplicate ignored: stripe/${providerEventId}`);
      return { received: true, duplicate: true };
    }

    const object = record?.data?.object ?? {};

    // checkout.session.completed is only a charge when payment_status === 'paid'.
    if (eventType === 'checkout.session.completed' && object?.payment_status !== 'paid') {
      this.logger.log(`Stripe session ${object?.id} not yet paid (${object?.payment_status}); skipping settlement`);
      return { received: true, eventType };
    }

    const paidEvents = ['checkout.session.completed', 'invoice.paid', 'payment_intent.succeeded'];
    if (paidEvents.includes(eventType)) {
      const subscriptionId =
        object?.client_reference_id || object?.metadata?.subscription_id || record?.data?.subscription_id;
      const paymentId =
        object?.payment_intent || object?.id || record?.data?.subscription_id || providerEventId;
      const invoiceId = object?.metadata?.invoice_id;

      if (subscriptionId) {
        await this.settle(subscriptionId, invoiceId, paymentId);
      }
    }

    return { received: true, eventType: eventType || 'unknown' };
  }

  private async settle(subscriptionId: string, invoiceId: string | undefined, paymentId: string) {
    const client = await this.pool.connect();
    try {
      await client.query('BEGIN');
      const sub = await client.query(
        `SELECT user_id, billing_interval FROM billing.subscriptions WHERE id = $1 FOR UPDATE`,
        [subscriptionId],
      );
      if (sub.rows.length === 0) {
        await client.query('ROLLBACK');
        return;
      }
      const owner = sub.rows[0].user_id;

      // Mark the period invoice paid (idempotent per subscription/period) and
      // ensure a payment row exists, then activate the subscription.
      const invId = invoiceId || (await this.billingService.generateInvoiceForSubscription(subscriptionId, owner, { markPaid: true, paymentId }));
      await client.query(
        `INSERT INTO billing.payments
         (user_id, subscription_id, invoice_id, provider, provider_payment_id, provider_event_id, amount, currency, payment_type, status)
         VALUES ($1,$2,$3,'stripe',$4,$4,0,'USD','SUBSCRIPTION','COMPLETED')
         ON CONFLICT (provider, provider_payment_id) WHERE provider_payment_id IS NOT NULL
         DO NOTHING`,
        [owner, subscriptionId, invId, paymentId],
      );
      await client.query(
        `UPDATE billing.invoices SET status = 'PAID', paid_at = now(), updated_at = now()
         WHERE id = $1 AND status <> 'PAID'`,
        [invId],
      );
      await client.query(
        `UPDATE billing.subscriptions
         SET status = 'ACTIVE',
             billing_period_start = now(),
             billing_period_end = now() + CASE WHEN billing_interval = 'ANNUAL'
                 THEN interval '1 year' ELSE interval '1 month' END,
             updated_at = now()
         WHERE id = $1`,
        [subscriptionId],
      );
      await client.query(
        `INSERT INTO billing.subscription_events (subscription_id, event_type, metadata, actor_id, created_at)
         VALUES ($1, 'ACTIVATED', $2::jsonb, $3, now())`,
        [subscriptionId, JSON.stringify({ provider: 'stripe', payment_id: paymentId }), owner],
      );
      await client.query('COMMIT');
      this.logger.log(`Stripe settled: subscription ${subscriptionId}`);
    } catch (e) {
      await client.query('ROLLBACK').catch(() => undefined);
      this.logger.error(`Stripe settlement failed: ${e instanceof Error ? e.message : e}`);
      throw e;
    } finally {
      client.release();
    }
  }
}
