# Anonymous Live Preview — live.predictatrade.com

Server-enforced 5-minute anonymous preview → registration wall → existing
account/entitlement lifecycle. Scope: Go realtime engine + static terminal
ONLY (control plane / Next.js untouched).

## Architecture

```
Anonymous visitor → GET /api/v1/live-preview/status
                    → trial created (HttpOnly cookie pat_live_trial)
                    → {status, server_time, trial_started_at,
                       trial_expires_at, remaining_seconds}
5 minutes (server-authoritative) → all protected endpoints return
    403 {"code":"REGISTRATION_REQUIRED","reason":"LIVE_PREVIEW_EXPIRED"}
WebSocket: TRIAL_EXPIRED event → close 4003 (mid-connection sweep, 15s)
Registration wall (frontend) → platform.predictatrade.com register/login
Registered users → existing subscription entitlements apply unchanged
```

## Access precedence (centralized guard — gateway/livepreview.go)

1. Authenticated (control-plane JWT, shared `JWT_SECRET`) → ACTIVE
   subscription in `billing.subscriptions` → ALLOW (60s/user cache).
2. Anonymous ACTIVE trial cookie → ALLOW.
3. Otherwise → 403 REGISTRATION_REQUIRED. Fail-closed on DB errors (503
   `LIVE_ACCESS_UNAVAILABLE`).

## Protected paths (guarded)

`/api/v1/{signals,signals/resume,market/*,candles,strategies,agents/status,
news,price/history,engines/status,system-health,cross-market/*}` and
`/ws`, `/ws/v1` (browser).

Exempt (never guarded): `/health`, `/ready`, `/metrics`, `/debug/*`,
`/ws/v1/agent`, `/ws/agent` (Windows Agent upstream — the data source, own
auth), `/api/v1/live-preview/*`, static assets.

## Trial lifecycle

- Cookie `pat_live_trial`: random 32-byte token, HttpOnly, Secure,
  SameSite=Lax. DB stores HMAC-SHA256(secret, token) only.
- `trial_started_at` immutable — refresh/new tab/multi-socket NEVER reset.
- Statuses: ACTIVE → EXPIRED | CONVERTED | BLOCKED | REVOKED.
- Only `/api/v1/live-preview/status` mints trials (data endpoints never do).
- In-process cache for hot checks; DB on create/expire; `last_seen` write
  throttled to 1/min per trial.

## Anti-abuse (privacy-conscious, §24–26)

- Repeat signal = HMAC(ip) AND HMAC(UA) both matching within 24h.
- 0–1 prior matches → allow; ≥2 → 403 REPEAT_TRIAL_BLOCKED.
- IP alone can NEVER block (shared NAT/office protection).
- No raw IPs, no fingerprints, no tokens in DB. Limitation: anonymous
  trials cannot be perfectly single-use per human across browsers,
  devices, VPNs, or cookie deletion — by design.

## Funnel analytics

Events in `live_preview.trial_events`; real metrics in
`live_preview.funnel_stats` (view). Admin endpoint:
`GET /api/v1/admin/live-preview/stats` (ADMIN-role JWT only).

## Config (infra/env/realtime.env)

```
LIVE_PREVIEW_ENABLED=true
LIVE_PREVIEW_DURATION_SECONDS=300
LIVE_PREVIEW_COOKIE_NAME=pat_live_trial
LIVE_PREVIEW_HMAC_SECRET=<random 64-hex>       # rotate = invalidate trials
LIVE_PREVIEW_ABUSE_DETECTION_ENABLED=true
```

## Rollback

Set `LIVE_PREVIEW_ENABLED=false` → `docker compose up -d realtime`.
Feature off = pre-funnel public behavior. No DB rollback needed.

## Verified (live domain, 2026-08-25)

- Fresh visitor: trial + 300s countdown + full live data (chart/engines)
- Refresh & multi-tab: same trial, no timer reset (287s ≡ 287s)
- Tampered cookie: 403 INVALID_TRIAL_TOKEN, no new trial minted
- DB-forced expiry: 403 LIVE_PREVIEW_EXPIRED; status → EXPIRED/NATURAL
- Real-time lock without reload: countdown 00:00 → wall (30s status sync)
- DevTools wall removal: REST still 403 REGISTRATION_REQUIRED (primary
  acceptance — server-side enforcement)
- Authenticated FREE/STANDARD user (JWT): 200, overrides expired trial
- Admin stats: real funnel counts; non-admin JWT: 403
- Windows Agent path exempt: agents stay connected
- Go tests: 12/12 livepreview + full suite 33/33 packages

## Known limitations

- Cross-browser/incognito repeat trials are mitigated (coarse signals),
  not guaranteed preventable — documented per §54.
- Seamless same-domain SSO after platform registration is not wired (would
  require control-plane changes, out of declared scope); the wall links to
  the existing platform register/login. Authenticated API access on the
  live domain works via `Authorization: Bearer <jwt>` / WS `?token=`.
