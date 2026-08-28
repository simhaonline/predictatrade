#!/usr/bin/env python3
"""Predict-A-Trade Backtesting CLI.

Commands:
  backtest run          — Run a backtest
  backtest validate-data — Validate data quality
  backtest precompute    — Precompute PTB features
  backtest walk-forward  — Run walk-forward analysis
  backtest monte-carlo   — Run Monte Carlo analysis
  backtest sensitivity   — Run sensitivity analysis
  backtest report        — Generate report from existing run
  backtest list          — List all backtest runs
  backtest show <run-id> — Show details of a specific run

Usage:
  python -m patresearch.backtesting.cli run --symbol XAUUSD --strategy STANDARD_SCALPING --timeframe M5 --start 2025-01-01 --end 2025-12-31 --seed 42
"""
from __future__ import annotations

import argparse
import sys
import os
import json
from datetime import datetime, timezone
from typing import Optional

from .data.loader import DataLoader, DatasetMetadata
from .data.quality import DataQualityValidator
from .engine.core import BacktestEngine, BacktestConfig
from .engine.execution import ExecutionConfig
from .strategy.ptb_strategy import PTBStrategyAdapter
from .analytics.metrics import calculate_metrics
from .analytics.walk_forward import WalkForwardAnalyzer, WalkForwardConfig
from .analytics.monte_carlo import MonteCarloAnalyzer, MonteCarloConfig
from .analytics.sensitivity import SensitivityAnalyzer, SensitivityConfig
from .reporting.report import ReportGenerator


def cmd_run(args):
    """Run a backtest."""
    # Load data
    if args.db_url:
        start = _parse_dt(args.start) if args.start else None
        end = _parse_dt(args.end) if args.end else None
        candles, meta = DataLoader.from_database(
            args.db_url, args.symbol, args.timeframe, start, end
        )
        print(f"Loaded {len(candles)} candles from TimescaleDB (market.candles) "
              f"from {meta.start_time} to {meta.end_time}")
    elif args.data_file:
        candles, meta = DataLoader.from_csv(args.data_file, args.symbol, args.timeframe)
        # Apply optional date-window filtering on genuine CSV data so we can
        # backtest a specific regime (e.g. the current high-volatility period)
        # without loading the entire multi-decade history into memory.
        if (args.start or args.end) and args.db_url is None:
            start = _parse_dt(args.start) if args.start else None
            end = _parse_dt(args.end) if args.end else None
            before = len(candles)
            candles = [c for c in candles if (start is None or c.timestamp >= start)
                       and (end is None or c.timestamp <= end)]
            if not candles:
                print(f"ERROR: date window {start}..{end} matched 0 of {before} candles")
                return 1
            meta = DatasetMetadata(
                symbol=meta.symbol, timeframe=meta.timeframe, source=meta.source,
                start_time=candles[0].timestamp, end_time=candles[-1].timestamp,
                record_count=len(candles), data_hash=meta.data_hash,
            )
            print(f"Windowed to {len(candles)} candles ({meta.start_time} .. {meta.end_time})")
    else:
        n = args.candles or 500
        candles, meta = DataLoader.generate_synthetic(
            args.symbol, args.timeframe, n_candles=n, seed=args.seed
        )

    if not candles:
        print("ERROR: no candles loaded — check data source / db-url / symbol / timeframe")
        return 1

    print(f"Loaded {len(candles)} candles ({meta.source}) from {meta.start_time} to {meta.end_time}")
    print(f"Data hash: {meta.data_hash}")

    # Anti-fake-data guard: never let a synthetic dataset stand in for genuine
    # market history in a reported backtest. Operator must pass --allow-synthetic
    # explicitly to run on generated/fixture data.
    if meta.source == "SYNTHETIC" and not args.allow_synthetic:
        print("\n[REFUSED] Data source is SYNTHETIC. Refusing to produce a backtest "
              "report on non-genuine data.\n"
              "          Re-run with --data-file pointing at real XAUUSD history, "
              "or pass --allow-synthetic if you intentionally want a self-test.")
        return 2

    # Configure
    config = BacktestConfig(
        symbol=args.symbol,
        strategy_id=args.strategy,
        primary_timeframe=args.timeframe,
        initial_balance=args.balance,
        random_seed=args.seed,
    )

    # Create engine and strategy
    engine = BacktestEngine(config)
    strategy = PTBStrategyAdapter(args.strategy)
    engine.set_strategy(strategy)

    # Run backtest
    result = engine.run(candles)

    # Generate report
    reporter = ReportGenerator(output_dir=args.output or "backtest_reports")
    artifacts = reporter.generate(result)

    # Print summary
    print(f"\n{'='*60}")
    print(f"Backtest Run: {result.run_id}")
    print(f"Status: {result.status}")
    print(f"Bars processed: {result.bars_processed}")
    print(f"Data quality: {result.data_quality.quality_score:.2f}")
    print(f"Trades: {len(result.trades)}")
    print(f"NO_TRADE decisions: {result.no_trade_count}")
    print(f"Blocked signals: {result.blocked_count}")

    if result.metrics:
        print(f"\n--- Metrics ---")
        print(f"Initial balance: ${result.metrics.initial_balance:,.2f}")
        print(f"Final balance:   ${result.metrics.final_balance:,.2f}")
        print(f"Net profit:     ${result.metrics.net_profit:,.2f}")
        print(f"Total return:    {result.metrics.total_return_pct:.2f}%")
        print(f"Win rate:        {result.metrics.win_rate_pct:.1f}%")
        print(f"Profit factor:   {result.metrics.profit_factor:.2f}")
        print(f"Sharpe ratio:    {result.metrics.sharpe_ratio:.2f}")
        print(f"Sortino ratio:   {result.metrics.sortino_ratio:.2f}")
        print(f"Max drawdown:    {result.metrics.max_drawdown_pct:.2f}%")

    print(f"\n--- Artifacts ---")
    for name, path in artifacts.items():
        print(f"  {name}: {path}")

    # Persist to TimescaleDB for online retrieval / reporting when a db-url is set.
    if args.db_url:
        try:
            from .reporting.db_writer import store_run
            run_id = store_run(
                result, args.db_url,
                data_source="TIMESCALEDB",
                data_hash=meta.data_hash if hasattr(meta, "data_hash") else None,
            )
            print(f"\n[STORED] run_id={run_id} persisted to trading.backtest_runs "
                  f"(equity + metrics in trading.backtest_artifacts)")
        except Exception as e:  # pragma: no cover - storage is best-effort here
            print(f"\n[WARN] failed to persist run to DB: {e}")

    return 0 if result.status == "COMPLETED" else 1


