# Hetzner VPS Deployment — Predict-A-Trade XAUUSD

**Status:** Production runbook (v1.0, 2026-09-11)
**Audience:** operator provisioning or migrating the live host
**Stack:** Hetzner VPS (Ubuntu 22.04/24.04) + Docker Compose, Cloudflare (DNS/Proxy + R2), off-host backups to Cloudflare R2.

> This guide is Hetzner-specific. For the generic Docker steps see
> [DOCKER_DEPLOYMENT.md](DOCKER_DEPLOYMENT.md). For backup/restore mechanics see
> [BACKUP_RESTORE.md](BACKUP_RESTORE.md). This file adds the host-provisioning,
> firewall, env-file wiring, and server-migration steps that the generic docs omit.

---

## 1. Why Hetzner (architecture fit)

The data tier is the product: PostgreSQL 17 + TimescaleDB (hypertables over
30M+ ticks) + pgvector + pg_cron (stale-data pruning job 1049). That requires a
**real, persistent, low-churn disk** and a **stable process** that holds an
in-memory market state and a persistent WebSocket to the MT4/MT5 Master Node.

| Requirement | Hetzner VPS | Cloudflare Containers / Fargate / Cloud Run |
|---|---|---|
| Persistent NVMe disk | ✅ native | ❌ ephemeral, killed on host move |
| Stable instance (no 10-min sleep) | ✅ | ❌ platform may stop at any time |
| Local Postgres + TimescaleDB + pgvector | ✅ | ❌ only D1 (SQLite, 50GB/acct, no Timescale/pgvector) |
| Cheap/generous egress | ✅ | ⚠️ egress metered |
| Broker WS continuity | ✅ stable | ❌ reconnection churn |

**Verdict:** Hetzner VPS runs the stateful core; Cloudflare fronts all
subdomains (DNS + proxy) and R2 stores off-host backups. This is the supported
topology — do NOT move the data tier onto Cloudflare Containers.

---

## 2. Recommended instance sizing

Current production host (reference): **16 vCPU / 62 GB RAM / 2 TB NVMe**.
Compose memory limits sum to ~33 GB, so this has comfortable headroom.

| Tier | Minimum | Notes |
|---|---|---|
| Dev / single-user | CPX21 (3 vCPU / 8 GB) | Postgres is memory-hungry; 8 GB is the floor |
| Production | CPX41 (8 vCPU / 32 GB) or CCX33 (8 vCPU / 32 GB) | run DB + engine comfortably, room for Grafana/Prometheus |
| High-traffic | CX/CCX with ≥16 GB RAM dedicated to Postgres | give `pat-postgres` its own volume |

**Disk:** ≥80 GB for small; production uses ~750 GB of 2 TB. Keep `/var/backups`
on the same NVMe (local dumps) and rely on R2 for off-host. Add a second volume
if you want DB and backups physically separated.

**Region:** no Hetzner Middle-East region. Use **Falkenstein (FSN1)** or
**Nuremberg (NBG1)** — both are close to the EU/UK broker (Xelans, GMT+3), which
matters more for the Master-Node feed than admin-UI latency from Dubai.

**Calibrate with real numbers:** before committing to a tier, run
`scripts/capacity/measure_capacity.py` against the live stack (see
[CAPACITY_PLAN.md](CAPACITY_PLAN.md)) — it measures the actual baseline and
projects to your target subscriber count using editable per-user assumptions,
with explicit dedicated-server trigger thresholds.

**Image:** Ubuntu 22.04 LTS or 24.04 LTS (64-bit).

---

## 3. Provision the host

### 3.1 Create + SSH
- Hetzner Cloud Console → new Project → "Add Server" → Ubuntu 24.04, your tier.
- Add your **SSH key** (password login disabled after step 3.3).
- Note the **public IPv4** and **IPv6** (Cloudflare will proxy; you may keep
  origin IP hidden behind Cloudflare, but the host still needs a public IP).

### 3.2 Base packages + Docker
```bash
sudo apt-get update && sudo apt-get -y upgrade
sudo apt-get -y install ufw git curl ca-certificates gnupg lsb-release \
  ncdu htop tmux fail2ban
# Docker (official, not snap)
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | \
  sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
  https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list >/dev/null
sudo apt-get update
sudo apt-get -y install docker-ce docker-ce-cli containerd.io docker-compose-plugin
sudo usermod -aG docker $USER   # log out/in afterwards
docker compose version          # expect v2+
```

### 3.3 Firewall (UFW) — only what's needed
The compose file binds Postgres (5432) and Valkey (6379) to **127.0.0.1 only**
(do NOT expose these). Publish only:
- `22/tcp` SSH (restrict to your IP if possible)
- `80/tcp`, `443/tcp` (nginx, Cloudflare proxy → origin)
- `587/tcp`, `465/tcp` (mail-relay — only if you send email directly; otherwise drop)
- `13081/tcp` etc. are bound inside the docker network; **not** published to host
  except where compose `ports:` shows a host mapping. Re-check `docker ps` after
  deploy and block anything unexpected.

