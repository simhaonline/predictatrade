# Predict-A-Trade — Core Reset Plan (v1 directional)

**Decision:** Do NOT rebuild from scratch. Do a *core reset* on the signal engine.
The existing planes (Go realtime / Python research / NestJS control / Next.js /
Windows agent) are sound. The gap is **strategy proof + execution fidelity +
honest packaging** — and that gap survives any rebuild anyway.

**Guiding rule:** No strategy is offered to a client until it is (a) proven on
real out-of-sample XAUUSD *with costs*, and (b) traded live as a faithful,
risk-gated clone of that exact strategy (paper/shadow first).

---

## What stays vs what resets
- **STAYS:** data pipeline, Windows agent, MT4/MT5 adapters, NestJS IAM/billing,
  Next.js shell, observability, Docker-first deploy.
- **RESETS:** signal→execution path discipline. One strategy at a time. Hard
  gates enforced. No silent drift (R:R, duplicates, null SL/TP).

---

## Phased plan

### Phase 0 — Stop the bleeding (1 day)
- Disable all LIVE automated execution. Run only PAPER/SHADOW.
- Freeze strategy changes. No EA/agent debugging until Phase 2.

### Phase 1 — Prove ONE strategy (ULTRA_SCALPING) on real data
- Real, cost-aware, walk-forward/OOS backtest on 2025 KAGGLE XAUUSD (PF 1.18 today).
- Publish honest stats: PF, win-rate, max drawdown, avg R:R, sample size, per-cost.
- Gate: PF > 1.0 with stable OOS + sufficient sample. If it fails, pick the next.

### Phase 2 — Faithful live execution (paper/shadow first)
- Live path = exact clone of the validated strategy + hard risk gates
  (SL>0 enforced, R:R floor, no duplicate positions, total-cost limit).
- Verify execution matches backtest on shadow for N signals before any real trade.
- Gate: execution P&L distribution matches validated profile within tolerance.

### Phase 3 — Package model on validated stats
- Packages = named strategy profiles with *published* honest stats + risk limits.
- STANDARD_SCALPING (PF 0.84) is NOT sold as "accurate/profitable" — fix or retire.
- Per-signal calibrated probability, never raw score.

### Phase 4 — Add other strategies only after the same bar
- STANDARD_SWING, TREND_SWING each repeat Phase 1→2 before being packaged.

---

## Definition of done (per phase)
- Phase 1: reproducible real-data validation report, signed off.
- Phase 2: shadow execution parity report vs backtest.
- Phase 3: package specs with honest, published performance + risk.
- No phase advances until its gate passes.

## Immediate next action
Start Phase 0 (freeze live execution) + re-run the ULTRA_SCALPING validation to
lock the v1 offering. Agent-install troubleshooting is parked.

---

## Status — 2026-08-28 (new standalone `pat-engine/`)

Decision shifted from "reset in place" to "build a clean standalone engine, reference
only, no existing service/DB, current repo untouched" (user direction). Delivered and
tested:

- **Engine (zero-dep Go):** 4 strategy products (`ULTRA_SCALPING`, `STANDARD_SCALPING`,
  `STANDARD_SWING`, `TREND_SWING`), single-source SL/TP/RR config, shared confluence
  scoring + trade geometry, broker-policy gating, hard risk gates (R:R floor + broker
  stop/freeze).
- **Broker scalping policy is first-class:** `AllowsScalping=false` excludes both
  scalpers (`BROKER_SCALPING_NOT_ALLOWED`); only swing/trend remain eligible. Verified
  by tests + backtest harness.
- **Backtest harness** runs the *exact same* strategy code/config as live; synthetic
  demo yields PF (ULTRA ~3.1, SWING ~7.7 on trend-friendly synthetic data) and a real
  `BARS_CSV` loader is wired for KAGGLE/MT5 data.
- **Live gateway (stdlib HTTP):** ingests bars (`POST /bar`), runs the pipeline, writes
  `SIGNAL|<json>` to `PAT_signals.txt` in the exact format the existing MQL EA parses.
- **End-to-end proven:** reference agent streams bars → gateway → signal file →
  EA-file simulator "executes". `go test ./...` green (executability, R:R floor,
  broker exclusion, high-spread block).
- **MQL reference EAs** (`mql/`) read `PAT_signals.txt` unchanged.

Deferred (per plan, backend+MQL first): frontend/Command Center, control-plane
licensing WS, rebuilt Windows Agent binary (reuse `windows-agent`, point at `/bar`).

Next: plug real 2025 KAGGLE/MT5 bars into `cmd/backtest` to lock honest v1 stats,
then wire the live agent + EA on a no-scalping-broker-eligible package (swing/trend).
