# Predict-A-Trade — Comprehensive Admin & User Dashboard UX/UI Audit

**Date:** 2026-08-25  
**Scope:** Admin Dashboard + User Dashboard (Next.js frontend), Control Plane (NestJS) API/permissions/entitlements, realtime/WebSocket integration, shared design system.  
**Auditor role:** Principal UX/UI Designer, SaaS Product Strategist, FinTech Dashboard Architect, Senior Frontend Product Auditor.  
**Authority:** `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md`, repository `AGENTS.md`, and `prompt.md` UX audit instructions.

---

# 1. EXECUTIVE SUMMARY

Predict-A-Trade is a large, functionally ambitious real-time XAUUSD signal platform. The frontend already has a cohesive Tailwind-based design token system, honest data-state logic in many places, and separates Admin (operational) and User (trading) dashboards at the route level. However, it currently carries **production-trust risks** and **usability debt** that must be resolved before a public launch.

**Overall Admin UX health:** The Admin dashboard is dense and covers the right operational domains, but the information architecture is overloaded (31 sidebar items), duplicated status components create inconsistency, several pages are disabled stubs, and key workflows require too much navigation. The Admin homepage does answer "Is the system live?" but buries revenue/alert context and risks `NaN` metric rendering.

**Overall User UX health:** The User dashboard has a clear live command-center concept, but the signal hierarchy is fragmented across multiple components (`command-center.tsx`, `SignalPipeline`, `SignalCard`, `signals/page.tsx`), free/unlicensed users receive misleading empty states, and growth-mode UI uses inaccessible emoji. The first screen is usable but not yet "trust at a glance."

**Largest usability risks:**
1. Flat/overloaded navigation in both dashboards.
2. Duplicate `StatusBadge` / `DegradedBanner` implementations causing inconsistent status semantics.
3. Generic or misleading empty states for unentitled users.
4. Tables with lexicographic-only sort, inaccessible column headers, and harsh light-mode dividers.
5. Fragmented signal presentation with no single canonical `SignalCard`.

**Largest trust/data-integrity risks:**
1. `live-dashboard.tsx` renders hardcoded placeholder prices (`2500.00`, `2500.50`, `0.50`) before the first WebSocket tick.
2. User feed badge says "LIVE" based purely on WebSocket connection state, while the Admin dashboard correctly derives feed liveness from backend tick age — the two are inconsistent.
3. `command-center.tsx` falls back to `0` for price/indicator values, which can look like real zero readings.
4. Client-side `AppShell` auth redirect only; no wired Next.js middleware for server-side route protection.

**Biggest improvement opportunities:**
1. Consolidate to one canonical `StatusBadge` and `FeedStatus` component.
2. Build a single source-of-truth `SignalCard` with Entry/SL/TP/Confidence/Freshness/Status hierarchy.
3. Replace hardcoded/placeholder market values with explicit uninitialized states.
4. Add entitlement-aware empty states and locked modules.
5. Reduce Admin navigation cognitive load via grouped/accordion menu.

**Scores (subjective, 1–10):**

| Area | Score | Rationale |
|------|-------|-----------|
| Admin UX | 5/10 | Comprehensive but dense, inconsistent components, disabled stubs, weak workflow shortcuts. |
| User UX | 5/10 | Clear concept but fragmented signal UX, misleading empty states, emoji/icons, inconsistent live labeling. |
| Design System Consistency | 4/10 | Good token base, but duplicate `StatusBadge`/`DegradedBanner`, inconsistent active-tab styling, divergent badge implementations. |
| Realtime Data Transparency | 5/10 | Honest states exist, but User "LIVE" is WS-only, `live-dashboard.tsx` has placeholder prices, no unified feed-status component. |
| Mobile/Responsive UX | 5/10 | Tailwind breakpoints present, but dense tables, horizontal scroll, no mobile-first card/table switch. |
| Accessibility | 4/10 | Some ARIA, but missing `role="tab"`, keyboard nav, focus-visible rings, `sr-only` labels, table semantics, axe/automated checks. |

---

# 2. EXISTING DASHBOARD INVENTORY

## Public / Auth routes

| Area | Route/Screen | Audience | Purpose | Current State | Major Problem |
|------|--------------|----------|---------|---------------|---------------|
| Public | `/` | All | Landing | Redirects to `/login` | No public marketing/entry surface. |
| Public | `/login` | All | Sign in | Functional | Form placeholders used as labels; no SSO/TOTP visible. |
| Public | `/register` | All | Sign up | Functional | MFA setup after register not visible in flow. |
| Public | `/forgot-password`, `/reset-password`, `/verify-otp` | All | Account recovery / MFA | Functional | Standard. |
| Public | `/preview` | Guests | Preview funnel | Functional | 5-minute server-side gated preview. |
| Public | `/terms`, `/privacy`, `/cookies`, `/complaints`, `/data-processing-agreement`, `/sitemap` | All | Legal/compliance | Static | Hardcoded last-updated dates; support email repeated. |

## User Dashboard routes

| Area | Route/Screen | Audience | Purpose | Current State | Major Problem |
|------|--------------|----------|---------|---------------|---------------|
| User | `/dashboard/live` | Subscriber | Live command center | Active | Fragmented signal UX, misleading free-user empty states, growth mode uses emoji. |
| User | `/dashboard/signals` | Subscriber | Signal history + evidence | Active | Filter shows all strategies regardless of entitlement; WS signal fields discarded; generic empty state. |
| User | `/dashboard/signal-accuracy` | Subscriber | Aggregate performance | Active | Historical stats exposed to all subscribers; must be clearly labeled as aggregate, not predictive guarantee. |
| User | `/dashboard/backtest` | Subscriber | Backtest runner | Active | Good honest error states; default dates hardcoded. |
| User | `/dashboard/strategies` | Subscriber | Strategy entitlement selection | Active | Depends on `/subscriptions/entitlements`. |
| User | `/dashboard/trading-reports` | Subscriber | Trading/position reports | Active | Reuses device data; good live/offline status. |
| User | `/dashboard/mt4-mt5-client` | Subscriber | MT account/device status | Active | Mixed MT4/MT5 label; helpful agent troubleshooting copy. |
| User | `/dashboard/referrals` | Subscriber | Referral network + earnings | Active | Empty state is good but commission table can be dense. |
| User | `/dashboard/billing` | Subscriber | Subscriptions + invoices | Active | Functional. |
| User | `/dashboard/payouts` | Subscriber | Payout requests + history | Active | Form placeholders; no empty-state guidance for first payout. |
| User | `/dashboard/license` | Subscriber | License key / device status | Active | Standard. |
| User | `/dashboard/security` | Subscriber | MFA, password, sessions | Active | Standard. |
| User | `/dashboard/activity-log` | Subscriber | Audit trail | Active | Standard. |
| User | `/dashboard/notifications` | Subscriber | Notification center | Active | Standard. |
| User | `/dashboard/support` | Subscriber | Support ticket | Active | Standard. |
| User | `/dashboard/settings` | Subscriber | Profile/preferences | Active | Standard. |
| User | `/dashboard/settings/accessibility` | Subscriber | Accessibility prefs | Active | Good: font scale, high contrast, reduced motion. |

## Admin Dashboard routes

| Area | Route/Screen | Audience | Purpose | Current State | Major Problem |
|------|--------------|----------|---------|---------------|---------------|
| Admin | `/admin/dashboard` | Admin/Super Admin | Operational overview | Active | Polls 7 endpoints independently; `NaN` risk; inconsistent badge logic; good engine cards. |
| Admin | `/admin/signals` | Admin | Signal monitor | Active | Strategy tabs + filters; inline status chip, not canonical `StatusBadge`. |
| Admin | `/admin/indicator-monitor` | Admin | Indicator liveness/charts | Active | Custom canvas charts hardcoded for dark mode; no lightweight-charts usage despite dependency. |
| Admin | `/admin/strategies` | Admin | Engine enable/disable | Active | Uses `StatusBadge`. |
| Admin | `/admin/regime-diagnostics` | Admin | Regime analysis | Active | Standard. |
| Admin | `/admin/scoring-board` | Admin | Scoring monitor | Active | Standard. |
| Admin | `/admin/risk-center` | Admin | Emergency controls | Active | Has reason input + audit; good safeguards. |
| Admin | `/admin/mt-accounts` | Admin | Linked MT accounts | Active | Functional. |
| Admin | `/admin/device-auth` | Admin | Device sessions | Active | Functional; good live/offline status. |
| Admin | `/admin/licenses` | Admin | License lifecycle | Active | Functional. |
| Admin | `/admin/activations` | Admin | Activation records | Active | Functional. |
| Admin | `/admin/users` | Admin | User management | Active | Row actions + bulk actions limited. |
| Admin | `/admin/subscriptions` | Admin | Subscription management | Active | Functional. |
| Admin | `/admin/plans-entitlements` | Admin | Plans + features | Active | Functional. |
| Admin | `/admin/billing` | Admin | Invoices + payments | Active | Functional. |
| Admin | `/admin/commission-operations` | Admin | Commission approval | Active | Functional. |
| Admin | `/admin/payout-operations` | Admin | Payout approval | Active | Functional; state-machine transitions present. |
| Admin | `/admin/referrals` | Admin | Referral network | Active | Functional. |
| Admin | `/admin/finance-referral-reports` | Admin | Financial reports | Active | Functional. |
| Admin | `/admin/market-data` | Admin | Data feeds | Active | Functional. |
| Admin | `/admin/macro-news` | Admin | Macro events | **Stub** | Disabled placeholder inputs; not live. |
| Admin | `/admin/macro-intelligence` | Admin | Macro confluence | Active | Defines its own `StatusBadge` vocabulary. |
| Admin | `/admin/ai-providers` | Admin | AI model management | Partially live | Only model activate/deactivate is live; rest is stub. |
| Admin | `/admin/broker-qualification` | Admin | Broker qualification | **Stub** | Disabled placeholder inputs. |
| Admin | `/admin/signal-accuracy` | Admin | Admin accuracy view | Active | Aggregate stats. |
| Admin | `/admin/releases` | Admin | Release management | **Stub** | Disabled placeholder inputs. |
| Admin | `/admin/backup-dr` | Admin | Backup / DR | **Stub** | Disabled placeholder inputs. |
| Admin | `/admin/feature-flags` | Admin | Feature flags | Active | Functional. |
| Admin | `/admin/trading-reports` | Admin | Admin trading reports | Active | Functional. |
| Admin | `/admin/backtesting` | Admin | Admin backtest queue | Active | Functional. |
| Admin | `/admin/operations` | Admin | Halt/resume signals | Active | Functional; good audit reason field. |
| Admin | `/admin/logs` | Admin | Audit logs | Active | Functional. |
| Admin | `/admin/health` | Admin | System health | Active | Real health checks. |
| Admin | `/admin/settings` | Admin | Admin settings | Active | Functional. |
| Admin | `/admin/settings/accessibility` | Admin | Accessibility prefs | Active | Functional. |

