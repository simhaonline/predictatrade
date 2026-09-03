import { Injectable, Inject, Logger, OnModuleInit, OnModuleDestroy } from '@nestjs/common';
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

/** Ranges longer than this (per timeframe) cannot finish inside the
 * synchronous 300s exec budget and are queued as async jobs instead.
 * Measured engine cost after the float64 optimization: ~0.3ms/bar
 * (27.5k M1 bars ≈ 8s → ~10.5 min for the full 6.7-year M1 history). */
const SYNC_MAX_DAYS: Record<string, number> = {
  M1: 150, // ~8s per 30d slice → ~40s per 150d; 6.7y ≈ 10.5 min (async)
  M5: 900, // ~1.6s per 30d
};

@Injectable()
export class BacktestService implements OnModuleInit, OnModuleDestroy {
  private readonly logger = new Logger(BacktestService.name);
  private jobWorker: ReturnType<typeof setInterval> | null = null;
  private jobRunning = false;

  constructor(@Inject(DB_POOL) private pool: Pool) {}

  /** Async job worker: polls QUEUED backtest jobs and runs them detached
   * from HTTP. One job at a time (the engine is CPU-bound; parallel runs
   * would just contend). Failures mark the job FAILED with the engine error. */
  onModuleInit() {
    this.jobWorker = setInterval(() => void this.processQueuedJob(), 5000);
    this.logger.log('Backtest async job worker started (poll 5s)');
  }

  onModuleDestroy() {
    if (this.jobWorker) clearInterval(this.jobWorker);
  }

  private async processQueuedJob(): Promise<void> {
    if (this.jobRunning) return;
    this.jobRunning = true;
    try {
      const res = await this.pool.query(
        `UPDATE trading.backtest_jobs SET status = 'RUNNING', started_at = now()
         WHERE id = (
           SELECT id FROM trading.backtest_jobs
           WHERE status = 'QUEUED' ORDER BY created_at LIMIT 1
           FOR UPDATE SKIP LOCKED
         )
         RETURNING id, user_id, strategy_id, timeframe, start_date, end_date, initial_balance`,
      );
      if (res.rows.length === 0) return;
      const job = res.rows[0];
      this.logger.log(`Async backtest job ${job.id} started (${job.strategy_id} ${job.timeframe} ${job.start_date}→${job.end_date})`);
      try {
        const outcome = await this.executeEngine({
          strategy: job.strategy_id,
          timeframe: job.timeframe,
          startDate: job.start_date.toISOString().slice(0, 10),
          endDate: job.end_date.toISOString().slice(0, 10),
          initialBalance: Number(job.initial_balance),
        } as RunBacktestDto, job.user_id, true, job.id);
        // executeEngine swallows engine failures into a FAILED-shaped result —
        // treat anything that is not COMPLETED as a job failure so the job
        // status never lies (seen: exec timeout returned, not threw).
        if (outcome.status !== 'COMPLETED') {
          throw new Error(outcome.error || `engine returned status ${outcome.status}`);
        }
        await this.pool.query(
          `UPDATE trading.backtest_jobs SET status = 'COMPLETED', finished_at = now()
           WHERE id = $1 AND status = 'RUNNING'`,
          [job.id],
        );
        // Backfill run_id from the newest stored run for this user+params.
        await this.pool.query(
          `UPDATE trading.backtest_jobs j SET run_id = r.run_id
           FROM trading.backtest_runs r
           WHERE j.id = $1 AND r.user_id = $2
             AND r.strategy_id = j.strategy_id
             AND r.primary_timeframe = j.timeframe
             AND r.start_timestamp >= j.start_date
             AND r.end_timestamp <= j.end_date
             AND r.started_at >= j.started_at`,
          [job.id, job.user_id],
        );
        this.logger.log(`Async backtest job ${job.id} COMPLETED`);
      } catch (err) {
        await this.pool
          .query(
            `UPDATE trading.backtest_jobs SET status = 'FAILED', error = $2, finished_at = now()
             WHERE id = $1`,
            [job.id, String(err.message || err).slice(0, 2000)],
          )
          .catch(() => undefined);
        this.logger.error(`Async backtest job ${job.id} FAILED: ${err.message}`);
      }
    } finally {
      this.jobRunning = false;
    }
  }