def cmd_validate_data(args):
    """Validate data quality."""
    if not args.data_file:
        print("Error: --data-file required for validation")
        return 1

    candles, meta = DataLoader.from_csv(args.data_file, args.symbol, args.timeframe)
    validator = DataQualityValidator()
    report = validator.validate(candles, args.symbol, args.timeframe)

    print(f"\nData Quality Report")
    print(f"{'='*40}")
    print(f"Symbol: {report.symbol}")
    print(f"Timeframe: {report.timeframe}")
    print(f"Total candles: {report.total_candles}")
    print(f"Errors: {report.errors}")
    print(f"Warnings: {report.warnings}")
    print(f"Quality score: {report.quality_score:.4f}")
    print(f"Passed: {report.passed}")

    if report.issues:
        print(f"\nIssues:")
        for issue in report.issues[:20]:
            print(f"  [{issue.severity}] {issue.category}: {issue.message}")

    return 0 if report.passed else 1


def cmd_precompute(args):
    """Precompute PTB features."""
    if args.data_file:
        candles, meta = DataLoader.from_csv(args.data_file, args.symbol, args.timeframe)
    else:
        candles, meta = DataLoader.generate_synthetic(args.symbol, args.timeframe, 500, seed=args.seed)

    from .features.precompute import FeaturePrecomputer, PrecomputeConfig
    config = PrecomputeConfig(symbol=args.symbol, timeframe=args.timeframe)
    precomputer = FeaturePrecomputer(config)

    strategy = PTBStrategyAdapter(args.strategy)
    features, metadata = precomputer.precompute(candles, strategy=strategy)

    output_file = args.output or f"precomputed_features_{args.symbol}_{args.timeframe}.json"
    precomputer.save_to_json(features, metadata, output_file)

    print(f"Precomputed {len(features)} feature records")
    print(f"Saved to: {output_file}")
    print(f"Data hash: {metadata.data_version_hash}")
    return 0


