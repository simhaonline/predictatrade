# Validate network recovery: disconnect, verify reconnection with bounded backoff
param(
    [string]$AgentPath = "C:\Program Files\PredictATrade\XAUUSD\pat-agent.exe",
    [int]$DisconnectSeconds = 10,
    [int]$ReconnectTimeout = 60
)
$ErrorActionPreference = "Continue"

Write-Host "=== Network Recovery Validation ===" -ForegroundColor Cyan

# Start agent if not running
$svc = Get-Service "pat-agent" -ErrorAction SilentlyContinue
if ($svc -and $svc.Status -ne "Running") {
    Start-Service "pat-agent"
    Start-Sleep 5
}

Write-Host "Step 1: Verify agent is connected..."
$logFile = "C:\ProgramData\PredictATrade\logs\agent.log"
if (Test-Path $logFile) {
    $connected = Select-String -Path $logFile -Pattern "connected|CONNECTED" -SimpleMatch | Select-Object -Last 1
    if ($connected) { Write-Host "[PASS] Agent connected" -ForegroundColor Green }
    else { Write-Host "[PENDING] Cannot verify connection from logs" -ForegroundColor Yellow }
}

Write-Host "Step 2: Simulate network disconnect (disabling network adapter)..."
# WARNING: This requires admin privileges and will disconnect the machine
Write-Host "[MANUAL] Disable network adapter, wait $DisconnectSeconds seconds, re-enable" -ForegroundColor Yellow
Write-Host "[MANUAL] Verify agent reconnects with exponential backoff (1s, 2s, 5s, 10s, 30s)" -ForegroundColor Yellow
Write-Host "[MANUAL] Check agent.log for reconnection sequence" -ForegroundColor Yellow
Write-Host "[MANUAL] Verify no stale signals are executed after reconnection" -ForegroundColor Yellow

Write-Host "Step 3: Verify state resync after reconnection..."
Write-Host "[MANUAL] After reconnection, agent should perform STATE RESYNC" -ForegroundColor Yellow
Write-Host "[MANUAL] Verify no duplicate signals are generated" -ForegroundColor Yellow
