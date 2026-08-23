# UI Audit Report — Admin & User Dashboards vs SOW

Date: 2026-08-23
Scope: Next.js frontend (`frontend/src/app/(admin)/admin/*` and `frontend/src/app/(user)/dashboard/*`)
Compared against: `docs/Predict-A-Trade_FINAL_SCOPE_OF_WORK.md` (SOW §32 User Dashboard, §33 Admin Dashboard, §69 commercial system, §127.2 Frontend Completion, §109/§110 Acceptance Criteria)

Legend: **REAL** = functional with API wiring; **PARTIAL** = functional but missing material SOW capability; **STUB** = placeholder; **EMPTY** = blank.

---

## 1. Admin Pages — Status

| Page | Status | Notes |
|------|--------|-------|
| `dashboard` | REAL | Status strip, XAUUSD market, master-node/agents, service health, metric cards, live WS signal pipeline, active strategies. |
| `users` | PARTIAL | List + suspend/activate + detail drawer. Missing: Create/Invite, Reset MFA, End Sessions, Assign Role, Assign Plan, Affiliate Hold. |
| `billing` | PARTIAL | Tabs: subscriptions/commissions/payouts; "Approve" payout only. Missing: invoices/refunds/chargebacks/coupons, full payout workflow. |
| `settings` | REAL (self) | Profile/password/MFA/notifications/accessibility. Notifications tab is a stub. |
| `operations` | PARTIAL | Halt/Resume Trading, Pause/Resume Signals, read-only AI models. Missing Risk Center capabilities. |
| `indicator-monitor` | REAL | Liveness/performance matrices + charts. |
| `activations` | PARTIAL | Read-only list + revoke. Missing: Reset Device, Force Upgrade, Disable Signal Access. |
| `health` | REAL | Postgres/Control-Plane/Go/Valkey/Windows-Agent health. |
| `indicators` | REAL | Live indicator values from `/market/snapshot`. |
| `licenses` | PARTIAL | List only. Missing full CRUD (create/suspend/revoke/renew/reset/force-logout/activation-history). |
| `logs` | PARTIAL | Read-only audit list. Missing search by actor/action/entity/IP/old-new-state/reason. |
| `referrals` | PARTIAL | Commission ledger + payouts + summary + plan base-rate (read-only). Missing rule config, downline tree, ops. |
| `regime-diagnostics` | REAL | Regime state-machine diagnostics. |
| `scoring-board` | REAL | Market state, per-strategy scoring, 12 hard-gate panel. |
| `signals` | REAL | Strategy/direction filters, expandable diagnostics (evidence, gate results, reason codes, entry/SL/TP). |
| `backtesting` | REAL | Wraps shared `backtest-panel.tsx`. |
| `strategies` | PARTIAL | Enable/disable 4 strategies + platform state. Missing: Version, Backtest, Walk-Forward, Shadow, Production Status, Feature Flags, Rollout %, Plan Availability. |
| `subscriptions` | PARTIAL | List only. Missing invoices/payments/refunds/chargebacks, coupons, provider references. |
| `trading-reports` | REAL | Signal stats by direction/strategy/regime/session, recent signals, gate vetoes. Signal-only (no finance reconciliation). |
| `device-auth` | PARTIAL | Read-only connections + registered devices. Missing device actions. |

### Admin routes that DO NOT exist (SOW-required)
- **Plans & Entitlements** management (CRUD).
- **Commission Control Center** (rule configuration).
- **Commission Operations** (hold/release/reverse/adjust/export).
- **Payout Operations** full workflow (review/reject/process/reconcile/retry/cancel/export).
- **Risk Center** (kill switches + limits).
- **MT Accounts** dedicated page.
- **AI Providers** management.
- **Market Data** feed-health page (divergence, tick rate, latency, candle health, backfill).
- **Macro / News** (DXY, yields, calendar, news feed, blackouts, provider health).
- **Finance / Referral Reports** (MRR/churn, setup-fee revenue, commission by plan/level/cycle, retention).
- **Releases** (client releases, checksums, signatures, rollback).
- **Backup/DR status**, **Feature flags/configuration**, **Broker execution qualification**.

---

## 2. User Pages — Status