def cmd_walk_forward(args):
    """Run walk-forward analysis."""
    if args.data_file:
        candles, meta = DataLoader.from_csv(args.data_file, args.symbol, args.timeframe)
    else:
        candles, meta = DataLoader.generate_synthetic(args.symbol, args.timeframe, 1000, seed=args.seed)

    config = BacktestConfig(
        symbol=args.symbol, strategy_id=args.strategy,
        primary_timeframe=args.timeframe, random_seed=args.seed,
    )

    wf_config = WalkForwardConfig(
        train_size=args.train_size or 500,
        test_size=args.test_size or 100,
        step_size=args.step_size or 100,
    )

    analyzer = WalkForwardAnalyzer(wf_config)
    result = analyzer.run(
        candles,
        strategy_factory=lambda: PTBStrategyAdapter(args.strategy),
        bt_config=config,
    )

    print(f"\nWalk-Forward Analysis: {len(result.folds)} folds")
    print(f"{'='*50}")

    for fold in result.folds:
        oos = fold.out_of_sample_result
        if oos and oos.metrics:
            print(f"Fold {fold.fold_id}: OOS trades={oos.metrics.total_trades}, "
                  f"return={oos.metrics.total_return_pct:.1f}%, "
                  f"sharpe={oos.metrics.sharpe_ratio:.2f}")

    if result.aggregate_oos_metrics:
        print(f"\nAggregate OOS:")
        for k, v in result.aggregate_oos_metrics.items():
            print(f"  {k}: {v}")

    if result.final_holdout_result:
        fh = result.final_holdout_result
        print(f"\nFinal Holdout:")
        if fh.metrics:
            print(f"  Trades: {fh.metrics.total_trades}")
            print(f"  Return: {fh.metrics.total_return_pct:.2f}%")
            print(f"  Sharpe: {fh.metrics.sharpe_ratio:.2f}")

    return 0


def cmd_monte_carlo(args):
    """Run Monte Carlo analysis."""
    # Run a backtest first
    if args.data_file:
        candles, meta = DataLoader.from_csv(args.data_file, args.symbol, args.timeframe)
    else:
        candles, meta = DataLoader.generate_synthetic(args.symbol, args.timeframe, 500, seed=args.seed)

    config = BacktestConfig(symbol=args.symbol, strategy_id=args.strategy,
                             primary_timeframe=args.timeframe, random_seed=args.seed)
    engine = BacktestEngine(config)
    engine.set_strategy(PTBStrategyAdapter(args.strategy))
    result = engine.run(candles)

    if not result.trades:
        print("No trades to analyze")
        return 1

    mc_config = MonteCarloConfig(n_simulations=args.runs or 1000, random_seed=args.seed)
    analyzer = MonteCarloAnalyzer(mc_config)
    mc_result = analyzer.run(result.trades, config.initial_balance)

    print(f"\nMonte Carlo Analysis ({mc_result.n_simulations} simulations)")
    print(f"{'='*50}")
    print(f"Probability of loss: {mc_result.prob_of_loss:.1%}")
    print(f"\nFinal Equity Percentiles:")
    for pct, val in sorted(mc_result.final_equity_percentiles.items()):
        print(f"  {pct}: ${val:,.2f}")
    print(f"\nMax Drawdown Percentiles:")
    for pct, val in sorted(mc_result.max_drawdown_percentiles.items()):
        print(f"  {pct}: {val:.2f}%")
    print(f"\nProbability of drawdown exceeding:")
    for threshold, prob in mc_result.prob_of_drawdown_exceeding.items():
        print(f"  {threshold}: {prob:.1%}")

    return 0


