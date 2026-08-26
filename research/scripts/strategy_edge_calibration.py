"""Strategy edge diagnostic + auto-calibration advisor.

Reads REAL executed trades from `trading.trade_results` and computes per-strategy
edge metrics (profit factor, win rate, expectancy, avg hold, avg points). It then
emits a calibration recommendation that mirrors the engine's EdgeValidationGate
logic (see realtime/internal/gates/capital_gates.go):

  * ENABLE  — live ProfitFactor >= 1.0 over >= EDGE_NEGATIVE_MIN_SAMPLE_SIZE trades
  * DISABLE — negative edge (PF < 1.0) or insufficient sample ("UNKNOWN")

This is the research-side counterpart to the server-side veto: it tells the
strategy/quant team *which* parameters to re-calibrate and *why*, using only
genuine fills (no fabricated data). The actual parameter (re)optimization must be
done with a cost-aware walk-forward / OOS harness (see patresearch.backtesting),
never by editing thresholds blind.

Usage:
    DATABASE_URL=postgresql://... python3 research/scripts/strategy_edge_calibration.py
"""
from __future__ import annotations

import json
import os
import sys
from dataclasses import dataclass, asdict
from typing import Optional

try:
    import psycopg2
    import psycopg2.extras
except ImportError:
    sys.exit("psycopg2 is required: pip install psycopg2-binary")


DEFAULT_DSN = "postgresql://pat_admin:pat_local_dev_only@127.0.0.1:5432/predictatrade?sslmode=disable"
MIN_SAMPLE = int(os.environ.get("EDGE_NEGATIVE_MIN_SAMPLE_SIZE", "4"))


@dataclass
class StrategyEdge:
    strategy_id: str
    trades: int
    wins: int
    losses: int
    breakeven: int
    win_rate: float
    gross_win: float
    gross_loss: float
    profit_factor: float
    expectancy_points: float
    avg_hold_seconds: float
    net_pnl: float
    recommendation: str = "UNKNOWN"

    def to_dict(self) -> dict:
        return asdict(self)


def load_edges(dsn: str) -> list[StrategyEdge]:
    conn = psycopg2.connect(dsn)
    try:
        cur = conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor)
        cur.execute("""
            SELECT strategy_id,
                   COUNT(*)                                            AS trades,
                   SUM(CASE WHEN is_win THEN 1 ELSE 0 END)            AS wins,
                   SUM(CASE WHEN is_loss THEN 1 ELSE 0 END)           AS losses,
                   SUM(CASE WHEN is_breakeven THEN 1 ELSE 0 END)      AS breakeven,
                   COALESCE(SUM(pnl) FILTER (WHERE is_win), 0)        AS gross_win,
                   COALESCE(-SUM(pnl) FILTER (WHERE is_loss), 0)      AS gross_loss,
                   COALESCE(SUM(pnl), 0)                              AS net_pnl,
                   COALESCE(AVG(time_in_trade_seconds), 0)            AS avg_hold_seconds,
                   COALESCE(AVG(pnl_points), 0)                       AS avg_points
            FROM trading.trade_results
            GROUP BY strategy_id
            ORDER BY net_pnl ASC
        """)
        rows = cur.fetchall()
    finally:
        conn.close()

    edges: list[StrategyEdge] = []
    for r in rows:
        trades = int(r["trades"])
        gross_win = float(r["gross_win"])
        gross_loss = float(r["gross_loss"])
        pf = (gross_win / gross_loss) if gross_loss > 0 else (float("inf") if gross_win > 0 else 0.0)
        win_rate = (int(r["wins"]) / trades * 100.0) if trades else 0.0
        avg_points = float(r["avg_points"])
        expectancy = avg_points if trades else 0.0

        if trades < MIN_SAMPLE:
            rec = "UNKNOWN (insufficient sample)"
        elif pf >= 1.0:
            rec = "ENABLE"
        else:
            rec = "DISABLE (negative live edge)"

        edges.append(StrategyEdge(
            strategy_id=r["strategy_id"],
            trades=trades,
            wins=int(r["wins"]),
            losses=int(r["losses"]),
            breakeven=int(r["breakeven"]),
            win_rate=round(win_rate, 1),
            gross_win=round(gross_win, 2),
            gross_loss=round(gross_loss, 2),
            profit_factor=round(pf, 3) if pf != float("inf") else 999.0,
            expectancy_points=round(expectancy, 2),
            avg_hold_seconds=round(float(r["avg_hold_seconds"]), 1),
            net_pnl=round(float(r["net_pnl"]), 2),
            recommendation=rec,
        ))
    return edges


def main() -> int:
    dsn = os.environ.get("DATABASE_URL", DEFAULT_DSN)
    edges = load_edges(dsn)

    print(f"Strategy edge diagnostic — min_sample={MIN_SAMPLE}\n")
    print(f"{'strategy':<20}{'trades':>7}{'win%':>7}{'PF':>8}{'exp(pt)':>9}{'hold(s)':>9}{'net_pnl':>10}  recommendation")
    print("-" * 100)
    for e in edges:
        print(f"{e.strategy_id:<20}{e.trades:>7}{e.win_rate:>7}{e.profit_factor:>8}{e.expectancy_points:>9}{e.avg_hold_seconds:>9}{e.net_pnl:>10}  {e.recommendation}")

    actionable = [e for e in edges if e.recommendation.startswith("DISABLE")]
    print(f"\nSummary: {len(edges)} strategies analysed, {len(actionable)} flagged DISABLE (negative live edge).")
    print("These should be re-calibrated via a cost-aware walk-forward harness")
    print("(research/src/patresearch/backtesting) before being re-armed for live trading.")

    # Emit machine-readable report for downstream pipelines.
    out = os.environ.get("EDGE_REPORT_PATH", "research/backtest_reports/strategy_edge_diagnostic.json")
    try:
        with open(out, "w") as f:
            json.dump([e.to_dict() for e in edges], f, indent=2)
        print(f"\nWrote JSON report -> {out}")
    except OSError as exc:
        print(f"WARN: could not write report: {exc}", file=sys.stderr)

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
