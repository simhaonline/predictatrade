---
name: prometheus-grafana-dashboards
description: "Prometheus rules, Grafana dashboards, SLOs."
---

# prometheus-grafana-dashboards

Use for PAT observability configs.

## Stack
pat-prometheus, pat-grafana (:3001), OpenTelemetry Go+NestJS

## Configs
infra/prometheus/prometheus.yml, rules.yml
infra/grafana/dashboards/gate-health.json

## Key Metrics
Gate p50/p95/p99, signal latency, WS delivery, strategy metrics
SL enforcement violations, agent heartbeats, DB pool, Valkey hits
Broker slippage, rejections, partial fills

## Dashboard: Gate Health
14 gates with status, p99, fail count. Red on P0 failure.
