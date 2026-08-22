# 33 — GO-LIVE Certification Checklist

## Market & Trading
- [x] Real XAUUSD agent feed (Fri session) — 7.1M ticks, healthy engine
- [ ] Feed freshness monitoring — no explicit MARKET_CLOSED/stale labeling end-to-end
- [~] Indicators independently verified — parity suites pass; corrupted-bar window invalidates period Aug18-21 until fixed/backfilled
- [x] No look-ahead bias in features/replay (closed-bar only; SYNTHETIC_REPLAY firewalled)
- [x] Regime logic verified (5 reachable states; hysteresis confirmed)
- [x] All 4 strategies produce BUY/SELL/NO-TRADE (DB-proven)
- [~] Score independently verified (deterministic; golden e2e fixture missing)
- [ ] **Probability meaning — FAILS (fabricated VALIDATED metadata)**
- [x] Entry/SL/TP geometry valid on samples
- [ ] **Risk sizing/execution — UNVERIFIED (needs live broker qualification)**
- [ ] **Daily-loss protection — NOT IMPLEMENTED in live plane**
- [~] Duplicate-signal protection — works when Valkey healthy/private; fail-open + public cache today
- [ ] Trade idempotency e2e — EXTERNAL BLOCKER (terminal)
- [~] MT reconnect — 60s permit semantics OK; spoofable channel

## Data
- [x] Timescale schema + hypertables/caggs verified; migrations 40/40 COMPLETED
- [ ] Persistence durability of delivery chain — deliveries/acks = 0 rows; outbox stuck
- [x] Valkey not authoritative for finance
- [ ] **Backups stale / no schedule / restore drill missing**

## Security
- [x] NestJS authn/RBAC guards live (401 reproduced)
- [ ] **Network exposure FAILS (DB superuser public, Valkey public, agent WS open)**
- [ ] Secrets rotation pending
- [x] TLS/HTTPS via nginx valid certs

## Commercial Platform
- [ ] **Subscriptions/billing — FAILING (stub; fake webhook activated plan)**
- [ ] Quotas — unenforced/unwired
- [ ] Referral commissions/payouts — broken/dead
- [x] Plan catalog + entitlement policy validation at create-time

## Frontend
- [~] Genuine data mostly server-rendered; hardcoded price placeholders + fabricated performance metrics violate §74
- [ ] WS entitlement path dead (no UI signal push)

## Infrastructure
- [x] Services containerized, healthy, restart policies
- [ ] Firewall/host hardening FAILING
- [~] Prometheus metrics exist; alerting/DR unproven

Legend: [x]=verified · [~]=partial · [ ]=failed/unverified
