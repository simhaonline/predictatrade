import { Injectable, Inject } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';

@Injectable()
export class CommissionsService {
  constructor(@Inject(DB_POOL) private pool: Pool) {}

  async findByRecipient(userId: string) {
    const r = await this.pool.query(
      `SELECT * FROM referral.commission_ledger WHERE recipient_user_id = $1 ORDER BY created_at DESC LIMIT 50`,
      [userId],
    );
    return r.rows;
  }

  async getSummary(userId: string) {
    const r = await this.pool.query(
      `SELECT COUNT(*) as total_entries,
              COALESCE(SUM(commission_amount), 0) as total_amount,
              COUNT(CASE WHEN status = 'PENDING' THEN 1 END) as pending_count,
              COALESCE(SUM(CASE WHEN status = 'PENDING' THEN commission_amount ELSE 0 END), 0) as pending_amount,
              COUNT(CASE WHEN status = 'CLEARED' THEN 1 END) as cleared_count,
              COALESCE(SUM(CASE WHEN status = 'CLEARED' THEN commission_amount ELSE 0 END), 0) as cleared_amount,
              COUNT(CASE WHEN status = 'AVAILABLE' THEN 1 END) as available_count,
              COALESCE(SUM(CASE WHEN status = 'AVAILABLE' THEN commission_amount ELSE 0 END), 0) as available_amount,
              COUNT(CASE WHEN status = 'PAID' THEN 1 END) as paid_count,
              COALESCE(SUM(CASE WHEN status = 'PAID' THEN commission_amount ELSE 0 END), 0) as paid_amount
       FROM referral.commission_ledger WHERE recipient_user_id = $1`, [userId],
    );
    return { ...r.rows[0], confirmed_count: r.rows[0]?.paid_count ?? 0, confirmed_amount: r.rows[0]?.paid_amount ?? 0 };
  }

  async listAll(page = 1, limit = 20) {
    const offset = (page - 1) * limit;
    const [data, count] = await Promise.all([
      this.pool.query(
        `SELECT c.*, ru.email as recipient_email, su.email as source_email
         FROM referral.commission_ledger c
         LEFT JOIN iam.users ru ON c.recipient_user_id = ru.id
         LEFT JOIN iam.users su ON c.source_user_id = su.id
         ORDER BY c.created_at DESC LIMIT $1 OFFSET $2`,
        [limit, offset],
      ),
      this.pool.query('SELECT count(*) as total FROM referral.commission_ledger'),
    ]);
    return { items: data.rows, total: parseInt(count.rows[0].total, 10), page, limit };
  }

  async getGlobalSummary() {
    const r = await this.pool.query(
      `SELECT count(*) as total_entries,
              COALESCE(SUM(commission_amount), 0) as total_amount,
              count(CASE WHEN status = 'PENDING' THEN 1 END) as pending_count,
              COALESCE(SUM(CASE WHEN status = 'PENDING' THEN commission_amount ELSE 0 END), 0) as pending_amount,
              count(CASE WHEN status = 'CLEARED' THEN 1 END) as cleared_count,
              COALESCE(SUM(CASE WHEN status = 'CLEARED' THEN commission_amount ELSE 0 END), 0) as cleared_amount,
              count(CASE WHEN status = 'AVAILABLE' THEN 1 END) as available_count,
              COALESCE(SUM(CASE WHEN status = 'AVAILABLE' THEN commission_amount ELSE 0 END), 0) as available_amount,
              count(CASE WHEN status = 'PAID' THEN 1 END) as paid_count,
              COALESCE(SUM(CASE WHEN status = 'PAID' THEN commission_amount ELSE 0 END), 0) as paid_amount,
              count(CASE WHEN status = 'REVERSED' THEN 1 END) as reversed_count,
              COALESCE(SUM(CASE WHEN status = 'REVERSED' THEN commission_amount ELSE 0 END), 0) as reversed_amount
       FROM referral.commission_ledger`,
    );
    return { ...r.rows[0], confirmed_count: r.rows[0]?.paid_count ?? 0, confirmed_amount: r.rows[0]?.paid_amount ?? 0 };
  }
}
