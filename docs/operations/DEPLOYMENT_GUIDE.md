# Deployment Guide

## Version: v1.4.0 — Color Palette + Signal Delivery + TP/SL Geometry Fix

## Services

| Service | Port | Binary | Systemd Unit |
|---------|------|--------|-------------|
| Realtime Engine (Go) | 13081 | `realtime-engine` | `predictatrade-realtime.service` |
| Control Plane (NestJS) | 13080 | `node dist/main.js` | `predictatrade-control.service` |
| Frontend (Next.js) | 13000 | `next start` | `predictatrade-frontend.service` |
| Status Page | 13083 | `next start` | `predictatrade-status.service` |
| PostgreSQL + TimescaleDB | 5432 | system | system |
| Valkey | 6379 | system | system |
| Nginx | 80/443 | system | system |
| Prometheus | 9090 | docker | docker |
| Grafana | 3001 | docker | docker |

## Environment Variables

### Realtime Engine (infra/env/realtime.env)

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | (from file) | Loaded from `/srv/predictatrade/xauusd/database_url.txt` |
| `VALKEY_ADDR` | `127.0.0.1:6379` | Valkey/Redis address |
| `HTTP_HOST` | `127.0.0.1` | HTTP bind host |
| `HTTP_PORT` | `13081` | HTTP bind port |
| `PROVIDER_MODE` | `agent` | Market data provider (agent=real MT5, simulated=DEV ONLY) |
| `SYMBOLS` | `XAUUSD` | Trading symbols |
| `MAX_SPREAD_PIPS` | `3.0` | Maximum spread |
| `MIN_RR` | `1.5` | Minimum risk-reward |
| `MAX_EXPOSURE` | `5.0` | Maximum exposure |
| `PTB_ENABLED` | `true` | PTB master switch |
| `PTB_SHADOW_MODE` | `true` | PTB shadow mode (zero score impact) |
| `LOSS_RECOVERY_ENABLED` | `true` | Loss recovery / capital protection |
| `MAX_DAILY_LOSS_PERCENT` | `2.0` | Max daily loss percentage |
| `MAX_CONSECUTIVE_LOSSES` | `2` | Max consecutive losses before recovery |
| `ADAPTATION_ENABLED` | `true` | Rule-based market phase adaptation |
| `HEDGING_ENABLED` | `false` | Controlled hedging (DISABLED by default) |
| `GRID_HEDGING_ENABLED` | `false` | Grid hedging (OFF by default) |
| `OPTIONS_HEDGING_ENABLED` | `false` | Options hedging (OFF by default) |
| `ML_ADAPTATION_ENABLED` | `false` | ML-based adaptation (research/offline) |
| `RL_MODE` | `disabled` | RL mode: disabled, shadow, filter_only, live_approved |
| `SENTIMENT_ENABLED` | `false` | Sentiment engine (requires API credentials) |
| `COT_ENABLED` | `false` | COT provider (Financial Modeling Prep API) |
| `FMP_API_KEY` | (not set) | Financial Modeling Prep API key for COT data |
| `COT_SYMBOL` | `GC` | COT futures contract symbol (Gold) |
| `DXY_ENABLED` | `false` | DXY provider (Twelve Data API) |
| `TWELVEDATA_API_KEY` | (not set) | Twelve Data API key for DXY computation |

### Control Plane (infra/env/control.env)

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | (from file) | Loaded from `/srv/predictatrade/xauusd/database_url.txt` |
| `JWT_SECRET` | (from file) | Loaded from `jwt_secret.txt` or env var (min 32 chars) |
| `CORS_ORIGINS` | (required) | Comma-separated allowed CORS origins |
| `CONTROL_HOST` | `127.0.0.1` | Bind host |
| `CONTROL_PORT` | `13080` | Bind port |
| `EMAIL_PROVIDER` | `smtp` | Email provider (smtp or console) |
| `EMAIL_FROM` | `no-reply@predictatrade.com` | Sender email address |
| `SMTP_HOST` | `mail.predictatrade.com` | SMTP server host |
| `SMTP_PORT` | `587` | SMTP port (587=STARTTLS, 465=SSL) |
| `SMTP_USERNAME` | `no-reply@predictatrade.com` | SMTP authentication username |
| `SMTP_PASSWORD` | (secret) | SMTP password (from env file, gitignored) |

### Frontend (infra/env/frontend.env)

| Variable | Default | Description |
|----------|---------|-------------|
| `NEXT_PUBLIC_API_URL` | `https://api.predictatrade.com/api/v1` | API base URL |
| `NEXT_PUBLIC_WS_URL` | `wss://live.predictatrade.com/ws/v1` | WebSocket URL |
| `NEXT_PUBLIC_PLATFORM_URL` | `https://platform.predictatrade.com` | Platform URL |

### Windows Agent (infra/env/windows-agent.env)

| Variable | Default | Description |
|----------|---------|-------------|
| `PAT_LIVE_WS_URL` | `wss://live.predictatrade.com/ws/v1/agent` | Realtime WS URL |
| `PAT_API_URL` | `https://api.predictatrade.com/api/v1` | Control API URL |
| `PAT_UPDATE_CHANNEL` | `STABLE` | Update channel |

