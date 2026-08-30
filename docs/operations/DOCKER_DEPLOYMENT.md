# Docker Deployment Guide
## v1.18.0 — 30 August 2026

Step-by-step guide to deploy Predict-A-Trade XAUUSD using Docker Compose.

---

## Prerequisites

| Requirement | Minimum | Check |
|-------------|:-------:|-------|
| Docker | 24.0+ | `docker --version` |
| Docker Compose | v2+ | `docker compose version` |
| RAM | 4GB free | `free -h` |
| Disk | 20GB free | `df -h` |
| Ports | 80, 443, 5432, 6379, 13090, 13081, 13080, 13082, 13083, 8091 available | `ss -tlnp` |
| Git | any recent | `git --version` |
| API keys | TwelveData, FMP | (see Step 2) |

---

## Step 1 — Clone the Repository

```bash
git clone https://github.com/simhaonline/predictatrade.git
cd predictatrade/xauusd
```

Verify the clone:
```bash
ls docker-compose.yml       # Should exist
ls realtime/Dockerfile       # Should exist
ls database/migrations/      # 69 SQL files (numbered to 099)
```

---

## Step 2 — Configure API Keys

Copy the example environment file and fill in your keys:

```bash
cp realtime/.env.example realtime/.env
nano realtime/.env
```

Required keys (all must be set):
```ini
TWELVEDATA_API_KEY=your_twelvedata_key_here    # XAUUSD candles + DXY
FMP_API_KEY=your_fmp_key_here                  # COT data
```

Optional — broker timezone (defaults to GMT+3 for XAUUSD brokers):
```ini
# BROKER_TIMEZONE=GMT+3       # Default. Set to your broker's server timezone.
# COT_ENABLED=true             # Commitment of Traders data
# DXY_ENABLED=true            # US Dollar Index
# SENTIMENT_ENABLED=true      # Ollama AI sentiment
# RL_MODE=filter_only         # RL optimizer
```

---

## Step 3 — Configure Infrastructure Environment

> **Secrets are NOT in `docker-compose.yml`** (remediated, SEC-1). All secret values live in
> `infra/env/.env` (gitignored). Create/copy it from the provided template and fill in:
> `JWT_SECRET`, `POSTGRES_PASSWORD`, `DATABASE_URL`, `BACKTEST_DB_URL`, `GF_SECURITY_ADMIN_PASSWORD`.

```bash
ls infra/env/.env || cp infra/env/.env.example infra/env/.env
nano infra/env/.env          # set the 5 secrets above
# Per-service config (API keys, CORS) stays in realtime/.env / control/.env / frontend/.env
```

For production:
1. Set a strong `JWT_SECRET` in `infra/env/.env` (32+ chars). **If you rotate it, re-issue Windows Agent tokens** (the agent link will drop until re-registered).
2. Set a strong `POSTGRES_PASSWORD` in `infra/env/.env`.
3. Update `CORS_ORIGINS` in `infra/env/control.env`.

> **Every `docker compose` command MUST use `--env-file infra/env/.env`** or containers start with blank secrets.

---

## Step 4 — Start All Services

```bash
docker compose --env-file infra/env/.env up -d
```

Expected output:
```
[+] Running 11/11
 ✔ Network predictatrade-xauusd_pat-net  Created
 ✔ Volume predictatrade-xauusd_pat-pgdata Created
 ✔ Container pat-postgres  Started
 ✔ Container pat-valkey    Started
 ✔ Container pat-realtime  Started
 ✔ Container pat-control   Started
 ✔ Container pat-frontend  Started
 ✔ Container pat-live-terminal Started
 ✔ Container pat-nginx     Started
 ...
```

Check service health (wait ~30 seconds for all to start):
```bash
docker compose ps
```

Expected: All services show `healthy` or `running`.

---

## Step 5 — Apply Database Migrations

> **Migrations are NOT auto-applied.** The Postgres container does **not** run
> `docker-entrypoint-initdb.d` to apply these files, and you must **never** rewrite or renumber
> already-applied migration history. Apply forward migrations explicitly with the runner:

```bash
# Canonical — applies only pending migrations, idempotent, never rewrites history
./scripts/migrate.sh up

# Or, inside the stack (use the migrate runner, do not hand-loop raw .sql):
docker compose --env-file infra/env/.env exec realtime ./scripts/migrate.sh up

# Verify applied migration count (do NOT re-run raw .sql files manually)
docker exec pat-postgres psql -U pat_admin -d predictatrade \
  -c "SELECT COUNT(*) FROM audit.migration_history;"
```

> **Why not `for f in database/migrations/*.sql; do psql < "$f"; done`?** Hand-looping raw files
> bypasses the migration ledger, can double-apply or partially apply, and risks rewriting history.
> Always use `./scripts/migrate.sh up` (or the compose-exec equivalent). Current migration set:
> **69 files** in `database/migrations/` (numbered to 099), all applied to the live DB.

---

## Step 6 — Verify Everything

