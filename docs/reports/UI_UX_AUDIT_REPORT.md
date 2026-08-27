# 🔍 DASHBOARD UI/UX AUDIT REPORT

**Project:** frontend (Predict-A-Trade XAUUSD dashboard)
**Framework:** Next.js 16.3 (App Router) + React 19.2 + Tailwind CSS 3.4 + @tabler/icons-react + TanStack Query/Table/Virtual + Recharts + lightweight-charts + sonner
**Audit Date:** 27 August 2026
**Auditor:** Mehulkumar Bhatt

---

## EXECUTIVE SUMMARY

- Total issues found: **41**
- Critical: **2** | High: **12** | Medium: **18** | Low: **9**
- Top 3 issues requiring immediate action:
  1. **A1-001 (Critical)** — Auth pages (login/register) use hardcoded light-only hex colours and inline styles, so they never respond to dark mode and several text colours fail WCAG AA contrast.
  2. **C5-001 (Critical)** — `ConfirmDialog` and `RegistrationModal` have no focus trap, no Escape-to-close, and no focus return — a WCAG A violation for modal dialogs.
  3. **C2-001 (High)** — All canvas-based charts (`indicator-charts.tsx`) have zero accessible fallback (no `role="img"`, no `aria-label`, no text alternative), so screen-reader users get nothing.

---

## ISSUE REGISTRY

