# Predict-A-Trade XAUUSD — Windows Agent

The Windows Agent bridges your MetaTrader 4/5 terminals with the Predict-A-Trade real-time engine. It collects market data (ticks, candles, indicators) from the Master Node EA and delivers trading signals to execution EAs.

## Quick Start

### Install (for subscribers)
```powershell
irm https://downloads.predictatrade.com/windows-agent/install.ps1 | iex
```

### Update (same command — detects existing install and updates)
```powershell
irm https://downloads.predictatrade.com/windows-agent/install.ps1 | iex
```

### Uninstall
```powershell
irm https://downloads.predictatrade.com/windows-agent/uninstall.ps1 | iex
```

### Check Health
Open in browser: `http://127.0.0.1:9000`

---

## What Gets Installed

| Item | Location |
|------|----------|
| Agent binary | `C:\PredictATrade\XAUUSD\pat-agent.exe` |
| Config file | `C:\PredictATrade\XAUUSD\settings.json` |
| Support scripts | `C:\PredictATrade\XAUUSD\health-check.ps1`, `status.ps1`, `notify.ps1` |
| Log files | `C:\PredictATrade\XAUUSD\logs\` |
| Agent logs (service mode) | `C:\ProgramData\PredictATrade\logs\agent.log` |
| Device identity | `C:\ProgramData\PredictATrade\device.key` |
| Windows Service | `pat-agent` (auto-start, auto-restart on crash) |
| Scheduled Task | `PredictATradeHealthCheck` (runs every 1 min) |
| Event Log Source | `pat-agent` (Application log) |
| Defender Exclusion | `C:\PredictATrade` folder + `pat-agent.exe` process |

## Windows Service

The agent runs as a **native Windows Service** (not NSSM). It communicates directly with the Windows Service Control Manager using `golang.org/x/sys/windows/svc`.

| Property | Value |
|----------|-------|
| Service name | `pat-agent` |
| Display name | Predict-A-Trade XAUUSD Agent |
| Start type | Automatic |
| Recovery | Restart after 5s, 10s, 30s (3 attempts, reset every 24h) |
| Runs as | LocalSystem |

### Service Commands
```powershell
# Check status
Get-Service pat-agent

# Start
Start-Service pat-agent

# Stop
Stop-Service pat-agent

# Restart
Restart-Service pat-agent

# View in Task Manager
# Task Manager → Services tab → pat-agent
```

## Health Endpoint

The agent serves a local-only HTTP status page on `http://127.0.0.1:9000`.

| URL | Returns |
|-----|---------|
| `http://127.0.0.1:9000/` | HTML status dashboard (auto-refreshing every 5s) |
| `http://127.0.0.1:9000/status` | Same HTML dashboard |
| `http://127.0.0.1:9000/health` | JSON health check (200 OK = healthy) |
| `http://127.0.0.1:9000/api/status` | Full JSON status snapshot |

The port can be changed with environment variable `PAT_HEALTH_PORT=9001`.

## MetaTrader Setup

### Master Node EA (data collection — no license needed)
1. Open MetaTrader 4 or 5
2. Attach `PredictATrade_MasterNode_MT4.mq4` or `PredictATrade_MasterNode_MT5.mq5` to an XAUUSD chart
3. The EA collects ticks, candles, and indicators and sends them to the agent via IPC files
4. No license key required — data collection only

### Execution EA (signal execution — license required)
1. Open MetaTrader 4 or 5
2. Attach `PredictATrade_MT4.mq4` or `PredictATrade_MT5.mq5` to an XAUUSD chart
3. Enter your license key in the EA input parameters
4. The EA receives signals from the agent and executes trades with server-provided SL/TP

### EA Version History
| Version | Features |
|---------|----------|
| v1.08 | TRADE_RESULT reporting, wrong-side SL rejection, watchdog, partial TP1/TP2/3 |
| v1.09 | + CLOSE_POSITION handler, EMERGENCY_STOP handler, KILL_SWITCH handler, position SL in snapshot |

## Configuration

The `settings.json` file in the install directory contains:

