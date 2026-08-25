import { Injectable, Inject, Logger } from '@nestjs/common';
import { Pool } from 'pg';
import { execFile } from 'child_process';
import { promisify } from 'util';
import { join } from 'path';
import { RunBacktestDto } from './backtest.dto';

const execFileAsync = promisify(execFile);
const DB_POOL = 'DB_POOL';

// Binary location. In the container it is mounted at /app/backtest-engine; the
// host path is kept as a fallback for non-container deployments.
const BACKTEST_BINARY =
  process.env.BACKTEST_BINARY || '/app/backtest-engine';

@Injectable()
export class BacktestService {
  private readonly logger = new Logger(BacktestService.name);

  constructor(@Inject(DB_POOL) private pool: Pool) {}

  async runBacktest(dto: RunBacktestDto, userId: string, isAdmin: boolean) {
    this.logger.log(`Backtest requested by ${userId}: ${dto.strategy} ${dto.timeframe} ${dto.startDate}→${dto.endDate}`);

    const args = [
      '--strategy', dto.strategy,
      '--timeframe', dto.timeframe,
      '--start', dto.startDate,
      '--end', dto.endDate,
      '--balance', String(dto.initialBalance || 10000),
      '--store',
    ];

    if (dto.higherTimeframes) {
      args.push('--higher-tfs', dto.higherTimeframes);
    }

    // Inject the DB URL from the environment so the engine connects using the
    // container-resolvable host (e.g. postgres:) instead of a host-only file
    // that is unavailable inside the container.
    if (process.env.DATABASE_URL) {
      args.push('--db-url', process.env.DATABASE_URL);
    }

    try {
      const { stdout, stderr } = await execFileAsync(BACKTEST_BINARY, args, {
        timeout: 300000, // 5 min max
        maxBuffer: 1024 * 1024 * 10,
      });

      // Parse the run_id from stdout
      const runIdMatch = stdout.match(/Backtest Run:\s*(\S+)/);
      const runId = runIdMatch ? runIdMatch[1] : 'unknown';

      // Parse key metrics from stdout
      const finalBalance = this.parseMetric(stdout, 'Final balance:');
      const totalReturn = this.parseMetric(stdout, 'Total return:');
      const winRate = this.parseMetric(stdout, 'Win rate:');
      const profitFactor = this.parseMetric(stdout, 'Profit factor:');
      const sharpe = this.parseMetric(stdout, 'Sharpe ratio:');
      const maxDD = this.parseMetric(stdout, 'Max drawdown:');
      const totalTrades = this.parseMetric(stdout, 'Total trades:');

      return {
        runId,
        status: 'COMPLETED',
        strategy: dto.strategy,
        timeframe: dto.timeframe,
        startDate: dto.startDate,
        endDate: dto.endDate,
        metrics: {
          finalBalance,
          totalReturn,
          winRate,
          profitFactor,
          sharpe,
          maxDD,
          totalTrades,
        },
        rawOutput: stdout.slice(-2000), // last 2KB of output
      };
    } catch (err) {
      this.logger.error(`Backtest failed: ${err.message}`);
      return {
        runId: 'failed',
        status: 'FAILED',
        error: err.message,
        strategy: dto.strategy,
        timeframe: dto.timeframe,
      };
    }
  }

  private parseMetric(stdout: string, label: string): string {
    const regex = new RegExp(`${label}\\s*\\$?([\\d.]+)`);
    const match = stdout.match(regex);
    return match ? match[1] : '0';
  }

  async listRuns(limit = 20) {
    const result = await this.pool.query(`
      SELECT run_id, strategy_id, strategy_mode, status, 
             to_char(start_timestamp, 'YYYY-MM-DD') as start_date,
             to_char(end_timestamp, 'YYYY-MM-DD') as end_date,
             round(initial_balance, 0) as initial_balance,
             round(final_balance, 2) as final_balance,
             round(total_return_pct, 2) as total_return_pct,
             round(win_rate_pct, 1) as win_rate,
             round(profit_factor, 2) as profit_factor,
             round(sharpe_ratio, 2) as sharpe,
             round(max_drawdown_pct, 2) as max_drawdown,
             trades_count, bars_processed,
             round(duration_seconds, 1) as duration_seconds,
             created_at
      FROM trading.backtest_runs
      ORDER BY created_at DESC
      LIMIT $1
    `, [limit]);
    return result.rows;
  }

  async getRunDetails(runId: string) {
    const runResult = await this.pool.query(`
      SELECT * FROM trading.backtest_runs WHERE run_id = $1
    `, [runId]);

    if (runResult.rows.length === 0) {
      return null;
    }

    const tradesResult = await this.pool.query(`
      SELECT direction, entry_price, exit_price, exit_reason,
             pnl, pnl_r, entry_time, exit_time, duration_bars as holding_bars,
             strategy_id
      FROM trading.backtest_trades
      WHERE run_id = $1
      ORDER BY entry_time
      LIMIT 500
    `, [runId]);

    return {
      run: runResult.rows[0],
      trades: tradesResult.rows,
    };
  }

  async getRunTradesCSV(runId: string): Promise<string> {
    const result = await this.pool.query(`
      SELECT direction, strategy_id, entry_price, exit_price, exit_reason,
             pnl, pnl_r, entry_time, exit_time, duration_bars as holding_bars
      FROM trading.backtest_trades
      WHERE run_id = $1
      ORDER BY entry_time
    `, [runId]);

    if (result.rows.length === 0) {
      return 'No trades found for this run';
    }

    const headers = ['Direction', 'Strategy', 'Entry Price', 'Exit Price', 'Exit Reason',
                     'PnL', 'R Multiple', 'Entry Time', 'Exit Time', 'Holding Bars'];
    const rows = result.rows.map(r => [
      r.direction, r.strategy_id, r.entry_price, r.exit_price, r.exit_reason,
      r.pnl, r.pnl_r, r.entry_time, r.exit_time, r.holding_bars
    ].join(','));

    return [headers.join(','), ...rows].join('\n');
  }

  async getAvailableData() {
    // Fast metadata lookup — no chunk scanning needed
    const result = await this.pool.query(`
      SELECT timeframe, candle_count,
             to_char(min_date, 'YYYY-MM-DD') as min_date,
             to_char(max_date, 'YYYY-MM-DD') as max_date,
             source
      FROM market.data_metadata
      ORDER BY CASE timeframe 
        WHEN 'M1' THEN 1 WHEN 'M5' THEN 2 WHEN 'M15' THEN 3 WHEN 'M30' THEN 4
        WHEN 'H1' THEN 5 WHEN 'H4' THEN 6 WHEN 'D1' THEN 7 WHEN 'W1' THEN 8 WHEN 'MN' THEN 9 END
    `);
    return result.rows;
  }
}
