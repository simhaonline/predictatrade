import { Injectable, Inject, NotFoundException, BadRequestException } from '@nestjs/common';
import { Pool, PoolClient } from 'pg';
import Decimal from 'decimal.js';
import { DB_POOL } from '../../common/database.module';
import { CommissionEngine } from './commission-engine';

type CommissionStatus =
  | 'PENDING' | 'CLEARED' | 'AVAILABLE' | 'PAID'
  | 'CANCELLED' | 'REVERSED' | 'CHARGEBACK' | 'FRAUD_HOLD';

// Allowed state-machine transitions. Terminal states have no outgoing edges.
const TRANSITION_MATRIX: Record<string, string[]> = {
  PENDING: ['CLEARED', 'FRAUD_HOLD', 'REVERSED', 'CANCELLED'],
  CLEARED: ['AVAILABLE', 'FRAUD_HOLD', 'REVERSED', 'CANCELLED'],
  AVAILABLE: ['PAID', 'FRAUD_HOLD', 'REVERSED', 'CANCELLED'],
  FRAUD_HOLD: ['AVAILABLE', 'REVERSED', 'CANCELLED'],
  PAID: [],
  REVERSED: [],
  CANCELLED: [],
  CHARGEBACK: [],
};

// Maps a status to the wallet bucket that holds its balance.
function bucketForStatus(status: string): string {
  switch (status) {
    case 'PENDING': return 'pending_balance';
    case 'CLEARED': return 'cleared_balance';
    case 'AVAILABLE': return 'available_balance';
    case 'PAID': return 'paid_balance';
    case 'FRAUD_HOLD': return 'on_hold_balance';
    case 'REVERSED':
    case 'CANCELLED': return 'reversed_balance';
    default: return 'pending_balance';
  }
}

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

    if (result.commissions.length === 0) return { credited: 0 };

    // M1 fix: idempotency check + all ledger inserts run inside ONE transaction
    // so a failure mid-loop cannot leave a partially-credited, inconsistent
    // ledger. A concurrent caller hits the unique idempotency_key and is treated
    // as a benign duplicate.
    const client = await this.pool.connect();
    try {
      await client.query('BEGIN');
      const dup = await client.query(
        'SELECT 1 FROM referral.commission_ledger WHERE idempotency_key = $1 LIMIT 1',
        [idemKey],
      );
      if (dup.rows.length > 0) {
        await client.query('COMMIT');
        return { credited: 0, skipped: true };
      }
      for (const c of result.commissions) {
        const levelRule = ratesByLevel.get(c.level);
        await client.query(
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
      await client.query('COMMIT');
      return { credited: result.commissions.length };
    } catch (e: any) {
      await client.query('ROLLBACK');
      if (e?.code === '23505') return { credited: 0, skipped: true };
      throw e;
    } finally {
      client.release();
    }
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

  /**
   * Core state-machine transition: validates the transition, moves the
   * commission amount between affiliate_wallet buckets consistently, and
   * stamps the relevant timestamp. Runs inside the supplied transaction client.
   * `amountOverride` lets reversals move only a partial amount.
   */
  private async transitionLedgerAndWallet(
    client: PoolClient,
    id: string,
    target: CommissionStatus,
    actorId: string,
    reason?: string,
    amountOverride?: Decimal | number,
  ) {
    const cur = await client.query(
      'SELECT * FROM referral.commission_ledger WHERE id = $1 FOR UPDATE',
      [id],
    );
    if (cur.rows.length === 0) throw new NotFoundException('Commission not found');
    const row = cur.rows[0];
    const oldStatus: string = row.status;
    const allowed = TRANSITION_MATRIX[oldStatus] ?? [];
    if (!allowed.includes(target)) {
      throw new BadRequestException(`Illegal transition ${oldStatus} -> ${target}`);
    }

    const amt = amountOverride != null ? new Decimal(amountOverride) : new Decimal(row.commission_amount);
    const srcBucket = bucketForStatus(oldStatus);
    const destBucket =
      target === 'FRAUD_HOLD' ? 'on_hold_balance'
        : (target === 'REVERSED' || target === 'CANCELLED') ? 'reversed_balance'
          : bucketForStatus(target);

    await client.query(
      `INSERT INTO referral.affiliate_wallets (user_id, currency)
       VALUES ($1, $2) ON CONFLICT (user_id, currency) DO NOTHING`,
      [row.recipient_user_id, row.currency],
    );

    const setParts = [`${srcBucket} = ${srcBucket} - $1`];
    if (destBucket !== srcBucket) setParts.push(`${destBucket} = ${destBucket} + $1`);
    setParts.push('updated_at = now()');
    await client.query(
      `UPDATE referral.affiliate_wallets SET ${setParts.join(', ')}
       WHERE user_id = $2 AND currency = $3`,
      [amt.toString(), row.recipient_user_id, row.currency],
    );

    const r = await client.query(
      `UPDATE referral.commission_ledger
       SET status = $1,
           cleared_at   = CASE WHEN $1 = 'CLEARED'   THEN now() ELSE cleared_at END,
           available_at = CASE WHEN $1 = 'AVAILABLE' THEN now() ELSE available_at END,
           paid_at      = CASE WHEN $1 = 'PAID'      THEN now() ELSE paid_at END,
           reversed_at  = CASE WHEN $1 = 'REVERSED'  THEN now() ELSE reversed_at END,
           reversal_reason = CASE WHEN $1 = 'REVERSED' THEN $2 ELSE reversal_reason END,
           reversed_by     = CASE WHEN $1 = 'REVERSED' THEN $3 ELSE reversed_by END,
           updated_at = now()
       WHERE id = $4
       RETURNING *`,
      [target, reason ?? null, actorId ?? null, id],
    );
    return r.rows[0];
  }

  /** Generic guarded transition used by the explicit lifecycle endpoints. */
  async transitionCommission(id: string, target: CommissionStatus, actorId: string, reason?: string) {
    const client = await this.pool.connect();
    try {
      await client.query('BEGIN');
      const row = await this.transitionLedgerAndWallet(client, id, target, actorId, reason);
      await client.query('COMMIT');
      return row;
    } catch (e) {
      await client.query('ROLLBACK');
      throw e;
    } finally {
      client.release();
    }
  }

  async holdCommission(id: string, reason: string, actorId: string) {
    return this.transitionCommission(id, 'FRAUD_HOLD', actorId, reason);
  }

  async releaseCommission(id: string, actorId: string) {
    return this.transitionCommission(id, 'AVAILABLE', actorId);
  }

  async reverseCommission(id: string, reason: string, actorId: string, amount?: number) {
    const client = await this.pool.connect();
    try {
      await client.query('BEGIN');
      const cur = await client.query(
        'SELECT * FROM referral.commission_ledger WHERE id = $1 FOR UPDATE',
        [id],
      ) as { rows: any[] };
      if (cur.rows.length === 0) throw new NotFoundException('Commission not found');
      const row = cur.rows[0];
      const full = new Decimal(row.commission_amount);
      const revAmount = amount != null ? new Decimal(amount) : full;
      if (!revAmount.gt(0) || revAmount.gt(full)) {
        throw new BadRequestException('Reversal amount must be > 0 and <= commission amount');
      }
      const type = revAmount.lt(full) ? 'PARTIAL_REVERSAL' : 'REVERSAL';
      let updated: any = null;
      if (revAmount.lt(full)) {
        // M6 fix: partial reversal must NOT flip the whole commission to
        // REVERSED (that would mis-attribute the full commission_amount to the
        // reversed bucket in summaries). Keep the original lifecycle status and
        // move only the reversed amount into reversed_balance.
        const srcBucket = bucketForStatus(row.status);
        await client.query(
          `UPDATE referral.affiliate_wallets
             SET ${srcBucket} = ${srcBucket} - $1,
                 reversed_balance = reversed_balance + $1,
                 updated_at = now()
           WHERE user_id = $2 AND currency = $3`,
          [revAmount.toString(), row.recipient_user_id, row.currency],
        );
      } else {
        updated = await this.transitionLedgerAndWallet(
          client, id, 'REVERSED', actorId, reason, revAmount,
        );
      }
      await client.query(
        `INSERT INTO referral.commission_adjustments
           (id, original_commission_id, adjustment_type, amount, currency, reason, adjusted_by, created_at)
         VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, now())`,
        [id, type, revAmount.negated().toString(), row.currency, reason, actorId],
      );
      await client.query('COMMIT');
      return updated;
    } catch (e) {
      await client.query('ROLLBACK');
      throw e;
    } finally {
      client.release();
    }
  }

  async adjustCommission(id: string, amount: number, reason: string, actorId: string) {
    const delta = new Decimal(amount);
    if (!delta.isFinite()) throw new BadRequestException('Invalid adjustment amount');
    const client = await this.pool.connect();
    try {
      await client.query('BEGIN');
      const cur = await client.query(
        'SELECT * FROM referral.commission_ledger WHERE id = $1 FOR UPDATE',
        [id],
      );
      if (cur.rows.length === 0) throw new NotFoundException('Commission not found');
      const row = cur.rows[0];
      const newAmt = new Decimal(row.commission_amount).plus(delta);
      if (newAmt.lt(0)) throw new BadRequestException('Adjustment would make commission negative');

      await client.query(
        `INSERT INTO referral.commission_adjustments
           (id, original_commission_id, adjustment_type, amount, currency, reason, adjusted_by, created_at)
         VALUES (gen_random_uuid(), $1, 'MANUAL_ADJUSTMENT', $2, $3, $4, $5, now())`,
        [id, delta.toString(), row.currency, reason, actorId],
      );
      await client.query(
        'UPDATE referral.commission_ledger SET commission_amount = $1, updated_at = now() WHERE id = $2',
        [newAmt.toString(), id],
      );

      const bucket = bucketForStatus(row.status);
      await client.query(
        `INSERT INTO referral.affiliate_wallets (user_id, currency)
         VALUES ($1, $2) ON CONFLICT (user_id, currency) DO NOTHING`,
        [row.recipient_user_id, row.currency],
      );
      await client.query(
        `UPDATE referral.affiliate_wallets SET ${bucket} = ${bucket} + $1, updated_at = now()
         WHERE user_id = $2 AND currency = $3`,
        [delta.toString(), row.recipient_user_id, row.currency],
      );
      await client.query('COMMIT');
      return { id, new_amount: newAmt.toString() };
    } catch (e) {
      await client.query('ROLLBACK');
      throw e;
    } finally {
      client.release();
    }
  }

  /** Admin-triggered bulk lifecycle. NOT auto-run. Returns transition counts. */
  async clearEligible() {
    const client = await this.pool.connect();
    try {
      await client.query('BEGIN');

      const pend = await client.query(
        `SELECT id, recipient_user_id, currency, commission_amount
         FROM referral.commission_ledger
         WHERE status = 'PENDING' AND created_at < now() - interval '14 days'
         FOR UPDATE`,
      );
      const clearedIds = pend.rows.map((r) => r.id);
      if (clearedIds.length) {
        await client.query(
          `UPDATE referral.commission_ledger
           SET status = 'CLEARED', cleared_at = now(), updated_at = now()
           WHERE id = ANY($1)`,
          [clearedIds],
        );
        const agg = new Map<string, Decimal>();
        for (const r of pend.rows) {
          const k = `${r.recipient_user_id}|${r.currency}`;
          agg.set(k, (agg.get(k) ?? new Decimal(0)).plus(new Decimal(r.commission_amount)));
        }
        for (const [k, amt] of agg) {
          const [uid, cur2] = k.split('|');
          await client.query(
            `INSERT INTO referral.affiliate_wallets (user_id, currency)
             VALUES ($1, $2) ON CONFLICT (user_id, currency) DO NOTHING`,
            [uid, cur2],
          );
          await client.query(
            `UPDATE referral.affiliate_wallets
             SET pending_balance = pending_balance - $1,
                 cleared_balance = cleared_balance + $1,
                 updated_at = now()
             WHERE user_id = $2 AND currency = $3`,
            [amt.toString(), uid, cur2],
          );
        }
      }

      const clr = await client.query(
        `SELECT id, recipient_user_id, currency, commission_amount
         FROM referral.commission_ledger
         WHERE status = 'CLEARED' AND cleared_at < now() - interval '30 days'
         FOR UPDATE`,
      );
      const availIds = clr.rows.map((r) => r.id);
      if (availIds.length) {
        await client.query(
          `UPDATE referral.commission_ledger
           SET status = 'AVAILABLE', available_at = now(), updated_at = now()
           WHERE id = ANY($1)`,
          [availIds],
        );
        const agg = new Map<string, Decimal>();
        for (const r of clr.rows) {
          const k = `${r.recipient_user_id}|${r.currency}`;
          agg.set(k, (agg.get(k) ?? new Decimal(0)).plus(new Decimal(r.commission_amount)));
        }
        for (const [k, amt] of agg) {
          const [uid, cur2] = k.split('|');
          await client.query(
            `INSERT INTO referral.affiliate_wallets (user_id, currency)
             VALUES ($1, $2) ON CONFLICT (user_id, currency) DO NOTHING`,
            [uid, cur2],
          );
          await client.query(
            `UPDATE referral.affiliate_wallets
             SET cleared_balance = cleared_balance - $1,
                 available_balance = available_balance + $1,
                 updated_at = now()
             WHERE user_id = $2 AND currency = $3`,
            [amt.toString(), uid, cur2],
          );
        }
      }

      await client.query('COMMIT');
      return { cleared: clearedIds.length, available: availIds.length };
    } catch (e) {
      await client.query('ROLLBACK');
      throw e;
    } finally {
      client.release();
    }
  }

  /** Persist commission rule changes (base_rate, active, effective_until). */
  async updateRule(
    ruleId: string,
    payload: { base_rate?: number; active?: boolean; effective_until?: string },
  ) {
    const sets: string[] = [];
    const params: unknown[] = [];
    let i = 1;
    if (payload.base_rate !== undefined) { sets.push(`base_rate = $${i++}`); params.push(payload.base_rate); }
    if (payload.active !== undefined) { sets.push(`active = $${i++}`); params.push(payload.active); }
    if (payload.effective_until !== undefined) { sets.push(`effective_until = $${i++}`); params.push(payload.effective_until); }
    if (sets.length === 0) throw new BadRequestException('No rule fields to update');
    params.push(ruleId);
    const r = await this.pool.query(
      `UPDATE referral.commission_rules SET ${sets.join(', ')} WHERE id = $${i} RETURNING *`,
      params,
    );
    if (r.rows.length === 0) throw new NotFoundException('Commission rule not found');
    return r.rows[0];
  }

  /** List commission rules (admin). */
  async listRules() {
    const r = await this.pool.query(
      `SELECT id, plan_id, level, base_rate, effective_from, effective_until, active, rule_version
       FROM referral.commission_rules ORDER BY plan_id, level`,
    );
    return r.rows;
  }
}
