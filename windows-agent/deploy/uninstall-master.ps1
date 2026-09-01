<#
.SYNOPSIS
    Predict-A-Trade XAUUSD — MASTER NODE ONLY Uninstaller
.DESCRIPTION
    Removes ONLY the Master Node (data role) — service, processes, health-check
    task (master), install directory C:\PredictATrade\Master, and its logs.

    The Client Agent (pat-agent-client / C:\PredictATrade\Client) is NEVER
    touched — safe to run on a co-located machine.

    Self-elevates: if not run as Administrator, a UAC prompt appears — click Yes.

    Usage (any PowerShell):
      irm https://downloads.predictatrade.com/windows-agent/uninstall-master.ps1 | iex
    Silent (no prompts):
      irm "https://downloads.predictatrade.com/windows-agent/uninstall-master.ps1?Silent=true" | iex
#>
param([switch]$Silent)

$ErrorActionPreference = "Continue"

$MasterDir    = "C:\PredictATrade\Master"
$ServiceName  = "pat-agent-master"
$ExeName      = "pat-master"
$TaskName     = "PredictATradeHealthCheckMaster"
$BaseUrl      = "https://downloads.predictatrade.com/windows-agent"

# Detect ?Silent=true from the irm command line (irm passes the URI via $MyInvocation)
if (-not $Silent) {
    $cmdLine = $MyInvocation.Line
    if ($cmdLine -match "Silent\s*=\s*\$?(true|1)") { $Silent = $true }
}

# ─── Self-elevation: re-download and re-execute as admin ───
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

