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
    Master uninstall:               irm https://downloads.predictatrade.com/windows-agent/uninstall.ps1 | iex   (run with -Mode master)
    Uninstall BOTH client + master: irm https://downloads.predictatrade.com/windows-agent/uninstall.ps1 | iex   (run with -Mode all)
.PARAMETER Mode
    Which role to uninstall: "client" (default), "master", or "all". NOTE: the
    uninstall always cleans up EVERY Predict-A-Trade agent service/process present
    (both client and master), so the default is sufficient for a full uninstall —
    -Mode only affects messaging, not what gets removed.
.PARAMETER Silent
    When specified, removes everything without any user prompts.
#>

param(
    [ValidateSet("client","master","all")][string]$Mode = "client",
    [switch]$Silent
)

# ─── Configuration ───
$BaseUrl       = "https://downloads.predictatrade.com/windows-agent"
$EventSource   = "pat-agent"
$TaskName      = "PredictATradeHealthCheck"
$NssmExe       = "nssm.exe"

# Per-role directories (must match install.ps1). A Master Node and a Client Agent
# now live in SEPARATE folders so they never share binaries/settings/logs, and a
# reinstall of one role cannot disturb the other.
$MasterDir = "C:\PredictATrade\Master"
$ClientDir = "C:\PredictATrade\Client"
$LegacyDir = "C:\PredictATrade\XAUUSD"   # old single-dir install; always cleaned

# Primary reference dir for shared logic.
$InstallDir = if ($Mode -eq "master") { $MasterDir } elseif ($Mode -eq "client") { $ClientDir } else { $ClientDir }

# Directories physically removed for this run. The legacy single-dir install is
# always removed because it is superseded by the per-role directories.
$DirsToRemove = if ($Mode -eq "master") { @($MasterDir, $LegacyDir) }
                elseif ($Mode -eq "client") { @($ClientDir, $LegacyDir) }
                else { @($MasterDir, $ClientDir, $LegacyDir) }

# Returns an existing nssm.exe from any role dir, the cached common copy, or PATH.
function Get-RoleNssm {
    foreach ($d in @($MasterDir, $ClientDir, $LegacyDir, "C:\ProgramData\PredictATrade")) {
        $p = Join-Path $d $NssmExe
        if (Test-Path $p) { return $p }
    }
    $cmd = Get-Command $NssmExe -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }
    return $null
}

