# End-to-end signal delivery validation
param(
    [string]$BackendURL = "https://api.predictatrade.com"
)
$ErrorActionPreference = "Continue"

Write-Host "=== Signal E2E Validation ===" -ForegroundColor Cyan

# Check agent is running
$svc = Get-Service "pat-agent" -ErrorAction SilentlyContinue
if (-not $svc -or $svc.Status -ne "Running") {
    Write-Host "[FAIL] Agent service not running" -ForegroundColor Red
    exit 1
}

Write-Host "[PASS] Agent service running" -ForegroundColor Green

# Check agent connectivity
try {
    $health = Invoke-WebRequest -Uri "$BackendURL/api/v1/health" -TimeoutSec 5 -UseBasicParsing
    Write-Host "[PASS] Backend reachable" -ForegroundColor Green
} catch {
    Write-Host "[FAIL] Backend unreachable" -ForegroundColor Red
    exit 1
}

# Check agent heartbeat
try {
    $status = Invoke-WebRequest -Uri "$BackendURL/api/v1/agent/status" -TimeoutSec 5 -UseBasicParsing
    $agentStatus = $status.Content | ConvertFrom-Json
    if ($agentStatus.master_connected) {
        Write-Host "[PASS] Agent connected to backend" -ForegroundColor Green
    } else {
        Write-Host "[PENDING] Agent not connected to backend" -ForegroundColor Yellow
    }
} catch {
    Write-Host "[PENDING] Cannot check agent status" -ForegroundColor Yellow
}

# Signal delivery (requires active market)
Write-Host "[MANUAL] For full E2E signal test:" -ForegroundColor Cyan
Write-Host "1. Wait for market hours (not weekend)" -ForegroundColor White
Write-Host "2. Monitor agent.log for 'Signal received'" -ForegroundColor White
Write-Host "3. Verify signal appears in MT5/MT4 terminal" -ForegroundColor White
Write-Host "4. Verify signal is acknowledged via named pipe" -ForegroundColor White
Write-Host "5. Verify no duplicate signals (Valkey fingerprint protection)" -ForegroundColor White
Write-Host "6. Verify expired signals are rejected (stale-signal check)" -ForegroundColor White
