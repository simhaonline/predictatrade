# Predict-A-Trade XAUUSD — Client Data Collection + Client Dashboard Rebuild

You are working inside the existing **Predict-A-Trade XAUUSD** production codebase.

The Windows Agent is responsible for connecting the client machine / MT4 / MT5 environment with the Predict-A-Trade backend.

Your task is to **collect genuine client/device/trading-terminal data from the Windows Agent, persist it correctly in the existing backend/database, and rebuild/update the Client Dashboard pages and menu so each client sees only their own live/account-specific information.**

Keep this implementation **focused and minimal**.

Do not redesign the entire platform.

Do not modify trading/scoring logic unless directly required to expose already-generated information.

Do not create duplicate services, duplicate APIs, duplicate database tables, or parallel architecture when equivalent functionality already exists.

---

# PRIMARY OBJECTIVE

Build the complete flow:

```text
Windows Client Agent
        ↓
Authenticated API
        ↓
Predict-A-Trade Backend
        ↓
TimescaleDB / existing database
        ↓
Client-specific authorization
        ↓
Client Dashboard
```

Each logged-in client must see information belonging **only to their own account, subscription, license, activated devices and MT4/MT5 terminals**.

Admin users may continue to see all clients through the existing Admin Dashboard.

---

# 1. AUDIT BEFORE MODIFYING

First inspect the existing implementation for:

* Windows Agent API calls
* device activation
* license validation
* device credentials
* access tokens
* heartbeat
* MT4/MT5 connection
* broker account mapping
* user/account mapping
* subscription mapping
* client dashboard
* admin dashboard
* frontend routes
* sidebar/menu
* TimescaleDB schema
* Valkey usage
* WebSocket implementation
* signal delivery
* notification system
* audit/event logs

Determine what already exists.

Do not rebuild existing working systems.

Return a short initial audit:

```text
CLIENT DATA FLOW AUDIT

Windows Agent:
Authentication:
License Mapping:
Device Mapping:
Trading Account Mapping:
Heartbeat:
Database Persistence:
WebSocket:
Client Dashboard:
Subscription Entitlements:
Existing APIs:
Missing Wiring:
Required Changes:
```

Then implement only the missing wiring/features.

---

# 2. CLIENT IDENTITY MODEL

Every incoming Windows Agent request must resolve to a genuine authenticated client.

Use the existing Predict-A-Trade chain where available:

```text
User
↓
Subscription
↓
License
↓
Device Activation
↓
Device Credential
↓
Access Token
↓
Windows Agent
↓
MT4/MT5 Account
```

Never trust a client-provided:

```text
user_id
client_id
license_owner
subscription_tier
admin flag
```

without resolving it server-side from the authenticated credential/token.

The backend must establish client ownership.

A client must never be able to request another client's data by changing an ID in a URL or request body.

---

# 3. WINDOWS AGENT DATA COLLECTION

Extend the existing Windows Agent heartbeat/status reporting only as needed.

Collect useful operational information such as:

## Agent

```text
agent_version
agent_status
agent_started_at
agent_last_seen_at
agent_uptime_seconds
machine_name
os_version
architecture
service_status
health_status
```

Do not collect unnecessary private client information.

---

# 4. DEVICE INFORMATION

Collect/map the existing activation/device information:

```text
device_id
device_name
device_fingerprint_hash
device_activation_status
activated_at
last_seen_at
device_credential_status
```

Do not expose raw hardware identifiers unnecessarily.

Prefer already-generated Predict-A-Trade device identifiers or hashes.

---

# 5. MT4 / MT5 TERMINAL INFORMATION

Where available from the existing bridge/agent collect:

```text
terminal_type
terminal_version
terminal_status
terminal_connected
terminal_last_seen
```

Example:

```text
MT4
MT5
```

If both terminals are available, store them independently.

Do not fabricate connection status.

---

# 6. TRADING ACCOUNT INFORMATION

Collect genuine broker/trading-account information already available through MT4/MT5.

