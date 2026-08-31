# Admin Guide
## v1.17.4 — 30 August 2026

### Overview

This guide covers system administration for Predict-A-Trade XAUUSD. The admin dashboard is accessible at `/admin` after logging in with an admin-role account. All admin operations require JWT authentication with the `ADMIN` role.

---

## 1. Dashboard & Monitoring

### System Health

The admin dashboard home shows real-time system health:

| Widget | What it shows | Alert threshold |
|--------|---------------|:---------------:|
| Engine Status | Strategy engine liveness (6 engines) | Any engine DEGRADED or DOWN |
| Market Feeds | TwelveData, FMP, Ollama status | Any feed UNAVAILABLE |
| Indicators | 35/42 live, 7 warming | <30 live indicators |
| Signal Pipeline | Signals generated, NO-TRADE count, gate failures | >50% NO-TRADE rate |
| WebSocket | Active connections (users + agents) | Connection drops |
| CPU/Memory | Host resource usage | >80% CPU or >85% memory |

Health endpoints available at:
- `GET /health` (Go realtime, port 13081)
- `GET /api/v1/health` (NestJS control, port 13080)
- `GET /` (Frontend, port 13082) — returns 307/200

### Monitoring Stack
- **Prometheus** (port 9090): metrics collection — Go runtime, WS connections, signal flow, gate state, provider health
- **Grafana** (port 3001): dashboards — engine health, signal throughput, latency, error rates
- **ntfy** (port 8091): push notifications — configure topics for alerts

---

## 2. User Management

### Viewing Users
Navigate to Admin → Users. Lists all registered users with:
- Email, name, role, plan tier, subscription status
- Created date, last login, MFA status
- License status, device count

### Managing Users
- **Edit role:** Promote to ADMIN or demote to USER
- **Suspend/Activate:** Disable a user account (preserves data, blocks access)
- **Delete:** Permanent removal (cascades to subscriptions, devices, signals)
- **View details:** Full profile, subscription history, device list, signal activity

### User Roles
| Role | Capabilities |
|------|-------------|
| ADMIN | Full system access, user management, billing, operations |
| USER | Dashboard access per plan tier, own signals/settings |

---

## 3. Subscription & Billing Management

### Plans (Admin → Plans)
Active plans: FREE ($0), STANDARD ($49/mo), PRO ($199/mo), ELITE ($499/mo)
- View/edit: price, strategy limits, feature entitlements
- Create new plans or hide legacy plans
- Plan changes take effect on next billing cycle

### Subscriptions (Admin → Subscriptions)
- View all active/cancelled subscriptions
- Manual subscription creation (for offline payments)
- Cancel/refund subscriptions
- View payment history per user

### Billing (Admin → Billing)
- View invoices, payments, refunds
- Stripe webhook events log
- NOWPayments IPN log (crypto payments)
- Manual payment reconciliation

---

## 4. Referral & Commission Management

### Referral Program
- **Commission tiers (level 1 / 2 / 3+):** Standard (10/4/1%), Pro (15/5/2%), Elite (20/6/2%). Free-tier referrals are excluded — commission is credited only on a **paid** conversion (settled revenue), never for a referred Free account.
- **Multipliers:** First purchase (100%), Second (75%), Recurring (50%)
- View referral tree per user (up to 3 levels)
- Commission ledger: all earned, reserved, settled entries

### Payouts (Admin → Payouts)
- View pending payouts (RESERVED state)
- Approve/reject payouts
- Payout methods: wallet, bank transfer, crypto
- Ledger: double-entry, RESERVED → SETTLED state machine

---

## 5. License & Device Management

### Licenses (Admin → Licenses)
- View all active licenses per user
- License types: TRIAL, STANDARD, PRO, ELITE
- Device binding: each license can bind N devices (plan-dependent)
- Revoke licenses (disconnects bound devices)

### Devices (Admin → Devices)
- View all activated devices per user
- Device details: OS, IP, activation date, last heartbeat
- Force-deactivate devices (revokes access)

