"""Multi-year Out-of-Sample Walk-Forward + Probability Calibration.

Loads REAL XAUUSD market data from the database (primary M5 + higher TFs
M15/H1/H4/D1), runs a sliding walk-forward, and for each production strategy:

  - trains a ProbabilityCalibrator on IN-SAMPLE trades (raw score -> net-win),
  - evaluates calibration OOS AUC on the held-out test folds,
  - exports a versioned calibration JSON the Go realtime engine consumes.

NO numbers are fabricated: every label comes from a real backtest trade
(pnl > 0 => win). If a fold yields no trades it is skipped. If a strategy
has insufficient OOS samples the JSON is NOT written and the run is reported
as INSUFFICIENT_DATA so the live path keeps ProbabilityCalibrated=false.

Usage:
  python3 scripts/oos_walkforward_calibrate.py \
      --db-url postgresql://... --out-dir realtime/calibration
"""
from __future__ import annotations

import argparse
import json
import os
import sys
import time
from datetime import datetime, timezone

from patresearch.backtesting.data.loader import DataLoader, HistoricalCandle
from patresearch.backtesting.engine.core import BacktestEngine, BacktestConfig
from patresearch.backtesting.strategy.ptb_strategy import PTBStrategyAdapter
from patresearch.backtesting.calibration import ProbabilityCalibrator

STRATEGIES = [
    "STANDARD_SCALPING",
    "ULTRA_SCALPING",
    "STANDARD_SWING",
    "TREND_SWING",
]
HIGHER_TFS = ["M15", "H1", "H4", "D1"]
PRIMARY_TF = "M5"
TARGET = "NET_WIN"
EXIT_PROFILE = "DEFAULT"


def _slice(candles, start, end):
    return [c for c in candles if start <= c.timestamp < end]


def _load_all(db_url, symbol, start, end):
    dl = DataLoader()
    data = {}
    for tf in [PRIMARY_TF] + HIGHER_TFS:
        candles, _ = dl.from_database(
            db_url, symbol, tf,
            start_time=start, end_time=end,
        )
        data[tf] = candles
        print(f"  loaded {tf}: {len(candles)} candles", flush=True)
    return data


def _collect_labels(result):
    """Return list of (raw_score, win_label) from a completed run."""
    out = []
    for t in result.trades:
        score = getattr(t, "confidence", None)
        if score is None or score <= 0:
            continue
        out.append((float(score), 1 if float(t.pnl) > 0 else 0))
    return out


