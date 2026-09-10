import { Injectable, Logger, BadRequestException, NotFoundException, Inject } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';
import { EMAIL_SERVICE } from '../../common/mail/email.service';
import type { EmailService } from '../../common/mail/email.service';

/**
 * EmailCampaignService — admin email notifications to clients.
 *
 * Powers the admin "Email Notifications" page:
 *   - alert       → service communications to ALL clients (no opt-out gate;
 *                   operational notices like maintenance/verification windows)
 *   - newsletter  → content sends; marketing consent respected
 *   - marketing   → promotional sends; marketing consent respected
 *
 * Compliance (CAN-SPAM / GDPR):
 *   * newsletter/marketing campaigns EXCLUDE addresses present in
 *     iam.marketing_unsubscribes at send time (skipped_unsubscribed).
 *   * Every campaign row carries per-recipient delivery status so a retry
 *     never double-sends.
 *   * Every send is audit-logged with actor, audience and counts.
 *
 * Sends are paced (200ms between recipients) to stay well within the
 * mail-relay's capacity and avoid upstream rate flags.
 */

const SEND_PACE_MS = 200;
const MAX_RECIPIENTS = 5_000; // hard cap per campaign send

export interface CampaignAudienceCounts {
  all_clients: number;
  active_subscribers: number;
  marketing_opt_in: number;
}

@Injectable()
export class EmailCampaignService {
  private readonly logger = new Logger(EmailCampaignService.name);

  constructor(
    @Inject(DB_POOL) private readonly pool: Pool,
    @Inject(EMAIL_SERVICE) private readonly email: EmailService,
  ) {}

  /**
   * Audience counts for the compose form (live DB truth).
   */
  async audienceCounts(): Promise<CampaignAudienceCounts> {
    // iam.users has a `status` column ('ACTIVE'|'DELETED') — there is no
    // is_active boolean (that mismatch caused the 500 on first load).
    const all = await this.pool.query<{ count: string }>(
      `SELECT count(*)::text AS count
         FROM iam.users
        WHERE status = 'ACTIVE' AND email IS NOT NULL AND email <> ''`,
    );
    const active = await this.pool.query<{ count: string }>(
      `SELECT count(DISTINCT u.id)::text AS count
         FROM iam.users u
         JOIN billing.subscriptions s ON s.user_id = u.id
        WHERE s.status = 'ACTIVE' AND u.status = 'ACTIVE' AND u.email IS NOT NULL AND u.email <> ''`,
    );
    const optIn = await this.pool.query<{ count: string }>(
      `SELECT count(DISTINCT u.id)::text AS count
         FROM iam.users u
         JOIN iam.consent_records c ON c.user_id = u.id
        WHERE c.marketing_opt_in = true AND u.status = 'ACTIVE' AND u.email IS NOT NULL AND u.email <> ''`,
    );
    return {
      all_clients: parseInt(all.rows[0]?.count ?? '0', 10),
      active_subscribers: parseInt(active.rows[0]?.count ?? '0', 10),
      marketing_opt_in: parseInt(optIn.rows[0]?.count ?? '0', 10),
    };
  }

  /**
   * Create a draft campaign (compose step). Does NOT send.
   */
  async createDraft(
    actorId: string,
    input: { subject: string; bodyHtml: string; bodyText: string; campaignType: string; audience: string },
  ): Promise<{ id: string }> {
    const subject = (input.subject ?? '').trim();
    if (!subject) throw new BadRequestException('Subject is required');
    if (!input.bodyHtml?.trim() || !input.bodyText?.trim()) {
      throw new BadRequestException('Both HTML and text body are required');
    }
    const type = ['alert', 'newsletter', 'marketing'].includes(input.campaignType)
      ? input.campaignType : 'newsletter';
    const audience = ['all_clients', 'active_subscribers', 'marketing_opt_in'].includes(input.audience)
      ? input.audience : 'all_clients';

    const res = await this.pool.query<{ id: string }>(
      `INSERT INTO control.email_campaigns (created_by, subject, body_html, body_text, campaign_type, audience)
       VALUES ($1, $2, $3, $4, $5, $6)
       RETURNING id`,
      [actorId, subject, input.bodyHtml, input.bodyText, type, audience],
    );
    await this.audit(actorId, 'email_campaign.draft_created', res.rows[0].id,
      { subject, campaign_type: type, audience });
    return { id: res.rows[0].id };
  }