  async runBacktest(dto: RunBacktestDto, userId: string, isAdmin: boolean) {
    this.logger.log(`Backtest requested by ${userId}: ${dto.strategy} ${dto.timeframe} ${dto.startDate}→${dto.endDate}`);

    // Range routing (2026-09-03, updated after 11.5x float64 optimization):
    // Ranges within the synchronous budget run inline as before. Longer
    // ranges are QUEUED as async jobs (trading.backtest_jobs) and executed
    // by the in-process worker detached from HTTP — the caller gets 202
    // semantics with a job id to poll. No more 504s on long studies.
    const startMs = Date.parse(`${dto.startDate}T00:00:00Z`);
    const endMs = Date.parse(`${dto.endDate}T00:00:00Z`);
    const tfKey = String(dto.timeframe || '').toUpperCase();
    const syncCap = SYNC_MAX_DAYS[tfKey] ?? 0;
    if (
      syncCap > 0 &&
      Number.isFinite(startMs) &&
      Number.isFinite(endMs) &&
      endMs - startMs > syncCap * 86_400_000
    ) {
      return this.enqueueJob(dto, userId);
    }

    return this.executeEngine(dto, userId, isAdmin);
  }

  /** Queue an over-budget backtest as an async job (202 semantics). */
  private async enqueueJob(dto: RunBacktestDto, userId: string) {
    const res = await this.pool.query(
      `INSERT INTO trading.backtest_jobs
         (user_id, strategy_id, timeframe, start_date, end_date, initial_balance)
       VALUES ($1, $2, $3, $4, $5, $6)
       RETURNING id, created_at`,
      [
        userId,
        dto.strategy,
        dto.timeframe,
        dto.startDate,
        dto.endDate,
        dto.initialBalance ?? 10000,
      ],
    );
    this.logger.log(`Backtest range over sync budget → queued as job ${res.rows[0].id}`);
    return {
      runId: res.rows[0].id,
      jobId: res.rows[0].id,
      status: 'QUEUED',
      queued: true,
      strategy: dto.strategy,
      timeframe: dto.timeframe,
      startDate: dto.startDate,
      endDate: dto.endDate,
      message:
        'Range exceeds the synchronous time budget. The backtest was queued and will run in the background (typically a few minutes). Poll GET /api/v1/backtest/jobs/:id — when COMPLETED, the run appears in history with full metrics, and the run id is attached to the job.',
    };
  }

  /** Job status for polling. Users see only their own jobs. */
  async getJob(jobId: string, userId: string, isAdmin: boolean) {
    const res = await this.pool.query(
      `SELECT id, user_id, strategy_id, timeframe, start_date, end_date,
              initial_balance, status, run_id, error, created_at, started_at, finished_at
       FROM trading.backtest_jobs WHERE id = $1`,
      [jobId],
    );
    if (res.rows.length === 0) return null;
    const job = res.rows[0];
    if (!isAdmin && job.user_id !== userId) return null;
    return {
      jobId: job.id,
      status: job.status,
      runId: job.run_id,
      error: job.error,
      strategy: job.strategy_id,
      timeframe: job.timeframe,
      startDate: job.start_date,
      endDate: job.end_date,
      initialBalance: Number(job.initial_balance),
      createdAt: job.created_at,
      startedAt: job.started_at,
      finishedAt: job.finished_at,
    };
  }

  /** List recent jobs for the caller (admins see all). */
  async listJobs(limit = 20, userId?: string, isAdmin = false) {
    const lim = Math.min(Math.max(Number(limit) || 20, 1), 100);
    const params: unknown[] = [lim];
    let where = '';
    if (!isAdmin && userId) {
      where = 'WHERE user_id = $2';
      params.push(userId);
    }
    const res = await this.pool.query(
      `SELECT id, user_id, strategy_id, timeframe, start_date, end_date,
              status, run_id, error, created_at, started_at, finished_at
       FROM trading.backtest_jobs ${where}
       ORDER BY created_at DESC LIMIT $1`,
      params,
    );
    return {
      jobs: res.rows.map((j: Record<string, unknown>) => ({
        jobId: j.id,
        strategy: j.strategy_id,
        timeframe: j.timeframe,
        startDate: j.start_date,
        endDate: j.end_date,
        status: j.status,
        runId: j.run_id,
        error: j.error,
        createdAt: j.created_at,
        startedAt: j.started_at,
        finishedAt: j.finished_at,
      })),
    };
  }

