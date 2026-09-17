# RESUME-TRIGGER SESSION REPORT — 2026-09-17 (UTC)

Operator fired the resume trigger ("fix all pending bugs, issue, task and
resume trigger fired. prepare final production ready to go live"). Engineering
exited standby and executed a full multi-plane sweep. Commits: `bb2eddf`,
`92abf18`, plus this session's doc/dashboard batch.

## Bugs found and fixed

| # | Bug | Root cause | Fix | Verified |
|---|---|---|---|---|
| 1 | Fixture test rows polluted the production execution-channel tables and inflated the Phase-1 gate (STANDARD_SCALPING N=47 incl. 9 fake; XAUUSD N=1 fully synthetic) | Go DB round-trip tests ran with `PERSISTENCE_TEST_URL` pointed at prod; harness had no guard | Guard refuses any URL whose database name lacks "test"; all 32 fixture rows purged from `prediction_outcomes` (10), `predictions` (11), `signal_feature_snapshots` (1, plus re-check 22); scratch DB `predictatrade_test` created (schema-only) for future round-trips | prod URL → SKIP, test URL → PASS; zero fixture rows left |
| 2 | Shadow outcome resolver dead 19 days (2026-08-29 → 09-17), ~137k snapshots stuck UNRESOLVED | Queue read `ORDER BY timestamp ASC LIMIT 100`; legacy zero-expiry rows permanently occupied the head | Newest-first + 14-day window; errors now logged (`SetLogger`) instead of silently swallowed | build + unit tests; live from engine 16:44 UTC |
| 3 | Driver snapshots table silently empty since 08-29 (RawValue coverage hole for calibration) | `SaveDriverSnapshot` had no call site | `Engine.SetDriverSink` hook persists every accepted driver snapshot asynchronously | rows flowing (4 fresh at 16:45 UTC) |
| 4 | Every online backtest on live data failed `DATA_QUALITY_FAILED` | `market.candles` holds multiple sources per (time, symbol, timeframe) bucket (MT4_MASTER/MT5_MASTER/AGGREGATOR) → duplicate, non-monotonic timestamps → ORDERING gate fail-closed | `DISTINCT ON` canonical dedupe in `DataLoader.from_database` (AGGREGATOR > MT5_MASTER > MT4_MASTER > others deterministic) | research suite 153 passed / 1 skipped / 0 failed |
| 5 | Control-plane test suite unrunnable on host (jest missing, bcrypt native build blocked, EmailService ESM import crash, admin spec missing providers) | node_modules absent; npm install-scripts allowlist; type used in value position; spec not updated for AdminService's newer deps | npm ci + bcrypt allowScripts pin; type-only imports; spec gains JwtService/EMAIL_SERVICE mocks | 240/240 suites/tests green, tsc clean |
| 6 | `operator_status.sh` UNLINKED ratio counted fixture rows (window = last 100 rows, all-time); time-to-300 divided by a 1-hour count but labeled weeks | Window bugs | UNLINKED → last 24h; time-to-300 → correct weeks math, `n/a` at zero rate | script runs; UNLINKED now 50.6% (honest legacy-backfill figure) |
| 7 | Compose env hole: no root `.env` → `${DATABASE_URL}`/`${JWT_SECRET}` interpolated blank; failed `up` recreated containers and crash-looped realtime (SASL fail), control, postgres | Compose interpolation reads shell/`.env`, not `env_file` | Root `.env` recreated (gitignored, 0600) from live container values; `realtime.env` synced; services force-recreated | all 16 containers healthy; engine ok/db ok/cache ok |

Also verified, no action needed:

- **TwelveData burn (7227 vs 800/day)** — pre-15:37 restart, the old un-batched
  image burned the API's day before the budgeted build deployed. The new
  engine's 640/day fail-fast budget works (verified in logs). Resets UTC
  midnight; external consumers may still eat into the same key's real quota.
- **`pat-pgdata` volume warning** — deliberate compose-file choice
  (`external:true` would silently skip creation on a fresh machine → data
  loss). Documented in docker-compose.yml; left as designed.
- **pat-realtime/pat-valkey 15:37 restarts** — deploy of the budgeted build,
  RestartCount 0, healthy since.

## Full verification sweep (all live)

- Go: 40/40 packages `go test -race` green; vet clean
- Control: 240/240; frontend: 84/84; research: 153/154 (1 skip)
- All 16 containers healthy; watchdog green; 0 Prometheus alerts firing
- Engine: status ok, db ok, cache ok, emergency halt off, schema guard verified
- Public edge (CF → Plesk → origin): web 200, api 200, live 200, status 200,
  platform 307 (auth redirect, correct)
- Signals flowing (85 in 15m), backups syncing to S3 (WAL + DB, minute cadence)
- Disk 33% (95G/301G), memory 5/15G

## What still gates go-live (not bugs — operator actions + data collection)

1. Client EAs attach on Windows terminals (patch 0001 recompile per
   `docs/OPERATOR_WINDOWS_RUNBOOK.md`) → device ONLINE → TRADE_RESULT flow.
2. Phase-1 calibration gate: ≥300 LINKED exec outcomes per strategy
   (`scripts/sample_sufficiency_gate.sql`) — currently max 38 (post-purge).
3. DXY/macro pillars: recover after UTC-midnight credit reset.