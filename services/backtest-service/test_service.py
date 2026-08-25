"""Integration test for the online backtesting FastAPI service.

Seeds candles into TimescaleDB, then drives the service API:
  POST /backtest  -> run on stored data, returns run_id + summary
  GET  /backtest/{run_id} -> stored report (summary + equity/metrics artifacts)
  GET  /backtest  -> list runs
Skips if TimescaleDB is unreachable.
"""
import os
import pytest
import psycopg2

DB_URL = os.environ.get(
    "PAT_DB_URL",
    "postgresql://pat_admin:pat_local_dev_only@127.0.0.1:5432/predictatrade?sslmode=disable",
)
SYMBOL = "XAUUSDT"
TF = "M5"


def _db_ok():
    try:
        c = psycopg2.connect(DB_URL, connect_timeout=3)
        c.close()
        return True
    except Exception:
        return False


pytestmark = pytest.mark.skipif(not _db_ok(), reason="TimescaleDB not reachable")


def _seed():
    from patresearch.backtesting.data.loader import DataLoader
    seed, _ = DataLoader.generate_synthetic(SYMBOL, TF, n_candles=400, seed=11)
    c = psycopg2.connect(DB_URL)
    try:
        with c.cursor() as cur:
            cur.execute("DELETE FROM market.candles WHERE symbol=%s AND timeframe=%s AND source=%s",
                        (SYMBOL, TF, "PYTEST_SVC"))
            for cd in seed:
                cur.execute(
                    "INSERT INTO market.candles "
                    "(time, symbol, timeframe, open, high, low, close, volume, source, is_closed) "
                    "VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s,TRUE) "
                    "ON CONFLICT (time, symbol, timeframe, source) DO NOTHING",
                    (cd.timestamp, SYMBOL, TF, cd.open, cd.high, cd.low, cd.close, cd.volume, "PYTEST_SVC"))
        c.commit()
    finally:
        c.close()


@pytest.fixture(scope="module")
def client():
    os.environ["BACKTEST_DB_URL"] = DB_URL
    _seed()
    # Ensure research src on path (service main.py also does this, but tests import directly).
    import sys
    sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", "research", "src")))
    from fastapi.testclient import TestClient
    from main import app  # service module
    return TestClient(app)


def test_backtest_service_flow(client):
    # 1) missing data path
    r = client.post("/backtest", json={"symbol": "NOPE", "strategy": "STANDARD_SCALPING",
                                        "timeframe": "M5", "start": "2024-01-01", "end": "2024-02-01"})
    assert r.status_code == 404

    # 2) run on stored data
    r = client.post("/backtest", json={"symbol": SYMBOL, "strategy": "STANDARD_SCALPING",
                                        "timeframe": TF, "start": "2024-01-01", "end": "2024-12-31",
                                        "balance": 10000.0, "seed": 42})
    assert r.status_code == 200, r.text
    body = r.json()
    assert body["status"] == "COMPLETED"
    assert body["run_id"]
    run_id = body["run_id"]

    # 3) retrieve stored report
    r = client.get(f"/backtest/{run_id}")
    assert r.status_code == 200, r.text
    data = r.json()
    assert data["summary"]["run_id"] == run_id
    assert "equity" in data["artifacts"]
    assert len(data["artifacts"]["equity"]) > 0

    # 4) list
    r = client.get("/backtest", params={"symbol": SYMBOL})
    assert r.status_code == 200
    assert any(item["run_id"] == run_id for item in r.json())
