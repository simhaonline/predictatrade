# Predict-A-Trade XAUUSD — Windows Agent Deployment

## Subscriber Quick Start

### Install
```powershell
irm https://downloads.predictatrade.com/windows-agent/install.ps1 | iex
```

### Update (same command)
```powershell
irm https://downloads.predictatrade.com/windows-agent/install.ps1 | iex
```

### Uninstall
```powershell
irm https://downloads.predictatrade.com/windows-agent/uninstall.ps1 | iex
```

### Check Health
Open: `http://127.0.0.1:9000`

## Deploy Files

| File | Purpose |
|------|---------|
| `install.ps1` | Installer with UAC elevation, downloads, service creation |
| `uninstall.ps1` | Uninstaller — stops service, kills processes, cleans IPC files |
| `pat-agent.exe` | The Go agent binary (v1.2.25) |
| `health-check.ps1` | Hang detection monitor (runs via Scheduled Task every 60s) |
| `notify.ps1` | Multi-channel notification dispatcher (Telegram/Discord/Email) |
| `status.ps1` | Quick status check script |
| `settings.json` | Configuration template (notification credentials + health params) |
| `install.bat` | Batch wrapper for double-click installation |
| `version.txt` | Current version number |
| `update-manifest.json` | Version + SHA256 checksum for auto-update |

## Install Location

| Item | Path |
|------|------|
| Agent binary | `C:\PredictATrade\XAUUSD\pat-agent.exe` |
| Config | `C:\PredictATrade\XAUUSD\settings.json` |
| Logs | `C:\PredictATrade\XAUUSD\logs\` |
| Service logs | `C:\ProgramData\PredictATrade\logs\agent.log` |
| Device key | `C:\ProgramData\PredictATrade\device.key` |

## Windows Service

The agent runs as a native Windows Service (no NSSM required).

- Service name: `pat-agent`
- Start type: Automatic
- Recovery: Restart after 5s/10s/30s
- Health endpoint: `http://127.0.0.1:9000`

## MetaTrader Setup

1. **Master Node EA** — Attach to XAUUSD chart (data collection, no license needed)
2. **Execution EA** — Attach to XAUUSD chart with license key (signal execution)

EA files are in the `mql/` directory of the repository:
- `mql/mt5/PredictATrade_MasterNode_MT5.mq5` — Master Node (MT5)
- `mql/mt5/PredictATrade_MT5.mq5` — Execution EA (MT5)
- `mql/mt4/PredictATrade_MasterNode_MT4.mq4` — Master Node (MT4)
- `mql/mt4/PredictATrade_MT4.mq4` — Execution EA (MT4)

## Troubleshooting

See the main [README.md](../README.md) in the windows-agent directory for full troubleshooting guide.
