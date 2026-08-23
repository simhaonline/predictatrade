# Validate Windows Service installation, auto-start, and crash recovery
param([string]$AgentPath = "C:\Program Files\PredictATrade\XAUUSD\pat-agent.exe")
$ErrorActionPreference = "Continue"

Write-Host "=== Service Validation ===" -ForegroundColor Cyan

# Install
& $AgentPath -install 2>&1
Start-Sleep 3
$svc = Get-Service -Name "pat-agent" -ErrorAction SilentlyContinue
if ($svc) {
    Write-Host "[PASS] Service installed" -ForegroundColor Green
    
    # Check auto-start
    if ($svc.StartType -eq "Automatic") {
        Write-Host "[PASS] Auto-start enabled" -ForegroundColor Green
    } else {
        Write-Host "[FAIL] Auto-start not configured" -ForegroundColor Red
        sc.exe config pat-agent start= auto
    }
    
    # Check recovery
    $recovery = sc.exe qfailure "pat-agent" 2>&1
    if ($recovery -match "RESTART.*5000.*RESTART.*10000.*RESTART.*30000") {
        Write-Host "[PASS] Crash recovery configured" -ForegroundColor Green
    } else {
        Write-Host "[CONFIG] Setting recovery actions..." -ForegroundColor Yellow
        sc.exe failure "pat-agent" reset=86400 actions=restart/5000/restart/10000/restart/30000
    }
    
    # Start/Stop/Restart cycle
    Start-Service -Name "pat-agent" -ErrorAction SilentlyContinue
    Start-Sleep 3
    if ((Get-Service "pat-agent").Status -eq "Running") {
        Write-Host "[PASS] Service starts" -ForegroundColor Green
    }
    
    Restart-Service -Name "pat-agent" -Force -ErrorAction SilentlyContinue
    Start-Sleep 3
    if ((Get-Service "pat-agent").Status -eq "Running") {
        Write-Host "[PASS] Service restarts" -ForegroundColor Green
    }
    
    Stop-Service -Name "pat-agent" -Force -ErrorAction SilentlyContinue
    Start-Sleep 2
    if ((Get-Service "pat-agent").Status -eq "Stopped") {
        Write-Host "[PASS] Service stops" -ForegroundColor Green
    }
    
    Start-Service -Name "pat-agent" -ErrorAction SilentlyContinue
} else {
    Write-Host "[FAIL] Service not installed" -ForegroundColor Red
}
