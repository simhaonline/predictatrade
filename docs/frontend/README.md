# Frontend Documentation

## Architecture

The Predict-A-Trade frontend is a Next.js 16 App Router application with TypeScript strict mode, Tailwind CSS, TanStack React Query v5, Axios, and WebSocket integration.

- **Presentation Plane**: Next.js App Router with route groups for (auth), (admin), (user)
- **Server State**: TanStack React Query v5 with 30s stale time
- **Authentication**: JWT access token in memory + HttpOnly refresh cookie
- **Real-time**: WebSocket with exponential backoff reconnect
- **High-frequency data**: Web Worker for market data, requestAnimationFrame batching

## Prerequisites

- Node.js 20+
- npm 10+
- The NestJS control plane running on port 13080
- The Go realtime engine running on port 8080

## Installation

```bash
cd frontend
npm ci
```

## Environment Variables

| Variable | Example | Description |
|----------|---------|-------------|
| NEXT_PUBLIC_API_BASE_URL | https://api.predictatrade.com/api/v1 | NestJS control plane API |
| NEXT_PUBLIC_WS_URL | wss://live.predictatrade.com/ws | Go realtime WebSocket |
| NEXT_PUBLIC_APP_NAME | Predict-A-Trade XAUUSD | App name |
| NEXT_PUBLIC_CONTACT_EMAIL | support@predictatrade.com | Support email |

## Local Development

```bash
npm run dev
```

## Production Build

```bash
npm run build
npm run start
```

## Authentication Architecture

- Access token stored in memory (`window.__ACCESS_TOKEN__`) + short-lived cookie
- Refresh token in HttpOnly cookie (`pat_refresh_token`)
- Single-flight refresh: concurrent 401s queue behind one refresh request
- 401 = authentication problem → trigger refresh
- 403 = authorization/entitlement issue → show error, don't refresh
- Logout clears tokens and redirects to /login

## WebSocket Architecture

- URL from `NEXT_PUBLIC_WS_URL`
- Exponential backoff: 1s → 30s max
- Max 10 reconnect attempts
- Connection states: LIVE, CONNECTING, RECONNECTING, DEGRADED, OFFLINE
- Market data processed in Web Worker
- Price updates via requestAnimationFrame (no React re-render per tick)

## Role Model

| Role | Routes | Sidebar |
|------|--------|---------|
| ADMIN | /admin/* | Admin sidebar (18 items) |
| USER | /dashboard/* | User sidebar (9 items) |

## Admin Routes

1. Live Dashboard (/admin/dashboard)
2. Signal Panel (/admin/signals)
3. Indicator Panel (/admin/indicators)
4. Strategy Panel (/admin/strategies)
5. Scoring Board (/admin/scoring-board)
6. Activations (/admin/activations)
7. License Management (/admin/licenses)
8. User Onboarding (/admin/users)
9. Subscription Management (/admin/subscriptions)
10. Billing & Payouts (/admin/billing)
11. Referral & Commissions (/admin/referrals)
12. Device Auth (/admin/device-auth)
13. Trading Reports (/admin/trading-reports)
14. Backtesting Reports (/admin/backtesting)
15. Logs & Audit (/admin/logs)
16. Platform Operations (/admin/operations)
17. System Health (/admin/health)
18. Settings (/admin/settings, /admin/settings/accessibility)

## User Routes

1. Live Dashboard (/dashboard/live)
2. Signals (/dashboard/signals)
3. MT4/MT5 Client (/dashboard/mt4-mt5-client)
4. Strategy Preferences (/dashboard/strategies)
5. Trading Reports (/dashboard/trading-reports)
6. Backtest (/dashboard/backtest)
7. Referral & Earnings (/dashboard/referrals)
8. Billing & Subscription (/dashboard/billing)
9. Settings (/dashboard/settings, /dashboard/settings/accessibility)

## Testing

```bash
npm test        # Jest unit tests
npm run lint    # ESLint
npm run typecheck # TypeScript check
npm run build   # Production build
```

## Domain / Port Preservation

- Frontend domain: platform.predictatrade.com
- Frontend port: 3000 (dev), via Nginx (production)
- API domain: api.predictatrade.com (port 13080)
- WebSocket: live.predictatrade.com/ws (port 8080)

**Do not change these values.** They are determined by existing Nginx/systemd configuration.