def cmd_sensitivity(args):
    """Run parameter sensitivity analysis."""
    if args.data_file:
        candles, meta = DataLoader.from_csv(args.data_file, args.symbol, args.timeframe)
    else:
        candles, meta = DataLoader.generate_synthetic(args.symbol, args.timeframe, 500, seed=args.seed)

    config = BacktestConfig(symbol=args.symbol, strategy_id=args.strategy,
                             primary_timeframe=args.timeframe, random_seed=args.seed)

    sens_config = SensitivityConfig(perturbation_pct=args.perturbation or 0.10)
    analyzer = SensitivityAnalyzer(sens_config)
    result = analyzer.run(
        candles, config,
        strategy_factory=lambda: PTBStrategyAdapter(args.strategy),
    )

    print(f"\nParameter Sensitivity Analysis (±{sens_config.perturbation_pct:.0%})")
    print(f"{'='*60}")
    for sr in result.results:
        print(f"\n{sr.parameter} (base={sr.base_value:.4f}):")
        print(f"  Low  ({sr.low_value:.4f}): sharpe={sr.low_metrics.get('sharpe', 0):.2f}, "
              f"return={sr.low_metrics.get('total_return_pct', 0):.1f}%")
        print(f"  Base ({sr.base_value:.4f}): sharpe={sr.base_metrics.get('sharpe', 0):.2f}, "
              f"return={sr.base_metrics.get('total_return_pct', 0):.1f}%")
        print(f"  High ({sr.high_value:.4f}): sharpe={sr.high_metrics.get('sharpe', 0):.2f}, "
              f"return={sr.high_metrics.get('total_return_pct', 0):.1f}%")
        print(f"  Sensitivity score: {sr.sensitivity_score:.2f}")

    if result.fragile_parameters:
        print(f"\n⚠️  Fragile parameters (sensitivity > 0.5):")
        for p in result.fragile_parameters:
            print(f"  - {p}")

    return 0


def cmd_list(args):
    """List all backtest runs."""
    reports_dir = args.output or "backtest_reports"
    if not os.path.exists(reports_dir):
        print("No backtest runs found")
        return 0

    runs = []
    for run_id in sorted(os.listdir(reports_dir)):
        manifest_path = os.path.join(reports_dir, run_id, "run_manifest.json")
        if os.path.exists(manifest_path):
            with open(manifest_path) as f:
                manifest = json.load(f)
            runs.append(manifest)

    if not runs:
        print("No backtest runs found")
        return 0

    print(f"{'Run ID':<10} {'Strategy':<20} {'Status':<12} {'Trades':>8} {'Return%':>10}")
    print(f"{'-'*60}")
    for r in runs:
        print(f"{r.get('run_id', ''):<10} {r.get('strategy', ''):<20} "
              f"{r.get('status', ''):<12} {r.get('metrics', {}).get('trades', 0):>8} "
              f"{r.get('metrics', {}).get('total_return_pct', 0):>10.2f}")

    return 0


def cmd_show(args):
    """Show details of a specific run."""
    reports_dir = args.output or "backtest_reports"
    manifest_path = os.path.join(reports_dir, args.run_id, "run_manifest.json")

    if not os.path.exists(manifest_path):
        print(f"Run {args.run_id} not found")
        return 1

    with open(manifest_path) as f:
        manifest = json.load(f)

    print(json.dumps(manifest, indent=2, default=str))
    return 0


def _parse_dt(s: str):
    """Parse a CLI datetime string (ISO or YYYY-MM-DD) into UTC datetime."""
    from datetime import datetime, timezone
    s = s.strip()
    for fmt in ("%Y-%m-%dT%H:%M:%S", "%Y-%m-%d %H:%M:%S", "%Y-%m-%d"):
        try:
            dt = datetime.strptime(s, fmt)
            return dt.replace(tzinfo=timezone.utc)
        except ValueError:
            continue
    # Try ISO with timezone
    try:
        dt = datetime.fromisoformat(s)
        return dt if dt.tzinfo else dt.replace(tzinfo=timezone.utc)
    except ValueError:
        raise SystemExit(f"Invalid --start/--end value: {s!r} (use YYYY-MM-DD or ISO8601)")


