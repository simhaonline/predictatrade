<#
.SYNOPSIS
    Predict-A-Trade XAUUSD — Health Check Monitor
.DESCRIPTION
    Runs as a Windows Scheduled Task every 60 seconds (adjustable in settings.json).
    Detects if the agent process has hung (stuck) or crashed, force-restarts it,
    and sends a notification via notify.ps1.

    Detection logic:
    1. Check if the agent process is running. If not → restart + notify.
    2. HTTP GET to health_check_url. If fails or times out → kill + restart + notify.
    3. If all OK → log success and exit.

    Special exit code -999 is passed to notify.ps1 for hang detection.
.NOTES
    All actions are logged to the Windows Application Event Log under
    source "pat-agent".
#>

$ErrorActionPreference = "Stop"

[CmdletBinding()]
param(
    [ValidateSet("client","master")][string]$Mode = "client"
)

if ($Mode -eq "master") {
    $ServiceName = "pat-agent-master"
    $AgentExe    = "pat-master.exe"
} else {
    $ServiceName = "pat-agent-client"
    $AgentExe    = "pat-agent.exe"
}

# ─── Paths ───
$ScriptDir    = Split-Path -Parent $MyInvocation.MyCommand.Path
$SettingsFile = Join-Path $ScriptDir "settings.json"
$NotifyScript = Join-Path $ScriptDir "notify.ps1"
$EventSource  = "pat-agent"
$InstallDir   = "C:\PredictATrade\XAUUSD"

# ─── Helper: Write to Event Log ───
function Write-PATEventLog {
    param([string]$Message, [string]$Level = "Information", [int]$EventId = 300)
    try {
        if (-not [System.Diagnostics.EventLog]::SourceExists($EventSource)) {
            [System.Diagnostics.EventLog]::CreateEventSource($EventSource, "Application")
        }
        $log = New-Object System.Diagnostics.EventLog("Application")
        $log.Source = $EventSource
        $entryType = switch ($Level) {
            "Error"   { [System.Diagnostics.EventLogEntryType]::Error }
            "Warning" { [System.Diagnostics.EventLogEntryType]::Warning }
            default   { [System.Diagnostics.EventLogEntryType]::Information }
        }
        $log.WriteEntry($Message, $entryType, $EventId)
    } catch {
        # Fallback to local log file
        $fallbackLog = Join-Path $ScriptDir "logs\health_fallback.log"
        $dir = Split-Path -Parent $fallbackLog
        if (-not (Test-Path $dir)) { New-Item -ItemType Directory -Path $dir -Force | Out-Null }
        Add-Content -Path $fallbackLog -Value "[$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')] [$Level] $Message"
    }
}

# ─── Safeguard: abort if running too long (max 120 seconds) ───
$jobTimeout = 120  # seconds

# ─── Load settings ───
if (-not (Test-Path $SettingsFile)) {
    Write-PATEventLog -Message "health-check.ps1: settings.json not found — aborting health check" -Level "Error" -EventId 301
    exit 1
}

try {
    $settings = Get-Content $SettingsFile -Raw | ConvertFrom-Json
} catch {
    Write-PATEventLog -Message "health-check.ps1: Failed to parse settings.json: $_" -Level "Error" -EventId 302
    exit 1
}

$healthUrl     = $settings.health_check_url
$timeoutSec    = if ($settings.health_check_timeout_seconds) { [int]$settings.health_check_timeout_seconds } else { 5 }

if ([string]::IsNullOrWhiteSpace($healthUrl)) {
    Write-PATEventLog -Message "health-check.ps1: health_check_url not configured in settings.json — aborting" -Level "Error" -EventId 303
    exit 1
}

