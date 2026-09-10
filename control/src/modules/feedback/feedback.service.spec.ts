/**
 * FeedbackService tests — submission validation, flood guard, admin
 * moderation, the Featured requirement toggle, and audit logging.
 *
 * Plain unit test with manual constructor injection (no @nestjs/testing —
 * that package hits the project's pre-existing Jest ESM-interop failure).
 * DB is mocked; queries are captured for assertions.
 */
import { FeedbackService } from './feedback.service';
import { DB_POOL } from '../../common/database.module';
import { jest } from '@jest/globals';

type QueryResult = { rows: unknown[]; rowCount: number };

describe('FeedbackService', () => {
  let service: FeedbackService;
  const queries: { sql: string; params: unknown[] }[] = [];

  const makePool = (opts: {
    recentCount?: number;
    userExists?: boolean;
    feedbackExists?: boolean;
  } = {}) => ({
    query: jest.fn(async (sql: string, params: unknown[] = []): Promise<QueryResult> => {
      queries.push({ sql, params });
      if (sql.includes("created_at > now() - interval '24 hours'")) {
        return { rows: [{ count: String(opts.recentCount ?? 0) }], rowCount: 1 };
      }
      if (sql.includes('FROM iam.users WHERE id = $1')) {
        return opts.userExists === false
          ? { rows: [], rowCount: 0 }
          : { rows: [{ email: 'client@x.com', full_name: 'Test Client' }], rowCount: 1 };
      }
      if (sql.includes('INSERT INTO control.customer_feedback')) {
        return { rows: [{ id: 'fb1', created_at: new Date() }], rowCount: 1 };
      }
      if (sql.includes('avg(rating)')) {
        return { rows: [{ total: 3, new: 1, hidden: 0, featured: 1, avg_rating: '4.33', detractors: 0, promoters: 2 }], rowCount: 1 };
      }
      if (sql.includes('FROM control.customer_feedback') && sql.includes('count(*)::int AS total')) {
        return { rows: [{ total: 1 }], rowCount: 1 };
      }
      if (sql.includes('UPDATE control.customer_feedback')) {
        return opts.feedbackExists === false
          ? { rows: [], rowCount: 0 }
          : { rows: [{ id: 'fb1', status: 'reviewed', featured: true, featured_at: new Date(), admin_note: 'n' }], rowCount: 1 };
      }
      return { rows: [], rowCount: 0 };
    }),
  });

  const build = (opts?: Parameters<typeof makePool>[0]) => {
    queries.length = 0;
    const pool = makePool(opts);
    service = new FeedbackService(pool as never);
    return pool;
  };

  /* ── submission ── */

  it('submit inserts feedback for a valid payload', async () => {
    build();
    const res = await service.submit('u1', { category: 'bug', rating: 4, message: 'Great platform overall, one bug to report' });
    expect(res).toHaveProperty('id', 'fb1');
    const insert = queries.find((q) => q.sql.includes('INSERT INTO control.customer_feedback'));
    expect(insert).toBeDefined();
    expect(insert!.params).toEqual(['u1', 'client@x.com', 'Test Client', 'bug', 4, 'Great platform overall, one bug to report']);
  });

  it('submit rejects invalid category', async () => {
    build();
    await expect(service.submit('u1', { category: 'spam', rating: 4, message: 'This is a long enough message' }))
      .rejects.toThrow('category must be one of');
  });

  it('submit rejects rating out of 1-5', async () => {
    build();
    await expect(service.submit('u1', { category: 'bug', rating: 6, message: 'This is a long enough message' }))
      .rejects.toThrow('rating must be an integer 1-5');
    await expect(service.submit('u1', { category: 'bug', rating: 0, message: 'This is a long enough message' }))
      .rejects.toThrow();
  });

  it('submit rejects message shorter than 10 chars', async () => {
    build();
    await expect(service.submit('u1', { category: 'bug', rating: 4, message: 'too short' }))
      .rejects.toThrow('message must be 10-2000 characters');
  });

  it('flood guard blocks the 6th submission in 24h', async () => {
    build({ recentCount: 5 });
    await expect(service.submit('u1', { category: 'bug', rating: 4, message: 'This is a long enough message' }))
      .rejects.toThrow('Feedback limit reached');
  });

  it('submit fails cleanly when the user row is missing', async () => {
    build({ userExists: false });
    await expect(service.submit('u1', { category: 'bug', rating: 4, message: 'This is a long enough message' }))
      .rejects.toThrow('User not found');
  });

  /* ── admin moderation ── */

  it('adminSetStatus validates the status value', async () => {
    build();
    await expect(service.adminSetStatus('admin1', 'fb1', 'deleted')).rejects.toThrow('status must be new | reviewed | hidden');
  });

  it('adminSetStatus marks reviewed', async () => {
    build();
    const res = await service.adminSetStatus('admin1', 'fb1', 'reviewed');
    expect(res).toMatchObject({ id: 'fb1', status: 'reviewed' });
    const audit = queries.find((q) => q.sql.includes('audit_events') && q.params.includes('FEEDBACK_STATUS'));
    expect(audit).toBeDefined();
  });

  it('adminSetStatus 404s on unknown id', async () => {
    build({ feedbackExists: false });
    await expect(service.adminSetStatus('admin1', 'nope', 'reviewed')).rejects.toThrow('Feedback not found');
  });

  /* ── the Featured requirement toggle ── */

  it('adminSetFeatured sets featured=true with timestamp + actor', async () => {
    build();
    const res = await service.adminSetFeatured('admin1', 'fb1', true);
    expect(res).toMatchObject({ featured: true });
    const upd = queries.find((q) => q.sql.includes('featured = $2'));
    expect(upd).toBeDefined();
    expect(upd!.params[1]).toBe(true);
    expect(upd!.params[2]).toBe('admin1');
    const audit = queries.find((q) => q.params.includes('FEEDBACK_FEATURED'));
    expect(audit).toBeDefined();
  });

  it('adminSetFeatured can un-feature (false clears featured_by)', async () => {
    build();
    const res = await service.adminSetFeatured('admin1', 'fb1', false);
    expect(res).toMatchObject({ featured: true }); // mock always returns featured:true; the params matter
    const upd = queries.find((q) => q.sql.includes('featured = $2'));
    expect(upd!.params[1]).toBe(false);
    const audit = queries.find((q) => q.params.includes('FEEDBACK_UNFEATURED'));
    expect(audit).toBeDefined();
  });

  it('adminSetFeatured rejects non-boolean values', async () => {
    build();
    await expect(service.adminSetFeatured('admin1', 'fb1', 'yes' as unknown as boolean))
      .rejects.toThrow('featured must be a boolean');
  });

  /* ── public featured list ── */

  it('public featured list filters on featured=true AND status<>hidden', async () => {
    build();
    await service.listPublicFeatured();
    const q = queries.find((q2) => q2.sql.includes('featured = true'));
    expect(q).toBeDefined();
    expect(q!.sql).toContain("status <> 'hidden'");
  });

  /* ── stats ── */

  it('adminStats returns aggregates', async () => {
    build();
    const s = await service.adminStats();
    expect(s).toMatchObject({ total: 3, new: 1, featured: 1, avg_rating: '4.33' });
  });
});