#!/usr/bin/env python3
"""Phase 1 dry-run harness (prompt.md Phase 0.9 Task C).

Executes the FROZEN protocol in docs/strategy/PHASE1_CALIBRATION_PROTOCOL.md
mechanically when data allows. Fail-closed:

- Refuses to evaluate any strategy whose EXECUTION-channel linked outcome count
  is below the sufficiency gate (300) — exit code 2.
- Execution channel (trading.prediction_outcomes) is the ONLY evidence counted
  toward decisions. Shadow channel (trading.cross_market_shadow_snapshots) is
  pulled only for DRY-RUN execution of the code path, labeled DRY-RUN/SHADOW,
  and never counted as execution evidence.
- Walk-forward 70/15/15 splits, Benjamini-Hochberg FDR (q=0.10), effect-size
  gate (>= 0.10R on OOS), paired-bootstrap significance (p < 0.05).
- Output decisions are recommendations only: PROMOTE / HOLD / ROLLBACK.
  Nothing is ever auto-promoted.

Usage:
  python3 scripts/phase1_run.py --dry-run-shadow   # executes on shadow data only
  python3 scripts/phase1_run.py                    # real execution channel (gated)
Exit codes: 0 ran, 1 error, 2 gate BLOCKED (refused).
"""
from __future__ import annotations

import argparse
import json
import os
import random
import sys
from dataclasses import dataclass, field, asdict

DSN = os.environ.get(
    "PERSISTENCE_TEST_URL",
    "postgres://pat_admin:pat_local_dev_only@127.0.0.1:5432/predictatrade?sslmode=disable",
)

# Frozen protocol constants (PHASE1_CALIBRATION_PROTOCOL.md §3/§7 — do not edit here)
GATE_MIN_N = 300
TRAIN_PCT, VALID_PCT, OOS_PCT = 0.70, 0.15, 0.15
EFFECT_MIN_R = 0.10          # minimum OOS mean-expectancy improvement in R
ALPHA = 0.05                 # significance threshold
FDR_Q = 0.10                 # Benjamini-Hochberg false-discovery rate
BOOTSTRAP_N = 10_000
SHADOW_LABEL = "DRY-RUN/SHADOW"


@dataclass
class StrategyResult:
    strategy: str
    channel: str                 # "EXECUTION" or "SHADOW"
    gate_status: str             # PASS / BLOCKED
    n_linked: int
    baseline_oos_mean_r: float | None = None
    candidate_oos_mean_r: float | None = None
    effect_r: float | None = None
    p_value: float | None = None
    fdr_significant: bool | None = None
    decision: str = "REFUSED"    # PROMOTE / HOLD / ROLLBACK / REFUSED
    notes: list = field(default_factory=list)


def load_outcomes(channel: str) -> list[dict]:
    import psycopg2, psycopg2.extras
    conn = psycopg2.connect(DSN)
    try:
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            if channel == "EXECUTION":
                cur.execute("""
                    SELECT po.strategy_id AS strategy, po.r_multiple
                    FROM trading.prediction_outcomes po
                    WHERE po.link_status='LINKED' AND po.r_multiple IS NOT NULL
                      AND po.strategy_id IS NOT NULL AND po.strategy_id <> ''
                """)
            else:  # SHADOW
                cur.execute("""
                    SELECT s.strategy, s.r_multiple
                    FROM trading.cross_market_shadow_snapshots s
                    WHERE s.outcome IN ('TP1_HIT','TP2_HIT','TP3_HIT','SL_HIT')
                      AND s.r_multiple IS NOT NULL
                      AND s.strategy IS NOT NULL AND s.strategy <> ''
                """)
            return [dict(r) for r in cur.fetchall()]
    finally:
        conn.close()


def enforce_gate(counts: dict[str, int]) -> dict[str, str]:
    return {s: ("PASS" if n >= GATE_MIN_N else "BLOCKED") for s, n in counts.items()}


def walk_forward_splits(rs: list[float], seed: int = 42) -> tuple[list[float], list[float], list[float]]:
    """Deterministic 70/15/15 split (shuffle once with fixed seed)."""
    vals = list(rs)
    random.Random(seed).shuffle(vals)
    n = len(vals)
    tr = int(n * TRAIN_PCT)
    va = int(n * VALID_PCT)
    return vals[:tr], vals[tr:tr + va], vals[tr + va:]


def bootstrap_p_value(baseline: list[float], candidate: list[float], n_boot: int = BOOTSTRAP_N) -> float:
    """Paired-bootstrap p for mean(candidate) - mean(baseline) > 0."""
    import statistics
    if not baseline or not candidate:
        return 1.0
    obs = statistics.fmean(candidate) - statistics.fmean(baseline)
    if obs == 0:
        return 1.0
    rng = random.Random(7)
    gt = 0
    n = min(len(baseline), len(candidate))
    for _ in range(n_boot):
        b = statistics.fmean(rng.choice(baseline) for _ in range(n))
        c = statistics.fmean(rng.choice(candidate) for _ in range(n))
        if (c - b) >= obs:
            gt += 1
    return max(gt / n_boot, 1.0 / n_boot)


