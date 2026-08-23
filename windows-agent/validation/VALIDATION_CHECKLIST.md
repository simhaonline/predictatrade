# Windows Agent Validation Checklist

**Status: WINDOWS_RUNTIME_VALIDATION_REQUIRED**

This checklist must be executed on a real Windows machine or VM.
The Linux development environment cannot validate Windows runtime behavior.

## Prerequisites
- Windows 10/11 or Windows Server 2019+
- MT4 and/or MT5 installed and running
- Network access to the Predict-A-Trade backend
- Administrator privileges for service installation

## Validation Steps

### 1. Binary Startup
- [ ] Agent binary starts without errors in console mode
- [ ] No crash or panic within 60 seconds
- [ ] Logs are written to `C:\ProgramData\PredictATrade\logs\`

### 2. Service Installation
- [ ] `pat-agent -install` succeeds
- [ ] Service appears in `services.msc`
- [ ] Service startup type is "Automatic"
- [ ] Recovery actions configured (restart on failure)

### 3. Service Start
- [ ] Service starts successfully
- [ ] Service status is "Running"
- [ ] Agent connects to backend WebSocket

### 4. Backend Connectivity
- [ ] Agent authenticates with backend
- [ ] Heartbeat messages sent regularly
- [ ] Agent appears in admin dashboard

### 5. MT5 Connectivity
- [ ] Named pipe `\\.\pipe\PredictATradeMT5` is accessible
- [ ] Tick data flows from MT5 to agent
- [ ] Tick data forwarded to backend

### 6. MT4 Connectivity (if applicable)
- [ ] Named pipe `\\.\pipe\PredictATradeMT4` is accessible
- [ ] Tick data flows from MT4 to agent

### 7. Signal Receipt
- [ ] Agent receives signals from backend WebSocket
- [ ] Signals are delivered to MT5 via named pipe
- [ ] MT5 EA acknowledges signal receipt

### 8. Telemetry
- [ ] Agent status visible in admin dashboard
- [ ] Agent version reported correctly
- [ ] Connection status is "online"

### 9. Service Restart
- [ ] `Restart-Service pat-agent` works
- [ ] Agent reconnects after restart
- [ ] No duplicate signals after restart

### 10. Reboot Persistence
- [ ] Service starts automatically after system reboot
- [ ] Agent reconnects to backend after reboot
- [ ] No data loss after reboot

### 11. Application Update
- [ ] Agent checks for updates
- [ ] Download succeeds with checksum verification
- [ ] Update is applied atomically
- [ ] Service restarts with new version

### 12. Failed Update Rollback
- [ ] If update fails, previous binary is restored
- [ ] Service continues running with old version

### 13. Uninstall
- [ ] `pat-agent -uninstall` succeeds
- [ ] Service removed from `services.msc`
- [ ] Config optionally preserved

### 14. Upgrade from Previous Version
- [ ] Upgrading from older version preserves config
- [ ] Service registration maintained
- [ ] New binary starts correctly

## Running the Validation

```powershell
# Run the automated validation script
.\windows_validation.ps1 -AgentPath "C:\Program Files\PredictATrade\XAUUSD\pat-agent.exe" `
    -BackendURL "https://api.predictatrade.com" `
    -LicenseKey "YOUR_LICENSE_KEY"

# Or run individual tests manually
.\windows_validation.ps1 -AgentPath "C:\path\to\pat-agent.exe"
```

## Result Reporting

After validation, report:
- Which tests passed/failed
- Windows version used
- MT4/MT5 versions used
- Any error messages or logs
- Whether the agent is ready for production use

**Until all tests pass on a real Windows machine, the Windows Agent status remains:**
`WINDOWS_RUNTIME_VALIDATION_REQUIRED`