  /** Run the compiled engine synchronously and parse its stdout. Shared by
   * the synchronous HTTP path and the async job worker (jobId marks async). */
  private async executeEngine(dto: RunBacktestDto, userId: string, isAdmin: boolean, jobId?: string) {
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

    // M8 fix: pass the DB URL via the child process ENVIRONMENT, never as a CLI
    // argument (which would expose the password to `ps`/process listings). The
    // Go engine falls back to DATABASE_URL when --db-url is not supplied.
    const childEnv = { ...process.env };

    try {
      const { stdout, stderr } = await execFileAsync(BACKTEST_BINARY, args, {
        // Async jobs get 15 min (full-history M1 ≈ 5-8 min engine + ~2-3 min
        // data load for 2.1M candles + higher TFs); the synchronous HTTP
        // path stays at 5 min (nginx backtest location allows 330s).
        timeout: jobId ? 900000 : 300000,
        maxBuffer: 1024 * 1024 * 10,
        env: childEnv,
      });

      // Parse the run_id from stdout
      const runIdMatch = stdout.match(/Backtest Run:\s*(\S+)/);
      const runId = runIdMatch ? runIdMatch[1] : 'unknown';

      // H3 fix: attribute the run to its owner so subsequent reads can be
      // scoped per-user. Best-effort: ignore errors (e.g. run_id parse failure).
      // R9: also stamp the subscription + plan snapshot so backtests can be
      // attributed per-plan (which subscription generated which runs/strategy).
      if (runId !== 'unknown' && userId) {
        await this.pool
          .query(`UPDATE trading.backtest_runs SET user_id = $2 WHERE run_id = $1`, [runId, userId])
          .catch(() => undefined);
        const sub = await this.resolveActiveSubscription(userId, dto.strategy).catch(() => null);
        if (sub) {
          await this.pool
            .query(
              `UPDATE trading.backtest_runs
               SET subscription_id = $2, plan_code = $3, plan_name = $4
               WHERE run_id = $1`,
              [runId, sub.subscriptionId, sub.planCode, sub.planName],
            )
            .catch(() => undefined);
        }
      }

      // Verification trail (R9-verify): persist the engine's verbatim stdout
      // so the admin UI can render it next to the parsed metrics — the
      // dashboard numbers must always be traceable to real engine output.
      // The engine prints its config (strategy, period, source), signal
      // decisions, NO-TRADE reasons, metrics, first 10 trades and the
      // storage confirmation, so this captures the full story of the run.
      if (runId !== 'unknown' && stdout) {
        await this.pool
          .query(
            `UPDATE trading.backtest_runs SET raw_output = $2 WHERE run_id = $1`,
            [runId, stdout.slice(-10240)],
          )
          .catch(() => undefined);
      }

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

  /* ─── Subscription attribution (R9) ─── */

  /** Resolve the caller's ACTIVE subscription whose plan allows the strategy
   * being backtested. Plan snapshot is denormalized onto the run so revenue
   * reporting stays correct even after plan changes. Returns null when the
   * user has no qualifying subscription (admin/platform runs, free tier). */
  private async resolveActiveSubscription(
    userId: string,
    strategy: string,
  ): Promise<{ subscriptionId: string; planCode: string; planName: string } | null> {
    const res = await this.pool.query(
      `SELECT s.id AS subscription_id, p.code AS plan_code, p.name AS plan_name
       FROM billing.subscriptions s
       JOIN control.plans p ON p.id = s.plan_id
       WHERE s.user_id = $1
         AND s.status IN ('ACTIVE', 'TRIALING')
         AND s.billing_period_end > now()
         AND (
           p.allowed_strategies @> $2::jsonb
           OR s.selected_strategies @> $2::jsonb
           OR jsonb_array_length(p.allowed_strategies) = 0
         )
       ORDER BY p.monthly_price DESC
       LIMIT 1`,
      [userId, JSON.stringify([strategy])],
    );
    if (res.rows.length === 0) return null;
    return {
      subscriptionId: res.rows[0].subscription_id,
      planCode: res.rows[0].plan_code,
      planName: res.rows[0].plan_name,
    };
  }

  /** Per-plan revenue + backtest-usage report (admin). Joins stamped runs
   * with subscriptions and actual collected payments so the operator sees,
   * per plan: MRR from ACTIVE subs, collected revenue, and how many
   * backtest runs / strategies the plan's users executed. */
  async getRevenueByPlan(): Promise<{
    plans: Array<{
      planCode: string;
      planName: string;
      monthlyPrice: number;
      activeSubscriptions: number;
      mrr: number;
      collectedRevenue: number;
      backtestRuns: number;
      strategiesUsed: string[];
    }>;
    totals: { mrr: number; collectedRevenue: number; backtestRuns: number };
  }> {
    const planRes = await this.pool.query(
      `SELECT p.code, p.name, p.monthly_price,
              count(s.id) FILTER (WHERE s.status IN ('ACTIVE','TRIALING')) AS active_subs,
              COALESCE(sum(p.monthly_price) FILTER (WHERE s.status IN ('ACTIVE','TRIALING')), 0) AS mrr,
              COALESCE((
                SELECT sum(pay.amount)
                FROM billing.payments pay
                JOIN billing.subscriptions s2 ON s2.id = pay.subscription_id
                WHERE s2.plan_id = p.id AND pay.status = 'SUCCEEDED'
              ), 0) AS collected
       FROM control.plans p
       LEFT JOIN billing.subscriptions s ON s.plan_id = p.id
       WHERE p.status = 'ACTIVE'
       GROUP BY p.id, p.code, p.name, p.monthly_price
       ORDER BY p.monthly_price`,
    );

    const runRes = await this.pool.query(
      `SELECT p.code AS plan_code,
              count(DISTINCT r.id) AS runs,
              COALESCE(jsonb_agg(DISTINCT r.strategy_id) FILTER (WHERE r.strategy_id IS NOT NULL), '[]'::jsonb) AS strategies
       FROM trading.backtest_runs r
       JOIN control.plans p ON p.code = r.plan_code
       WHERE r.plan_code IS NOT NULL
       GROUP BY p.code`,
    );
    const runMap = new Map<string, { runs: number; strategies: string[] }>();
    for (const row of runRes.rows) {
      runMap.set(row.plan_code, { runs: parseInt(row.runs, 10), strategies: row.strategies ?? [] });
    }

    const plans = planRes.rows.map((p) => {
      const usage = runMap.get(p.code);
      return {
        planCode: p.code,
        planName: p.name,
        monthlyPrice: parseFloat(p.monthly_price),
        activeSubscriptions: parseInt(p.active_subs ?? '0', 10),
        mrr: parseFloat(p.mrr ?? '0'),
        collectedRevenue: parseFloat(p.collected ?? '0'),
        backtestRuns: usage?.runs ?? 0,
        strategiesUsed: usage?.strategies ?? [],
      };
    });
    return {
      plans,
      totals: {
        mrr: plans.reduce((a, p) => a + p.mrr, 0),
        collectedRevenue: plans.reduce((a, p) => a + p.collectedRevenue, 0),
        backtestRuns: plans.reduce((a, p) => a + p.backtestRuns, 0),
      },
    };
  }

  /** Per-plan BACKTEST PERFORMANCE (P/L) report (admin): for each plan, the
   * aggregated backtest results of every strategy that plan allows — run
   * count, total P/L (sum of final-initial across runs), average return %,
   * win rate, profit factor and trades. Runs are grouped by strategy_id and
   * mapped to plans via control.plans.allowed_strategies, so historical
   * (unstamped) runs are attributed by strategy; runs stamped with a
   * subscription plan count toward that plan too. */
  async getPerformanceByPlan(): Promise<{
    plans: Array<{
      planCode: string;
      planName: string;
      monthlyPrice: number;
      strategies: Array<{
        strategyId: string;
        runs: number;
        totalPnl: number;
        avgReturnPct: number;
        bestReturnPct: number;
        worstReturnPct: number;
        avgWinRate: number;
        avgProfitFactor: number;
        profitableRuns: number;
      }>;
      totals: { runs: number; totalPnl: number; avgWinRate: number };
    }>;
  }> {
    // Aggregate all COMPLETED runs by strategy once…
    const stratRes = await this.pool.query(
      `SELECT strategy_id,
              count(*) AS runs,
              COALESCE(sum(final_balance - initial_balance), 0) AS total_pnl,
              COALESCE(avg(total_return_pct), 0) AS avg_ret,
              COALESCE(max(total_return_pct), 0) AS best_ret,
              COALESCE(min(total_return_pct), 0) AS worst_ret,
              COALESCE(avg(win_rate_pct), 0) AS avg_win_rate,
              COALESCE(avg(profit_factor), 0) AS avg_profit_factor,
              count(*) FILTER (WHERE total_return_pct > 0) AS profitable_runs
       FROM trading.backtest_runs
       WHERE status = 'COMPLETED' AND strategy_id IS NOT NULL
       GROUP BY strategy_id`,
    );
    const byStrategy = new Map<string, {
      runs: number; totalPnl: number; avgRet: number; bestRet: number;
      worstRet: number; avgWinRate: number; avgProfitFactor: number; profitableRuns: number;
    }>();
    for (const r of stratRes.rows) {
      byStrategy.set(r.strategy_id, {
        runs: parseInt(r.runs, 10),
        totalPnl: parseFloat(r.total_pnl),
        avgRet: parseFloat(r.avg_ret),
        bestRet: parseFloat(r.best_ret),
        worstRet: parseFloat(r.worst_ret),
        avgWinRate: parseFloat(r.avg_win_rate),
        avgProfitFactor: parseFloat(r.avg_profit_factor),
        profitableRuns: parseInt(r.profitable_runs, 10),
      });
    }

    // …then fan each plan's allowed strategies out into its own card data.
    const planRes = await this.pool.query(
      `SELECT p.code, p.name, p.monthly_price, p.allowed_strategies
       FROM control.plans p
       WHERE p.status = 'ACTIVE'
       ORDER BY p.monthly_price`,
    );
    const plans = planRes.rows.map((p) => {
      const allowed: string[] = Array.isArray(p.allowed_strategies) ? p.allowed_strategies : [];
      const strategies = allowed
        .filter((s) => byStrategy.has(s))
        .map((s) => {
          const a = byStrategy.get(s)!;
          return {
            strategyId: s,
            runs: a.runs,
            totalPnl: a.totalPnl,
            avgReturnPct: a.avgRet,
            bestReturnPct: a.bestRet,
            worstReturnPct: a.worstRet,
            avgWinRate: a.avgWinRate,
            avgProfitFactor: a.avgProfitFactor,
            profitableRuns: a.profitableRuns,
          };
        });
      const runs = strategies.reduce((acc, s) => acc + s.runs, 0);
      const totalPnl = strategies.reduce((acc, s) => acc + s.totalPnl, 0);
      const avgWinRate = runs > 0
        ? strategies.reduce((acc, s) => acc + s.avgWinRate * s.runs, 0) / runs
        : 0;
      return {
        planCode: p.code,
        planName: p.name,
        monthlyPrice: parseFloat(p.monthly_price),
        strategies,
        totals: { runs, totalPnl, avgWinRate },
      };
    });
    return { plans };
  }

  private parseMetric(text: string, label: string): string | null {
    const regex = new RegExp(`${label}\\s*\\$?(-?[\\d.]+)`);
    const match = text.match(regex);
    return match ? match[1] : '0';
  }

  async listRuns(limit = 20, userId?: string, isAdmin = false) {
    const where = !isAdmin && userId ? 'WHERE user_id = $2' : '';
    const params = !isAdmin && userId ? [limit, userId] : [limit];
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
      ${where}
      ORDER BY created_at DESC
      LIMIT $1
    `, params);
    return result.rows;
  }

  async getRunDetails(runId: string, userId?: string, isAdmin = false) {
    const runResult = await this.pool.query(`
      SELECT * FROM trading.backtest_runs WHERE run_id = $1
    `, [runId]);

    if (runResult.rows.length === 0) {
      return null;
    }

    const run = runResult.rows[0];
    // H3: non-admins may only access their own runs.
    if (!isAdmin && userId && run.user_id && run.user_id !== userId) {
      return null;
    }

    // Verification trail: only admins see the raw engine output.
    if (!isAdmin) {
      delete run.raw_output;
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
      run,
      trades: tradesResult.rows,
    };
  }

  async getRunTradesCSV(runId: string, userId?: string, isAdmin = false): Promise<string> {
    // H3: verify ownership before streaming another user's trade data.
    const ownerRes = await this.pool.query(
      `SELECT user_id FROM trading.backtest_runs WHERE run_id = $1`,
      [runId],
    );
    if (ownerRes.rows.length === 0) {
      return 'Run not found';
    }
    if (!isAdmin && userId && ownerRes.rows[0].user_id && ownerRes.rows[0].user_id !== userId) {
      return 'Access denied';
    }

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
