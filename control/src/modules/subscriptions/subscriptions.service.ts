import { Injectable, Inject } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';

@Injectable()
export class SubscriptionsService {
  constructor(@Inject(DB_POOL) private pool: Pool) {}

  async findByUser(userId: string) {
    const r = await this.pool.query(
      `SELECT s.*, p.name as plan_name, p.code as plan_code
       FROM billing.subscriptions s JOIN control.plans p ON s.plan_id = p.id
       WHERE s.user_id = $1 ORDER BY s.created_at DESC`, [userId],
    );
    return r.rows;
  }

  async create(userId: string, dto: { planId: string; strategyIds: string }) {
    const id = crypto.randomUUID();
    const r = await this.pool.query(
      `INSERT INTO billing.subscriptions (id, user_id, plan_id, status, billing_cycle, current_period_start, current_period_end)
       VALUES ($1, $2, $3, 'TRIAL', 'MONTHLY', now(), now() + interval '14 days')
       RETURNING *`,
      [id, userId, dto.planId],
    );
    return r.rows[0];
  }
}
