# Predict-A-Trade Windows Agent (`pat-agent`)

Service name: **`pat-agent`** (Windows Service registered via NSSM).
Binary: **`pat-agent.exe`**. Install dir: `C:\Program Files\PredictATrade\XAUUSD`.
Data/config dir: `C:\ProgramData\PredictATrade`. Logs: `%LOCALAPPDATA%\PredictATrade\logs\`.

The agent is a lightweight, signed-signal adapter/guard (per architecture boundaries). It
connects to the server over a **named pipe** (`.\pipe\predictatrade`), receives signed
signals, and runs local execution guards. It must **not** contain server/private signing
credentials or primary predictive intelligence.

## Build & Sign

```powershell
# Windows operator build (embedded version metadata + optional Authenticode)
.\build.ps1                      # dev / self-signed
.\build.ps1 -Version 1.2.5 -NoSign
$env:PAT_SIGN_CERT        = "C:\certs\pat.pfx"
$env:PAT_SIGN_CERT_PASSWORD = "***"
$env:PAT_TIMESTAMP_URL    = "http://timestamp.digicert.com"
.\build.ps1                # production-signed
```

Linux/CI cross-build (Docker pipeline): `scripts/build-windows-agent.sh --bump` produces
`windows-agent/bin/pat-agent.exe` and copies it to `windows-agent/deploy/pat-agent.exe`.
Version metadata is embedded from `cmd/agent/versioninfo.json` + `manifest.xml` via
`goversioninfo` into `cmd/agent/resource_windows_amd64.syso` (manifest identity `pat-agent`).

> Production installs MUST use a legitimate organization code-signing certificate. An
> unsigned or self-signed binary triggers SmartScreen/Defender and may be blocked.

## Install / Upgrade / Uninstall

```powershell
irm https://downloads.predictatrade.com/windows-agent/install.ps1 | iex
irm https://downloads.predictatrade.com/windows-agent/uninstall.ps1 | iex
irm "https://downloads.predictatrade.com/windows-agent/uninstall.ps1?Silent=true" | iex
```

The installer self-elevates (UAC), downloads files, **interactively collects configuration**
(notification type + health params; secrets entered via secure prompt; `settings.json` is
backed up to `settings.json.bak` and ACL-restricted to Administrators + SYSTEM only), removes
stale prior service names (`agent`, `PredictATradeAgent`, `PredictATradeXAUUSD`), installs the
NSSM service with auto-restart (5s on crash), and registers a health-check Scheduled Task.

## Status & Health Endpoint

Local HTTP server on `http://127.0.0.1:9000` (`PAT_HEALTH_PORT` to override).

| Path | Purpose |
|------|---------|
| `/` , `/status` | HTML dashboard (auto-refresh 5s): **Client** panel (pipe connection, masked license key, server URL, last heartbeat, service name) and **Server** panel (last HTTP status code + response body + timestamp). Plus live process/uptime/version. |
| `/health` | Plain JSON health probe consumed by `health-check.ps1`. |
| `/api/status` | Full JSON status object. |

### `/api/status` schema

```json
{
  "agent_version": "1.2.5",
  "build_info": "1.2.5",
  "status": "running",
  "uptime_seconds": 123,
  "install_dir": "C:\\Program Files\\PredictATrade\\XAUUSD",
  "service_name": "pat-agent",
  "pipe": {
    "connected": true,
    "license_key": "PA-PRO-****....****",
    "server_url": "https://api.predictatrade.com",
    "last_heartbeat": "2026-08-21T12:00:00Z"
  },
  "health": { "status": "ok", "uptime_seconds": 123, "timestamp": "..." },
  "server_last_response": { "status_code": 200, "body": "{...}", "timestamp": "..." }
}
```

The license key is **masked** in all outputs (`maskSecret` → `PA-PRO-****....****`) so the
dashboard/logs never expose the full secret.

## Robustness (anti-termination)

- **Crash protection**: every background goroutine is wrapped in `safe()` which recovers panics
  and logs them; `main()` also has a top-level `recover()`. A panic no longer silently kills
  the process without a trail.
- **Startup diagnostic banner** prints version, build info, service name, install dir, OS/arch
  and pipe name so a failed start is self-describing in logs.
- **Health HTTP server** is started in `Start()` (regression fixed — previously never called).
- **Hang detection**: `health-check.ps1` probes `/health` every 60s; on failure it kills the
  process; NSSM restarts it (5s) and `notify.ps1` alerts with exit code `-999`.
- **Windows metadata**: embedded version info + manifest (longPathAware) reduces SmartScreen
  friction and UAC virtualization issues.

## Troubleshooting

| Symptom | Check |
|---------|-------|
| Service won't start / blocked | Ensure binary is Authenticode-signed with a valid cert; check Event Log source `pat-agent`. |
| `127.0.0.1:9000` not responding | Verify service running (`nssm status pat-agent`); check pipe connection; view dashboard `/status`. |
| Stale `agent` service present | Installer removes `agent`/`PredictATradeAgent`/`PredictATradeXAUUSD`; if manual, `nssm remove <name> confirm`. |
| Secret exposed | License key is masked everywhere; `settings.json` ACL-restricted to Administrators + SYSTEM. |

## Deviation Notes (vs `go-prompt.md`)

- **Service registration**: `go-prompt.md` prefers a native Windows service (`golang.org/x/sys/windows/svc`)
  with no external dependency. The current build uses **NSSM** because `main.go` does not yet
  wire `svc.Run`/Handler and NSSM cannot be validated in this environment. A native service
  implementation exists in `internal/service.go` and should replace NSSM in a follow-up.
- **Resource tooling**: `winres` CLI was unavailable; version metadata is embedded via
  `goversioninfo` from `versioninfo.json` + `manifest.xml` (functionally equivalent). `winres.json`
  is retained as the canonical metadata source for operators who prefer `winres`.
- **Service name**: kept as `pat-agent` per product requirement (overrides the `PredictATradeAgent`
  name referenced in `go-prompt.md`).
