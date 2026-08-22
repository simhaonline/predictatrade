import { Injectable, NotFoundException, Inject } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';

@Injectable()
export class PlansService {
  constructor(@Inject(DB_POOL) private pool: Pool) {}

  async listActive() {
    const r = await this.pool.query(
      `SELECT p.id, p.code, p.name, p.description, p.monthly_price, p.annual_price, p.setup_fee,
              p.status, p.visible, p.legacy, p.billing_enabled,
              p.max_active_strategy_slots, p.allowed_strategies,
              (SELECT jsonb_agg(jsonb_build_object('key', pe.entitlement_key, 'value', pe.entitlement_value))
               FROM control.plan_entitlements pe WHERE pe.plan_id = p.id) as entitlements
       FROM control.plans p WHERE p.status = 'ACTIVE' AND p.visible = TRUE ORDER BY p.sort_order, p.monthly_price`,
    );
    return r.rows;
  }

  async findById(id: string) {
    const r = await this.pool.query('SELECT * FROM control.plans WHERE id = $1', [id]);
    if (r.rows.length === 0) throw new NotFoundException('Plan not found');
    return r.rows[0];
  }
}