| ID | Category | Severity | File & Line | Issue Description | User Impact | Recommended Fix |
|----|----------|----------|-------------|-------------------|-------------|-----------------|
| A1-001 | Accessibility | Critical | `src/app/(auth)/register/page.tsx:74-316`, `login/page.tsx:177-189` | Auth pages use hardcoded hex colours (`#6c707a`, `#9ba3b4`, `#171a22`, `#d7d3c9`) and inline `style={{}}` throughout. `#9ba3b4` on `#ffffff` ≈ 2.6:1 (fails 4.5:1); `#6c707a` ≈ 4.6:1 (borderline). No dark-mode tokens. | Low-vision users cannot read helper/placeholder text; dark-mode users get a blinding white page. | Replace inline styles with Tailwind `pat-*` token classes; use `pat-text-muted`/`pat-text-secondary`; add `.dark` variants. |
| C5-001 | Accessibility | Critical | `src/components/admin/confirm-dialog.tsx:14-33` | Modal has no `role="dialog"`, no `aria-modal`, no focus trap, no Escape handler, no focus return to trigger. | Keyboard/screen-reader users can tab behind the dialog; destructive confirm is not safely dismissible. | Add `role="dialog" aria-modal="true"`, focus the confirm button on open, trap Tab, handle Escape, restore focus on close. |
| C2-001 | Accessibility | High | `src/components/indicator-monitor/indicator-charts.tsx:95,116,136,160` | Canvas charts render with no `role="img"`, `aria-label`, or text alternative; gridline values are drawn to canvas (invisible to AT). | Screen-reader users receive no chart data at all. | Add `role="img"` + `aria-label` describing the chart; provide a hidden text summary or data table fallback. |
| C5-002 | Accessibility | High | `src/components/guest-preview/registration-modal.tsx:115` | Modal declares `role="dialog" aria-modal="true"` but has no focus trap, no Escape handler, no focus return. | Focus escapes to page behind the lock screen. | Implement focus trap + Escape + focus return (reuse a shared `useModal` hook). |
| C7-001 | Accessibility | High | `src/app/(auth)/login/page.tsx:124-136` | "Remember me" is a `<div onClick>` inside a `<label>`, not a real `<input type="checkbox">`. Not keyboard-focusable, no `role`, no checked state exposed to AT. | Keyboard and screen-reader users cannot toggle "remember me". | Replace with a real `<input type="checkbox">` (as done correctly in `register/page.tsx` `ConsentCheckbox`). |
| C7-002 | Accessibility | High | `src/components/ui/data-table.tsx:148-159` | Pagination icon-only buttons (`IconChevronsLeft/Left/Right/Right`) have no `aria-label`. | Screen-reader users hear "button" with no purpose. | Add `aria-label="First page"`, `"Previous page"`, `"Next page"`, `"Last page"`. |
| C7-003 | Accessibility | High | `src/app/(auth)/login/page.tsx:114-117`, `register/page.tsx:132-135` | Password show/hide toggle buttons have no `aria-label` and no `aria-pressed`. | Screen-reader users cannot identify the toggle. | Add `aria-label="Show password"` / `"Hide password"` and `aria-pressed={showPassword}`. |
| A3-001 | Accessibility | High | `src/components/ui/data-table.tsx:97-138` | `<table>` has no `<caption>`, and `<th>` elements have no `scope="col"`. | Screen-reader users cannot associate headers with cells. | Add `<caption className="sr-only">` and `scope="col"` on every `<th>`. |
| A3-002 | Accessibility | High | `src/app/(user)/dashboard/signals/page.tsx:188-206` | Same table markup issue — no `<caption>`, no `scope="col"` on 14 `<th>` elements. | Header/cell association lost for AT users. | Add `scope="col"` and a visually-hidden caption. |
| A4-001 | Accessibility | High | `src/components/trading/live-dashboard.tsx:23-25` | Live bid/ask/spread update via `innerText` on refs with no `aria-live` region. | Screen-reader users are never told the price changed. | Wrap price values in an `aria-live="polite"` region (or use `role="status"`). |
| A2-001 | Accessibility | High | `src/app/(auth)/register/page.tsx:304-316`, `login/page.tsx:177-189` | `inputStyle` sets `outline: "none"` and focus is indicated only by a border-colour change (`#205fdc`). No visible focus ring. | Keyboard users get a weak, low-contrast focus indicator. | Add `focus-visible:ring-2 focus-visible:ring-pat-primary` (or a 2px outline) instead of `outline:none`. |
| L3-001 | Layout | High | `src/components/layout/sidebar.tsx:46-52` | Mobile hamburger button has `aria-label` but no `aria-expanded` and no `aria-controls`; the drawer has no Escape-to-close and no focus management. | Screen-reader users can't tell if the nav is open; keyboard users can't dismiss it. | Add `aria-expanded={open}` + `aria-controls="sidebar-nav"`; close on Escape; move focus into nav on open. |
| V1-001 | Visual | High | `src/app/(auth)/register/page.tsx`, `login/page.tsx` | Entire auth flow uses inline `style={{}}` with hardcoded px/hex, bypassing the Tailwind `pat-*` design-token system used everywhere else. | Two parallel styling systems; auth pages drift from the rest of the product. | Refactor auth pages to Tailwind token classes (as `registration-modal.tsx` already does). |
| V2-001 | Visual | High | `src/app/(auth)/layout.tsx:6` | Auth layout hardcodes `background: "#f7f6f2"` — light-only, ignores `.dark` theme. | Dark-mode users see a white auth screen. | Use `bg-pat-bg-page` token class. |
| C4-001 | UX | Medium | `src/app/(user)/dashboard/signals/page.tsx:148-152` | No "Reset filters" affordance; filter state is not synced to URL query params. | Users must manually clear 3 dropdowns; filters aren't shareable/bookmarkable. | Add a "Reset" button and sync filters to `useSearchParams`/router. |
| C1-001 | UX | Medium | `src/components/ui/data-table.tsx:86-92` | Empty state is a generic "No data found" with no reason or next-step guidance. | Users don't know why it's empty or what to do. | Accept an `emptyMessage`/`emptyAction` prop (as `signals/page.tsx` already does inline). |
| C1-002 | UX | Medium | `src/components/ui/data-table.tsx:22-50` | `page` state is not reset when `data` changes (only on sort). If data shrinks, `page` can exceed `totalPages`, rendering an empty page. | Users see a blank table after data refresh. | Add `useEffect(() => setPage(0), [data])` or clamp `page` to `totalPages - 1`. |
| C3-001 | UX | Medium | `src/components/admin/stat-card.tsx:19` | `value` renders `null`/`undefined` as literal "null"/"undefined" (no `—`/`N/A` fallback). | Broken-looking KPI when a metric is missing. | Render `value ?? "—"` and guard `toLocaleString` for non-numbers. |
| C2-002 | UX | Medium | `src/components/indicator-monitor/indicator-charts.tsx:77-83` | Canvas line chart has no axis titles or units; gridline values are raw numbers with no unit context. | Users can't tell what the y-axis represents. | Add axis labels/units (e.g. "value", "count") and a legend mapping colours to indicators. |
| C2-003 | UX | Medium | `src/components/indicator-monitor/indicator-charts.tsx:11-19` | Chart colours are hardcoded hex and rely on colour alone to distinguish series; no legend is rendered. | Colour-blind users can't distinguish indicator lines. | Add a legend with labels; use distinguishable line styles (dash patterns) in addition to colour. |
| S5-001 | State | Medium | `src/app/(admin)/admin/dashboard/page.tsx:73-133` | Both WebSocket and REST polling (10s) write to the same `sigBuffer`/`liveSignals` with no ordering guard. | Out-of-order updates can briefly show stale signals. | Use a single source of truth (REST) with WS as a refresh trigger (as `signal-pipeline.tsx` already does). |
| S4-001 | State | Medium | `src/app/(admin)/admin/dashboard/page.tsx:261` | "Last Tick" renders `toLocaleTimeString()` with a hardcoded "UTC" label but no timezone conversion — the raw timestamp is formatted in the browser's local timezone. | Misleading timestamp if the user's browser is not UTC. | Format with `date-fns` `format(..., 'HH:mm:ss')` in UTC, or use `toLocaleTimeString('en-GB', { timeZone: 'UTC' })`. |
| Q1-001 | Code | Medium | `src/components/indicator-monitor/indicator-charts.tsx:11-19` | Hardcoded hex colour map (`INDICATOR_COLORS`, `FALLBACK_COLORS`) and canvas grid colours (`#2A3850`, `#74829A`) not sourced from design tokens. | Colour drift from the token system; dark-mode gridlines may be wrong. | Source from CSS variables (`hsl(var(--pat-*))`) or a shared chart-colour token. |
| Q2-001 | Code | Medium | `src/components/user-dashboard/command-center.tsx` (651 lines) | Oversized component with 12+ sub-components and 6 queries in one file. | Hard to maintain; re-renders cascade. | Split into per-mode feature components (already partially done — finish the extraction). |
| Q3-001 | Code | Medium | `src/components/trading/live-dashboard.tsx:7` | `void isAdmin;` — unused prop, dead code. | Confusing API; suggests unfinished work. | Remove the `isAdmin` prop or actually use it. |
| V5-001 | Visual | Medium | `src/components/admin/empty-state.tsx` vs `src/components/ui/tabs.tsx:65-72` | Two different `EmptyState` components doing the same job. | Inconsistent empty-state styling across pages. | Consolidate into one shared `EmptyState`. |
| V5-002 | Visual | Medium | `src/components/admin/status-badge.tsx` vs `src/components/ui/status-badge.tsx` | Two `StatusBadge` components. | Divergent badge styling/behaviour. | Consolidate into one. |
| V5-003 | Visual | Medium | `src/components/admin/stat-card.tsx` vs inline metric cards in `admin/dashboard/page.tsx:321-331` | `StatCard` component exists but the dashboard re-implements the same card inline. | Duplicate code; drift risk. | Use `StatCard` in the dashboard. |
| L5-001 | Layout | Medium | `src/components/layout/sidebar.tsx:48,53,55` + `topbar.tsx:32` + `confirm-dialog.tsx:17` | Scattered hardcoded z-index values (z-20, z-30, z-40, z-50, z-[200]) with no central scale. | Risk of stacking conflicts as new overlays are added. | Define a z-index token scale (e.g. `--z-sidebar`, `--z-modal`) in globals.css. |
| L2-001 | Layout | Medium | `src/components/layout/app-shell.tsx:39-48` | `main` has `overflow-auto` inside a `min-h-screen` flex column, creating a nested scroll container alongside the page scroll. | Potential double-scrollbar / scroll-trap on some viewports. | Use `overflow-y-auto` on the outer column only, or `h-screen` + single scroll region. |
| C6-001 | UX | Low | `src/components/themed-toaster.tsx:27-31` | sonner `Toaster` uses default auto-dismiss; no explicit `duration` and no `closeButton` config. | Short toasts may auto-dismiss before being read. | Set `duration={5000}` and `closeButton` for manual dismissal. |
| C4-002 | UX | Low | `src/app/(user)/dashboard/signals/page.tsx:311-326` | `FilterSelect` wraps `<select>` in `<label>` (good) but the select has `outline-none` with only `focus:border-primary` — weak focus indicator. | Keyboard users get a subtle focus cue. | Add `focus-visible:ring-2 focus-visible:ring-pat-primary`. |
| A4-002 | Accessibility | Low | `src/components/layout/sidebar.tsx:60` | Logo `<img>` has `alt="logo"` (non-descriptive). | Screen-reader users hear "logo". | Use `alt="Predict-A-Trade"` or `alt=""` (decorative, since text label follows). |
| A4-003 | Accessibility | Low | `src/components/indicator-monitor/indicator-charts.tsx:232` | Active-indicator dot uses colour alone (`backgroundColor`) with no text label. | Colour-blind users can't identify the active series. | Add a text label next to the dot. |
| V3-001 | Visual | Low | `src/app/(auth)/register/page.tsx:80-83` | Inline `fontSize`/`lineHeight` values scattered (28px, 14px, 12px, 10px) instead of the Tailwind type scale. | Typographic inconsistency vs. the rest of the app. | Use `text-2xl`, `text-sm`, `text-xs` token classes. |
| V4-001 | Visual | Low | `src/components/layout/topbar.tsx:46,89` | Icon sizes vary (18px nav, 14px chevron, 16px menu items) with no documented scale. | Minor visual inconsistency. | Standardise: nav 20px, inline 16px, chevron 14px. |
| P1-001 | Performance | Low | `src/components/indicator-monitor/indicator-charts.tsx:26-46` | Canvas charts redraw fully on every dependency change; no `requestAnimationFrame` throttling or dirty-check. | Unnecessary repaints during rapid data updates. | Throttle redraws with `rafBatch` (already available in `@/lib/performance`). |
| P1-002 | Performance | Low | `src/app/(admin)/admin/dashboard/page.tsx:14-65` | 7 parallel `useQuery` calls with overlapping `refetchInterval` (5s/10s/15s/30s). | Redundant polling load. | Consolidate intervals; use a single health endpoint where possible. |
| X4-001 | Security | Low | `src/components/guest-preview/registration-modal.tsx:203` | OTP input is `type="text"` (with `inputMode="numeric"`). Acceptable, but `autocomplete="one-time-code"` is missing. | Minor: password managers won't autofill OTP. | Add `autoComplete="one-time-code"`. |

