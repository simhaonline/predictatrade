/**
 * EmailCampaignService tests — audience resolution, unsubscribe exclusion,
 * draft validation, and retry-safety (sent rows never double-send).
 *
 * Plain unit test with manual constructor injection (no @nestjs/testing — that
 * package hits the project's pre-existing Jest ESM-interop failure).
 * DB is mocked; the real send path is exercised through a stub EmailService.
 */
import { EmailCampaignService } from './email-campaigns.service';
import { EMAIL_SERVICE } from '../../common/mail/email.service';
import { DB_POOL } from '../../common/database.module';
import { jest } from '@jest/globals';

type QueryResult = { rows: unknown[]; rowCount: number };

describe('EmailCampaignService', () => {
  let service: EmailCampaignService;
  let sentTo: string[];
  const queries: { sql: string; params: unknown[] }[] = [];

  const makePool = (campaignOverride?: Record<string, unknown>) => ({
    query: jest.fn(async (sql: string, params: unknown[] = []): Promise<QueryResult> => {
      queries.push({ sql, params });
      if (sql.includes('count(DISTINCT u.id)')) {
        return { rows: [{ count: '7' }], rowCount: 1 };
      }
      if (sql.includes('consent_records') && sql.includes('marketing_opt_in = true')) {
        return { rows: [{ count: '3' }], rowCount: 1 };
      }
      if (sql.trim().startsWith('SELECT count(*)::text')) {
        return { rows: [{ count: '45' }], rowCount: 1 };
      }
      if (sql.includes('FROM iam.users u')) {
        if (campaignOverride?.audience === 'marketing_opt_in') {
          return { rows: [{ id: 'u1', email: 'a@x.com' }], rowCount: 1 };
        }
        return {
          rows: [
            { id: 'u1', email: 'a@x.com' },
            { id: 'u2', email: 'b@x.com' },
            { id: 'u3', email: 'unsub@x.com' },
          ],
          rowCount: 3,
        };
      }
      if (sql.includes('marketing_unsubscribes')) {
        return { rows: [{ email: 'unsub@x.com' }], rowCount: 1 };
      }
      if (sql.includes('INSERT INTO control.email_campaigns')) {
        return { rows: [{ id: 'c1' }], rowCount: 1 };
      }
      if (sql.includes('SELECT id, subject, body_html')) {
        return {
          rows: [
            campaignOverride ?? {
              id: 'c1', subject: 'Hi', body_html: '<p>x</p>', body_text: 'x',
              campaign_type: 'newsletter', audience: 'all_clients', status: 'draft',
            },
          ],
          rowCount: 1,
        };
      }
      if (sql.includes('email_campaign_recipients') && sql.includes('SELECT email, status')) {
        return { rows: [], rowCount: 0 };
      }
      return { rows: [], rowCount: 0 };
    }),
  });

  const makeEmail = () => ({
    sendPasswordResetEmail: jest.fn(),
    sendOtpEmail: jest.fn(),
    sendWelcomeEmail: jest.fn(),
    sendRawEmail: jest.fn(async (input: { to: string }) => {
      sentTo.push(input.to);
    }),
  });

  const build = (campaignOverride?: Record<string, unknown>) => {
    sentTo = [];
    queries.length = 0;
    const pool = makePool(campaignOverride);
    const email = makeEmail();
    service = new EmailCampaignService(pool as never, email as never);
    return { pool, email };
  };

  it('audienceCounts returns DB truth for all three audiences', async () => {
    build();
    const counts = await service.audienceCounts();
    expect(counts).toEqual({ all_clients: 45, active_subscribers: 7, marketing_opt_in: 3 });
  });

  it('createDraft rejects empty subject/body', async () => {
    build();
    await expect(
      service.createDraft('admin1', { subject: '', bodyHtml: 'x', bodyText: 'y', campaignType: 'alert', audience: 'all_clients' }),
    ).rejects.toThrow('Subject is required');
    await expect(
      service.createDraft('admin1', { subject: 's', bodyHtml: '', bodyText: 'y', campaignType: 'alert', audience: 'all_clients' }),
    ).rejects.toThrow();
  });

  it('sendCampaign excludes unsubscribed addresses for newsletter type and records skip', async () => {
    build();
    const result = await service.sendCampaign('admin1', 'c1');
    expect(sentTo.sort()).toEqual(['a@x.com', 'b@x.com']);
    expect(result.sent).toBe(2);
    expect(result.skipped).toBe(1);
    expect(result.failed).toBe(0);
    const skipUpdate = queries.find(
      (q) => q.sql.includes('email_campaign_recipients') && q.params.includes('skipped_unsubscribed'),
    );
    expect(skipUpdate).toBeDefined();
    expect(skipUpdate!.params).toContain('unsub@x.com');
  });

  it('sendCampaign does NOT skip unsubscribed for alert type (service communication)', async () => {
    build({
      id: 'c1', subject: 'Hi', body_html: '<p>x</p>', body_text: 'x',
      campaign_type: 'alert', audience: 'all_clients', status: 'draft',
    });
    const result = await service.sendCampaign('admin1', 'c1');
    expect(result.sent).toBe(3);
    expect(result.skipped).toBe(0);
    expect(sentTo).toContain('unsub@x.com');
  });

  it('sendCampaign refuses to resend a sent campaign', async () => {
    build({
      id: 'c1', subject: 'Hi', body_html: '<p>x</p>', body_text: 'x',
      campaign_type: 'alert', audience: 'all_clients', status: 'sent',
    });
    await expect(service.sendCampaign('admin1', 'c1')).rejects.toThrow('already sent');
  });

  it('sendCampaign marks failed recipients and continues', async () => {
    const { email } = build();
    email.sendRawEmail.mockImplementationOnce(async (input: { to: string }) => {
      if (input.to === 'a@x.com') throw new Error('relay down');
      sentTo.push(input.to);
    });
    const result = await service.sendCampaign('admin1', 'c1');
    expect(result.failed).toBeGreaterThanOrEqual(1);
    expect(result.sent).toBeGreaterThanOrEqual(1);
  });
});

// Silence unused-import lint for the DI tokens re-exported for reference.
export { EMAIL_SERVICE, DB_POOL };