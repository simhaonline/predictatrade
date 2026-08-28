# Windows Agent Guide
## v1.17.2 — 28 August 2026 · Agent v1.2.35

### Overview

The Windows Agent bridges MetaTrader 4/5 (MT4/MT5) terminals with the Predict-A-Trade real-time engine. It runs as a native Windows Service and is installed with a single PowerShell command. There are **two roles** that can be installed on the same machine (or separate machines) without conflict:

| Role | Purpose | Binary | Windows Service | Engine Port | Engine URL | Health Port |
|------|---------|--------|-----------------|:-----------:|------------|:-----------:|
| **Client Agent** | Receives signals and **places/closes XAUUSD orders** (execution). | `pat-agent.exe` | `pat-agent-client` | 13081 (exec) | `wss://<host>/ws/v1/agent` | 9000 |
| **Master Node** | Streams market/structure **data only** — never executes. | `pat-master.exe` | `pat-agent-master` | 13091 (data) | `wss://<host>/ws/v1/data` | 9001 |

Both roles ship as **separate Go binaries** — `pat-agent.exe` (Client) and `pat-master.exe` (Master Node) — built from distinct entrypoints (`cmd/client` and `cmd/master`). The role is **fixed by the binary itself**, not a runtime flag, so a Client and a Master Node can run side-by-side on one Windows box with no ambiguity. Each role is published for **three architectures** (`386`, `amd64`, `arm64`) under its own subfolder, with a per-arch `update-manifest.json` for auto-update.

> Default engine host is `live.predictatrade.com`. To point at a different server, run the installer with `-EngineHost your.host`.

---

## 1. Install

Each role has a **dedicated install URL** under its own subfolder on the download server, so the role-specific binary is fetched from an unambiguous location while shared assets (NSSM, settings, scripts, version) always come from the root.

### Client Agent (execution)
```powershell
irm https://downloads.predictatrade.com/windows-agent/client/install.ps1 | iex
```

### Master Node (data-only)
```powershell
irm https://downloads.predictatrade.com/windows-agent/master/install.ps1 | iex
```

### Shared installer (role selected via `-Mode`)
```powershell
irm https://downloads.predictatrade.com/windows-agent/install.ps1 | iex   # -Mode client (default) | master
```

The installer:
1. Self-elevates to Administrator (UAC prompt).
2. Creates `C:\PredictATrade\XAUUSD\` and downloads the role binary (`pat-agent.exe` or `pat-master.exe`).
3. Persists the engine WebSocket URL as a machine environment variable (`PAT_LIVE_WS_URL` for client, `PAT_DATA_WS_URL` for master).
4. Sets a unique local health port (`PAT_HEALTH_PORT` = 9000 client / 9001 master) so both roles can coexist.
5. Installs the Windows Service (auto-start, restart on crash) via NSSM (fallback: `sc.exe`).
6. Verifies the local health endpoint.

### Download URL scheme (BaseUrl / RootUrl)
The shared `install.ps1` distinguishes two URLs:
- **`$BaseUrl`** — may be overridden to a role subdir (`…/client` or `…/master`); the **role binary** is fetched from here.
- **`$RootUrl`** — always the root (`…/windows-agent`); **shared assets** (NSSM, `settings.json`, `health-check.ps1`, `status.ps1`, `notify.ps1`, `version.txt`) are fetched from here.

The thin role wrappers (`client/install.ps1`, `master/install.ps1`) download the shared installer and re-run it elevated with `-BaseUrl …/client` or `…/master`.

### MetaTrader EA setup
1. **Master Node EA** — attach to an **XAUUSD** chart (data collection; no license required).
2. **Execution EA** — attach to an **XAUUSD** chart with your license key (places/closes trades from signals).

> The EA can be on **any chart timeframe** (M1/M5/M15/H1…). Execution is by symbol + price levels, not chart timeframe, so a client chart on M15 still executes an M5 signal correctly.

#### Execution EA input parameters
| Input | Default | Description |
|-------|:-------:|-------------|
| `AutoExecute` | **false** | When `true`, the EA auto-executes received signals. Default `false` = **signal-only** (display signals; you place trades manually). |
| `ExecuteCandidates` | false | When `true` (and `AutoExecute=true`), candidate signals are also executed as real trades. |
| `BypassDailyLossBlock` | false | When `true`, the EA keeps trading past the **soft** daily-loss limit. The **hard** halt at `MaxDailyLossPct` is **never** bypassed. Use with caution. |

#### EA-side daily-loss guard (capital protection)
The Execution EA enforces its own daily-loss guard, independent of the server-side risk gates:
- **Soft limit (`WarningLossPct`)** — blocks only *new* entries; it re-evaluates every tick and **recovers** (unblocks) intraday if the daily loss recedes below the limit. The day boundary is the **broker/server day** (not UTC), and the day-open balance is derived from realized P&L so an EA re-attach mid-day does not overstate the loss.
- **Hard limit (`MaxDailyLossPct`)** — closes **all** positions as an emergency backstop. This is **never** bypassable, even with `BypassDailyLossBlock=true`.

#### Client terminal logs
The EA writes prefixed lines to the MT4/MT5 **Experts** log and to `error.log` in the MetaQuotes `Common\Files` folder (all times are **broker/server time**, not UTC):
- `[Predict-A-Trade] STATUS: Access Granted | License: ACTIVE | Subscription: ELITE` (printed only when license state changes)
- `[Predict-A-Trade] SIGNAL RECEIVED | Symbol: XAUUSD | Type: BUY | Price: … | Lot: …`
- `[Predict-A-Trade] CAPITAL | dayOpenBal: … | dailyPnL: … | lossPct: … | status: BLOCKED/RESUMED …`
- `[Predict-A-Trade] CAPITAL DEAL #n | date(Broker): … | profit: … | swap: … | commission: …` (each deal counted as "today" when the block triggers — use to verify which deals feed the daily loss)
- License *strategy* detail is intentionally omitted from these terminal logs.

