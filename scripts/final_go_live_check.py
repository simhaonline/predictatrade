#!/usr/bin/env python3
"""
Final Go-Live Readiness Checker
Verifies all evidence files exist and contain required fields.
Outputs validation_passed.txt or validation_failed.txt with details.
Usage: python scripts/final_go_live_check.py --evidence-dir artifacts/go_live_evidence/
"""
import argparse, csv, json, os, sys, subprocess
from datetime import datetime, timezone
from pathlib import Path

def check_file(path, label):
    if not os.path.exists(path):
        return False, f"MISSING: {label} ({path})"
    return True, f"FOUND: {label} ({path})"

def check_broker_observations(path):
    if not os.path.exists(path):
        return False, "MISSING: broker_observations.csv"
    with open(path) as f:
        reader = csv.DictReader(f)
        rows = list(reader)
    if len(rows) < 1000:
        return False, f"INSUFFICIENT: {len(rows)} rows (need 1000+)"
    return True, f"OK: {len(rows)} observations"

def check_provider_manifest(path):
    if not os.path.exists(path):
        return False, "MISSING: provider_manifest.json"
    with open(path) as f:
        data = json.load(f)
    if "provenance_hash" not in data:
        return False, "MISSING: provenance_hash field"
    if "providers" not in data:
        return False, "MISSING: providers field"
    return True, f"OK: {list(data['providers'].keys())} signed"

def check_webhook_log(path):
    if not os.path.exists(path):
        return False, "MISSING: webhook_signed_delivery.log"
    with open(path) as f:
        lines = f.readlines()
    if len(lines) < 4:
        return False, f"INSUFFICIENT: {len(lines)} events (need 4+)"
    return True, f"OK: {len(lines)} webhook events"

def check_restore_report(path):
    if not os.path.exists(path):
        return False, "MISSING: restore_report.md"
    with open(path) as f:
        content = f.read()
    if "RTO" not in content:
        return False, "MISSING: RTO metric"
    if "RPO" not in content:
        return False, "MISSING: RPO metric"
    return True, f"OK: RTO/RPO metrics present"

def check_quant_evidence(path):
    if not os.path.exists(path):
        return False, "MISSING: quant_evidence.json"
    with open(path) as f:
        data = json.load(f)
    # Reject simulated/fabricated evidence (AGENTS.md: never fabricate metrics).
    if data.get("mode") == "DRY_RUN":
        return False, "REJECTED: quant_evidence.json is a DRY_RUN simulation (not valid evidence)"
    if "strategies" not in data:
        return False, "MISSING: strategies field"
    if "overall_pass" not in data:
        return False, "MISSING: overall_pass field"
    # Evidence must prove it was computed from real data, not invented.
    if "provenance" not in data and "data_hash" not in data and "source" not in data:
        return False, "MISSING: provenance/data_hash/source (cannot verify real data was used)"
    return True, f"OK: {len(data['strategies'])} strategies validated"


def check_strategy_change_gate(ev_dir: Path, strategy_id: str) -> tuple:
    """Invoke scripts/strategy_change_gate.py for a strategy's real backtest result.

    Extends the final go-live readiness check: every production strategy must pass
    the quantitative promotion gate (leakage-safe, OOS net expectancy > 0 after
    costs, sample sufficiency, calibration AUC reported). Missing/insufficient data
    fails loudly — never fabricated.
    """
    script = Path(__file__).resolve().parent / "strategy_change_gate.py"
    if not script.exists():
        return False, f"MISSING: {script}"
    bt_result = ev_dir / f"backtest_{strategy_id}.json"
    if not bt_result.exists():
        return False, f"MISSING: {bt_result.name} (no real backtest result to gate)"
    try:
        proc = subprocess.run(
            [sys.executable, str(script), "--strategy-id", strategy_id,
             "--backtest-result", str(bt_result)],
            capture_output=True, text=True,
        )
    except Exception as e:  # pragma: no cover
        return False, f"ERROR invoking gate: {e}"
    if proc.returncode != 0:
        # Surface the gate's reason.
        reason = proc.stdout.strip().splitlines()
        reason = reason[-1] if reason else proc.stderr.strip().splitlines()[-1] if proc.stderr.strip() else "GATE FAIL"
        return False, f"GATE FAIL: {strategy_id}: {reason}"
    return True, f"OK: {strategy_id} quant gate passed"
    with open(path) as f:
        data = json.load(f)
    # Reject simulated/fabricated evidence (AGENTS.md: never fabricate metrics).
    if data.get("mode") == "DRY_RUN":
        return False, "REJECTED: quant_evidence.json is a DRY_RUN simulation (not valid evidence)"
    if "strategies" not in data:
        return False, "MISSING: strategies field"
    if "overall_pass" not in data:
        return False, "MISSING: overall_pass field"
    # Evidence must prove it was computed from real data, not invented.
    if "provenance" not in data and "data_hash" not in data and "source" not in data:
        return False, "MISSING: provenance/data_hash/source (cannot verify real data was used)"
    return True, f"OK: {len(data['strategies'])} strategies validated"

def main():
    parser = argparse.ArgumentParser(description="Final go-live readiness checker")
    parser.add_argument("--evidence-dir", default="artifacts/go_live_evidence/")
    args = parser.parse_args()

    ev = Path(args.evidence_dir)
    checks = [
        ("Broker Observations", check_broker_observations(ev / "broker_observations.csv")),
        ("Provider Manifest", check_provider_manifest(ev / "provider_manifest.json")),
        ("Webhook Delivery Log", check_webhook_log(ev / "webhook_signed_delivery.log")),
        ("Restore Report", check_restore_report(ev / "restore_report.md")),
        ("Quant Evidence", check_quant_evidence(ev / "quant_evidence.json")),
    ]

    # Strategy-change quant gates (one per production strategy). These require a
    # REAL backtest result JSON (backtest_<STRATEGY>.json) in the evidence dir.
    for sid in ("STANDARD_SCALPING", "ULTRA_SCALPING", "STANDARD_SWING", "TREND_SWING"):
        checks.append((f"Strategy Gate: {sid}", check_strategy_change_gate(ev, sid)))

    print("\n" + "=" * 60)
    print("FINAL GO-LIVE READINESS CHECK")
    print("=" * 60)

    all_pass = True
    for label, (passed, msg) in checks:
        status = "✅" if passed else "❌"
        print(f"  {status} {label}: {msg}")
        if not passed:
            all_pass = False

    print("=" * 60)
    if all_pass:
        print("  RESULT: ALL CHECKS PASSED — READY FOR GO-LIVE")
        with open(ev / "validation_passed.txt", "w") as f:
            f.write(f"PASSED at {datetime.now(timezone.utc).isoformat()}\n")
    else:
        print("  RESULT: CHECKS FAILED — NOT READY FOR GO-LIVE")
        with open(ev / "validation_failed.txt", "w") as f:
            f.write(f"FAILED at {datetime.now(timezone.utc).isoformat()}\n")

    print("=" * 60)
    return 0 if all_pass else 1

if __name__ == "__main__":
    sys.exit(main())
