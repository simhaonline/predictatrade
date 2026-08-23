# Predict-A-Trade Windows Agent Validation Script
# SOW Section 25: Windows Verification Package
# Run this script on a real Windows machine/VM to validate the agent.

param(
    [string]$AgentPath = "C:\Program Files\PredictATrade\XAUUSD\pat-agent.exe",
    [string]$BackendURL = "https://api.predictatrade.com",
    [string]$LicenseKey = ""
)

$ErrorActionPreference = "Stop"
$PassCount = 0
$FailCount = 0
$Results = @()

function Test-Step {
    param([string]$Name, [bool]$Passed, [string]$Detail = "")
    $status = if ($Passed) { "PASS" } else { "FAIL" }
    $Results += [PSCustomObject]@{ Test = $Name; Status = $status; Detail = $Detail }
    if ($Passed) { $script:PassCount++ } else { $script:FailCount++ }
    Write-Host "[$status] $Name $Detail" -ForegroundColor $(if ($Passed) { 'Green' } else { 'Red' })
}

Write-Host "=== Predict-A-Trade Windows Agent Validation ===" -ForegroundColor Cyan
Write-Host ""

# Test 1: Binary exists
Test-Step "Binary Exists" (Test-Path $AgentPath) "Path: $AgentPath"

# Test 2: Binary startup (run in console mode for 5 seconds)
Write-Host "`n--- Test: Binary Startup ---" -ForegroundColor Yellow
try {
    $proc = Start-Process -FilePath $AgentPath -ArgumentList "-console" -PassThru -NoNewWindow
    Start-Sleep -Seconds 5
    $started = -not $proc.HasExited
    Test-Step "Binary Startup" $started "PID: $($proc.Id)"
    if (-not $proc.HasExited) { $proc.Kill() }
} catch {
    Test-Step "Binary Startup" $false $_.Exception.Message
}

# Test 3: Service installation
Write-Host "`n--- Test: Service Installation ---" -ForegroundColor Yellow
try {
    & $AgentPath -install
    Start-Sleep -Seconds 2
    $svc = Get-Service -Name "pat-agent" -ErrorAction SilentlyContinue
    Test-Step "Service Installation" ($svc -ne $null) "Service exists"
} catch {
    Test-Step "Service Installation" $false $_.Exception.Message
}

# Test 4: Service start
Write-Host "`n--- Test: Service Start ---" -ForegroundColor Yellow
try {
    Start-Service -Name "pat-agent"
    Start-Sleep -Seconds 3
    $svc = Get-Service -Name "pat-agent"
    Test-Step "Service Start" ($svc.Status -eq 'Running') "Status: $($svc.Status)"
} catch {
    Test-Step "Service Start" $false $_.Exception.Message
}

# Test 5: Backend connectivity
Write-Host "`n--- Test: Backend Connectivity ---" -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "$BackendURL/api/v1/health" -TimeoutSec 10 -UseBasicParsing
    Test-Step "Backend Connectivity" ($response.StatusCode -eq 200) "Status: $($response.StatusCode)"
} catch {
    Test-Step "Backend Connectivity" $false $_.Exception.Message
}

# Test 6: MT5 Pipe connectivity (if MT5 is running)
Write-Host "`n--- Test: MT5 Pipe Connectivity ---" -ForegroundColor Yellow
try {
    $pipe = New-Object System.IO.Pipes.NamedPipeClient(".", "PredictATradeMT5", [System.IO.Pipes.PipeDirection]::InOut)
    $pipe.Connect(5000)
    Test-Step "MT5 Pipe Connectivity" $true "Pipe connected"
    $pipe.Dispose()
} catch {
    Test-Step "MT5 Pipe Connectivity" $false "MT5 not running or pipe not available"
}

# Test 7: Service restart
Write-Host "`n--- Test: Service Restart ---" -ForegroundColor Yellow
try {
    Restart-Service -Name "pat-agent" -Force
    Start-Sleep -Seconds 3
    $svc = Get-Service -Name "pat-agent"
    Test-Step "Service Restart" ($svc.Status -eq 'Running') "Status: $($svc.Status)"
} catch {
    Test-Step "Service Restart" $false $_.Exception.Message
}

# Test 8: Reboot persistence (check service start type)
Write-Host "`n--- Test: Reboot Persistence ---" -ForegroundColor Yellow
try {
    $svc = Get-Service -Name "pat-agent"
    Test-Step "Reboot Persistence" ($svc.StartType -eq 'Automatic') "StartType: $($svc.StartType)"
} catch {
    Test-Step "Reboot Persistence" $false $_.Exception.Message
}

# Test 9: Service stop
Write-Host "`n--- Test: Service Stop ---" -ForegroundColor Yellow
try {
    Stop-Service -Name "pat-agent" -Force
    Start-Sleep -Seconds 2
    $svc = Get-Service -Name "pat-agent"
    Test-Step "Service Stop" ($svc.Status -eq 'Stopped') "Status: $($svc.Status)"
} catch {
    Test-Step "Service Stop" $false $_.Exception.Message
}

# Test 10: Service uninstall
Write-Host "`n--- Test: Service Uninstall ---" -ForegroundColor Yellow
try {
    & $AgentPath -uninstall
    Start-Sleep -Seconds 2
    $svc = Get-Service -Name "pat-agent" -ErrorAction SilentlyContinue
    Test-Step "Service Uninstall" ($svc -eq $null) "Service removed"
} catch {
    Test-Step "Service Uninstall" $false $_.Exception.Message
}

# Summary
Write-Host "`n=== VALIDATION SUMMARY ===" -ForegroundColor Cyan
Write-Host "Passed: $PassCount" -ForegroundColor Green
Write-Host "Failed: $FailCount" -ForegroundColor Red
Write-Host ""

if ($FailCount -eq 0) {
    Write-Host "RESULT: ALL TESTS PASSED — Windows Agent validated" -ForegroundColor Green
} else {
    Write-Host "RESULT: $FailCount TESTS FAILED — see details above" -ForegroundColor Red
}

# Export results
$Results | Format-Table -AutoSize
$Results | Export-Csv -Path "validation_results.csv" -NoTypeInformation
Write-Host "`nResults exported to validation_results.csv"
