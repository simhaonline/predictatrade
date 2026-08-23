<#
.SYNOPSIS
    Predict-A-Trade XAUUSD — Windows Agent Uninstaller
.DESCRIPTION
    Removes the Predict-A-Trade XAUUSD Windows service, scheduled task,
    event log source, and optionally the installation directory.

    Self-elevates to Administrator (UAC prompt appears — user clicks Yes).
    Re-downloads itself from the remote URL inside the elevated session
    to prevent tampering.

    Client uninstall (interactive): irm https://downloads.predictatrade.com/windows-agent/uninstall.ps1 | iex
    Client uninstall (silent):      irm "https://downloads.predictatrade.com/windows-agent/uninstall.ps1?Silent=true" | iex
.PARAMETER Silent
    When specified, removes everything without any user prompts.
#>

param(
    [switch]$Silent
)

# ─── Configuration ───
$BaseUrl       = "https://downloads.predictatrade.com/windows-agent"
$InstallDir    = "C:\Program Files\PredictATrade\XAUUSD"
$ServiceName   = "pat-agent"
$EventSource   = "pat-agent"
$TaskName      = "PredictATradeHealthCheck"
$NssmExe       = "nssm.exe"

# ─── Helper: Write to Event Log ───
function Write-PATEventLog {
    param([string]$Message, [string]$Level = "Information", [int]$EventId = 400)
    try {
        if ([System.Diagnostics.EventLog]::SourceExists($EventSource)) {
            $log = New-Object System.Diagnostics.EventLog("Application")
            $log.Source = $EventSource
            $entryType = switch ($Level) {
                "Error"   { [System.Diagnostics.EventLogEntryType]::Error }
                "Warning" { [System.Diagnostics.EventLogEntryType]::Warning }
                default   { [System.Diagnostics.EventLogEntryType]::Information }
            }
            $log.WriteEntry($Message, $entryType, $EventId)
        }
    } catch {
        Write-Host "[EventLog fallback] $Message"
    }
}

# ─── Detect -Silent from URL query string (for irm | iex pattern) ───
if (-not $Silent) {
    $cmdLine = $MyInvocation.Line
    if ($cmdLine -match "Silent\s*=\s*\$?(true|1)") {
        $Silent = $true
    }
}

# ─── Self-elevation: re-download and re-execute as admin ───
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