Minimum useful fields:

```text
trading_account
broker_name
broker_server
account_currency
account_type
leverage
balance
equity
margin
free_margin
margin_level
floating_profit_loss
open_positions_count
pending_orders_count
```

If the broker/terminal does not provide a field, return:

```text
null
```

Do not guess.

---

# 7. SYMBOL / XAUUSD STATUS

Because this platform focuses on XAUUSD, collect relevant terminal information where already available:

```text
symbol
symbol_available
market_open
bid
ask
spread
digits
last_tick_time
```

Do not turn the Windows Agent into a second market-data engine.

Use the existing MT4/MT5 bridge/feed architecture.

---

# 8. HEARTBEAT PAYLOAD

Reuse the existing heartbeat endpoint where possible.

Do not create multiple heartbeat systems.

Extend the payload only where appropriate.

Conceptual example:

```json
{
  "agent_version": "2.x.x",
  "agent_status": "online",
  "machine_name": "CLIENT-PC",

  "terminal": {
    "type": "MT5",
    "version": "...",
    "connected": true
  },

  "account": {
    "number": "12345678",
    "broker": "Broker Name",
    "server": "Broker-Live",
    "currency": "USD",
    "balance": 10000.00,
    "equity": 10025.45,
    "margin": 125.00,
    "free_margin": 9900.45,
    "floating_profit_loss": 25.45
  },

  "xauusd": {
    "available": true,
    "bid": 0,
    "ask": 0,
    "spread": 0,
    "last_tick_time": "..."
  }
}
```

Use the project's actual naming conventions.

Do not unnecessarily break existing API contracts.

---

# 9. HEARTBEAT FREQUENCY

Do not overload the API/database.

Use the current heartbeat interval if already configured.

If none exists, retain the existing Predict-A-Trade design target of approximately:

```text
15–30 seconds
```

for lightweight operational heartbeat.

High-frequency market ticks must continue through the existing real-time market-data architecture, not through this general client heartbeat.

---

# 10. DATABASE PERSISTENCE

Audit the existing database first.

Reuse existing tables whenever possible.

Persist the necessary current-state information for:

```text
client
license
device
terminal
trading account
agent heartbeat
connection health
```

Avoid creating a new table for every screen.

Prefer normalized relationships such as:

```text
users
subscriptions
licenses
device_activations
devices
trading_accounts
agent_heartbeats / device_status
```

using existing schema names if already present.

---

# 11. CURRENT STATE VS HISTORY

Separate current operational state from historical telemetry.

The dashboard needs fast current-state retrieval.

For example:

```text
last_seen_at
current balance
current equity
connection status
agent version
terminal connection
```

should be easily retrievable without scanning millions of historical rows.

Historical records should only be kept where actually required.

Do not store every heartbeat forever if it provides no business value.

Use existing retention policies if already configured.

---

# 12. CLIENT DASHBOARD AUTHORIZATION

Every Client Dashboard API must enforce:

```text
authenticated user
+
resource ownership
+
subscription entitlement
```

Never depend only on frontend hiding.

Backend authorization is mandatory.

For example:

```text
GET /client/devices
```

must internally restrict the query to the authenticated client's devices.

Not:

```text
GET /client/devices?user_id=123
```

where the client can change the user ID.

---

# 13. CLIENT DASHBOARD MENU

Audit the existing user/client sidebar.

Rebuild/update it into a clean professional structure.

Recommended menu:

```text
Dashboard

Trading
 ├─ Live Signals
 ├─ Signal History
 └─ Performance

My Trading Account
 ├─ Account Overview
 ├─ Open Positions
 └─ Trade History

Predict-A-Trade
 ├─ Market Status
 ├─ Indicator Analysis
 └─ Scoring / Analysis

Connections
 ├─ Windows Agent
 ├─ MT4 / MT5
 └─ Devices

Account
 ├─ Subscription
 ├─ License
 ├─ Billing
 ├─ Referrals
 └─ Profile

Support
 ├─ Notifications
 ├─ Help / Support
 └─ System Status
```

