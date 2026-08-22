# 26 — Infrastructure / Deployment Audit

## Runtime (docker compose, verified `ps`)

10/10 services up: postgres(timescale pg17,healthy), valkey(healthy), realtime(healthy), control(healthy), frontend(healthy), status, nginx, prometheus, grafana, ntfy. Restart policies present; healthchecks on core services.

## Configuration defects

| ID | Sev | Finding |
|---|---|---|
| 26-1 | P0 | Host firewall inactive; DB/cache/engine ports published on 0.0.0.0 (18). |
| 26-2 | P1 | Committed credentials in tracked `docker-compose.yml` (Postgres pat_admin/pat_local_dev_only, Grafana admin) while containers run production data. |
| 26-3 | P2 | Two nginx config trees exist (`infra/nginx` vs repo-root `nginx`); container mounts the root one; `infra/nginx/sites-available/*` is stale/unmounted — duplicate source of truth. |
| 26-4 | P2 | MANIFEST.md documents **systemd** units + host-binary deploy that contradict AGENTS.md Docker-first rule and actual runtime (docker). Docs drift. |
| 26-5 | P2 | Frontend container runs as root; `npm install --legacy-peer-deps` in build (non-reproducible installs). |
| 26-6 | P3 | `predictatrade-live-patched.html`, `error.txt`, `jwt_secret.txt`, `database_url.txt` at repo root — working-tree hygiene; the two secret files are chmod-600 + gitignored (OK) but should move to a secrets store. |

## Deployment process

Build: compose builds per service; Makefile targets match MANIFEST. Migrations: `scripts/migrate.sh up` with bookkeeping table (04 ✅). Rollback: documented pattern absent for app images; DB forward-only policy respected. No CI/CD pipeline runs deploys (.github/workflows present for tests only — UNVERIFIED content depth).

## Reproducibility gaps

No image version pinning strategy beyond base tags (grafana/prometheus `latest`); env files outside compose are the de-facto config store with plaintext secrets (17).
