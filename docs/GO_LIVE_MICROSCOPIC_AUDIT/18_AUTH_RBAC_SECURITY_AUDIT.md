# 18 — Auth / RBAC / Security Audit

## Reproduced live tests (no credentials used)

| Test | Result |
|---|---|
| `GET /api/v1/admin/overview` anonymous | **401** ✅ |
| `GET /api/v1/audit` anonymous | **401** ✅ |
| `GET /api/v1/users` anonymous | **401** ✅ |
| `psql -h 152.53.67.111 -U pat_admin` w/ committed password | **LOGIN OK, superuser** ❌ P0 |
| Valkey WAN `PING` | **+PONG**, no auth ❌ P0 |
| WS `/ws/v1/agent?agentId=AUDIT-PROBE` no token | **101 + CONNECTED ack** ❌ P0 |
| `GET /api/v1/signals` anonymous (Go) | **200 full payloads incl. paid-grade fields** ❌ P1 |
| `/api/v1/admin/regime-diagnostics`, `/debug/pprof/*`, `/metrics`, `/api/v1/signals/resume?deviceId=` | all unauthenticated on 13081 ❌ |

## IAM internals (code-audited)

Strong: bcrypt hashing; TOTP MFA with challenge flow; refresh rotation with family revocation on reuse (`FOR UPDATE`, jti consume); advisory-lock password reset; login event capture to `iam.login_events` + `audit.audit_events`; metadata secret-scrubbing list.

Weak: audit writes best-effort swallowed; no IP/UA captured on auth events; JWT secret has dev fallback reachable in device-auth crypto paths (`'pat_local_dev_secret_change_in_production'`) not covered by production startup guard; device credential AES key derived FROM the JWT secret (key separation violation).

## RBAC / tenancy

Roles ADMIN/SUPER_ADMIN/USER via memberships; AdminGuard server-side class-level on admin+operations controllers (401s reproduced). IDOR findings: legacy licensing `PUT /licensing/devices/:id/heartbeat` and `POST /licensing/devices/:id/revoke` lack ownership checks; `/signals/resume` leaks any device's undelivered signals given a deviceId. Guest-preview lock IS server-enforced (signed cookie) — but page content itself is delivered and CSS-hidden (see 24).

## Network exposure (host: ufw INACTIVE)

Published on 0.0.0.0: 5432 Postgres(superuser), 6379 Valkey(noauth), 13080 control, **13081 realtime (agent WS + pprof)**, 13082 frontend, 13083 status, 3001 Grafana(admin/pass committed), 8091 ntfy(public read/write topic). Only 80/443 should be public.

## CORS/TLS

Nginx TLS valid for *.predictatrade.com (Let's Encrypt mounted); HTTP→HTTPS redirect verified. Go REST sets no CORS (nginx-mediated); WS Origin allowlist exists but agent hub accepts ALL origins (`CheckOrigin→true`). Next.js lacks CSP/security headers config.

## Verdict

SECURITY = NOT VERIFIED / FAILING. Three remotely-exploitable P0s (DB superuser, cache write, agent channel) plus entitlement bypass at signal REST layer satisfy §100 NO-GO conditions ("production secrets exposed", "unauthorized access", "subscription bypass exposing paid signals").