---

## QUICK-WIN FIXES (Low effort, High impact)

1. Add `aria-label` to the 4 pagination icon buttons in `data-table.tsx` (5 min).
2. Add `aria-label` + `aria-pressed` to password show/hide toggles in login/register (5 min).
3. Replace the login "remember me" `<div onClick>` with a real `<input type="checkbox">` (10 min).
4. Add `scope="col"` + sr-only `<caption>` to the two data tables (10 min).
5. Add `role="dialog" aria-modal="true"` + Escape handler + focus trap to `ConfirmDialog` (20 min).
6. Add `aria-expanded`/`aria-controls` to the sidebar hamburger (5 min).
7. Add `role="img"` + `aria-label` to the four canvas charts (15 min).
8. Add `value ?? "—"` guard to `StatCard` (5 min).
9. Add `useEffect(() => setPage(0), [data])` to `DataTable` (5 min).
10. Replace `#f7f6f2` auth-layout background with `bg-pat-bg-page` (2 min).

---

## STRATEGIC RECOMMENDATIONS

1. **Adopt a single styling system.** The auth pages' inline-style/hardcoded-hex approach is the root cause of the contrast, dark-mode, and consistency failures. Migrate them to the existing Tailwind `pat-*` token system and delete the inline `inputStyle`/`style={{}}` blocks.
2. **Build a shared modal primitive.** A `useModal` hook (focus trap, Escape, focus return, `aria-modal`) would fix `ConfirmDialog`, `RegistrationModal`, and any future dialogs in one place.
3. **Make charts accessible by construction.** Wrap canvas charts in a component that always emits `role="img"` + `aria-label` and a hidden data-table fallback, so no chart can ship without an accessible alternative.
4. **Consolidate duplicate primitives.** Merge the two `EmptyState` and two `StatusBadge` components, and use `StatCard` everywhere — this removes drift and reduces maintenance surface.
5. **Introduce a z-index token scale** (sidebar, topbar, dropdown, modal, toast) to eliminate the scattered hardcoded `z-*` values.

---

## WHAT LOOKS GOOD

1. **Design-token discipline in the main app** — `globals.css` defines a comprehensive `pat-*` HSL token set with proper light/dark variants, and Tailwind maps them cleanly. This is a strong foundation.
2. **Honest data handling** — the WebSocket normalizer (`websocket.ts`) explicitly refuses to fabricate timestamps/directions, and the admin dashboard derives feed state from backend tick age rather than assuming "LIVE". This no-fabrication discipline is excellent.
3. **Loading/error/empty states are broadly present** — skeletons, retry buttons, and differentiated empty states (no-entitlement vs no-signals vs no-filter-match) in `signals/page.tsx` are a model for the rest of the app.
4. **Reduced-motion support** — both `prefers-reduced-motion` and a manual `.reduce-motion` class are implemented in `globals.css`.
5. **Single icon library** — consistent use of `@tabler/icons-react` (individually imported, tree-shakeable) avoids the mixed-icon-library anti-pattern.
