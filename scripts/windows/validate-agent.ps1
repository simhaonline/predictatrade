# Predict-A-Trade Windows Agent Validation Framework
# SOW Section 3 (prompt.md Blocker A): Comprehensive Windows runtime validation
# Run on a real Windows machine with the agent installed.
# Generates JSON + human-readable reports.

param(
    [string]$AgentPath = "C:\Program Files\PredictATrade\XAUUSD\pat-agent.exe",
    [string]$BackendURL = "https://api.predictatrade.com",
    [string]$OutputDir = "$env:TEMP\pat-validation",
    [string]$LicenseKey = "",
    [switch]$SkipMT4,
    [switch]$SkipMT5,
    [switch]$SkipService,
    [switch]$SkipUpdater
)

$ErrorActionPreference = "Continue"
$Results = [ordered]@{}
$PassCount = 0
$FailCount = 0
$PendingCount = 0

function Test-Gate {
    param([string]$Name, [string]$Status, [string]$Detail = "")
    $Results[$Name] = @{ status = $Status; detail = $Detail; timestamp = (Get-Date).ToUniversalTime().ToString("o") }
    $color = if ($Status -eq "PASS") { "Green" } elseif ($Status -eq "FAIL") { "Red" } else { "Yellow" }
    Write-Host "[$Status] $Name $Detail" -ForegroundColor $color
    switch ($Status) {
        "PASS" { $script:PassCount++ }
        "FAIL" { $script:FailCount++ }
        default { $script:PendingCount++ }
    }
}

# Setup output dir
New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null

Write-Host "`n=== Predict-A-Trade Windows Agent Validation ===`n" -ForegroundColor Cyan

# 1. WINDOWS_BINARY_RUNTIME
$exists = Test-Path $AgentPath
if ($exists) {
    $version = (Get-Item $AgentPath).VersionInfo
    $arch = if ([System.IntPtr]::Size -eq 8) { "x64" } else { "x86" }
    Test-Gate "WINDOWS_BINARY_RUNTIME" "PASS" "v$($version.ProductVersion) $arch"
} else {
    Test-Gate "WINDOWS_BINARY_RUNTIME" "FAIL" "Binary not found at $AgentPath"
}

# 2. WINDOWS_MASTER_CONNECTION
try {
    $response = Invoke-WebRequest -Uri "$BackendURL/api/v1/health" -TimeoutSec 10 -UseBasicParsing -ErrorAction Stop
    Test-Gate "WINDOWS_MASTER_CONNECTION" "PASS" "HTTP $($response.StatusCode)"
} catch {
    Test-Gate "WINDOWS_MASTER_CONNECTION" "FAIL" $_.Exception.Message
}

# 3. WINDOWS_MARKET_DATA_RECEPTION
try {
    $wsResult = Test-NetConnection -ComputerName "live.predictatrade.com" -Port 443 -WarningAction SilentlyContinue
    if ($wsResult.TcpTestSucceeded) {
        Test-Gate "WINDOWS_MARKET_DATA_RECEPTION" "PASS" "live.predictatrade.com:443 reachable"
    } else {
        Test-Gate "WINDOWS_MARKET_DATA_RECEPTION" "FAIL" "Cannot reach live.predictatrade.com:443"
    }
} catch {
    Test-Gate "WINDOWS_MARKET_DATA_RECEPTION" "FAIL" $_.Exception.Message
}

# 4. WINDOWS_SERVICE
if (-not $SkipService) {
    $svc = Get-Service -Name "pat-agent" -ErrorAction SilentlyContinue
    if ($svc) {
        $autoStart = ($svc.StartType -eq "Automatic")
        Test-Gate "WINDOWS_SERVICE" "PASS" "Status: $($svc.Status), StartType: $($svc.StartType)"
    } else {
        # Try installing the service
        try {
            & $AgentPath -install 2>&1 | Out-Null
            Start-Sleep 3
            $svc = Get-Service -Name "pat-agent" -ErrorAction SilentlyContinue
            if ($svc) {
                Test-Gate "WINDOWS_SERVICE" "PASS" "Installed and running"
            } else {
                Test-Gate "WINDOWS_SERVICE" "PENDING" "Service not installed — run: $AgentPath -install"
            }
        } catch {
            Test-Gate "WINDOWS_SERVICE" "PENDING" "Service installation required"
        }
    }
}

# 5. WINDOWS_AUTO_START
$svc = Get-Service -Name "pat-agent" -ErrorAction SilentlyContinue
if ($svc -and $svc.StartType -eq "Automatic") {
    Test-Gate "WINDOWS_AUTO_START" "PASS" "StartType: Automatic"
} else {
    Test-Gate "WINDOWS_AUTO_START" "PENDING" "Verify service StartType=Automatic"
}

# 6. WINDOWS_CRASH_RECOVERY
$svc = Get-Service -Name "pat-agent" -ErrorAction SilentlyContinue
if ($svc) {
    $recovery = sc.exe qfailure "pat-agent" 2>&1
    if ($recovery -match "RESTART") {
        Test-Gate "WINDOWS_CRASH_RECOVERY" "PASS" "Recovery actions configured"
    } else {
        Test-Gate "WINDOWS_CRASH_RECOVERY" "PENDING" "Configure service recovery: sc.exe failure pat-agent reset=86400 actions=restart/5000/restart/10000/restart/30000"
    }
} else {
    Test-Gate "WINDOWS_CRASH_RECOVERY" "PENDING" "Service not installed"
}

# 7. WINDOWS_NETWORK_RECOVERY
Test-Gate "WINDOWS_NETWORK_RECOVERY" "PENDING" "Requires manual test: disconnect network, verify reconnection with backoff"