### Windows Agent Monitoring
- Agent connection status (WebSocket heartbeat) — now bridged into the control-plane database for unified monitoring
- Two agent roles: **Client Agent** (execution, `pat-agent-client`, port 13081) and **Master Node** (data-only, `pat-agent-master`, port 13091)
- Agent version tracking, health endpoint verification, and license validation (proactive server-side, no agent changes)
- Suspended agents list (3-strike SL violation system)
- Agent commands: disconnect, emergency stop, kill switch
- Live agent status visible via `/api/v1/agents/status` and admin dashboard
- **EA Diagnostics (v1.16.x):** TRADE-CONFIG startup diagnostic confirms AutoExecute/ExecuteCandidates/algo-trading flags. SIGNAL-EXEC-CHECK reveals execution decision per signal — traces swap/triple-swap vetoes, duplicate ID filtering, license status, and all silent veto reasons
- **EA defaults & capital protection:** `AutoExecute` now defaults to **false** (signal-only — the EA displays signals; operators must opt in to auto-trade). The Execution EA enforces a client-side daily-loss guard: a **soft** limit blocks new entries only (recovers intraday, bypassable via the `BypassDailyLossBlock` EA input) and a **hard** limit (`MaxDailyLossPct`) closes all positions and is **never** bypassable. Client terminal logs (`error.log` / Experts) emit `STATUS`, `SIGNAL RECEIVED`, and `CAPITAL` lines (all times broker/server time, not UTC); license *strategy* detail is intentionally omitted.

---

## 6. Signal & Trading Oversight

### Signal Monitoring (Admin → Signals)
- Real-time signal feed from all 5 strategy engines with **20-signal-per-page pagination** (prevents browser lockup)
- Multi-tab strategy filtering: ALL, STANDARD_SCALPING, ULTRA_SCALPING, STANDARD_SWING, TREND_SWING, MARNIE_FIB
- Direction filters: BUY, BUY_CANDIDATE, SELL, SELL_CANDIDATE, NO-TRADE
- Full table columns: Time, Direction, Strategy, Symbol, Probability, Score, Entry, SL, TP1, TP2, TP3, Regime, Session, Quality Grade (A+/A/B), Expectancy (EV_R), Rejection Reason, Status
- Expand rows: Click any signal to reveal full evidence chain, lot sizing (SuggestedLot), risk metrics (RiskDollars, RiskPctOfEquity, SLDistancePoints), pillar contributions, gate results
- Export signals to CSV

### NO-TRADE Analysis
- View all NO-TRADE events with reason codes:
  - `NTInsufficientScore`, `NTConflictingDirection`, `NTRegimeMismatch`
  - `NTHighNewsRisk`, `NTATRNotReady`, `NTHTFBearishVeto`
- Diagnostics: why each NO-TRADE fired (evidence breakdown)

### Gate Status
- 16 gates, ordered execution
- Gate health: PASS/FAIL/DEGRADED per signal
- Gate failure trends over time

---

## 7. Backtesting (Admin → Backtests)

### Running Backtests
- Select: strategy, symbol (XAUUSD), date range, timeframes
- Configurable: starting capital, risk %, commission
- Results: total trades, win rate, profit factor, max drawdown, Sharpe ratio

### Viewing Results
- Trade log per backtest
- Equity curve chart
- Compare multiple backtest runs
- Export results to CSV/JSON

---

## 8. Audit & Compliance

### Audit Log (Admin → Audit)
- All security events: logins, logouts, MFA changes, password resets
- All admin actions: user management, plan changes, payouts
- Signal delivery audit: who received which signal, when
- Filterable by: user, event type, date range

### Compliance Dashboard
- Client event log: user actions, consent changes
- Data retention: active accounts, market data (unbounded pending policy)
- Legal documents: Terms, Privacy, DPA versions in effect

---

## 9. Operations

### Backups
```bash
# PostgreSQL backup
docker exec pat-postgres pg_dump -U pat_admin predictatrade > backup_$(date +%Y%m%d).sql

# Restore
docker exec -i pat-postgres psql -U pat_admin predictatrade < backup.sql
```
- Schedule: daily automated backup recommended
- Retention: keep 30 days of daily backups
- Test restore quarterly

### Service Management
```bash
# View all services
docker compose ps

# Restart a service
docker compose restart realtime
docker compose restart frontend

# View logs
docker compose logs -f realtime
docker compose logs --tail=100 control

# Rebuild and restart
docker compose up -d --build realtime
```

### Database Migrations
```bash
# Forward migration
./scripts/migrate.sh up

# Status check
psql -U pat_admin -d predictatrade -c "SELECT * FROM audit.migration_history ORDER BY applied_at DESC LIMIT 10;"
```

### Incident Response
1. **Service down:** Check `docker compose ps`, verify ports with `ss -tlnp`, check logs
2. **Market data stale:** Verify TwelveData/FMP API keys, check provider logs
3. **Signal stopped:** Check engine liveness, gate failures, NO-TRADE rate
4. **Database issues:** Check connections, disk space, run `pg_isready`
5. **Security incident:** Revoke affected tokens, disable user, check audit log

---

## 10. Security