---

## 2. Update

Re-run the **same** installer command for the role you want to update. It downloads the latest binary, stops the existing service, swaps the exe, and restarts — preserving `settings.json`.

```powershell
# Update Client
irm https://downloads.predictatrade.com/windows-agent/client/install.ps1 | iex

# Update Master
irm https://downloads.predictatrade.com/windows-agent/master/install.ps1 | iex
```

---

## 3. Uninstall

The uninstaller is role-aware but **always cleans up BOTH roles** (client + master) so a default uninstall never leaves a Master Node service running behind. `-Mode` only affects messaging, not what gets removed.

```powershell
# Uninstall (default — removes both roles)
irm https://downloads.predictatrade.com/windows-agent/uninstall.ps1 | iex

# Silent mode (no prompts, removes everything)
irm "https://downloads.predictatrade.com/windows-agent/uninstall.ps1?Silent=true" | iex
```

The uninstaller stops and deletes the Windows Service(s), kills the agent process(es), cleans IPC files in the MetaQuotes `Common\Files` folder, removes the scheduled-task/event-log source, removes Defender exclusions, and (optionally) the install directory.

---

## 4. Status & Health

Both roles expose a local health endpoint (HTTP 200 = healthy):
- **Client:** `http://127.0.0.1:9000/health`
- **Master:** `http://127.0.0.1:9001/health`

Open the root URL (`http://127.0.0.1:9000/` or `:9001/`) — or `/status` — for a human-readable, auto-refreshing HTML dashboard. The cards shown are **role specific**:

- **Master Node (data, port 9001)** shows:
  - **MASTER NODE (Terminal)** — MT4/MT5 terminal link, license, plan.
  - **CANDLE DELIVERY → Engine** — backend data-WS status, candles delivered count, last candle timestamp, clock drift.
- **Client Agent (exec, port 9000)** shows:
  - **CLIENT (EA / MetaTrader)** — MT4/MT5 terminal link, license, plan.
  - **SIGNAL DELIVERY → EA** — backend exec-WS status, signals delivered count, last-signal timestamp.

JSON equivalents: `/api/status` (full snapshot, includes `mode`, `candles_delivered`, `signals_delivered`, etc.).

```powershell
# Client status
irm https://downloads.predictatrade.com/windows-agent/status.ps1 | iex   # add -Mode master for Master Node
```

Or locally:
```powershell
& "C:\PredictATrade\XAUUSD\status.ps1"              # client
& "C:\PredictATrade\XAUUSD\status.ps1" -Mode master  # master
```

Check the Windows Service:
```powershell
Get-Service pat-agent-client      # client
Get-Service pat-agent-master      # master
```

---

## 5. Install Location

| Item | Path |
|------|------|
| Client binary | `C:\PredictATrade\XAUUSD\pat-agent.exe` |
| Master binary | `C:\PredictATrade\XAUUSD\pat-master.exe` |
| Config | `C:\PredictATrade\XAUUSD\settings.json` |
| Logs | `C:\PredictATrade\XAUUSD\logs\` (client `agent.log`, master `master_agent.log`) |
| Service logs | `C:\ProgramData\PredictATrade\logs\agent.log` |
| Device key | `C:\ProgramData\PredictATrade\device.key` |

---

## 6. Deploy Files

The download server serves `windows-agent/deploy/` at `https://downloads.predictatrade.com/windows-agent/`. The folder is organized into a shared root plus two role subfolders:

| File | Purpose |
|------|---------|
| `install.ps1` | Shared installer (role selected via `-Mode client\|master`). |
| `install-client.ps1` | Thin wrapper → installs `pat-agent-client` (exec, port 13081). |
| `install-master.ps1` | Thin wrapper → installs `pat-agent-master` (data, port 13091). |
| `uninstall.ps1` | Uninstaller (role-aware via `-Mode client\|master\|all`; always removes both). |
| `status.ps1` | Status report (role-aware via `-Mode`). |
| `health-check.ps1` | Hang/crash monitor (role-aware via `-Mode`); used by Scheduled Task. |
| `verify-cleanup.ps1` | Plain-English post-uninstall verification checklist (self-elevates; reports Master Node + Client Agent separately). |
| `pat-agent.exe` | Client Agent binary. |
| `pat-master.exe` | Master Node binary (separate build from the distinct `cmd/master` entrypoint). |
| `notify.ps1` | Multi-channel notification dispatcher. |
| `settings.json` | Config template (notification + health params). |
| `install.bat` | Batch wrapper for double-click install. |
| `version.txt` | Current version number (single source of truth). |
| `update-manifest.json` | Version + SHA256 for auto-update. |
| `client/{386,amd64,arm64}/` | Per-arch Client Agent binaries + `update-manifest.json`. |
| `master/{386,amd64,arm64}/` | Per-arch Master Node binaries + `update-manifest.json`. |

---

## 7. Safety Features (mandatory, cannot be disabled)

| Feature | Behaviour |
|---------|-----------|
| Server-side SL | Server verifies SL is set; closes position if missing |
| Daily-loss guard (soft) | EA blocks new entries after the soft daily-loss limit; recovers intraday. Bypassable via `BypassDailyLossBlock`. |
| Daily-loss halt (hard) | At `MaxDailyLossPct` the EA closes all positions. Emergency backstop — never bypassable. |
| Max spread gate | Signals blocked if spread exceeds limit |
| Slippage guard | Post-fill slippage check, reports violations |
| Margin check | OrderCalcMargin before every order |
| Martingale ban | MaxLotRatioVsBase = 1.0 (no doubling) |
| License enforcement | EA checks license status, fails closed |

Server-side commands for capital protection: `CLOSE_POSITION`, `EMERGENCY_STOP`, `KILL_SWITCH`.

---

## 8. Notes

- **Same machine, two roles:** install Client then Master (or vice-versa). Distinct service names, binaries, and ports mean they never collide.
- **Paper vs Live:** in live mode the engine reads the client's **real** broker equity from the connected agent, so risk caps are per-client real capital. The `PAT_PAPER_EQUITY` fallback is demo-only and never overrides a live account.
- **Production signing:** code-signing is **optional** — unsigned builds are fully supported. The installer auto-adds a scoped Windows Defender exclusion for the install dir and uses `Unblock-File` to strip the SmartScreen "downloaded from internet" mark, so a self-signed/unsigned agent is not blocked. No certificate is required.
- **Auto-update:** the agent checks `update-manifest.json` (per-arch) and self-updates by downloading the new binary, verifying SHA256, stopping the service, swapping the exe, and restarting. Manual update = re-run the install command.

---

## 9. Troubleshooting

### Service won't start
1. Check log file: `C:\PredictATrade\XAUUSD\logs\agent.log` (or `master_agent.log`).
2. Check Windows Event Viewer → Application → Source: `pat-agent`.
3. Try running manually: `C:\PredictATrade\XAUUSD\pat-agent.exe` (or `pat-master.exe`).
4. Check the health port is free: `netstat -an | findstr :9000` (or `:9001`).

### No data feed from MetaTrader
1. Verify Master Node EA is attached to an XAUUSD chart.
2. Check EA is enabled (smiley face in top-right of chart).
3. Verify IPC files exist in MetaQuotes `Common\Files` (look for `PAT_ticks.txt`).
4. Restart the service: `Restart-Service pat-agent-master`.

### No signals received
1. Verify Execution EA is attached with the correct license key.
2. Check license status: `http://127.0.0.1:9000/api/status`.
3. Verify allowed strategies in the license match the EA strategy selection.
4. Ensure MetaTrader has AutoTrading enabled.

### Windows Defender blocking the agent
1. The installer adds a Defender exclusion for `C:\PredictATrade` automatically.
2. If still blocked: Windows Security → Virus & threat protection → Protection history → Allow on device.
3. Or manually: `Add-MpPreference -ExclusionPath "C:\PredictATrade"`.

### After uninstall, data feed continues
The uninstaller kills the agent and cleans IPC files, but the MetaTrader EAs are still running — remove them manually (right-click chart → Expert Advisors → Remove) for each terminal.

---

## Support

- **Documentation**: https://predictatrade.com/docs
- **Support email**: support@predictatrade.com
- **GitHub**: https://github.com/simhaonline/predictatrade
- **Status page**: https://status.predictatrade.com
