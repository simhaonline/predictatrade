# Predict-A-Trade XAUUSD — Windows Agent Deployment

The Windows Agent bridges MetaTrader 4/5 (MT4/MT5) and the Predict-A-Trade
real-time engine. It runs as a native Windows Service and is installed with a
single PowerShell command. There are **two roles** that can be installed on the
same machine (or separate machines) without conflict:

| Role | Purpose | Binary | Windows Service | Engine Port | Engine URL |
|------|---------|--------|-----------------|-------------|------------|
| **Client Agent** | Receives signals and **places/Closes XAUUSD orders** (execution). | `pat-agent.exe` | `pat-agent-client` | **13081** (exec) | `wss://<host>:13081/ws` |
| **Master Node** | Streams market/structure **data only** — never executes. | `pat-master.exe` | `pat-agent-master` | **13091** (data) | `wss://<host>:13091/ws/v1/data` |

Both roles ship as **separate Go binaries** — `pat-agent.exe` (Client, built
from `cmd/client`) and `pat-master.exe` (Master Node, built from `cmd/master`).
The role is fixed by the binary itself (no runtime `--mode` flag). This lets a
Client and a Master Node run side-by-side on one Windows box.

> Default engine host is `live.predictatrade.com`. To point at a different
> server, run the installer with `-EngineHost your.host`.

---

## 1. Install

### Client Agent (execution)
```powershell
irm https://downloads.predictatrade.com/windows-agent/install-client.ps1 | iex
```

### Master Node (data-only)
```powershell
irm https://downloads.predictatrade.com/windows-agent/install-master.ps1 | iex
```

The installer:
1. Self-elevates to Administrator (UAC prompt).
2. Creates the role-specific directory and downloads the role binary
   (`pat-agent.exe` or `pat-master.exe`):
   - **Client Agent** → `C:\PredictATrade\Client\`
   - **Master Node** → `C:\PredictATrade\Master\`
   The two roles live in **separate folders** so a Master Node and a Client
   Agent can coexist on one device without sharing binaries/settings/logs.
3. Persists the engine WebSocket URL as a machine environment variable
   (`PAT_SERVER_URL` for client, `PAT_DATA_WS_URL` for master).
4. Installs the Windows Service (auto-start, restart on crash) using NSSM. NSSM
   is **verified/reused** if one already exists on the device (PATH, the cached
   `C:\ProgramData\PredictATrade\nssm.exe`, or the other role's folder) before
   any download — so reinstalling/co-installing both roles never clobbers a
   working nssm.exe.
5. Verifies the local health endpoint.

### MetaTrader EA setup
1. **Master Node EA** — attach to an **XAUUSD** chart (data collection; no license required).
2. **Execution EA** — attach to an **XAUUSD** chart with your license key (places/Closes trades from signals).

> The EA can be on **any chart timeframe** (M1/M5/M15/H1…). Execution is by
> symbol + price levels, not chart timeframe, so a client chart on M15 still
> executes an M5 signal correctly.

---

## 2. Update

Re-run the **same** installer command for the role you want to update. It
downloads the latest binary, stops the existing service, swaps the exe, and
restarts — preserving `settings.json`.

```powershell
# Update Client
irm https://downloads.predictatrade.com/windows-agent/install-client.ps1 | iex

# Update Master
irm https://downloads.predictatrade.com/windows-agent/install-master.ps1 | iex
```
*(Use the correct host: `downloads.predictatrade.com`.)*

---

## 3. Uninstall

The uninstaller is role-aware. Run it elevated and pass the role:

```powershell
# Uninstall Client only
irm https://downloads.predictatrade.com/windows-agent/uninstall.ps1 | iex
#   (then, when prompted, it targets the client; or pass -Mode client)

# Uninstall Master only
irm https://downloads.predictatrade.com/windows-agent/uninstall.ps1 | iex
#   pass -Mode master

# Uninstall BOTH client + master
irm https://downloads.predictatrade.com/windows-agent/uninstall.ps1 | iex
#   pass -Mode all
```

Silent mode (no prompts, removes everything):
```powershell
irm "https://downloads.predictatrade.com/windows-agent/uninstall.ps1?Silent=true" | iex
```

The uninstaller stops and deletes the role's Windows Service, kills the agent
process, cleans IPC files in the MetaQuotes `Common\Files` folder, removes the
scheduled-task/event-log source, and (optionally) the install directory. It ends
with a **cleanup-verification report** listing any leftover service, process,
directory, task, event-source, or IPC file.

To independently **prove** the agent is completely gone (after uninstall, or
before a clean reinstall), run the standalone audit:

```powershell
irm https://downloads.predictatrade.com/windows-agent/verify-cleanup.ps1 | iex
```

It exits `0` (PASS) only when no Predict-A-Trade agent remnants are detected.

---

## 4. Status & Health

Both roles expose a local health endpoint (HTTP 200 = healthy):
- **Client:** `http://127.0.0.1:9000/health`
- **Master:** `http://127.0.0.1:9001/health`

