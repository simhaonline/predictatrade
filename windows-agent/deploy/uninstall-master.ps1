<#
.SYNOPSIS
    Predict-A-Trade XAUUSD — MASTER NODE ONLY Uninstaller
.DESCRIPTION
    Removes ONLY the Master Node (data role) — service, processes, health-check
    task (master), install directory C:\PredictATrade\Master, and its logs.

    The Client Agent (pat-agent-client / C:\PredictATrade\Client) is NEVER
    touched — safe to run on a co-located machine.

    Usage (elevated PowerShell):
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

# ─── 3. Remove the master scheduled task (per-role name; also legacy shared name ONLY if master-only machine) ───
Write-Host "[3] Removing scheduled task..."
foreach ($t in @($TaskName, "PredictATradeHealthCheck")) {
    $t = Get-ScheduledTask -TaskName $t -ErrorAction SilentlyContinue
    if ($t) {
        # Only remove the SHARED legacy task if it does not reference the client install
        if ($t.TaskName -eq "PredictATradeHealthCheck") {
            $action = ($t.Actions | ForEach-Object { $_.Execute + " " + $_.Arguments }) -join " "
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
        if (Test-Path $MasterDir) { Write-Host "  WARN: could not fully remove (file in use? reboot will clear it)" }
        else { Write-Host "  OK: removed" }
    }
} else {
    Write-Host "[4] $MasterDir not found — skipping"
}

# ─── 5. Remove master Defender exclusion (leave client's alone) ───
try {
    Remove-MpPreference -ExclusionProcess "pat-master.exe" -ErrorAction SilentlyContinue
    # Keep C:\PredictATrade exclusion if the Client still exists (it covers both).
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