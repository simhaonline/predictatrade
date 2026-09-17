# MIGRATION_STEPS — New VPS bring-up (320 GB, PAT + Hermes)

> Target: Hetzner hel1 bucket `pat-backup` (creds in old host `infra/env/.env` as
> `MIGRATION_S3_*`). Old host = this netcup box; stays warm ≥7 days after cutover.
> Everything below is copy-paste ready. Run PHASES 0–2 on the NEW host; nothing
> touches production until PHASE 4 (freeze + DNS).

## Legend

- `[old]` = run on THIS host (ops.simhaonline.com)
- `[new]` = run on the NEW VPS as root
- Secrets never appear in these blocks — they come from env files you recreate.

---

## PHASE 0 — New VPS base setup `[new]`

```bash
# 0.1 system + docker (Ubuntu 22.04/24.04 or Debian 12)
apt update && apt -y upgrade
apt -y install curl git ca-certificates gnupg ufw rsync
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] \
  https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo $VERSION_CODENAME) stable" \
  > /etc/apt/sources.list.d/docker.list
apt update && apt -y install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
docker --version && docker compose version

# 0.2 firewall (open only what nginx needs)
ufw allow 22/tcp && ufw allow 80/tcp && ufw allow 443/tcp && ufw --force enable

# 0.3 hermes user (agent runs as hermes, uid will differ from old host — fine)
useradd -m -s /bin/bash hermes || true
mkdir -p /srv/predictatrade/xauusd/infra/env
chown -R hermes:hermes /srv/predictatrade
```

## PHASE 1 — Pull code + secrets `[new]`

```bash
# 1.1 clone repo (GitHub; fallback: bundle below)
git clone https://github.com/simhaonline/predictatrade.git /srv/predictatrade/xauusd
cd /srv/predictatrade/xauusd

# 1.2 if GitHub unavailable, pull the bundle from Hetzner instead:
export H_AK=VQYU8PAA7853PG8CGY4U H_SK=0g7pkgY8ZgJrfZPQYeVjPUcKUfFA0dqutBidI3A3
docker run --rm -v /tmp/mig:/mig -e AWS_ACCESS_KEY_ID=$H_AK -e AWS_SECRET_ACCESS_KEY=$H_SK \
  -e AWS_DEFAULT_REGION=fsn1 --entrypoint /bin/sh amazon/aws-cli:latest -c '
  aws s3 cp s3://pat-backup/predictatrade/code/code_bundle_20260916_123318_UTC.bundle /mig/ \
    --endpoint-url https://hel1.your-objectstorage.com'
git clone /tmp/mig/code_bundle_20260916_123318_UTC.bundle /srv/predictatrade/xauusd
cd /srv/predictatrade/xauusd && git fetch origin && git reset --hard origin/main

# 1.3 SECRETS: copy the env files from the OLD host (they never touch S3).
#      From THIS host run (fill NEW_IP; key must be authorized on new host):
#   [old]  rsync -av /srv/predictatrade/xauusd/infra/env/ root@NEW_IP:/srv/predictatrade/xauusd/infra/env/
#   [old]  rsync -av /srv/predictatrade/xauusd/nginx/ root@NEW:/srv/predictatrade/xauusd/nginx/   # TLS certs live in a volume — see PHASE 3
#      Then on [new] fix ownership:
chown -R hermes:hermes /srv/predictatrade/xauusd/infra/env

# 1.4 validate compose before anything else
docker compose --env-file infra/env/.env config -q && echo COMPOSE-OK
```

## PHASE 2 — Database restore (physical base + WAL replay) `[new]`

