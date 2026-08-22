import { Injectable, BadRequestException, Inject } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';

@Injectable()
export class ReferralsService {
  constructor(@Inject(DB_POOL) private pool: Pool) {}

  async getReferralCode(userId: string): Promise<string> {
    let r = await this.pool.query(
      `SELECT code FROM referral.referral_codes WHERE user_id = $1 AND active = true LIMIT 1`,
      [userId],
    );
    if (r.rows.length === 0) {
      const code = 'PAT-' + userId.replace(/-/g, '').toUpperCase().substring(0, 32);
      await this.pool.query(
        `INSERT INTO referral.referral_codes (id, user_id, code, active, created_at)
         VALUES (gen_random_uuid(), $1, $2, true, now())
         ON CONFLICT DO NOTHING`,
        [userId, code],
      );
      r = await this.pool.query(
        `SELECT code FROM referral.referral_codes WHERE user_id = $1 AND active = true LIMIT 1`,
        [userId],
      );
    }
    return r.rows[0]?.code || '';
  }

  async getReferralNetwork(userId: string) {
    const direct = await this.pool.query(
      `SELECT r.child_user_id, u.email, u.full_name, r.level, r.created_at
       FROM referral.referral_relationships r
       JOIN iam.users u ON r.child_user_id = u.id
       WHERE r.parent_user_id = $1
       ORDER BY r.level, r.created_at`,
      [userId],
    );
    return { referrals: direct.rows, count: direct.rows.length };
  }

  async getCommissions(userId: string) {
    const r = await this.pool.query(
      `SELECT c.*, u.email as source_email
       FROM referral.commission_ledger c
       LEFT JOIN iam.users u ON c.source_user_id = u.id
       WHERE c.recipient_user_id = $1
       ORDER BY c.created_at DESC LIMIT 50`,
      [userId],
    );
    return r.rows;
  }
}
