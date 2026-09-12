# Server Migration Runbook — Predict-A-Trade VPS → New VPS

> **Status:** VERIFIED against live host 2026-09-12 (netcup, `152.53.67.111`).
> Fresh migration-grade snapshot confirmed on Cloudflare R2 the same day.
> Owner: Mehul Kumar Bhatt. Companion docs: [BACKUP_RESTORE](BACKUP_RESTORE.md), [DR_PLAN](DR_PLAN.md), [HOST_DEPLOYMENT](HOST_DEPLOYMENT.md).

## 0. What is being migrated (verified inventory)

| Asset | Size | Location | Migration vehicle |
|---|---|---|---|
| Git repo (all history) | 83 MB bundle | `/srv/predictatrade/xauusd` @ `2e629f8`, clean tree | R2 `predictatrade/code/code_bundle_*.bundle` (or GitHub) |
| Working tree (nginx sites, compose overrides, untracked files; **no secrets**) | 149 MB tar.gz | same | R2 `predictatrade/code/code_tree_*.tar.gz` |
| Migration manifest (secret file PATHS, image/volume list, restore steps) | 4 KB | same | R2 `predictatrade/code/code_manifest_*.txt` |
| Database `predictatrade` (logical, 109/109 migrations applied) | 6.9 GB → 1.14 GB dump | `xauusd_pat-pgdata` volume | R2 `predictatrade/db/backup_20260912_091501_UTC.dump` + `.sha256` |
| WAL archive (continuous PITR) | 236.7 GB on disk ⚠️ | `xauusd_pat-pgdata/wal_archive` → R2 `predictatrade/wal/` | R2 WAL objects (only needed for PITR-to-timestamp) |
| Prometheus / Grafana / ntfy / mail-spool / live-dashboard / ssl-certs volumes | 118M / 113M / 120K / 52K / 2.1M / 404K | `xauusd_*` docker volumes | `docker run alpine tar` pipe (§5 step 5) |
| Docker images (Go 1.25 engine, Next.js 16.3, NestJS control ×2, …) | rebuilt on new host | — | `docker compose up -d --build` (§5 step 9) |
| **Hermes Agent** (skills 53 MB, `state.db` 433 MB, config, memories, sessions) | 2.2 GB home (→ ~500 MB tarred) | `/var/lib/hermes-agent/.hermes` (v0.20.2, single user) | rsync/tar (§6) |
| Cron jobs (root: health 5m, DB 03:13, codebase 03:23, restore-test 03:43, migrate-status Mon 04:17) | — | `root` crontab | `scripts/setup_crons.sh` on new host (§5 step 10) |

Not migrated (do NOT copy): `/srv/predictatrade/backups/wiped_pgdata_snapshot` (legacy, 535 MB), `pat-engine_*` volumes (dead compose project), other tenants on this host (`simhaonline` 8 services, `aiops-dashboard`, `license-server`, `nityaksa`), `~/.hermes/config.yaml.bak.*` (25 stale backups), stale repo-root `prompt.md` (already-executed audit brief).

## 1. Fresh snapshot — ALREADY DONE 2026-09-12

Ran on the live host (11:30 UTC):

```bash
BACKUP_DIR=/tmp/migration-snapshot ./scripts/dr-kit.sh codebase   # bundle+tree+manifest → R2 predictatrade/code/
./scripts/dr-kit.sh offhost                                        # latest dump → R2 predictatrade/db/ (sidecar had lagged 6h)
```

Verified in R2 (`s3://predictatrade-backups`, account `da104d0b…`):
- `code/code_bundle_20260912_113050_UTC.bundle` (83 MB) + `.sha256`
- `code/code_tree_20260912_113050_UTC.tar.gz` (149 MB) + `.sha256`
- `code/code_manifest_20260912_113050_UTC.txt` + `.sha256`
- `db/backup_20260912_091501_UTC.dump` (1.14 GB, sha256 `50be244f…`) + `.sha256`

`dr-kit.sh verify-s3` = PASS. Re-run both commands immediately before the cutover delta (§7 step 2).