def cmd_db_list(args):
    """List persisted backtest runs from TimescaleDB."""
    import psycopg2
    conn = psycopg2.connect(args.db_url)
    try:
        with conn.cursor() as cur:
            where, params = [], []
            if args.symbol:
                where.append("symbol = %s"); params.append(args.symbol)
            if args.strategy:
                where.append("strategy_id = %s"); params.append(args.strategy)
            sql = (
                "SELECT run_id, symbol, strategy_id, primary_timeframe, status, "
                "start_timestamp, end_timestamp, final_balance, total_return_pct, "
                "sharpe_ratio, trades_count, created_at "
                "FROM trading.backtest_runs"
            )
            if where:
                sql += " WHERE " + " AND ".join(where)
            sql += " ORDER BY created_at DESC LIMIT %s"
            params.append(args.limit)
            cur.execute(sql, params)
            rows = cur.fetchall()
    finally:
        conn.close()

    if not rows:
        print("No stored runs found.")
        return 0
    print(f"{'RUN_ID':<10} {'SYM':<7} {'STRATEGY':<18} {'TF':<4} {'STATUS':<9} "
          f"{'RET%':>8} {'SHARPE':>7} {'TRADES':>6} CREATED")
    for r in rows:
        print(f"{r[0]:<10} {r[1]:<7} {r[2]:<18} {r[3]:<4} {r[4]:<9} "
              f"{_fmt(r[7]):>8} {_fmt(r[8]):>7} {r[9]:>6} {r[11]}")
    return 0


def cmd_db_show(args):
    """Show a stored backtest run (summary + equity curve) from TimescaleDB."""
    import psycopg2
    conn = psycopg2.connect(args.db_url)
    try:
        with conn.cursor() as cur:
            cur.execute(
                "SELECT run_id, symbol, strategy_id, primary_timeframe, status, "
                "start_timestamp, end_timestamp, initial_balance, final_balance, "
                "total_return_pct, sharpe_ratio, sortino_ratio, max_drawdown_pct, "
                "win_rate_pct, profit_factor, expectancy, trades_count, "
                "no_trade_count, data_source, data_hash, created_at "
                "FROM trading.backtest_runs WHERE run_id = %s",
                (args.run_id,),
            )
            run = cur.fetchone()
            if not run:
                print(f"Run {args.run_id} not found in TimescaleDB.")
                return 1
            cols = ["run_id", "symbol", "strategy_id", "primary_timeframe", "status",
                    "start_timestamp", "end_timestamp", "initial_balance", "final_balance",
                    "total_return_pct", "sharpe_ratio", "sortino_ratio", "max_drawdown_pct",
                    "win_rate_pct", "profit_factor", "expectancy", "trades_count",
                    "no_trade_count", "data_source", "data_hash", "created_at"]
            summary = dict(zip(cols, run))
            cur.execute(
                "SELECT artifact_type, artifact_payload FROM trading.backtest_artifacts "
                "WHERE run_id = %s",
                (args.run_id,),
            )
            artifacts = {row[0]: row[1] for row in cur.fetchall()}
    finally:
        conn.close()

    print(json.dumps(summary, indent=2, default=str))
    if "equity" in artifacts:
        eq = artifacts["equity"]
        print(f"\nEquity curve points: {len(eq)}")
        if eq:
            print(f"  first: {eq[0]}")
            print(f"  last:  {eq[-1]}")
    return 0


def _fmt(v) -> str:
    if v is None:
        return "-"
    if isinstance(v, float):
        return f"{v:.2f}"
    return str(v)


