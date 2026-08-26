---
name: observability-sre
description: "OpenTelemetry, Prometheus, Grafana, structured logs."
---

# observability-sre

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and `AGENTS.md`.

## Canonical Stack
OpenTelemetry + Prometheus + Grafana + structured JSON logs + ntfy notifications.

## Installed Services
- pat-prometheus (prom/prometheus)
- pat-grafana (grafana/grafana on port 3001)
- pat-ntfy (binwiederhier/ntfy on port 8091)

## Workflow
1. Propagate correlation IDs across HTTP/WS/signal/license/device/execution/broker flows.
2. Instrument data quality and receipt-to-publish pipeline.
3. Gate p50/p95/p99, fail-closed/degradation, strategy/no-trade/cost/slippage.
4. Calibration/drift/parity, WS delivery/reconnect/resync/ACK SLO.
5. Control-plane/financial reconciliation, Windows/MT heartbeats.
6. SLOs/alerts/runbooks, backup/restore/DR, selected chaos tests.

## Configs
- infra/prometheus/prometheus.yml, rules.yml
- infra/grafana/dashboards/gate-health.json
