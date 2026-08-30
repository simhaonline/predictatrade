import { Injectable, Inject } from '@nestjs/common';
import { Pool } from 'pg';

// Broker Account Types + Strategy Cost Gates (check.md 2026-08-30 playbook §8)
@Injectable()
export class BrokersService {
  constructor(@Inject('DB_POOL') private pool: Pool) {}

  async listAccountTypes() {
    return (await this.pool.query(
      `SELECT * FROM licensing.broker_account_types ORDER BY
        CASE execution_model WHEN 'ecn' THEN 1 WHEN 'institutional' THEN 2
        WHEN 'stp' THEN 3 WHEN 'micro' THEN 4 WHEN 'swapfree' THEN 5
        WHEN 'dealing_desk' THEN 6 ELSE 7 END`
    )).rows;
  }

  async getStrategyGate(strategyId: string) {
    return (await this.pool.query(
      `SELECT g.*, t.label, t.execution_model, t.typical_spread_pips, t.commission_per_side, t.min_deposit
       FROM licensing.strategy_cost_gates g
       JOIN licensing.broker_account_types t ON t.code = g.broker_account_code
       WHERE g.strategy_id = $1 ORDER BY g.cost_as_pct_of_1r ASC`,
      [strategyId],
    )).rows;
  }

  async listAllGates() {
    return (await this.pool.query(
      `SELECT s.strategy_id, s.broker_account_code, s.cost_as_pct_of_1r, s.suitability, s.allowed,
              t.label, t.execution_model
       FROM licensing.strategy_cost_gates s
       JOIN licensing.broker_account_types t ON t.code = s.broker_account_code
       ORDER BY s.strategy_id, s.cost_as_pct_of_1r`)).rows;
  }
}
