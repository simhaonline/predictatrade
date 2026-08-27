# Deployment Guide
## v1.16.0 — 26 August 2026

### Quick Start (Docker Compose)

```bash
git clone https://github.com/simhaonline/predictatrade.git
cd predictatrade/xauusd
cp realtime/.env.example realtime/.env
# Edit realtime/.env with required API keys
# Secrets (JWT_SECRET, POSTGRES_PASSWORD, DATABASE_URL, BACKTEST_DB_URL, GF_SECURITY_ADMIN_PASSWORD)
# live in infra/env/.env (gitignored) — fill those in too.
docker compose --env-file infra/env/.env up -d
curl http://localhost:13081/health
```

> Secrets are no longer in `docker-compose.yml`. **Always pass `--env-file infra/env/.env`** to every
> `docker compose` command. See `docs/reports/REMEDIATION_REPORT_2026-08-28.md` (SEC-1).

### Required Environment Variables

| Variable | Required | Description |
|----------|:--------:|-------------|
| DATABASE_URL | YES | PostgreSQL connection string |
| VALKEY_ADDR | YES | Valkey address (default: 127.0.0.1:6379) |
| TWELVEDATA_API_KEY | YES | XAUUSD candles + DXY |
| FMP_API_KEY | YES | COT + macro data |
| JWT_SECRET | YES | Signing key (min 32 chars) |
| PROVIDER_MODE | YES | agent=MT5, simulated=dev |

### Optional Features

| Variable | Default | Controls |
|----------|:-------:|----------|
| PTB_ENABLED | true | PTB intelligence layer |
| COT_ENABLED | false | COT data provider |
| DXY_ENABLED | false | DXY provider |
| SENTIMENT_ENABLED | false | Ollama sentiment analysis |
| RL_MODE | disabled | RL strategy optimizer |

### Service Ports

| Service | Port |
|---------|:----:|
| Realtime Engine | 13081 |
| Control Plane | 13080 |
| Frontend | 13082 |
| Status Page | 13083 |
| Live Terminal | 13090 |
| PostgreSQL | 5432 |
| Valkey | 6379 |
| Prometheus | 9090 |
| Grafana | 3001 |
| ntfy Alerts | 8091 |
| Nginx | 80/443 |

### Build Commands

```bash
make go-build       # Go realtime engine
make control-build  # NestJS control plane
make frontend-build # Next.js frontend
make test           # All 3 test suites
make lint           # All linters
```

### Production Checklist
- [ ] Supply production API keys (TwelveData, FMP, Stripe, NOWPayments)
- [ ] Set strong JWT_SECRET (32+ characters)
- [ ] Configure CORS_ORIGINS to production domains
- [ ] Set up SSL certificates
- [ ] Enable firewall (ports 80/443 only public)
- [ ] Configure database backups
- [ ] Set up Prometheus/Grafana alerts
- [ ] Compile MQL4/MT5 EAs on Windows
- [ ] Test broker connectivity in dry-run mode