### Health checks
```bash
# Go realtime engine
curl http://localhost:13081/health
# Expected: {"status":"ok"}

# NestJS control plane
curl http://localhost:13080/api/v1/health
# Expected: {"status":"ok"}

# Frontend
curl -I http://localhost:13082
# Expected: HTTP/1.1 200 OK or 307

# Nginx reverse proxy
curl http://localhost/
# Expected: HTML response

# Live Terminal
curl http://localhost:13090/health
# Expected: {"status":"ok"}

# PostgreSQL
docker exec pat-postgres pg_isready -U pat_admin -d predictatrade
# Expected: accepting connections

# Valkey
docker exec pat-valkey valkey-cli ping
# Expected: PONG
```

### Verify engines running
```bash
curl http://localhost:13081/api/v1/engine/status 2>/dev/null | python3 -m json.tool
```

### Check logs for errors
```bash
docker compose logs realtime | tail -20
docker compose logs control | tail -20
```

---

## Step 7 — Nginx Configuration

The Nginx container auto-configures via mounted volumes:
- `nginx/nginx.conf` — main config
- `nginx/sites-available/` — virtual host configs
- `nginx/snippets/` — reusable config fragments

For production:
```bash
# Edit the site config
nano nginx/sites-available/live.predictatrade.com.conf

# SSL certificates are expected at /etc/letsencrypt/
# Mounted in docker-compose.yml as:
#   - ssl-certs:/etc/letsencrypt:ro

# After editing, reload nginx:
docker compose restart nginx
```

> **⚠ Rate-limit changes require a full container RESTART, not just a reload.**
> `limit_req_zone` shared-memory zones are created at nginx **startup**; a `reload`
> (`nginx -s reload`) keeps the old zones. If you change `nginx/snippets/rate-limit.conf`
> (e.g. auth/api/general/ws rates) you MUST restart the container for the new limits
> to take effect:
> ```bash
> docker compose --env-file infra/env/.env restart nginx
> # verify: docker exec pat-nginx nginx -T 2>/dev/null | grep limit_req_zone
> ```
> See `docs/reports/REMEDIATION_REPORT_2026-08-28.md` §7.6.

---

## Step 8 — Grafana & Monitoring

Grafana is accessible at `http://localhost:3001`:
- Username: `admin`
- Password: `pat_local_dev_only` (change in production)

Pre-configured dashboards are at `infra/grafana/dashboards/`.

Prometheus scrapes metrics from `pat-realtime:13081/metrics`.

---

## Daily Operations

### View logs
```bash
docker compose logs -f realtime          # Follow realtime logs
docker compose logs --tail=100 control   # Last 100 lines
docker compose logs --since 30m          # Last 30 minutes
```

### Restart a service
```bash
docker compose restart realtime
docker compose restart frontend
```

### Rebuild after code changes
```bash
git pull
docker compose up -d --build realtime
```

### Stop everything
```bash
docker compose down            # Stop containers, keep volumes
docker compose down -v         # Stop AND delete volumes (DESTRUCTIVE)
```

### Database backup
```bash
docker exec pat-postgres pg_dump -U pat_admin predictatrade \
  > backup_$(date +%Y%m%d_%H%M%S).sql

# Compress
gzip backup_*.sql
```

### Database restore
```bash
gunzip -c backup_20260826_120000.sql.gz | \
  docker exec -i pat-postgres psql -U pat_admin -d predictatrade
```

---

## Troubleshooting

### Service won't start
```bash
docker compose ps                    # Check status
docker compose logs realtime         # Check logs
docker inspect pat-realtime          # Inspect container
```

### Database connection refused
```bash
# Check postgres is running
docker compose ps postgres
# Check port is accessible
docker exec pat-realtime nc -zv postgres 5432
```

### Stale binary after code changes
After `git pull`, the running container may have an older binary:
```bash
docker compose up -d --build realtime   # Force rebuild
```

### Out of memory / shm errors
PostgreSQL needs large shared memory. docker-compose.yml already sets:
```yaml
shm_size: '2gb'
```
If you still see errors, increase it.

### Permission denied on mounted volumes
```bash
# Fix ownership for nginx volumes
sudo mkdir -p /var/www/pat-live /etc/letsencrypt /var/log/nginx
sudo chown -R $(whoami):$(whoami) /var/www/pat-live
```

---

## Production Hardening Checklist

- [ ] Change `POSTGRES_PASSWORD` from dev default
- [ ] Set strong `JWT_SECRET` (32+ chars, random)
- [ ] Change Grafana admin password
- [ ] Set production API keys (TwelveData, FMP, Stripe, NOWPayments)
- [ ] Configure real SSL certificates (Let's Encrypt / certbot)
- [ ] Set `CORS_ORIGINS` to production domains
- [ ] Enable firewall: only ports 80 and 443 public
- [ ] Set `BROKER_TIMEZONE` to match your broker's server time
- [ ] Set up automated database backups (cron job)
- [ ] Configure Prometheus alerting rules
- [ ] Test backup restore procedure
- [ ] Review resource limits per container
- [ ] Set up log rotation for nginx logs

---

## Service Architecture (Docker)

```
                    Nginx :80,:443
                         │
        ┌────────────────┼────────────────┐
        ▼                ▼                ▼
   Frontend          Control          Realtime
   :13082            :13080           :13081
                                          │
                     ┌────────────────────┼────────────┐
                     ▼                    ▼            ▼
              PostgreSQL              Valkey        Ollama
              :5432                   :6379        (host)

   Monitoring: Prometheus :9090 → Grafana :3001
   Alerts: ntfy :8091
   Status: :13083
   Live Terminal: :13090
```
