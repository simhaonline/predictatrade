---
name: frontend-trading-ui
description: "Build Next.js trading dashboards and command center."
---

# frontend-trading-ui

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and `AGENTS.md`.

## Tech Stack
- Next.js 16.3.1, React 19.2.8, lightweight-charts 5.2.1, recharts 3.10.1
- Tailwind CSS 3.4, TanStack Query/Table/Virtual, Zod 4.4
- Axios, Sonner toasts, next-themes (dark/light)

## Pages
### Public: /register, /login, /verify-otp, /reset-password, /terms, /privacy, /sitemap, /preview
### User Dashboard: /dashboard/live, /dashboard/backtest, /dashboard/mt4-mt5-client, /dashboard/billing, /dashboard/license, /dashboard/activity-log
### Admin: /admin/operations, /admin/market-data, /admin/macro-intelligence, /admin/macro-news, /admin/indicators, /admin/licenses, /admin/mt-accounts, /admin/payout-operations, /admin/logs

## Workflow
1. Use real APIs/WS; client guards are UX only.
2. MARKET/TRADING/GROWTH/COMMAND_CENTER modes.
3. Honest loading/empty/stale/degraded/error/demo/replay states.
4. Dark/light tokens, responsive/full-screen/4K, accessibility/reduced-motion.

## Forbidden
Fake live activity; client-side production indicators/risk/probability/finance.
