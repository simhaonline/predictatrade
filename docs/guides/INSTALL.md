# Installation Guide

**Version:** v1.4.0 — Color Palette + Signal Delivery + TP/SL Geometry Fix  
**Date:** 18 August 2026

---

## Prerequisites

- Go 1.21+
- Node.js 18+
- Python 3.10+
- PostgreSQL 14+ with TimescaleDB extension
- Valkey/Redis 7+
- Nginx (production ingress)

---

## Installation

### 1. Database

```bash
createdb predictatrade
psql predictatrade -c "CREATE EXTENSION IF NOT EXISTS timescaledb;"
psql predictatrade -c "CREATE EXTENSION IF NOT EXISTS pgcrypto;"
psql predictatrade -c "CREATE EXTENSION IF NOT EXISTS vector;"  # pgvector for AI

# Apply all migrations (001-015)
./scripts/migrate.sh up

# Or manually:
for f in database/migrations/*.sql; do
    psql predictatrade -f "$f"
done
```

This creates 12 schemas, 165 tables, 12 hypertables, and 6 database roles.

### 2. Realtime Engine

```bash
cd realtime
go build -o bin/realtime-engine ./cmd/realtime-engine
export DATABASE_URL="postgres://user:pass@localhost/predictatrade"
export VALKEY_ADDR="127.0.0.1:6379"
export HTTP_HOST="127.0.0.1"
export HTTP_PORT="13081"
export PROVIDER_MODE="agent"       # agent=real MT5, simulated=DEV ONLY
export PTB_ENABLED=true
export PTB_SHADOW_MODE=true

# Advanced Risk (v1.1.0)
export LOSS_RECOVERY_ENABLED=true
export MAX_DAILY_LOSS_PERCENT=2.0
export ADAPTATION_ENABLED=true
export HEDGING_ENABLED=false        # DISABLED by default
export ML_ADAPTATION_ENABLED=false  # Research only
export RL_MODE=disabled              # disabled → shadow → filter_only → live_approved
export SENTIMENT_ENABLED=false       # Requires API credentials

./bin/realtime-engine
```

### 3. Control Plane

```bash
cd control
npm install
npm run build
export DATABASE_URL="postgres://user:pass@localhost/predictatrade"
export JWT_SECRET="your-secret-here"
export CORS_ORIGINS="https://platform.predictatrade.com"
npm start
```

### 4. Frontend

```bash
cd frontend
npm install
export NEXT_PUBLIC_API_URL="https://api.predictatrade.com/api/v1"
export NEXT_PUBLIC_WS_URL="wss://live.predictatrade.com/ws/v1"
export NEXT_PUBLIC_PLATFORM_URL="https://platform.predictatrade.com"
npm run build
npm start
```

### 5. Windows Agent

```bash
cd windows-agent
GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o bin/pat-pat-agent.exe ./cmd/agent
```

Distribute `pat-pat-agent.exe` to Windows machines running MT4/MT5 terminals.

### 6. MT4/MT5 EAs

Install the following EAs on MT4/MT5 charts:

| EA | File | Purpose |
|----|------|---------|
| PredictATrade_MT4 | `mql/mt4/PredictATrade_MT4.mq4` | Signal display/execution (MT4) |
| PredictATrade_MT5 | `mql/mt5/PredictATrade_MT5.mq5` | Signal display/execution (MT5) |
| PredictATrade_MasterNode_MT4 | `mql/mt4/PredictATrade_MasterNode_MT4.mq4` | Tick data provider (MT4) |
| PredictATrade_MasterNode_MT5 | `mql/mt5/PredictATrade_MasterNode_MT5.mq5` | Tick data provider (MT5) |

- Set license key in EA inputs
- Enable AutoExecute for automated trading (default: false = signal only)

### 7. Nginx (Production)

```bash
# Copy configs
cp infra/nginx/nginx.conf /etc/nginx/nginx.conf
cp infra/nginx/sites-available/*.conf /etc/nginx/sites-available/
ln -s /etc/nginx/sites-available/api.predictatrade.com.conf /etc/nginx/sites-enabled/
ln -s /etc/nginx/sites-available/live.predictatrade.com.conf /etc/nginx/sites-enabled/
ln -s /etc/nginx/sites-available/platform.predictatrade.com.conf /etc/nginx/sites-enabled/
ln -s /etc/nginx/sites-available/predictatrade.com.conf /etc/nginx/sites-enabled/
ln -s /etc/nginx/sites-available/status.predictatrade.com.conf /etc/nginx/sites-enabled/
```

### 8. Prometheus + Grafana (Docker)

```bash
docker compose up -d prometheus grafana
```

### 9. Systemd Services

```bash
cp infra/systemd/predictatrade-realtime.service /etc/systemd/system/
cp infra/systemd/predictatrade-control.service /etc/systemd/system/
cp infra/systemd/predictatrade-frontend.service /etc/systemd/system/
cp infra/systemd/predictatrade-status.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable predictatrade-realtime predictatrade-control predictatrade-frontend predictatrade-status
```

---

## Verification

```bash
# Check health
curl http://127.0.0.1:13081/health          # Go realtime
curl http://127.0.0.1:13080/api/v1/health    # NestJS control plane

# Check readiness
curl http://127.0.0.1:13081/ready

# Check agent status
curl http://127.0.0.1:13081/api/v1/agents/status

# Check market state
curl http://127.0.0.1:13081/api/v1/market/state

# Check strategies
curl http://127.0.0.1:13081/api/v1/strategies
```

---

## Testing

