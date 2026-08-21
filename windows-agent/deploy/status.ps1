<#
.SYNOPSIS
    Predict-A-Trade XAUUSD — Agent Status Checker
.DESCRIPTION
    Shows comprehensive status of the Predict-A-Trade Windows Agent:
    - Windows service status (running/stopped)
    - Agent process (alive/dead, PID, memory)
    - Local health endpoint (HTTP 200 = healthy)
    - Agent version
    - Clock drift (server vs local time)
    - Connection to Go real-time server
    - MT4/MT5 terminal connection
    - Recent log tail
    - Install directory info

    Usage:
      irm https://downloads.predictatrade.com/windows-agent/status.ps1 | iex

    Or if already installed:
      & "C:\Program Files\PredictATrade\XAUUSD\status.ps1"
#>

$ServiceName = "PredictATradeXAUUSD"
$InstallDir  = "C:\Program Files\PredictATrade\XAUUSD"
$HealthUrl   = "http://127.0.0.1:9000/health"
$EventSource = "PredictATradeXAUUSD"

Write-Host ""
Write-Host "═══════════════════════════════════════════════════════════════"
Write-Host "  Predict-A-Trade XAUUSD — Agent Status Report"
Write-Host "  $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss zzz')"
Write-Host "═══════════════════════════════════════════════════════════════"
Write-Host ""

# ─── 1. Windows Service Status ───
Write-Host "[1] Windows Service: $ServiceName"
$svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($svc) {
    $color = if ($svc.Status -eq "Running") { "Green" } else { "Red" }
    Write-Host "    Status:      " -NoNewline
    Write-Host "$($svc.Status)" -ForegroundColor $color
    Write-Host "    Start Type:  $($svc.StartType)"
    Write-Host "    Can Stop:    $($svc.CanStop)"
} else {
    Write-Host "    Status:      " -NoNewline
    Write-Host "NOT INSTALLED" -ForegroundColor Red
    Write-Host "    Run: irm https://downloads.predictatrade.com/windows-agent/install.ps1 | iex"
}
Write-Host ""

# ─── 2. Agent Process ───
Write-Host "[2] Agent Process"
$proc = Get-Process -Name "agent" -ErrorAction SilentlyContinue
if ($proc) {
    $mem = [math]::Round($proc.WorkingSet64 / 1MB, 1)
    $cpu = $proc.CPU
    Write-Host "    Status:      " -NoNewline
    Write-Host "RUNNING" -ForegroundColor Green
    Write-Host "    PID:         $($proc.Id)"
    Write-Host "    Memory:      ${mem} MB"
    Write-Host "    CPU Time:    ${cpu}s"
    Write-Host "    Uptime:      $([math]::Round(((Get-Date) - $proc.StartTime).TotalMinutes, 1)) min"
} else {
    Write-Host "    Status:      " -NoNewline
    Write-Host "NOT RUNNING" -ForegroundColor Red
}
Write-Host ""

# ─── 3. Health Endpoint ───
Write-Host "[3] Health Endpoint: $HealthUrl"
try {
    $resp = Invoke-WebRequest -Uri $HealthUrl -Method Get -UseBasicParsing -TimeoutSec 5
    if ($resp.StatusCode -eq 200) {
        $health = $resp.Content | ConvertFrom-Json
        Write-Host "    Status:      " -NoNewline
        Write-Host "HEALTHY (HTTP 200)" -ForegroundColor Green
        Write-Host "    Agent Ver:   $($health.agent_version)"
        Write-Host "    Uptime:      $($health.uptime_seconds)s"
        Write-Host "    Server Time: $($health.timestamp)"

        # ─── 3a. Clock Drift Check ───
        $serverTime = [DateTime]::Parse($health.timestamp).ToUniversalTime()
        $localTime  = [DateTime]::UtcNow
        $drift = ($serverTime - $localTime).TotalSeconds
        $driftStr = "{0:N1}s" -f $drift
        if ([math]::Abs($drift) -lt 30) {
            Write-Host "    Clock Drift: $driftStr" -ForegroundColor Green
        } elseif ([math]::Abs($drift) -lt 120) {
            Write-Host "    Clock Drift: $driftStr" -ForegroundColor Yellow
            Write-Host "                 ⚠ Minor drift — consider running: w32tm /resync"
        } else {
            Write-Host "    Clock Drift: $driftStr" -ForegroundColor Red
            Write-Host "                 ⚠ CRITICAL drift — run: w32tm /resync"
        }
    } else {
        Write-Host "    Status:      HTTP $($resp.StatusCode)" -ForegroundColor Yellow
    }
} catch {
    Write-Host "    Status:      " -NoNewline
    Write-Host "UNREACHABLE" -ForegroundColor Red
    Write-Host "    The agent process may not be running or health endpoint is down."
}
Write-Host ""