def main():
    """Main CLI entry point."""
    parser = argparse.ArgumentParser(
        prog="backtest",
        description="Predict-A-Trade Backtesting CLI",
    )
    subparsers = parser.add_subparsers(dest="command", help="Available commands")

    # run
    p_run = subparsers.add_parser("run", help="Run a backtest")
    p_run.add_argument("--symbol", default="XAUUSD")
    p_run.add_argument("--strategy", default="STANDARD_SCALPING")
    p_run.add_argument("--timeframe", default="M5")
    p_run.add_argument("--start", default=None)
    p_run.add_argument("--end", default=None)
    p_run.add_argument("--balance", type=float, default=10000.0)
    p_run.add_argument("--seed", type=int, default=42)
    p_run.add_argument("--data-file", default=None)
    p_run.add_argument("--candles", type=int, default=500)
    p_run.add_argument("--allow-synthetic", action="store_true",
                      help="Permit running on SYNTHETIC data (self-tests only; never for reported results)")
    p_run.add_argument("--output", default=None)
    p_run.add_argument("--db-url", default=None,
                       help="PostgreSQL/TimescaleDB connection string. When set, "
                            "candles are read from market.candles and the run is "
                            "persisted to trading.backtest_runs for online retrieval.")
    p_run.set_defaults(func=cmd_run)

    # db-list — list persisted runs from TimescaleDB
    p_dbl = subparsers.add_parser("db-list", help="List backtest runs stored in TimescaleDB")
    p_dbl.add_argument("--db-url", required=True)
    p_dbl.add_argument("--symbol", default=None)
    p_dbl.add_argument("--strategy", default=None)
    p_dbl.add_argument("--limit", type=int, default=50)
    p_dbl.set_defaults(func=cmd_db_list)

    # db-show — show a persisted run (summary + equity curve) from TimescaleDB
    p_dbs = subparsers.add_parser("db-show", help="Show a stored backtest run from TimescaleDB")
    p_dbs.add_argument("--db-url", required=True)
    p_dbs.add_argument("run_id")
    p_dbs.set_defaults(func=cmd_db_show)

    # validate-data
    p_vd = subparsers.add_parser("validate-data", help="Validate data quality")
    p_vd.add_argument("--data-file", required=True)
    p_vd.add_argument("--symbol", default="XAUUSD")
    p_vd.add_argument("--timeframe", default="M5")
    p_vd.set_defaults(func=cmd_validate_data)

    # precompute
    p_pc = subparsers.add_parser("precompute", help="Precompute PTB features")
    p_pc.add_argument("--symbol", default="XAUUSD")
    p_pc.add_argument("--strategy", default="STANDARD_SCALPING")
    p_pc.add_argument("--timeframe", default="M5")
    p_pc.add_argument("--data-file", default=None)
    p_pc.add_argument("--seed", type=int, default=42)
    p_pc.add_argument("--output", default=None)
    p_pc.set_defaults(func=cmd_precompute)

    # walk-forward
    p_wf = subparsers.add_parser("walk-forward", help="Run walk-forward analysis")
    p_wf.add_argument("--symbol", default="XAUUSD")
    p_wf.add_argument("--strategy", default="STANDARD_SCALPING")
    p_wf.add_argument("--timeframe", default="M5")
    p_wf.add_argument("--data-file", default=None)
    p_wf.add_argument("--seed", type=int, default=42)
    p_wf.add_argument("--train-size", type=int, default=500)
    p_wf.add_argument("--test-size", type=int, default=100)
    p_wf.add_argument("--step-size", type=int, default=100)
    p_wf.set_defaults(func=cmd_walk_forward)

    # monte-carlo
    p_mc = subparsers.add_parser("monte-carlo", help="Run Monte Carlo analysis")
    p_mc.add_argument("--symbol", default="XAUUSD")
    p_mc.add_argument("--strategy", default="STANDARD_SCALPING")
    p_mc.add_argument("--timeframe", default="M5")
    p_mc.add_argument("--data-file", default=None)
    p_mc.add_argument("--seed", type=int, default=42)
    p_mc.add_argument("--runs", type=int, default=1000)
    p_mc.set_defaults(func=cmd_monte_carlo)

    # sensitivity
    p_sens = subparsers.add_parser("sensitivity", help="Run sensitivity analysis")
    p_sens.add_argument("--symbol", default="XAUUSD")
    p_sens.add_argument("--strategy", default="STANDARD_SCALPING")
    p_sens.add_argument("--timeframe", default="M5")
    p_sens.add_argument("--data-file", default=None)
    p_sens.add_argument("--seed", type=int, default=42)
    p_sens.add_argument("--perturbation", type=float, default=0.10)
    p_sens.set_defaults(func=cmd_sensitivity)

    # list
    p_list = subparsers.add_parser("list", help="List all backtest runs")
    p_list.add_argument("--output", default=None)
    p_list.set_defaults(func=cmd_list)

    # show
    p_show = subparsers.add_parser("show", help="Show details of a specific run")
    p_show.add_argument("run_id")
    p_show.add_argument("--output", default=None)
    p_show.set_defaults(func=cmd_show)

    args = parser.parse_args()

    if not args.command:
        parser.print_help()
        return 1

    return args.func(args)


if __name__ == "__main__":
    sys.exit(main())