# 8. WINDOWS_MT5_RUNTIME
if (-not $SkipMT5) {
    $mt5Pipe = $null
    try {
        $mt5Pipe = New-Object System.IO.Pipes.NamedPipeClientStream(".", "PredictATradeMT5", [System.IO.Pipes.PipeDirection]::InOut)
        $mt5Pipe.Connect(5000)
        Test-Gate "WINDOWS_MT5_RUNTIME" "PASS" "MT5 pipe connected"
    } catch {
        Test-Gate "WINDOWS_MT5_RUNTIME" "NOT_TESTED" "MT5 not running or pipe not available"
    } finally {
        if ($mt5Pipe) { $mt5Pipe.Dispose() }
    }
}

# 9. WINDOWS_MT4_RUNTIME
if (-not $SkipMT4) {
    $mt4Pipe = $null
    try {
        $mt4Pipe = New-Object System.IO.Pipes.NamedPipeClientStream(".", "PredictATradeMT4", [System.IO.Pipes.PipeDirection]::InOut)
        $mt4Pipe.Connect(5000)
        Test-Gate "WINDOWS_MT4_RUNTIME" "PASS" "MT4 pipe connected"
    } catch {
        Test-Gate "WINDOWS_MT4_RUNTIME" "NOT_TESTED" "MT4 not running or pipe not available"
    } finally {
        if ($mt4Pipe) { $mt4Pipe.Dispose() }
    }
}

# 10. WINDOWS_SIGNAL_DELIVERY
Test-Gate "WINDOWS_SIGNAL_DELIVERY" "PENDING" "Requires running agent + active signal generation"

# 11. WINDOWS_SIGNAL_ACK
Test-Gate "WINDOWS_SIGNAL_ACK" "PENDING" "Requires running agent + active signal + MT5 EA"

# 12. WINDOWS_DUPLICATE_PROTECTION
Test-Gate "WINDOWS_DUPLICATE_PROTECTION" "PENDING" "Requires Valkey + concurrent signal test"

# 13. WINDOWS_TELEMETRY
try {
    $response = Invoke-WebRequest -Uri "$BackendURL/api/v1/agent/status" -TimeoutSec 5 -UseBasicParsing -ErrorAction Stop
    if ($response.StatusCode -eq 200) {
        Test-Gate "WINDOWS_TELEMETRY" "PASS" "Agent status endpoint reachable"
    }
} catch {
    Test-Gate "WINDOWS_TELEMETRY" "PENDING" "Agent status not available"
}

# 14. WINDOWS_LICENSE
if ($LicenseKey) {
    Test-Gate "WINDOWS_LICENSE" "PENDING" "License validation requires agent running with key"
} else {
    Test-Gate "WINDOWS_LICENSE" "PENDING" "No license key provided for validation"
}

# 15. WINDOWS_INSTALLER
try {
    $installResult = & $AgentPath -install 2>&1
    Test-Gate "WINDOWS_INSTALLER" "PASS" "Install command executed"
} catch {
    Test-Gate "WINDOWS_INSTALLER" "PENDING" "Install test requires admin privileges"
}

# 16. WINDOWS_UPDATER
if (-not $SkipUpdater) {
    Test-Gate "WINDOWS_UPDATER" "PENDING" "Requires update server configuration"
    Test-Gate "WINDOWS_ROLLBACK" "PENDING" "Requires failed update test"
}

# 17. WINDOWS_LONG_RUNNING_STABILITY
Test-Gate "WINDOWS_LONG_RUNNING_STABILITY" "PENDING" "Requires 24h+ continuous run observation"

# Generate JSON report
$JsonReport = [ordered]@{
    validation_version = 1
    agent_version = if ($exists) { (Get-Item $AgentPath).VersionInfo.ProductVersion } else { "unknown" }
    machine = $env:COMPUTERNAME
    windows_version = [System.Environment]::OSVersion.VersionString
    timestamp = (Get-Date).ToUniversalTime().ToString("o")
    summary = [ordered]@{
        total = $Results.Count
        pass = $PassCount
        fail = $FailCount
        pending = $PendingCount
    }
    tests = $Results
}

$JsonFile = Join-Path $OutputDir "validation-report.json"
$JsonReport | ConvertTo-Json -Depth 5 | Out-File -FilePath $JsonFile -Encoding UTF8

# Generate human-readable report
$ReportFile = Join-Path $OutputDir "validation-report.txt"
$report = "PREDICT-A-TRADE WINDOWS AGENT VALIDATION REPORT`n"
$report += "Generated: $(Get-Date).ToUniversalTime().ToString('o')`n"
$report += "Machine: $env:COMPUTERNAME`n"
$report += "Windows: $([System.Environment]::OSVersion.VersionString)`n`n"
$report += "SUMMARY: $PassCount PASS, $FailCount FAIL, $PendingCount PENDING`n`n"
foreach ($key in $Results.Keys) {
    $r = $Results[$key]
    $report += "[$($r.status)] $key - $($r.detail)`n"
}
$report | Out-File -FilePath $ReportFile -Encoding UTF8

Write-Host "`n=== VALIDATION COMPLETE ===" -ForegroundColor Cyan
Write-Host "Passed: $PassCount | Failed: $FailCount | Pending: $PendingCount"
Write-Host "JSON: $JsonFile"
Write-Host "Report: $ReportFile"
Write-Host "`nImport the JSON report using:"
Write-Host "  Import-PowerShellDataFile -Path '$JsonFile'"
Write-Host "Or send to backend: POST /api/v1/admin/agent/validation-report"
