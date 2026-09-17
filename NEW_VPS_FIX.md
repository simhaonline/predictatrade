# FIX for new VPS — postgres "no response" (initdb permission loop)

## Root cause (from your error.log)

```
initdb: error: could not change permissions of directory "/var/lib/postgresql/data": Operation not permitted
fixing permissions on existing directory /var/lib/postgresql/data ... using nss-wrapper
Waiting for permissions on /var/lib/postgresql/data (0:0 -> 1000:1000)........
```

The pgdata volume root was left owned by `root:root`. Postgres (uid 1000) cannot
chown it, initdb aborts, container exits → `pg_isready` = no response.
(The same block also implies the volume was empty when postgres started — the
tar extraction from PHASE 2 hadn't landed in the volume yet.)

## THE FIX — run this whole block on the NEW VPS as root

```bash
cd /srv/predictatrade/xauusd

# 1. stop postgres (it is crash-looping)
docker compose --env-file infra/env/.env stop postgres

# 2. verify the migration artifacts are actually downloaded (must show 4 files, ~1.6GB)
ls -la /tmp/mig/clean_base/
#    If MISSING, re-download from Hetzner first:
#    docker run --rm -v /tmp/mig:/mig \
#      -e AWS_ACCESS_KEY_ID=VQYU8PAA7853PG8CGY4U \
#      -e AWS_SECRET_ACCESS_KEY=0g7pkgY8ZgJrfZPQYeVjPUcKUfFA0dqutBidI3A3 \
#      -e AWS_DEFAULT_REGION=fsn1 --entrypoint /bin/sh amazon/aws-cli:latest -c '
#      aws s3 cp s3://pat-backup/predictatrade/db/clean_base_20260916 /mig/clean_base \
#        --recursive --endpoint-url https://hel1.your-objectstorage.com'

# 3. wipe the (empty/initdb-crashed) volume and extract INTO it with correct ownership
docker volume rm xauusd_pat-pgdata
docker volume create xauusd_pat-pgdata

# KEY FIX: chown the mount ROOT itself to 1000:1000 BEFORE extracting,
# and extract as uid 1000 so every file lands already owned.
docker run --rm -v xauusd_pat-pgdata:/pgdata -v /tmp/mig/clean_base:/cb:ro alpine sh -c '
  set -e
  chown 1000:1000 /pgdata
  chmod 700 /pgdata
  cd /pgdata
  tar -xzf /cb/base.tar.gz
  mkdir -p pg_wal
  tar -xzf /cb/pg_wal.tar.gz -C pg_wal
  chown -R 1000:1000 /pgdata
  echo "--- verify ---"
  ls /pgdata | head -12
  echo "PG_VERSION: $(cat /pgdata/PG_VERSION 2>/dev/null || echo MISSING)"
  echo "pg_wal segments: $(ls /pgdata/pg_wal | grep -cE "^[0-9A-F]{24}$")"'

# 4. start postgres (the entrypoint will now see a valid cluster, run recovery
#    on the streamed WAL from the backup, and come up healthy)
docker compose --env-file infra/env/.env up -d postgres
sleep 45
docker logs pat-postgres --tail 15 2>&1
docker exec pat-postgres pg_isready -U pat_admin -d predictatrade
```

## Expected result

- `docker logs` shows `PostgreSQL init process complete; ready for start up` or
  `database system is ready to accept connections` (NOT initdb)
- `pg_isready` → `/var/run/postgresql:5432 - accepting connections`

## Then continue PHASE 2 from step 2.4 (WAL bridge replay)

The base backup already contains the WAL through segment `5C00000047` (12:32 UTC
Sep 16). The bridge segments (`5C00000048…5B`) are in `/tmp/mig/wal_bridge`.
Because the base ends mid-recovery (backup_label), postgres on first start will
replay what is inside `pg_wal/` and stop at its end — to catch up to NOW you must
give it the bridge and a recovery target at end of WAL:

```bash
# 5. copy the bridge INTO the volume (readable by postgres)
docker run --rm -v xauusd_pat-pgdata:/pgdata -v /tmp/mig/wal_bridge:/wb:ro alpine sh -c '
  mkdir -p /pgdata/pg_wal_bridge && cp /wb/* /pgdata/pg_wal_bridge/ && chown -R 1000:1000 /pgdata/pg_wal_bridge'

# 6. tell postgres where to pull WAL, then restart so it replays to end
docker exec pat-postgres psql -U pat_admin -d postgres -c \
  "ALTER SYSTEM SET restore_command = 'cp /pgdata/pg_wal_bridge/%f %p 2>/dev/null || cp /var/lib/postgresql/wal_archive/%f %p 2>/dev/null || exit 1';" \
  && docker exec pat-postgres psql -U pat_admin -d postgres -c "SELECT pg_reload_conf();"

# restart so recovery re-enters replay and consumes the bridge
docker compose --env-file infra/env/.env restart postgres
sleep 45
docker logs pat-postgres --tail 20 | grep -E 'redo|recovery|consistent|ready'
docker exec pat-postgres psql -U pat_admin -d predictatrade -Atc "SELECT count(*) FROM pg_tables WHERE schemaname NOT IN ('pg_catalog','information_schema','_timescaledb_internal','_timescaledb_catalog');"
#   expect ≈ 235 ; then market.ticks count:
docker exec pat-postgres psql -U pat_admin -d predictatrade -Atc "SELECT count(*) FROM market.ticks;"

# 7. PROOF + clear restore path when satisfied
docker exec pat-postgres psql -U pat_admin -d postgres -c "ALTER SYSTEM SET restore_command = ''; SELECT pg_reload_conf();"
docker exec pat-postgres psql -U pat_admin -d predictatrade -Atc "SELECT now();"
```

> Note: `wal_archive` (bind mount from `./infra/wal_archive`) is empty on the new
> host — that is fine. The `restore_command` exit-1 on a missing segment is what
> tells postgres "replay target reached".

Send me the output of step 3 (`docker logs`) and the two PROOF counts and I'll
confirm green or give the next fix.