```bash
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow from 173.245.48.0/20 to any port 80,443  # optional: Cloudflare ranges only
sudo ufw enable
sudo systemctl enable --now fail2ban
```
> Cloudflare edge IPs: restrict 80/443 to Cloudflare's published ranges if you
> want to force all traffic through the proxy (recommended). See
> `nginx/snippets/cloudflare-realip.conf` for the same-range real_ip trust.

---

## 4. Clone + prepare environment files

Secrets live in `infra/env/*.env` (gitignored — **never commit them**). The
compose services load them via `env_file`:

| File | Loaded by | Template |
|---|---|---|
| `infra/env/realtime.env` | live-terminal, realtime, ntfy | `infra/env/realtime.env.example` (tracked) |
| `infra/env/control.env` | control, control-b | **create from example / copy** |
| `infra/env/frontend.env` | frontend | **create from example / copy** |
| `infra/env/status.env` | status | **create from example / copy** |
| `infra/env/.env` | backup-sync (R2 creds + shared secrets) | `infra/env/.env.example` (tracked) |

```bash
git clone https://github.com/simhaonline/predictatrade.git
cd predictatrade/xauusd
cp infra/env/realtime.env.example infra/env/realtime.env
# control.env / frontend.env / status.env: copy realtime.env.example as a base
# and adjust per service (or use your existing host's files when migrating — see §7)
touch infra/env/control.env infra/env/frontend.env infra/env/status.env
cp infra/env/.env.example infra/env/.env
```

Edit each file. Minimum required (see `DEPLOYMENT_GUIDE.md` for the full table):
- `DATABASE_URL`, `POSTGRES_PASSWORD`, `JWT_SECRET` (≥32 chars), `VALKEY_ADDR`
- `TWELVEDATA_API_KEY`, `FMP_API_KEY` (market data)
- `PROVIDER_MODE` (`agent` for live Master-Node feed, `simulated` for dev)
- In `infra/env/.env`: the `BACKUP_S3_*` block (see §5)

> **Do NOT** run `docker compose` with a stray root `.env` — the services read
> per-service `env_file:` entries, not a project-root `.env`. The `backup-sync`
> sidecar reads `infra/env/.env` via `env_file` (single source of truth).

---

## 5. Off-host backup target — Cloudflare R2

Backups go to **three separate R2 prefixes** for clean restore / migration:

| Prefix | Contents | Produced by |
|---|---|---|
| `predictatrade/wal/` | WAL archive (continuous PITR) | `pat-backup-sync` sidecar |
| `predictatrade/db/` | logical `pg_dump` dumps + `.sha256` | `pat-backup-sync` (from `/var/backups/predictatrade`) |
| `predictatrade/code/` | git bundle + working tree (no secrets) + migration manifest | `dr-kit.sh codebase` (nightly cron) |

Set these in `infra/env/.env` (gitignored):
```bash
BACKUP_S3_ACCESS_KEY=...        # from Cloudflare R2 API tokens
BACKUP_S3_SECRET_KEY=...
BACKUP_S3_REGION=auto
BACKUP_S3_BUCKET=predictatrade-backups
BACKUP_S3_ENDPOINT=https://<accountid>.r2.cloudflarestorage.com
BACKUP_S3_PREFIX=predictatrade/wal
BACKUP_S3_DB_PREFIX=predictatrade/db
BACKUP_S3_CODE_PREFIX=predictatrade/code
```
Verify without leaking secrets:
```bash
./scripts/dr-kit.sh verify-s3     # PASS = bucket reachable, creds valid
```
On a fresh host, create the bucket once:
```bash
docker run --rm -e AWS_ACCESS_KEY_ID -e AWS_SECRET_ACCESS_KEY -e AWS_DEFAULT_REGION \
  amazon/aws-cli:latest s3 mb s3://predictatrade-backups \
  --endpoint-url https://<accountid>.r2.cloudflarestorage.com
```

---

## 6. Deploy

```bash
# from repo root
docker compose build            # first time / after code changes
docker compose up -d
docker compose ps               # all services "Up"/"healthy"
```

First-run order is handled by `depends_on` (postgres healthy → others). Expect:
`pat-postgres` (16g), `pat-realtime` (3g), `pat-control`/`control-b`,
`pat-frontend`, `pat-backtest`, `pat-valkey` (2g), `pat-nginx`, `pat-status`,
`pat-live-terminal`, `pat-prometheus`, `pat-grafana`, `pat-ntfy`,
`pat-backup-sync`, `pat-watchdog`.

Smoke test:
```bash
curl -fsS http://127.0.0.1:13081/health      # realtime engine
curl -fsS http://127.0.0.1:13081/api/v1/market/snapshot
docker logs --tail 5 pat-backup-sync          # should show WAL uploads to R2
```

