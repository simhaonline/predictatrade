# 34 — Master Traceability Matrix

| Capability | Code | DB | API | Runtime | UI | Test | Evidence | Status |
|---|---|---|---|---|---|---|---|---|
| XAUUSD tick ingestion | agent_provider, aggregator | ticks 7.1M | /market/* | LIVE Fri session | live cards | unit+integration | 06 | **VERIFIED** (auth gap) |
| Candle persistence/OHLC | aggregator | candles+hypertables | /candles | yes | charts admin-only | parity suites | 04/08 | PARTIAL (553 corrupted bars) |
| Indicator engine (42f) | features registry | indicator_history | snapshot | yes | indicator-monitor | pytest127/go | 08 | VERIFIED |
| Regime engine v2 | regime.go | regime_history | state/diagnostics | yes | scoring-board | go tests | 10 | VERIFIED |
| 4 strategies | pkg/strategy+configs | strategy_evaluations | /strategies | yes | strategies pages | go tests | 09 | VERIFIED |
| Scoring | scorer.go | raw_score | signals | yes | pipeline UI | go tests | 11 | VERIFIED |
| Calibrated probability | calibration/consumer | calibrated_probability | signals | yes | prob badges | none real | 11 | **SIMULATED metadata** |
| Hard gates ×14 | gates/ | risk_decisions | n/a | 12 evaluated | decorative panel | partial | 13 | PARTIAL (2 dead,1 vacuous) |
| Emergency stop / daily loss | types only | capital_protection_events empty | none | absent | none | none | 13 | **UNWIRED** |
| Signal refs/idempotency | signal/ + valkey | signals/outbox | resume | partial | n/a | cooldown tests | 12 | PARTIAL (fail-open, outbox stuck) |
| Delivery/acks/reconciliation | delivery.go,reconciler | deliveries=0 | WS dead to UI | no | fallback REST | none | 12/25 | **BROKEN** |
| AuthN/MFA/RBAC | control/auth | iam.* | /auth/* | 401s proven | login/MFA pages | jest subset | 18 | VERIFIED (MFA page bug) |
| Licensing/devices | licensing,device-auth | licensing.* | activate/refresh | 2 devices | mt4-mt5 page | specs | 19 | PARTIAL (IDOR,dead tokens) |
| Subscriptions/plans | billing/subscriptions | billing.*,control.plans | plans/subs | 1 fake-active row | billing page | policy specs | 20 | **STUBBED** |
| Billing/PSP | billing.service | payments(1 fake) | webhook stub | unverified event | invoices page | none | 22 | **STUB** |
| Referrals/commissions | referrals,commission-engine | ledger=0 | read-only APIs | no data | referrals page | spec w/ wrong rates | 21 | **UNWIRED writers** |
| Payouts | payouts.service | payouts=0 | broken INSERT | crash | admin approve btn | none | 21 | **BROKEN** |
| Audit log | audit module | audit_events 186 | /audit admin | yes | logs page | none | 18 | PARTIAL (best-effort writes) |
| News gate | pkg/news+fmp | economic_events | news_risk on signals | LIVE sync | badge on signals | none live | 17 | VERIFIED |
| COT/DXY | pkg/macro | cot_*/ptb tables | ptb shadow | loops run | none visible | none | 16/17 | WIRED weight-0 honest |
| ML ONNX | mlengine | predictions tables? | none | model load FAILS | none | engine tests | 16 | INERT |
| Sentiment LLM | ollama client | sentiment_* | none | unreachable host | none | client tests | 16 | INERT |
| Windows agent/EA | windows-agent,mql | device rows | agent WS | 1 connected | download page | validation dir | 15 | PARTIAL (unauth channel; broker UNVERIFIED) |
| Backtest API | backtest controller/runs | backtest_runs | /backtest/* | endpoint live | backtest panel | api specs | 23 | PARTIAL (no entitlement gate) |
| Observability | prom/grafana configs | metrics series | /metrics | scraped | grafana up | n/a | 27 | PARTIAL (no alerting proof) |
| Backups/DR | scripts | dump files Aug18 | n/a | stale | n/a | restore drill missing | 27 | **FAILING process** |

Status legend per §105. Counts: VERIFIED 8 · PARTIAL 12 · BROKEN/STUBBED/SIMULATED 6 · UNWIRED 3 · failing-process 1.
