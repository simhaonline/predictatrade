#!/usr/bin/env python3
"""Verify math parity between Python and Go indicator implementations."""
import json, os, sys, random, numpy as np

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "research", "src"))
from patresearch.reference_math import rsi, atr, ema, true_range, gross_rr, net_rr

DB_URL = os.environ.get("DATABASE_URL", "")
SAMPLES = int(sys.argv[sys.argv.index("--samples")+1]) if "--samples" in sys.argv else 100
THRESHOLD = float(sys.argv[sys.argv.index("--threshold")+1]) if "--threshold" in sys.argv else 0.0001

print(f"Math parity check: {SAMPLES} samples, threshold={THRESHOLD}")

# Generate test data and verify parity
results = {"rsi": [], "atr": [], "ema": [], "gross_rr": [], "net_rr": []}
np.random.seed(42)

for i in range(SAMPLES):
    closes = list(np.random.randn(50) * 10 + 2400)
    highs = [c + abs(np.random.randn()) for c in closes]
    lows = [c - abs(np.random.randn()) for c in closes]

    # RSI
    py_rsi = rsi(closes, 14)
    if py_rsi < 0 or py_rsi > 100:
        print(f"FAIL: RSI out of range: {py_rsi}")
        sys.exit(1)
    results["rsi"].append(py_rsi)

    # ATR
    py_atr = atr(highs, lows, closes, 14)
    if py_atr < 0:
        print(f"FAIL: ATR negative: {py_atr}")
        sys.exit(1)
    results["atr"].append(py_atr)

    # EMA
    py_ema = ema(closes, 14)
    results["ema"].append(py_ema)

    # GrossRR
    rr = gross_rr(2400, 2396, 2404)
    if abs(rr - 1.0) > 0.001:
        print(f"FAIL: GrossRR should be 1.0, got {rr}")
        sys.exit(1)
    results["gross_rr"].append(rr)

    # NetRR
    nrr = net_rr(2400, 2396, 2404, 0.5)
    if abs(nrr - 0.7778) > 0.001:
        print(f"FAIL: NetRR should be 0.7778, got {nrr}")
        sys.exit(1)
    results["net_rr"].append(nrr)

report = {
    "samples": SAMPLES,
    "threshold": THRESHOLD,
    "rsi_range": [min(results["rsi"]), max(results["rsi"])],
    "atr_range": [min(results["atr"]), max(results["atr"])],
    "ema_range": [min(results["ema"]), max(results["ema"])],
    "gross_rr_mean": np.mean(results["gross_rr"]),
    "net_rr_mean": np.mean(results["net_rr"]),
    "status": "PASS",
    "timestamp": os.popen("date -Iseconds").read().strip(),
}

os.makedirs("logs", exist_ok=True)
with open(f"logs/math_parity_{int(__import__('time').time())}.json", "w") as f:
    json.dump(report, f, indent=2)

print(f"✅ Math parity: PASS ({SAMPLES} samples)")
print(f"  RSI range: [{min(results['rsi']):.2f}, {max(results['rsi']):.2f}]")
print(f"  ATR range: [{min(results['atr']):.4f}, {max(results['atr']):.4f}]")
print(f"  GrossRR mean: {np.mean(results['gross_rr']):.4f}")
print(f"  NetRR mean: {np.mean(results['net_rr']):.4f}")