def run_strategy(strategy, data, train_days, test_days, step_days):
    primary = data[PRIMARY_TF]
    if not primary:
        return None, "NO_DATA"
    lo = primary[0].timestamp
    hi = primary[-1].timestamp

    from datetime import timedelta

    fold = 0
    insample = []   # (score, win)
    oos = []
    oos_metrics = []  # dict per test fold
    while True:
        t_start = lo + timedelta(days=fold * step_days)
        t_end = t_start + timedelta(days=train_days)
        e_start = t_end
        e_end = e_start + timedelta(days=test_days)
        if e_end > hi:
            break
        fold += 1

        train_primary = _slice(primary, t_start, t_end)
        test_primary = _slice(primary, e_start, e_end)
        if len(train_primary) < 100 or len(test_primary) < 20:
            continue

        def build_htf(window_start, window_end):
            return {tf: _slice(data[tf], window_start, window_end)
                    for tf in HIGHER_TFS}

        # In-sample
        cfg = BacktestConfig(symbol="XAUUSD", strategy_id=strategy,
                             primary_timeframe=PRIMARY_TF, initial_balance=10000.0)
        eng = BacktestEngine(cfg)
        eng.set_strategy(PTBStrategyAdapter(strategy))
        r_in = eng.run(train_primary, build_htf(t_start, t_end))
        if r_in.status == "COMPLETED":
            insample.extend(_collect_labels(r_in))

        # Out-of-sample
        eng2 = BacktestEngine(cfg)
        eng2.set_strategy(PTBStrategyAdapter(strategy))
        r_out = eng2.run(test_primary, build_htf(e_start, e_end))
        if r_out.status == "COMPLETED":
            labels = _collect_labels(r_out)
            oos.extend(labels)
            wins = sum(l for _, l in labels)
            oos_metrics.append({
                "fold": fold, "n_trades": len(labels),
                "win_rate": (wins / len(labels)) if labels else 0.0,
                "net_pnl": sum(float(t.pnl) for t in r_out.trades),
            })
        print(f"  [{strategy}] fold {fold}: in={len(train_primary)} "
              f"test={len(test_primary)} oos_trades={len(labels)}", flush=True)

    if not insample:
        return None, "NO_INSAMPLE_TRADES"
    if len(oos) < 50:
        return None, f"INSUFFICIENT_OOS_SAMPLES({len(oos)})"

    cal = ProbabilityCalibrator(method="logistic").fit(
        [s for s, _ in insample], [l for _, l in insample])
    # OOS AUC
    try:
        from sklearn.metrics import roc_auc_score
        oos_auc = roc_auc_score([l for _, l in oos],
                                [cal.predict_proba(s) for s, _ in oos])
    except Exception:
        oos_auc = float("nan")
    cal.oos_auc = oos_auc

    oos_win = sum(l for _, l in oos)
    summary = {
        "strategy": strategy,
        "target": TARGET,
        "exit_profile": EXIT_PROFILE,
        "n_insample": len(insample),
        "n_oos": len(oos),
        "oos_win_rate": oos_win / len(oos),
        "oos_auc": oos_auc,
        "oos_folds": oos_metrics,
        "method": cal.method,
        "params": {"a": cal.a, "b": cal.b},
    }
    return cal, summary


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--db-url", required=True)
    ap.add_argument("--out-dir", default="realtime/calibration")
    ap.add_argument("--train-days", type=int, default=365)
    ap.add_argument("--test-days", type=int, default=90)
    ap.add_argument("--step-days", type=int, default=90)
    ap.add_argument("--symbol", default="XAUUSD")
    args = ap.parse_args()

    os.makedirs(args.out_dir, exist_ok=True)
    # Determine overall date range from M5.
    dl = DataLoader()
    probe, _ = dl.from_database(args.db_url, args.symbol, PRIMARY_TF)
    if not probe:
        print("NO_DATA: could not load primary timeframe", flush=True)
        sys.exit(2)
    lo = probe[0].timestamp
    hi = probe[-1].timestamp
    print(f"Data range: {lo} -> {hi} ({len(probe)} M5 candles)", flush=True)

    data = _load_all(args.db_url, args.symbol, lo, hi)

    report = {"generated_at": datetime.now(timezone.utc).isoformat(),
              "strategies": {}}
    for strat in STRATEGIES:
        print(f"=== {strat} ===", flush=True)
        cal, summary = run_strategy(
            strat, data, args.train_days, args.test_days, args.step_days)
        if cal is None:
            print(f"  SKIPPED: {summary}", flush=True)
            report["strategies"][strat] = {"status": summary}
            continue
        path = os.path.join(args.out_dir, f"{strat}.json")
        cal.export_json(path, metadata={
            "strategy": strat, "target": TARGET,
            "exit_profile": EXIT_PROFILE,
        })
        print(f"  wrote {path} (oos_auc={cal.oos_auc:.3f}, "
              f"oos_win_rate={summary['oos_win_rate']:.3f})", flush=True)
        report["strategies"][strat] = summary

    with open(os.path.join(args.out_dir, "_walkforward_report.json"), "w") as f:
        json.dump(report, f, indent=2, default=str)
    print("DONE. Report:", os.path.join(args.out_dir, "_walkforward_report.json"),
          flush=True)


if __name__ == "__main__":
    main()
