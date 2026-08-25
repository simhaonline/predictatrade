#!/usr/bin/env python3
"""
Strategy-Change Quant Gate (CI gate for go-live promotion).

Enforces quantitative go-live gates for a single strategy before its calibration
model / trade logic may be promoted to live. It reads a REAL backtest result
JSON (produced by the research backtest / walk-forward pipeline) and exits
non-zero on ANY failure. It NEVER fabricates numbers — if inputs are missing or
inconsistent it fails loudly (AGENTS.md: never fabricate performance/metrics).

Gates (per strategy):
  (a) Leakage-safe      — feature timestamps must be strictly < signal timestamps.
  (b) OOS computed      — win/expectancy reported on held-out data (oos_total_trades > 0).
  (c) Net expectancy>0  — after realistic costs (spread, commission, slippage, swap).
  (d) Sample sufficiency — oos_total_trades >= min (default 100).
  (e) Calibration AUC   — reported and finite (must be present; warning if < 0.5).

Usage:
  python scripts/strategy_change_gate.py \
      --strategy-id STANDARD_SCALPING \
      --backtest-result artifacts/go_live_evidence/backtest_STANDARD_SCALPING.json \
      [--calibration-json artifacts/.../cal_STANDARD_SCALPING.json] \
      [--min-oos-trades 100]
"""
import argparse
import json
import os
import sys

# Mirror the style of scripts/quant_validation.py (no live validation without data).
VALID_STRATEGIES = {
    "STANDARD_SCALPING",
    "ULTRA_SCALPING",
    "STANDARD_SWING",
    "TREND_SWING",
    "MARNIE_FIB",
}


def fail(msg):
    print(f"GATE FAIL: {msg}")
    sys.exit(1)


def warn(msg):
    print(f"GATE WARN: {msg}")


def main():
    parser = argparse.ArgumentParser(description="Strategy-change quant gate (CI)")
    parser.add_argument("--strategy-id", required=True)
    parser.add_argument("--backtest-result", required=True)
    parser.add_argument("--calibration-json", default="")
    parser.add_argument("--min-oos-trades", type=int, default=100)
    args = parser.parse_args()

    if args.strategy_id not in VALID_STRATEGIES:
        fail(f"unknown strategy_id: {args.strategy_id}")

    if not os.path.exists(args.backtest_result):
        fail(f"backtest result missing: {args.backtest_result} (cannot gate without real data)")

    with open(args.backtest_result) as f:
        data = json.load(f)

    # Strategy id consistency
    if data.get("strategy_id") and data["strategy_id"] != args.strategy_id:
        fail(f"strategy_id mismatch: file={data['strategy_id']} arg={args.strategy_id}")

    print(f"Strategy-change gate: {args.strategy_id}")
    print("-" * 60)

    # (a) Leakage-safe
    leakage_safe = data.get("leakage_safe", data.get("oos", {}).get("leakage_safe", False))
    if not leakage_safe:
        fail("LEAKAGE: feature timestamps are not strictly before signal timestamps "
             "(look-ahead risk). Gate blocked.")
    print("[a] leakage-safe: PASS")

    # Pull OOS metrics (accept top-level or nested under 'oos')
    oos = data.get("oos", {})
    oos_total = int(data.get("oos_total_trades", oos.get("total_trades", 0)))
    oos_exp = float(data.get("oos_net_expectancy", oos.get("net_expectancy", float("nan"))))

    # (b) OOS computed
    if oos_total <= 0:
        fail("OOS: no out-of-sample trades reported — expectancy not computed on held-out data.")
    print(f"[b] oos-computed: PASS (oos_trades={oos_total})")

    # (c) Net expectancy > 0 after costs
    if oos_exp != oos_exp:  # NaN
        fail("OOS net expectancy missing from backtest result.")
    if not (oos_exp > 0):
        fail(f"OOS net expectancy not positive after costs: {oos_exp:.4f} "
             f"(must be > 0 for promotion).")
    print(f"[c] net-expectancy>0: PASS (oos_net_expectancy={oos_exp:.4f})")

    # (d) Sample sufficiency
    if oos_total < args.min_oos_trades:
        fail(f"OOS sample insufficient: {oos_total} < min {args.min_oos_trades}.")
    print(f"[d] sample-sufficiency: PASS (>= {args.min_oos_trades})")

    # (e) Calibration AUC reported
    cal_auc = data.get("calibration_auc", oos.get("calibration_auc", float("nan")))
    if cal_auc != cal_auc:  # NaN -> missing
        fail("calibration AUC not reported in backtest result.")
    print(f"[e] calibration-auc: reported={cal_auc:.4f}")
    if cal_auc < 0.5:
        warn(f"calibration AUC {cal_auc:.4f} <= 0.5 (no better than random) — review model.")

    # Optional: if a calibration JSON is supplied, verify it parses and version matches.
    if args.calibration_json:
        if not os.path.exists(args.calibration_json):
            fail(f"calibration json missing: {args.calibration_json}")
        with open(args.calibration_json) as f:
            cal = json.load(f)
        if cal.get("strategy") != args.strategy_id:
            fail(f"calibration strategy mismatch: {cal.get('strategy')} != {args.strategy_id}")
        if cal.get("version") != "1.0.0":
            fail(f"calibration schema version unsupported: {cal.get('version')}")
        print(f"[+] calibration json: OK (method={cal.get('method')}, "
              f"n_samples={cal.get('n_samples')}, oos_auc={cal.get('oos_auc')})")

    print("-" * 60)
    print(f"RESULT: {args.strategy_id} GATE PASSED")
    sys.exit(0)


if __name__ == "__main__":
    main()