### 6.1 Install cron jobs (health + backups)
Run **as root** (the backup scripts write to `/var/backups/predictatrade`, which
is root-owned):
```bash
sudo ./scripts/setup_crons.sh
```
Installs: 5-min production health check (incl. Cloudflare edge), nightly DB
backup (03:13), nightly restore self-test (03:43), nightly codebase snapshot to
R2 (03:23), weekly migration-status (Mon 04:17), weekly retraining (Sun 02:00).
Verify the schedule: `sudo crontab -l`.

---

## 7. Server migration / rebuild (the R2 escape hatch)

Because R2 holds three independent, restorable layers, moving to a new Hetzner
VPS (or any host) is mechanical:

1. **Provision** the new VPS (§3), install Docker + UFW.
2. **Code:** `git clone` the repo OR restore the latest
   `predictatrade/code/code_bundle_<ts>.bundle`:
   ```bash
   git clone https://github.com/simhaonline/predictatrade.git
   cd predictatrade/xauusd
   # or: git clone <repo> && git bundle unbundle code_bundle_<ts>.bundle
   ```
3. **Config:** recreate `infra/env/*.env` from the committed `.example`
   templates. The `code/code_manifest_<ts>.txt` in R2 lists exactly which secret
   *files* exist (paths only — values are NOT stored), so you know what to fill.
4. **Database:** restore the latest `predictatrade/db/` dump (or use the WAL
   archive for point-in-time):
   ```bash
   # pull latest dump from R2 into /var/backups/predictatrade
   docker run --rm -v /var/backups/predictatrade:/out -e AWS_* amazon/aws-cli:latest \
     s3 sync s3://predictatrade-backups/predictatrade/db/ /out/ \
     --endpoint-url https://<accountid>.r2.cloudflarestorage.com
   ./scripts/dr-kit.sh restore-test    # validates into throwaway DB, drops it
   # for real restore: load the chosen .dump into pat-postgres
   ```
5. **Deploy:** `docker compose up -d` (§6).
6. **DNS cutover:** point Cloudflare `api/docs/downloads/live/platform/status
   .predictatrade.com` A/AAAA records at the new host IP. Cloudflare proxy stays
   on; clients see no change.

No data is lost because the source of truth is in R2, independent of the box.

---

## 8. Daily operations

| Task | Command |
|---|---|
| Health (origin + Cloudflare edge) | `./scripts/verify_live_production.sh` |
| Backup DB now | `./scripts/dr-kit.sh backup` |
| Code snapshot now | `./scripts/dr-kit.sh codebase` |
| Restore self-test (safe) | `./scripts/dr-kit.sh restore-test` |
| S3 config check | `./scripts/dr-kit.sh verify-s3` |
| Migration status | `./scripts/dr-kit.sh migrate-status` |
| Full DR pass | `./scripts/dr-kit.sh all` |
| Logs | `docker compose logs -f <service>` |
| Update + rebuild | `git pull && docker compose build && docker compose up -d` |

### 8.1 Resource guardrails
- Keep `pat-postgres` memory limit (16g) below host RAM; if the host has <24 GB,
  lower it and raise `shared_buffers` proportionally in `postgres.conf`.
- Watch `/var/backups` fill rate — `ncdu /var/backups`; local dumps rotate at
  30 days (backup.sh), R2 keeps per-bucket retention (set lifecycle in R2 console).

---

## 9. Pre-flight checklist (new host)

- [ ] Hetzner VPS created (Ubuntu 24.04, sized per §2, Falkenstein/Nuremberg)
- [ ] Docker + compose-plugin installed, user in `docker` group
- [ ] UFW enabled: 22/80/443 (+ optional mail ports), 5432/6379 NOT exposed
- [ ] Repo cloned; `infra/env/*.env` created from `.example` and filled
- [ ] `infra/env/.env` has `BACKUP_S3_*` (R2) — `dr-kit.sh verify-s3` PASS
- [ ] R2 bucket `predictatrade-backups` exists with `wal/ db/ code/` prefixes
- [ ] `docker compose up -d` → all services Up/healthy
- [ ] `/health` + market snapshot respond on 13081
- [ ] `pat-backup-sync` logs show R2 uploads
- [ ] `sudo ./scripts/setup_crons.sh` installed (crontab -l confirms)
- [ ] Cloudflare DNS A/AAAA → host IP, proxy ON for all subdomains

---

## 10. Decision record

- **2026-09-11** — Chose Hetzner VPS over Cloudflare Containers. Containers lack
  persistent disk + stable instances + a TimescaleDB-class datastore; the data
  tier (Postgres/TimescaleDB/pgvector/pg_cron) requires a real host. Cloudflare
  retained for DNS/proxy + R2 off-host backups (three separate prefixes).
- See `docs/operations/BACKUP_RESTORE.md` and `DR_PLAN.md` for recovery detail.
