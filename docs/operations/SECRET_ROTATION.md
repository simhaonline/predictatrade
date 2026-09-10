# Runbook — Secret Rotation (JWT_SECRET, DEVICE_SECRET, DB credentials)

**Trigger context:** The live `JWT_SECRET` was published verbatim in git history twice: once hardcoded in `docker-compose.yml` (pre-2026-08-30) and once re-leaked inside `docs/reports/MACROSCOPIC_AUDIT_REPORT.md` (redacted 2026-09-10, commit 7385241 era). **Treat every secret that ever appeared in a tracked file as COMPROMISED** — deleting it now does not remove it from git history.

## Non-negotiable rules

1. Secrets live ONLY in `infra/env/*.env` (gitignored). Never in compose, never in docs, never in code.
2. Rotation is atomic across BOTH consumers: `control.env` AND `realtime.env` (JWT auth is shared between NestJS and Go). Restart both after rotation.
3. Never print the new secret in chat, docs, or commit messages.

## JWT_SECRET rotation (current value is COMPROMISED — do this next maintenance window)

JWT_SECRET authenticates: dashboard users (NestJS), admin actions on the engine (`/api/v1/admin/*` on :13081), and live-terminal preview tokens.

```bash
# 1. Generate (do NOT display): 44-char url-safe random
NEW=$(openssl rand -base64 33 | tr '+/' '-_')
# 2. Write to both env files atomically
sed -i "s|^JWT_SECRET=.*|JWT_SECRET=${NEW}|" infra/env/control.env
sed -i "s|^JWT_SECRET=.*|JWT_SECRET=${NEW}|" infra/env/realtime.env
grep -E "^JWT_SECRET=" infra/env/realtime.env | sed -E 's/=(.{4}).*/=\1***/'   # verify prefix only
# 3. Rolling restart (dashboard users re-login; refresh tokens are invalidated by design)
docker compose --env-file infra/env/.env up -d control control-b realtime
# 4. Verify: login via UI, one admin API call, one edge-poll device heartbeat
```

**Impact:** all active sessions invalidate at rotation (expected); devices re-authenticate via their stored refresh/device credentials (HMAC device auth does NOT use JWT_SECRET — see DEVICE_SECRET below). **Rollback:** keep the old value in a temporary local file for 24h; if a subsystem fails to verify, re-point that service and restart.

## DEVICE_SECRET (new — Phase-1 P0 prerequisite)

`device-auth.service.ts` currently derives AES-256-GCM + HMAC pepper from JWT_SECRET. Splitting it requires code change (see audit P2-7): introduce `DEVICE_SECRET` env, re-encrypt `licensing` device secrets in one transaction, then rotate JWT_SECRET independently. Do both in the same window.

## DB credential rotation

Postgres credential rotation requires `ALTER ROLE` + simultaneous DATABASE_URL updates in `.env` + all service restarts + backtest BACKTEST_DB_URL + backup tooling creds. Schedule with the PITR rehearsal; verify pat-backup-sync after.

## Verification checklist (after ANY rotation)

- [ ] `docker compose --env-file infra/env/.env up -d control control-b realtime` all healthy
- [ ] Admin login works; admin API call succeeds
- [ ] Edge-poll heartbeat 200 (device HMAC path unaffected)
- [ ] No `[TG]`/mail/notification failures in logs
- [ ] `git log -S <old-secret>` returns only historical redacted docs