# Role-specific service + binary names (must match install.ps1).
if ($Mode -eq "master") {
    $ServiceName = "pat-agent-master"
    $AgentExe    = "pat-master.exe"
} elseif ($Mode -eq "all") {
    $ServiceName = "pat-agent-client"   # primary; loop below also removes master
    $AgentExe    = "pat-agent.exe"
} else {
    $ServiceName = "pat-agent-client"
    $AgentExe    = "pat-agent.exe"
}

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
        # Build arguments — preserve the role (-Mode) and -Silent flag so the
        # elevated re-run doesn't silently drop them and default to client.
        $procArgs = @("-ExecutionPolicy", "Bypass", "-NoProfile", "-File", "`"$tempScript`"", "-Mode", $Mode)
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

# ─── 0.4 Stop + remove the health-check Scheduled Task FIRST ───
# health-check.ps1 restarts the agent whenever it is not running. If we left it
# in place, it would re-launch the agent during/after uninstall. Remove (and
# disable) it before touching the service or the process so nothing resurrects
# the agent. Done once, up front, and again later (idempotent).
if (-not $Silent) {
    Write-Host "[uninstall] Stopping + removing health-check Scheduled Task (if any)..."
}
try {
    $hTask = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
    if ($hTask) {
        Stop-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
        Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false -ErrorAction SilentlyContinue
        if (-not $Silent) { Write-Host "  OK: Health-check task removed" }
    }
} catch {}

# ─── 0. Remove any stale prior service names of THIS product ───
# Older installs used different service names (agent / PredictATradeAgent /
# PredictATradeXAUUSD). Remove them so uninstall fully cleans up and there is
# no overlap with a future reinstall.
Write-Host "[uninstall] Checking for stale prior service names..."
$PriorServiceNames = @("agent", "PredictATradeAgent", "PredictATradeXAUUSD", "pat-agent")
$nssmPath = Get-RoleNssm
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
        }
    } catch {}
    # Guaranteed removal regardless of how the service was registered.
    Stop-Service -Name $prior -Force -ErrorAction SilentlyContinue
    sc.exe stop $prior 2>&1 | Out-Null
    sc.exe delete $prior 2>&1 | Out-Null
    Start-Sleep -Seconds 1
}

# ─── 1. Stop and delete the Windows service(s) ───
Write-Host "[uninstall] Stopping and removing service(s)..."
$nssmPath = Get-RoleNssm
# Always clean up BOTH roles (client + master). Running the default uninstall
# must not leave a Master Node service running behind.
$ServicesToRemove = @("pat-agent-client", "pat-agent-master")
foreach ($svcName in $ServicesToRemove) {
    $svc = Get-Service -Name $svcName -ErrorAction SilentlyContinue
    if (-not $svc) {
        Write-Host "  OK: Service $svcName not found — skipping"
        continue
    }

    # Stop the service: try nssm stop, then guarantee with native stop/delete.
    try {
        if (Test-Path $nssmPath) { & $nssmPath stop $svcName 2>&1 | Out-Null }
    } catch {}
    Stop-Service -Name $svcName -Force -ErrorAction SilentlyContinue
    sc.exe stop $svcName 2>&1 | Out-Null
    Start-Sleep -Seconds 2

    # Delete the service. ALWAYS use sc.exe delete as the guaranteed path so the
    # service is removed whether it was wrapped by NSSM OR registered as a native
    # Windows service. NSSM remove is attempted first (harmless if not applicable).
    try {
        if (Test-Path $nssmPath) { & $nssmPath remove $svcName confirm 2>&1 | Out-Null }
    } catch {}
    sc.exe delete $svcName 2>&1 | Out-Null
    Start-Sleep -Seconds 1

    # Verify the service is actually gone.
    $still = Get-Service -Name $svcName -ErrorAction SilentlyContinue
    if ($still) {
        Write-Host "  WARN: Service $svcName could not be removed (may need a reboot)"
    } else {
        Write-Host "  OK: Service $svcName removed"
    }
}

# ─── 1.5. Kill any running agent processes (in case service didn't stop) ───
Write-Host "[uninstall] Killing any running agent processes..."
# Kill BOTH role processes regardless of -Mode so the uninstall is thorough.
$procNames = @("pat-agent", "pat-master")
$agentProcs = @()
foreach ($pn in $procNames) {
    $p = Get-Process -Name $pn -ErrorAction SilentlyContinue
    if ($p) { $agentProcs += $p }
}
if ($agentProcs) {
    $agentProcs | Stop-Process -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 2
    Write-Host "  OK: Killed $($agentProcs.Count) pat-agent process(es)"
} else {
    Write-Host "  OK: No running pat-agent processes found"
}

# ─── 1.6. Clean up IPC files in MetaQuotes Common folder ───
# The Master Node EA and execution EAs write to IPC files (PAT_ticks.txt,
# PAT_signals.txt, etc.) in the MetaQuotes Common\Files folder.
# Even after the agent is uninstalled, the EAs keep writing to these files.
# We clean them up so stale data doesn't cause confusion.
Write-Host "[uninstall] Cleaning up IPC files in MetaQuotes Common folder..."
$commonPaths = @(
    "$env:APPDATA\MetaQuotes\Terminal\Common\Files",
    "C:\Users\$env:USERNAME\AppData\Roaming\MetaQuotes\Terminal\Common\Files",
    "C:\Users\Public\Documents\MetaQuotes\Terminal\Common\Files"
)
$ipcFiles = @("PAT_ticks.txt", "PAT_signals.txt", "PAT_license.txt", "PAT_init.txt", "PAT_commands.txt")
$cleanedCount = 0
foreach ($commonPath in $commonPaths) {
    if (Test-Path $commonPath) {
        foreach ($ipcFile in $ipcFiles) {
            $ipcPath = Join-Path $commonPath $ipcFile
            if (Test-Path $ipcPath) {
                try {
                    Remove-Item $ipcPath -Force -ErrorAction SilentlyContinue
                    $cleanedCount++
                } catch {}
            }
        }
    }
}
if ($cleanedCount -gt 0) {
    Write-Host "  OK: Cleaned up $cleanedCount IPC file(s) in MetaQuotes Common folder"
} else {
    Write-Host "  OK: No IPC files found to clean"
}

# ─── 1.7. Remove Windows Defender exclusions (if we added them) ───
try {
    Remove-MpPreference -ExclusionPath "C:\PredictATrade" -ErrorAction SilentlyContinue
    Remove-MpPreference -ExclusionProcess "pat-agent.exe" -ErrorAction SilentlyContinue
    Remove-MpPreference -ExclusionProcess "pat-master.exe" -ErrorAction SilentlyContinue
} catch {}

# ─── 2. Delete the Scheduled Task ───
Write-Host "[uninstall] Removing Scheduled Task '$TaskName'..."
$task = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
if ($task) {
    Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false
    Write-Host "  OK: Scheduled Task removed"
} else {
    Write-Host "  OK: Scheduled Task not found — skipping"
}

# ─── 3. Remove installation directory(ies) ───
$keepLogs = $false
if (-not $Silent) {
    # CLI-based prompt (no GUI popup)
    $response = Read-Host "Do you want to keep log files? (y/n)"
    if ($response -match "^[Yy]") {
        $keepLogs = $true
        Write-Host "  Logs will be preserved under each role's 'logs' folder"
    }
}

foreach ($dir in $DirsToRemove) {
    if ($keepLogs) {
        Write-Host "[uninstall] Removing binaries/scripts in $dir (keeping logs)..."
        if (Test-Path $dir) {
            Get-ChildItem -Path $dir -Exclude "logs" | Remove-Item -Recurse -Force -ErrorAction SilentlyContinue
        }
    } else {
        Write-Host "[uninstall] Removing directory: $dir"
        if (Test-Path $dir) {
            Remove-Item -Path $dir -Recurse -Force -ErrorAction SilentlyContinue
            if (Test-Path $dir) { Write-Host "  WARN: Could not fully remove $dir" } else { Write-Host "  OK: Removed $dir" }
        } else {
            Write-Host "  OK: $dir not found"
        }
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
    Write-Host "  Logs preserved under each role's 'logs' folder"
}
Write-Host "  Service:     Removed"
Write-Host "  Health Task:  Removed"
Write-Host "  Event Log:    Removed"
Write-Host "  Install Dirs: $(($DirsToRemove -join ', '))"
Write-Host "=========================================="
Write-Host ""
Write-Host "  IMPORTANT: The MetaTrader EAs are still running!"
Write-Host "  To fully stop data feed, remove the EAs from your MT4/MT5 terminals:"
Write-Host "    1. Open MetaTrader"
Write-Host "    2. Right-click the chart -> Expert Advisors -> Remove"
Write-Host "    3. Repeat for each terminal (Master Node + execution EAs)"
Write-Host ""
Write-Host "=========================================="

# ─── Verification: confirm no agent remnants remain ───
Write-Host ""
Write-Host "=== Cleanup verification ==="
$remnants = @()
foreach ($svc in @("pat-agent-client","pat-agent-master","pat-agent","PredictATradeAgent","PredictATradeXAUUSD","agent")) {
    if (Get-Service -Name $svc -ErrorAction SilentlyContinue) { $remnants += "Service present: $svc" }
}
foreach ($pn in @("pat-agent","pat-master")) {
    if (Get-Process -Name $pn -ErrorAction SilentlyContinue) { $remnants += "Process running: $pn" }
}
foreach ($d in @($MasterDir, $ClientDir, $LegacyDir)) {
    if (Test-Path $d) { $remnants += "Directory remains: $d" }
}
if (Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue) { $remnants += "Scheduled task remains: $TaskName" }
if ([System.Diagnostics.EventLog]::SourceExists($EventSource)) { $remnants += "Event log source remains: $EventSource" }
$ipcCommon = "$env:APPDATA\MetaQuotes\Terminal\Common\Files"
foreach ($f in @("PAT_ticks.txt","PAT_signals.txt","PAT_license.txt","PAT_init.txt","PAT_commands.txt","PAT_heartbeat.txt","PAT_status.txt")) {
    if (Test-Path (Join-Path $ipcCommon $f)) { $remnants += "IPC file remains: $f" }
}
if ($remnants.Count -eq 0) {
    Write-Host "  PASS: No Predict-A-Trade agent remnants detected."
} else {
    Write-Host "  WARN: The following remnants remain (may require manual cleanup or a reboot):"
    $remnants | ForEach-Object { Write-Host "   - $_" }
}
Write-Host ""