## Secret Files (Gitignored)

The following secret files are gitignored and must be created on the production server:

| File | Purpose | How to Generate |
|------|---------|-----------------|
| `/srv/predictatrade/xauusd/database_url.txt` | PostgreSQL connection string | `echo "postgresql://user:pass@host:5432/db?sslmode=disable" > database_url.txt` |
| `/srv/predictatrade/xauusd/jwt_secret.txt` | JWT signing secret | `openssl rand -base64 32 > jwt_secret.txt` |
| `/srv/predictatrade/xauusd/infra/env/*.env` | Service environment files | Copy from templates, fill in real values |

All secret files should have `chmod 600` (owner read/write only).

## Deployment Steps

### 1. Database

```bash
# Apply all migrations (1-15)
./scripts/migrate.sh up

# Or manually:
for f in database/migrations/*.sql; do
  psql $DATABASE_URL -f "$f"
done
```

### 2. Realtime Engine

```bash
cd realtime
go build -ldflags "-s -w" -o bin/realtime-engine ./cmd/realtime-engine
./bin/realtime-engine
```

Production: Copy binary to `/opt/predictatrade/realtime/bin/`, use systemd unit.

### 3. Control Plane

```bash
cd control
npm ci
npm run build
npm start
```

Production: Copy `dist/` to `/opt/predictatrade/control/dist/`, use systemd unit.

### 4. Frontend

```bash
cd frontend
npm ci
npm run build
npm start
```

Production: Copy `.next/` to `/opt/predictatrade/frontend/`, use systemd unit.

### 5. Windows Agent

```bash
cd windows-agent
GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o bin/pat-pat-agent.exe ./cmd/agent
```

Distribute to MT4/MT5 terminal machines. Requires Windows.

### 6. Nginx

Copy `infra/nginx/` configs to `/etc/nginx/`. Enable sites:
```bash
ln -s /etc/nginx/sites-available/api.predictatrade.com.conf /etc/nginx/sites-enabled/
ln -s /etc/nginx/sites-available/live.predictatrade.com.conf /etc/nginx/sites-enabled/
ln -s /etc/nginx/sites-available/platform.predictatrade.com.conf /etc/nginx/sites-enabled/
ln -s /etc/nginx/sites-available/predictatrade.com.conf /etc/nginx/sites-enabled/
ln -s /etc/nginx/sites-available/status.predictatrade.com.conf /etc/nginx/sites-enabled/
```

### 7. Prometheus + Grafana (Docker)

```bash
docker compose up -d prometheus grafana
```

## Feature Configuration

### PTB (Professional Trader Brain)
PTB defaults to SHADOW mode — all modules calculate and persist but have zero production score impact.
```bash
export PTB_SHADOW_MODE=false  # Activate after validation
```

### Loss Recovery
Enabled by default with conservative thresholds. State machine: NORMAL → RECOVERY → HALTED / DAILY_LIMIT.
```bash
export LOSS_RECOVERY_ENABLED=true   # Default
export MAX_DAILY_LOSS_PERCENT=2.0   # 2% daily loss limit
```

### Adaptation
Rule-based market phase adaptation. Adjusts parameters conservatively — never increases risk above hard limits.
```bash
export ADAPTATION_ENABLED=true  # Default
```

### Hedging
DISABLED by default. Must be explicitly enabled. Requires broker hedging support.
```bash
export HEDGING_ENABLED=true  # Enable (requires broker support)
```

### ML / RL / Sentiment
All disabled by default — require external configuration and validation.
```bash
export ML_ADAPTATION_ENABLED=true   # Requires trained model
export RL_MODE=shadow               # disabled → shadow → filter_only → live_approved
export SENTIMENT_ENABLED=true       # Requires API credentials
```

## Health Checks

| Endpoint | Service |
|----------|---------|
| `GET /api/v1/health` | NestJS control plane |
| `GET /metrics` | Go realtime engine (Prometheus) |
| `GET /metrics` | NestJS control plane (Prometheus) |

## Domains

| Domain | Purpose | Upstream |
|--------|---------|----------|
| `predictatrade.com` | Public website | Next.js frontend |
| `platform.predictatrade.com` | User dashboard | Next.js frontend |
| `api.predictatrade.com` | REST API | NestJS + Go realtime |
| `live.predictatrade.com` | WebSocket + agent | Go realtime |
| `status.predictatrade.com` | Status page | Next.js status |

## Testing

```bash
# All tests
make test

# Individual
cd realtime && go test -race -count=1 -timeout=120s ./...
cd control && npm test
cd frontend && npm test
cd research && python3 -m pytest tests/

# Total: 448 tests (243 Go + 98 Python + 68 NestJS + 39 Frontend)
```

## Backup

```bash
# Database backup
./scripts/backup/backup.sh

# Restore test
./scripts/backup/restore_test.sh

# Off-host backup
./scripts/backup/offhost_backup.sh
```
