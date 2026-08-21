# Predict-A-Trade Frontend

Production Next.js 16 frontend for the Predict-A-Trade XAUUSD trading platform.

## Quick Start

```bash
npm ci
npm run dev
```

## Build

```bash
npm run lint
npm run typecheck
npm test
npm run build
```

## Environment

See `.env.example`. Key variables:
- `NEXT_PUBLIC_API_BASE_URL` - NestJS API (default: http://localhost:3000/api/v1)
- `NEXT_PUBLIC_WS_URL` - Go realtime WebSocket (default: ws://localhost:8080/ws/v1)

## Architecture

- Next.js App Router with (auth), (admin), (user) route groups
- TanStack React Query v5 for server state
- Axios with single-flight refresh token rotation
- WebSocket with exponential backoff reconnect
- Web Worker for high-frequency market data
- Middleware-based route protection (RBAC from JWT)

## Domain / Port

- Frontend: platform.predictatrade.com (port 3000)
- API: api.predictatrade.com (port 13080)
- WebSocket: live.predictatrade.com/ws (port 8080)

Do not change these values without updating Nginx/systemd configuration.

## Signal Display

The frontend displays signals from the Go real-time engine with the following direction types:

| Direction | Color | Meaning |
|-----------|-------|---------|
| BUY | Green (#10B981) | Qualified long (executable) |
| SELL | Red (#EF4444) | Qualified short (executable) |
| BUY_CANDIDATE | Amber (#F59E0B) | Advisory long (not executable) |
| SELL_CANDIDATE | Orange (#FB923C) | Advisory short (not executable) |
| NO-TRADE | Gray | No valid trade opportunity |
| BLOCKED | Gray | Gate veto or safety block (direction preserved) |

The PROB (calibrated probability) column shows "Pending" until a calibration model is validated. See `docs/SIGNAL_TYPES_AND_PROBABILITY.md` for details.

## Color Palette (v1.4.0)

The frontend uses the approved Predict-A-Trade color palette via CSS variables in `src/styles/globals.css` and semantic Tailwind tokens in `tailwind.config.ts`:

- **Light mode**: sidebar #0F172A, main bg #F8FAFC, cards #FFFFFF, borders #E2E8F0
- **Dark mode**: sidebar #020617, main bg #0F172A, cards #1E293B, borders #334155
- **Trading**: BUY/TP #10B981, SELL/SL #EF4444, SESSION #EAB308, INFO #3B82F6
- **Candidate**: BUY_CANDIDATE #F59E0B, SELL_CANDIDATE #FB923C

All colors are theme-aware via CSS variables with `%` signs on HSL S/L components.
