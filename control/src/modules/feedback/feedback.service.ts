import { Injectable, Logger, BadRequestException, NotFoundException, ForbiddenException, Inject } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';

/**
 * FeedbackService — customer feedback with admin moderation + Featured toggle.
 *
 * User side (dashboard):
 *   POST /feedback            submit feedback (category + rating + message)
 *   GET  /feedback/mine       the caller's own submissions
 *
 * Admin side (admin console, Customer Management):
 *   GET  /admin/feedback            list (status filter + paging)
 *   GET  /admin/feedback/stats      aggregate counts + average rating
 *   POST /admin/feedback/:id/status   moderate: new → reviewed | hidden
 *   POST /admin/feedback/:id/featured set the Featured requirement toggle
 *   POST /admin/feedback/:id/note     attach admin note
 *
 * Rules:
 *   * Submission is rate-limited (controller) + DB-guarded (max 5 open per user
 *     per 24h) to prevent flooding.
 *   * Users can never edit or delete feedback after submission — moderation is
 *     admin-only, soft-hide (status='hidden') keeps history intact.
 *   * Every admin action is audit-logged.
 */

const CATEGORIES = ['bug', 'feature_request', 'usability', 'performance', 'billing', 'other'] as const;

@Injectable()
export class FeedbackService {
  private readonly logger = new Logger(FeedbackService.name);

  constructor(@Inject(DB_POOL) private readonly pool: Pool) {}

  /** Submit feedback (authenticated user). */
  async submit(userId: string, input: { category: string; rating: number; message: string }): Promise<unknown> {
    const category = String(input.category ?? '').trim().toLowerCase();
    if (!CATEGORIES.includes(category as (typeof CATEGORIES)[number])) {
      throw new BadRequestException(`category must be one of: ${CATEGORIES.join(', ')}`);
    }
    const rating = Number(input.rating);
    if (!Number.isInteger(rating) || rating < 1 || rating > 5) {
      throw new BadRequestException('rating must be an integer 1-5');
    }
    const message = String(input.message ?? '').trim();
    if (message.length < 10 || message.length > 2000) {
      throw new BadRequestException('message must be 10-2000 characters');
    }

    // Flood guard: max 5 submissions per user in 24h.
    const recent = await this.pool.query<{ count: string }>(
      `SELECT count(*)::text AS count FROM control.customer_feedback
        WHERE user_id = $1 AND created_at > now() - interval '24 hours'`,
      [userId],
    );
    if (parseInt(recent.rows[0]?.count ?? '0', 10) >= 5) {
      throw new ForbiddenException('Feedback limit reached (5 per 24 hours). Please wait before submitting again.');
    }

    const profile = await this.pool.query<{ email: string; full_name: string | null }>(
      `SELECT email, full_name FROM iam.users WHERE id = $1`,
      [userId],
    );
    if (profile.rowCount === 0) throw new NotFoundException('User not found');

    const res = await this.pool.query<{ id: string; created_at: Date }>(
      `INSERT INTO control.customer_feedback (user_id, email, full_name, category, rating, message)
       VALUES ($1, $2, $3, $4, $5, $6)
       RETURNING id, created_at`,
      [userId, profile.rows[0].email, profile.rows[0].full_name, category, rating, message],
    );
    this.logger.log(`[FEEDBACK] ${profile.rows[0].email} submitted ${category} (${rating}/5)`);
    return { id: res.rows[0].id, created_at: res.rows[0].created_at, message: 'Thank you — your feedback was received.' };
  }

  /** The caller's own submissions. */
  async listMine(userId: string, limit = 20): Promise<unknown> {
    const res = await this.pool.query(
      `SELECT id, category, rating, message, status, featured, created_at,
              admin_note
         FROM control.customer_feedback
        WHERE user_id = $1
        ORDER BY created_at DESC
        LIMIT $2`,
      [userId, Math.min(Math.max(limit, 1), 100)],
    );
    return { items: res.rows };
  }

  /** Admin list with status filter + paging. */
  async adminList(status: string | undefined, page = 1, limit = 20): Promise<unknown> {
    const offset = (page - 1) * limit;
    const params: unknown[] = [Math.min(Math.max(limit, 1), 100), offset];
    let where = '';
    if (status && ['new', 'reviewed', 'hidden'].includes(status)) {
      where = `WHERE f.status = $3`;
      params.push(status);
    }
    const [data, count] = await Promise.all([
      this.pool.query(
        `SELECT f.id, f.user_id, f.email, f.full_name, f.category, f.rating, f.message,
                f.status, f.featured, f.featured_at, f.admin_note, f.created_at, f.reviewed_at
           FROM control.customer_feedback f
           ${where}
          ORDER BY f.featured DESC, f.created_at DESC
          LIMIT $1 OFFSET $2`,
        params,
      ),
      this.pool.query(
        `SELECT count(*)::int AS total FROM control.customer_feedback f ${where}`,
        status ? [status] : [],
      ),
    ]);
    return { items: data.rows, total: count.rows[0]?.total ?? 0, page, limit };
  }