Do not expose admin-only pages.

The exact menu should reuse existing pages/routes where they already exist.

Do not create empty pages just to match this menu.

---

# 14. MAIN CLIENT DASHBOARD

Rebuild the primary Client Dashboard into a useful account-specific overview.

Recommended top status cards:

```text
Agent Status
MT4 / MT5 Status
Trading Account
Subscription
License
XAUUSD Market
```

Example:

```text
Agent
ONLINE
Last seen 6 sec ago

MT5
CONNECTED
Broker-Live

Account
12345678
USD

Subscription
PRO
Active

License
ACTIVE
1 / 1 device

XAUUSD
MARKET OPEN
Spread: ...
```

All values must come from real backend data.

No hardcoded placeholders.

---

# 15. ACCOUNT SUMMARY

Display:

```text
Balance
Equity
Floating P/L
Margin
Free Margin
Margin Level
Open Positions
Pending Orders
```

If the trading account is disconnected:

Do not continue displaying stale data as if it is live.

Clearly indicate:

```text
Last updated: <time>
Terminal disconnected
```

---

# 16. CONNECTION STATUS PANEL

Create a compact connection flow:

```text
Predict-A-Trade Cloud
        ↕
Windows Agent
        ↕
MT4 / MT5
        ↕
Broker
```

Each should show:

```text
CONNECTED
DEGRADED
DISCONNECTED
UNKNOWN
```

based on genuine state.

Do not show fake green indicators.

---

# 17. WINDOWS AGENT PAGE

Create/update:

```text
/client/agent
```

or the project's existing equivalent.

Show:

```text
Agent Status
Agent Version
Installed Device
Operating System
Last Heartbeat
Uptime
Health Status
```

Optionally show:

```text
Update Available
```

only if the project already has agent version/update infrastructure.

Do not build an update platform solely for this task.

---

# 18. MT4 / MT5 PAGE

Create/update the existing terminal page.

Show:

```text
Terminal Type
Connection Status
Trading Account
Broker
Broker Server
Account Type
Currency
Leverage
Last Connected
Last Data Received
```

If multiple terminals are activated, display each separately.

---

# 19. DEVICES PAGE

Display only devices belonging to the logged-in client.

Fields:

```text
Device Name
Device ID
Agent Version
Terminal
Activation Date
Last Seen
Status
```

Actions should depend on existing licensing rules.

Possible action:

```text
Deactivate Device
```

only if the existing backend already allows it or it can be safely added to the current activation architecture.

Do not permit clients to bypass device limits.

---

# 20. SIGNALS PAGE

Use existing Predict-A-Trade signal data.

Do not rewrite the signal engine.

Client should see signals according to their subscription.

Recommended tabs where already supported:

```text
All
Standard Scalping
Ultra Scalping
Standard Swing
Trend Swing
```

For each signal show existing fields such as:

```text
Signal Reference
Strategy
Action
Entry
Stop Loss
TP1
TP2
TP3
Score
Confidence
Risk
Risk/Reward
Timeframe
Session
Regime
Status
Generated At
Expiry
```

Display only genuine stored/generated values.

---

# 21. SIGNAL ENTITLEMENTS

This is mandatory.

A client must only receive/view signals allowed by their subscription.

Enforce in backend APIs, WebSocket subscriptions and frontend.

Example concept:

```text
FREE
limited monthly signals

BASIC
larger signal allowance
selected strategies

PRO
higher limits
additional strategies/features

PREMIUM / ENTERPRISE
full entitled functionality
```

Use the actual subscription definitions from the existing database/admin configuration.

Do not hardcode plan limits in multiple places.

Plan rules should come from the existing subscription/entitlement configuration.

---

# 22. FREE USER RESTRICTIONS

Free users must not automatically see all generated signals.

If the existing plan configuration limits free clients monthly, enforce that exact entitlement.

