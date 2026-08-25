"""Predict-A-Trade Online Backtesting Service (FastAPI).

Exposes the research backtest engine over HTTP so any authenticated user can run
a backtest against data stored in TimescaleDB (market.candles) and retrieve a
report (equity curve + metrics) online.

Design:
- This service is a thin API over the already-verified research engine
  (patresearch.backtesting.*). It does NOT reimplement strategy logic.
- A run reads candles via DataLoader.from_database, runs the engine, and persists
  the result via reporting.db_writer (trading.backtest_runs + artifacts).
- Reports are served back from TimescaleDB so they are durable and shareable.

Run: uvicorn main:app --host 0.0.0.0 --port 8080  (from services/backtest-service,
with PYTHONPATH=../../research/src)
"""
from __future__ import annotations

import os
import sys
from datetime import datetime, timezone
from typing import Optional

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

# Make the research package importable regardless of CWD.
_SYS_PATH = os.environ.get("PAT_RESEARCH_SRC", os.path.join(os.path.dirname(__file__), "..", "..", "research", "src"))
if _SYS_PATH not in sys.path:
    sys.path.insert(0, os.path.abspath(_SYS_PATH))

from patresearch.backtesting.data.loader import DataLoader  # noqa: E402
from patresearch.backtesting.engine.core import BacktestEngine, BacktestConfig  # noqa: E402
from patresearch.backtesting.strategy.ptb_strategy import PTBStrategyAdapter  # noqa: E402
from patresearch.backtesting.reporting.db_writer import store_run  # noqa: E402
import psycopg2  # noqa: E402

DB_URL = os.environ.get(
    "BACKTEST_DB_URL",
    os.environ.get("DATABASE_URL",
                   "postgresql://pat_admin:pat_local_dev_only@127.0.0.1:5432/predictatrade?sslmode=disable"),
)

VALID_STRATEGIES = {
    "STANDARD_SCALPING", "ULTRA_SCALPING", "STANDARD_SWING", "TREND_SWING", "MARNIE_FIB"
}
VALID_TIMEFRAMES = {"M1", "M5", "M15", "M30", "H1", "H4", "D1"}

app = FastAPI(title="Predict-A-Trade Online Backtesting", version="1.0.0")


class BacktestRequest(BaseModel):
    symbol: str = "XAUUSD"
    strategy: str = "STANDARD_SCALPING"
    timeframe: str = "M5"
    start: Optional[str] = None
    end: Optional[str] = None
    balance: float = 10000.0
    seed: int = 42


class BacktestResponse(BaseModel):
    run_id: str
    status: str
    trades: int
    final_balance: Optional[float] = None
    total_return_pct: Optional[float] = None
    sharpe_ratio: Optional[float] = None
    equity_points: int = 0


def _parse_dt(s: Optional[str]):
    if not s:
        return None
    for fmt in ("%Y-%m-%dT%H:%M:%S", "%Y-%m-%d %H:%M:%S", "%Y-%m-%d"):
        try:
            return datetime.strptime(s, fmt).replace(tzinfo=timezone.utc)
        except ValueError:
            continue
    return datetime.fromisoformat(s).replace(tzinfo=timezone.utc)


@app.get("/health")
def health():
    try:
        c = psycopg2.connect(DB_URL, connect_timeout=3)
        c.close()
        db = "ok"
    except Exception as e:  # pragma: no cover
        db = f"unavailable: {e}"
    return {"status": "ok", "db": db}


@app.post("/backtest", response_model=BacktestResponse)
def run_backtest(req: BacktestRequest):
    if req.strategy not in VALID_STRATEGIES:
        raise HTTPException(400, f"unknown strategy {req.strategy}")
    if req.timeframe not in VALID_TIMEFRAMES:
        raise HTTPException(400, f"unknown timeframe {req.timeframe}")

    start = _parse_dt(req.start)
    end = _parse_dt(req.end)
    candles, meta = DataLoader.from_database(DB_URL, req.symbol, req.timeframe, start, end)
    if not candles:
        raise HTTPException(404,
            f"no candles in market.candles for {req.symbol} {req.timeframe} "
            f"in range {req.start}..{req.end}")

    config = BacktestConfig(symbol=req.symbol, strategy_id=req.strategy,
                            primary_timeframe=req.timeframe,
                            initial_balance=req.balance, random_seed=req.seed)
    engine = BacktestEngine(config)
    engine.set_strategy(PTBStrategyAdapter(req.strategy))
    result = engine.run(candles)
    if result.status != "COMPLETED":
        raise HTTPException(502, f"backtest failed: {result.status}")

    run_id = store_run(result, DB_URL, data_source="TIMESCALEDB", data_hash=meta.data_hash)

    eq_len = len(result.equity_curve)
    return BacktestResponse(
        run_id=run_id, status=result.status, trades=len(result.trades),
        final_balance=result.metrics.final_balance if result.metrics else None,
        total_return_pct=result.metrics.total_return_pct if result.metrics else None,
        sharpe_ratio=result.metrics.sharpe_ratio if result.metrics else None,
        equity_points=eq_len,
    )


@app.get("/backtest/{run_id}")
def get_backtest(run_id: str):
    conn = psycopg2.connect(DB_URL)
    try:
        with conn.cursor() as cur:
            cur.execute(
                "SELECT run_id, symbol, strategy_id, primary_timeframe, status, "
                "initial_balance, final_balance, total_return_pct, sharpe_ratio, "
                "sortino_ratio, max_drawdown_pct, win_rate_pct, profit_factor, "
                "expectancy, trades_count, no_trade_count, data_source, created_at "
                "FROM trading.backtest_runs WHERE run_id=%s", (run_id,))
            row = cur.fetchone()
            if not row:
                raise HTTPException(404, "run not found")
            cols = ["run_id", "symbol", "strategy_id", "primary_timeframe", "status",
                    "initial_balance", "final_balance", "total_return_pct", "sharpe_ratio",
                    "sortino_ratio", "max_drawdown_pct", "win_rate_pct", "profit_factor",
                    "expectancy", "trades_count", "no_trade_count", "data_source", "created_at"]
            summary = dict(zip(cols, row))
            cur.execute(
                "SELECT artifact_type, artifact_payload FROM trading.backtest_artifacts "
                "WHERE run_id=%s", (run_id,))
            artifacts = {r[0]: r[1] for r in cur.fetchall()}
    finally:
        conn.close()
    return {"summary": summary, "artifacts": artifacts}


@app.get("/backtest")
def list_backtests(symbol: Optional[str] = None, strategy: Optional[str] = None, limit: int = 50):
    conn = psycopg2.connect(DB_URL)
    try:
        where, params = [], []
        if symbol:
            where.append("symbol=%s"); params.append(symbol)
        if strategy:
            where.append("strategy_id=%s"); params.append(strategy)
        sql = ("SELECT run_id, symbol, strategy_id, primary_timeframe, status, "
               "total_return_pct, sharpe_ratio, trades_count, created_at "
               "FROM trading.backtest_runs")
        if where:
            sql += " WHERE " + " AND ".join(where)
        sql += " ORDER BY created_at DESC LIMIT %s"
        params.append(limit)
        with conn.cursor() as cur:
            cur.execute(sql, params)
            rows = cur.fetchall()
    finally:
        conn.close()
    return [dict(zip(
        ["run_id", "symbol", "strategy_id", "primary_timeframe", "status",
         "total_return_pct", "sharpe_ratio", "trades_count", "created_at"], r)) for r in rows]
