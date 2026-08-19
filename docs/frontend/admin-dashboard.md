# Admin Dashboard

## Architecture

The Admin Dashboard is an operations command center at `/admin/dashboard`, completely separate from the User Dashboard.

## Data Sources

- **NestJS Control Plane** (port 13080): /admin/overview, /operations/state, /health
- **Go Realtime Engine** (port 13081): /api/v1/health, /api/v1/market/state, /api/v1/agents/status
- **WebSocket** (wss://live.predictatrade.com/ws): live signals, market ticks, agent status

## Dashboard Sections

1. **Platform Status Strip**: Trading, Signals, Master Node, Market Feed, RT Engine, Control Plane, Database, WebSocket
2. **Market Data Panel**: Live XAUUSD Bid/Ask/Spread with feed state (LIVE/STALE/DEGRADED/OFFLINE)
3. **Master Node Panel**: Windows Agent connection status, agent count, WS clients
4. **Service Health**: RT Engine, Control Plane, Database, Valkey status badges
5. **Platform Metrics**: Users, Subscriptions, Commissions, Payouts, Plans, Agents stat cards
6. **Signal Pipeline**: Live signal feed from WebSocket
7. **Active Strategies**: Strategy enable/disable state
8. **Platform Operations State**: Trading mode, signal generation, engine version

## Feed State Detection

- LIVE: WebSocket connected, market tick within 30 seconds
- STALE: Market data older than 30 seconds
- DEGRADED: WebSocket disconnected but data exists
- OFFLINE: No market data available

## No Fake Data

All prices come from real Go engine market state. If feed is unavailable, the dashboard shows "No live market data" — never hardcoded prices.