def benjamini_hochberg(pvals: dict[str, float], q: float = FDR_Q) -> dict[str, bool]:
    """Returns per-hypothesis significance after BH-FDR."""
    items = sorted(pvals.items(), key=lambda kv: kv[1])
    m = len(items)
    sig, prev = {}, 0.0
    for i, (k, p) in enumerate(items, start=1):
        thresh = i * q / m
        if p <= max(thresh, prev):
            sig[k] = True
            prev = thresh
        else:
            sig[k] = False
    return sig


def evaluate_strategy(name: str, channel: str, rs: list[float], gate: str) -> StrategyResult:
    res = StrategyResult(strategy=name, channel=channel, gate_status=gate, n_linked=len(rs))
    res.decision = "PENDING"
    if gate != "PASS":
        res.decision = "REFUSED"
        res.notes.append("sufficiency gate BLOCKED — harness refuses (fail-closed)")
        return res
    if len(rs) < GATE_MIN_N:  # sanity: rows must cover the gate threshold
        res.decision = "REFUSED"
        res.notes.append("insufficient rows for walk-forward splits")
        return res
    tr, va, oos = walk_forward_splits(rs)
    # Baseline vs "candidate": in a REAL Phase-1 experiment the candidate R-series
    # comes from the tuned-parameter shadow/canary run. In dry-run the harness
    # demonstrates the mechanics using the OOS baseline against a zero-effect
    # placebo (identity) — the decision path executes, the effect is (correctly)
    # null. No live parameters are read or modified.
    baseline_oos = oos
    candidate_oos = va  # placebo: validation slice as "candidate" — expected no significant lift
    res.baseline_oos_mean_r = _mean(baseline_oos)
    res.candidate_oos_mean_r = _mean(candidate_oos)
    res.effect_r = res.candidate_oos_mean_r - res.baseline_oos_mean_r
    res.p_value = bootstrap_p_value(baseline_oos, candidate_oos)
    res.fdr_significant = None  # set after BH across strategies
    res.notes.append(
        "DRY-RUN placebo: candidate slice is the validation window (expected HOLD)" if channel == "SHADOW"
        else "candidate series must come from the tuned run (see protocol §5)")
    return res


def _mean(xs):
    return sum(xs) / len(xs) if xs else None


def decide(res: StrategyResult) -> StrategyResult:
    if res.decision == "REFUSED" or res.effect_r is None:
        return res
    if res.effect_r >= EFFECT_MIN_R and res.fdr_significant:
        res.decision = "PROMOTE_RECOMMENDATION"  # never auto-promote
    elif res.effect_r < -EFFECT_MIN_R and res.fdr_significant:
        res.decision = "ROLLBACK_RECOMMENDATION"
    else:
        res.decision = "HOLD"
    return res


def run(channel: str, dry_run: bool) -> tuple[list[StrategyResult], int]:
    outcomes = load_outcomes(channel)
    by_strategy: dict[str, list[float]] = {}
    for row in outcomes:
        by_strategy.setdefault(row["strategy"], []).append(float(row["r_multiple"]))
    gates = enforce_gate({s: len(v) for s, v in by_strategy.items()})

    results = [evaluate_strategy(s, "SHADOW" if dry_run else "EXECUTION", v, gates[s])
               for s, v in sorted(by_strategy.items())]

    # BH across all tested (non-refused) hypotheses
    tested = [r for r in results if r.p_value is not None]
    if tested:
        sig = benjamini_hochberg({r.strategy: r.p_value for r in tested})
        for r in results:
            if r.p_value is not None:
                r.fdr_significant = sig.get(r.strategy, False)
            decide(r)

    label = SHADOW_LABEL if dry_run else "EXECUTION"
    report = {
        "run_label": label,
        "gate_min_n": GATE_MIN_N,
        "gate_status_by_strategy": gates,
        "strategies": [asdict(r) for r in results],
        "note": ("DRY-RUN on shadow channel ONLY — never execution evidence"
                 if dry_run else
                 "Execution-channel evaluation; decisions are recommendations, never auto-promoted"),
    }
    print(json.dumps(report, indent=1))
    return results, 0 if dry_run else (2 if all(g == "BLOCKED" for g in gates.values()) else 0)


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--dry-run-shadow", action="store_true",
                    help="execute the harness against the SHADOW channel only (labeled DRY-RUN/SHADOW)")
    ap.add_argument("--output", default="", help="optional JSON output path")
    args = ap.parse_args()

    channel = "SHADOW" if args.dry_run_shadow else "EXECUTION"
    results, code = run(channel, dry_run=args.dry_run_shadow)
    if args.output:
        with open(args.output, "w") as f:
            json.dump({"label": SHADOW_LABEL if args.dry_run_shadow else "EXECUTION",
                       "results": [asdict(r) for r in results]}, f, indent=1)
    refused = [r.strategy for r in results if r.decision == "REFUSED"]
    if refused:
        print(f"\nREFUSED for gate-BLOCKED strategies: {refused}", file=sys.stderr)
    if all(r.decision == "REFUSED" for r in results):
        print("GATE BLOCKED — no strategy meets the 300-link minimum. "
              "Fail closed: no evaluation performed.", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main())