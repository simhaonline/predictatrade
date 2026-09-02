# Runbook — HTTP 429 on Edge-Poll (All MT Clients)

**Incident:** 2026-09-02, 12:20–14:45 UTC. Every MT4/MT5 client logged
`edge-poll failed: HTTP 429 — {"statusCode":429,"message":"ThrottlerException: Too Many Requests"}`.

## Root Cause

Two compounding defects:

1. **MQL4 EA had no poll cadence guard.** `PollFromCloud()` ran on EVERY
   tick. XAUUSD M1 tick rate = 100–290 POSTs/min per terminal (nginx
   access-log verified). MT5 has had a `PATPollMs` guard since v1.10; MT4
   was the unguarded regression.
2. **Global NestJS ThrottlerModule bucket is per-IP, and terminals share
   IPs.** `ThrottlerModule.forRoot([{ ttl: 60000, limit: 300 }])` +
   `trust proxy 1` → all terminals behind one VPS IP (e.g. Equiti VPS
   94.200.173.6 running MT4 + MT5) share ONE 300/min bucket. Aggregate
   machine traffic blew through it → `ThrottlerException` for every
   client, even well-behaved ones.

Evidence: 7,321 × 429 on `/api/v1/devices/edge-poll` between 12:20 and
14:45 UTC; bursts of ~290 req/min from a single VPS IP vs the 300/min cap.

## Fix (shipped)

| Layer | Change |
|---|---|
| NestJS | `@SkipThrottle()` on `EdgePollController` + `DeviceAuthController` — HMAC machine traffic exempt from the interactive-traffic global throttle; abuse bounded per-device by device identity + nginx `10r/s/IP` zone |
| MQL4 v1.23 | `PATPollMs` input (default 3000 ms, floor 1000 ms) cadence guard in `PollFromCloud()` — parity with MT5 |
| MQL5 v1.20 | `PATPollMs` default 1000 → 3000 ms (parity + nginx headroom) |

Commits: `b584f5d` (code) — deployed to pat-control.

## Verification

- Post-deploy: 0 × 429 across the last 300 edge-poll requests; ~53
  polls/min aggregate (old client builds still fast-polling — absorbed).
- Devices keep polling + acking (`licensing.edge_device_state`
  `polls_total` advancing, `last_poll_at` fresh).

## Client Action Required

Recompile from `frontend/public/downloads/`:
- **MT4 → v1.23** (cadence guard)
- **MT5 → v1.20** (PATPollMs default 3000)

With new builds each device polls ~20/min — comfortable under both
nginx (10 r/s/IP) and any future per-device server limits.

## If 429s Recur

1. `docker exec pat-control printenv` → confirm ThrottlerModule config
   unchanged (`ttl 60000 / limit 300` global; edge-poll exempt).
2. `grep " 429 " /var/log/nginx/api.predictatrade.com.access.log | tail` —
   nginx 429 (different body, no "ThrottlerException") = nginx zone, check
   `rate-limit.conf` zones.
3. Body contains `ThrottlerException` = NestJS throttler; check whether
   the endpoint lost its `@SkipThrottle()`.
4. Per-device poll cadence:
   `SELECT device_id, polls_total FROM licensing.edge_device_state;` —
   a single device climbing >100/min means an old unguarded EA build.