Possible restricted UI:

```text
Signal unavailable on your current plan.
Upgrade to access this signal.
```

But do not leak:

```text
entry
SL
TP
direction
confidence
full reasoning
```

for signals the client is not entitled to view.

The backend must remove restricted fields.

Do not merely blur them with CSS while sending the data to the browser.

---

# 23. CLIENT PERFORMANCE

Where existing execution/trade result data is genuinely available, show:

```text
Total Trades
Wins
Losses
Win Rate
Net P/L
Average R:R
Best Trade
Worst Trade
```

Do not calculate fake strategy performance from signals unless the existing platform already has validated signal outcome tracking.

Clearly distinguish:

```text
Signal Performance
```

from:

```text
Client Account Performance
```

They are not automatically the same.

---

# 24. OPEN POSITIONS

If execution data is available from the client's terminal, show current positions:

```text
Ticket
Symbol
Buy / Sell
Lot Size
Entry
Current Price
SL
TP
Floating P/L
Open Time
```

This page must be read-only unless Predict-A-Trade already explicitly supports remote trade management.

Do not add remote order execution as part of this task.

---

# 25. TRADE HISTORY

Where available from MT4/MT5/backend show:

```text
Ticket
Symbol
Direction
Volume
Open Price
Close Price
SL
TP
Opened At
Closed At
Profit/Loss
Swap
Commission
```

Use pagination.

Do not load unlimited history into one request.

---

# 26. SUBSCRIPTION PAGE

Use the existing billing/subscription architecture.

Display:

```text
Current Plan
Subscription Status
Start Date
Renewal Date
Billing Cycle
Signal Allowance
Signals Used
Signals Remaining
Device Limit
Active Devices
Available Features
```

Do not duplicate subscription calculations in frontend JavaScript.

Backend remains source of truth.

---

# 27. LICENSE PAGE

Display client-owned licensing information:

```text
License Status
License Reference
Activated Devices
Device Limit
Issued Date
Expiry Date
Last Validation
```

Do not display sensitive license secrets or cryptographic material.

Mask license keys if they must be shown.

Example:

```text
PAT-XXXX-XXXX-7821
```

---

# 28. REFERRALS

Reuse the existing Predict-A-Trade referral system.

Display:

```text
Referral Code
Referral Link
Total Referrals
Active Referrals
Pending Commission
Approved Commission
Paid Commission
```

Do not recalculate commission differently from the backend referral commission engine.

The dashboard must display backend-calculated values.

---

# 29. BILLING

Reuse current billing APIs.

Display:

```text
Plan
Amount
Billing Period
Payment Status
Invoice
Payment Date
Next Billing Date
```

Never expose another client's billing information.

---

# 30. NOTIFICATIONS PAGE

Allow the client to view/edit only supported notification preferences.

Potential options:

```text
Signal Notifications
Agent Offline
MT4/MT5 Disconnected
Subscription Expiry
License Expiry
Billing
Security
```

Delivery channels may include:

```text
Telegram
Discord
Email
```

only where the backend already supports them.

Do not expose secrets after they are saved.

Display:

```text
Configured
```

rather than the raw token/password.

---

# 31. REAL-TIME DASHBOARD UPDATES

Use the existing WebSocket infrastructure where available.

Do not implement aggressive browser polling if WebSockets already exist.

Real-time candidates:

```text
Agent online/offline
MT4/MT5 status
Account balance/equity
Open positions
Market status
New signals
Signal status changes
```

Use REST APIs for normal static/history/account queries.

Use WebSocket for live state.

---

# 32. ONLINE / OFFLINE RULE

Use deterministic heartbeat status.

Example:

```text
ONLINE
heartbeat within expected window

DEGRADED
heartbeat delayed

OFFLINE
heartbeat exceeds timeout
```

Reuse existing timing rules where present.

Do not mark a client online simply because they logged into the web dashboard.

Agent connectivity and dashboard login are separate states.

---

