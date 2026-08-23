import { Injectable, Inject } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';
import { CommissionEngine } from './commission-engine';

@Injectable()
export class CommissionsService {
  constructor(@Inject(DB_POOL) private pool: Pool) {}

  /**
   * Credit referral commissions when a referred user is issued a license.
   * Reads the active commission rules + purchase rules for the plan, runs the
   * CommissionEngine, and writes ledger entries idempotently (keyed by license).
   */
  async creditReferralForLicense(
    sourceUserId: string,
    planId: string,
    licenseId: string,
    commissionableAmount: number,
    currency: string,
  ) {
    const chain = await this.pool.query(
      `SELECT parent_user_id, level
       FROM referral.referral_relationships
       WHERE child_user_id = $1 AND level BETWEEN 1 AND 5
       ORDER BY level`,
      [sourceUserId],
    );
    if (chain.rows.length === 0) return { credited: 0 };

    const sponsorChain: string[] = [];
    chain.rows.forEach((r) => {
      sponsorChain[Number(r.level) - 1] = r.parent_user_id;
    });

    const rules = await this.pool.query(
      `SELECT id, level, base_rate
       FROM referral.commission_rules
       WHERE plan_id = $1 AND active = true
         AND (effective_from IS NULL OR effective_from <= now())
         AND (effective_until IS NULL OR effective_until >= now())
       ORDER BY level, effective_from DESC`,
      [planId],
    );
    const ratesByLevel = new Map<number, { id: string; rate: number }>();
    for (const row of rules.rows) {
      const lvl = Number(row.level);
      if (!ratesByLevel.has(lvl)) {
        ratesByLevel.set(lvl, { id: row.id, rate: Number(row.base_rate) });
      }
    }
    const baseRates = [1, 2, 3, 4, 5].map((l) => ratesByLevel.get(l)?.rate ?? 0);

    const pr = await this.pool.query(
      `SELECT id, purchase_type, multiplier, max_referral_level
       FROM referral.purchase_commission_rules
       WHERE purchase_type = 'FIRST_PURCHASE' AND active = true
         AND (effective_from IS NULL OR effective_from <= now())
         AND (effective_until IS NULL OR effective_until >= now())
       ORDER BY effective_from DESC LIMIT 1`,
    );
    if (pr.rows.length === 0) return { credited: 0 };
    const purchaseRule = pr.rows[0];

    const idemKey = `license:${licenseId}`;
    const dup = await this.pool.query(
      'SELECT 1 FROM referral.commission_ledger WHERE idempotency_key = $1 LIMIT 1',
      [idemKey],
    );
    if (dup.rows.length > 0) return { credited: 0, skipped: true };

    const engine = new CommissionEngine();
    engine.setBaseRates(planId, baseRates);
    engine.setPurchaseRule('FIRST_PURCHASE', {
      multiplier: Number(purchaseRule.multiplier),
      maxReferralLevel: Number(purchaseRule.max_referral_level),
    });

    const result = engine.calculate({
      planId,
      commissionableAmount,
      paymentNumber: 1,
      sponsorChain,
      sourceUserId,
      sourceSubscriptionId: '',
      purchaseId: licenseId,
      invoiceId: '',
      eventType: 'NEW_SUBSCRIPTION',
    });

    for (const c of result.commissions) {
      const levelRule = ratesByLevel.get(c.level);
      await this.pool.query(
        `INSERT INTO referral.commission_ledger (
          id, recipient_user_id, source_user_id, source_subscription_id, purchase_id, invoice_id,
          plan_id, plan_version, purchase_number, purchase_type, level,
          base_commission_rate, purchase_multiplier, effective_commission_rate,
          commissionable_amount, commission_amount, currency, status,
          commission_rule_id, purchase_rule_id, idempotency_key, created_at, updated_at
        ) VALUES (
          gen_random_uuid(), $1, $2, $3, $4, $5, $6, 1, $7, $8, $9,
          $10, $11, $12, $13, $14, $15, 'PENDING',
          $16, $17, $18, now(), now()
        )`,
        [
          c.recipientUserId,
          c.sourceUserId,
          c.sourceSubscriptionId || null,
          null,
          c.invoiceId || null,
          c.planId,
          c.purchaseNumber,
          c.purchaseType,
          c.level,
          c.baseCommissionRate.toString(),
          c.purchaseMultiplier.toString(),
          c.effectiveCommissionRate.toString(),
          c.commissionableAmount.toString(),
          c.commissionAmount.toString(),
          c.currency,
          levelRule?.id || null,
          purchaseRule.id,
          idemKey,
        ],
      );
    }

    return { credited: result.commissions.length };
  }

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
