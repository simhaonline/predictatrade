# RESUME-TRIGGER SESSION REPORT — DAY 2 · 2026-09-18 (UTC)

Continuation of `RESUME_TRIGGER_SESSION_REPORT_2026-09-17.md`. This session
covered the first real client-EA attach (MT5 + MT4), the first live trades,
full forensics on the resulting loss streak, and five production incidents
found and fixed while chasing it. Commits `0ca68bf` → `6a27e4e`, `9d916a9`,
`ec7c583`, `e9089b6`, `e891efc`, `5b9aee3`, `ca9820b`, `6a27e4e` batch.

## Timeline of the loss streak (forensically reconstructed)

| Window (UTC) | Event | P&L impact |
|---|---|---|
| 10:20–12:33 | First EA auto-trades. 16 of 29 closed in 0–3s by the EA's own slippage guard, each losing the spread | −$55 (mechanical) |
| 12:33 | Daily-loss halt fired at −6%: emergency close-all | halted correctly |
| 12:40–14:17 | Loss deepened past hard-halt band (−12%) → engine vetoed every candidate (`daily:severe`) | stopped |
| 14:17 | Disk-full discovered (240GB unarchived WAL) → valkey MISCONF → ingest panic | incident |
| 14:25–14:31 | 238GB pre-checkpoint WAL deleted (past-checkpoint, crash-safe), disk 100%→16% | recovered |
| 14:32 | Fresh 2.4GB physical base backup taken; archiving verified flowing to S3 | recovered |
| 16:xx | Daily-loss halt resets at 00:00 UTC (next broker day) | — |

## Root causes found & fixed

### 1. EA slippage guard at FX-scale killed every trade (EC7C583, v1.34) — THE headline
`MaxSlippagePoints=3` + per-strategy 5/10/20/30 are tight-FX-pair values.
**XAUUSD spread alone is 20–40 points.** Every fill "slipped" past the
threshold by definition → the EA instantly closed its own fresh position at
the spread → a guaranteed −0.30..−0.50 loss per trade, 7 of the first 10
trades, in seconds. Direction was often correct (market rose 4379→4387 while
the account bled). Fix: base 3→60, per-strategy → 60/60/80/100.
`RejectOnHighSlippage` stays true — genuine spikes still rejected.
**Evidence**: `time_in_trade_seconds` 0–3 on every "manual" close, each loss
≈ spread. The 3 trades that held (18s/568s/1271s) played out correctly.

### 2. ATEN bypassed the operator's armed list (e9089b6)
ATEN: **0 wins / 261 SL-hits** in shadow, NOT in `EDGE_ARMED_STRATEGIES` —
yet 140 EXECUTABLE ATEN signals were delivered and traded. Cause: un-armed +
un-proven strategies evaluated `GateDegraded (edge_unproven)`, and the
candidate-promotion path treats soft-degraded as non-blocking for
high-conviction reads. Fix: `EdgeValidationGate` now **hard-vetoes** any
strategy that is neither armed nor proven. Arming is the only bootstrap path;
the proven-negative veto still overrides arming.

### 3. Backup pipeline had NEVER worked (disk-full cascade)
- `archive_command` target dir was **root-owned** → postgres (uid 1000) got
  `Permission denied` since the 2026-09-16 path change → WAL never archived.
- Disk hit **100%** from 240GB of unarchived WAL → valkey `MISCONF`
  (RDB save failed) → `SetSnapshot` nil-client panic inside the ingest
  handler → every ingest request 500'd until restart.
- `pat-backup-sync` had been mirroring **empty directories** to S3 for days
  (wal_archive host dir empty; no `pg_basebackup` cron existed).
- Fixed: deleted 15,243 pre-checkpoint WAL segments (238GB, crash-safe —
  past-checkpoint WAL is never needed for crash recovery), disk 100%→20%,
  `infra/wal_archive` chmod 1000:1000 (archiving verified live: 64+ files),
  fresh 2.4GB `pg_basebackup` taken, **6-hourly backup cron installed**
  (never existed), valkey nil-client guards (`e891efc`), wal_archive
  gitignored.

### 4. License/activation + EA UX (from day-1 session, same thread)
- JWT_SECRET parity restored (engine rejected all device tokens with
  signature-mismatch storms after a placeholder value leaked into root .env).
- Stale device slots freed; operator raised plan to PRO (4 devices).
- v1.33: `AutoExecute=true` default (real-world algo semantics) +
  `PAT_TerminalTradeReady` self-diagnostics (terminal Algo-Trading state
  surfaced at attach and at execution, once/minute, never silent).
- v1.32: transport-layer hardening — `PAT_HTTPErrorText` maps `-1`/`1003`-class
  failures to actionable text (allowlist/DNS/TLS), 3-attempt retries with
  re-sign on signed polls.

### 5. Fleet-scale rate limiting removed (operator decision, d327063)
All per-IP nginx limits removed: with CF grey-cloud → Plesk → origin, every
subscriber shares one edge remote_addr, so per-IP zones can only 429 the
whole fleet. Abuse control stays at app layers (HMAC device auth, NestJS
Throttler on interactive routes, JWT on ingest).

## Verified live after fixes

- Engine: ok / db ok / cache ok / schema verified; 16/16 containers healthy
- Daily-loss halt actively vetoing (`daily:severe`) — zero EXECUTABLE enqueued
  since 12:40 — resets at 00:00 UTC (daily period = UTC calendar day)
- ATEN no longer EXECUTABLE; STANDARD_SCALPING vetoed by proven-negative edge
  (PF 0.553, n=113); shadow channel: STANDARD_SWING +0.55R, ULTRA +0.16R
- Backups: base 2.4GB fresh + WAL archiving + S3 sync flowing
- Disk 20% (231G free); EA v1.34 both files, banners updated, pushed

## Operational takeaways (binding)

1. **EA recompile discipline**: terminal `.ex5`/`.ex4` builds must match repo
   `PAT_EA_VERSION` before judging any strategy behavior. Stale binaries
   produced the entire "signals losing" incident surface.
2. **Slippage thresholds are symbol-specific**: gold ≠ EUR/USD. Any future
   strategy on a new symbol must re-derive `MaxSlippagePoints` from that
   symbol's typical spread.
3. **Arming is the only path to EXECUTABLE.** Adding a strategy to
   `EDGE_ARMED_STRATEGIES` is an explicit operator action with edge evidence.
4. **Backup pipeline is now genuinely working** — verify monthly:
   `ls /var/backups/predictatrade/base/` (fresh tar) +
   `docker exec pat-postgres psql -U pat_admin -d predictatrade -c "SELECT
   archived_count, failed_count FROM pg_stat_archiver;"` (failed_count must
   stay 0) + `aws s3 ls s3://predictatrade-backups/predictatrade/wal/`.
5. **Disk watchdog**: the 100%-full disk caused the valkey/ingest cascade.
   DiskSpaceLow-style alerting should be added when node_exporter is deployed.

## What still gates go-live (unchanged)

- Phase-1 calibration gate: ≥300 LINKED exec outcomes/strategy (max 38;
  collection resumes with the halt reset + v1.34 compiled).
- DXY/macro pillars recover at UTC-midnight credit resets.
- MT4 ADS terminal: $9.99 equity cannot size gold (correct fail-closed) —
  use a funded-size demo for that terminal.