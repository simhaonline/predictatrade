# Predict-A-Trade XAUUSD — Full Production Audit Report

**Date:** 2026-08-21T12:16:51+03:00  
**Version:** v1.7.0  
**Overall Status:** PASS  
**Failed:** 0 | **Warned:** 0  

---

## Executive Summary

| Metric | Value |
|--------|-------|
| Overall Status | PASS |
| Failed Checks | 0 |
| Warning Checks | 0 |
| Go Test Packages | 24/24 |
| Python Tests | 127 |
| Frontend Tests | 70 |
| API Latency | < 50ms |
| Goroutines | < 600 |

---

## Detailed Findings

go_vet: PASS — 0 issues
go_build: PASS — 0 errors
go_tests: PASS — 28/24 packages pass, 0 FAIL
go_mod_verify: PASS — all modules verified
frontend_build: PASS — build successful (next/ detected)
frontend_tests: PASS — 70 tests pass
typescript_check: PASS — 0 errors
python_tests: PASS — 127 passed, 7 warnings in 2.84s
port_go_engine: PASS — port 13081 listening
port_frontend: PASS — port 13082 listening
port_nestjs: PASS — port 13080 listening
port_postgres: PASS — port 5432 listening
port_valkey: PASS — port 6379 listening
port_ollama: PASS — port 11434 listening
nginx_config: PASS — syntax ok
valkey_ping: PASS — PONG
db_connection: PASS — connected
env_vars: PASS — all required keys present
cron_health_check: PASS — installed
cron_training: PASS — installed
parity_check: PASS — 1000 samples, MAPE < 0.0001
wilder_smoothing: PASS — RSI/ATR/TR verified against known vectors
onnx_model_sanity: PASS — non-constant output
indicator_count: PASS — 35/42 live
geometry_validation: PASS — 49/49 valid
notrade_reasons: PASS — 1/1 have reasons
signal_flow: PASS — 49 directional signals
evidence_scoring: PASS — 49/49 have evidence
dashboard_load: PASS — HTTP 307 (200 or redirect=ok)
signal_endpoint: PASS — HTTP 200
health_endpoint: PASS — HTTP 200
world_writable: PASS — 0 world-writable files
hardcoded_secrets: PASS — 0 hardcoded secrets in production code (test fixtures excluded)
ssl_certificate: PASS — valid > 24h
cpu_usage: PASS — 0.0% used
memory_usage: PASS — 8% used (5622/64295MB)
disk_usage: PASS — 25% used
goroutines: PASS — 20 via pprof (< 2000)
api_latency: PASS — 1.085000ms (< 50ms)
service_predictatrade-realtime: PASS — active
service_predictatrade-frontend: PASS — active
ml_enabled: PASS — true
ollama_connected: PASS — HTTP 200
cot_data: PASS — available (4 log entries)
dxy_data: PASS — available (13 log entries)
health_manager: PASS — wired (6 refs)
geometry_validator: PASS — file exists
capital_protection: PASS — file exists
candidate_geometry: PASS — file exists
mql_eas: PASS — MT4 + MT5 exist
ml_models: PASS — 5 files in models/
output=['output']
scaler_json: PASS — OK: 42 features

---

## Evidence

See full log at: `logs/audit_20260821_121651.log`  
JSON report at: `audit/report_20260821.json`

---

## Blockers

✅ Zero blockers — all critical checks passed

---

## Sign-off

**STATUS: PASS**  
**Date:** 2026-08-21T12:16:51+03:00  
**Auditor:** Automated Full Audit Script  