  /** Aggregate stats for the admin overview card. */
  async adminStats(): Promise<unknown> {
    const res = await this.pool.query(
      `SELECT count(*)::int AS total,
              count(*) FILTER (WHERE status = 'new')::int AS new,
              count(*) FILTER (WHERE status = 'hidden')::int AS hidden,
              count(*) FILTER (WHERE featured)::int AS featured,
              COALESCE(round(avg(rating)::numeric, 2), 0) AS avg_rating,
              count(*) FILTER (WHERE rating <= 2)::int AS detractors,
              count(*) FILTER (WHERE rating >= 4)::int AS promoters
         FROM control.customer_feedback`,
    );
    return res.rows[0];
  }

  /** Moderate: set status (new → reviewed → hidden). Hiding is reversible. */
  async adminSetStatus(actorId: string, id: string, status: string): Promise<unknown> {
    if (!['new', 'reviewed', 'hidden'].includes(status)) {
      throw new BadRequestException('status must be new | reviewed | hidden');
    }
    const res = await this.pool.query(
      `UPDATE control.customer_feedback
          SET status = $2,
              reviewed_at = CASE WHEN $2 = 'reviewed' THEN now() ELSE reviewed_at END
        WHERE id = $1::uuid
        RETURNING id, status`,
      [id, status],
    );
    if (res.rowCount === 0) throw new NotFoundException('Feedback not found');
    await this.audit(actorId, 'FEEDBACK_STATUS', id, { status });
    return res.rows[0];
  }

  /** THE requirement toggle: mark feedback as Featured (or un-feature). */
  async adminSetFeatured(actorId: string, id: string, featured: boolean): Promise<unknown> {
    if (typeof featured !== 'boolean') {
      throw new BadRequestException('featured must be a boolean');
    }
    const res = await this.pool.query(
      `UPDATE control.customer_feedback
          SET featured = $2,
              featured_at = CASE WHEN $2 THEN now() ELSE NULL END,
              featured_by = CASE WHEN $2 THEN $3::uuid ELSE NULL END
        WHERE id = $1::uuid
        RETURNING id, featured, featured_at`,
      [id, featured, actorId],
    );
    if (res.rowCount === 0) throw new NotFoundException('Feedback not found');
    await this.audit(actorId, featured ? 'FEEDBACK_FEATURED' : 'FEEDBACK_UNFEATURED', id, { featured });
    this.logger.log(`[FEEDBACK] ${featured ? 'featured' : 'unfeatured'} ${id} by admin`);
    return res.rows[0];
  }

  /** Attach/replace an admin note. */
  async adminSetNote(actorId: string, id: string, note: string): Promise<unknown> {
    const trimmed = String(note ?? '').trim().slice(0, 1000);
    const res = await this.pool.query(
      `UPDATE control.customer_feedback SET admin_note = $2 WHERE id = $1::uuid RETURNING id, admin_note`,
      [id, trimmed],
    );
    if (res.rowCount === 0) throw new NotFoundException('Feedback not found');
    await this.audit(actorId, 'FEEDBACK_NOTE', id, { note: trimmed });
    return res.rows[0];
  }

  /** Public featured list (marketing surface — only featured + not hidden). */
  async listPublicFeatured(limit = 10): Promise<unknown> {
    const res = await this.pool.query(
      `SELECT full_name, category, rating, message, featured_at
         FROM control.customer_feedback
        WHERE featured = true AND status <> 'hidden'
        ORDER BY featured_at DESC
        LIMIT $1`,
      [Math.min(Math.max(limit, 1), 50)],
    );
    // Privacy: attribute by FIRST NAME ONLY on the public surface.
    const items = res.rows.map((r) => ({
      first_name: (r.full_name || '').split(' ')[0] || 'A client',
      category: r.category,
      rating: r.rating,
      message: r.message,
      featured_at: r.featured_at,
    }));
    return { items };
  }

  private async audit(actorId: string, action: string, entityId: string, newValue: unknown) {
    try {
      await this.pool.query(
        `INSERT INTO audit.audit_events (actor_type, actor_id, action, entity_type, entity_id, new_value)
         VALUES ('admin', $1, $2, 'customer_feedback', $3, $4)`,
        [actorId, action, entityId, JSON.stringify(newValue ?? {})],
      );
    } catch (e) {
      this.logger.warn(`feedback audit insert failed: ${e instanceof Error ? e.message : e}`);
    }
  }
}