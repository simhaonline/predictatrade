# 31 — Production Blockers (prioritized register)

| ID | Sev | Area | Problem | Evidence | Required before go-live | Type |
|---|---|---|---|---|---|---|
| B-01 | P0 | Infra | Public DB superuser / no firewall | WAN psql OK; ufw inactive | Firewall + least-privilege + rotate | Infrastructure |
| B-02 | P0 | Infra | Public unauth Valkey | +PONG WAN | Firewall + auth | Infrastructure |
| B-03 | P0 | Security | Unauth agent WS | 101 probe reproduced | Device auth on agent channel | Code |
| B-04 | P0 | Risk | No emergency stop; daily-loss unwired | KillSwitch zero usages; capital_protection only in cmd/audit | Wire both into gate order + ops endpoint | Code |
| B-05 | P0 | Billing | Payment verification bypass; billing stub; fake Stripe row active | signature_verified=false event; ACTIVE ELITE sub | PSP integration or disable checkout; purge fake financial rows (operator decision) | Code+Business |
| B-06 | P0 | Data/AI | Fabricated calibration metadata | consumer.go seeder stamps VALIDATED | Remove metadata or implement real calibration | Code |
| B-07 | P1 | Gates | StopHunt/MinATR unreachable; Slippage always-pass | order slice omits; state never hydrated | Fix order slice; real slippage source | Code |
| B-08 | P1 | Entitlement | Paid signals served anonymously; quotas nonexistent | anonymous curl reproduced; ledger empty | Enforce plan gates at delivery; wire quota ledger | Code |
| B-09 | P1 | Finance | Commissions never computed; payouts crash (42703) | ledger 0 rows; column mismatch | Implement writers w/ idempotency; fix schema alignment | Code |
| B-10 | P1 | Delivery | Outbox stuck (2697); acks/expiry/reconciliation dead | DB counts; dead methods | Dispatcher+sweeper+ack ingestion | Code |
| B-11 | P1 | Data | Corrupted candles open/low=0 labeled COMPLETE | 553 rows Aug18-21 | Fix aggregator; backfill; quality-gate | Code |
| B-12 | P1 | Security | IDORs + secret handling (JWT-derived KEK, plaintext env) | V-05/07/08 | Fix routes; key separation; vault | Code+Infra |
| B-13 | P1 | Frontend | Fabricated metrics/hardcoded prices/guest blur leak | V-09, 24-1..3 | Honest states + server redaction | Code |
| B-14 | P2 | Ops | Backups stale/no cron; DR undefined | last dump Aug18 | Schedule + restore drill | Infrastructure |

Types: Code = internal repair; Infrastructure = deployment/config; Credential = rotation needed (B-01/02/12 overlap); Broker = live-terminal validation still pending (15); Business = pricing/purge decisions (B-05).
Optional enhancements (non-blocking): fullscreen/4K UI, chart in command center, prefers-reduced-motion media query, admin entitlement editor UI, multi-instance WS scale-out.