if (-not $isAdmin) {
    Write-Host ""
    Write-Host "[uninstall-master] Administrator rights required."
    Write-Host "[uninstall-master] A UAC prompt will appear - please click Yes."
    Write-Host ""

    $remoteScript = "$BaseUrl/uninstall-master.ps1"
    try {
        $scriptContent = Invoke-WebRequest -Uri $remoteScript -UseBasicParsing -TimeoutSec 30 | Select-Object -ExpandProperty Content
    } catch {
        Write-Host "[uninstall-master] FATAL: failed to download $remoteScript : $_"
        exit 1
    }

    $tempScript = Join-Path $env:TEMP "pat_uninstall_master_$(Get-Random).ps1"
    Set-Content -Path $tempScript -Value $scriptContent -Encoding UTF8

    try {
        $procArgs = @("-ExecutionPolicy", "Bypass", "-NoProfile", "-File", "`"$tempScript`"")
        if ($Silent) { $procArgs += "-Silent" }
        $process = Start-Process -FilePath "powershell.exe" `
            -ArgumentList $procArgs `
            -Verb RunAs `
            -Wait `
            -PassThru
        # NOTE: the elevated window closes when done — the UAC'd run does the real work.
        exit $process.ExitCode
    } catch {
        Write-Host "[uninstall-master] Elevation declined or failed: $_"
        Write-Host "[uninstall-master] Open PowerShell as Administrator and re-run the command."
        exit 1
    } finally {
        Remove-Item $tempScript -Force -ErrorAction SilentlyContinue
    }
}

# ════════════════════════════════════════════════════════════
# Now running as Administrator — proceed with uninstall
# ════════════════════════════════════════════════════════════
Write-Host ""
Write-Host "=========================================="
Write-Host "  Predict-A-Trade — MASTER NODE Uninstall"
Write-Host "  (Client Agent is NOT touched)"
Write-Host "=========================================="
Write-Host ""

# ─── 0. Find an nssm.exe if one exists (any role dir) ───
$nssmPath = $null
foreach ($d in @($MasterDir, "C:\PredictATrade\Client", "C:\ProgramData\PredictATrade")) {
    $p = Join-Path $d "nssm.exe"
    if (Test-Path $p) { $nssmPath = $p; break }
}

# ─── 1. Stop + delete the master service ───
Write-Host "[1] Stopping service $ServiceName..."
$svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($svc) {
    try { if ($nssmPath) { & $nssmPath stop $ServiceName 2>&1 | Out-Null } } catch {}
    Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
    sc.exe stop $ServiceName 2>&1 | Out-Null
    Start-Sleep -Seconds 2
    try { if ($nssmPath) { & $nssmPath remove $ServiceName confirm 2>&1 | Out-Null } } catch {}
    sc.exe delete $ServiceName 2>&1 | Out-Null
    Start-Sleep -Seconds 1
    if (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue) {
        Write-Host "  WARN: service still present (may need reboot)"
    } else {
        Write-Host "  OK: service removed"
    }
} else {
    Write-Host "  OK: service not found (already removed)"
}

# ─── 2. Kill master processes only ───
Write-Host "[2] Killing $ExeName processes..."
$p = Get-Process -Name $ExeName -ErrorAction SilentlyContinue
if ($p) {
    $p | Stop-Process -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 1
    Write-Host "  OK: killed $($p.Count) process(es)"
} else {
    Write-Host "  OK: no $ExeName processes running"
}

# ─── 3. Remove the master scheduled task ───
Write-Host "[3] Removing scheduled task..."
foreach ($tName in @($TaskName, "PredictATradeHealthCheck")) {
    $t = Get-ScheduledTask -TaskName $tName -ErrorAction SilentlyContinue
    if ($t) {
        # Shared legacy task: only remove when it does NOT reference the Client install
        if ($t.TaskName -eq "PredictATradeHealthCheck") {
            $action = ($t.Actions | ForEach-Object { "$($_.Execute) $($_.Arguments)" }) -join " "
            if ($action -match "Client") {
                Write-Host "  SKIP: shared task belongs to Client — not removed"
                continue
            }
        }
        Unregister-ScheduledTask -TaskName $t.TaskName -Confirm:$false -ErrorAction SilentlyContinue
        Write-Host "  OK: task '$($t.TaskName)' removed"
    }
}

# ─── 4. Remove the master install directory (ask about logs) ───
$keepLogs = $false
if (-not $Silent) {
    $response = Read-Host "Keep Master log files (C:\PredictATrade\Master\logs)? (y/n)"
    if ($response -match "^[Yy]") { $keepLogs = $true }
}
if (Test-Path $MasterDir) {
    if ($keepLogs) {
        Write-Host "[4] Clearing $MasterDir (keeping logs)..."
        Get-ChildItem -Path $MasterDir -Exclude "logs" | Remove-Item -Recurse -Force -ErrorAction SilentlyContinue
        Write-Host "  OK: binaries removed; logs preserved in $MasterDir\logs"
    } else {
        Write-Host "[4] Removing $MasterDir..."
        Remove-Item -Path $MasterDir -Recurse -Force -ErrorAction SilentlyContinue
        if (Test-Path $MasterDir) { Write-Host "  WARN: could not fully remove (file in use? reboot clears it)" }
        else { Write-Host "  OK: removed" }
    }
} else {
    Write-Host "[4] $MasterDir not found — skipping"
}

# ─── 5. Remove master Defender exclusion (leave client's alone) ───
try {
    Remove-MpPreference -ExclusionProcess "pat-master.exe" -ErrorAction SilentlyContinue
    if (-not (Test-Path "C:\PredictATrade\Client")) {
        Remove-MpPreference -ExclusionPath "C:\PredictATrade" -ErrorAction SilentlyContinue
    }
} catch {}

# ─── 6. Clean master IPC files in MetaQuotes Common folder ───
$ipcCommon = "$env:APPDATA\MetaQuotes\Terminal\Common\Files"
$ipcFiles  = @("PAT_ticks.txt","PAT_status.txt","PAT_heartbeat.txt","PAT_init.txt","PAT_resync.txt")
$cleaned = 0
foreach ($f in $ipcFiles) {
    $fp = Join-Path $ipcCommon $f
    if (Test-Path $fp) { Remove-Item $fp -Force -ErrorAction SilentlyContinue; $cleaned++ }
}
Write-Host "[5] IPC files cleaned: $cleaned"

# ─── Verification ───
Write-Host ""
Write-Host "=== Master uninstall verification ==="
$remnants = @()
if (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue) { $remnants += "service still present" }
if (Get-Process -Name $ExeName -ErrorAction SilentlyContinue) { $remnants += "process still running" }
if (Test-Path $MasterDir) { $remnants += "dir still present: $MasterDir" }
if ($remnants.Count -eq 0) {
    Write-Host "  PASS: Master Node fully removed."
} else {
    $remnants | ForEach-Object { Write-Host "  WARN: $_" }
}
Write-Host ""
Write-Host "  Client Agent (pat-agent-client) was NOT touched."
Write-Host ""
Write-Host "  Reinstall the Master Node with:"
Write-Host "    irm https://downloads.predictatrade.com/windows-agent/master/install.ps1 | iex"
Write-Host "  (Watch for the SmartScreen prompt and click 'Run anyway'.)"
Write-Host ""
if (-not $Silent) {
    Write-Host ""
    $null = Read-Host "Press Enter to close this window"
}