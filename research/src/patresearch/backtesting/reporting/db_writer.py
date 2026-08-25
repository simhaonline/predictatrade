"""Persist backtest run results into PostgreSQL/TimescaleDB.

Stores the run summary into trading.backtest_runs and the equity curve +
metrics into trading.backtest_artifacts (JSONB payload, migration 072). This
makes backtests triggered on stored market.candles retrievable online by the
control plane / reporting UI without any file I/O.
"""
from __future__ import annotations

import json
from datetime import datetime, timezone
from typing import Optional

from .report import ReportGenerator  # reuse for file artifacts if needed


def _config_dict(config) -> dict:
    try:
        return config.to_dict()
    except AttributeError:
        pass
    # Fallback: best-effort serialization of known fields.
    return {
        "symbol": getattr(config, "symbol", None),
        "strategy_id": getattr(config, "strategy_id", None),
        "primary_timeframe": getattr(config, "primary_timeframe", None),
        "initial_balance": getattr(config, "initial_balance", None),
        "random_seed": getattr(config, "random_seed", None),
    }


def store_run(result, db_url: str, data_source: str = "TIMESCALEDB",
              data_hash: Optional[str] = None, git_commit: Optional[str] = None,
              application_version: Optional[str] = None) -> str:
    """Persist a BacktestRunResult and return its run_id.

    Raises on DB errors so callers can decide whether to fail or warn.
    """
    import psycopg2

    cfg = result.config
    m = result.metrics

    start_ts = None
    end_ts = None
    if result.equity_curve:
        try:
            start_ts = datetime.fromisoformat(result.equity_curve[0]["timestamp"])
            end_ts = datetime.fromisoformat(result.equity_curve[-1]["timestamp"])
        except (KeyError, TypeError, ValueError):
            pass

    now = datetime.now(timezone.utc)

    conn = psycopg2.connect(db_url)
    try:
        with conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO trading.backtest_runs (
                    run_id, symbol, strategy_id, strategy_mode, primary_timeframe,
                    start_timestamp, end_timestamp, initial_balance, random_seed,
                    status, bars_processed, trades_count, no_trade_count, blocked_count,
                    final_balance, total_return_pct, sharpe_ratio, sortino_ratio,
                    max_drawdown_pct, win_rate_pct, profit_factor, expectancy,
                    configuration, execution_assumptions, risk_config,
                    data_source, data_hash, git_commit_sha, application_version,
                    started_at, completed_at, duration_seconds
                ) VALUES (
                    %s,%s,%s,%s,%s, %s,%s,%s,%s, %s,%s,%s,%s,%s,
                    %s,%s,%s,%s,%s,%s,%s,%s, %s::jsonb,%s::jsonb,%s::jsonb,
                    %s,%s,%s,%s, %s,%s,%s
                )
                ON CONFLICT (run_id) DO UPDATE SET
                    status = EXCLUDED.status,
                    final_balance = EXCLUDED.final_balance,
                    total_return_pct = EXCLUDED.total_return_pct,
                    sharpe_ratio = EXCLUDED.sharpe_ratio,
                    sortino_ratio = EXCLUDED.sortino_ratio,
                    max_drawdown_pct = EXCLUDED.max_drawdown_pct,
                    win_rate_pct = EXCLUDED.win_rate_pct,
                    profit_factor = EXCLUDED.profit_factor,
                    expectancy = EXCLUDED.expectancy,
                    bars_processed = EXCLUDED.bars_processed,
                    trades_count = EXCLUDED.trades_count,
                    completed_at = EXCLUDED.completed_at,
                    duration_seconds = EXCLUDED.duration_seconds
                """,
                (
                    result.run_id, cfg.symbol, cfg.strategy_id, "ptb",
                    cfg.primary_timeframe,
                    start_ts, end_ts, cfg.initial_balance, cfg.random_seed,
                    result.status, result.bars_processed, len(result.trades),
                    result.no_trade_count, result.blocked_count,
                    m.final_balance if m else None,
                    m.total_return_pct if m else None,
                    m.sharpe_ratio if m else None,
                    m.sortino_ratio if m else None,
                    m.max_drawdown_pct if m else None,
                    m.win_rate_pct if m else None,
                    m.profit_factor if m else None,
                    m.expectancy if m else None,
                    json.dumps(_config_dict(cfg)),
                    json.dumps(getattr(cfg, "execution_config", None).__dict__
                               if getattr(cfg, "execution_config", None) else {}),
                    json.dumps({}),
                    data_source, data_hash, git_commit, application_version,
                    now, now, result.duration_seconds,
                ),
            )

            # Equity curve artifact (JSONB payload, migration 072).
            cur.execute(
                """
                INSERT INTO trading.backtest_artifacts
                    (run_id, artifact_type, file_path, artifact_payload)
                VALUES (%s, 'equity', %s, %s::jsonb)
                ON CONFLICT DO NOTHING
                """,
                (result.run_id, f"online:{result.run_id}:equity",
                 json.dumps(result.equity_curve)),
            )

            if m:
                cur.execute(
                    """
                    INSERT INTO trading.backtest_artifacts
                        (run_id, artifact_type, file_path, artifact_payload)
                    VALUES (%s, 'metrics', %s, %s::jsonb)
                    ON CONFLICT DO NOTHING
                    """,
                    (result.run_id, f"online:{result.run_id}:metrics",
                     json.dumps(m.__dict__)),
                )

        conn.commit()
    finally:
        conn.close()

    return result.run_id
