# Predict-A-Trade Docker Compose Reference
## v1.16.0 — 26 August 2026

### Service Architecture

```
                    ┌─────────┐
                    │  Nginx  │  :80, :443
                    └────┬────┘
           ┌─────────────┼─────────────┐
           ▼             ▼             ▼
    ┌──────────┐  ┌──────────┐  ┌──────────┐
    │ Frontend │  │ Control  │  │ Realtime │
    │  :13082  │  │  :13080  │  │  :13081  │
    └──────────┘  └──────────┘  └────┬─────┘
                                     │
              ┌──────────────────────┼──────────────────────┐
              ▼                      ▼                      ▼
       ┌──────────┐          ┌──────────┐          ┌──────────┐
       │ Postgres │          │  Valkey  │          │  Ollama  │
       │  :5432   │          │  :6379   │          │  :11434  │
       └──────────┘          └──────────┘          └──────────┘
```

### All Services

| Service | Container | Image | Port | Volumes |
|---------|-----------|-------|:----:|---------|
| postgres | pat-postgres | timescale/timescaledb-ha:pg17 | 5432 | pat-pgdata |
| valkey | pat-valkey | valkey/valkey:8.0 | 6379 | pat-valkey |
| realtime | pat-realtime | built (./realtime) | 13081 | — |
| control | pat-control | built (./control) | 13080 | — |
| frontend | pat-frontend | built (./frontend) | 13082 | — |
| nginx | pat-nginx | built (./nginx) | 80,443 | certs |
| status | pat-status | built (./status) | 13083 | — |
| live-terminal | pat-live-terminal | built (./) | 13090 | — |
| prometheus | pat-prometheus | prom/prometheus | 9090 | pat-prometheus |
| grafana | pat-grafana | grafana/grafana | 3001 | pat-grafana |
| ntfy | pat-ntfy | binwiederhier/ntfy | 8091 | pat-ntfy |

### Networks
- `pat-net`: Internal bridge network (all services)

### Volume Management
```bash
# List volumes
docker volume ls | grep pat-

# Backup PostgreSQL
docker exec pat-postgres pg_dump -U pat_admin predictatrade > backup.sql

# Wipe all data (DESTRUCTIVE)
docker compose down -v
```

### Health Checks
| Service | Check | Interval | Retries |
|---------|-------|:--------:|:-------:|
| postgres | pg_isready | 5s | 30 |
| valkey | PING | 5s | 10 |
| realtime | HTTP /health | 10s | 5 |
| control | HTTP /health | 10s | 5 |
| frontend | HTTP / | 10s | 5 |
| nginx | curl localhost:80 | 10s | 5 |

### Common Commands
```bash
# Start all
docker compose up -d

# View logs
docker compose logs -f realtime

# Restart one service
docker compose restart realtime

# Rebuild and restart
docker compose up -d --build realtime

# Stop all
docker compose down

# Check status
docker compose ps
```

### Environment Files
| File | Consumed By |
|------|-------------|
| infra/env/realtime.env | realtime |
| infra/env/control.env | control |
| infra/env/frontend.env | frontend |
| realtime/.env | realtime (build env) |

### Resource Limits
- PostgreSQL: shm_size=2gb (required for large sort/hash operations)
- All containers: default Docker limits (consider adding memory limits for production)
- No CPU pinning configured