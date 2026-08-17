---
name: observability-sre
description: Implement/review OpenTelemetry, Prometheus, Grafana, structured logs, SLOs, alerts, backup/restore, DR and failure coverage across trading and finance.
---

# observability-sre

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and repository `AGENTS.md`.

## Workflow
1. Propagate correlation IDs across HTTP/WS/signal/license/device/execution/broker flows.
2. Instrument data quality and receipt-to-publish pipeline.
3. Instrument gate p50/p95/p99, fail-closed/degradation, strategy/no-trade/cost/slippage/outcomes.
4. Instrument calibration/drift/parity/claim expiry and WS delivery/reconnect/resync/ACK SLO.
5. Instrument control-plane/financial reconciliation and Windows/MT heartbeats.
6. Define SLOs/alerts/runbooks and test backup/restore/DR/selected chaos.

## Canonical stack
OpenTelemetry + Prometheus + Grafana + structured JSON logs.

## Output Contract
Return SOW sections addressed, files examined/changed, tests/checks + exact results, unresolved risks/blockers, and rollback/next action where applicable.
