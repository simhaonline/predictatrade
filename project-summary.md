# Predict-A-Trade XAUUSD — Production Readiness Audit Summary

Audit date: 2026-08-22 (UTC) · HEAD audited: `e6fc7dfb` · Final report commit: `3f72698`
Full evidence: `docs/GO_LIVE_MICROSCOPIC_AUDIT/00…38_*.md` (39 documents)

```
==================================================
PREDICT-A-TRADE XAUUSD
FINAL PRODUCTION READINESS VERDICT

VERDICT: NO-GO

P0 BLOCKERS: 6
P1 BLOCKERS: 12
P2 FINDINGS: 14
P3 FINDINGS: 8

LIVE MARKET DATA:    VERIFIED (agent feed live Fri; correct weekend gap; news sync live)
DATABASE INTEGRITY:  NOT VERIFIED (public superuser exposure; candle corruption found)
SIGNAL MATHEMATICS:  PARTIAL (geometry OK on samples; calibration metadata fabricated)
RISK MANAGEMENT:     NOT VERIFIED (no emergency stop, no daily-loss gate in live plane,
                     two hard gates unreachable, slippage gate always-pass)
MT4/MT5:             PARTIAL (agent connects, but WS is unauthenticated and spoofable)
AI/ML:               SIMULATED / INERT (models absent at runtime; results discarded;
                     sentiment unreachable → always neutral; PTB honest SHADOW zero-weight)
SUBSCRIPTIONS:       NOT VERIFIED (billing stub; only ACTIVE sub created via unverified fake webhook)
REFERRALS:           NOT VERIFIED (commission engine unreachable, ledger empty, payouts broken SQL)
SECURITY:            NOT VERIFIED (public DB superuser + unauth Valkey + unauth agent WS verified)
BACKUP/DR:           NOT VERIFIED (last backup Aug 18, stale; no scheduled dump cron)

SAFE FOR PRODUCTION USERS:        NO
SAFE FOR LIVE BROKER EXECUTION:   NO
==================================================
```

## P0 Blockers (all reproduced with live evidence)

| # | Finding | Proof |
|---|---------|-------|
| F-001 | PostgreSQL reachable from public internet as **superuser** `pat_admin` with password committed in `docker-compose.yml`; `ufw` inactive | `psql -h 152.53.67.111 -U pat_admin …` → login OK, `rolsuper=true` |
| F-002 | Valkey publicly reachable with **no auth** (`+PONG`) — dedup/cooldown keys writable ⇒ duplicate-signal protection bypassable | WAN `nc PING → +PONG`; fail-open dedup in `cooldown.go:108-112` |
| F-003 | Agent WS `/ws/v1/agent` requires zero credentials; fabricated agentId accepted (`101 + CONNECTED`); forged `account_info` flips Exposure/Margin/ExecutionPermit gates for 30–60s; port published `0.0.0.0:13081` | Live upgrade transcript with `AUDIT-PROBE` id |
| F-004 | No emergency stop/kill switch in the live plane; daily-loss module unwired; EA CAPITAL_PROTECTION messages only logged | `KillSwitch` zero usages; `capital_protection.go` referenced only by `cmd/audit` |
| F-005 | Billing is a stub (unauthenticated webhook no-op, no PSP SDK); the single ACTIVE ELITE subscription was activated by payment event with **`signature_verified=false`** | DB rows quoted in audit doc 22; `billing.service.ts:19-22` |
| F-006 | Calibrated probability = untrained hardcoded sigmoid stamped `VALIDATED / SampleSize=100 / Brier=0.21`; every NO-TRADE row persists prob≈0.450166 | `calibration/consumer.go:83-107`; signals table samples |

## Key P1 findings

- Two registered hard gates never evaluated (StopHuntFilter, MinAbsoluteATR missing from order slice); SlippageGate always-pass stub.
- Entitlements unenforced on delivery: anonymous `/api/v1/signals` returns full payloads (reproduced); quota ledger has no writer; WS entitlement filter never populated ⇒ UI gets no WS signals at all.
- Licensing IDORs (heartbeat/revoke without ownership); signed-request verification implemented but never invoked.
- Guest preview delivers full paid data behind CSS blur; frontend fabricates Hit-Rate/Accuracy/Avg-R metrics and renders hardcoded `2500/.00/.50` Bid/Ask/Spread.
- 553 corrupted candles (`open=0`,`low=0`, quality=`COMPLETE`) fed to indicators Aug 18–21.
- Transactional outbox stuck: 2,697 rows PENDING forever (dispatcher dead); signal expiry sweeper missing; EA acks logged-and-dropped; reconciliation memory-only.
- Commission engine unreachable (ledger 0 rows; spec tests validate different rates than seeded config); payout INSERT crashes on nonexistent column (42703).
- ML_ENABLED=true but models absent in container → inert; results discarded anyway (`_ = mlDir`); Ollama host unreachable from container; NTFY port mismatch (8090 vs 8091).
- Plaintext secrets on disk (FMP/TwelveData/SMTP/Telegram keys), JWT-secret-derived AES key with reachable dev fallback.

## What genuinely works (verified)

- Deterministic spine MT5-agent → ingestion → closed-bar features → 4 distinct strategies → scoring → short-circuit gates → persisted signals with provenance/versioning.
- Signal geometry correct on samples (SELL: SL > Entry > TP1); OHLC invariant violations = 0 across 418k M5 candles.
- Migrations 40/40 COMPLETED incl. reconciliation entries; TimescaleDB hypertables + continuous aggregates present.
- NestJS authn/RBAC: admin endpoints return 401 anonymous (reproduced); refresh rotation w/ reuse detection; append-only audit schema.
- News gate LIVE (FMP sync every 5 min, fail-closed `BLOCK_TRADING` policy).
- Correct weekend-close behavior (tick gap Fri 20:59 UTC → Sun open is market reality, not staleness).
- Tests pass: research pytest 127/127; go vet clean; targeted Go suites green.

## Minimum remediation before re-review (Phase A)

1. Firewall host (default-deny; publish only 80/443); remove DB/cache/engine port publishes; add Valkey auth.
2. Rotate ALL exposed secrets; replace superuser with least-privilege per-service DB roles.
3. Authenticate the agent WebSocket (device token bound to activation; bind agentId to device).
4. Wire emergency stop + daily-loss gate into live gate order; add StopHunt/MinATR to order slice; fix or remove SlippageGate.
5. Integrate a real PSP (signature-verified webhooks, server-side activation) or disable paid checkout; operator must authorize quarantine of the fake payment/invoice/subscription rows via compensating entries.
6. Remove fabricated calibration metadata; label probability UNCALIBRATED until real calibration exists.

Phase B (P1): entitlement enforcement at delivery + quota writers; commission writers w/ idempotency using seeded rate table; payout schema alignment; outbox dispatcher + expiry sweeper + ack ingestion; candle aggregator fix + backfill of 553 bars; frontend honest-state fixes; audit durability.

Then re-certification requires: Phase A+B complete, restore drill, broker execution qualification on a live terminal (EXTERNAL BLOCKER), and one paper/shadow week with delivery-ack reconciliation.

## Operator decision points

- Requires explicit authorization: purging/quarantining fake financial records; enabling live automated trading (currently absent by design).
- Requires credentials/hardware: live MT4/MT5 terminal qualification; PSP sandbox account; SMTP/Telegram send tests.