## 2. New-server prerequisites

- Docker Engine + compose v2. Disk **≥ 500 GB** (244 GB pgdata as-is, or ~15 GB if you skip WAL-archive copy and rely on logical restore + fresh WAL), headroom for images.
- Open inbound: 22, 80, 443. IPv4 + IPv6 as available.
- Decide DNS handling BEFORE cutover (§7 step 4).

## 3. Pull artifacts on the new host

**Path A — via R2 (default; both hosts need only the R2 keys):**

```bash
set -a; . /path/to/infra/env/.env; set +a   # BACKUP_S3_*
docker run --rm -v /tmp/mig:/mig -e AWS_ACCESS_KEY_ID="$BACKUP_S3_ACCESS_KEY" \
  -e AWS_SECRET_ACCESS_KEY="$BACKUP_S3_SECRET_KEY" -e AWS_DEFAULT_REGION=auto \
  --entrypoint /bin/sh amazon/aws-cli:latest -c '
  for f in code_bundle code_tree code_manifest; do
    aws s3 cp "s3://'$BACKUP_S3_BUCKET'/predictatrade/code/${f}_" - /mig/ --recursive --exclude "*" --include "${f}_*" --endpoint-url '$BACKUP_S3_ENDPOINT' || break
  done
  aws s3 cp "s3://'$BACKUP_S3_BUCKET'/predictatrade/db/backup_LATEST.dump" /mig/ --endpoint-url '$BACKUP_S3_ENDPOINT''
sha256sum -c /tmp/mig/*.sha256   # MANDATORY before any restore
```

(Pull the exact filenames listed in §1 — `--include` per prefix; do not glob blindly.)

**Path B — direct rsync (faster, needs both hosts up):**

```bash
rsync -aAX --info=progress2 old:/srv/predictatrade /srv/ \
  --exclude node_modules --exclude .next --exclude dist
rsync -a old:/var/lib/hermes-agent/.hermes /var/lib/hermes-agent/.hermes \
  --exclude cache --exclude image_cache --exclude images \
  --exclude audio_cache --exclude logs --exclude 'config.yaml.bak.*'
```

Path B also copies secrets — §4 lists what must exist either way.

## 4. Secrets to recreate (values NEVER leave any host)

From the codebase manifest (paths only — values never leave a host):

| File | Contents |
|---|---|
| `infra/env/.env` | BACKUP_S3_* (R2 keys), POSTGRES_PASSWORD, DATABASE_URL, JWT_SECRET, WEBHOOK_SECRET, etc. |
| `infra/env/canonical.env` | canonical runtime vars |
| `infra/env/control.env` | NOWPayments (use `usdterc20`, bare `usdt` rejected), SMTP_HOST **must be** `pat-mail-relay`, JWT_SECRET |
| `infra/env/realtime.env` | engine secrets (JWT_SECRET shared with control — rotate atomically, never one side) |
| `infra/env/frontend.env` | public URLs |
| `infra/env/status.env` | status page |
| root `.env` (496 B) | compose-level vars |
| `infra/env/ADMIN_CREDENTIALS.local.md` | admin login references (keep out of git) |
| DKIM keys for `pat-mail-relay` | inside `xauusd_pat-mail-spool` volume / mail-relay config — copy volume or re-export DNS TXT |

Transport options: `rsync`/`scp` directly old→new host, or your password manager. Never commit, never print. **Rotate** (per `docs/operations/SECRET_ROTATION.md`) after cutover if the old host is destroyed: JWT_SECRET is shared by `control.env` AND `realtime.env` — rotate both atomically.

## 5. Restore steps on the NEW host

