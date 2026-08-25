"""End-to-end test: market.candles (TimescaleDB) -> online backtest -> stored report.

This proves the core "capture data into TimescaleDB -> any user can run online
backtesting -> generate reports" path. It requires a reachable PostgreSQL/TimescaleDB
(matching infra/env/realtime.env DATABASE_URL). If the DB is unreachable the test
is skipped rather than failing, so CI without a DB still passes.
"""
import os
import pytest
import psycopg2

DB_URL = os.environ.get(
    "PAT_DB_URL",
    "postgresql://pat_admin:pat_local_dev_only@127.0.0.1:5432/predictatrade?sslmode=disable",
)

SYMBOL = "XAUUSDT"  # test-only symbol so we never touch real market data
TIMEFRAME = "M5"
SOURCE = "PYTEST_E2E"


def _db_ok():
    try:
        c = psycopg2.connect(DB_URL, connect_timeout=3)
        c.close()
        return True
    except Exception:
        return False


pytestmark = pytest.mark.skipif(not _db_ok(), reason="TimescaleDB not reachable")


def _seed_candles(db_url, candles):
    c = psycopg2.connect(db_url)
    try:
        with c.cursor() as cur:
            cur.execute(
                "DELETE FROM market.candles WHERE symbol=%s AND timeframe=%s AND source=%s",
                (SYMBOL, TIMEFRAME, SOURCE),
            )
            for cd in candles:
                cur.execute(
                    "INSERT INTO market.candles "
                    "(time, symbol, timeframe, open, high, low, close, volume, source, is_closed) "
                    "VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s,TRUE) "
                    "ON CONFLICT (time, symbol, timeframe, source) DO NOTHING",
                    (cd["timestamp"], SYMBOL, TIMEFRAME, cd["open"], cd["high"],
                     cd["low"], cd["close"], cd["volume"], SOURCE),
                )
        c.commit()
    finally:
        c.close()


def test_postgres_backtest_roundtrip():
    from patresearch.backtesting.data.loader import DataLoader
    from patresearch.backtesting.cli import cmd_run
    from patresearch.backtesting.reporting.db_writer import store_run
    from patresearch.backtesting.engine.core import BacktestEngine, BacktestConfig
    from patresearch.backtesting.strategy.ptb_strategy import PTBStrategyAdapter
    from types import SimpleNamespace

    # 1) Seed synthetic candles into TimescaleDB (clearly labeled test source).
    seed, _ = DataLoader.generate_synthetic(SYMBOL, TIMEFRAME, n_candles=400, seed=7)
    _seed_candles(DB_URL, [
        {"timestamp": c.timestamp, "open": c.open, "high": c.high,
         "low": c.low, "close": c.close, "volume": c.volume}
        for c in seed
    ])

    # 2) Load from TimescaleDB (the path a user triggers online).
    candles, meta = DataLoader.from_database(DB_URL, SYMBOL, TIMEFRAME, source=SOURCE)
    assert len(candles) == 400, f"expected 400 candles from DB, got {len(candles)}"

    # 3) Run backtest on stored data and persist the report.
    config = BacktestConfig(symbol=SYMBOL, strategy_id="STANDARD_SCALPING",
                            primary_timeframe=TIMEFRAME, initial_balance=10000.0,
                            random_seed=42)
    engine = BacktestEngine(config)
    engine.set_strategy(PTBStrategyAdapter("STANDARD_SCALPING"))
    result = engine.run(candles)
    assert result.status == "COMPLETED"

    run_id = store_run(result, DB_URL, data_source="TIMESCALEDB", data_hash=meta.data_hash)
    assert run_id == result.run_id

    # 4) Verify the report is retrievable online (what the UI/service will query).
    c = psycopg2.connect(DB_URL)
    try:
        with c.cursor() as cur:
            cur.execute(
                "SELECT final_balance, total_return_pct, trades_count, status "
                "FROM trading.backtest_runs WHERE run_id=%s", (run_id,))
            row = cur.fetchone()
            assert row is not None, "backtest_runs row missing"
            assert row[3] == "COMPLETED"

            cur.execute(
                "SELECT artifact_type, jsonb_array_length(artifact_payload) "
                "FROM trading.backtest_artifacts WHERE run_id=%s "
                "AND jsonb_typeof(artifact_payload) = 'array'", (run_id,))
            arrays = {r[0]: r[1] for r in cur.fetchall()}
            assert arrays.get("equity", 0) > 0, "equity artifact missing/empty"

            cur.execute(
                "SELECT COUNT(*) FROM trading.backtest_artifacts "
                "WHERE run_id=%s AND artifact_type='metrics' "
                "AND artifact_payload != '{}'::jsonb", (run_id,))
            assert cur.fetchone()[0] == 1, "metrics artifact missing/empty"
    finally:
        c.close()

    # 5) CLI wiring: run --db-url should also persist.
    args = SimpleNamespace(symbol=SYMBOL, strategy="STANDARD_SCALPING", timeframe=TIMEFRAME,
                           start=None, end=None, balance=10000.0, seed=42,
                           data_file=None, candles=400, output=None, db_url=DB_URL)
    rc = cmd_run(args)
    assert rc == 0
