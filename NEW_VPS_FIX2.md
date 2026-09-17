# NEW VPS FIX 2 — postgres won't start: shared memory too big for small host

## Root cause (from error.log)

```
FATAL: could not map anonymous shared memory: Cannot allocate memory
request = 17,346,256,896 bytes (16.15 GB)
HINT: reduce shared_buffers or max_connections
```

GOOD NEWS: the restore itself is perfect — `PG_VERSION: 17`, `backup_label` present,
`"Database directory appears to contain a database; Skipping initialization"`.
The cluster boots its startup sequence; it dies only when sizing shared memory.
The restored `postgresql.conf` carries the OLD host's tuning
(`shared_buffers = 16073MB` — sized for the 64 GB netcup box). Your new VPS has
less RAM, so the 17.3 GB mapping fails.

## First — tell me your RAM (for final tuning)

```bash
free -m
```

## THE FIX — right-size memory on the restored cluster (no data risk)

The settings live in files inside the volume; you can override them before
starting postgres — no single-user mode needed. Append overrides to
`postgresql.auto.conf` directly in the volume:

```bash
cd /srv/predictatrade/xauusd
docker compose --env-file infra/env/.env stop postgres

# 1. figure out host RAM in MB
RAM_MB=$(free -m | awk '/^Mem:/{print $2}')
echo "host RAM: ${RAM_MB} MB"

# 2. size shared_buffers ≈ 25% of RAM (min 512MB, cap 16GB)
SB_MB=$(( RAM_MB / 4 )); [ $SB_MB -lt 512 ] && SB_MB=512
if [ $SB_MB -gt 16384 ]; then SB_MB=16384; fi
echo "setting shared_buffers = ${SB_MB}MB"

# 3. append overrides to auto.conf inside the volume (postgres reads this last)
docker run --rm -v xauusd_pat-pgdata:/pgdata -e SB_MB=$SB_MB -e RAM_MB=$RAM_MB alpine sh -c '
  cat >> /pgdata/postgresql.auto.conf <<EOF

# —— NEW HOST memory right-sizing (2026-09-17, RAM=${RAM_MB}MB) ——
shared_buffers = '"${SB_MB}"'MB
max_connections = 60
work_mem = 16MB
maintenance_work_mem = 256MB
effective_cache_size = '"$(( RAM_MB * 3 / 4 ))"'MB
wal_buffers = 16MB
max_wal_size = 2GB
min_wal_size = 256MB
EOF
  tail -9 /pgdata/postgresql.auto.conf'

# 4. ALSO shrink the compose mem_limit for postgres so docker does not OOM-kill it.
#    Edit docker-compose.yml: postgres → mem_limit 16g → e.g. for 8GB host: 6g
#    (keep it ~75% of host RAM). Quick sed for an 8GB host:
#    sed -i "s/mem_limit: 16g/mem_limit: 6g/" docker-compose.yml
#    (adapt to your RAM; check with `grep -n "mem_limit: 16g" docker-compose.yml`)

# 5. start
docker compose --env-file infra/env/.env up -d postgres
sleep 45
docker logs pat-postgres --tail 15
docker exec pat-postgres pg_isready -U pat_admin -d predictatrade
```

If `pg_isready` → **accepting connections**, continue with the WAL-bridge replay
(steps 5–7 in NEW_VPS_FIX.md) and the PROOF counts.

> Note for the new host's compose: with less RAM the other services' mem_limits
> also matter (realtime 3g, control ~1g, frontend...). If the host is small,
> `docker compose config` may need several mem_limit reductions — tell me
> `free -m` and I'll give you exact numbers for every service.
>
> Longer-term (after cutover): the right fix is a permanent memory profile for
> the small host — I'll add `infra/env/memory-overrides.env` + compose overrides
> so the tuning is explicit, not invisible.

## Sizing cheatsheet (shared_buffers by host RAM)

| Host RAM | shared_buffers | max_connections | compose mem_limit (postgres) |
|---|---|---|---|
| 4 GB | 1GB | 40 | 3g |
| 8 GB | 2GB | 60 | 6g |
| 16 GB | 4GB | 80 | 12g |
| 32 GB | 8GB | 100 | 24g |