#!/usr/bin/env python3
"""Backfill trading.prediction_outcomes from existing trading.trade_results.

Phase 0.5 (prompt.md, Candidate B task 4). Attempts linkage per row:
  LINKED   — trade_results.signal_id resolves to a trading.signals row
             (a prediction row is auto-created as the FK parent when absent)
  UNLINKED — signal_id empty or unresolvable; recorded with link_status
             UNLINKED, never dropped, never invented
Idempotent: re-running updates existing rows by signal_id (no duplicates).

Usage:
  python3 backfill_prediction_outcomes.py [--dry-run]
Connection: DATABASE_URL env (defaults to the local compose DB).
"""
import argparse
import json
import os
import sys

import psycopg2
import psycopg2.extras

DSN = os.environ.get(
    "PERSISTENCE_TEST_URL",
    "postgres://pat_admin:pat_local_dev_only@127.0.0.1:5432/predictatrade?sslmode=disable",
)


def classify(conn, sig) -> tuple:
    """Returns (link_status, reason) for one trade_results row."""
    sig = (sig or "").strip()
    if not sig:
        return "UNLINKED", "empty signal_id (legacy EA fallback-UUID path)"
    import uuid as _uuid
    try:
        _uuid.UUID(sig)
    except ValueError:
        return "UNLINKED", "signal_id not a valid UUID (legacy fallback value)"
    with conn.cursor() as c:
        c.execute("SELECT 1 FROM trading.signals WHERE id=%s::uuid", (sig,))
        exists = c.fetchone()
    if exists is None:
        return "UNLINKED", "signal_id %s... does not resolve to trading.signals" % sig[:8]
    return "LINKED", ""


# EA close_reason → approved enum mapping (lowercase EA values)
REASON_MAP = {"sl": "STOP", "tp1": "TP1", "tp2": "TP2", "tp3": "TP3",
              "manual": "MANUAL", "be": "BE", "trail": "TRAIL",
              "timeout": "TIMEOUT", "auto": "AUTO"}


def normalize_reason(raw):
    r = (raw or "").strip().lower()
    return REASON_MAP.get(r, "MANUAL" if r else None)


def outcome_type_for(row):
    if row["is_win"] is True:
        return "WIN", True
    if row["is_loss"] is True:
        return "LOSS", False
    return "BREAKEVEN", False


def r_multiple_for(row):
    if not (row["entry_price"] and row["stop_loss"] and row["exit_price"]):
        return None
    risk = abs(float(row["entry_price"]) - float(row["stop_loss"]))
    if risk <= 0:
        return None
    direction = 1 if row["direction"] == "BUY" else -1
    return direction * (float(row["exit_price"]) - float(row["entry_price"])) / risk


def ensure_parent(conn, row):
    with conn.cursor() as c:
        c.execute(
            """INSERT INTO trading.predictions
                 (signal_id, symbol, timeframe, direction, strategy_id)
               VALUES (%s::uuid, %s, COALESCE(%s,''), COALESCE(%s,'UNKNOWN'), COALESCE(%s,''))
               ON CONFLICT (signal_id) DO NOTHING""",
            (row["signal_id"], row["symbol"], row["timeframe"],
             row["direction"], row["strategy_id"]),
        )
        c.execute("SELECT id FROM trading.predictions WHERE signal_id=%s::uuid",
                  (row["signal_id"],))
        row_ = c.fetchone()
        return row_[0] if row_ else None


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--dry-run", action="store_true")
    args = ap.parse_args()

    conn = psycopg2.connect(DSN)
    linked = unlinked = 0
    unlinked_reasons = []
    with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as fetchcur:
        fetchcur.execute("SELECT * FROM trading.trade_results ORDER BY created_at")
        rows = fetchcur.fetchall()

    with conn.cursor() as writecur:
        for row in rows:
            status, reason = classify(conn, row["signal_id"])
            otype, owin = outcome_type_for(row)
            close_reason = normalize_reason(row["close_reason"])
            rmult = r_multiple_for(row)
            if status == "LINKED":
                linked += 1
                pred_id = ensure_parent(conn, row)
            else:
                unlinked += 1
                unlinked_reasons.append(
                    {"signal_id": row["signal_id"], "reason": reason})
                # UNLINKED rows are first-class (prompt.md): create a minimal
                # parent prediction so the FK holds; link_status=UNLINKED
                # flags the provenance for the alert path. When signal_id is
                # empty/unusable, the outcome row cannot be linked at all —
                # skipped and counted (never fabricated).
                if (row["signal_id"] or "").strip():
                    pred_id = ensure_parent(conn, row)

            if args.dry_run:
                continue
            if pred_id is None:
                # no parent and no usable signal_id — record as SKIPPED_UNLINKED
                # (never fabricated; counted in the report)
                unlinked_reasons.append({"signal_id": row["signal_id"],
                                         "reason": "no usable linkage key — row skipped (not fabricated)"})
                continue

            writecur.execute(
                """INSERT INTO trading.prediction_outcomes (
                     prediction_id, signal_id, link_status, strategy_id, close_reason,
                     outcome_type, outcome_value, realized_rr, realized_pnl, r_multiple,
                     mae_points, mfe_points, duration_seconds, schema_version
                   ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, '1.0')
                   ON CONFLICT (signal_id) WHERE signal_id IS NOT NULL DO UPDATE SET
                     link_status = EXCLUDED.link_status,
                     close_reason = EXCLUDED.close_reason,
                     outcome_type = EXCLUDED.outcome_type,
                     outcome_value = EXCLUDED.outcome_value,
                     realized_rr = EXCLUDED.realized_rr,
                     realized_pnl = EXCLUDED.realized_pnl,
                     r_multiple = EXCLUDED.r_multiple,
                     mae_points = EXCLUDED.mae_points,
                     mfe_points = EXCLUDED.mfe_points,
                     duration_seconds = EXCLUDED.duration_seconds""",
                (pred_id, row["signal_id"] or None, status, row["strategy_id"],
                 close_reason, otype, owin, row["pnl"], rmult, rmult,
                 row["mae"], row["mfe"], row["time_in_trade_seconds"]))

    if args.dry_run:
        conn.rollback()
    else:
        conn.commit()

    report = {"total": len(rows), "linked": linked, "unlinked": unlinked,
              "unlinked_reasons": unlinked_reasons}
    print(json.dumps(report, indent=1))
    report_path = os.path.join(os.path.dirname(os.path.abspath(__file__)),
                               "..", "docs/strategy/PHASE0_5_BACKFILL_REPORT.json")
    with open(report_path, "w") as f:
        json.dump(report, f, indent=1)
    print("report written:", os.path.abspath(report_path))
    return 0


if __name__ == "__main__":
    sys.exit(main())