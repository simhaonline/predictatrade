# Deployment Guide
## v1.16.0 — 26 August 2026

### Quick Start

```bash
git clone https://github.com/simhaonline/predictatrade.git
cd predictatrade/xauusd
cp realtime/.env.example realtime/.env
# Edit: TWELVEDATA_API_KEY, FMP_API_KEY, PROVIDER_MODE
docker compose up -d
curl http://localhost:13081/health
```

### Required Environment Variables
| Variable | Required | Description |
|----------|:--------:|-------------|
| DATABASE_URL | YES | PostgreSQL connection |
| VALKEY_ADDR | YES | Valkey address |
| TWELVEDATA_API_KEY | YES | XAUUSD candles |
| FMP_API_KEY | YES | COT/DXY data |
| JWT_SECRET | YES | Auth signing (32+ chars) |
| PROVIDER_MODE | YES | agent=MT5, simulated=dev |

### Service Ports
pat-postgres(5432), pat-valkey(6379), pat-realtime(13081), pat-control(13080), pat-frontend(13082), pat-nginx(80/443), pat-status(13083), pat-live-terminal(13090), pat-prometheus(9090), pat-grafana(3001), pat-ntfy(8091)

### Build
```
make go-build       # Realtime engine
make control-build  # NestJS
make frontend-build # Next.js
make test           # All 3 suites
```
