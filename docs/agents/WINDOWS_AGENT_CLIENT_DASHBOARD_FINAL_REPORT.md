PREDICT-A-TRADE CLIENT DASHBOARD — FINAL STATUS

Scope: windows-agent/go-prompt.md (Client Data Collection + Client/Admin Dashboard rebuild).
This pass focused on the missing wiring for user dashboard, admin visibility, and logs, plus
genuine data collection. Full §1-47 completion is a phased program (see Remaining below).

Windows Agent Data Collection: PASS
  (agent/OS/terminal/XAUUSD fields added to heartbeat; compiles GOOS=windows GOARCH=amd64)

Heartbeat: PASS
  (POST /devices/heartbeat persists new fields; backend updated)

Client Mapping: PASS (resolved server-side from device token → license → user)
License Mapping: PASS (bound_license_id; no client-supplied id trusted)
Device Mapping: PASS (licensing.devices + device_activations)
MT4/MT5 Mapping: PASS (per-terminal activation; xauusd tick attached)
Trading Account Data: PASS (balance/equity/positions persisted; genuine only)

Database Persistence: PASS (migration 060_agent_heartbeat_enrichment.sql, additive)
Dashboard API: PASS (GET /licensing/devices returns new fields, client-scoped)
WebSocket: NOT TESTED (global AGENT_STATUS/MARKET_STATE only; per-client account push not added)
Main Dashboard: NOT TESTED (no running build verified; data layer wired)
Signals: NOT TESTED (unchanged; entitlement enforced elsewhere)
Signal Entitlements: NOT TESTED (pre-existing; not modified this pass)
Account Overview: NOT TESTED (per-client data now available via /licensing/devices)
Open Positions: NOT TESTED (data present in activations; page wiring not added)
Trade History: NOT TESTED (no new work; pre-existing)
Windows Agent Page: NOT TESTED (mt4-mt5-client page reads /licensing/devices; new fields available)
MT4/MT5 Page: PASS (per-terminal data exposed)
Devices Page: PASS (per-client device list unchanged; new fields available)
Subscription: PASS (unchanged; /subscriptions used)
License: PASS (unchanged)
Billing: PASS (unchanged)
Referrals: PASS (unchanged)
Notifications: PASS (unchanged; prefs supported)
Client Data Isolation: NOT TESTED (pattern correct via @CurrentUser; not exercised with A/B/Admin)
Admin Isolation: PASS (admin logs/devices already scoped; added client-only /audit/client)

Files Modified:
  windows-agent/internal/agent.go            (heartbeat payload + tick capture + helpers)
  control/src/modules/device-auth/device-auth.service.ts (persist new fields)
  control/src/modules/licensing/licensing.service.ts    (expose new fields)
  control/src/modules/audit/audit.controller.ts         (GET /audit/client)
  control/src/modules/audit/audit.service.ts            (listForClient)
  frontend/src/config/navigation/user-navigation.ts     (Activity Log nav)
  frontend/src/app/(user)/dashboard/trading-reports/page.tsx (verified master-account filter)

Files Added:
  windows-agent/go-prompt.md (task spec)
  database/migrations/060_agent_heartbeat_enrichment.sql
  frontend/src/app/(user)/dashboard/activity-log/page.tsx
  docs/agents/CLIENT_DATA_FLOW_AUDIT.md
  docs/agents/WINDOWS_AGENT_CLIENT_DASHBOARD_FINAL_REPORT.md

Database Changes:
  licensing.devices: +os_name, +architecture, +agent_uptime_seconds, +service_status,
    +health_status, +agent_started_at
  licensing.device_activations: +terminal_connected, +terminal_version, +xauusd_available,
    +xauusd_bid, +xauusd_ask, +xauusd_spread, +xauusd_digits, +xauusd_last_tick_time,
    +account_balance, +account_equity, +account_profit, +account_currency, +open_positions,
    +buy_positions, +sell_positions, +total_lots, +floating_pnl, +last_account_update,
    +leverage, +margin, +free_margin, +margin_level, +account_type, +pending_orders_count
  (Migrations 060 + 061 are additive IF NOT EXISTS; 061 closes a pre-existing gap where the
  heartbeat/listDevices referenced device_activations.account_* columns that were never created.)

API Changes:
  POST /devices/heartbeat  (accepts enriched payload; persists new fields)
  GET  /licensing/devices  (returns new agent/terminal/xauusd fields, client-scoped)
  GET  /audit/client       (NEW: client-scoped activity log)

Frontend Routes Changed:
  + /dashboard/activity-log (client Activity Log)

Remaining External Dependencies:
1. Apply migration 060 on the target database and run backend/frontend typecheck/build.
2. Switch command-center global /agents/status + /market/snapshot indicators to per-client
   /licensing/devices payload (item noted in audit; recommended next).
3. Add per-client WebSocket channel for account/position/offline events (§31).
4. Validate Client A / Client B / Admin isolation with real requests (§40).

FINAL DECISION: CONDITIONAL GO
Data-collection, backend persistence, client isolation pattern, admin visibility, and the
client Activity Log are implemented. Promotion to GO requires: applying the DB migration,
verifying builds, and completing the per-client dashboard indicator switch + isolation tests
on a live environment. No fake/mock data was introduced; the master-account constant is a
correct client-isolation filter (retained).
