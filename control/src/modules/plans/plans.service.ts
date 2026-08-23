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
               FROM control.plan_entitlements pe WHERE pe.plan_id = p.id) as entitlements,
              COALESCE((SELECT jsonb_object_agg(r.level::text, r.base_rate ORDER BY r.level)
               FROM referral.commission_rules r
               WHERE r.plan_id = p.id AND r.active = TRUE
                 AND r.effective_from <= now()
                 AND (r.effective_until IS NULL OR r.effective_until > now())
                 AND r.effective_from = (
                   SELECT max(r2.effective_from) FROM referral.commission_rules r2
                   WHERE r2.plan_id = r.plan_id AND r2.level = r.level
                     AND r2.active = TRUE AND r2.effective_from <= now()
                 )), '{}'::jsonb) as referral_rates,
              (SELECT jsonb_agg(jsonb_build_object(
                 'purchase_type', pr.purchase_type,
                 'multiplier', pr.multiplier,
                 'max_referral_level', pr.max_referral_level,
                 'rule_version', pr.rule_version
               ) ORDER BY pr.purchase_type)
               FROM referral.purchase_commission_rules pr
               WHERE pr.active = TRUE AND pr.effective_from <= now()
                 AND (pr.effective_until IS NULL OR pr.effective_until > now())
                 AND pr.effective_from = (
                   SELECT max(pr2.effective_from) FROM referral.purchase_commission_rules pr2
                   WHERE pr2.purchase_type = pr.purchase_type AND pr2.active = TRUE
                     AND pr2.effective_from <= now()
                 )) as referral_event_rules
       FROM control.plans p WHERE p.status = 'ACTIVE' AND p.visible = TRUE ORDER BY p.sort_order, p.monthly_price`,
    );
    return r.rows.map((row) => ({
      ...row,
      annual_savings_percent: row.monthly_price && row.annual_price
        ? Number((100 - (Number(row.annual_price) / (Number(row.monthly_price) * 12)) * 100).toFixed(2))
        : null,
      referral_eligible: ['STANDARD', 'PRO', 'ELITE'].includes(row.code),
    }));
  }

  async findById(id: string) {
    const r = await this.pool.query('SELECT * FROM control.plans WHERE id = $1', [id]);
    if (r.rows.length === 0) throw new NotFoundException('Plan not found');
    return r.rows[0];
  }

  private static readonly EDITABLE_FIELDS = [
    'name',
    'monthly_price',
    'annual_price',
    'setup_fee',
    'max_active_strategy_slots',
    'allowed_strategies',
    'status',
    'billing_enabled',
    'visible',
    'grace_period_days',
  ] as const;

  async update(id: string, body: Record<string, any>) {
    const sets: string[] = [];
    const values: any[] = [];
    let idx = 1;

    for (const field of PlansService.EDITABLE_FIELDS) {
      if (!(field in body)) continue;
      let value = body[field];
      if (field === 'allowed_strategies' && value !== null && typeof value === 'object') {
        value = JSON.stringify(value);
      }
      sets.push(`${field} = $${idx}`);
      values.push(value);
      idx++;
    }

    if (sets.length === 0) {
      return this.findById(id);
    }

    sets.push(`updated_at = now()`);
    values.push(id);

    const r = await this.pool.query(
      `UPDATE control.plans SET ${sets.join(', ')} WHERE id = $${idx} RETURNING *`,
      values,
    );
    if (r.rows.length === 0) throw new NotFoundException('Plan not found');
    return r.rows[0];
  }
}