# ─── 4. Connection to Go Server ───
Write-Host "[4] Go Server Connection"
$logsDir = Join-Path $InstallDir "logs"
$stdoutLog = Join-Path $logsDir "stdout.log"
if (Test-Path $stdoutLog) {
    $lastLines = Get-Content $stdoutLog -Tail 30 -ErrorAction SilentlyContinue
    $connected = ($lastLines | Where-Object { $_ -match "Connected to" } | Select-Object -Last 1)
    $error = ($lastLines | Where-Object { $_ -match "ERROR|FATAL|failed" } | Select-Object -Last 1)

    if ($connected) {
        Write-Host "    Status:      " -NoNewline
        Write-Host "CONNECTED" -ForegroundColor Green
        Write-Host "    Detail:      $connected"
    } else {
        Write-Host "    Status:      " -NoNewline
        Write-Host "CHECK LOGS" -ForegroundColor Yellow
    }

    if ($error) {
        Write-Host "    Last Error:  $error" -ForegroundColor Red
    }
} else {
    Write-Host "    Status:      No logs found (may not be installed)"
}
Write-Host ""

# ─── 5. MT4/MT5 Terminal Connection ───
Write-Host "[5] MT4/MT5 Terminal"
if (Test-Path $stdoutLog) {
    $mtLines = Get-Content $stdoutLog -Tail 50 -ErrorAction SilentlyContinue
    $mt5 = ($mtLines | Where-Object { $_ -match "MT5.*connected|MT5.*tick|Master.*tick" } | Select-Object -Last 1)
    $mt4 = ($mtLines | Where-Object { $_ -match "MT4.*connected|MT4.*tick" } | Select-Object -Last 1)
    $master = ($mtLines | Where-Object { $_ -match "Master Node|MARKET_SNAPSHOT|master" } | Select-Object -Last 1)

    if ($mt5) {
        Write-Host "    MT5:         " -NoNewline
        Write-Host "ACTIVE" -ForegroundColor Green
        Write-Host "    Last:        $mt5"
    } elseif ($mt4) {
        Write-Host "    MT4:         " -NoNewline
        Write-Host "ACTIVE" -ForegroundColor Green
        Write-Host "    Last:        $mt4"
    } elseif ($master) {
        Write-Host "    Master Node: " -NoNewline
        Write-Host "ACTIVE" -ForegroundColor Green
        Write-Host "    Last:        $master"
    } else {
        Write-Host "    Status:      " -NoNewline
        Write-Host "NO MT4/MT5 DATA" -ForegroundColor Yellow
        Write-Host "    Ensure MT4/MT5 terminal is open with the PredictATrade EA attached."
    }
} else {
    Write-Host "    Status:      No logs found"
}
Write-Host ""

# ─── 6. Version Info ───
Write-Host "[6] Version Info"
$versionFile = Join-Path $InstallDir "version.txt"
if (Test-Path $versionFile) {
    $localVer = (Get-Content $versionFile -Raw).Trim()
    Write-Host "    Installed:   v$localVer"
    try {
        $serverVer = (Invoke-WebRequest -Uri "https://downloads.predictatrade.com/windows-agent/version.txt" -UseBasicParsing -TimeoutSec 10).Content.Trim()
        Write-Host "    Available:   v$serverVer"
        if ($localVer -eq $serverVer) {
            Write-Host "    Update:      " -NoNewline
            Write-Host "UP TO DATE" -ForegroundColor Green
        } else {
            Write-Host "    Update:      " -NoNewline
            Write-Host "UPDATE AVAILABLE ($localVer → $serverVer)" -ForegroundColor Yellow
            Write-Host "    Run:         irm https://downloads.predictatrade.com/windows-agent/install.ps1 | iex"
        }
    } catch {
        Write-Host "    Available:   (could not check server)"
    }
} else {
    Write-Host "    Installed:   (not found)"
}
Write-Host ""

# ─── 7. Recent Log Tail ───
Write-Host "[7] Recent Log Output (last 10 lines)"
if (Test-Path $stdoutLog) {
    $tail = Get-Content $stdoutLog -Tail 10 -ErrorAction SilentlyContinue
    foreach ($line in $tail) {
        Write-Host "    $line"
    }
} else {
    Write-Host "    No log file found at $stdoutLog"
}
Write-Host ""

# ─── 8. Scheduled Task ───
Write-Host "[8] Health Check Task"
$task = Get-ScheduledTask -TaskName "PredictATradeHealthCheck" -ErrorAction SilentlyContinue
if ($task) {
    $taskInfo = Get-ScheduledTaskInfo -TaskName "PredictATradeHealthCheck"
    Write-Host "    Status:      " -NoNewline
    Write-Host "CONFIGURED" -ForegroundColor Green
    Write-Host "    Last Run:    $($taskInfo.LastRunTime)"
    Write-Host "    Next Run:    $($taskInfo.NextRunTime)"
    Write-Host "    Last Result: $($taskInfo.LastTaskResult)"
} else {
    Write-Host "    Status:      " -NoNewline
    Write-Host "NOT CONFIGURED" -ForegroundColor Yellow
}
Write-Host ""

# ─── Summary ───
Write-Host "═══════════════════════════════════════════════════════════════"
$allOK = ($svc -and $svc.Status -eq "Running" -and $proc -and $health)
if ($allOK) {
    Write-Host "  ✅ Agent is HEALTHY and running normally." -ForegroundColor Green
} else {
    Write-Host "  ⚠ Issues detected — check the details above." -ForegroundColor Yellow
    Write-Host "  To reinstall: irm https://downloads.predictatrade.com/windows-agent/install.ps1 | iex"
}
Write-Host "  To check status: irm https://downloads.predictatrade.com/windows-agent/status.ps1 | iex"
Write-Host "═══════════════════════════════════════════════════════════════"
Write-Host ""
