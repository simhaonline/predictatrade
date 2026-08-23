# Predict-A-Trade — Client Data Flow Audit & Implementation

Spec source: `windows-agent/go-prompt.md` (Client Data Collection + Client/Admin Dashboard rebuild).
This document satisfies §1 (audit before modifying) and records the wiring implemented.

## CLIENT DATA FLOW AUDIT

| Area | Finding |
|------|---------|
| Windows Agent | `internal/agent.go:sendHeartbeatToBackend()` POSTs to `POST /devices/heartbeat` (NestJS `device-auth`). Payload carried `device_id, session_id, mt_connected, terminals[], hostname`. **Missing** per §3-8: `agent_version, agent_started_at, agent_uptime_seconds, os_name, architecture, service_status, health_status`, per-terminal `connected`, and XAUUSD `bid/ask/spread/last_tick_time`. |
| Authentication | Device access token (Bearer) from activation; backend resolves device → license → user. Client identity derived server-side from token. ✅ |
| License Mapping | `licensing.devices.bound_license_id` + `licensing.licenses.user_id`. ✅ |
| Device Mapping | `licensing.devices` keyed by `id`; `device_activations` per `mt_account_login`. ✅ |
| Trading Account Mapping | `device_activations.account_balance/equity/...` updated by heartbeat. ✅ |
| Heartbeat | `POST /devices/heartbeat` renews `session_leases`; returns independent connection/auth/license/device/session states. ✅ |
| Database | TimescaleDB/Postgres. `licensing.devices`, `licensing.device_activations` exist. **Gap**: missing columns for agent OS/uptime/health + per-terminal XAUUSD tick. |
| WebSocket | Go realtime gateway emits `AGENT_STATUS` + `MARKET_STATE`; frontend `lib/websocket.ts` consumes. Per-client account/position/offline push NOT delivered via WS (polled via REST). |
| Client Dashboard | Next.js `(user)/dashboard/*`. Fetches per-client `/licensing/devices`, `/subscriptions`, `/billing/invoices`, `/commissions`, `/referrals/code`. **Bug**: command-center "Windows Agent Connection" indicators polled GLOBAL `/agents/status` + `/market/snapshot` instead of the client's own agent/account. |
| Admin Dashboard | `(admin)/admin/*` already surfaces users, licenses, subscriptions, `device-auth`, activations, `mt-accounts`, referrals, billing, **audit/logs** (`admin/logs` → `GET /audit`). Largely intact. |
| Notifications | `notify.ps1` + `/licensing` notification prefs. ✅ |
| Audit/Event Logs | `audit.audit_events` table + admin `GET /audit`. **Gap**: no client-scoped endpoint/page (clients could not see their own events). |

## CHANGES IMPLEMENTED (this pass)

1. **Windows Agent heartbeat enrichment** (`internal/agent.go`): payload now includes
   `agent_version`, `agent_started_at` (RFC3339), `agent_uptime_seconds`, `os_name`
   (`runtime.GOOS`), `architecture` (`runtime.GOARCH`), `service_status`, `health_status`,
   per-terminal `connected`, and a genuine `xauusd` block (bid/ask/spread/last_tick_time)
   captured from real MT5 ticks (`onTickFromEA` stores latest XAUUSD tick). Compiles
   (`GOOS=windows GOARCH=amd64 go build` → exit 0).
2. **Backend persistence** (`control` `device-auth.service.ts:heartbeat`): stores the new
   agent-level fields into `licensing.devices` and terminal/XAUUSD fields into
   `licensing.device_activations`. Additive only.
 3. **DB migrations** (additive, `IF NOT EXISTS`, safe to re-run):
    - `060_agent_heartbeat_enrichment.sql`: `os_name, architecture, agent_uptime_seconds,
      service_status, health_status` on `licensing.devices`; `terminal_connected,
      terminal_version, xauusd_*` on `licensing.device_activations`.
    - `061_device_activation_account_columns.sql`: **fixes a real schema gap** — the existing
      `heartbeat` service and `listDevices` already referenced
      `device_activations.account_balance / account_equity / account_profit / account_currency
      / open_positions / ...` and `da.account_currency`, but no committed migration created
      those columns. Adds the full §6 trading-account column set plus `agent_started_at`.
 3b. **Heartbeat persistence hardened**: `device-auth.service.ts:heartbeat` now also persists
     `account_currency`, `leverage`, `margin`, `free_margin`, `margin_level`, `account_type`,
     `pending_orders_count` (genuine values only; null when the agent doesn't send them).
4. **Exposure** (`licensing.service.ts:listDevices`): returns the new columns incl. the
   `xauusd` object inside each activation, so the client dashboard shows the user's own
   agent/terminal/market state.
5. **Hardcoded master account**: `trading-reports/page.tsx` `MASTER_NODE_ACCOUNTS` is a
   *filter that hides the platform master node* from a user's own terminal list (line 107),
   i.e. correct client-isolation — left intact (not fake data).
6. **Client Activity Log (logs)**: new `GET /audit/client` (`audit.controller.ts` +
   `audit.service.ts:listForClient`) returns only events where `actor_id = current user`
   (no admin role). New user page `(user)/dashboard/activity-log/page.tsx` + nav item
   "Activity Log". Honors loading/empty/error states (§36).
7. **Docs**: this audit; `docs/agents/WINDOWS_AGENT.md` updated; `go-prompt.md` FINAL
   REPORT produced.

## NOT YET DONE (remaining, per §1 "implement only missing wiring")

- Switch the command-center "Windows Agent Connection" panel from GLOBAL `/agents/status` +
  `/market/snapshot` to the per-client `/licensing/devices` payload (§12/§32).
- Dedicated per-client WS channel for account/position/offline events (§31).
- Dedicated `/api/client/dashboard` aggregation endpoint (§34-35) — currently pages call
  individual `/licensing/*` endpoints (acceptable; ownership enforced via `@CurrentUser`).
- Remove any remaining mock/degraded placeholders per page (audited: none found beyond the
  global-indicator noted above; `notifications`/`support` pages already show `DegradedNote`).
- Run backend/frontend typecheck + DB migration on a real environment; validate Client A /
  Client B / Admin isolation (§40).
