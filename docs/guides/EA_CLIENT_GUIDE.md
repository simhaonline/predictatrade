# MT4/MT5 EA-Client Guide (Option B)
## v1.23 (MT4) / v1.20 (MT5) — 2 September 2026

- **Edge-poll cadence guard**: new `PATPollMs` input (default 3000 ms, floor
  1000 ms). Polls are throttled to one per interval instead of one per tick —
  fixes the 2026-09-02 HTTP 429 storm (ThrottlerException) where tick-rate
  polling from several terminals behind one VPS IP exhausted the shared
  300 req/min bucket. **Recompile required** — old builds still work but may
  hit 429s under load.
- **Restart race fix**: signals arriving before the first license activation
  completed (status still PENDING) are no longer dropped client-side. They
  were already filtered server-side by license + plan. Explicit negative
  license states (REVOKED / SUSPENDED / EXPIRED / DENIED) still block.
  Fixes "Strategy check: license not validated — blocking STANDARD_SWING"
  after terminal restarts.

## v1.19.0 — 1 September 2026

### Overview

Since v1.19.0 the MetaTrader EAs talk to the Predict-A-Trade cloud **directly over HTTPS** — the Windows Agent (`pat-agent.exe` / `pat-master.exe`, services, health ports) is REMOVED. There are two EA roles, both single-file MQL with no includes, no scripts, no file dependencies beyond their own bootstrap state:

| Role | EA file | Purpose | Cloud endpoint |
|------|---------|---------|----------------|
| **Client EA** | `mql/mt5/PredictATrade_MT5.mq5`, `mql/mt4/PredictATrade_MT4.mq4` | Activates a device with your license key, polls for executable signals + server commands, ACKs each, executes on the broker account. | `api.predictatrade.com` |
| **Master EA** (data node) | `mql/mt5/PredictATrade_MasterNode_MT5.mq5`, `mql/mt4/PredictATrade_MasterNode_MT4.mq4` | Streams XAUUSD ticks/snapshots **data only** — never executes. Feeds the engine's authoritative live feed. | same |

### Traffic (what to allowlist)

| Direction | Endpoint | Auth |
|-----------|----------|------|
| EA → cloud (data) | `POST https://api.predictatrade.com/ingest/agent` | Bearer device JWT (refresh-token grant) |
| EA ← cloud (signals/commands) | `POST https://api.predictatrade.com/api/v1/devices/edge-poll` | device HMAC (v1 canonical string) |
| EA → cloud (ack) | `POST https://api.predictatrade.com/api/v1/devices/edge-ack` | device HMAC |
| EA → cloud (liveness) | `POST https://api.predictatrade.com/api/v1/devices/edge-heartbeat` | device HMAC |
| EA → cloud (bootstrap) | `POST https://api.predictatrade.com/api/v1/devices/activate` + `/refresh` | license key |

**One-time terminal setup:** Tools → Options → Expert Advisors → tick *"Allow WebRequest for listed URL"* → add `https://api.predictatrade.com`.

### Install (client EA)

1. Dashboard → **MetaTrader Client** page → copy your license key.
2. Download the EA source (`.mq5`/`.mq4`) from `https://downloads.predictatrade.com/mql/`.
3. Copy it into `MQL5/Experts` (or `MQL4/Experts`), compile in MetaEditor (F7) — 0 errors.
4. Drag onto an **XAUUSD** chart, paste the license key into the `LicenseKey` input, enable algo trading.
5. The EA activates its cloud device automatically (state file in the common folder stores `device_id|device_secret|refresh_token`) and begins polling.

**State files are per-platform** — MT4 and MT5 terminals share one FILE_COMMON (`Common\Files`) folder, so the EAs use distinct names: MT5 `PAT_device.txt` / `PAT_master_device.txt`, MT4 `PAT_device_mt4.txt` / `PAT_master_device_mt4.txt`. Never share these between terminals — each file holds one device's cloud identity.

### Signal delivery guarantees

- **Fail-closed**: only signals with `Executable == true` are ever queued for devices; advisory / gate-blocked / entitlement-unsatisfied signals never reach the EA.
- **Plan-gated (two layers)**: the engine filters at enqueue time (license status + license `allowed_strategies` + plan `allowed_strategies` in SQL) and the control plane re-checks at poll time — a license revoked or plan downgraded between enqueue and poll expires the queued signal.
- **Always-ACK**: every polled item is ACKed (`{"status":"PROCESSED",...}`) so it leaves the queue permanently; stale IN_FLIGHT items (EA crash mid-batch) are reclaimed after 30s; signals past TTL expire.
- **Server-enforced SL**: a trade without a stop-loss is auto-closed server-side (`CLOSE_POSITION` command arrives on the same poll channel).

### Master (data) EA

Point it at the same license infrastructure (`MasterLicenseKey` input); it activates with `role=data` and streams `MASTER_TICK`/`MARKET_SNAPSHOT` to `POST /ingest/agent`. Engine recovery nudges (`REQUEST_SNAPSHOT`) arrive on the edge queue and force an immediate snapshot. The engine never fabricates ticks — if the Master EA stops streaming, the feed reports `NO_DATA`.

### Troubleshooting

| Symptom | Fix |
|---------|-----|
| `WebRequest not allowed` in Experts log | Add `https://api.predictatrade.com` to the WebRequest allowlist (step above), restart terminal. |
| `Device activation failed: HTTP 4xx` | License key wrong / max devices reached — check the MetaTrader Client page. |
| `edge-heartbeat failed: HTTP 401` repeatedly | Device secret rotated — delete the EA's state file (MT5 `PAT_device.txt`, MT4 `PAT_device_mt4.txt` in the common folder) and re-enter the license key. |
| `array out of range in 'Predict-A-Trade.mq4'` / EA removes itself | Fixed in v1.19.x — MT4 Client parsed a FILE_COMMON state file written by the MT5 EA into an unsized array. Update both MT4 EAs from downloads (state files now per-platform `*_mt4.txt`); if a stale shared file remains, delete `PAT_device.txt` / `PAT_master_device.txt` from the common folder so each terminal re-activates under its own identity. |
| `ingest failed: HTTP 401` repeatedly | Access token missing/stale — the EA refreshes it automatically; if the storm persists, remove + re-attach the EA (re-activation mints a fresh JWT). Server-side: `[INGEST-AUTH]` lines in `docker logs pat-realtime` show the exact rejection reason (never token material). |
| Dashboard "Market Feed Stale" | The Master EA's snapshots are not reaching the engine. Check: (1) nginx routes `POST /ingest/agent` to the Go engine (a `{"service":"Predict-A-Trade API"...}` banner answer means nginx served the catch-all — run `nginx -s reload` after config changes), (2) `access_token` must be a real JWT (3 dot-separated parts) — opaque strings 401 with `invalid token format`, (3) refreshed Master JWTs must carry `role: data` (`licensing.devices.role`, migration 119). |
| Device Online but no signals | Engine has no executable signals for your plan's strategies; check the dashboard signal feed and your plan's `allowed_strategies`. |
| Dashboard shows Offline | EA not polling — check WebRequest allowlist and that the terminal is running with AutoTrading/Algo Trading enabled. |