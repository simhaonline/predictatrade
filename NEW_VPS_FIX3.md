# NEW VPS FIX 3 — memory profile for the 16 GB host (verified: 15,603 MB total)

## Your memory (from error.log)

```
Mem: 15603 MB total · 12761 MB available · no swap
```

Why postgres died: restored `shared_buffers = 16073MB` → 17.3 GB shared-memory
map > 15.6 GB host. Impossible by construction — must right-size.

## Numbers for THIS host (16 GB tier)

| Setting | Value |
|---|---|
| shared_buffers | **3900MB** |
| effective_cache_size | 11700MB |
| max_connections | 80 |
| work_mem | 16MB |
| maintenance_work_mem | 256MB |
| wal_buffers | 16MB |
| max_wal_size / min_wal_size | 2GB / 256MB |
| postgres compose mem_limit | **12g** |

## Step 1 — memory overrides into the restored cluster

```bash
cd /srv/predictatrade/xauusd
docker compose --env-file infra/env/.env stop postgres

docker run --rm -v xauusd_pat-pgdata:/pgdata alpine sh -c '
  cat >> /pgdata/postgresql.auto.conf <<EOF

# —— 16GB HOST profile (2026-09-17) ——
shared_buffers = 3900MB
max_connections = 80
work_mem = 16MB
maintenance_work_mem = 256MB
effective_cache_size = 11700MB
wal_buffers = 16MB
max_wal_size = 2GB
min_wal_size = 256MB
EOF
  tail -9 /pgdata/postgresql.auto.conf'
```

## Step 2 — right-size ALL compose mem_limits (sum must fit ~12.5 GB)

Current sum = 35.5 GB — impossible on 15.6 GB. Replace the whole set:

| Service | Old | New (16GB host) |
|---|---|---|
| pat-postgres | 16g | **12g** |
| pat-realtime | 3g | **2g** |
| pat-valkey | 2g | **1g** |
| pat-backtest | 2g | **1g** |
| pat-prometheus | 2g | **1g** |
| pat-control | 1.5g | **1g** |
| pat-control-b | 1.5g | **1g** |
| pat-frontend | 1.5g | **1g** |
| pat-grafana | 1g | **768m** |
| pat-backup-sync | 1g | **512m** |
| live-terminal / nginx / discord-bot / mail-relay | 512–768m | keep |
| ntfy / watchdog | 256m | keep |

New sum ≈ **11.7 GB** + overhead → fits 15.6 GB with headroom for buff/cache.

One-shot sed to apply on the NEW host's compose (safe — all patterns unique
enough within their service blocks, run once):

```bash
cd /srv/predictatrade/xauusd
sed -i \
  -e "s/mem_limit: 16g/mem_limit: 12g/" \
  -e "s/mem_limit: 3g/mem_limit: 2g/" \
  -e "s/mem_limit: 2g/mem_limit: 1g/" \
  -e "s/mem_limit: 1.5g/mem_limit: 1g/" \
  -e "s/mem_limit: 1g/mem_limit: 768m/" \
  -e "s/mem_limit: 768m/mem_limit: 640m/" \
  docker-compose.yml
# ⚠️ run config check after sed:
docker compose --env-file infra/env/.env config -q && echo COMPOSE-OK
```

> CAREFUL: the seds cascade (16g→12g then 3g→2g then 2g→1g...). Run them in the
> order given (16g first, then 3g, then 2g, then 1.5g, then 1g, then 768m).
> Postgres ends at 12g because it was 16g→(skipped)→ wait: 16g matches "16g" sed
> → 12g. The later "1g→768m" sed does NOT re-hit 12g (pattern is "mem_limit: 1g"
> exact). Verify after with: `grep -n mem_limit docker-compose.yml`.

## Step 3 — start + verify

```bash
docker compose --env-file infra/env/.env up -d postgres
sleep 45
docker logs pat-postgres --tail 15
docker exec pat-postgres pg_isready -U pat_admin -d predictatrade
```

## Step 4 — WAL bridge replay (data up to freeze time)

```bash
# bridge into volume
docker run --rm -v xauusd_pat-pgdata:/pgdata -v /tmp/mig/wal_bridge:/wb:ro alpine sh -c '
  mkdir -p /pgdata/pg_wal_bridge && cp /wb/* /pgdata/pg_wal_bridge/ && chown -R 1000:1000 /pgdata/pg_wal_bridge'

# restore_command → bridge, restart, replay to end
docker exec pat-postgres psql -U pat_admin -d postgres -c \
  "ALTER SYSTEM SET restore_command = 'cp /pgdata/pg_wal_bridge/%f %p 2>/dev/null || exit 1';"
docker compose --env-file infra/env/.env restart postgres
sleep 45

# PROOF
docker exec pat-postgres psql -U pat_admin -d predictatrade -Atc \
  "SELECT count(*) FROM pg_tables WHERE schemaname NOT IN ('pg_catalog','information_schema','_timescaledb_internal','_timescaledb_catalog');"
# expect ≈ 235
docker exec pat-postgres psql -U pat_admin -d predictatrade -Atc "SELECT count(*) FROM market.ticks;"
# expect ≈ 30.9M

# clear restore_command
docker exec pat-postgres psql -U pat_admin -d postgres -c \
  "ALTER SYSTEM SET restore_command = ''; SELECT pg_reload_conf();"
```

Then PHASE 3: `docker compose --env-file infra/env/.env up -d --build` (20–40 min).