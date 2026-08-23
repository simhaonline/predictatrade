import { Injectable, BadRequestException, Inject } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';

@Injectable()
export class PayoutsService {
  constructor(@Inject(DB_POOL) private pool: Pool) {}

  async findByUser(userId: string) {
    const r = await this.pool.query(
      'SELECT * FROM referral.payouts WHERE user_id = $1 ORDER BY created_at DESC LIMIT 20', [userId],
    );
    return r.rows;
  }

  async requestPayout(userId: string, dto: { amount: number; method: string; destination: string }) {
    const id = crypto.randomUUID();
    const r = await this.pool.query(
      `INSERT INTO referral.payouts (id, user_id, amount, status, created_at)
       VALUES ($1, $2, $3, 'PENDING', now()) RETURNING *`,
      [id, userId, dto.amount],
    );
    return r.rows[0];
  }

  async approvePayout(id: string) {
    const r = await this.pool.query(
      `UPDATE referral.payouts SET status = 'APPROVED', approved_at = now() WHERE id = $1 AND status = 'PENDING' RETURNING *`,
      [id],
    );
    if (r.rows.length === 0) throw new BadRequestException('Payout not found or not pending');
    return r.rows[0];
  }

  async rejectPayout(id: string, reason: string) {
    const r = await this.pool.query(
      `UPDATE referral.payouts
        SET status = 'REJECTED', reviewed_at = now(),
            metadata = jsonb_set(COALESCE(metadata, '{}'::jsonb), '{reason}', to_jsonb($2::text))
        WHERE id = $1 AND status NOT IN ('PAID', 'CANCELLED', 'REJECTED')
        RETURNING *`,
      [id, reason],
    );
    if (r.rows.length === 0) throw new BadRequestException('Payout not found or not in a rejectable state');
    return r.rows[0];
  }

  async processPayout(id: string) {
    const r = await this.pool.query(
      `UPDATE referral.payouts SET status = 'PROCESSING', processed_at = now()
        WHERE id = $1 AND status IN ('APPROVED', 'FAILED', 'UNDER_REVIEW') RETURNING *`,
      [id],
    );
    if (r.rows.length === 0) throw new BadRequestException('Payout not found or not in a processable state');
    return r.rows[0];
  }

  async reconcilePayout(
    id: string,
    payload: { provider_reference?: string; net_amount?: number; fee_amount?: number },
  ) {
    const client = await this.pool.connect();
    try {
      await client.query('BEGIN');

      const payoutRes = await client.query(
        `SELECT requested_amount FROM referral.payouts WHERE id = $1 FOR UPDATE`,
        [id],
      );
      if (payoutRes.rows.length === 0) throw new BadRequestException('Payout not found');

      const requested = Number(payoutRes.rows[0].requested_amount);
      const fee = payload.fee_amount ?? 0;
      const net = payload.net_amount ?? requested - fee;

      const upd = await client.query(
        `UPDATE referral.payouts
          SET status = 'PAID', paid_at = now(),
              provider_reference = $2, fee_amount = $3, net_amount = $4
          WHERE id = $1 AND status IN ('PROCESSING', 'APPROVED', 'UNDER_REVIEW', 'FAILED')
          RETURNING *`,
        [id, payload.provider_reference ?? null, fee, net],
      );
      if (upd.rows.length === 0) {
        throw new BadRequestException('Payout not found or not in a reconcilable state');
      }

      await client.query(
        `UPDATE referral.commission_ledger cl
          SET status = 'PAID', paid_at = now(), updated_at = now()
          WHERE id IN (SELECT commission_id FROM referral.payout_items WHERE payout_id = $1)`,
        [id],
      );

      await client.query(
        `WITH agg AS (
          SELECT cl.recipient_user_id AS uid, SUM(pi.amount) AS amt
          FROM referral.payout_items pi
          JOIN referral.commission_ledger cl ON cl.id = pi.commission_id
          WHERE pi.payout_id = $1
          GROUP BY cl.recipient_user_id
        )
        UPDATE referral.affiliate_wallets w
          SET available_balance = available_balance - agg.amt,
              paid_balance = paid_balance + agg.amt,
              updated_at = now()
        FROM agg
        WHERE w.user_id = agg.uid AND w.currency = 'USD'`,
        [id],
      );

      await client.query('COMMIT');
      return upd.rows[0];
    } catch (err) {
      await client.query('ROLLBACK');
      throw err;
    } finally {
      client.release();
    }
  }

  async retryPayout(id: string) {
    const r = await this.pool.query(
      `UPDATE referral.payouts SET status = 'UNDER_REVIEW'
        WHERE id = $1 AND status = 'FAILED' RETURNING *`,
      [id],
    );
    if (r.rows.length === 0) throw new BadRequestException('Payout not found or not in FAILED state');
    return r.rows[0];
  }

  async cancelPayout(id: string, reason: string) {
    const r = await this.pool.query(
      `UPDATE referral.payouts
        SET status = 'CANCELLED', cancelled_at = now(),
            metadata = jsonb_set(COALESCE(metadata, '{}'::jsonb), '{reason}', to_jsonb($2::text))
        WHERE id = $1 AND status NOT IN ('PAID', 'CANCELLED', 'REJECTED')
        RETURNING *`,
      [id, reason],
    );
    if (r.rows.length === 0) throw new BadRequestException('Payout not found or not in a cancellable state');
    return r.rows[0];
  }

  async listAll(page = 1, limit = 20) {
    const offset = (page - 1) * limit;
    const [data, count] = await Promise.all([
      this.pool.query(
        `SELECT p.*, u.email as user_email
         FROM referral.payouts p JOIN iam.users u ON p.user_id = u.id
         ORDER BY p.created_at DESC LIMIT $1 OFFSET $2`,
        [limit, offset],
      ),
      this.pool.query('SELECT count(*) as total FROM referral.payouts'),
    ]);
    return { items: data.rows, total: parseInt(count.rows[0].total, 10), page, limit };
  }

  async getStats() {
    const r = await this.pool.query(
      `SELECT count(*) as total,
              count(CASE WHEN status = 'PENDING' THEN 1 END) as pending,
              count(CASE WHEN status = 'APPROVED' THEN 1 END) as approved,
              count(CASE WHEN status = 'REJECTED' THEN 1 END) as rejected,
              COALESCE(SUM(CASE WHEN status = 'PENDING' THEN amount ELSE 0 END), 0) as pending_amount,
              COALESCE(SUM(CASE WHEN status = 'APPROVED' THEN amount ELSE 0 END), 0) as approved_amount
       FROM referral.payouts`,
    );
    return r.rows[0];
  }
}