# 33. TIME ZONES

Store timestamps in UTC in backend/database.

Display timestamps according to the client's configured timezone where supported.

Show timezone clearly for signal times where confusion is possible.

Do not mix:

```text
UTC
server local time
broker time
browser time
```

without explicit conversion.

---

# 34. API DESIGN

Prefer existing endpoints.

Where missing, add a minimal client-specific API structure such as:

```text
GET /api/client/dashboard
GET /api/client/agent
GET /api/client/devices
GET /api/client/terminal
GET /api/client/account
GET /api/client/positions
GET /api/client/trades
GET /api/client/signals
GET /api/client/subscription
GET /api/client/license
GET /api/client/referrals
GET /api/client/notifications
```

Do not require client ID in these endpoints.

Resolve client identity from authentication.

---

# 35. DASHBOARD AGGREGATION ENDPOINT

Avoid making the homepage issue 20 separate requests.

Where appropriate create/reuse:

```text
GET /api/client/dashboard
```

returning a concise dashboard summary:

```json
{
  "agent": {},
  "terminal": {},
  "account": {},
  "subscription": {},
  "license": {},
  "market": {},
  "signal_summary": {}
}
```

Do not return huge historical datasets through this endpoint.

---

# 36. FRONTEND LOADING STATES

Every client page must handle:

```text
Loading
No Data
Disconnected
Permission Restricted
API Error
Healthy
```

Do not show:

```text
undefined
NaN
0
```

when the genuine state is unknown.

Examples:

```text
Not Connected
No trading account detected
Waiting for Windows Agent
No signals available under current plan
```

---

# 37. FIRST-TIME CLIENT EXPERIENCE

If a new client has:

```text
subscription
+
license
```

but no Windows Agent/device connection, show a useful onboarding state.

Example:

```text
Connect Your Trading Account

1. Download/install Predict-A-Trade Windows Agent
2. Activate your license
3. Connect MT4/MT5
4. Wait for connection verification
```

Once connected, automatically replace onboarding cards with live information.

Do not require the client to manually refresh when WebSocket/live state already supports updates.

---

# 38. SUBSCRIPTION-BASED MENU ACCESS

Client sidebar/menu must reflect entitlement.

But backend authorization remains mandatory.

Example:

```text
Available feature
→ normal menu item

Locked feature
→ lock icon / upgrade indicator

Admin feature
→ completely hidden
```

Do not expose admin routes to clients.

---

# 39. ADMIN CONTROL

Do not remove existing admin control.

Admin should be able to manage:

```text
plan limits
signal allowances
feature access
device limits
license status
subscription status
client state
```

Client Dashboard must consume those backend policies dynamically.

Do not hardcode plan rules into the frontend.

---

# 40. CLIENT DATA ISOLATION TESTS

Test with at least:

```text
Client A
Client B
Admin
```

Verify:

```text
Client A cannot access Client B data.
Client B cannot access Client A data.
Client cannot access Admin APIs.
Admin retains authorized visibility.
```

Test direct API manipulation.

Do not rely only on UI testing.

---

# 41. SECURITY

Ensure:

* authenticated API calls
* ownership checks
* entitlement checks
* rate limiting where existing
* no secrets returned
* no raw device credentials
* no token leakage
* no passwords in logs
* no SQL/client-ID manipulation
* no IDOR vulnerabilities

Do not expose entire backend database objects directly.

Use sanitized API responses.

---

# 42. DO NOT TOUCH TRADING LOGIC

Do not modify:

```text
MasterScoring
DynamicRegime
EnterpriseRegime
XAU2
UMS2
Vision AI
Astro/KP
Macro
COT
DXY
Session Overlap
indicator calculations
mathematical scoring
signal thresholds
trade management
risk management
```

unless an existing output field cannot be connected to the dashboard due to a genuine wiring bug.

This task is:

```text
DATA COLLECTION
+
BACKEND WIRING
+
CLIENT AUTHORIZATION
+
CLIENT DASHBOARD
```