  /**
   * Send a draft campaign: resolve the audience, snapshot recipients, then
   * deliver paced. Idempotent per-recipient (sent rows are skipped on retry).
   * newsletter/marketing audiences exclude unsubscribed addresses.
   */
  async sendCampaign(
    actorId: string,
    campaignId: string,
  ): Promise<{ total: number; sent: number; failed: number; skipped: number }> {
    const camp = await this.pool.query<{
      id: string; subject: string; body_html: string; body_text: string;
      campaign_type: string; audience: string; status: string;
    }>(
      `SELECT id, subject, body_html, body_text, campaign_type, audience, status
         FROM control.email_campaigns WHERE id = $1`,
      [campaignId],
    );
    if (camp.rowCount === 0) throw new NotFoundException('Campaign not found');
    const c = camp.rows[0];
    if (c.status === 'sending') throw new BadRequestException('Campaign is already sending');
    if (c.status === 'sent') throw new BadRequestException('Campaign already sent');

    // Resolve audience → recipient rows.
    const recipients = await this.pool.query<{ id: string; email: string }>(
      this.audienceQuery(c.audience),
    );
    if (recipients.rowCount === 0) throw new BadRequestException('Audience resolved to zero recipients');

    // Unsubscribed set (marketing compliance) — checked at send time.
    const unsub = c.campaign_type === 'alert'
      ? new Set<string>()
      : new Set(
          (await this.pool.query<{ email: string }>(
            `SELECT email FROM iam.marketing_unsubscribes`,
          )).rows.map((r) => r.email.trim().toLowerCase()),
        );

    if (recipients.rowCount > MAX_RECIPIENTS) {
      throw new BadRequestException(`Audience exceeds hard cap of ${MAX_RECIPIENTS} recipients`);
    }

    await this.pool.query(
      `UPDATE control.email_campaigns
          SET status = 'sending', total_recipients = $2, sent_at = now()
        WHERE id = $1`,
      [campaignId, recipients.rowCount],
    );

    // Snapshot per-recipient rows (pending) — idempotency anchor.
    const idToUser = new Map<string, string>();
    for (const r of recipients.rows) idToUser.set(r.email.toLowerCase(), r.id);
    const existing = await this.pool.query<{ email: string; status: string }>(
      `SELECT email, status FROM control.email_campaign_recipients WHERE campaign_id = $1`,
      [campaignId],
    );
    const alreadySent = new Set(
      existing.rows.filter((r) => r.status === 'sent').map((r) => r.email.toLowerCase()),
    );
    if (existing.rowCount === 0) {
      const values: unknown[] = [];
      const tuples = recipients.rows.map((r, idx) => {
        values.push(campaignId, r.id, r.email);
        return `($${idx * 3 + 1}, $${idx * 3 + 2}, $${idx * 3 + 3}, 'pending')`;
      });
      await this.pool.query(
        `INSERT INTO control.email_campaign_recipients (campaign_id, user_id, email, status)
         VALUES ${tuples.join(', ')}`,
        values,
      );
    }

    let sent = 0, failed = 0, skipped = 0;

    for (const r of recipients.rows) {
      const email = r.email.toLowerCase();
      if (alreadySent.has(email)) { sent++; continue; } // retry-safe
      if (unsub.has(email)) {
        await this.markRecipient(campaignId, email, 'skipped_unsubscribed', null);
        skipped++;
        continue;
      }
      try {
        await this.sendOne(email, c.subject, c.body_html, c.body_text, c.campaign_type);
        await this.markRecipient(campaignId, email, 'sent', null);
        sent++;
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e);
        await this.markRecipient(campaignId, email, 'failed', msg);
        failed++;
        this.logger.warn(`[CAMPAIGN] send failed to ${email}: ${msg}`);
      }
      await new Promise((res) => setTimeout(res, SEND_PACE_MS));
    }