```bash
cd /srv/predictatrade/xauusd
# 2.1 pull migration artifacts from Hetzner (~3.3 GB)
mkdir -p /tmp/mig && cd /tmp/mig
docker run --rm -v /tmp/mig:/mig -e AWS_ACCESS_KEY_ID=$H_AK -e AWS_SECRET_ACCESS_KEY=$H_SK \
  -e AWS_DEFAULT_REGION=fsn1 --entrypoint /bin/sh amazon/aws-cli:latest -c '
  aws s3 cp s3://pat-backup/predictatrade/db/clean_base_20260916 /mig/clean_base --recursive \
    --endpoint-url https://hel1.your-objectstorage.com
  aws s3 cp s3://pat-backup/predictatrade/wal /mig/wal_bridge --recursive \
    --endpoint-url https://hel1.your-objectstorage.com'
cd /srv/predictatrade/xauusd

# 2.2 start postgres alone
docker compose --env-file infra/env/.env up -d postgres
sleep 30 && docker exec pat-postgres pg_isready -U pat_admin

# 2.3 stop the empty cluster, replace PGDATA with the physical backup
docker compose --env-file infra/env/.env stop postgres
docker volume rm xauusd_pat-pgdata
docker volume create xauusd_pat-pgdata
# untar base INTO the volume (as postgres uid 1000)
docker run --rm -v xauusd_pat-pgdata:/pgdata -v /tmp/mig/clean_base:/cb:ro alpine sh -c '
  tar -xzf /cb/base.tar.gz -C /var/lib/postgresql/data
  mkdir -p /var/lib/postgresql/data/pg_wal
  tar -xzf /cb/pg_wal.tar.gz -C /var/lib/postgresql/data/pg_wal
  chmod 700 /var/lib/postgresql/data
  chown -R 1000:1000 /var/lib/postgresql/data'

# 2.4 restore_command pointing at the bridge dir + start
docker compose --env-file infra/env/.env up -d postgres
sleep 15
docker exec pat-postgres psql -U pat_admin -d postgres -c \
  "ALTER SYSTEM SET restore_command = 'cp /var/lib/postgresql/wal_bridge/%f %p';"
docker exec pat-postgres psql -U pat_admin -d postgres -c "SELECT pg_reload_conf();"
# create recovery.signal (container restart picks it up and replays to end of WAL)
docker compose --env-file infra/env/.env stop postgres
docker run --rm -v xauusd_pat-pgdata:/pgdata alpine sh -c 'touch /pgdata/recovery.signal; chown 1000:1000 /pgdata/recovery.signal'
docker compose --env-file infra/env/.env up -d postgres
sleep 30
docker exec pat-postgres psql -U pat_admin -d postgres -c \
  "ALTER SYSTEM SET restore_command = ''; SELECT pg_reload_conf();"
# 2.5 PROOF
docker exec pat-postgres psql -U pat_admin -d predictatrade -Atc \
  "SELECT count(*) FROM pg_tables WHERE schemaname NOT IN ('pg_catalog','information_schema','_timescaledb_internal','_timescaledb_catalog');"
#   expect ≈ 235 tables; market.ticks ≈ 30.9M rows
docker exec pat-postgres psql -U pat_admin -d predictatrade -Atc "SELECT count(*) FROM market.ticks;"
```

> If the volume name differs (`xauusd_pat-pgdata` vs other), check:
> `docker volume ls | grep pgdata`.

## PHASE 3 — Full stack build + Hermes `[new]`

```bash
cd /srv/predictatrade/xauusd
# 3.1 full stack (first build 20–40 min)
docker compose --env-file infra/env/.env up -d --build
docker compose --env-file infra/env/.env ps          # all healthy?

# 3.2 verify BOTH control replicas (stale control-b = incomplete deploy)
docker exec pat-control   wget -qO- http://127.0.0.1:13080/api/v1/health; echo
docker exec pat-control-b wget -qO- http://127.0.0.1:13080/api/v1/health; echo
docker exec pat-nginx wget -qO- http://pat-realtime:13081/health; echo

# 3.3 state volumes (grafana/prometheus/ntfy/ssl-certs) — tar-piped from OLD host
#      [old] for v in pat-prometheus pat-grafana pat-ntfy pat-mail-spool ssl-certs live-dashboard; do
#        docker run --rm -v xauusd_$v:/from:ro alpine tar -C /from -c . | \
#        ssh root@NEW 'docker run --rm -i -v xauusd_$v:/to alpine sh -c "tar -C /to -x"'
#      done
# 3.4 Hermes agent
curl -fsSL https://hermes-agent.nousresearch.com/install.sh | bash
#      [old] tar home and rsync:
#   [old] tar -C /var/lib/hermes-agent -czf /tmp/hermes-home.tgz .hermes \
#     --exclude '.hermes/cache' --exclude '.hermes/image_cache' --exclude '.hermes/images' \
#     --exclude '.hermes/audio_cache' --exclude '.hermes/logs' --exclude '.hermes/config.yaml.bak.*'
#   [old] rsync -av /tmp/hermes-home.tgz root@NEW:/tmp/
#      [new] mkdir -p /var/lib/hermes-agent && tar -C /var/lib/hermes-agent -xzf /tmp/hermes-home.tgz
#      [new] hermes doctor && hermes chat -q 'reply OK'
# 3.5 root crons
bash scripts/setup_crons.sh && crontab -l | tail -8
```

## PHASE 4 — Cutover (30–60 min window) — tell me when you are ready

```bash
# [old] 4.1 freeze writers + FINAL dump + FINAL wal upload
cd /srv/predictatrade/xauusd
docker compose --env-file infra/env/.env stop realtime control control-b
./scripts/dr-kit.sh backup && ./scripts/dr-kit.sh offhost
# [old] 4.2 upload the delta: newest dump + wal segments newer than 5C0000005B
#   → to Hetzner pat-backup (I will run this when you say go — it is automated here)

# [you/CF] 4.3 DNS: change origin IP for live/api/platform/downloads/status.predictatrade.com
#   → new VPS IP. MT5 EAs need nothing (they use public domains).

# [new] 4.4 final wal delta download + start stack (commands I will give live)
# [old] 4.5 leave old stack STOPPED (warm rollback ≥7 days)
```

## PHASE 5 — Verification (I run these once you flip DNS)

- restore-test PASS, migrate-status 109/109
- `docker compose ps` all healthy (control AND control-b!)
- Public: live/api/platform/downloads/status 200s
- EA edge-poll flowing (no 401 loops), mail-relay DKIM test
- First new backup lands in `pat-backup` (then we retarget `.env` BACKUP_S3_* → Hetzner and wipe old R2)