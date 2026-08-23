import { BadRequestException, Injectable, Inject, NotFoundException, Logger } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';
import { planPolicyFromRow, validateStrategySelection } from './entitlement-policy';
import { BillingService } from '../billing/billing.service';

interface ProviderConfig {
  provider: string | null;
  configured: boolean;
  note: string;
  detected_from_env?: boolean;
}

@Injectable()
export class SubscriptionsService {
  private logger = new Logger(SubscriptionsService.name);
  constructor(
    @Inject(DB_POOL) private pool: Pool,
    private billingService: BillingService,
  ) {}

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
      `SELECT code, allowed_strategies, max_active_strategy_slots, billing_enabled,
              monthly_price, annual_price
       FROM control.plans WHERE id = $1 AND status = 'ACTIVE'`, [dto.planId],
    );
    if (!plan.rows[0]) throw new NotFoundException('Active plan not found');
    if (plan.rows[0].code !== 'FREE' && !plan.rows[0].billing_enabled) {
      throw new BadRequestException('Plan is not available for new subscriptions');
    }
    const billingInterval = dto.billingInterval ?? 'MONTHLY';
    if (billingInterval === 'ANNUAL' && plan.rows[0].annual_price === null) {
      throw new BadRequestException('Annual billing is not available for this plan');
    }
    const requested = dto.selectedStrategies ?? (dto.strategyIds ? dto.strategyIds.split(',').map((s) => s.trim()) : ['STANDARD_SCALPING']);
    const decision = validateStrategySelection(planPolicyFromRow(plan.rows[0]), requested);
    if (!decision.allowed) throw new BadRequestException(decision.reason);

    const id = crypto.randomUUID();
    const r = await this.pool.query(
      `INSERT INTO billing.subscriptions
       (id, user_id, plan_id, status, billing_interval, billing_period_start, billing_period_end, selected_strategies)
       VALUES ($1, $2, $3, $4, $5, now(), now() + CASE WHEN $5 = 'ANNUAL' THEN interval '1 year' ELSE interval '1 month' END, $6::jsonb)
       RETURNING *`,
      [id, userId, dto.planId, plan.rows[0].code === 'FREE' ? 'ACTIVE' : 'INCOMPLETE', billingInterval, JSON.stringify(decision.selected)],
    );
    const subscription = r.rows[0];

    // Generate a branded invoice for the subscription period. Failures here must
    // never break subscription creation — they are logged and retried via the
    // billing/invoices/generate endpoint or the payment webhook.
    try {
      await this.billingService.generateInvoiceForSubscription(id, userId, {
        markPaid: plan.rows[0].code === 'FREE',
      });
    } catch (e) {
      this.logger.warn(`Invoice generation skipped for subscription ${id}: ${e instanceof Error ? e.message : e}`);
    }

    return subscription;
  }

  async getEntitlements(userId: string) {
    const r = await this.pool.query(
      `SELECT p.code, p.name, p.annual_price, p.monthly_price, p.visible,
              p.max_active_strategy_slots, p.allowed_strategies, COALESCE(NULLIF(s.selected_strategies, '[]'::jsonb), p.allowed_strategies) as selected_strategies,
              COALESCE(jsonb_object_agg(pe.entitlement_key, pe.entitlement_value)
                FILTER (WHERE pe.entitlement_key IS NOT NULL), '{}'::jsonb) AS entitlements
       FROM billing.subscriptions s JOIN control.plans p ON p.id = s.plan_id
       LEFT JOIN control.plan_entitlements pe ON pe.plan_id = p.id
       WHERE s.user_id = $1 AND s.status IN ('ACTIVE','TRIAL','GRACE','CANCEL_AT_PERIOD_END')
       GROUP BY p.code, p.name, p.annual_price, p.monthly_price, p.visible,
                p.max_active_strategy_slots, p.allowed_strategies, COALESCE(NULLIF(s.selected_strategies, '[]'::jsonb), p.allowed_strategies)
       ORDER BY MAX(s.created_at) DESC LIMIT 1`, [userId],
    );
    return r.rows[0] ?? { code: 'FREE', selected_strategies: ['STANDARD_SCALPING'], entitlements: {} };
  }
}