    await this.pool.query(
      `UPDATE control.email_campaigns
          SET status = $2, sent_count = $3, failed_count = $4, skipped_count = $5, completed_at = now()
        WHERE id = $1`,
      [campaignId, failed > 0 && sent === 0 ? 'failed' : 'sent', sent, failed, skipped],
    );
    await this.audit(actorId, 'email_campaign.sent', campaignId,
      { subject: c.subject, audience: c.audience, total: recipients.rowCount, sent, failed, skipped });
    this.logger.log(`[CAMPAIGN] ${campaignId} "${c.subject}": ${sent} sent, ${failed} failed, ${skipped} skipped`);
    return { total: recipients.rowCount, sent, failed, skipped };
  }

  /**
   * Campaign history (admin page list view).
   */
  async listCampaigns(limit = 50): Promise<unknown> {
    const res = await this.pool.query(
      `SELECT c.id, c.subject, c.campaign_type, c.audience, c.status,
              c.total_recipients, c.sent_count, c.failed_count, c.skipped_count,
              c.created_at, c.sent_at, c.completed_at,
              u.email AS created_by_email
         FROM control.email_campaigns c
         LEFT JOIN iam.users u ON u.id = c.created_by
        ORDER BY c.created_at DESC
        LIMIT $1`,
      [Math.min(Math.max(limit, 1), 200)],
    );
    return { campaigns: res.rows };
  }

  /**
   * Per-recipient detail for one campaign (admin drill-down).
   */
  async campaignDetail(campaignId: string): Promise<unknown> {
    const head = await this.pool.query(
      `SELECT id, subject, body_html, body_text, campaign_type, audience, status,
              total_recipients, sent_count, failed_count, skipped_count,
              created_at, sent_at, completed_at
         FROM control.email_campaigns WHERE id = $1`,
      [campaignId],
    );
    if (head.rowCount === 0) throw new NotFoundException('Campaign not found');
    const recips = await this.pool.query(
      `SELECT email, status, error, sent_at
         FROM control.email_campaign_recipients
        WHERE campaign_id = $1
        ORDER BY id`,
      [campaignId],
    );
    return { campaign: head.rows[0], recipients: recips.rows };
  }

  /**
   * Audience SQL — shared truth for counts and send resolution.
   */
  private audienceQuery(audience: string): string {
    const base = `SELECT u.id, u.email
                    FROM iam.users u`;
    switch (audience) {
      case 'active_subscribers':
        return `${base}
                  JOIN billing.subscriptions s ON s.user_id = u.id
                 WHERE s.status = 'ACTIVE' AND u.status = 'ACTIVE' AND u.email IS NOT NULL AND u.email <> ''`;
      case 'marketing_opt_in':
        return `${base}
                  JOIN iam.consent_records c ON c.user_id = u.id
                 WHERE c.marketing_opt_in = true AND u.status = 'ACTIVE' AND u.email IS NOT NULL AND u.email <> ''`;
      case 'all_clients':
      default:
        return `${base}
                 WHERE u.status = 'ACTIVE' AND u.email IS NOT NULL AND u.email <> ''`;
    }
  }

  /** Send one campaign email with an unsubscribe header for marketing types. */
  private async sendOne(email: string, subject: string, html: string, text: string, type: string): Promise<void> {
    if (!this.email.sendRawEmail) {
      throw new Error('Email provider does not support raw campaign sends');
    }
    await this.email.sendRawEmail({
      to: email,
      subject,
      bodyHtml: html,
      bodyText: text,
      listUnsubscribe: type === 'marketing' || type === 'newsletter',
    });
  }

  private async markRecipient(campaignId: string, email: string, status: string, error: string | null) {
    await this.pool.query(
      `UPDATE control.email_campaign_recipients
          SET status = $3,
              error = $4,
              sent_at = CASE WHEN $3 = 'sent' THEN now() ELSE sent_at END
        WHERE campaign_id = $1 AND email = $2`,
      [campaignId, email, status, error],
    );
  }

  private async audit(actorId: string, action: string, entityId: string, newValue: unknown) {
    await this.pool.query(
      `INSERT INTO audit.audit_events (actor_type, actor_id, action, entity_type, entity_id, new_value)
       VALUES ('admin', $1, $2, 'email_campaign', $3, $4)`,
      [actorId, action, entityId, JSON.stringify(newValue ?? {})],
    );
  }
}