not a signal-engine redesign.

---

# 43. DO NOT CREATE FAKE DATA

Remove dashboard mock data wherever the real backend equivalent exists.

Do not display fake:

```text
balances
signals
win rates
connection statuses
subscription data
agent statuses
prices
performance
```

If data is unavailable:

display:

```text
Unavailable
Waiting for Agent
Not Connected
No Data
```

instead.

---

# 44. RESPONSIVE UI

Preserve the existing Predict-A-Trade design system.

Support:

```text
Desktop
Laptop
Tablet
Mobile
```

Do not change branding unnecessarily.

Preserve:

```text
light/dark/system theme
existing logos
existing colors
existing component library
```

unless fixing a clear UI defect.

---

# 45. FINAL CLIENT MENU TARGET

After implementation the Client Dashboard should approximately provide:

```text
OVERVIEW
Dashboard

TRADING
Live Signals
Signal History
Performance

MY TRADING ACCOUNT
Account Overview
Open Positions
Trade History

ANALYSIS
Market Status
Indicator Analysis
Scoring / Analysis

CONNECTIONS
Windows Agent
MT4 / MT5
Devices

ACCOUNT
Subscription
License
Billing
Referrals
Profile

SETTINGS
Notifications
Support
System Status
```

Only retain entries supported by actual working functionality.

Do not ship empty placeholder routes.

---

# 46. VALIDATION

Run appropriate backend/frontend tests.

Verify:

```text
Agent heartbeat reaches backend
Client identity resolves correctly
Device resolves correctly
Trading account resolves correctly
Data persists correctly
Client dashboard reads correct account
WebSocket updates function
Subscription limits function
Signal restrictions function
Device limits function
Admin remains unaffected
Client isolation passes
No mock data remains where live data exists
```

---

# 47. FINAL REPORT

Return:

```text
PREDICT-A-TRADE CLIENT DASHBOARD — FINAL STATUS

Windows Agent Data Collection:
PASS / FAIL

Heartbeat:
PASS / FAIL

Client Mapping:
PASS / FAIL

License Mapping:
PASS / FAIL

Device Mapping:
PASS / FAIL

MT4/MT5 Mapping:
PASS / FAIL

Trading Account Data:
PASS / FAIL

Database Persistence:
PASS / FAIL

Dashboard API:
PASS / FAIL

WebSocket:
PASS / FAIL

Main Dashboard:
PASS / FAIL

Signals:
PASS / FAIL

Signal Entitlements:
PASS / FAIL

Account Overview:
PASS / FAIL

Open Positions:
PASS / FAIL

Trade History:
PASS / FAIL

Windows Agent Page:
PASS / FAIL

MT4/MT5 Page:
PASS / FAIL

Devices Page:
PASS / FAIL

Subscription:
PASS / FAIL

License:
PASS / FAIL

Billing:
PASS / FAIL

Referrals:
PASS / FAIL

Notifications:
PASS / FAIL

Client Data Isolation:
PASS / FAIL

Admin Isolation:
PASS / FAIL

Files Modified:
Files Added:
Database Changes:
API Changes:
Frontend Routes Changed:

Remaining External Dependencies:

1.
2.
3.

FINAL DECISION:
GO / CONDITIONAL GO / NO-GO
```

# MOST IMPORTANT RULES

1. **Collect genuine client/agent/MT4/MT5 information and display it to the correct authenticated client.**
2. **Every client sees only their own information.**
3. **Subscription and license restrictions must be enforced in the backend, not merely hidden in the frontend.**
4. **Admin retains complete control through the existing Admin Dashboard.**
5. **Reuse existing Predict-A-Trade architecture before creating anything new.**
6. **Do not create fake dashboard data.**
7. **Do not modify signal mathematics or trading logic unnecessarily.**
8. **Do not over-engineer.**
9. **Fix wiring first, then UI.**
10. **Leave the existing working Predict-A-Trade platform intact.**