---

# 3. ADMIN DASHBOARD FINDINGS

## Critical Issues / UX Blockers

### A1. Hardcoded placeholder market prices in legacy `live-dashboard.tsx`
- **Severity:** P0 (production trust blocker — also listed under User findings because the component is shared/legacy).
- **Affected screen:** `/components/trading/live-dashboard.tsx` lines 44, 51, 58.
- **Issue:** Bid/Ask/Spread render `2500.00`, `2500.50`, `0.50` before the first WebSocket tick. They are not labeled as placeholder/demo.
- **Why it matters:** A user or admin could make a trading decision on fake data before real ticks arrive.
- **Evidence:** Static HTML rendered server-side/hydrated with those numbers; DOM refs overwrite later.
- **Recommended fix:** Remove `live-dashboard.tsx` from production routes or render `—` / "Waiting for market data…" until a real tick arrives.
- **Effort:** S.

### A2. Admin metric cards can render `NaN`
- **Severity:** P1.
- **Affected screen:** `/admin/dashboard/page.tsx` lines 172–178.
- **Issue:** `parseInt(overview?.users?.total ?? "0")` and similar calls will produce `NaN` when the value is `undefined` rather than the string `"0"`. The fallback `"0"` only applies when the whole expression is falsy; `undefined` is not coerced before `parseInt`.
- **Why it matters:** A broken API response causes the homepage to show "NaN users" — catastrophic trust hit.
- **Evidence:** `value: parseInt(overview?.users?.total ?? "0")`.
- **Recommended fix:** Use `Number(overview?.users?.total || 0)` or a safe formatter; validate API shape with a schema/Zod.
- **Effort:** XS.

### A3. Client-side-only route protection for Admin routes
- **Severity:** P1 (security + UX).
- **Affected screen:** `app-shell.tsx` and missing root `middleware.ts`.
- **Issue:** `AppShell` redirects non-admin users away from `/admin/*` after hydration. There is a `src/proxy.ts` but no `middleware.ts`, so server-rendered admin pages can leak HTML to unauthenticated clients before the redirect runs.
- **Why it matters:** Defense-in-depth failure; page structure and potentially data could be visible in initial HTML.
- **Evidence:** Subagent confirmed compiled middleware manifest is empty; `AppShell` uses `window.location.href` for redirect.
- **Recommended fix:** Implement `middleware.ts` using `src/proxy.ts` logic to enforce admin/user role checks at the edge.
- **Effort:** M.

### A4. Duplicate `StatusBadge` implementations cause inconsistent semantics
- **Severity:** P1.
- **Affected screen:** Systemic: `components/ui/status-badge.tsx`, `components/admin/status-badge.tsx`, plus local badges in `indicator-monitor/active-reactive-table.tsx`, `admin/macro-intelligence/page.tsx`, and inline badge code in `/admin/dashboard/page.tsx`.
- **Issue:** Multiple components solve the same problem with different status vocabularies and fallbacks.
- **Why it matters:** Same backend status may render with different colors/text depending on which page the admin is on.
- **Evidence:** UI version has 62 case-paired entries; admin version has clean vocabulary; macro-intelligence defines `CONNECTED`/`HEALTHY`; dashboard page uses inline ternary classes.
- **Recommended fix:** Consolidate into one `StatusBadge` with lowercased normalization, `role="status"`, unknown fallback, and a documented vocabulary.
- **Effort:** M.

### A5. Admin navigation is cognitively overloaded
- **Severity:** P1.
- **Affected screen:** `/config/navigation/admin-navigation.ts` + `sidebar.tsx`.
- **Issue:** 31 flat items in 5 sections with no accordion/collapse. Labels like "Macro & News" vs "Macro Intelligence" and "Signal Panel" vs "Signal Accuracy" are ambiguous.
- **Why it matters:** High-frequency operational tasks require scanning a very long list; operators will miss controls or click the wrong item.
- **Evidence:** 62-line nav file, 31 entries; sidebar renders all at once.
- **Recommended fix:** Collapsible sections, icon differentiation, restructure into fewer top-level groups, rename ambiguous labels.
- **Effort:** M.

### A6. Several Admin pages are disabled stubs without clear UX
- **Severity:** P1.
- **Affected screen:** `/admin/releases`, `/admin/backup-dr`, `/admin/macro-news`, `/admin/broker-qualification`, `/admin/commission-control-center`.
- **Issue:** Inputs are `disabled` with `opacity-60`; some pages have a small "LIVE" note but the form is not functional.
- **Why it matters:** Operators may attempt to fill/submit forms that cannot work; looks broken/unprofessional.
- **Evidence:** Disabled inputs with placeholder text in the files above.
- **Recommended fix:** Either implement the backend wiring or show a clear "Coming soon / read-only" state with explanation and expected timeline.
- **Effort:** S–M per page.

### A7. `DataTable` sorts all values as strings
- **Severity:** P1.
- **Affected screen:** `/components/ui/data-table.tsx` lines 31–35.
- **Issue:** All values cast to `String` and compared with `localeCompare`; numeric, date, boolean columns sort lexicographically.
- **Why it matters:** Revenue, dates, signal scores, and user IDs will sort incorrectly.
- **Evidence:** `const aStr = String(a[sortKey]); const bStr = String(b[sortKey]); return aStr.localeCompare(bStr);`.
- **Recommended fix:** Accept an optional `sortFn` per column or auto-detect number/date/boolean.
- **Effort:** S.

### A8. No keyboard/ARIA sort semantics in tables
- **Severity:** P1 (accessibility + usability).
- **Affected screen:** `/components/ui/data-table.tsx` lines 89–101.
- **Issue:** Column header is a `div` with `cursor-pointer`, not focusable, no `aria-sort`, no Enter/Space handler.
- **Why it matters:** Keyboard and screen-reader users cannot sort tables.
- **Evidence:** `<div ... onClick={() => handleSort(key)} className="cursor-pointer ...">`.
- **Recommended fix:** Use `<button>` with `aria-sort`, visible focus ring, and keyboard handler.
- **Effort:** S.

### A9. Admin dashboard polls many endpoints independently without deduplication
- **Severity:** P2 (performance/operational load).
- **Affected screen:** `/admin/dashboard/page.tsx` lines 12–64.
- **Issue:** 7 separate `useQuery` hooks on load with different intervals; some data may be available in a single `/admin/overview` call.
- **Why it matters:** Increases NestJS/Go load and produces waterfall layout shifts.
- **Evidence:** `admin-overview`, `ops-state`, `engine-signals`, `market-state`, `agents-status`, `nestjs-health`, `go-system-health`, `engines-status`.
- **Recommended fix:** Consolidate into one admin overview endpoint or use a single React Query observer with dependent queries.
- **Effort:** M (backend + frontend).

### A10. `DegradedBanner` redefined in 8 admin pages
- **Severity:** P2.
- **Affected screen:** Multiple admin pages.
- **Issue:** Inline copy-paste component; any style or wording change must be edited in many files.
- **Why it matters:** Maintenance burden; inconsistent degraded-state UX.
- **Evidence:** Subagent search found repeated banner patterns.
- **Recommended fix:** Move to `components/ui/degraded-banner.tsx` with consistent icon + message + retry CTA.
- **Effort:** S.

