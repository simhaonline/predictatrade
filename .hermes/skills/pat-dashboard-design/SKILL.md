---
name: pat-dashboard-design
description: "Design Predict-A-Trade dashboards and UI surfaces."
---

# pat-dashboard-design

Use when designing, reviewing, or rebuilding any Predict-A-Trade UI surface:
admin dashboard, user dashboard, live command center, marketing pages, or component library.

## Project Design Tokens
The frontend uses Tailwind CSS 3.4 with next-themes (dark/light).
See tailwind.config.ts for exact values.

## Screen Inventory (from codebase audit)

### User Dashboard Pages
- /dashboard/live — XAUUSD Live Command Center (Monitor surface)
- /dashboard/backtest — Backtest configuration and results
- /dashboard/mt4-mt5-client — MT4/MT5 client download and setup
- /dashboard/billing — Subscription management
- /dashboard/license — License key management
- /dashboard/activity-log — User activity history

### Admin Dashboard Pages
- /admin/operations — System operations overview (Monitor surface)
- /admin/market-data — Market data feed management
- /admin/macro-intelligence — Macro economic analysis
- /admin/macro-news — Economic news feed
- /admin/indicators — Indicator monitor (liveness, performance)
- /admin/licenses — License management
- /admin/mt-accounts — MT account management
- /admin/payout-operations — Payout processing
- /admin/logs — Audit and system logs

### Public Pages
- /register, /login, /verify-otp, /reset-password
- /terms, /privacy, /preview
- Landing page

## Design Principles (from AGENTS.md)
1. Server-authoritative truth — UI renders, never recomputes indicators/risk/probability/finance
2. MARKET, TRADING, GROWTH, COMMAND_CENTER modes
3. Honest states: loading, empty, stale, degraded, error, demo, replay
4. Dark/light tokens, responsive/full-screen/4K, accessibility/reduced-motion
5. Admin is operations/business console, NOT a duplicate trader terminal

## Known UI Issues (from audit)
- F4: Relay feed status parsed then dropped — stale feeds shown as LIVE
- F5: Backtest FAILED status never surfaced in UI
- F6: Custom checkbox div not keyboard accessible
- F7: No prefers-reduced-motion handling
- F8: Dead components (command-center.tsx, live-dashboard.tsx, orval.config.ts)

## Components of Interest
- user-command-center/ (market-header, signal-pipeline, indicator-cards, mtf-pulse, growth-panel)
- indicator-monitor/ (active-reactive-table, indicator-charts, liveness-matrix, performance-matrix, summary-cards)
- trading/live-dashboard.tsx
- backtest/backtest-panel.tsx
- signal/signal-evidence.tsx
- market-context/market-context-panel.tsx

## Workflow
1. Identify which surface type (Monitor/Operate/Compare/Configure/Decide/Explore/Command)
2. Load the relevant design system (popular-web-designs for reference, claude-design for process)
3. Inspect the actual component source files before designing
4. Match existing Tailwind tokens and component patterns
5. Verify against AGENTS.md UI requirements
