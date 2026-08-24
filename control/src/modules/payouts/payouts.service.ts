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

  /**
   * P0-CP2/CP3 fix: transactional payout request.
   * - writes `requested_amount` (the column that actually exists)
   * - validates amount against affiliate_wallets.available_balance
   * - creates referral.payout_items from CLEARED commissions (FIFO) so the
   *   completion path can mark ledger PAID and debit wallets exactly once
   * - idempotency_key honored (unique per user when supplied by client)
   */
  async requestPayout(
    userId: string,
    dto: { amount: number; method: string; destination: string; idempotency_key?: string; currency?: string },
  ) {
    const amount = Number(dto.amount);
    if (!Number.isFinite(amount) || amount <= 0) throw new BadRequestException('amount must be positive');
    const MIN_PAYOUT = 50;
    if (amount < MIN_PAYOUT) throw new BadRequestException(`minimum payout is ${MIN_PAYOUT}`);

    const currency = dto.currency || 'USD';
    const client = await this.pool.connect();
    try {
      await client.query('BEGIN');

      if (dto.idempotency_key) {
        const dup = await client.query(
          `SELECT id FROM referral.payouts WHERE user_id = $1 AND idempotency_key = $2`,
          [userId, dto.idempotency_key],
        );
        if (dup.rows.length > 0) {
          await client.query('COMMIT');
          return { ...(await this.findById(dup.rows[0].id)), duplicate: true };
        }
      }

      // Lock wallet and validate balance
      const walletRes = await client.query(
        `SELECT available_balance, currency FROM referral.affiliate_wallets
          WHERE user_id = $1 AND currency = $2 FOR UPDATE`,
        [userId, currency],
      );
      if (walletRes.rows.length === 0) throw new BadRequestException(`no ${currency} wallet for user`);
      const available = Number(walletRes.rows[0].available_balance);
      if (amount > available) {
        throw new BadRequestException(`requested ${amount} exceeds available balance ${available}`);
      }

      // Reserve CLEARED commissions FIFO to cover the requested amount
      const eligible = await client.query(
        `SELECT id, commission_amount FROM referral.commission_ledger
          WHERE recipient_user_id = $1 AND currency = $2 AND status = 'CLEARED'
          ORDER BY cleared_at ASC, created_at ASC
          FOR UPDATE`,
        [userId, currency],
      );
      let remaining = amount;
      const items: Array<{ id: string; amt: number }> = [];
      for (const row of eligible.rows) {
        if (remaining <= 0) break;
        const amt = Math.min(Number(row.commission_amount), remaining);
        items.push({ id: row.id, amt });
        remaining -= amt;
      }
      if (remaining > 0.00000001) {
        throw new BadRequestException('insufficient CLEARED commissions to cover payout');
      }

      const id = crypto.randomUUID();
      const inserted = await client.query(
        `INSERT INTO referral.payouts
           (id, user_id, requested_amount, approved_amount, currency, status,
            payout_method_id, metadata, idempotency_key)
         VALUES ($1, $2, $3, 0, $4, 'REQUESTED', NULL,
                 $5::jsonb, $6) RETURNING *`,
        [
          id,
          userId,
          amount,
          currency,
          JSON.stringify({ method: dto.method, destination: dto.destination }),
          dto.idempotency_key ?? null,
        ],
      );

      for (const item of items) {
        await client.query(
          `INSERT INTO referral.payout_items (payout_id, commission_id, amount)
           VALUES ($1, $2, $3)`,
          [id, item.id, item.amt],
        );
      }

      await client.query('COMMIT');
      return inserted.rows[0];
    } catch (err) {
      await client.query('ROLLBACK');
      throw err;
    } finally {
      client.release();
    }
  }

  private async findById(id: string) {
    const r = await this.pool.query(`SELECT * FROM referral.payouts WHERE id = $1`, [id]);
    return r.rows[0];
  }

  async approvePayout(id: string, approverId?: string) {
    const client = await this.pool.connect();
    try {
      await client.query('BEGIN');
      // approved_amount = sum of reserved items (exact decimal in SQL)
      const r = await client.query(
        `UPDATE referral.payouts p
            SET status = 'APPROVED', approved_at = now(), approved_by = $2::uuid,
                approved_amount = COALESCE((SELECT SUM(pi.amount)
                                              FROM referral.payout_items pi
                                             WHERE pi.payout_id = p.id), p.requested_amount)
          WHERE p.id = $1 AND p.status = 'REQUESTED'
          RETURNING *`,
        [id, approverId ?? null],
      );
      if (r.rows.length === 0) throw new BadRequestException('Payout not found or not pending');
      await client.query('COMMIT');
      return r.rows[0];
    } catch (err) {
      await client.query('ROLLBACK');
      throw err;
    } finally {
      client.release();
    }
  }

  async rejectPayout(id: string, reason: string) {
    const r = await this.pool.query(
      `UPDATE referral.payouts
        SET status = 'REJECTED', reviewed_at = now(),
            metadata = jsonb_set(COALESCE(metadata, '{}'::jsonb), '{reason}', to_jsonb($2::text))
        WHERE id = $1 AND status NOT IN ('PAID', 'CANCELLED', 'REJECTED')
        RETURNING *`,
      [id, reason],
    );
    if (r.rows.length === 0) throw new BadRequestException('Payout not found or not in a rejectable state');
    return r.rows[0];
  }

  async processPayout(id: string) {
    const r = await this.pool.query(
      `UPDATE referral.payouts SET status = 'PROCESSING', processed_at = now()
        WHERE id = $1 AND status IN ('APPROVED', 'FAILED', 'UNDER_REVIEW') RETURNING *`,
      [id],
    );
    if (r.rows.length === 0) throw new BadRequestException('Payout not found or not in a processable state');
    return r.rows[0];
  }

  async reconcilePayout(
    id: string,
    payload: { provider_reference?: string; net_amount?: number; fee_amount?: number },
  ) {
    const client = await this.pool.connect();
    try {
      await client.query('BEGIN');

      const payoutRes = await client.query(
        `SELECT requested_amount FROM referral.payouts WHERE id = $1 FOR UPDATE`,
        [id],
      );
      if (payoutRes.rows.length === 0) throw new BadRequestException('Payout not found');

      const requested = Number(payoutRes.rows[0].requested_amount);
      if (!Number.isFinite(requested) || requested <= 0) {
        throw new BadRequestException('payout has invalid requested_amount');
      }
      // P0-CP3: negative fees would inflate the net payout
      const fee = payload.fee_amount ?? 0;
      if (fee < 0 || fee > requested) throw new BadRequestException('invalid fee_amount');
      const net = payload.net_amount ?? requested - fee;
      if (net <= 0 || net > requested) throw new BadRequestException('invalid net_amount');

      const upd = await client.query(
        `UPDATE referral.payouts
          SET status = 'PAID', paid_at = now(),
              provider_reference = $2, fee_amount = $3, net_amount = $4
          WHERE id = $1 AND status IN ('PROCESSING', 'APPROVED', 'UNDER_REVIEW', 'FAILED')
          RETURNING *`,
        [id, payload.provider_reference ?? null, fee, net],
      );
      if (upd.rows.length === 0) {
        throw new BadRequestException('Payout not found or not in a reconcilable state');
      }

      // Mark reserved commissions PAID
      await client.query(
        `UPDATE referral.commission_ledger cl
          SET status = 'PAID', paid_at = now(), updated_at = now()
          WHERE id IN (SELECT commission_id FROM referral.payout_items WHERE payout_id = $1)`,
        [id],
      );

      // P0-CP3: guard against double payout — wallet debit must affect exactly
      // the reserved items, in the payout's own currency (never hardcoded USD).
      const payoutCurrencyRes = await client.query(
        `SELECT currency FROM referral.payouts WHERE id = $1`, [id],
      );
      const payoutCurrency = payoutCurrencyRes.rows[0]?.currency || 'USD';

      await client.query(
        `WITH agg AS (
           SELECT cl.recipient_user_id AS uid, SUM(pi.amount) AS amt
           FROM referral.payout_items pi
           JOIN referral.commission_ledger cl ON cl.id = pi.commission_id
           WHERE pi.payout_id = $1
           GROUP BY cl.recipient_user_id
         )
         UPDATE referral.affiliate_wallets w
           SET available_balance = w.available_balance - agg.amt,
               paid_balance = w.paid_balance + agg.amt,
               updated_at = now()
         FROM agg
         WHERE w.user_id = agg.uid AND w.currency = $2
           AND w.available_balance >= agg.amt`,
        [id, payoutCurrency],
      );

      await client.query(
        `INSERT INTO finance.ledger_entries
           (account_user_id, entry_type, direction, amount, currency, source_type, source_id, idempotency_key, metadata)
         SELECT p.user_id, 'PAYOUT', 'DEBIT', p.net_amount, p.currency, 'payout', p.id,
                'payout:' || p.id::text,
                jsonb_build_object('provider_reference', p.provider_reference)
         FROM referral.payouts p WHERE p.id = $1
         ON CONFLICT (idempotency_key) DO NOTHING`,
        [id],
      );

      await client.query('COMMIT');
      return upd.rows[0];
    } catch (err) {
      await client.query('ROLLBACK');
      throw err;
    } finally {
      client.release();
    }
  }

  async retryPayout(id: string) {
    const r = await this.pool.query(
      `UPDATE referral.payouts SET status = 'UNDER_REVIEW'
        WHERE id = $1 AND status = 'FAILED' RETURNING *`,
      [id],
    );
    if (r.rows.length === 0) throw new BadRequestException('Payout not found or not in FAILED state');
    return r.rows[0];
  }

  async cancelPayout(id: string, reason: string) {
    const r = await this.pool.query(
      `UPDATE referral.payouts
        SET status = 'CANCELLED', cancelled_at = now(),
            metadata = jsonb_set(COALESCE(metadata, '{}'::jsonb), '{reason}', to_jsonb($2::text))
        WHERE id = $1 AND status NOT IN ('PAID', 'CANCELLED', 'REJECTED')
        RETURNING *`,
      [id, reason],
    );
    if (r.rows.length === 0) throw new BadRequestException('Payout not found or not in a cancellable state');
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
              count(CASE WHEN status IN ('PENDING', 'REQUESTED') THEN 1 END) as pending,
              count(CASE WHEN status = 'APPROVED' THEN 1 END) as approved,
              count(CASE WHEN status = 'REJECTED' THEN 1 END) as rejected,
              COALESCE(SUM(CASE WHEN status IN ('PENDING', 'REQUESTED') THEN requested_amount ELSE 0 END), 0) as pending_amount,
              COALESCE(SUM(CASE WHEN status = 'APPROVED' THEN approved_amount ELSE 0 END), 0) as approved_amount
       FROM referral.payouts`,
    );
    return r.rows[0];
  }
}