(The health port is set via `PAT_HEALTH_PORT` so both roles can run on one
machine without a bind conflict.)

Open the root URL (`http://127.0.0.1:9000/` or `:9001/`) — or `/status` — for a
human-readable, auto-refreshing HTML dashboard. The cards shown are **role
specific**; there is deliberately **no generic "SERVER (Backend)" card**:

- **Master Node (data, port 9001)** shows:
  - **MASTER NODE (Terminal)** — MT4/MT5 terminal link, license, plan.
  - **CANDLE DELIVERY → Engine** — backend data-WS status, candles delivered
    count, last candle timestamp, and clock drift. This is the live
    data-delivery telemetry for the broker TF candles the engine consumes.
- **Client Agent (exec, port 9000)** shows:
  - **CLIENT (EA / MetaTrader)** — MT4/MT5 terminal link, license, plan.
  - **SIGNAL DELIVERY → EA** — backend exec-WS status, signals delivered count,
    and last-signal timestamp. The server/backend connection is shown only as
    the signal-delivery channel, not as a separate server card.

JSON equivalents: `/api/status` (full snapshot, includes `mode`,
`candles_delivered`, `signals_delivered`, etc.).

```powershell
# Client status
irm https://downloads.predictatrade.com/windows-agent/status.ps1 | iex   # add -Mode master for Master Node
```

Or locally:
```powershell
& "C:\PredictATrade\Client\status.ps1"            # client
& "C:\PredictATrade\Master\status.ps1" -Mode master   # master
```

Check the Windows Service:
```powershell
Get-Service pat-agent-client      # client
Get-Service pat-agent-master     # master
```

---

## 5. Install Location

| Item | Client Agent path | Master Node path |
|------|-------------------|------------------|
| Binary | `C:\PredictATrade\Client\pat-agent.exe` | `C:\PredictATrade\Master\pat-master.exe` |
| Config | `C:\PredictATrade\Client\settings.json` | `C:\PredictATrade\Master\settings.json` |
| Logs | `C:\PredictATrade\Client\logs\` | `C:\PredictATrade\Master\logs\` |
| Service | `pat-agent-client` | `pat-agent-master` |
| Health | `http://127.0.0.1:9000/health` | `http://127.0.0.1:9001/health` |

Shared (both roles): `C:\ProgramData\PredictATrade\logs\` (service logs),
`C:\ProgramData\PredictATrade\device.key`, and a cached `nssm.exe` at
`C:\ProgramData\PredictATrade\nssm.exe`.

> The legacy single-directory layout (`C:\PredictATrade\XAUUSD`) is **no longer
> used** and is fully removed by `uninstall.ps1 -Mode all`.

---

## 6. Deploy Files

| File | Purpose |
|------|---------|
| `install.ps1` | Shared installer (role selected via `-Mode client|master`). |
| `install-client.ps1` | Thin wrapper → installs `pat-agent-client` (exec, port 13081). |
| `install-master.ps1` | Thin wrapper → installs `pat-agent-master` (data, port 13091). |
| `uninstall.ps1` | Uninstaller (role-aware via `-Mode client|master|all`); ends with a cleanup-verification report. |
| `verify-cleanup.ps1` | Standalone, non-destructive audit proving no agent remnants remain. Checks BOTH roles explicitly (Master Node + Client Agent) plus shared/legacy items (services/processes/dirs/task/event-source/IPC). Optional mode arg: `master`, `client`, or `all` (default). Run after uninstall. |
| `status.ps1` | Status report (role-aware via `-Mode`). |
| `health-check.ps1` | Hang/crash monitor (role-aware via `-Mode`); used by Scheduled Task. |
| `pat-agent.exe` | Client Agent binary. |
| `pat-master.exe` | Master Node binary (separate build from the distinct `cmd/master` entrypoint). |
| `notify.ps1` | Multi-channel notification dispatcher. |
| `settings.json` | Config template (notification + health params). |
| `install.bat` | Batch launcher — delegates to `install.ps1 -Mode` (keeps separate dirs + nssm reuse). |
| `version.txt` | Current version number. |
| `update-manifest.json` | Version + SHA256 for auto-update. |

---

## 7. Notes

- **Same machine, two roles:** install Client then Master (or vice-versa).
  Distinct service names, binaries, and ports mean they never collide.
- **Paper vs Live:** in live mode the engine reads the client's **real** broker
  equity from the connected agent, so risk caps are per-client real capital. The
  `PAT_PAPER_EQUITY` fallback is demo-only and never overrides a live account.
- **Production signing:** builds should use a valid Authenticode certificate.
  Self-signed binaries are acceptable for labelled local dev/test only.
