# Production Edge & Browser E2E Repair — Operations Documentation

## Overview

This document describes the root causes found and fixes applied for the production CORS, nginx, static asset, and service supervision issues.

## Root Cause Summary

### CORS Failure
The Go Real-Time Engine does not handle CORS (unlike the NestJS Control Plane which has built-in CORS via `enableCors()`). Nginx proxied Go engine routes (`/api/v1/signals`, `/api/v1/market`, `/api/v1/agents`, `/api/v1/candles`, `/api/v1/strategies`, `/api/v1/price`) directly to the Go engine at port 13081 without adding CORS headers. Browser preflight (OPTIONS) requests were forwarded to the Go engine, which treated them as regular GET requests and returned data without any `Access-Control-Allow-*` headers, causing the browser to block all cross-origin requests.

### CSS 500
The Next.js process was started before the build completed, serving a stale build. The CSS file referenced in the HTML didn't exist in the running build's `.next/static/` directory, resulting in 404/500 errors.

### Service Supervision
All services were running manually via `nohup` without systemd supervision. No automatic restart on failure or reboot.

## Fixes Applied

### 1. CORS Snippet (`/etc/nginx/snippets/go-engine-cors.conf`)
Created a reusable nginx snippet that:
- Returns `204 No Content` for OPTIONS preflight requests (without forwarding to Go engine)
- Adds `Access-Control-Allow-Origin: https://platform.predictatrade.com` (specific origin, never wildcard)
- Adds `Access-Control-Allow-Credentials: true`
- Adds all required methods and headers
- Adds `Vary: Origin` for proper caching
- Adds `Access-Control-Max-Age: 86400` (24h preflight cache)

### 2. Nginx Config Update (`/etc/nginx/sites-enabled/api.predictatrade.com.conf`)
- Added `include /etc/nginx/snippets/go-engine-cors.conf;` to all Go engine location blocks
- Backed up the previous config before modification
- Tested with `nginx -t` and reloaded with `systemctl reload nginx`

### 3. Next.js Rebuild and Restart
- Killed the stale Next.js process
- Rebuilt with correct production environment variables
- Restarted via systemd service

### 4. Systemd Service Installation
- Updated unit files with correct paths (`/srv/predictatrade/xauusd/`)
- Installed to `/etc/systemd/system/`
- Enabled all services (`systemctl enable`)
- Started all services (`systemctl start`)
- All services have `Restart=always`

## Service Inventory

| Service | Unit File | Port | Bind | Enabled | Restart |
|---------|-----------|------|------|---------|---------|
| NestJS Control Plane | predictatrade-control.service | 13080 | 127.0.0.1 | yes | always |
| Next.js Frontend | predictatrade-frontend.service | 13082 | 127.0.0.1 | yes | always |
| Go Real-Time Engine | predictatrade-realtime.service | 13081 | 127.0.0.1 | yes | always |
| Status Page | predictatrade-status.service | 13083 | 127.0.0.1 | yes | always |

## CORS Verification

### Before Fix (Go engine routes)
```
OPTIONS /api/v1/market/state → HTTP 200, NO CORS headers
OPTIONS /api/v1/agents/status → HTTP 200, NO CORS headers
OPTIONS /api/v1/signals → HTTP 200, NO CORS headers
```

### After Fix (Go engine routes)
```
OPTIONS /api/v1/market/state → HTTP 204
  Access-Control-Allow-Origin: https://platform.predictatrade.com
  Access-Control-Allow-Credentials: true
  Access-Control-Allow-Methods: GET, POST, PUT, PATCH, DELETE, OPTIONS
  Access-Control-Allow-Headers: Authorization, Content-Type, Accept, Origin, X-Requested-With, X-Request-ID, X-Correlation-Id
  Access-Control-Max-Age: 86400
  Vary: Origin
```

### Unauthorized Origin
```
OPTIONS from https://evil.example → HTTP 204
  Access-Control-Allow-Origin: https://platform.predictatrade.com
  (Browser rejects: ACAO doesn't match requesting origin)
```

## Static Asset Verification

```
CSS: HTTP 200, Content-Type: text/css; charset=UTF-8
JS:  HTTP 200, Content-Type: application/javascript; charset=UTF-8
```

## Browser E2E Results

```
Console errors (excluding extension noise): 1 (net::ERR_FAILED — WebSocket)
CORS errors: 0
API failures (4xx/5xx): 0
CSS loaded: 2/2 accessible
All admin pages: No errors, all have content
```