### A11. Admin `Signals Today` count uses raw signal array length, not date
- **Severity:** P2.
- **Affected screen:** `/admin/dashboard/page.tsx` line 414.
- **Issue:** `engineSignals?.signals?.length ?? 0` counts all returned signals regardless of creation date/timezone.
- **Why it matters:** Could overcount or undercount today's signals depending on API paging/filtering.
- **Evidence:** No date filtering in the frontend; assumes backend returns only today.
- **Recommended fix:** Use `CreatedAt` filtered to UTC day or request a dedicated metric from backend.
- **Effort:** XS.

### A12. No visible market-data timestamp on Admin homepage
- **Severity:** P2.
- **Affected screen:** `/admin/dashboard/page.tsx` lines 254–257.
- **Issue:** Source/session/regime are shown, but not the actual tick time or age.
- **Why it matters:** Operators cannot verify freshness of the displayed price.
- **Evidence:** `tickSource`, `currentSession`, `currentRegime` rendered; no `tickTimeStr` display.
- **Recommended fix:** Show "Last tick: 14:31:08 UTC (12s ago)" near the price.
- **Effort:** XS.

### A13. Spread warning threshold hardcoded
- **Severity:** P2.
- **Affected screen:** `/admin/dashboard/page.tsx` line 248 (and User equivalents).
- **Issue:** `spread > 0.5` warning is not derived from broker profile `symbol_info`.
- **Why it matters:** Different brokers/instruments have different normal spread ranges.
- **Evidence:** Hardcoded `0.5`.
- **Recommended fix:** Use `symbol_info` or backend-configured spread threshold.
- **Effort:** S.

## UI / Visual Improvements

| Topic | Finding | Severity | Evidence / Recommended Fix |
|-------|---------|----------|------------------------------|
| Layouts | Admin homepage is a long vertical stack of cards; no clear visual grouping beyond sections. | P2 | Add section headings/background bands; use a 2-column layout for secondary metrics. |
| Spacing | Heavy reliance on `space-y-4` and `gap-4`; feels uniform, not hierarchical. | P2 | Increase vertical rhythm between major sections; use tighter spacing within cards. |
| Hierarchy | Page titles (`text-xl`) and card titles (`text-sm`) are close in weight; cards compete for attention. | P2 | Increase title weight/size contrast; de-emphasize card borders in dark mode. |
| Cards | All cards share identical border/shadow styling; critical operational cards do not stand out. | P2 | Use subtle left-accent borders for status cards (e.g., red left border for halted). |
| Tables | `DataTable` uses `divide-y divide-neutral-800` in both themes, creating harsh light-mode lines. | P1 | Replace with `divide-pat-border` or theme-aware divider. |
| Badges | Inline ternary badge classes duplicated in admin dashboard. | P2 | Use `StatusBadge` everywhere. |
| Status design | WebSocket badge uses `wsState` string directly; `DISCONNECTED` and `RECONNECTING` are long for a small badge. | P2 | Show compact status labels: "WS Live", "WS Retry", "WS Off". |
| Typography | Numeric values use `font-mono` inconsistently; some percentages lack tabular nums. | P2 | Apply `tabular-nums` to all price/score/probability values. |
| Visual density | Admin sidebar + topbar consume significant horizontal space on 1366px screens. | P2 | Collapsible sections + optional compact sidebar mode. |

## Workflow Optimizations

### Find user → inspect subscription → device → license → action
**Current flow:** `Admin → Users → click user → drawer opens → mentally note email → navigate to Subscriptions/Licenses/Device-Auth → search/filter by email → action.`  
**Recommended flow:** Add contextual row actions and deep links in the user drawer to "View subscription", "View licenses", "View devices".  
**Step reduction:** ~4 clicks → ~1 click.

### Detect stale feed → identify provider → inspect error → recover
**Current flow:** `Admin dashboard → scan status strip → Market Data page → inspect provider table → check logs → action.`  
**Recommended flow:** Make status strip items clickable deep-links to the relevant detail page; show last error inline.  
**Step reduction:** ~4 navigations → 1 click.

### Inspect failed payment → subscription → resolve
**Current flow:** `Admin Billing → find invoice → open subscription id in new tab → change status.`  
**Recommended flow:** Inline "View subscription" link in invoice row action menu.  
**Step reduction:** ~3 clicks → 1 click.

### Approve/reject payout
**Current flow:** `Payout Operations → search row → open modal → select action → submit.`  
**Recommended flow:** Add quick approve/reject buttons in row actions with reason prompt.  
**Step reduction:** ~3 clicks → 1–2 clicks.

---

# 4. USER DASHBOARD FINDINGS

## Critical Issues / UX Blockers

### U1. Hardcoded placeholder market prices in `live-dashboard.tsx`
- **Severity:** P0.
- **Affected screen:** `/components/trading/live-dashboard.tsx` lines 44, 51, 58.
- **Issue:** Same as Admin A1 — static `2500.00` / `2500.50` / `0.50` displayed before WebSocket update.
- **Why it matters:** User may believe these are live prices.
- **Evidence:** Static rendered HTML with fake decimals.
- **Recommended fix:** Remove from user routes or replace with placeholder state. This component appears legacy; `command-center.tsx` and `user-command-center/*` should be used instead.
- **Effort:** S.

### U2. User feed status says "LIVE" based only on WebSocket connection, not data freshness
- **Severity:** P0/P1 (trust).
- **Affected screen:** `/components/user-dashboard/command-center.tsx` line 235; `/components/user-command-center/market-header.tsx` line 155.
- **Issue:** `command-center.tsx` badge text is `"LIVE"` when `wsState === "CONNECTED"`. The market header uses the relay's honest `FeedStatus` and falls back to `"REST 3s"` / `"CONNECTING"`, which is better, but there is no single source of truth.
- **Why it matters:** A connected WebSocket can carry stale or replay data; "LIVE" must reflect data freshness, not socket state.
- **Evidence:** `wsState === "CONNECTED" ? "LIVE" : wsState`.
- **Recommended fix:** Use the relay `FeedStatus` (`LIVE`, `DEGRADED`, `STALE`, `REPLAY`, `UNKNOWN`) consistently across both components.
- **Effort:** S.

### U3. Free/unlicensed users see a misleading empty signal state
- **Severity:** P1.
- **Affected screen:** `/app/(user)/dashboard/signals/page.tsx` lines 170–173.
- **Issue:** The empty message is "No signals match the current filters" even when the real cause is missing entitlement.
- **Why it matters:** User will adjust filters repeatedly instead of understanding they need a subscription.
- **Evidence:** `if (filtered.length === 0)` returns generic message; license `allowed_strategies` is loaded but not used to explain restriction.
- **Recommended fix:** Detect empty because of entitlement vs. genuinely no signals; render "Your current plan does not include these strategies. Upgrade to unlock."
- **Effort:** S.

### U4. Signal filters show all four strategies regardless of entitlement
- **Severity:** P1.
- **Affected screen:** `/app/(user)/dashboard/signals/page.tsx` lines 152–155.
- **Issue:** Strategy filter chips always show all four strategies.
- **Why it matters:** User can select strategies they are not entitled to, producing empty results and confusion.
- **Evidence:** `STRATEGIES.map(...)` with no `allowed_strategies` filtering.
- **Recommended fix:** Disable or hide unentitled strategy filters with a lock tooltip.
- **Effort:** S.

### U5. WebSocket signal normalization discards key fields
- **Severity:** P1.
- **Affected screen:** `/app/(user)/dashboard/signals/page.tsx` lines 47–64.
- **Issue:** `mapWs` keeps only `ID`, `StrategyID`, `Direction`, `EntryPrice`, `StopLoss`, `TP1`, `Status`, `CreatedAt`; drops `TP2`, `TP3`, `Regime`, `Session`, `RawScore`, `CalibratedProbability`, `ReasonCodes`.
- **Why it matters:** Live-updated signals look different from REST-loaded signals in the same table.
- **Evidence:** Field mapping in `mapWs`.
- **Recommended fix:** Preserve all signal fields during normalization or share a single signal schema converter.
- **Effort:** S.

### U6. `SignalCard` direction color fallback is wrong
- **Severity:** P1.
- **Affected screen:** `/components/user-dashboard/command-center.tsx` lines 528–533.
- **Issue:** Any unknown direction falls through to `text-pat-candidate-sell` (orange) instead of a neutral color.
- **Why it matters:** `NO-TRADE`, `WAIT`, or future directions will look like a sell candidate.
- **Evidence:** Fallback clause `signal.Direction === "BUY_CANDIDATE" ? "text-pat-warning" : "text-pat-candidate-sell"`.
- **Recommended fix:** Explicitly map every direction; default to `text-pat-text-muted` for unknown.
- **Effort:** XS.

### U7. `SignalCard` throws if `StrategyID` is missing
- **Severity:** P1.
- **Affected screen:** `/components/user-dashboard/command-center.tsx` lines 534, 631.
- **Issue:** `signal.StrategyID.replace(/_/g, " ")` assumes `StrategyID` is always a string.
- **Why it matters:** A malformed signal crashes the UI.
- **Evidence:** Direct method call on potentially undefined value.
- **Recommended fix:** `signal.StrategyID?.replace(/_/g, " ") || "Unknown"`.
- **Effort:** XS.