```bash
# 1. Code
git clone https://github.com/simhaonline/predictatrade.git /srv/predictatrade/xauusd
cd /srv/predictatrade/xauusd
#    (or offline: git clone code_bundle_*.bundle && git fetch origin)

# 2. Recreate every secret file from §4 (exact paths in code_manifest_*.txt)

# 3. Validate compose before anything else
docker compose --env-file infra/env/.env config -q     # must exit 0

# 4. Postgres FIRST (matching version — never a host psql client older than 17)
docker compose up -d pat-postgres && sleep 30
docker exec -i pat-postgres psql -U pat_admin -d postgres \
  -c "CREATE DATABASE predictatrade;"
cat /tmp/mig/backup_20260912_091501_UTC.dump | \
  docker exec -i pat-postgres pg_restore -U pat_admin -d predictatrade \
  --no-owner --no-privileges
#    TimescaleDB compressed-chunk warnings ("hypertable id N") are EXPECTED and safe.
#    Need PITR-to-timestamp instead? restore base/ + replay predictatrade/wal/ (see BACKUP_RESTORE §PITR).

# 5. Small state volumes (prometheus, grafana, ntfy, mail-spool, live-dashboard, ssl-certs)
#    run AFTER their services exist; tar-pipe per volume:
#    docker run --rm -v xauusd_<v>:/from:ro alpine tar -C /from -c . | \
#    docker exec -i pat-<svc> sh -c 'tar -C /to -x'   # or via intermediate tar over ssh

# 6. Restore proof
BACKUP_DIR=/tmp DB_CONTAINER=pat-postgres DB_USER=pat_admin bash scripts/backup/restore_test.sh
#    Expected PASS: Schemas 25 | Tables ~12,975 | market.ticks ~30.6M | extensions incl. timescaledb

# 7. Migration parity
./scripts/dr-kit.sh migrate-status        # expect 109/109 applied, 0 pending

# 8. Point env at new host
#    grep -rn "152.53.67.111\|127.0.0.1:1308\|old-ip" infra/env/ docker-compose.yml nginx/
#    Replace host-IP references; keep BACKUP_S3_* (same R2 bucket from new host).

# 9. Full stack (first build ~20–40 min: Go engine, Next.js, NestJS ×2)
docker compose up -d --build

# 10. Crons (as ROOT — /var/backups/predictatrade is root-owned)
sudo bash scripts/setup_crons.sh && sudo crontab -l | tail -8
```

## 6. Hermes Agent on the new host

```bash
# OLD host — tar the agent home (state.db WAL will checkpoint on untar; stop gateway first if running)
tar -C /var/lib/hermes-agent -czf /tmp/hermes-home.tgz .hermes \
  --exclude '.hermes/cache' --exclude '.hermes/image_cache' --exclude '.hermes/images' \
  --exclude '.hermes/audio_cache' --exclude '.hermes/logs' --exclude '.hermes/config.yaml.bak.*'

# NEW host
curl -fsSL https://hermes-agent.nousresearch.com/install.sh | bash
mkdir -p /var/lib/hermes-agent && tar -C /var/lib/hermes-agent -xzf hermes-home.tgz
hermes doctor && hermes chat -q 'reply OK'
```

Project context files (`.hermes.md`, `AGENTS.md`) ship with the repo and are auto-loaded from the working directory. Re-point delivery-channel webhooks (Discord/Telegram/WhatsApp/ntfy) at the new origin after DNS flip — see `predictatrade-platform-ops` skill / `docs/operations`. The one Hermes cron job (`p1-soak-driver`, one-shot, expired 31-Aug) is stale — delete with `cronjob action='list'` + remove after migration.

## 7. Cutover order (downtime ≈ dump-restore + DNS ≈ 30–60 min)

1. **Pre-stage** (new host fully built per §5, stack **STOPPED** — never run both stacks live simultaneously: Discord bot, Telegram poller, EA edge-poll and COT fetches would double-fire).
2. **Freeze writers on OLD:** `docker compose stop realtime control control-b` (leave nginx up for static until DNS flip). Cut a FINAL dump: `./scripts/dr-kit.sh backup && ./scripts/dr-kit.sh offhost`.
3. Restore the delta dump on NEW (§5 step 4 with the final dump filename), rsync any changed secret files.
4. **DNS flip:**
   - Cloudflare-proxied hosts → change origin IP to new server: `live/api/platform/downloads/status.predictatrade.com` (proxied = effective in seconds).
   - `predictatrade.com` apex A `159.195.54.152` → either point at new host directly or, if keeping Plesk TLS edge, edit that vhost's `proxy_pass` to the new origin via Plesk UI (no SSH/API access there).
   - MT5/MT4 EAs need nothing — they call the public domains.
