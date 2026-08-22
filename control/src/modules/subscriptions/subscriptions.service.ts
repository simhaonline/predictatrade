import { BadRequestException, Injectable, Inject, NotFoundException } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';
import { planPolicyFromRow, validateStrategySelection } from './entitlement-policy';

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

  async create(userId: string, dto: { planId: string; strategyIds?: string; selectedStrategies?: string[]; billingInterval?: 'MONTHLY' | 'ANNUAL' }) {
    const plan = await this.pool.query(
      `SELECT code, allowed_strategies, max_active_strategy_slots, billing_enabled
       FROM control.plans WHERE id = $1 AND status = 'ACTIVE'`, [dto.planId],
    );
    if (!plan.rows[0]) throw new NotFoundException('Active plan not found');
    if (plan.rows[0].code !== 'FREE' && !plan.rows[0].billing_enabled) {
      throw new BadRequestException('Plan is not available for new subscriptions');
    }
    const requested = dto.selectedStrategies ?? (dto.strategyIds ? dto.strategyIds.split(',').map((s) => s.trim()) : ['STANDARD_SCALPING']);
    const decision = validateStrategySelection(planPolicyFromRow(plan.rows[0]), requested);
    if (!decision.allowed) throw new BadRequestException(decision.reason);

    const id = crypto.randomUUID();
    const r = await this.pool.query(
      `INSERT INTO billing.subscriptions
       (id, user_id, plan_id, status, billing_period_start, billing_period_end, selected_strategies)
       VALUES ($1, $2, $3, $4, now(), now() + CASE WHEN $5 = 'ANNUAL' THEN interval '1 year' ELSE interval '1 month' END, $6::jsonb)
       RETURNING *`,
      [id, userId, dto.planId, plan.rows[0].code === 'FREE' ? 'ACTIVE' : 'INCOMPLETE', dto.billingInterval ?? 'MONTHLY', JSON.stringify(decision.selected)],
    );
    return r.rows[0];
  }

  async getEntitlements(userId: string) {
    const r = await this.pool.query(
      `SELECT p.code, p.name, p.annual_price, p.monthly_price, p.visible,
              p.max_active_strategy_slots, p.allowed_strategies, s.selected_strategies,
              COALESCE(jsonb_object_agg(pe.entitlement_key, pe.entitlement_value)
                FILTER (WHERE pe.entitlement_key IS NOT NULL), '{}'::jsonb) AS entitlements
       FROM billing.subscriptions s JOIN control.plans p ON p.id = s.plan_id
       LEFT JOIN control.plan_entitlements pe ON pe.plan_id = p.id
       WHERE s.user_id = $1 AND s.status IN ('ACTIVE','TRIAL','GRACE','CANCEL_AT_PERIOD_END')
       GROUP BY p.code, p.name, p.annual_price, p.monthly_price, p.visible,
                p.max_active_strategy_slots, p.allowed_strategies, s.selected_strategies
       ORDER BY s.created_at DESC LIMIT 1`, [userId],
    );
    return r.rows[0] ?? { code: 'FREE', selected_strategies: ['STANDARD_SCALPING'], entitlements: {} };
  }
}
