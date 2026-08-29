import { Injectable, Inject, Logger, NotFoundException, BadRequestException } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';
import * as crypto from 'crypto';
import * as fs from 'fs';
import * as path from 'path';

const BRAND = {
  name: 'Predict-A-Trade',
  primary: '#2563EB',
  logoCandidates: [
    '/srv/predictatrade/xauusd/frontend/public/predict-a-trade_horizontal.svg',
    process.cwd() + '/public/predict-a-trade_horizontal.svg',
    path.resolve(__dirname, '../../../frontend/public/predict-a-trade_horizontal.svg'),
    path.resolve(__dirname, '../../frontend/public/predict-a-trade_horizontal.svg'),
  ],
};

function esc(s: unknown): string {
  return String(s ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

@Injectable()
export class BillingService {
  private logger = new Logger(BillingService.name);
  constructor(@Inject(DB_POOL) private pool: Pool) {}

  async listInvoices(userId: string) {
    const r = await this.pool.query(
      `SELECT i.*,
              COALESCE((SELECT SUM(ii.amount) FROM billing.invoice_items ii WHERE ii.invoice_id = i.id), 0) as items_total
       FROM billing.invoices i
       JOIN billing.subscriptions s ON i.subscription_id = s.id
       WHERE s.user_id = $1 ORDER BY i.created_at DESC LIMIT 50`,
      [userId],
    );
    return r.rows;
  }

  /**
   * Subscriber-visible USDT payment status (anti-scam transparency):
   * one row per payment attempt with gateway status, amounts, and the hosted
   * NOWPayments URL to finish/abandoned checkout. Sensitive gateway ids are
   * truncated; payment_status is derived from billing.payments + the latest
   * payment_event so the dashboard mirrors settlement truth exactly.
   */
  async listPaymentsForUser(userId: string) {
    const r = await this.pool.query(
      `SELECT p.id, p.subscription_id, p.invoice_id, p.provider,
              p.amount, p.currency, p.status,
              (SELECT event_type FROM billing.payment_events pe
                WHERE pe.provider = p.provider
                  AND pe.provider_event_id LIKE p.provider_payment_id || ':%'
                ORDER BY pe.received_at DESC LIMIT 1) AS gateway_event,
              i.provider_hosted_url AS hosted_url,
              p.created_at, p.processed_at
       FROM billing.payments p
       LEFT JOIN billing.invoices i ON i.id = p.invoice_id
       WHERE p.user_id = $1
       ORDER BY p.created_at DESC LIMIT 20`,
      [userId],
    );
    return r.rows.map((row) => ({
      ...row,
      // User-friendly payment status for the dashboard banner:
      display_status:
        row.status === 'COMPLETED' ? 'confirmed'
        : row.status === 'UNDERPAID' ? 'underpaid'
        : row.status === 'FAILED' ? 'failed'
        : 'awaiting_payment',
    }));
  }

  /** True when the subscription belongs to the given user. */
  async assertSubscriptionOwnership(userId: string, subscriptionId: string) {
    const r = await this.pool.query(
      `SELECT 1 FROM billing.subscriptions WHERE id = $1 AND user_id = $2`,
      [subscriptionId, userId],
    );
    if (r.rows.length === 0) throw new NotFoundException('Subscription not found');
  }

  /** True when the invoice belongs to the given user. */
  async assertInvoiceOwnership(userId: string, invoiceId: string) {
    const r = await this.pool.query(
      `SELECT 1 FROM billing.invoices WHERE id = $1 AND user_id = $2`,
      [invoiceId, userId],
    );
    if (r.rows.length === 0) throw new NotFoundException('Invoice not found');
  }

  private async nextInvoiceNumber(): Promise<string> {
    const r = await this.pool.query(`SELECT nextval('billing.invoice_seq')::text as n`);
    return 'PAT-INV-' + (r.rows[0]?.n ?? '0').padStart(6, '0');
  }

  /**
   * Create a branded, auditable invoice for a subscription billing period.
   * Idempotent per (subscription, period). Wires line items, totals and the
   * commissionable amount. No fabricated tax/discount is applied (defaults to 0).
   */
  async generateInvoiceForSubscription(
    subscriptionId: string,
    actorUserId: string,
    opts?: { billingPeriodStart?: string; billingPeriodEnd?: string; markPaid?: boolean; paymentId?: string },
  ): Promise<string> {
    await this.assertSubscriptionOwnership(actorUserId, subscriptionId);

    const sub = await this.pool.query(
      `SELECT s.*, p.name as plan_name, p.code as plan_code, p.monthly_price, p.annual_price,
              p.setup_fee, p.currency, p.plan_version
       FROM billing.subscriptions s JOIN control.plans p ON p.id = s.plan_id
       WHERE s.id = $1`,
      [subscriptionId],
    );
    if (!sub.rows[0]) throw new NotFoundException('Subscription not found');
    const s = sub.rows[0];
    const interval = (s.billing_interval || 'MONTHLY').toUpperCase();
    const unitPrice = Number(interval === 'ANNUAL' ? s.annual_price : s.monthly_price) || 0;
    const currency = s.currency || 'USD';

    const periodStart = opts?.billingPeriodStart ? new Date(opts.billingPeriodStart) : new Date(s.billing_period_start || Date.now());
    const periodEnd = opts?.billingPeriodEnd ? new Date(opts.billingPeriodEnd) : new Date(s.billing_period_end || Date.now());

    const existing = await this.pool.query(
      `SELECT id FROM billing.invoices WHERE subscription_id = $1 AND billing_period_start = $2 AND billing_period_end = $3 LIMIT 1`,
      [subscriptionId, periodStart, periodEnd],
    );
    if (existing.rows.length) return existing.rows[0].id;

    const prior = await this.pool.query(`SELECT count(*)::int as c FROM billing.invoices WHERE subscription_id = $1`, [subscriptionId]);
    const isFirst = (prior.rows[0]?.c ?? 0) === 0;

    const items: { description: string; item_type: string; unit_price: number; quantity: number; commissionable: boolean }[] = [];
    let subtotal = 0;
    if (isFirst && Number(s.setup_fee) > 0) {
      items.push({ description: `${s.plan_name} — setup fee`, item_type: 'SETUP_FEE', unit_price: Number(s.setup_fee), quantity: 1, commissionable: false });
      subtotal += Number(s.setup_fee);
    }
    items.push({ description: `${s.plan_name} subscription (${interval})`, item_type: 'SUBSCRIPTION', unit_price: unitPrice, quantity: 1, commissionable: true });
    subtotal += unitPrice;

    const settings = await this.pool.query(`SELECT default_tax_rate, tax_label FROM billing.invoice_settings LIMIT 1`);
    const taxRate = Number(settings.rows[0]?.default_tax_rate ?? 0);
    const discounts = 0;
    const taxes = Math.round(subtotal * taxRate * 1e8) / 1e8;
    const total = Math.round((subtotal - discounts + taxes) * 1e8) / 1e8;
    const commissionable = subtotal;

    const inv = await this.pool.query(
      `INSERT INTO billing.invoices
       (subscription_id, user_id, invoice_number, plan_id, plan_version,
        billing_period_start, billing_period_end, subtotal, discounts, taxes, total,
        commissionable_amount, currency, status, due_date, created_at, updated_at)
       VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,now(),now())
       RETURNING *`,
      [
        subscriptionId, s.user_id, await this.nextInvoiceNumber(), s.plan_id, s.plan_version || 1,
        periodStart, periodEnd, subtotal, discounts, taxes, total, commissionable, currency,
        opts?.markPaid ? 'PAID' : 'OPEN', new Date(Date.now() + 14 * 86400000),
      ],
    );
    const invId = inv.rows[0].id;

    for (const it of items) {
      await this.pool.query(
        `INSERT INTO billing.invoice_items
         (invoice_id, description, item_type, quantity, unit_price, amount, commissionable, metadata, created_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,'{}'::jsonb,now())`,
        [invId, it.description, it.item_type, it.quantity, it.unit_price, it.unit_price * it.quantity, it.commissionable],
      );
    }

    if (opts?.markPaid) {
      await this.pool.query(`UPDATE billing.invoices SET status='PAID', paid_at=now(), updated_at=now() WHERE id=$1`, [invId]);
      if (opts?.paymentId) await this.pool.query(`UPDATE billing.payments SET invoice_id=$1 WHERE id=$2`, [invId, opts.paymentId]);
    }
    return invId;
  }

  async getInvoice(id: string) {
    const inv = await this.pool.query(
      `SELECT i.*, p.name as plan_name, u.email as user_email, u.full_name as user_name
       FROM billing.invoices i JOIN control.plans p ON p.id = i.plan_id
       JOIN iam.users u ON u.id = i.user_id WHERE i.id = $1`,
      [id],
    );
    if (!inv.rows[0]) throw new NotFoundException('Invoice not found');
    const items = await this.pool.query(`SELECT * FROM billing.invoice_items WHERE invoice_id = $1 ORDER BY created_at`, [id]);
    return { ...inv.rows[0], items: items.rows };
  }

  async markInvoicePaid(id: string, paymentId?: string) {
    await this.pool.query(`UPDATE billing.invoices SET status='PAID', paid_at=now(), updated_at=now() WHERE id=$1`, [id]);
    if (paymentId) await this.pool.query(`UPDATE billing.payments SET invoice_id=$1 WHERE id=$2`, [id, paymentId]);
    return this.getInvoice(id);
  }

  private loadLogoSrc(): string {
    for (const c of BRAND.logoCandidates) {
      try {
        if (fs.existsSync(c)) {
          const data = fs.readFileSync(c);
          const ext = c.endsWith('.svg') ? 'svg+xml' : 'png';
          return `data:image/${ext};base64,${data.toString('base64')}`;
        }
      } catch {
        /* try next */
      }
    }
    return '/predict-a-trade_horizontal.svg';
  }

  /** Render a self-contained, branded HTML invoice (brand colors + logo). */
  async renderBrandedInvoiceHtml(id: string): Promise<string> {
    const inv = await this.getInvoice(id);
    const logo = this.loadLogoSrc();
    const fmt = (n: number | string) => Number(n || 0).toFixed(2);
    const itemsRows = (inv.items || [])
      .map(
        (it: any) => `<tr>
        <td style="padding:10px 12px;border-bottom:1px solid #e5e7eb;color:#0f172a">${esc(it.description)}</td>
        <td style="padding:10px 12px;border-bottom:1px solid #e5e7eb;color:#475569;text-align:center">${esc(it.item_type)}</td>
        <td style="padding:10px 12px;border-bottom:1px solid #e5e7eb;color:#475569;text-align:right">${fmt(it.quantity)}</td>
        <td style="padding:10px 12px;border-bottom:1px solid #e5e7eb;color:#475569;text-align:right">${fmt(it.unit_price)}</td>
        <td style="padding:10px 12px;border-bottom:1px solid #e5e7eb;color:#0f172a;text-align:right;font-weight:600">${fmt(it.amount)}</td>
      </tr>`,
      )
      .join('');

    const statusColor = inv.status === 'PAID' ? '#16a34a' : inv.status === 'VOID' ? '#dc2626' : '#d97706';

    return `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<title>Invoice ${esc(inv.invoice_number)} — ${BRAND.name}</title>
<style>
  @media print { body { -webkit-print-color-adjust: exact; print-color-adjust: exact; } .no-print { display:none; } }
  body { font-family: -apple-system, Segoe UI, Roboto, Helvetica, Arial, sans-serif; color:#0f172a; margin:0; background:#f8fafc; }
  .sheet { max-width: 820px; margin: 24px auto; background:#ffffff; border:1px solid #e2e8f0; border-radius:12px; overflow:hidden; }
  .brandbar { background:${BRAND.primary}; padding:22px 32px; display:flex; align-items:center; justify-content:space-between; }
  .brandbar img { height:34px; }
  .brandbar .tag { color:#fff; font-size:12px; letter-spacing:.04em; opacity:.9; }
  .body { padding:32px; }
  .meta { display:flex; justify-content:space-between; gap:24px; flex-wrap:wrap; }
  .meta h1 { font-size:22px; margin:0 0 4px; color:${BRAND.primary}; }
  .status { display:inline-block; padding:4px 12px; border-radius:999px; font-size:12px; font-weight:700; color:#fff; background:${statusColor}; }
  table { width:100%; border-collapse:collapse; margin-top:24px; }
  th { text-align:left; font-size:12px; text-transform:uppercase; letter-spacing:.05em; color:#64748b; border-bottom:2px solid #e2e8f0; padding:10px 12px; }
  tfoot td { padding:10px 12px; }
  .totals { margin-top:18px; margin-left:auto; width:320px; }
  .totals .row { display:flex; justify-content:space-between; padding:6px 0; color:#475569; }
  .totals .grand { font-size:18px; font-weight:700; color:#0f172a; border-top:2px solid #e2e8f0; margin-top:6px; padding-top:10px; }
  .footer { margin-top:28px; padding-top:16px; border-top:1px solid #e2e8f0; color:#94a3b8; font-size:12px; }
  .btn { display:inline-block; margin-top:18px; background:${BRAND.primary}; color:#fff; padding:10px 18px; border-radius:8px; text-decoration:none; font-weight:600; }
</style>
</head>
<body>
  <div class="sheet">
    <div class="brandbar">
      <img src="${logo}" alt="${BRAND.name}" />
      <span class="tag">${BRAND.name} · Billing</span>
    </div>
    <div class="body">
      <div class="meta">
        <div>
          <h1>Invoice ${esc(inv.invoice_number)}</h1>
          <div style="color:#64748b;font-size:13px">${esc(inv.user_name || inv.user_email)} · ${esc(inv.user_email)}</div>
          <div style="margin-top:8px"><span class="status">${esc(inv.status)}</span></div>
        </div>
        <div style="text-align:right;font-size:13px;color:#475569;line-height:1.7">
          <div><strong>Plan:</strong> ${esc(inv.plan_name)}</div>
          <div><strong>Issued:</strong> ${esc(new Date(inv.created_at).toUTCString())}</div>
          <div><strong>Period:</strong> ${esc(new Date(inv.billing_period_start).toISOString().slice(0,10))} → ${esc(new Date(inv.billing_period_end).toISOString().slice(0,10))}</div>
          <div><strong>Due:</strong> ${esc(inv.due_date ? new Date(inv.due_date).toISOString().slice(0,10) : '—')}</div>
          ${inv.paid_at ? `<div><strong>Paid:</strong> ${esc(new Date(inv.paid_at).toISOString().slice(0,10))}</div>` : ''}
        </div>
      </div>

      <table>
        <thead><tr><th>Description</th><th style="text-align:center">Type</th><th style="text-align:right">Qty</th><th style="text-align:right">Unit</th><th style="text-align:right">Amount</th></tr></thead>
        <tbody>${itemsRows || `<tr><td colspan="5" style="padding:16px;text-align:center;color:#94a3b8">No line items</td></tr>`}</tbody>
      </table>

      <div class="totals">
        <div class="row"><span>Subtotal</span><span>${esc(inv.currency)} ${fmt(inv.subtotal)}</span></div>
        <div class="row"><span>Discounts</span><span>${esc(inv.currency)} ${fmt(inv.discounts)}</span></div>
        <div class="row"><span>Taxes</span><span>${esc(inv.currency)} ${fmt(inv.taxes)}</span></div>
        <div class="row grand"><span>Total</span><span>${esc(inv.currency)} ${fmt(inv.total)}</span></div>
      </div>

      <div class="footer">
        ${BRAND.name} · This invoice was generated automatically by the Predict-A-Trade billing system.
        Amounts are shown in ${esc(inv.currency)}. For questions, contact billing@predictatrade.com.
      </div>
      <div class="no-print"><a class="btn" href="javascript:window.print()">Print / Save as PDF</a></div>
    </div>
  </div>
</body>
</html>`;
  }

  /**
   * Webhook handler (P0-CP1 fix).
   * Security: HMAC-SHA256 signature verification against BILLING_WEBHOOK_SECRET
   * (timing-safe compare) + event-id idempotency via billing.payment_events
   * unique (provider, provider_event_id). Unsigned/replayed events are rejected.
   */
  async handleWebhook(body: any, headers: any, rawBody?: Buffer) {
    const secret = process.env.BILLING_WEBHOOK_SECRET;
    if (!secret) {
      this.logger.error('Webhook rejected: BILLING_WEBHOOK_SECRET not configured');
      throw new Error('webhook_secret_not_configured');
    }

    const signatureHeader =
      headers?.['x-pat-signature'] || headers?.['x-signature'] || headers?.['x-webhook-signature'] || '';
    const provided = String(signatureHeader).replace(/^sha256=/, '');
    if (!provided || !rawBody) {
      this.logger.warn('Webhook rejected: missing signature or raw body');
      throw new Error('invalid_webhook_signature');
    }
    const expected = crypto.createHmac('sha256', secret).update(rawBody).digest('hex');
    const a = Buffer.from(provided, 'utf8');
    const b = Buffer.from(expected, 'utf8');
    if (a.length !== b.length || !crypto.timingSafeEqual(a, b)) {
      this.logger.warn('Webhook rejected: HMAC signature mismatch');
      throw new Error('invalid_webhook_signature');
    }

    const event = body?.event_type || body?.type;
    const provider = String(headers?.['x-provider'] || body?.provider || 'internal');
    const providerEventId = String(body?.id || body?.event_id || `${event}:${body?.created_at ?? ''}`);
    const subId = body?.subscription_id || body?.data?.subscription_id || body?.data?.object?.subscription;
    // NOTE: subscription.active intentionally NOT treated as payment (audit CP1).
    // checkout.session.completed is ONLY the session-creation event, NOT a charge
    // (payment is confirmed by payment.succeeded / invoice.paid). Treating it as
    // paid would mark invoices settled before money is received.
    const paidEvents = ['payment.succeeded', 'invoice.paid'];
    const paymentId = body?.payment_id || body?.id || body?.data?.id;

    // Idempotency: record the delivery first; unique(provider, provider_event_id)
    // makes replays no-ops.
    const inserted = await this.pool.query(
      `INSERT INTO billing.payment_events (provider, provider_event_id, event_type, raw_payload, signature_verified)
       VALUES ($1, $2, $3, $4, true)
       ON CONFLICT (provider, provider_event_id) DO NOTHING
       RETURNING id`,
      [provider, providerEventId, String(event || 'unknown'), JSON.stringify(body)],
    );
    if (inserted.rowCount === 0) {
      this.logger.log(`Webhook duplicate ignored: ${provider}/${providerEventId}`);
      return { received: true, duplicate: true, eventId: providerEventId };
    }

    this.logger.log(`Webhook accepted: ${event || 'unknown'} (${providerEventId})`);
    if (subId && paidEvents.includes(event)) {
      try {
        const id = await this.generateInvoiceForSubscription(subId, (await this.subscriptionOwner(subId)) || subId, { markPaid: true, paymentId });
        return { received: true, eventType: event, invoiceId: id };
      } catch (e) {
        this.logger.warn(`Webhook invoice generation failed: ${e instanceof Error ? e.message : e}`);
      }
    }
    return { received: true, eventType: event || 'unknown' };
  }

  private async subscriptionOwner(subId: string): Promise<string | null> {
    const r = await this.pool.query(`SELECT user_id FROM billing.subscriptions WHERE id = $1`, [subId]);
    return r.rows[0]?.user_id ?? null;
  }
}