```json
{
  "live_ws_url": "wss://live.predictatrade.com/ws/v1/agent",
  "api_url": "https://api.predictatrade.com/api/v1",
  "license_key": "",
  "health_check_interval_minutes": 1,
  "notifications": {
    "telegram": { "enabled": false, "bot_token": "", "chat_id": "" },
    "discord": { "enabled": false, "webhook_url": "" },
    "email": { "enabled": false, "smtp_host": "", "smtp_port": 587 }
  }
}
```

### Environment Variables (optional overrides)
| Variable | Default | Description |
|----------|---------|-------------|
| `PAT_LIVE_WS_URL` | `wss://live.predictatrade.com/ws/v1/agent` | WebSocket server URL |
| `PAT_API_URL` | `https://api.predictatrade.com/api/v1` | REST API base URL |
| `PAT_LICENSE_KEY` | (from settings.json) | License key |
| `PAT_HEALTH_PORT` | `9000` | Health endpoint port |
| `PAT_DATA_DIR` | `C:\ProgramData\PredictATrade` | Data directory |
| `PAT_IPC_DIR` | (auto-detected) | Override IPC file directory |
| `PAT_UPDATE_CHANNEL` | `STABLE` | Update channel |

## Troubleshooting

### Service won't start
1. Check log file: `C:\ProgramData\PredictATrade\logs\agent.log`
2. Check Windows Event Viewer → Application → Source: `pat-agent`
3. Try running manually: Open Command Prompt → `C:\PredictATrade\XAUUSD\pat-agent.exe`
4. Check if port 9000 is in use: `netstat -an | findstr :9000`
5. Check Windows Defender: Security → Protection history → look for blocked items

### Health endpoint not responding
1. Verify service is running: `Get-Service pat-agent`
2. Check if port 9000 is in use by another process
3. Try different port: set `PAT_HEALTH_PORT=9001` environment variable
4. Check agent log: `C:\ProgramData\PredictATrade\logs\agent.log`

### No data feed from MetaTrader
1. Verify Master Node EA is attached to XAUUSD chart in MetaTrader
2. Check EA is enabled (smiley face in top-right of chart)
3. Verify IPC files exist in MetaQuotes Common\Files folder (look for `PAT_ticks.txt`)
4. Check MetaTrader Journal tab for EA errors
5. Restart the agent service: `Restart-Service pat-agent`

### No signals received
1. Verify Execution EA is attached with correct license key
2. Check license status in health endpoint: `http://127.0.0.1:9000/api/status`
3. Verify allowed strategies in license match EA strategy selection
4. Check agent log for WebSocket connection status
5. Ensure MetaTrader has AutoTrading enabled

### Windows Defender blocking the agent
1. The installer adds a Defender exclusion for `C:\PredictATrade` automatically
2. If still blocked: Windows Security → Virus & threat protection → Protection history → Allow on device
3. Or manually add exclusion: `Add-MpPreference -ExclusionPath "C:\PredictATrade"`

### After uninstall, data feed continues
1. The uninstaller kills the agent process and cleans up IPC files
2. BUT the MetaTrader EAs are still running — you must remove them manually:
   - Open MetaTrader
   - Right-click the chart → Expert Advisors → Remove
   - Repeat for each terminal (Master Node + execution EAs)
3. If feed still continues, close and reopen MetaTrader

## Server-Side SL Enforcement

The agent supports server-side commands for capital protection:

| Command | When | Action |
|---------|------|--------|
| `CLOSE_POSITION` | EA executes trade without SL | Closes the specific position |
| `EMERGENCY_STOP` | Capital protection triggered | Closes ALL PAT positions + halts |
| `KILL_SWITCH` | Security incident | Closes all + stops EA + disconnects |

## Auto-Update

The agent checks for updates every hour by downloading `update-manifest.json` from the server. If a new version is available, it:
1. Downloads the new `pat-agent.exe`
2. Verifies SHA256 checksum
3. Stops the service, swaps the binary, restarts

Manual update: just re-run the install command — it handles updates automatically.

## Build (for developers)

### Cross-compile from Linux/macOS
```bash
cd /srv/predictatrade/xauusd
./scripts/build-windows-agent.sh --bump
```