### U8. Growth mode uses emoji icons
- **Severity:** P1 (accessibility + credibility).
- **Affected screen:** `/components/user-dashboard/command-center.tsx` lines 434–437.
- **Issue:** `📊`, `💰`, `✅`, `⏳` used as card icons in a premium FinTech product.
- **Why it matters:** Emojis lack alt text, render inconsistently across OS/browsers, and cheapen the brand.
- **Evidence:** `icon="📊"` etc.
- **Recommended fix:** Replace with Tabler icons (`IconChartBar`, `IconCoins`, `IconCheck`, `IconClockHourglass`).
- **Effort:** XS.

### U9. Mode tabs duplicated and inconsistent between page and component
- **Severity:** P1.
- **Affected screen:** `/app/(user)/dashboard/live/page.tsx` lines 50–65 vs `/components/user-dashboard/command-center.tsx` lines 117–131.
- **Issue:** `page.tsx` uses icon+text tabs; `command-center.tsx` uses text-only tabs. If both are rendered, UX diverges. `page.tsx` appears to render `MarketHeader` and then switch modes; `command-center.tsx` also has a header.
- **Why it matters:** Two different tab systems for the same conceptual modes.
- **Evidence:** Two implementations of the same `Mode` union.
- **Recommended fix:** Use `command-center.tsx` as the canonical command center; remove duplicate tabs from `page.tsx` or make `page.tsx` a thin wrapper.
- **Effort:** S.

### U10. No clear empty/loading/error states for `MarketMode`, `TradingMode`, `GrowthMode`
- **Severity:** P1.
- **Affected screen:** `/components/user-dashboard/command-center.tsx` lines 244–459.
- **Issue:** If a query fails, panels silently render zeros/dashes or empty lists without an error/retry banner.
- **Why it matters:** Users cannot distinguish "no data" from "data failed to load."
- **Evidence:** No `isError`/`isLoading` handling inside mode subcomponents.
- **Recommended fix:** Pass `isLoading`/`isError`/`error` props to each mode and render skeleton/error states.
- **Effort:** M.

### U11. `IndicatorCard` hides real zero values as `—`
- **Severity:** P2.
- **Affected screen:** `/components/user-dashboard/command-center.tsx` line 501.
- **Issue:** `value === 0 || value === false ? "—"` means an actual RSI of 0 or MACD of 0 is hidden.
- **Why it matters:** Real zero readings are meaningful for some indicators.
- **Evidence:** Display helper.
- **Recommended fix:** Distinguish `undefined/null` from `0`; only mask missing values.
- **Effort:** XS.

### U12. Signal history table expanded row `colSpan` mismatch
- **Severity:** P1 (rendering bug).
- **Affected screen:** `/app/(user)/dashboard/signals/page.tsx` lines 179, 223.
- **Issue:** Header has 11 `<th>` elements plus an empty first column for the expand icon (12 columns total). Expanded evidence row uses `colSpan={11}`.
- **Why it matters:** Expanded row will not span the full table width.
- **Evidence:** Counted columns vs `colSpan`.
- **Recommended fix:** Set `colSpan={12}` or compute from header count.
- **Effort:** XS.

### U13. Clickable signal rows are not keyboard accessible
- **Severity:** P1 (accessibility).
- **Affected screen:** `/app/(user)/dashboard/signals/page.tsx` line 197.
- **Issue:** Expand/collapse is triggered by `onClick` on a `tr`; no keyboard handler or focus indicator.
- **Why it matters:** Keyboard and screen-reader users cannot expand evidence.
- **Evidence:** `onClick={() => toggleRow(...)}` on `<tr>`.
- **Recommended fix:** Add a focusable expand button in the first cell with `aria-expanded`.
- **Effort:** XS.

### U14. User navigation has 16 flat items with no grouping
- **Severity:** P1.
- **Affected screen:** `/config/navigation/user-navigation.ts` + `sidebar.tsx`.
- **Issue:** No sections; trading, growth, account, and support items are interleaved.
- **Why it matters:** New users cannot build a mental model of the product.
- **Evidence:** 16 top-level entries.
- **Recommended fix:** Group into Trading, Growth, Account, Support sections in the sidebar.
- **Effort:** S.

## UI / Visual Improvements

| Screen | Finding | Severity | Recommended Fix |
|--------|---------|----------|-----------------|
| Home page | `MarketContextPanel` sits at bottom with no heading boundary. | P2 | Add section title and divider; or move into Market mode only. |
| Home page | `MarketHeader` is always visible but `page.tsx` fetches its own snapshot redundantly. | P2 | Pass snapshot down or let `command-center.tsx` own the header. |
| Signal card | Candidate vs qualified uses background colors only; no text label distinction. | P2 | Add "Candidate" badge for advisory signals. |
| Signal history | Status chips use inline colored text + border, not `StatusBadge`. | P2 | Use canonical `StatusBadge`. |
| Market context | `MarketContextPanel` has no loading/error skeleton. | P2 | Add skeleton and degraded state. |
| Subscription | No usage quota counter on the dashboard home. | P2 | Show "X of Y signals used this month" prominently. |
| Profile/settings | Standard; grouped reasonably. | — | — |
| Notifications | Standard. | — | — |
| Billing | Standard. | — | — |
| Referral | Commission table is dense on mobile. | P2 | Convert to cards on narrow screens. |

## Engagement & Onboarding

| Step | Finding | Severity | Recommended Fix |
|------|---------|----------|-----------------|
| Account creation | Standard register + MFA; no guided plan selection immediately after. | P2 | After email verification, prompt plan selection before first dashboard view. |
| First login | User lands on `/dashboard/live` with Market mode; lots of indicators before explanation. | P1 | Show a one-time, dismissible coach mark explaining signal card hierarchy and freshness. |
| Empty states | Free user signal empty state is misleading (see U3). | P1 | Entitlement-aware empty states. |
| Time-to-value | Signal is visible quickly if entitlement and market conditions allow. | — | Good. |
| Subscription onboarding | Strategy selection page exists but is not highlighted to new users. | P2 | Add "Choose your strategies" CTA after plan selection. |
| Quota reached | Not prominently surfaced on the live dashboard. | P1 | Add quota card/banner when near or at limit. |
| Upgrades | Upgrade prompts not aggressively present; could be clearer about what unlocks. | P2 | Use context-specific unlock messages, e.g., "Unlock Standard Swing signals." |
| Retention | No in-app engagement nudges except referral. | P3 | Add signal-alerts/opt-in notifications after MVP trust issues are fixed. |

---

# 5. REALTIME / SYSTEM-HEALTH UX FINDINGS

| Component | Current UX | Problem | Recommended Status UX | Severity |
|-----------|------------|---------|-----------------------|----------|
| XAUUSD market feed (User) | Badge says "LIVE" when WebSocket is connected. | Confuses socket state with data freshness; could show stale prices as live. | Use relay `FeedStatus`: LIVE / DEGRADED / STALE / REPLAY / UNKNOWN with last tick age. | P0/P1 |
| XAUUSD market feed (Admin) | Derived from backend tick age: OFFLINE / LIVE / DEGRADED / STALE. | Good. | Make this the canonical component and reuse on user side. | — |
| WebSocket connection | Badge shows CONNECTED/DISCONNECTED/RECONNECTING. | Fine as transport indicator, but should not be labeled "LIVE". | Rename transport badge to "WS Connected" / "WS Retry" / "WS Off"; separate from feed-status badge. | P1 |
| Master Node / Windows Agent | ONLINE/OFFLINE badge based on `agentsStatus`. | Clear. | Keep; add last heartbeat age. | P2 |
| Signal engines | `AdminEngineCards` show LIVE/WAITING/STALE/ERROR per engine. | Good. | Ensure engine cards surface last evaluation time and evaluation count. | — |
| RT Engine | "Operational" if any engine has `running && evaluation_count > 0`. | Good evidence-based logic. | Surface engine count + last evaluation. | — |
| Control Plane / Database / WebSocket | Inline badges on Admin dashboard. | Duplicated inline classes. | Use canonical `StatusBadge`; add last check timestamp. | P2 |
| Market data source | Source label shown in Admin and User. | Good. | Also show last tick time and age. | P2 |
| Service health page | Real health checks with HEALTHY/DEGRADED/OFFLINE/UNKNOWN. | Good. | Use consistent badge vocabulary with dashboard. | — |
| Guest preview | Honest countdown + limited preview. | Good. | Ensure preview cannot be mistaken for live subscription data. | — |

---

# 6. SIGNAL EXPERIENCE AUDIT

## Standard Scalping
- **Visual differentiation:** Strategy name shown as text; no dedicated icon/color.
- **Data clarity:** Entry/SL/TP/Score/Prob/R:R present in `SignalCard`.
- **Freshness:** CreatedAt timestamp shown.
- **Current status:** Status field rendered.
- **No-trade UX:** "No directional signals. Market is quiet — this is correct behavior." — good honest message.
- **Confidence:** Calibrated probability shown when available.
- **Risk display:** SL and R:R present; no explicit position-size suggestion.
- **History:** Available in `/dashboard/signals`.
- **Explanation:** `ReasonCodes` shown only for candidates; no general explanation field.