# ─── Check 1: Is the agent process running? ───
$agentProcess = Get-Process -Name ($AgentExe -replace '.exe', '') -ErrorAction SilentlyContinue
if (-not $agentProcess) {
    # Also check by service name
    $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if (-not $svc -or $svc.Status -ne "Running") {
        $msg = "health-check.ps1: Agent process not running — attempting restart"
        Write-Host $msg
        Write-PATEventLog -Message $msg -Level "Warning" -EventId 304

        # Restart the service via NSSM with retry logic
        $nssm = Join-Path $InstallDir "nssm.exe"
        $restartSuccess = $false
        $maxRetries = 3

        for ($attempt = 1; $attempt -le $maxRetries; $attempt++) {
            try {
                if (Test-Path $nssm) {
                    & $nssm restart $ServiceName 2>&1 | Out-Null
                } else {
                    Restart-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
                }
                Start-Sleep -Seconds (3 + $attempt * 2)  # 5s, 7s, 9s

                # Verify restart
                $svc2 = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
                if ($svc2 -and $svc2.Status -eq "Running") {
                    $restartSuccess = $true
                    Write-PATEventLog -Message "health-check.ps1: Service restarted successfully on attempt $attempt" -EventId 305
                    break
                }
            } catch {
                Write-PATEventLog -Message "health-check.ps1: Restart attempt $attempt failed: $_" -Level "Warning" -EventId 307
            }
        }

        if ($restartSuccess) {
            # Send notification (includes Windows popup)
            if (Test-Path $NotifyScript) {
                & powershell -ExecutionPolicy Bypass -NoProfile -NonInteractive -WindowStyle Hidden -File $NotifyScript -ExitCode -999
            }
        } else {
            # ALL retries failed — show critical alert
            Write-PATEventLog -Message "health-check.ps1: FAILED to restart after $maxRetries attempts — CRITICAL" -Level "Error" -EventId 306

            # Show Windows popup alert
            try {
                Add-Type -AssemblyName System.Windows.Forms
                $job = Start-Job -ScriptBlock {
                    [System.Windows.Forms.MessageBox]::Show(
                        "Predict-A-Trade Agent has CRASHED and could NOT be restarted after 3 attempts!`n`nHost: $env:COMPUTERNAME`nTime: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')`n`nTo reinstall:`nirm https://downloads.predictatrade.com/windows-agent/install.ps1 | iex",
                        "Predict-A-Trade CRITICAL ALERT",
                        [System.Windows.Forms.MessageBoxButtons]::OK,
                        [System.Windows.Forms.MessageBoxIcon]::Error
                    ) | Out-Null
                }
                # Also try msg.exe for immediate popup
                & msg.exe * /TIME:120 "Predict-A-Trade Agent CRASHED and could NOT restart! Manual intervention required." 2>&1 | Out-Null
            } catch {}

            # Send notification
            if (Test-Path $NotifyScript) {
                & powershell -ExecutionPolicy Bypass -NoProfile -NonInteractive -WindowStyle Hidden -File $NotifyScript -ExitCode -998
            }
        }
        exit 0
    }
}

# ─── Check 2: HTTP health endpoint probe ───
Write-Host "[health-check] Probing $healthUrl (timeout: ${timeoutSec}s)..."

$httpOK = $false
try {
    $resp = Invoke-WebRequest -Uri $healthUrl -Method Get -UseBasicParsing -TimeoutSec $timeoutSec
    if ($resp.StatusCode -eq 200) {
        $httpOK = $true
        Write-Host "[health-check] Health endpoint returned HTTP 200 — agent is healthy."
    } else {
        Write-Host "[health-check] Health endpoint returned HTTP $($resp.StatusCode) — unexpected status."
    }
} catch {
    Write-Host "[health-check] Health endpoint probe failed: $_"
}

if ($httpOK) {
    # Agent is healthy — no action needed
    # (Don't spam the event log on every healthy check — only log if previously unhealthy)
    exit 0
}

# ─── Health check FAILED: agent is hung ───
$msg = "health-check.ps1: Agent process is running but health endpoint failed — force-restarting (HANG DETECTED)"
Write-Host $msg
Write-PATEventLog -Message $msg -Level "Warning" -EventId 308

# Step a: Force-stop the service/process
try {
    $nssm = Join-Path $InstallDir "nssm.exe"
    if (Test-Path $nssm) {
        & $nssm stop $ServiceName 2>&1 | Out-Null
    } else {
        Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
    }
    Start-Sleep -Seconds 2

    # Ensure process is killed
    $proc = Get-Process -Name ($AgentExe -replace '.exe', '') -ErrorAction SilentlyContinue
    if ($proc) {
        $proc | Stop-Process -Force -ErrorAction SilentlyContinue
        Start-Sleep -Seconds 1
    }
} catch {
    Write-PATEventLog -Message "health-check.ps1: Force-stop failed: $_" -Level "Error" -EventId 309
}

# Step b: Restart the service with retry
$hangRestartOK = $false
for ($attempt = 1; $attempt -le 3; $attempt++) {
    try {
        $nssm = Join-Path $InstallDir "nssm.exe"
        if (Test-Path $nssm) {
            & $nssm start $ServiceName 2>&1 | Out-Null
        } else {
            Start-Service -Name $ServiceName -ErrorAction SilentlyContinue
        }
        Start-Sleep -Seconds (3 + $attempt * 2)

        $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
        if ($svc -and $svc.Status -eq "Running") {
            $hangRestartOK = $true
            Write-PATEventLog -Message "health-check.ps1: Service restarted on attempt $attempt after hang" -EventId 310
            break
        }
    } catch {
        Write-PATEventLog -Message "health-check.ps1: Hang restart attempt $attempt failed: $_" -Level "Warning" -EventId 312
    }
}

if (-not $hangRestartOK) {
    Write-PATEventLog -Message "health-check.ps1: FAILED to restart after hang — 3 attempts exhausted" -Level "Error" -EventId 311
    # Show critical popup
    try {
        & msg.exe * /TIME:120 "Predict-A-Trade Agent HUNG and could NOT restart! Manual intervention required." 2>&1 | Out-Null
    } catch {}
}

# Step c: Send notification (special exit code -999 = hang)
if (Test-Path $NotifyScript) {
    try {
        & powershell -ExecutionPolicy Bypass -NoProfile -NonInteractive -WindowStyle Hidden -File $NotifyScript -ExitCode -999
    } catch {
        Write-PATEventLog -Message "health-check.ps1: Failed to call notify.ps1: $_" -Level "Error" -EventId 313
    }
}

# Step d: Already logged to Event Log above
exit 0
