# 00 — Executive GO-LIVE Verdict

Audit date: 2026-08-22 (UTC) · Auditor: independent forensic session · HEAD: `e6fc7dfb0e81ea74ff4d73ff79fd9a09e232f1bd` (main, clean)

```
==================================================
PREDICT-A-TRADE XAUUSD
FINAL PRODUCTION READINESS VERDICT

VERDICT: NO-GO

P0 BLOCKERS: 6
P1 BLOCKERS: 12
P2 FINDINGS: 14
P3 FINDINGS: 8 (see 31/32)

LIVE MARKET DATA:    VERIFIED (agent feed live Fri; correct weekend gap; news sync live)
DATABASE INTEGRITY:  NOT VERIFIED (superuser publicly exposed; candle corruption found)
SIGNAL MATHEMATICS:  PARTIAL (geometry OK on samples; calibration metadata fabricated)
RISK MANAGEMENT:     NOT VERIFIED (no emergency stop, no daily-loss gate in live plane,
                     two hard gates unreachable, slippage gate always-pass)
MT4/MT5:             PARTIAL (agent connects, but WS is unauthenticated and spoofable)
AI/ML:               SIMULATED / INERT (ML enabled-in-config, absent at runtime, results discarded;
                     sentiment endpoint unreachable → always neutral; PTB honest SHADOW zero-weight)
SUBSCRIPTIONS:       NOT VERIFIED (billing stub; only ACTIVE sub created via unverified fake webhook)
REFERRALS:           NOT VERIFIED (commission engine unreachable, ledger empty, payouts broken SQL)
SECURITY:            NOT VERIFIED (public DB superuser + unauth Valkey + unauth agent WS verified)
BACKUP/DR:           NOT VERIFIED (last backup Aug 18, stale; no scheduled dump cron)

SAFE FOR PRODUCTION USERS:        NO
SAFE FOR LIVE BROKER EXECUTION:   NO
==================================================
```

## Verdict rationale (evidence-first)

| # | Finding | Severity | Proof (see referenced doc) |
|---|---------|----------|---------------------------|
| F-001 | PostgreSQL reachable from public internet as **superuser** `pat_admin` with password committed in `docker-compose.yml`; `ufw` inactive | P0 | Live test: `psql -h 152.53.67.111 -U pat_admin …` → `REMOTE_DB_ACCESS_OK current_user=pat_admin`; `pg_roles`: `rolsuper=true` (18_AUTH) |
| F-002 | Valkey reachable from public internet, **no auth** (`+PONG` from WAN IP); cache can be flushed/poisoned (cooldown/duplicate keys = duplicate-signal risk) | P0 | Live `nc` PING → `+PONG` (18_AUTH) |
| F-003 | Agent WebSocket `/ws/v1/agent` requires **zero credentials** (`CheckOrigin→true`, self-declared agentId). Upgrade proven: HTTP 101 + CONNECTED ack with fabricated id `AUDIT-PROBE`. Port published `0.0.0.0:13081`. Enables signal exfiltration + forged `account_info` that flips Exposure/Margin/ExecutionPermit gates to PASS | P0 | Live upgrade transcript (06_MARKET_DATA, 18_AUTH) |
| F-004 | No emergency stop / kill switch anywhere in the live plane; daily-loss protection module not wired into engine gates; EA CAPITAL_PROTECTION messages are only logged | P0 | `KillSwitch` type has zero usages; `capital_protection.go` referenced only by `cmd/audit` (13_RISK) |
| F-005 | Billing is a stub: `handleWebhook()` logs and returns `{received:true}`; **unauthenticated** route; no PSP SDK in control plane. The single ACTIVE ELITE subscription was activated by payment event `evt_local_invoice_1787381953` with **`signature_verified=false`** — payment verification bypass demonstrated in production data | P0 | DB row quoted above; `billing.service.ts:19-22` (22_BILLING) |
| F-006 | Calibrated probability carries fabricated metadata: untrained hardcoded sigmoid stamped `Status="VALIDATED" SampleSize=100 Brier=0.21` by `SeedDefaultModels`; NO-TRADE rows persist prob≈0.45 default | P0/P1 | `consumer.go:83-107`; signals table sample (11_SCORING) |

P1 highlights: commission engine unreachable + payouts broken SQL (42703); entitlements not enforced on REST/WS delivery (signals API served anonymously — reproduced); licensing IDORs (heartbeat/revoke without ownership check); guest preview delivers paid data behind CSS blur; frontend fabricates Hit-Rate/Avg-R metrics and renders hardcoded 2500 prices as Bid/Ask; StopHuntFilter+MinAbsoluteATR gates registered but never evaluated; SlippageGate always-pass; 553 corrupted candles (`open=0`,`low=0`, quality=`COMPLETE`) fed to indicators Aug 18–21; outbox 2697 rows stuck PENDING; signal expiry sweeper missing; ML/sentiment inert with misleading config.

## What genuinely works (verified, not assumed)

- Deterministic spine MT5-agent → ingestion → closed-bar features → 4 distinct strategies → scoring → short-circuit gate framework → persisted signals with provenance/versioning (code-traced end-to-end).
- Signal geometry on sampled SELL candidates correct (SL>Entry>TP1 for SELL); OHLC invariant violations = 0 across 418k M5 candles.
- Migration discipline: 40/40 recorded COMPLETED incl. reconciliation/catalog migrations; TimescaleDB hypertables + continuous aggregates present.
- AuthN on NestJS: admin endpoints return 401 anonymous (reproduced); refresh rotation w/ reuse detection; device fingerprint binding; append-only audit schema.
- News gate live (FMP sync every 5 min, fail-closed policy `BLOCK_TRADING` when provider fails).
- Weekend market close handled honestly (tick gap Fri 20:59 UTC → Sun open; NOT staleness).
- Test suites pass: research pytest 127/127; go vet clean; targeted go packages pass.

## Minimum remediation before re-review (Phase A, all P0)

1. Firewall the host (ufw default-deny; expose only 80/443), remove 5432/6379/1308x port publishes from compose, enforce strong per-service auth.
2. Rotate ALL secrets (DB passwords, JWT secret, FMP/TwelveData keys, SMTP, Telegram token) — they are committed/exposed; create non-superuser app roles.
3. Authenticate the agent WebSocket (device token from activation flow; bind agentId to device).
4. Implement emergency-stop + daily-loss gates wired into the live evaluation order; wire StopHuntFilter/MinATR into the order slice; make SlippageGate real or remove it from advertised gates.
5. Either integrate a real PSP with signature-verified webhooks + server-side activation, or disable paid checkout entirely until done; purge/quarantine the fake payment event and its subscription.
6. Remove fabricated calibration metadata; mark probability UNCALIBRATED until real calibration exists.

Then Phase B (P1) per `32_REMEDIATION_ROADMAP.md`. Re-certification required after Phase A+B.