## Ultra Scalping
- Same observations as Standard Scalping. No unique visual differentiation beyond strategy name.

## Standard Swing
- Same observations.

## Trend Swing
- Same observations.

**Cross-engine findings:**
1. No visual icon/brand differentiation between the four engines.
2. `SignalCard` background uses success/warning colors but does not include a strategy badge.
3. Candidate vs qualified distinction exists in `TradingMode` but not in `CompactSignalsView`.
4. Signal history table does not show which TP levels were hit or outcome.
5. No per-engine freshness indicator (e.g., "Standard Scalping last signal 14s ago").

---

# 7. SUBSCRIPTION & ENTITLEMENT UX

| Plan | Visibility | Finding |
|------|------------|---------|
| Free | Fallback when no active subscription | Hardcoded in `subscriptions.service.ts`: `selected_strategies: ['STANDARD_SWING']`, `max_signals_per_day: 3`. UX does not explain this gracefully on the live dashboard. |
| STANDARD / PRO / ELITE / ENTERPRISE | Plan metadata visible to all authenticated users | Higher-tier entitlements/referral rates are visible; this is transparent pricing but could be used for leakage concerns. |
| Trial | Supported status | Trial users get full active entitlements during trial. |

**Findings:**
1. **Frontend-only hiding risk:** The signals page filters by `allowed_strategies` but shows all strategy filter chips. Data leakage is mitigated because the backend returns only allowed signals, but the UX is confusing.
2. **No centralized entitlement hook:** `subscription-access.ts` has helpers, but entitlement checks are scattered (`signals/page.tsx`, `strategies/page.tsx`).
3. **No `canViewSignalCategory()` / `remainingSignalQuota()` helpers:** Prompt recommended these; the codebase does not have them as a single utility.
4. **Locked features:** Not implemented as explicit locked cards in the UI; unentitled users see empty/generic states instead.
5. **Upgrade prompts:** Not contextually tied to locked features.
6. **Payment failure / expiry:** Billing page handles status; no prominent banner on the live dashboard when subscription expires or payment fails.
7. **Automatic unlock after upgrade:** Entitlements refetched via React Query on interval; likely works, but no explicit success toast.

---

# 8. CROSS-PLATFORM & SYSTEMIC ISSUES

| Issue | Evidence | Severity | Recommended Fix |
|-------|----------|----------|-----------------|
| Component inconsistency | Two `StatusBadge` implementations + local inline badges. | P1 | Consolidate. |
| Duplicated UX | `DegradedBanner` defined in 8 admin pages. | P2 | Shared component. |
| Conflicting terminology | "Signal Panel" vs "Signals" vs "Signal Accuracy"; "Macro & News" vs "Macro Intelligence"; "Subscription" vs "Plans & Entitlements"; "MetaTrader Client" vs "MT Accounts". | P1 | Terminology standard (see section 12). |
| Inconsistent statuses | User "LIVE" from WS; Admin feed state from tick age. | P1 | Single `FeedStatus` component. |
| Responsive problems | Tables use `whitespace-nowrap` and `divide-neutral-800`; mobile horizontal scroll likely. | P1/P2 | Responsive table cards; theme-aware dividers. |
| Duplicated design logic | Mode tabs defined in both `page.tsx` and `command-center.tsx`. | P1 | Canonical command-center component. |
| Design-token problems | Dark-mode semantic surface tokens equal solid colors; light-mode `divide-neutral-800`. | P2 | Audit dark surfaces; replace neutral-800 divider. |
| Repeated subscription checks | Entitlement filtering in multiple pages. | P2 | Centralized `useEntitlements` hook. |
| Inconsistent error handling | Some pages show inline retry banners; others silently fail. | P2 | Standard `ErrorState` component with retry. |

---

# 9. ACCESSIBILITY AUDIT

**Practical WCAG 2.2 issues found:**

| Criterion / Issue | Severity | Evidence | Recommended Fix |
|-------------------|----------|----------|-----------------|
| Tabs lack `role="tab"`, `aria-selected`, `aria-controls`, keyboard arrow navigation. | P1 | `components/ui/tabs.tsx` lines 25–37. | Implement full tab semantics or use a headless tab library. |
| Sortable table headers are not focusable and have no `aria-sort`. | P1 | `components/ui/data-table.tsx` lines 89–101. | Use `<button>` with `aria-sort` and keyboard handler. |
| Many icon-only buttons lack `aria-label`. | P1 | Sidebar toggle is labeled, but other icon buttons (logout, expand row, refresh) are not consistently labeled. | Audit all icon-only controls. |
| Clickable table rows are not keyboard operable. | P1 | `signals/page.tsx` line 197. | Add focusable expand button. |
| Mobile sidebar has no focus trap. | P1 | `components/layout/sidebar.tsx`. | Trap focus while open; close on Escape. |
| Color-only communication for BUY/SELL/ERROR/LIVE. | P1 | Direction colors and status dots rely on color. | Add text labels/badges and icons; do not rely on color alone. |
| Focus-visible rings missing across many interactive elements. | P2 | Many buttons use `transition-colors` only. | Add `focus-visible:ring` tokens. |
| Form placeholders used as visible labels in some auth pages. | P2 | Login/reset inputs. | Add visible labels or floating labels. |
| No skip-to-content link. | P2 | Neither layout has one. | Add skip link for keyboard users. |
| No automated a11y testing. | P2 | No axe/Playwright a11y scans in test scripts. | Add `axe-core` or Playwright accessibility assertions. |
| Emoji icons in Growth mode have no text alternative. | P1 | `command-center.tsx` lines 434–437. | Replace with SVG/Tabler icons. |

---

# 10. RESPONSIVE AUDIT

| Breakpoint | Observation | Severity | Recommended Adaptation |
|------------|-------------|----------|------------------------|
| 1920×1080 | Works; multiple columns have room. | — | Good. |
| 1440×900 | Works; sidebar + 3-col cards fit. | — | Good. |
| 1366×768 | Sidebar takes meaningful width; dense tables begin to feel tight. | P2 | Add collapsible sidebar sections; compact mode. |
| 1024×768 (tablet landscape) | Tables with `whitespace-nowrap` will overflow; mode tabs wrap. | P2 | Convert dense tables to cards; allow tab wrapping gracefully. |
| 768×1024 (tablet portrait) | Sidebar becomes overlay; command-center 3-col becomes 1-col. | — | Acceptable. |
| 430/390/375px (mobile) | Tables horizontal-scroll; signal card grid becomes 4-col tiny text; mode tabs text may truncate. | P1 | Convert signal cards to stacked layout; convert tables to vertical cards; reduce mode tabs to icons-only or bottom nav. |

**Specific mobile recommendations:**
- Signal card: stack Entry/SL/TP/R:R vertically instead of 4-column grid.
- User signal history: render each signal as a card with expandable details.
- Admin tables: use row cards with primary metadata + "View" button.
- Sidebar: consider bottom tab bar on mobile for user dashboard (4–5 primary actions).

---

# 11. DESIGN SYSTEM RECOMMENDATIONS

## Recommended component hierarchy

```
components/
  ui/
    button.tsx          (existing — verify focus rings)
    input.tsx           (existing)
    select.tsx          (existing)
    tabs.tsx            (add ARIA/keyboard)
    status-badge.tsx    (canonical — merge admin + ui + local variants)
    feed-status.tsx     (NEW — LIVE/DEGRADED/STALE/REPLAY/UNKNOWN/UNKNOWN)
    data-table.tsx      (fix sort, ARIA, dividers, loading skeleton)
    card.tsx            (existing or NEW — consistent card shell)
    metric-card.tsx     (NEW or consolidate from admin stat-card)
    signal-card.tsx     (NEW — canonical signal display)
    empty-state.tsx     (canonical — icon, title, hint, CTA)
    error-state.tsx     (NEW — retry, context message)
    degraded-banner.tsx (NEW — replace 8 local copies)
    skeleton.tsx        (existing or improve)
    alert.tsx           (existing)
    toast.tsx           (existing)
  layout/
    app-shell.tsx
    sidebar.tsx         (add sections, accordion, focus trap)
    topbar.tsx
    footer.tsx
  command-center/
    command-center.tsx (canonical — remove duplicate page tabs)
    market-header.tsx
    signal-pipeline.tsx
    growth-panel.tsx
  signal/
    signal-evidence.tsx
  admin/
    engine-cards.tsx
    confirm-dialog.tsx
    stat-card.tsx       (merge into ui/metric-card or keep admin-specific)
```

## Components to consolidate
1. `components/ui/status-badge.tsx` + `components/admin/status-badge.tsx` → `components/ui/status-badge.tsx`.
2. Inline badge ternaries in admin dashboard → `StatusBadge`.
3. Local `StatusBadge` in macro-intelligence/indicator-monitor → `StatusBadge`.
4. `DegradedBanner` copies → `components/ui/degraded-banner.tsx`.
5. `EmptyState` in `tabs.tsx` + inline empty states → `components/ui/empty-state.tsx`.
6. Two command-center mode tab implementations → single canonical component.
7. `StatCard` and metric cards → one `MetricCard`.

