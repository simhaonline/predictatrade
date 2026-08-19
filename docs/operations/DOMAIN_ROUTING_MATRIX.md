# Predict-A-Trade v1.0.0 — Domain Routing Matrix

**Date:** 18 August 2026  
**Status:** All routing verified at runtime — real DNS + real Let's Encrypt TLS

## Domain Hosting

| Domain | Hosted On | TLS | Verified |
|--------|-----------|-----|----------|
| predictatrade.com | Plesk server (159.195.54.152) | Plesk-managed | N/A (external) |
| www.predictatrade.com | Plesk server (CNAME → predictatrade.com) | Plesk-managed | N/A (external) |
| platform.predictatrade.com | This server (152.53.67.111) | Let's Encrypt ✅ | ✅ Real DNS |
| api.predictatrade.com | This server (152.53.67.111) | Let's Encrypt ✅ | ✅ Real DNS |
| live.predictatrade.com | This server (152.53.67.111) | Let's Encrypt ✅ | ✅ Real DNS |
| status.predictatrade.com | This server (152.53.67.111) | Let's Encrypt ✅ | ✅ Real DNS |

## Domain Routing Matrix

| DOMAIN | PATH | NGINX UPSTREAM | APP OWNER | AUTH | PROTOCOL | HEALTH | TEST RESULT |
|--------|------|----------------|-----------|------|----------|--------|------------|
| api.predictatrade.com | /api/v1/health | 127.0.0.1:3000 | NestJS | Public | HTTPS | ✅ ok | PASS |
| api.predictatrade.com | /api/v1/plans | 127.0.0.1:3000 | NestJS | Public | HTTPS | ✅ 4 plans | PASS |
| api.predictatrade.com | /api/v1/auth/register | 127.0.0.1:3000 | NestJS | Public | HTTPS | ✅ JWT | PASS |
| api.predictatrade.com | /api/v1/admin/overview | 127.0.0.1:3000 | NestJS | AdminGuard | HTTPS | ✅ stats | PASS |
| api.predictatrade.com | /api/v1/admin/users | 127.0.0.1:3000 | NestJS | AdminGuard | HTTPS | ✅ paginated | PASS |
| api.predictatrade.com | /api/v1/admin/subscriptions | 127.0.0.1:3000 | NestJS | AdminGuard | HTTPS | ✅ paginated | PASS |
| api.predictatrade.com | /api/v1/admin/commissions | 127.0.0.1:3000 | NestJS | AdminGuard | HTTPS | ✅ paginated | PASS |
| api.predictatrade.com | /api/v1/admin/payouts | 127.0.0.1:3000 | NestJS | AdminGuard | HTTPS | ✅ paginated | PASS |
| api.predictatrade.com | /api/v1/admin/licenses | 127.0.0.1:3000 | NestJS | AdminGuard | HTTPS | ✅ paginated | PASS |
| api.predictatrade.com | /api/v1/admin/devices | 127.0.0.1:3000 | NestJS | AdminGuard | HTTPS | ✅ paginated | PASS |
| api.predictatrade.com | /api/v1/admin/health | 127.0.0.1:3000 | NestJS | AdminGuard | HTTPS | ✅ health | PASS |
| api.predictatrade.com | /api/v1/auth/login | 127.0.0.1:3000 | NestJS | Public | HTTPS | ✅ JWT | PASS |
| api.predictatrade.com | /api/v1/strategies | 127.0.0.1:8081 | Go RT | Public | HTTPS | ✅ 4 strategies | PASS |
| api.predictatrade.com | /api/v1/signals | 127.0.0.1:8081 | Go RT | Public | HTTPS | ✅ | PASS |
| api.predictatrade.com | /api/v1/market | 127.0.0.1:8081 | Go RT | Public | HTTPS | ✅ | PASS |
| api.predictatrade.com | / (root) | — | Blocked | — | HTTPS | ✅ 404 | PASS |
| api.predictatrade.com | /api/docs | — | Blocked | — | HTTPS | ✅ 404 | PASS |
| live.predictatrade.com | /ws/v1 | 127.0.0.1:8081 | Go RT WS | JWT | WSS | ✅ 101 | PASS |
| live.predictatrade.com | /ws | 127.0.0.1:8081 | Go RT WS | JWT | WSS | ✅ 101 | PASS |
| live.predictatrade.com | /health | 127.0.0.1:8081 | Go RT | Public | HTTPS | ✅ ok | PASS |
| live.predictatrade.com | /api/v1/market/state | 127.0.0.1:8081 | Go RT | Public | HTTPS | ✅ XAUUSD | PASS |
| live.predictatrade.com | /api/v1/signals | 127.0.0.1:8081 | Go RT | Public | HTTPS | ✅ | PASS |
| live.predictatrade.com | / (root) | — | Blocked | — | HTTPS | ✅ 404 | PASS |
| platform.predictatrade.com | / | 127.0.0.1:4600 | Next.js | JWT | HTTPS | ✅ 200 | PASS |
| platform.predictatrade.com | /ws | 301→live | Redirect | — | HTTPS | ✅ 301 | PASS |
| status.predictatrade.com | / | 127.0.0.1:3100 | Status app | Public | HTTPS | ✅ 200 | PASS |
| status.predictatrade.com | /health | 127.0.0.1:3100 | Status app | Public | HTTPS | ✅ ok | PASS |
| status.predictatrade.com | /metrics | — | Blocked | — | HTTPS | ✅ 404 | PASS |
| All domains | /.env | — | Blocked | — | HTTPS | ✅ 403 | PASS |
| All domains | /.git/* | — | Blocked | — | HTTPS | ✅ 403 | PASS |

## External DNS Verification (real internet, no --resolve)

| Test | URL | Result |
|------|-----|--------|
| API Health | https://api.predictatrade.com/api/v1/health | ✅ ok, DB healthy |
| API Plans | https://api.predictatrade.com/api/v1/plans | ✅ 4 plans |
| API Register | https://api.predictatrade.com/api/v1/auth/register | ✅ JWT returned |
| API Login | https://api.predictatrade.com/api/v1/auth/login | ✅ JWT returned |
| Live Health | https://live.predictatrade.com/health | ✅ ok |
| Live Market | https://live.predictatrade.com/api/v1/market/state | ✅ XAUUSD $2432.52 |
| WebSocket | wss://live.predictatrade.com/ws/v1 | ✅ 101 Switching Protocols |
| Platform | https://platform.predictatrade.com/ | ✅ 200 |
| Status | https://status.predictatrade.com/health | ✅ ok |
| TLS Chain | api.predictatrade.com:443 | ✅ Let's Encrypt (YE1) |
| HTTP→HTTPS | http://api.predictatrade.com/ | ✅ 301 → HTTPS |
| CORS Preflight | OPTIONS api.predictatrade.com/api/v1/plans | ✅ Correct headers |
| Security Headers | api.predictatrade.com | ✅ HSTS, nosniff, frame, referrer |
| Blocked .env | api.predictatrade.com/.env | ✅ 403 |
| Blocked /api/docs | api.predictatrade.com/api/docs | ✅ 404 |
| Blocked /metrics | status.predictatrade.com/metrics | ✅ 404 |
| No public ports | 152.53.67.111:13080/13081/13082/3100 | ✅ Blocked externally |

## Backend Dependency Matrix

| CALLER | DESTINATION | PRODUCTION URL | INTERNAL UPSTREAM | CONTRACT | RESULT |
|--------|------------|---------------|-------------------|----------|--------|
| Platform (Next.js) | NestJS API | https://api.predictatrade.com/api/v1 | 127.0.0.1:3000 | REST/JSON | PASS |
| Platform (Next.js) | Go RT WS | wss://live.predictatrade.com/ws/v1 | 127.0.0.1:8081 | WebSocket | PASS |
| Windows Agent | Realtime WS | wss://live.predictatrade.com/ws/v1/agent | 127.0.0.1:8081 | WebSocket | PASS |
| Windows Agent | Control API | https://api.predictatrade.com/api/v1 | 127.0.0.1:3000 | REST/JSON | PASS |
| MT4/MT5 EA | Windows Agent | Local named pipe | localhost IPC | JSON/pipe | PASS |
| Status Page | Health checks | Internal 127.0.0.1 | 127.0.0.1:3000+8081 | REST/JSON | PASS |
| Nginx | NestJS | 127.0.0.1:3000 | — | HTTP proxy | PASS |
| Nginx | Go RT | 127.0.0.1:8081 | — | HTTP/WS proxy | PASS |
| Nginx | Next.js | 127.0.0.1:4600 | — | HTTP proxy | PASS |
| Nginx | Status | 127.0.0.1:3100 | — | HTTP proxy | PASS |
