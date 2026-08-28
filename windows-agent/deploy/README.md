# Predict-A-Trade XAUUSD — Windows Agent Deployment

The Windows Agent bridges MetaTrader 4/5 (MT4/MT5) and the Predict-A-Trade
real-time engine. It runs as a Windows Service and is installed with a
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

The uninstaller **fully removes the agent**. Run it elevated:

```powershell
# Full uninstall — removes BOTH the Client Agent and the Master Node, plus the
# legacy C:\PredictATrade\XAUUSD install, their Windows services, processes,
# scheduled task, event-log source, MetaTrader IPC files and the Defender
# exclusion. -Mode is optional and only changes the on-screen messaging.
irm https://downloads.predictatrade.com/windows-agent/uninstall.ps1 | iex
```

Scope is optional (default is `all`):
```powershell
# Download first if you want to pass -Mode / -Silent explicitly:
$uri = "https://downloads.predictatrade.com/windows-agent/uninstall.ps1"
irm $uri -OutFile $env:TEMP\pat_uninstall.ps1
& $env:TEMP\pat_uninstall.ps1 -Mode all          # all | client | master
& $env:TEMP\pat_uninstall.ps1 -Silent           # no prompts, removes everything
```
*(Note: a query string like `?Silent=true` does NOT reach the script when piped
via `irm | iex`, so download to a file first to use `-Silent`.)*

What the uninstaller does:
- Always stops **and deletes BOTH** `pat-agent-client` and `pat-agent-master`
  services (and any stale legacy service names), using `sc.exe delete` as the
  guaranteed removal path — so the service is gone whether it was NSSM-wrapped or
  a native Windows service.
- Kills any running `pat-agent` / `pat-master` processes.
- Cleans MetaTrader IPC files in `Common\Files`.
- Removes the `PredictATradeHealthCheck` scheduled task, the `pat-agent`
  event-log source, the install directories, and the Defender exclusion.
- Ends with a **cleanup-verification report** listing any leftover service,
  process, directory, task, event-source, or IPC file (and a WARN if something
  needs a reboot).

To independently **prove** the agent is completely gone (after uninstall, or
before a clean reinstall), run the standalone audit:

```powershell
irm https://downloads.predictatrade.com/windows-agent/verify-cleanup.ps1 | iex
```

The script **self-elevates to Administrator** if needed, then checks BOTH roles
explicitly — Master Node (`pat-agent-master` / `pat-master` / `C:\PredictATrade\Master`)
and Client Agent (`pat-agent-client` / `pat-agent` / `C:\PredictATrade\Client`) — plus
shared/legacy items (old `C:\PredictATrade\XAUUSD` folder, scheduled task, event-log
source, Defender exclusion, MetaTrader IPC files). It prints a color-coded checklist
(`[OK]` = nothing found, `[!]` = still present). If anything remains it shows the
exact items and the one-line uninstall command to remove them.

Exit code `0` = CLEAN, `1` = leftovers found. A full report is also saved to
`%TEMP%\pat_verify_cleanup.log`. Optional mode arg: `master`, `client`, or `all` (default).

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
> used** and is fully removed by `uninstall.ps1`.

---

## 6. Deploy Files

| File | Purpose |
|------|---------|
| `install.ps1` | Shared installer (role selected via `-Mode client|master`). |
| `install-client.ps1` | Thin wrapper → installs `pat-agent-client` (exec, port 13081). |
| `install-master.ps1` | Thin wrapper → installs `pat-agent-master` (data, port 13091). |
| `uninstall.ps1` | Uninstaller; always removes BOTH roles + legacy dir, their services (via `sc.exe delete` so it works for NSSM- or native-registered services), task, event source, IPC files and Defender exclusion; ends with a cleanup-verification report. `-Mode` is optional (affects messaging only). |
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
| `<role>/<arch>/<exe>.exe` | Per-architecture binaries (`client`\|`master` × `amd64`\|`386`\|`arm64`) fetched by the installer based on detected Windows arch. |
| `<role>/<arch>/update-manifest.json` | Per-architecture auto-update manifest. |

---

## 7. Notes