## Do not refactor merely for purity
- `components/admin/engine-cards.tsx` can remain admin-specific.
- `components/trading/live-dashboard.tsx` should be deprecated/removed, not refactored.

---

# 12. TERMINOLOGY STANDARD

| Existing Terms | Recommended Canonical Term | Notes |
|----------------|----------------------------|-------|
| Signal / Trade Signal / Recommendation | **Signal** | Use consistently; "Recommendation" only in disclaimers. |
| Subscription / Package / Plan | **Plan** = product tier; **Subscription** = user's active instance | Do not use "Package" in UI. |
| Activation / Device / Terminal | **Device** = Windows Agent instance; **MT Account** = broker account; **Activation** = license-device binding | Disambiguate in Admin nav. |
| User / Client / Customer | **User** in app; **Customer** in finance/reports | Pick one per context. |
| Engine / Strategy / Signal Type | **Engine** = runtime process; **Strategy** = product category (Standard Scalping, etc.); **Signal Type** = BUY/SELL/NO-TRADE | Do not mix. |
| Live / Active / Running / Connected | **LIVE** = data feed freshness < threshold; **Active** = strategy/trading enabled; **Connected** = transport/agent online; **Running** = engine process evaluating | Define per component. |
| Signal Panel / Signals / Signal Accuracy | **Signals** = signal history; **Signal Monitor** = admin live panel; **Signal Accuracy** = aggregate historical stats | Rename Admin "Signal Panel" to "Signal Monitor" to avoid duplicating user "Signals". |
| Macro & News / Macro Intelligence | **Macro Calendar** = scheduled events; **Macro Intelligence** = confluence/scoring | Rename "Macro & News" to "Macro Calendar". |
| MetaTrader Client / MT4-MT5 Client | **MetaTrader Client** (user) / **MT Accounts** (admin) | Keep user label simple. |
| Commission Operations / Commission Control Center | **Commission Operations** (live); deprecate or merge "Commission Control Center" stub. | Remove duplicate commission page or rename to "Commission Audit". |
| Real-Time Console / Market Pulse / Command Center | User = **Live Command Center**; Admin = **Real-Time Console** | Align but keep distinct. |

---

# 13. DEAD / DUPLICATE / LEGACY UI

| File/Route/Component | Issue | Recommendation | Risk |
|----------------------|-------|--------------|------|
| `src/components/trading/live-dashboard.tsx` | Legacy dashboard with hardcoded placeholder prices and direct DOM ref mutation. | **REMOVE/DEPRECATE** from production routes; keep only if needed as a fixture with demo labels. | P0 trust risk if still routed. |
| `src/components/layout/auth-footer.tsx` | Only re-exports `Footer`; unused. | **REMOVE** or merge into `Footer`. | Low. |
| `src/components/admin/empty-state.tsx` | Not imported anywhere. | **REMOVE** or replace inline empty states. | Low. |
| `src/workers/marketDataWorker.ts` | Not referenced in any import search. | **VERIFY** usage; if unused, remove. | Low. |
| Public logo asset variants (predict-a-trade_*.png/svg) | Not referenced in code. | **AUDIT** asset kit; remove unused from `public/` or document as brand kit. | Low. |
| `src/proxy.ts` (no `middleware.ts`) | Auth middleware exists but is not wired. | **WIRE** as `middleware.ts` or remove if intentionally unused. | P1 security gap. |
| `src/components/user-dashboard/command-center.tsx` duplicate mode tabs vs `page.tsx` | Two implementations of the same modes. | **MERGE** into `command-center.tsx`; make page a thin wrapper. | P1 inconsistency. |
| `components/ui/status-badge.tsx` + `components/admin/status-badge.tsx` | Duplicate status badge. | **MERGE** into canonical `StatusBadge`. | P1 inconsistency. |
| `DegradedBanner` in 8 admin pages | Inline copy-paste component. | **MERGE** into shared `DegradedBanner`. | P2 maintenance. |
| `app/(admin)/admin/macro-news/page.tsx` | Disabled stub. | **DEPRECATE** or implement; show read-only state clearly. | P1 confusion. |
| `app/(admin)/admin/releases/page.tsx` | Disabled stub. | **DEPRECATE** or implement. | P1 confusion. |
| `app/(admin)/admin/backup-dr/page.tsx` | Disabled stub. | **DEPRECATE** or implement. | P1 confusion. |
| `app/(admin)/admin/broker-qualification/page.tsx` | Disabled stub. | **DEPRECATE** or implement. | P1 confusion. |
| `app/(admin)/admin/commission-control-center/page.tsx` | Duplicate/partial commission page. | **MERGE** with Commission Operations or remove. | P1 confusion. |

---

# 14. MOCK / HARDCODED / FAKE DATA FINDINGS

| Location | Data | Current Behavior | Risk | Required Fix |
|----------|------|-------------------|------|--------------|
| `frontend/src/components/trading/live-dashboard.tsx` lines 44, 51, 58 | `2500.00`, `2500.50`, `0.50` placeholder prices | Rendered as real bid/ask/spread before first WebSocket tick. | **Production risk** — user may trade on fake prices. | Remove component from production or render "—" / "Waiting for market data" until real tick. |
| `frontend/src/components/user-dashboard/command-center.tsx` line 235 | "LIVE" badge when `wsState === "CONNECTED"` | Shows live even if relay data is stale/replay. | **Production risk** — false liveness impression. | Use relay `FeedStatus` and tick age. |
| `frontend/src/app/(admin)/admin/macro-intelligence/page.tsx` lines 61–62 | Hardcoded `CONNECTED`/`HEALTHY` badge classes | Local status vocabulary. | Medium — inconsistent semantics. | Use canonical `StatusBadge`. |
| `frontend/src/components/admin/engine-cards.tsx` line 15 | Hardcoded `LIVE` badge style map | Local but consistent. | Low. | Keep if consolidated into `StatusBadge`. |
| `frontend/src/app/(user)/dashboard/backtest/page.tsx` | Default dates `2025-06-01` to `2025-06-30`, balance `$10,000` | Pre-populated form defaults. | Low — dev fixture, not live market data. | Acceptable as defaults; label clearly. |
| `frontend/src/app/terms/page.tsx` and other legal pages | Hardcoded last-updated dates, support email | Static compliance copy. | Low. | Maintain as part of release process. |
| `frontend/src/components/user-dashboard/command-center.tsx` lines 434–437 | Emoji icons `📊💰✅⏳` | Rendered as icons. | Low production risk but accessibility/credibility issue. | Replace with SVG/Tabler icons. |
| `control/src/modules/subscriptions/subscriptions.service.ts` | Hardcoded FREE fallback: `selected_strategies: ['STANDARD_SWING']`, `max_signals_per_day: 3` | Used when no active subscription. | Low — real policy fallback, but should be config-driven. | Move to `plans`/`entitlement-policy` config or document as policy. |
| `control/src/modules/device-auth/device-auth.service.ts` | Default device name `Windows Client`, agent version `1.0.0` | Defaults for new devices. | Low. | Acceptable defaults. |
| `control/src/modules/auth/auth.service.ts` | Hardcoded consent text/version | Compliance defaults. | Low. | Acceptable. |

**No evidence found of fabricated signal outcomes, fake commission/payout data, or simulated market volume in production code.** All financial/trading data observed is read from DB or Go engine.

---

# 15. PRIORITIZED ACTION PLAN

## PHASE 0 — PRODUCTION / TRUST BLOCKERS

| Priority | Recommendation | Dashboard | Impact | Effort | Dependency |
|----------|----------------|-----------|--------|--------|------------|
| P0 | Remove or replace `live-dashboard.tsx` hardcoded prices; never render as live. | User + Admin | Prevents fake-price trading decisions | S | None |
| P0 | Make User feed status use backend tick-age / relay `FeedStatus`, not just WS connected. | User | Prevents false "LIVE" on stale data | S | Go relay already emits status |
| P0 | Fix `parseInt(undefined)` causing `NaN` in Admin metric cards. | Admin | Prevents broken metric rendering | XS | None |
| P0 | Wire `middleware.ts` for server-side admin route protection. | Admin | Prevents SSR leakage of admin pages | M | `src/proxy.ts` exists |
| P0 | Fix `SignalCard` fallback color for unknown directions and null `StrategyID`. | User | Prevents crashes and misdirection | XS | None |

## QUICK WINS