This produces:
- `windows-agent/bin/pat-agent.exe` — Client binary (execution)
- `windows-agent/bin/pat-master.exe` — Master Node binary (data-only)
- `windows-agent/deploy/client/pat-agent.exe` — Client copy for nginx serving
- `windows-agent/deploy/master/pat-master.exe` — Master Node copy for nginx serving
- `windows-agent/deploy/version.txt` — shared version number
- `windows-agent/deploy/update-manifest.json` — Client checksum + metadata
- `windows-agent/deploy/master/update-manifest.json` — Master Node checksum + metadata

The two roles ship as **separate binaries** built from distinct entrypoints
(`cmd/client` and `cmd/master`); the role is fixed by the binary, not a runtime
flag. See `docs/guides/WINDOWS_AGENT.md` for install/update/uninstall details.

### Build flags
```
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o pat-agent.exe ./cmd/client/
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o pat-master.exe ./cmd/master/
```

No `.syso` manifest file is used — the binary is a plain Go executable. The Windows Service protocol is handled by `golang.org/x/sys/windows/svc` in `internal/service.go`.

## Architecture

```
MetaTrader 4/5 Terminal
  ├── Master Node EA (data collection, no license)
  │     └── writes ticks/candles/indicators to IPC files
  │
  └── Execution EA (signal execution, license required)
        ├── reads signals from IPC files
        ├── executes trades with server-provided SL/TP
        └── writes EXECUTION_ACK + TRADE_RESULT to IPC files
              │
              ▼
    MetaQuotes Common\Files (IPC)
              │
              ▼
    Windows Agent (TWO separate binaries)
    ├── Master Node (pat-master.exe, data-only)
    │     ├── reads IPC ticks/candles from Master Node EA
    │     ├── forwards ticks to Go RT engine (data WS, port 13091)
    │     ├── serves health endpoint on :9001
    │     └── never executes — ignores all execution server messages
    └── Client Agent (pat-agent.exe, execution)
          ├── reads IPC from Execution EA + writes signals to IPC
          ├── receives signals from Go RT engine (exec WS, port 13081)
          ├── forwards CLOSE_POSITION/EMERGENCY_STOP/KILL_SWITCH to EA
          ├── serves health endpoint on :9000
          └── auto-updates from download server (role-specific manifest)
              │
              ▼
    Go Real-Time Engine (server)
    ├── processes ticks → indicators → strategies → signals
    ├── verifies EXECUTION_ACK (SL enforcement)
    ├── monitors position SLs via broker snapshot
    └── sends CLOSE_POSITION/EMERGENCY_STOP/KILL_SWITCH if needed
```

## Version History

| Version | Date | Key Changes |
|---------|------|-------------|
| v1.2.16 | Aug 24 | Terminal auto-recovery, signal delivery fixes |
| v1.2.17 | Aug 24 | TRADE_RESULT forwarding |
| v1.2.18 | Aug 25 | CLOSE_POSITION/EMERGENCY_STOP/KILL_SWITCH forwarding |
| v1.2.19 | Aug 25 | License validation sends MT account info |
| v1.2.20 | Aug 25 | Manifest update (Simha FinTech publisher) |
| v1.2.22 | Aug 25 | Fix device key panic on corrupt files |
| v1.2.23 | Aug 25 | Remove .syso manifest (was causing App Control block) |
| v1.2.24 | Aug 25 | Use native Windows Service (svc.Run) instead of NSSM |
| v1.2.25 | Aug 25 | Fix sc.exe path quoting + file logging in service mode |
| v1.2.32 | Aug 27 | Split into two separate binaries: Client (`pat-agent.exe`, `cmd/client`) and Master Node (`pat-master.exe`, `cmd/master`). Role fixed by binary; no runtime `--mode`. Distinct `deploy/client` + `deploy/master` artifacts with per-role update manifests. |

## Support

- **Documentation**: https://predictatrade.com/docs
- **Support email**: support@predictatrade.com
- **GitHub**: https://github.com/simhaonline/predictatrade
- **Status page**: https://status.predictatrade.com

## License

Copyright (c) 2026 Simha FinTech. All rights reserved.

Predict-A-Trade is a trademark of Simha FinTech. Unauthorized copying, distribution, or modification of this software is prohibited.