- **Same machine, two roles:** install Client then Master (or vice-versa).
  Distinct service names, binaries, and ports mean they never collide.
- **Paper vs Live:** in live mode the engine reads the client's **real** broker
  equity from the connected agent, so risk caps are per-client real capital. The
  `PAT_PAPER_EQUITY` fallback is demo-only and never overrides a live account.
- **Code-signing is OPTIONAL for production.** The agent ships **UNSIGNED** by default
  (no self-signed cert is ever used). To eliminate every AV/SmartScreen prompt, supply a
  CA-issued Authenticode certificate via `PAT_SIGN_CERT`/`PAT_SIGN_CERT_PASSWORD` and
  `build.ps1` applies it automatically. **Without a cert, the unsigned binary is still a
  supported production path:** the installer (1) runs `Unblock-File` to strip the download's
  Mark-of-the-Web — which suppresses the SmartScreen "unrecognized app" prompt — and
  (2) adds a scoped Windows Defender exclusion so the binary is never quarantined. A handful
  of Windows builds may still show a reputation-based SmartScreen prompt; clicking
  **More info → Run anyway** proceeds normally. Self-signed certificates are **NOT** used.
- **Downloads are HTTPS (certbot / Let's Encrypt):** binaries are served over TLS by
  nginx, so the *transport* is authenticated. A TLS server certificate, however, **cannot**
  Authenticode-sign a Windows executable — that requires a separate code-signing certificate,
  which is optional (see above).

---

## 7b. Auto-Update & Multi-Architecture Support

**Auto-update (no manual reinstalls).** Every agent checks its role+arch manifest
(`windows-agent/<role>/<arch>/update-manifest.json`) once an hour. If a newer version is
published, it downloads it over HTTPS, verifies the SHA-256 checksum, then a detached helper
stops the exact Windows service (`pat-agent-client` / `pat-agent-master`), swaps the binary,
and restarts the service. **Clients therefore receive fixes automatically** — you should never
need to ask them to reinstall. The service name is passed to the agent at install time via the
`PAT_SERVICE_NAME` machine env var, so the swap always targets the correct service.

**Multi-arch.** Binaries are built for `amd64`, `386`, and `arm64`. The installer detects the
Windows architecture (`PROCESSOR_ARCHITECTURE` / `PROCESSOR_ARCHITEW6432`) and downloads the
matching binary from `windows-agent/<role>/<arch>/<exe>.exe`. If a per-arch manifest is
missing, the agent falls back to the per-role (amd64) manifest so updates still apply.

**Telemetry.** Each agent sends an `AGENT_TELEMETRY` snapshot (version, role, goarch, MT4/MT5
connectivity, backend connectivity, uptime, candles delivered, license status/plan) to the
realtime engine over its existing WebSocket once per minute. These appear in the backend logs
(`[AGENT-TELEMETRY] ...`) for fleet observability.

---

## 8. Troubleshooting

### EA shows "Access Denied | License: PENDING" even though the license is ACTIVE
The EA reads `PAT_license.txt`, which the agent writes from the server's
license-validation response. `PENDING` (with plan defaulting to `ELITE`) means the
agent's HTTP call to `POST /api/v1/licensing/validate` did **not** return `ACTIVE`.
The most common cause is a **stale `PAT_API_URL` machine environment variable**
pointing at `live.predictatrade.com/api/v1`. That edge host proxies `/api/v1` to the
Go realtime engine, **not** the NestJS control plane, so validation 404s and the
agent records `UNKNOWN` → the EA denies access.

Fix — re-run the installer (it now pins the correct value):
```powershell
irm https://downloads.predictatrade.com/windows-agent/install-client.ps1 | iex
irm https://downloads.predictatrade.com/windows-agent/install-master.ps1 | iex
```
Or set it manually and restart both services:
```powershell
[Environment]::SetEnvironmentVariable("PAT_API_URL","https://api.predictatrade.com/api/v1","Machine")
Restart-Service pat-agent-client; Restart-Service pat-agent-master
```
Confirm via the agent log line `License validation response: HTTP 200 — {... "status":"ACTIVE" ...}`
or the health endpoint's `license` field.