| Priority | Recommendation | Dashboard | Impact | Effort | Dependency |
|----------|----------------|-----------|--------|--------|------------|
| QW1 | Replace emoji icons in Growth mode with Tabler icons. | User | Accessibility + credibility | XS | None |
| QW2 | Fix `DataTable` `divide-neutral-800` and signals table divider for light mode. | Admin + User | Visual polish + light-mode usability | XS | None |
| QW3 | Add visible focus rings to interactive elements. | Both | Accessibility | XS | None |
| QW4 | Fix signal history expanded row `colSpan`. | User | Rendering bug | XS | None |
| QW5 | Add keyboard-accessible expand button to signal rows. | User | Accessibility | XS | None |
| QW6 | Add entitlement-aware empty state to `/dashboard/signals`. | User | Conversion + clarity | S | None |
| QW7 | Hide/disable unentitled strategy filters on `/dashboard/signals`. | User | Reduces confusion | S | None |
| QW8 | Add last tick time/age to Admin and User market headers. | Both | Trust | XS | None |
| QW9 | Consolidate `DegradedBanner` into one shared component. | Admin | Maintenance | S | None |
| QW10 | Rename ambiguous Admin nav labels (Macro & News → Macro Calendar; Signal Panel → Signal Monitor). | Admin | Cognitive load | XS | None |

## STRATEGIC INITIATIVES

| Priority | Recommendation | Dashboard | Impact | Effort | Dependency |
|----------|----------------|-----------|--------|--------|------------|
| S1 | Consolidate all `StatusBadge` variants into one canonical component with `role="status"` and documented vocabulary. | Both | Consistency + maintainability | M | None |
| S2 | Build canonical `SignalCard` and use it across Trading mode, Signal History, Admin Signal Panel. | Both | Trust + consistency | M | None |
| S3 | Add centralized `useEntitlements` hook and locked-state components for unentitled features. | User | Conversion + prevents leakage confusion | M | Backend entitlements endpoint |
| S4 | Implement grouped/collapsible sidebar navigation for Admin and User. | Both | Information architecture | M | None |
| S5 | Add proper loading/error/empty states to all command-center mode panels. | User | Robustness | M | None |
| S6 | Improve `DataTable` with typed column sorting, ARIA, keyboard, and mobile card view. | Admin | Usability + accessibility | M | None |
| S7 | Deprecate/remove legacy `live-dashboard.tsx`; make `command-center.tsx` the canonical live view. | User | Reduces duplication and trust risk | M | None |
| S8 | Implement or clearly deprecate disabled Admin stub pages (releases, backup-dr, macro-news, broker-qualification). | Admin | Professionalism | M | Backend wiring |
| S9 | Add usage quota counter and subscription-status banner to user live dashboard. | User | Conversion + retention | M | Backend quota endpoint |
| S10 | Add skip-link, focus trap, and automated a11y testing. | Both | Accessibility compliance | M | None |

## NICE-TO-HAVES

| Priority | Recommendation | Dashboard | Impact | Effort | Dependency |
|----------|----------------|-----------|--------|--------|------------|
| N1 | Dark-mode semantic surface token audit/fix. | Both | Visual polish | S | None |
| N2 | Add page-transition focus management. | Both | Accessibility | S | None |
| N3 | Reduce polling frequency or consolidate dashboard queries. | Admin | Performance | M | Backend |
| N4 | Add per-engine last-signal timestamp in engine cards. | Admin | Operational awareness | S | None |
| N5 | Convert mobile tables to card lists. | Both | Mobile UX | M | None |
| N6 | Add subtle left-accent borders to critical status cards. | Both | Visual hierarchy | XS | None |
| N7 | Add contextual deep links in user drawer to subscriptions/licenses/devices. | Admin | Workflow efficiency | S | None |
| N8 | Add signal-alerts/opt-in notifications (after trust issues resolved). | User | Retention | L | Notification backend |

---

# 16. RECOMMENDED INFORMATION ARCHITECTURE

## ADMIN DASHBOARD

```
Real-Time Console          /admin/dashboard
Signals                      /admin/signals
│
Trading Intelligence
├── Indicator Monitor       /admin/indicator-monitor
├── Strategy Control        /admin/strategies
├── Regime Diagnostics      /admin/regime-diagnostics
├── Scoring Board           /admin/scoring-board
├── Signal Accuracy         /admin/signal-accuracy
│
Risk & Execution
├── Risk Center             /admin/risk-center
├── MT Accounts             /admin/mt-accounts
├── Device Auth             /admin/device-auth
├── License Management      /admin/licenses
│
Customers & Revenue
├── Users                   /admin/users
├── Subscriptions           /admin/subscriptions
├── Plans & Entitlements    /admin/plans-entitlements
├── Billing & Invoices      /admin/billing
├── Referrals               /admin/referrals
│
Finance Operations
├── Commissions             /admin/commission-operations
├── Payouts                 /admin/payout-operations
├── Finance Reports         /admin/finance-referral-reports
│
Market & Intelligence
├── Market Data             /admin/market-data
├── Macro Calendar          /admin/macro-news  (rename)
├── Macro Intelligence      /admin/macro-intelligence
├── AI Providers            /admin/ai-providers
├── Broker Qualification    /admin/broker-qualification
│
Platform & System
├── Feature Flags           /admin/feature-flags
├── Trading Reports         /admin/trading-reports
├── Backtesting             /admin/backtesting
├── Platform Operations     /admin/operations
├── System Health           /admin/health
├── Logs & Audit            /admin/logs
├── Settings                /admin/settings
```

**Rationale:**
- Collapses 31 items into 7 top-level groups.
- Renames ambiguous "Macro & News" → "Macro Calendar" and "Signal Panel" → "Signals" (under Real-Time Console).
- Merges duplicate commission pages under Finance Operations.
- Moves "Signal Accuracy" under Trading Intelligence (it is operational analytics).

## USER DASHBOARD

```
Live Command Center         /dashboard/live
Signals                     /dashboard/signals
Signal Accuracy             /dashboard/signal-accuracy
│
Trading
├── Strategy Preferences    /dashboard/strategies
├── MetaTrader Client       /dashboard/mt4-mt5-client
├── Trading Reports         /dashboard/trading-reports
├── Backtest                /dashboard/backtest
│
Growth
├── Referral & Earnings     /dashboard/referrals
│
Account
├── Billing & Subscription  /dashboard/billing
├── Payouts                 /dashboard/payouts
├── License                 /dashboard/license
├── Security                /dashboard/security
├── Activity Log            /dashboard/activity-log
├── Notifications           /dashboard/notifications
├── Settings                /dashboard/settings
│
Help
├── Support                 /dashboard/support
```

**Rationale:**
- Groups 16 flat items into 5 logical sections.
- Keeps signal-related items at the top.
- Separates Trading, Growth, Account, and Help so users can scan faster.

---

# 17. RECOMMENDED ADMIN HOMEPAGE

### Row 1 — Critical Operational Status (full width, always visible)
- Compact status strip: Trading (Active/Halted), Signals (Active/Paused), Market Feed (LIVE/DEGRADED/STALE/OFFLINE with last tick age), Master Node (ONLINE/OFFLINE), RT Engine (Operational/Unknown), Control Plane, Database, WebSocket.
- Use canonical `StatusBadge` and `FeedStatus` components.
- Click each item to deep-link to detail page.

### Row 2 — Live Market Data + Master Node + Service Health (3-column)
- XAUUSD Bid/Ask/Spread card with source, session, regime, last tick timestamp.
- Master Node / Windows Agents card (connected count, snapshots, last heartbeat).
- Service Health card (RT Engine, Control Plane, DB, WebSocket).

### Row 3 — Four Strategy Engine Cards (full width)
- `AdminEngineCards` showing LIVE/WAITING/STALE/ERROR per engine with last evaluation count and timestamp.

### Row 4 — Key Operational Metrics (6-column)
- Total Users, Active Subscriptions, MRR, Commissions Confirmed, Payouts Pending, Connected Agents.
- Guard against `NaN`; show "—" for missing data.

### Row 5 — Live Signal Pipeline + Active Strategies (2-column)
- Recent directional signals from WS/REST with strategy/probability/timestamp.
- Active/inactive strategy list.

### Row 6 — Alerts & Operations State (full width)
- Trading halt/pause controls, last operations update, signals today count (date-filtered).
- Pinned operational alerts if any.

**Why:** Operators need system health first, then market data, then engine health, then business metrics, then signals. This hierarchy matches incident-response priority.

---

# 18. RECOMMENDED USER HOMEPAGE

### Row 1 — Global Market Header (full width, always visible)
- XAUUSD symbol, Bid/Ask/Spread, Regime, Session, ATR/ADX/RSI summary.
- Honest `FeedStatus` badge: LIVE / DEGRADED / STALE / REPLAY / UNKNOWN with last tick age.
- Transport indicator separate: WS Connected / Reconnecting / Off.

### Row 2 — Primary Signal Card (prominent, full width when signal exists)
- Direction: BUY / SELL / WAIT / NO-TRADE (large, with icon + text — never color-only).
- Entry, Stop Loss, TP1, TP2, TP3.
- Confidence (calibrated probability) + Signal Score.
- Strategy category, timeframe, session, regime.
- Generated at / Expires at / Freshness: "Updated 14s ago".
- Signal status + execution badge.
- Short explanation / reason codes.

### Row 3 — Signal Pipeline / Additional Signals (if multiple active)
- Compact list of other directional signals; candidates separated.

### Row 4 — Subscription / Quota Status (when relevant)
- Current plan, renewal date, signals used/remaining, locked features, upgrade CTA.
- Prominent banner when quota reached or subscription expired/payment failed.

