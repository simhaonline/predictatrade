# Admin Dashboard Forensic Repair — Operations Documentation

## Overview

This document describes the root causes found and fixes applied to the Predict-A-Trade Admin Dashboard.

## Architecture

```
Browser → Next.js (13082) → Nginx → NestJS Control Plane (13080)  for admin/SaaS APIs
                                       → Go Real-Time Engine (13081) for signals/market/agents
Windows Agent → Go Engine ingestion → indicators → scoring → signals → WebSocket
```

## Service Ports

| Service                  | Port  | Bind Address |
|--------------------------|-------|--------------|
| NestJS Control Plane     | 13080 | 127.0.0.1    |
| Go Real-Time Engine      | 13081 | 127.0.0.1    |
| Next.js Frontend         | 13082 | 127.0.0.1    |
| Status Page              | 13083 | 127.0.0.1    |
| PostgreSQL/TimescaleDB   | 5432  | 127.0.0.1    |
| Valkey/Redis             | 6379  | 127.0.0.1    |

## Nginx Routing

- `api.predictatrade.com/api/v1/signals` → Go Engine (13081)
- `api.predictatrade.com/api/v1/market` → Go Engine (13081)
- `api.predictatrade.com/api/v1/candles` → Go Engine (13081)
- `api.predictatrade.com/api/v1/strategies` → Go Engine (13081)
- `api.predictatrade.com/api/v1/agents` → Go Engine (13081) **[ADDED]**
- `api.predictatrade.com/api/v1/health` → NestJS (13080)
- `api.predictatrade.com/api/v1/*` → NestJS (13080)
- `live.predictatrade.com/ws` → Go Engine WebSocket (13081)
- `platform.predictatrade.com/*` → Next.js (13082)

## Environment Variables

### Control Plane (control.env)
- `CONTROL_HOST` / `CONTROL_PORT` — bind address
- `JWT_SECRET` — token signing
- `CORS_ORIGINS` — allowed origins
- `DATABASE_URL` — loaded from `/srv/predictatrade/xauusd/database_url.txt`
- `GO_ENGINE_HEALTH_URL` — Go engine health check URL (default: http://127.0.0.1:13081/health)
- `GO_ENGINE_AGENTS_URL` — Go engine agents status URL
- `FRONTEND_HEALTH_URL` — Frontend health check URL

### Frontend (frontend.env)
- `NEXT_PUBLIC_API_BASE_URL` — API base URL (also checks `NEXT_PUBLIC_API_URL` as fallback)
- `NEXT_PUBLIC_WS_URL` — WebSocket URL for live signals

## Stale-Data Handling

### Market Data
- `LIVE` — last tick < 10 seconds old
- `STALE` — last tick 10-60 seconds old
- `OFFLINE` — no tick data or > 60 seconds old

### Agent/Master Node
- `ONLINE` — heartbeat < 45 seconds old
- `OFFLINE` — no heartbeat or expired

### Engine Health
- `HEALTHY` — health endpoint responds with status=ok
- `DEGRADED` — responds but status≠ok
- `OFFLINE` — connection refused or timeout
- `UNKNOWN` — cannot determine

## Database Schema Notes

### Key Tables and Correct Column Names

| Table                          | Correct Columns                                         |
|--------------------------------|---------------------------------------------------------|
| billing.subscriptions          | billing_period_start, billing_period_end (NOT current_period_*) |
| licensing.devices              | connection_status (NOT status), first_seen_at (NOT registered_at), windows_version (NOT os) |
| referral.commission_ledger     | level (NOT commission_level)                            |
| referral.payouts               | requested_amount (NOT amount), requested_at             |
| audit.audit_events             | action, actor_id, timestamp (NOT event_type, user_id, created_at) |

## Diagnostics

### Check all admin APIs
```bash
TOKEN=$(curl -s -X POST http://127.0.0.1:13080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@simhaonline.com","password":"..."}' | jq -r .accessToken)

for endpoint in admin/overview admin/users admin/subscriptions admin/commissions admin/payouts admin/devices admin/licenses admin/health audit; do
  echo "=== $endpoint ==="
  curl -s -w "\nHTTP %{http_code}\n" "http://127.0.0.1:13080/api/v1/$endpoint" -H "Authorization: Bearer $TOKEN"
done
```

### Check Go engine
```bash
curl -s http://127.0.0.1:13081/health
curl -s http://127.0.0.1:13081/api/v1/market/state | jq .
curl -s http://127.0.0.1:13081/api/v1/agents/status
curl -s http://127.0.0.1:13081/api/v1/signals | jq '.signals | length'
```

### Check services
```bash
ss -lntp | grep -E '13080|13081|13082|13083'
systemctl status predictatrade-control predictatrade-realtime predictatrade-frontend
```
