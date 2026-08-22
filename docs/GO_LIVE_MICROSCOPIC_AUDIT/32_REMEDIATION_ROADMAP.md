# 32 — Remediation Roadmap

## Phase A — P0 Emergency (mandatory before any user traffic)

| # | Action | Risk | Regression testing required |
|---|---|---|---|
| A1 | Enable ufw default-deny; publish only 80/443; remove 5432/6379/1308x/3001/8091 publishes | LOW | service smoke matrix (§99); agent reconnect |
| A2 | Rotate DB, Valkey(requirepass), JWT, Grafana, FMP/TwelveData, SMTP, Telegram secrets | LOW | all services restart; auth e2e |
| A3 | Create least-privilege DB roles per service; drop superuser usage | MEDIUM | migrations rerun on staging clone; control+realtime integration |
| A4 | Agent WS auth: device token issued at activation, verified at upgrade; bind agentId | MEDIUM | windows-agent e2e vs engine; gate hydration tests |
| A5 | Wire KillSwitch endpoint + daily-loss gate into EvaluateAll order; add StopHunt/MinATR to order; remove or implement slippage | MEDIUM | golden fixture suite; gate unit tests; replay regression |
| A6 | Disable paid checkout OR integrate PSP with signed webhooks + server-side activation; quarantine fake payment/invoice/subscription rows (operator-authorized compensating entries, no deletes of audit trail) | HIGH | billing e2e in PSP sandbox; reconciliation report |
| A7 | Strip fabricated calibration metadata; expose UNCALIBRATED until real calibration exists | LOW | scoring snapshot diff; UI label tests |

## Phase B — P1 blockers

B7-B14 items: entitlement enforcement at delivery + quota ledger writers; commission writers w/ idempotency using SEEDED rate table (align spec tests); payout schema fix + balance checks; outbox dispatcher + expiry sweeper + EA ack ingestion; candle aggregator fix + backfill of 553 rows + quality gate; frontend honest-state fixes; audit durability.
Risk: MEDIUM overall; each requires targeted integration tests + one replay/paper validation cycle.

## Phase C — P2 hardening

Unreachable regime states cleanup; InputHash/DecisionHash population; config-version snapshots on signals; CSP/security headers; IP/UA in auth audits; admin entitlement editor; nginx tree consolidation; docs sync (systemd→docker); restore drill; load test staging (p50/p95/p99 capture).

## Phase D — P3 enhancements

Fullscreen/4K command center, price chart, prefers-reduced-motion, referral code randomization, dead-code removal (rl/adaptation/hedging/sentiment/breakout/oco), stale openapi regeneration.

No human-hour estimates provided (not requested).