5. Start full stack on NEW; verify **every instance independently** (see §8).
6. Watch first EA edge-polls (`docs/runbooks/signal-delivery-verification.md`); if 401 loops appear, load skill `pat-ea-auth-diagnostics`.
7. Keep OLD host **warm ≥ 7 days** as instant rollback (flip DNS back); do NOT delete its pgdata (incl. 236 GB wal_archive) until the new host's WAL chain is proven (one `dr-kit.sh restore-test` + one fresh WAL object landing in R2).

## 8. Verification checklist (run all, on the new host)

- [ ] `BACKUP_DIR=/tmp DB_CONTAINER=pat-postgres bash scripts/backup/restore_test.sh` → PASS line
- [ ] `./scripts/dr-kit.sh migrate-status` → 109 applied / 0 pending
- [ ] `docker compose ps` → 15–16 services healthy (control, control-b, frontend, realtime, postgres, valkey, nginx, watchdog…)
- [ ] `docker images --format '{{.Repository}} {{.ID}} {{.CreatedAt}}' | grep xauusd-control` → control AND control-b same recent build time (stale control-b = incomplete deploy)
- [ ] `docker exec pat-control wget -qO- http://127.0.0.1:13080/health` AND same on `pat-control-b`
- [ ] Public: `https://live.predictatrade.com` 200 + signals feed; `https://api.predictatrade.com/health`; admin login with `infra/env/ADMIN_CREDENTIALS.local.md`
- [ ] EA path: device edge-poll 200s in logs, no 401 loops
- [ ] `pat-mail-relay` DKIM-signed send test (SMTP_HOST must be pat-mail-relay, never Plesk SMTP)
- [ ] Grafana dashboards + Prometheus targets up; ntfy test notification
- [ ] Backup loop: new dump appears in R2 `predictatrade/db/` within ~60 s of cron; `./scripts/dr-kit.sh verify-s3` PASS
- [ ] `hermes doctor` PASS; skills list intact; repo `.hermes.md` loads in session

## 9. Known pitfalls (each one bit in production)

- **control-b stale image:** `docker compose build control control-b frontend` then `docker rm -f pat-control-b && docker compose up -d control-b` — `--force-recreate` alone can keep the old image.
- **compose `env_file` + `$$VAR`:** inside containers reference sidecar secrets as `$$VAR`; `${VAR}` expands from the host shell and silently becomes empty.
- **Host psql 16 vs server 17:** always run psql/pg_restore through the `pat-postgres` container; stream dumps via stdin.
- **`/var/backups/predictatrade` is root-owned** (hermes uid 995): run backup scripts as root or with `BACKUP_DIR=/tmp`.
- **aws-cli container:** ENTRYPOINT is `aws`; use `--entrypoint /bin/sh … -c`, and pass R2 vars with **double quotes** (single quotes block expansion).
- **R2 tuning:** serialized uploads (`max_concurrent_requests=1`, adaptive retry) — bursty multipart fails; omit `--sse` when encryption=none.
- **`wal_archive` has no retention prune** (236.7 GB and growing on the old host). Decide BEFORE migration: either migrate as-is (disk budget!) or schedule a WAL-retention prune first. The physical chain is the only PITR path — never prune back past the newest base backup.
- **Do not blindly copy `xauusd_pat-pgdata`** (244 GB). Logical restore (§5) is the default; physical copy only when PITR-to-timestamp is required.
- **Never run both stacks live in parallel** — delivery bots and EA polling would double-fire.
- Repo-root `prompt.md` is a stale, already-executed audit brief — ignore (do not treat as current instructions).

## 10. Total transfer size

R2 path: ~3.5 GB (dump 1.14 GB + bundle/tree 232 MB + optional WAL). Direct rsync: ~20 GB including state volumes. Hermes home ~500 MB tarred.