if (-not $isAdmin) {
    Write-Host "[uninstall] Administrator rights required."
    Write-Host "[uninstall] A UAC prompt will appear — please click Yes to accept elevation."

    # Re-download the latest uninstall.ps1 from the remote server (tamper-proof)
    $remoteScript = "$BaseUrl/uninstall.ps1"
    try {
        $scriptContent = Invoke-WebRequest -Uri $remoteScript -UseBasicParsing -TimeoutSec 30 | Select-Object -ExpandProperty Content
    } catch {
        Write-Host "[uninstall] FATAL: Failed to download uninstall.ps1 from $remoteScript"
        Write-Host "[uninstall] Error: $_"
        Write-Host "[uninstall] Please open PowerShell as Administrator and run the command again."
        exit 1
    }

    $tempScript = Join-Path $env:TEMP "pat_uninstall_$(Get-Random).ps1"
    Set-Content -Path $tempScript -Value $scriptContent -Encoding UTF8

    try {
        # Build arguments — preserve -Silent flag
        $procArgs = @("-ExecutionPolicy", "Bypass", "-NoProfile", "-File", "`"$tempScript`"")
        if ($Silent) { $procArgs += "-Silent" }

        # Elevate: UAC prompt appears, user clicks Yes, elevated PowerShell runs the script
        $process = Start-Process -FilePath "powershell.exe" `
            -ArgumentList $procArgs `
            -Verb RunAs `
            -Wait `
            -PassThru
        $exitCode = $process.ExitCode
    } catch {
        Write-Host "[uninstall] Elevation was declined or failed: $_"
        Write-Host "[uninstall] Please open PowerShell as Administrator and run:"
        Write-Host "  irm $remoteScript | iex"
        exit 1
    } finally {
        Remove-Item $tempScript -Force -ErrorAction SilentlyContinue
    }
    exit $exitCode
}

# ─═══════════════════════════════════════════════════════════
# Now running as Administrator — proceed with uninstall
# ─═══════════════════════════════════════════════════════════

Write-Host ""
Write-Host "=========================================="
Write-Host "  Predict-A-Trade XAUUSD — Uninstaller"
Write-Host "=========================================="
Write-Host ""

if ($Silent) {
    Write-Host "[uninstall] Silent mode — removing everything without prompts"
}

Write-PATEventLog -Message "Predict-A-Trade XAUUSD uninstall started$(if ($Silent) { ' (silent mode)' })" -EventId 401

# ─── 0. Remove any stale prior service names of THIS product ───
# Older installs used different service names (agent / PredictATradeAgent /
# PredictATradeXAUUSD). Remove them so uninstall fully cleans up and there is
# no overlap with a future reinstall.
Write-Host "[uninstall] Checking for stale prior service names..."
$PriorServiceNames = @("agent", "PredictATradeAgent", "PredictATradeXAUUSD")
$nssmPath = Join-Path $InstallDir $NssmExe
foreach ($prior in $PriorServiceNames) {
    if ($prior -eq $ServiceName) { continue }
    $pSvc = Get-Service -Name $prior -ErrorAction SilentlyContinue
    if (-not $pSvc) { continue }
    # Safety: only remove services that belong to THIS product.
    $pPath = (Get-CimInstance Win32_Service -Filter "Name='$prior'" -ErrorAction SilentlyContinue).PathName
    if ($pPath -and $pPath -notmatch [regex]::Escape($InstallDir) -and $pPath -notmatch 'PredictATrade' -and $pPath -notmatch 'pat-agent') {
        Write-Host "  Skipping '$prior' — not a Predict-A-Trade agent service."
        continue
    }
    Write-Host "  Removing stale prior service '$prior'..."
    try {
        if (Test-Path $nssmPath) {
            & $nssmPath stop $prior 2>&1 | Out-Null
            & $nssmPath remove $prior confirm 2>&1 | Out-Null
        } else {
            Stop-Service -Name $prior -Force -ErrorAction SilentlyContinue
            sc.exe delete $prior 2>&1 | Out-Null
        }
        Start-Sleep -Seconds 1
    } catch {
        sc.exe delete $prior 2>&1 | Out-Null
    }
}

# ─── 1. Stop and delete the Windows service ───
Write-Host "[uninstall] Stopping and removing service..."
$nssmPath = Join-Path $InstallDir $NssmExe
$svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue

if ($svc) {
    # Stop the service
    try {
        if ($svc.Status -eq "Running") {
            if (Test-Path $nssmPath) {
                & $nssmPath stop $ServiceName 2>&1 | Out-Null
            } else {
                Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
            }
            Start-Sleep -Seconds 2
        }
        Write-Host "  OK: Service stopped"
    } catch {
        Write-Host "  WARN: Could not stop service: $_"
    }

    # Delete the service
    try {
        if (Test-Path $nssmPath) {
            & $nssmPath remove $ServiceName confirm 2>&1 | Out-Null
        } else {
            sc.exe delete $ServiceName 2>&1 | Out-Null
        }
        Start-Sleep -Seconds 1
        Write-Host "  OK: Service removed"
    } catch {
        # Fallback to sc.exe
        sc.exe delete $ServiceName 2>&1 | Out-Null
        Write-Host "  OK: Service removed (via sc.exe fallback)"
    }
} else {
    Write-Host "  OK: Service not found — skipping"
}

# ─── 2. Delete the Scheduled Task ───
Write-Host "[uninstall] Removing Scheduled Task '$TaskName'..."
$task = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
if ($task) {
    Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false
    Write-Host "  OK: Scheduled Task removed"
} else {
    Write-Host "  OK: Scheduled Task not found — skipping"
}

# ─── 3. Remove installation directory ───
$keepLogs = $false
if (-not $Silent) {
    # CLI-based prompt (no GUI popup)
    $response = Read-Host "Do you want to keep log files? (y/n)"
    if ($response -match "^[Yy]") {
        $keepLogs = $true
        Write-Host "  Logs will be preserved at $(Join-Path $InstallDir 'logs')"
    }
}

if ($keepLogs) {
    # Remove everything except the logs directory
    Write-Host "[uninstall] Removing binaries and scripts (keeping logs)..."
    Get-ChildItem -Path $InstallDir -Exclude "logs" | Remove-Item -Recurse -Force -ErrorAction SilentlyContinue
} else {
    # Remove everything
    Write-Host "[uninstall] Removing entire installation directory: $InstallDir"
    if (Test-Path $InstallDir) {
        Remove-Item -Path $InstallDir -Recurse -Force -ErrorAction SilentlyContinue
        Write-Host "  OK: Installation directory removed"
    } else {
        Write-Host "  OK: Installation directory not found"
    }
}

# ─── 4. Remove Event Log source ───
Write-Host "[uninstall] Removing Event Log source '$EventSource'..."
try {
    if ([System.Diagnostics.EventLog]::SourceExists($EventSource)) {
        $regPath = "HKLM:\SYSTEM\CurrentControlSet\Services\Eventlog\Application\$EventSource"
        if (Test-Path $regPath) {
            Remove-Item -Path $regPath -Recurse -Force -ErrorAction SilentlyContinue
        }
        Write-Host "  OK: Event Log source removed"
    } else {
        Write-Host "  OK: Event Log source not found — skipping"
    }
} catch {
    Write-Host "  WARN: Could not remove Event Log source: $_"
    try {
        Remove-EventLog -Source $EventSource -ErrorAction SilentlyContinue
        Write-Host "  OK: Event Log source removed (via Remove-EventLog)"
    } catch {
        Write-Host "  WARN: Fallback also failed: $_"
    }
}

# ─── Summary ───
Write-Host ""
Write-Host "=========================================="
Write-Host "  Uninstall Complete!"
Write-Host "=========================================="
if ($keepLogs) {
    Write-Host "  Logs preserved at: $(Join-Path $InstallDir 'logs')"
}
Write-Host "  Service:     Removed"
Write-Host "  Health Task:  Removed"
Write-Host "  Event Log:    Removed"
Write-Host "  Install Dir:  $(if ($keepLogs) { 'Logs kept' } else { 'Removed' })"
Write-Host "=========================================="
Write-Host ""