### Row 5 — Mode-Specific Content (Market / Trading / Growth / Command Center)
- Default to **Trading** mode for fastest decision; allow switching to Market/Growth/Command Center.

### Row 6 — Data Provenance Footer
- "Server-authoritative data from Go engine · Source: {source} · Last tick: {time} UTC ({age}s ago)."

**Why:** The user's primary questions are answered within seconds: What is Gold doing? Is there a valid signal? What are Entry/SL/TP? How fresh is it? What do I do next?

---

# 19. TOP 20 MOST IMPORTANT CHANGES

1. **Remove/replace hardcoded placeholder prices in `live-dashboard.tsx`.**
2. **Unify feed-status logic across User and Admin dashboards using relay `FeedStatus` + tick age.**
3. **Fix `NaN` risk in Admin metric cards.**
4. **Wire `middleware.ts` for server-side admin route protection.**
5. **Consolidate all `StatusBadge` variants into one canonical component.**
6. **Build a canonical `SignalCard` used everywhere.**
7. **Add entitlement-aware empty states and locked modules for unentitled users.**
8. **Hide/disable strategy filters that the user is not entitled to.**
9. **Replace emoji icons in Growth mode with accessible SVG/Tabler icons.**
10. **Fix `DataTable` sorting, ARIA, and keyboard accessibility.**
11. **Fix light-mode table divider color (`divide-neutral-800`).**
12. **Fix `SignalCard` unknown-direction fallback and null `StrategyID`.**
13. **Fix signal history expanded row `colSpan` and keyboard expand.**
14. **Group Admin sidebar into collapsible sections and rename ambiguous labels.**
15. **Group User sidebar into Trading/Growth/Account/Help sections.**
16. **Add loading/error/empty states to all command-center mode panels.**
17. **Consolidate duplicate mode tabs — make `command-center.tsx` canonical.**
18. **Implement or clearly deprecate disabled Admin stub pages.**
19. **Add usage quota and subscription-status banner to user live dashboard.**
20. **Add focus-visible rings, skip link, and automated a11y testing.**

---

# 20. FINAL VERDICT

### Current Admin Dashboard Readiness
**CONDITIONAL**

The Admin dashboard is structurally comprehensive and has honest data-state logic, but it is not production-ready without resolving: hardcoded legacy prices (if any route still uses them), `NaN` metric risk, server-side route protection, duplicate status components, overloaded navigation, and disabled stub pages. Once those are fixed, it is ready for internal operations.

### Current User Dashboard Readiness
**CONDITIONAL**

The User dashboard has a strong command-center concept and honest no-trade states, but it needs the trust blockers (feed-status consistency, placeholder prices, misleading empty states) fixed, plus a canonical signal card and better entitlement UX before customer-facing launch.

### Biggest Production Risk
**False liveness / stale or placeholder prices shown as live market data.** This can lead to users making trading decisions on data that is not genuine or current.

### Biggest UX Risk
**Information overload and inconsistent components** causing users and operators to misread status, miss controls, or lose trust in the interface.

### Biggest Revenue Opportunity
**Contextual, value-focused upgrade prompts** when a free/low-tier user encounters a locked strategy, empty signal list due to entitlement, or quota limit.

### Biggest Trust Opportunity
**Transparent data provenance and freshness** — explicitly showing source, last tick time, engine status, and why a signal is or is not present.

### Recommended First Implementation Sprint
**Sprint: "Trust + Signal UX Foundation"**

1. Remove/replace `live-dashboard.tsx` placeholder prices.
2. Unify `FeedStatus` component across User/Admin.
3. Fix Admin metric-card `NaN` and wire `middleware.ts`.
4. Consolidate `StatusBadge` and `DegradedBanner`.
5. Build canonical `SignalCard` and apply to User Trading mode + Signal History + Admin Signal Panel.
6. Add entitlement-aware empty states and locked strategy filters.
7. Replace Growth-mode emoji icons.
8. Group sidebars and rename ambiguous Admin labels.
9. Fix table sorting/dividers/ARIA.
10. Add automated a11y scan to CI.

---

# Evidence & Traceability

## Files examined (frontend)
- `/srv/predictatrade/xauusd/frontend/src/app/layout.tsx`
- `/srv/predictatrade/xauusd/frontend/src/app/(user)/dashboard/layout.tsx`
- `/srv/predictatrade/xauusd/frontend/src/app/(admin)/admin/layout.tsx`
- `/srv/predictatrade/xauusd/frontend/src/app/(user)/dashboard/live/page.tsx`
- `/srv/predictatrade/xauusd/frontend/src/components/user-dashboard/command-center.tsx`
- `/srv/predictatrade/xauusd/frontend/src/app/(user)/dashboard/signals/page.tsx`
- `/srv/predictatrade/xauusd/frontend/src/app/(admin)/admin/dashboard/page.tsx`
- `/srv/predictatrade/xauusd/frontend/src/components/trading/live-dashboard.tsx`
- `/srv/predictatrade/xauusd/frontend/src/components/layout/sidebar.tsx`
- `/srv/predictatrade/xauusd/frontend/src/components/layout/topbar.tsx`
- `/srv/predictatrade/xauusd/frontend/src/components/ui/status-badge.tsx`
- `/srv/predictatrade/xauusd/frontend/src/components/admin/status-badge.tsx`
- `/srv/predictatrade/xauusd/frontend/src/components/ui/data-table.tsx`
- `/srv/predictatrade/xauusd/frontend/src/components/ui/tabs.tsx`
- `/srv/predictatrade/xauusd/frontend/src/config/navigation/admin-navigation.ts`
- `/srv/predictatrade/xauusd/frontend/src/config/navigation/user-navigation.ts`
- `/srv/predictatrade/xauusd/frontend/src/styles/globals.css`
- `/srv/predictatrade/xauusd/frontend/src/tailwind.config.ts`
- `/srv/predictatrade/xauusd/frontend/src/lib/websocket.ts`
- `/srv/predictatrade/xauusd/frontend/src/lib/subscription-access.ts`
- `/srv/predictatrade/xauusd/frontend/src/providers/auth-provider.tsx`
- Plus additional admin/user pages and components discovered during subagent exploration.

## Files examined (control plane)
- `/srv/predictatrade/xauusd/control/openapi.json`
- `/srv/predictatrade/xauusd/control/src/common/guards/admin.guard.ts`
- `/srv/predictatrade/xauusd/control/src/common/guards/jwt-auth.guard.ts`
- `/srv/predictatrade/xauusd/control/src/modules/subscriptions/subscriptions.service.ts`
- `/srv/predictatrade/xauusd/control/src/modules/subscriptions/entitlement-policy.ts`
- `/srv/predictatrade/xauusd/control/src/modules/users/users.controller.ts`
- `/srv/predictatrade/xauusd/control/src/modules/billing/billing.controller.ts`
- `/srv/predictatrade/xauusd/control/src/modules/referrals/referrals.controller.ts`
- `/srv/predictatrade/xauusd/control/src/modules/commissions/commissions.controller.ts`
- `/srv/predictatrade/xauusd/control/src/modules/payouts/payouts.controller.ts`
- `/srv/predictatrade/xauusd/control/src/modules/licensing/licensing.controller.ts`
- `/srv/predictatrade/xauusd/control/src/modules/device-auth/device-auth.controller.ts`
- `/srv/predictatrade/xauusd/control/src/modules/admin/admin.controller.ts`
- `/srv/predictatrade/xauusd/control/src/modules/operations/operations.controller.ts`
- `/srv/predictatrade/xauusd/control/src/modules/health/health.controller.ts`
- Plus supporting service files.

## Tests / checks run
- Subagent frontend exploration: route inventory, component inventory, mock/fake data scan, duplicate/dead UI scan.
- Subagent design-system/state exploration: detailed read of tokens, `StatusBadge`, `DataTable`, `Tabs`, navigation, Admin/User homepages, sidebar, topbar.
- Subagent control-plane exploration: endpoint inventory, role/permission review, entitlement review, leakage assessment, mock/hardcoded data scan.
- Grep scans for `Math.random`, `fake`, `mock`, `placeholder`, `LIVE`, `ONLINE`, `HEALTHY`, `ACTIVE`, `CONNECTED`.
- Direct reads of key files listed above with line-number evidence.

## Unresolved risks / blockers
1. The actual production routing status of `live-dashboard.tsx` is not 100% confirmed from code alone; it may be used in `/dashboard` redirects or preview. If still routed, it is a P0 fix.
2. Whether `src/proxy.ts` is intentionally not wired as middleware (e.g., handled by nginx) needs confirmation before implementing `middleware.ts`.
3. The exact behavior of the Go relay `FeedStatus` field needs runtime verification; the frontend code references it but user command-center does not consume it yet.
4. No automated UI/E2E tests were executed during this audit; recommendations should be validated with Playwright tests.

## Rollback
- All recommendations are additive or component-level; no database migrations required for Phase 0 / Quick Wins.
- Legacy `live-dashboard.tsx` removal can be reverted by restoring the file.
- `middleware.ts` addition can be disabled by renaming the file.
- Component consolidation should be done behind feature flags or in small PRs to isolate regressions.

---

*End of UX/UI Audit Report.*
