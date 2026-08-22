# 28 — Performance / Reliability Audit

## Measured during audit (weekend load ≈ idle)

| Metric | Evidence | Result |
|---|---|---|
| Control REST latency | /health, /plans responses | sub-second (no p50/p95 population at this traffic — NOT FABRICATED) |
| Go API latency | /market/state ~large JSON | fast; MANIFEST claims <3ms — UNVERIFIED (no benchmark rerun) |
| DB size/perf | 7.1M ticks, 418k M5 candles; hypertables + caggs present | healthy counts; no query-latency sample taken under load |
| WS delivery latency | not measurable (UI signal push dead — 25) | N/A |
| Tick→signal pipeline latency | signals created within same second as bar close (created_at vs bar time on samples) | consistent with per-close evaluation |

## Load/concurrency (static + config)

- Go: bounded worker pools (FEATURE_WORKERS=4), drop-on-full backpressure, 100-agent cap.
- NestJS: single pg Pool(max=20) shared hot-path+admin, no statement timeout → contention risk under load; throttler only on auth/guest routes.
- Quota race: moot today (quota unwired); subscription-create race real (20).
- WS scale-out impossible without Valkey pub/sub (25).

## Reliability posture

- Restart durability: Postgres/Valkey durable; in-memory-only state that dies with process: reconciler map, gate cached states (re-hydrated ≤60s), delivery manager contents (empty anyway).
- Chaos scenarios NOT executed against live prod (per §1); code-level failure paths reviewed instead (13/16/17).

**Verdict: UNVERIFIED for production load claims.** No fabricated numbers reported; insufficient samples honestly labeled. Load testing must run in staging before GO.
