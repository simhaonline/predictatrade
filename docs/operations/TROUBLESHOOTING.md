# Troubleshooting

**Version:** v1.2.0 — Advanced Risk + Backtesting  
**Date:** 18 August 2026

---

## Common Issues

### No signals generated (all NO-TRADE)

- Check if Windows Agent is connected: `GET /api/v1/agents/status`
- Check market data quality: `GET /api/v1/market/snapshot`
- Verify regime/session suitability for each strategy
- Check Prometheus: `pat_signals_generated_total`
- **Note: NO-TRADE is a valid result — not a bug**

### Score 209.7 / P 99.1% / NO-TRADE

- This was a Stage 2 defect — **FIXED** in v1.0.0
- Calibration now clamps to [0,100]
- StandardScalping now applies family caps
- Conflict cases now produce WAIT (not NO-TRADE)

### PTB modules not contributing to score

- This is by design — all modules are SHADOW by default
- Check `trading.ptb_feature_flags` for module modes
- Shadow mode = zero score impact
- To activate: see Admin Guide > PTB Feature Flag Management

### DXY/silver/yield correlations unavailable

- External feeds not connected through Master Node
- Gold role returns UNKNOWN (correct behavior)
- Correlation engine ready — awaiting live data feed

### Loss recovery state stuck in RECOVERY or HALTED

- Check recovery state: `SELECT * FROM trading.recovery_states WHERE state != 'NORMAL'`
- RECOVERY exits after `recovery_exit_after_wins` (default: 2 wins)
- HALTED exits after `halt_cooldown` (default: 60 minutes)
- DAILY_LIMIT resets at new UTC trading day
- Manually reset (admin only): update `trading.recovery_states` state to 'NORMAL'

### Adaptation reducing risk too aggressively

- Check current market phase: `SELECT * FROM trading.adaptation_history ORDER BY timestamp DESC LIMIT 5`
- HIGH_VOLATILITY and MANIPULATIVE phases intentionally reduce risk
- Adaptation can only make the system more conservative — never increases above hard limits
- Disable temporarily: `export ADAPTATION_ENABLED=false`

### Hedging not executing

- Hedging is DISABLED by default — check `HEDGING_ENABLED` env var
- Verify broker supports hedging and netting mode
- Check `trading.hedge_positions` for active hedges
- Grid and options hedging are OFF by default

### ML adaptation not working

- ML is disabled by default: `ML_ADAPTATION_ENABLED=false`
- Requires a trained model artifact
- Check model registry: `SELECT * FROM ai.model_registry WHERE status = 'active'`
- Python training pipeline: `research/src/patresearch/ml_training.py`

### RL optimizer not working

- RL is disabled by default: `RL_MODE=disabled`
- Modes: disabled → shadow → filter_only → live_approved
- Filter mode can only veto (NO_TRADE) — cannot create trades
- Check: `SELECT * FROM trading.rl_training_history ORDER BY created_at DESC LIMIT 5`

### Sentiment not updating

- Sentiment is disabled by default: `SENTIMENT_ENABLED=false`
- Requires API credentials for sentiment provider
- Check provider health: `SELECT * FROM trading.sentiment_snapshots`
- Falls back to neutral when unavailable — does not break signals
- Async background refresh — never blocks signal hot path

### Backtest results not matching live performance

- Verify same strategy config is used
- Check `git_commit_sha` in `trading.backtest_runs` matches production
- Backtest uses conservative same-bar SL/TP (worst case)
- Execution simulation includes spread, slippage, commission
- Walk-forward analysis required for robust validation
- Past performance does not guarantee future results

### Database migration errors

- Migrations 006-015 are additive (`CREATE TABLE IF NOT EXISTS`) — no destructive changes
- TimescaleDB hypertable creation is wrapped in DO$$ — fails silently if extension absent
- Check: `SELECT * FROM trading.ptb_feature_flags`
- Verify all 15 migrations applied: `SELECT count(*) FROM pg_migrations` (if tracking table exists)
- Or check: `ls database/migrations/*.sql | wc -l` (should be 15)

### Build errors

```bash
# Go realtime
cd realtime && go build ./...
cd realtime && go vet ./...

# NestJS control
cd control && npm run build

# Python research
cd research && python3 -m pytest tests/

# Frontend
cd frontend && npm run build
```

### Test failures

```bash
# Go tests with verbose output
cd realtime && go test ./... -v -count=1 -timeout=120s

# NestJS tests
cd control && npm test

# Python tests
cd research && python3 -m pytest tests/ -v

# Frontend tests
cd frontend && npm test

# Total: 448 tests (243 Go + 98 Python + 68 NestJS + 39 Frontend)
```

### WebSocket connection issues

- Check origin validation in Nginx config
- Verify JWT token is valid and not expired
- Check: `wss://live.predictatrade.com/ws/v1` (dashboard) or `/ws/v1/agent` (agent)
- Agent requires device token, not user JWT

### Windows Agent not connecting

- Verify `PAT_LIVE_WS_URL` points to correct WebSocket URL
- Check `PAT_API_URL` points to correct API URL
- Verify license key is valid
- Check Windows firewall allows outbound connections
- Run validation script: `scripts/windows/validate-agent.ps1`

### MT4/MT5 EA not receiving signals

- Verify Windows Agent is running and connected
- Check shared file (FILE_COMMON) path is accessible
- Verify AutoTrading is enabled in terminal
- Check EA log for errors
- Ensure Master Node EA is running on a separate chart

### Data authenticity guard rejecting data

- All production signals require `source_type = LIVE_MASTER_NODE`
- Test, mock, demo, fixture, synthetic data is rejected
- Check: `SELECT * FROM trading.data_provenance_log ORDER BY market_timestamp DESC LIMIT 10`
- Verify Master Node EA is running and publishing data

### Prometheus metrics not appearing

- Check Prometheus target health: `http://localhost:9090/targets`
- Verify Go engine is exposing `/metrics` on port 8081
- Verify NestJS is exposing `/metrics`
- Check Grafana dashboard: `infra/grafana/dashboards/gate-health.json`

### Emergency stop

- Halt all trading: `POST /api/v1/operations/halt-trading` (admin)
- Resume trading: `POST /api/v1/operations/resume-trading` (admin)
- Pause signal delivery: `POST /api/v1/operations/pause-signals` (admin)


## v1.3.0 Troubleshooting

### SMTP Not Sending Emails
- Check: journalctl -u pat-control.service | grep SMTP
- Verify SMTP ports open: timeout 5 bash -c "echo >/dev/tcp/mail.predictatrade.com/587"
- If blocked by cloud provider (netcup): request SMTP unblock or use HTTP email API
- Verify credentials in infra/env/control.env (gitignored)

### Gate Fail-Closed (All Signals BLOCKED)
- This is CORRECT behavior after restart - safety-critical gates start as UNKNOWN
- Gates hydrate when Windows Agent connects (TICK/HEARTBEAT/MASTER_INIT)
- Exposure/margin gates hydrate from MARKET_SNAPSHOT with account_info
- Check: journalctl -u pat-rt.service | grep "gate|Broker account|execution permit"

### DXY Unavailable (STANDARD_SWING/TREND_SWING NO-TRADE)
- Verify TWELVEDATA_API_KEY is set in env config
- Check: journalctl -u pat-rt.service | grep dxy
- DXY is computed from 6 currencies via Twelve Data API
- If rate limited (429): DXY marked RATE_LIMITED, will retry in 5 minutes

### COT Unavailable (HTTP 402)
- FMP free tier does not include COT endpoints
- Upgrade FMP subscription or use alternative COT data source
- COT is optional (weight=0 by default) - does not block signals
