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
