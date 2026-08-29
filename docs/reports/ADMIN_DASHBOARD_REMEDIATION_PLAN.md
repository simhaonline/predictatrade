# Admin Dashboard Remediation Plan (from 23 screenshots, 30 Aug 2026)

## Catalogued issues
1. **User approval workflow missing** — users register and land ACTIVE
   immediately; requirement: PENDING by default, admin reviews on Users page
   (Approve / Reject / Suspend / role change) before login is allowed.
2. **Payments management absent in admin** — no page to configure/see gateway
   (NOWPayments/USDT) state, no payment approvals view. Only invoice/payout
   pages exist.
3. **AI providers page**: model activation is live but provider CRUD shows
   "Pending Backend / schema-only".
4. **Market Data → Feed Monitoring**: explicitly labeled "Pending Backend"
   (no feed health/divergence/tick-rate/latency/candle-health/backfill API).
5. **Releases page**: "Publish Release (Pending Backend)".
6. **MT Accounts page**: create is user-scoped; admin-wide listing depends on
   a future admin endpoint — page notes it honestly.
7. **Menu quality**: 35 flat items under 12 sections; duplicates in scope
   (e.g., Trading Reports vs Backtesting; Logs & Audit vs Signal Accuracy
   overlap no; real dups: Macro Calendar + Macro Intelligence vs Market Data
   groupings). Needs professional grouping + removal of duplicates.
8. **CSS/JS stale**: several pages report failure to load fresh bundles
   (cache) — add hashed asset cache-busting + service-worker disable-on-stale.
9. **Settings page shows "Failed to change password"** — backend PATCH/me
   supports displayName only; password change endpoint missing.
10. **MARNIE_FIB rebrand → EQFE (Equilibrium Fibonacci Engine)** across
    config, DB signal rows (display), strategies, EAs, docs.

## Prioritized fix order (this weekend batch)
- P0: menu restructure + user-approval workflow + payments config page
- P1: MARNIE_FIB → EQFE rebrand; honest-state pages wired or hidden
- P1: AI providers CRUD backend + feed-health backend endpoint
- P2: password-change endpoint for admin settings; CSS cache-busting
