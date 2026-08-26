# Predict-A-Trade Deployment & Setup Guide
## v1.16.0 — 26 August 2026

### Quick Start (Docker Compose)

```bash
# 1. Clone and enter
git clone https://github.com/simhaonline/predictatrade.git
cd predictatrade/xauusd

# 2. Configure environment
cp realtime/.env.example realtime/.env
# Edit realtime/.env:
#   - Set TWELVEDATA_API_KEY=your_key (required for XAUUSD candles)
#   - Set FMP_API_KEY=your_key (required for COT/DXY)
#   - Set PROVIDER_MODE=agent (for MT5) or simulated (for dev)

cp infra/env/realtime.env.example infra/env/realtime.env
# Fill in database URL, JWT secret, optional API keys

# 3. Start everything
docker compose up -d

# 4. Run migrations
docker compose exec postgres psql -U pat_admin -d predictatrade   -f /docker-entrypoint-initdb.d/001_create_schemas_and_roles.sql
# (run all 001-030 migrations in order)

# 5. Verify
curl http://localhost:13081/health
curl http://localhost:13080/health
curl http://localhost:13082
```

### Service Ports
| Service | Port | URL |
|---------|:----:|-----|
| Realtime Engine | 13081 | http://localhost:13081 |
| Control Plane | 13080 | http://localhost:13080 |
| Frontend | 13082 | http://localhost:13082 |
| Status Page | 13083 | http://localhost:13083 |
| PostgreSQL | 5432 | postgres://localhost:5432/predictatrade |
| Valkey | 6379 | valkey://localhost:6379 |
| Prometheus | 9090 | http://localhost:9090 |
| Grafana | 3001 | http://localhost:3001 |
| ntfy | 8091 | http://localhost:8091 |
| Live Terminal | 13090 | http://localhost:13090 |

### Prerequisites
- Docker 24+ and Docker Compose v2
- Go 1.26 (for local builds)
- Node.js 18+ (for NestJS + Next.js)
- Python 3.10+ (for research plane)
- PostgreSQL 17 with TimescaleDB extension
- Valkey 8.0

### Environment Variables (Realtime Engine)
| Variable | Default | Required | Description |
|----------|---------|:--------:|-------------|
| DATABASE_URL | — | YES | PostgreSQL connection string |
| VALKEY_ADDR | 127.0.0.1:6379 | YES | Valkey address |
| HTTP_HOST | 127.0.0.1 | YES | Bind host |
| HTTP_PORT | 13081 | YES | HTTP port |
| PROVIDER_MODE | agent | YES | agent=MT5, simulated=dev |
| TWELVEDATA_API_KEY | — | YES | TwelveData API key |
| FMP_API_KEY | — | YES | FMP API key for COT |
| JWT_SECRET | — | YES | JWT signing secret |
| PTB_ENABLED | true | NO | PTB master switch |
| COT_ENABLED | false | NO | COT data provider |
| DXY_ENABLED | false | NO | DXY provider |
| SENTIMENT_ENABLED | false | NO | Ollama sentiment |
| RL_MODE | disabled | NO | RL optimizer |

### Build Commands
```bash
# All services
make build

# Individual
make go-build        # Realtime engine
make control-build   # NestJS control plane
make frontend-build  # Next.js frontend
make test            # All tests
make lint            # All linters
```

### Production Checklist
- [ ] Supply production API keys (TwelveData, FMP, NOWPayments, Stripe)
- [ ] Set strong JWT_SECRET (min 32 chars)
- [ ] Configure CORS_ORIGINS to production domains
- [ ] Set up SSL certificates (Let's Encrypt)
- [ ] Enable firewall (only 80/443 public)
- [ ] Configure database backups
- [ ] Set up monitoring alerts (Prometheus/Grafana)
- [ ] Compile MQL EAs on Windows
- [ ] Test broker connectivity in dry-run mode
- [ ] Review and approve candle retention policy