| Page | Status | Notes |
|------|--------|-------|
| `live` | REAL | Command center: MARKET/TRADING/GROWTH/COMMAND_CENTER modes. |
| `trading-reports` | REAL | Connection status, account cards, terminals, strategy performance, commissions. |
| `mt4-mt5-client` | REAL | License info, connection, devices/terminals, downloads, install wizard. |
| `settings` | PARTIAL/STUB | **Only AccessibilitySettings.** Missing: password, MFA, sessions/trusted devices, login history, notifications, support. |
| `signals` | PARTIAL | History table + WS live. Missing: filters + explainability (evidence, reason codes, pillar contributions, AI verification, risk decision). |
| `billing` | PARTIAL/REAL | Plans + select-plan + invoices. Missing: cancel state, upgrade/downgrade clarity, next-billing/auto-renew. |
| `strategies` | PARTIAL | Display-only (allowed/selected/locked). No mutation to change selection via server entitlement. |
| `referrals` | PARTIAL | Code/link + copy, stats grid, commission history. Missing: L1–L5 network, earnings-by-level, payout request/history. |
| `backtest` | REAL | Wraps shared `backtest-panel.tsx`. |

### User routes that DO NOT exist (SOW-required)
- **Security / Sessions / MFA** page.
- **Notifications** config.
- **Support / tickets**.
- **Payouts** request + history.
- **License detail** dedicated page (beyond snippet in `mt4-mt5-client`).

---

## 3. Top Missing Points (prioritized)

**A. Admin commercial/operations command center incomplete (highest impact).**
- No Plans & Entitlements CRUD.
- No Commission Control Center (rule config) or Commission Operations (hold/release/reverse/adjust/export).
- No full Payout Operations workflow beyond "Approve".
- No Finance / Referral Reports (commercial reconciliation).

**B. Risk Center not implemented as a page.**
- Only global Halt + Pause Signals. Missing strategy/account/broker/symbol kill switches, exposure/spread/slippage/drawdown/daily-loss limits, session & news-blackout config.

**C. Licenses & Devices read-only on admin side.**
- No management actions (create/suspend/revoke/renew/reset/force-logout).

**D. Referral network visualization absent on both sides.**
- No sponsor chain / L1–L5 downline tree, network stats, conversion funnels, payout request/history (user).

**E. User strategy selection display-only; no License detail page.**
- No mutation to change selection; no dedicated License page.

**F. User explainability + signal filters missing.**
- `signals` is a bare table; no evidence/reason codes/pillar contributions/AI verification or filters.

**G. User Security / Notifications / Support / Payouts pages missing.**
- `settings` is accessibility-only.

**H. Several SOW admin sections have no route.**
- AI Providers, Market Data feed-health, Macro/News, Releases, Backup/DR, Feature Flags, Broker Qualification, MT Accounts.

**I. Audit log read-only with no search.**
- Missing search by actor/action/entity/timestamp/IP/old-new-state/reason + coverage of pricing/referral/sponsor/commission/payout/refund/affiliate-hold.

**J. Admin subscriptions & billing lack financial detail.**
- No invoices/refunds/chargebacks/coupons/provider references/billing-cycle numbers.

---

## 4. Summary Verdict
- **Admin:** Most named pages REAL/PARTIAL. Dominant gap = missing dedicated admin pages (Plans/Entitlements, Commission Control Center, Commission Operations, full Payout Operations, Risk Center, MT Accounts, AI Providers, Market Data, Macro/News, Finance/Referral Reports, Releases, Backup/DR, Feature Flags, Broker Qualification) plus read-only Licenses/Devices/Subscriptions.
- **User:** Pages largely REAL/PARTIAL, but Security/MFA/Sessions, Notifications, Support, and Payouts have no page; `settings` is accessibility-only; `strategies` display-only; `signals` lacks explainability/filters.

## 5. Recommended Priority Order
1. **User Security/MFA/Sessions page** + **Payouts page** (blocks §109 acceptance).
2. **Admin Risk Center** + **full Payout/Commission Operations workflow** (blocks §110 acceptance).
3. **Plans/Entitlements mgmt** + **Referral network tree (both sides)**.
