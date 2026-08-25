#!/usr/bin/env python3
"""
Quant Validation Script — OOS/Calibration/Promotion
Splits historical data into in-sample/out-of-sample, runs calibration,
backtest, and walk-forward promotion. Produces quant_evidence report.
Usage: python scripts/quant_validation.py --data-dir data/ --output artifacts/go_live_evidence/quant_evidence.json
"""
import argparse, json, os, sys
from datetime import datetime, timezone

def main():
    parser = argparse.ArgumentParser(description="Quant validation: OOS, calibration, promotion")
    parser.add_argument("--data-dir", default="data/")
    parser.add_argument("--output", default="artifacts/go_live_evidence/quant_evidence.json")
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()

    os.makedirs(os.path.dirname(args.output), exist_ok=True)

    if args.dry_run:
        # SIMULATION ONLY. No historical data is processed and NO evidence file
        # is written. Fabricated metrics must never be emitted as go-live
        # evidence (see AGENTS.md). Real validation is produced by the research
        # backtester / walk-forward pipeline against historical XAUUSD data.
        print("SIMULATION ONLY — NOT VALID EVIDENCE.")
        print("Real quant validation requires historical XAUUSD data + walk-forward engine.")
        print("No quant_evidence.json was written.")
        sys.exit(0)

    # Real validation path: requires --data-dir with historical candles and the
    # walk-forward/calibration engine. Fabricated output is intentionally absent.
    print("ERROR: Live quant validation requires historical data. No evidence produced.")
    sys.exit(1)

if __name__ == "__main__":
    main()