```bash
# Go realtime engine (243 tests)
cd realtime && go build ./... && go vet ./... && go test -race -count=1 -timeout=120s ./...

# NestJS control plane (68 tests)
cd control && npm test

# Python research (98 tests — includes backtesting)
cd research && python3 -m pytest tests/

# Next.js frontend (39 tests)
cd frontend && npm test

# Total: 448 tests, 0 failures
```

---

## PTB Configuration

PTB defaults to SHADOW mode. All modules calculate and persist but have zero production score impact.

```bash
export PTB_ENABLED=true         # master switch
export PTB_SHADOW_MODE=true    # zero score impact (default)
```

All thresholds configurable in `realtime/internal/ptb/config.go`.

---

## Advanced Risk Configuration

### Loss Recovery / Capital Protection

```bash
export LOSS_RECOVERY_ENABLED=true      # Default: enabled
export MAX_DAILY_LOSS_PERCENT=2.0       # 2% daily loss limit
export MAX_DAILY_LOSS_COUNT=3           # Max daily loss count
export MAX_CONSECUTIVE_LOSSES=2         # Before entering recovery
```

State machine: NORMAL → RECOVERY → HALTED / DAILY_LIMIT

### Rule-Based Adaptation

```bash
export ADAPTATION_ENABLED=true  # Default: enabled
```

Adjusts parameters conservatively based on market phase. Never increases risk above hard limits.

### Controlled Hedging

```bash
export HEDGING_ENABLED=false       # DISABLED by default — requires broker support
export GRID_HEDGING_ENABLED=false  # OFF by default
export OPTIONS_HEDGING_ENABLED=false  # OFF by default
```

### ML Adaptation

```bash
export ML_ADAPTATION_ENABLED=false  # Research only — requires trained model
```

### RL Strategy Optimizer

```bash
export RL_MODE=disabled  # disabled → shadow → filter_only → live_approved
```

### Sentiment Engine

```bash
export SENTIMENT_ENABLED=false  # Requires API credentials
```

Async background refresh — never blocks signal hot path.

---

## Backtesting (v1.2.0)

```bash
# Run a backtest
cd research && python3 -m patresearch.backtesting.cli run --strategy STANDARD_SCALPING --seed 42

# Walk-forward analysis
cd research && python3 -m patresearch.backtesting.cli walk-forward --strategy STANDARD_SCALPING

# Monte Carlo analysis
cd research && python3 -m patresearch.backtesting.cli monte-carlo --runs 1000

# Parameter sensitivity
cd research && python3 -m patresearch.backtesting.cli sensitivity --strategy STANDARD_SCALPING
```

See: [Backtesting Guide](../BACKTESTING.md)

---

## Environment Files

Canonical environment files are in `infra/env/`:

| File | Service |
|------|---------|
| `infra/env/canonical.env` | Shared defaults |
| `infra/env/realtime.env` | Go realtime engine |
| `infra/env/control.env` | NestJS control plane |
| `infra/env/frontend.env` | Next.js frontend |
| `infra/env/status.env` | Status page |
| `infra/env/windows-agent.env` | Windows Agent |

---

## Database Backup

```bash
# Database backup
./scripts/backup/backup.sh

# Restore test
./scripts/backup/restore_test.sh

# Off-host backup
./scripts/backup/offhost_backup.sh
```

See: [Database Backup and Recovery](../database/DATABASE_BACKUP_AND_RECOVERY_REPORT.md)


## v1.3.0 Secret File Setup

Before starting services, create the following secret files:

### 1. Database URL
```bash
echo "postgresql://pat_admin:pat_local_dev_only@127.0.0.1:5432/predictatrade?sslmode=disable" > /srv/predictatrade/xauusd/database_url.txt
chmod 600 /srv/predictatrade/xauusd/database_url.txt
```

### 2. JWT Secret
```bash
openssl rand -base64 32 > /srv/predictatrade/xauusd/jwt_secret.txt
chmod 600 /srv/predictatrade/xauusd/jwt_secret.txt
```

### 3. Environment Files
Copy templates from `infra/env/` and fill in real values:
- `control.env` — SMTP credentials, CORS origins
- `realtime.env` — COT/DXY API keys, provider mode
- `canonical.env` — Domain routing

All secret files are gitignored.

## v1.4.0 Updates (19 August 2026)

### Frontend Color Palette

The frontend now uses the approved Predict-A-Trade color palette. No installation changes needed — colors are defined in CSS variables and Tailwind tokens. If colors appear invisible, ensure HSL values in `globals.css` include `%` signs (e.g., `210 40% 98%`).

### Signal Delivery to MT4/MT5

The Go engine now broadcasts signals to Windows Agents automatically. No configuration change needed — `BroadcastSignalToAgents()` is called for every directional signal. Verify by checking logs for "Signal broadcast to Windows Agents for MT4/MT5 delivery".

### TP/SL Geometry

TP/SL levels are now ATR-based. No configuration change needed — ATR multipliers are defined per-strategy in `strategies.go`. The MinRR gate still validates R:R and rejects insufficient signals.

### MQL EA v1.05

Update both MT4 and MT5 EAs to v1.05. New input parameters:
- Strategy toggles: `ReceiveStandardScalping`, `ReceiveUltraScalping`, `ReceiveStandardSwing`, `ReceiveTrendSwing` (all `true` by default)
- Direction filters: `ReceiveBuy`, `ReceiveSell`, `ReceiveBuyCandidate`, `ReceiveSellCandidate` (all `true` by default)

Attach the EA to a chart and configure inputs in the EA properties dialog.