### Access Control
- Admin accounts require MFA (TOTP)
- JWT tokens: HttpOnly cookie, no localStorage exposure
- Session rotation: force re-auth for sensitive operations
- Rate limiting: login throttling, API rate limits per plan

### Emergency Procedures
- **KILL_SWITCH:** Stops a specific agent — closes all positions, disconnects
- **EMERGENCY_STOP:** Stops an agent — closes all PAT-managed positions
- **CLOSE_POSITION:** Close a specific position by ticket ID
- **Agent suspension:** Automatic after 3 SL violations

### Secrets Management
- Never commit secrets to git (already in .gitignore)
- Rotate JWT_SECRET on security incidents
- API keys in environment variables, not in code
- Production credentials via Docker secrets or env files

---

## 9. Platform Operations (Admin → Operations)

Emergency and lifecycle controls, backed by `control.platform_operations` (audited):

| Action | Endpoint | Effect |
|--------|----------|--------|
| Halt trading | `POST /operations/halt-trading {reason}` | Full execution halt — engine refuses EXECUTABLE delivery |
| Resume trading | `POST /operations/resume-trading {reason}` | Reverts the halt record; instant action |
| Pause signals | `POST /operations/pause-signals {reason}` | Stops new signal generation (delivery of already-sent signals continues) |
| Resume signals | `POST /operations/resume-signals {reason}` | Reverts PAUSE_SIGNALS (verifiable in Operations → Active) |
| Strategy kill switches | `POST /operations/strategy/{id}/enable|disable` | Per-engine enable/disable (replaces the need to redeploy config) |

`{reason}` is **required** — the endpoint returns 500 without a JSON body (by design,
operations must be attributable). Every action is persisted and visible in **Operations → Active**
and the audit log.

## 10. Feature Flags & AI Models

- **Feature Flags** — PTB pillar registry (`trading.ptb_feature_flags`); modes
  `OFF | SHADOW | ACTIVE | DISABLED | UNSUPPORTED | RESEARCH`. Editing takes the flag `id`
  (UUID) — the UI sends the row id selected from the table, never the module name.
- **AI Providers / Models** — read-only registry + explicit activate/deactivate per model;
  models can never self-promote.

## 11. Backup-DR, Releases, Broker Qualification, Macro News

Admin-extras views (`admin-extras.controller.ts`): backup configuration + last- validated
restore, client release registry (agent auto-updater manifest history), broker execution
qualification evidence (commission/latency/reject observations), and macro news ingestion
status (FMP events sync).

## 12. Reconciliation Health (v1.17.3 — expanded in v1.17.4 with market_closed gate)

Prometheus (port 13081 `/metrics`):
- `pat_reconciliation_acks_timeout` — delivered but never ACKed within 2 min
- `pat_reconciliation_fills_timeout` — ACKed but no fill/trade-result within 10 min
- `pat_reconciliation_tracked_signals` — registry size

Alerts (ntfy): `SIGNAL_ACK_TIMEOUT`, `SIGNAL_FILL_TIMEOUT`, deduped per signal with a
10-minute re-alert window. Sustained non-zero values usually mean the Windows Agent is
disconnected, the EA is detached, or a broker order was silently rejected — cross-check
Admin → Signals (delivery state) and the agent's local health port (:9000 client / :9001 master).

---

## 13. Payments (USDT-only) & Mail (v1.17.4)

**USDT-only.** Stripe checkout/webhook are controller-disabled (403/204). The
only gateway is NOWPayments with `pay_currency:'usdt'`.

Anti-scam guarantees (verified in code + DB):
- IPN signature HMAC-SHA512 (timing-safe) — forged notices are 401'd.
- Replay dedupe via `billing.payment_events` exact-key.
- Transactional one-shot settlement (`FOR UPDATE` + status guard).
- **Amount verification** — paid amount must cover the invoice expected
  (±`NOWPAYMENTS_UNDERPAY_TOLERANCE_PCT`, default 2%) or the payment is
  marked **UNDERPAID** with an audit row; subscription is not activated.
- Users see live status on the billing page: awaiting (Resume link),
  underpaid, verified-confirmed (with timestamp), failed.

**Mail** — `pat-mail-relay` (Go) submits platform email from
`pat.predictatrade.com`; the control plane uses it via `SMTP_HOST`.
Operator DNS: A + SPF `v=spf1 a:pat.predictatrade.com -all` + DKIM
(`pat1._domainkey`) + DMARC on predictatrade.com. Runbook:
`mail-relay/README.md`. Rate limits: rcpt cap 25/msg, 10 MB cap, spool
dead-letter after 24h of retries.
