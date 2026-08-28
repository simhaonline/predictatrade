<#
.SYNOPSIS
    Verify that no Predict-A-Trade Windows Agent (Master Node and/or Client
    Agent) remains on this computer.
.DESCRIPTION
    Non-destructive audit for leftover services, processes, folders, scheduled
    task, event-log source and MetaTrader IPC files from any prior install —
    including the old single-folder (C:\PredictATrade\XAUUSD) install.

    It checks BOTH roles explicitly:
      • Master Node  (data-only)  — service pat-agent-master, process pat-master,
                                     folder C:\PredictATrade\Master
      • Client Agent (execution)  — service pat-agent-client, process pat-agent,
                                     folder C:\PredictATrade\Client
    plus shared/legacy items (old XAUUSD folder, scheduled task, event source,
    Defender exclusion, MetaTrader IPC files).

    Run this AFTER uninstall.ps1 to confirm the agent is fully gone, or BEFORE a
    reinstall to make sure nothing old will conflict.

    Usage (admin PowerShell):
      irm https://downloads.predictatrade.com/windows-agent/verify-cleanup.ps1 | iex
      irm https://downloads.predictatrade.com/windows-agent/verify-cleanup.ps1 | iex   # master
      irm https://downloads.predictatrade.com/windows-agent/verify-cleanup.ps1 | iex   # client

    The script ends with a plain-language verdict. (Technically it returns
    exit code 0 = clean, 1 = leftovers found — but you only need to read the
    on-screen message.)
#>

# Self-elevate to Administrator. Some checks (Windows Defender exclusion,
# Event Log source) require admin rights; without them the check errors out and
# the verdict becomes unreliable. Mirrors the install/uninstall scripts.
function Test-IsAdmin {
    return ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}
if (-not (Test-IsAdmin)) {
    Write-Host "This verification needs Administrator rights. Restarting elevated..."
    $scriptPath = $PSCommandPath
    if (-not $scriptPath -or -not (Test-Path $scriptPath)) {
        $scriptPath = Join-Path $env:TEMP ("pat_verify_" + [guid]::NewGuid().ToString("N") + ".ps1")
        try {
            Invoke-WebRequest -Uri "https://downloads.predictatrade.com/windows-agent/verify-cleanup.ps1" -OutFile $scriptPath -UseBasicParsing -TimeoutSec 30 -ErrorAction Stop
        } catch {
            Write-Host "ERROR: Could not download the script for elevation: $_"
            Write-Host "Please re-run this script as Administrator manually."
            if ([Environment]::UserInteractive) { Read-Host -Prompt "Press Enter to close" }
            exit 1
        }
    }
    $argList = "-ExecutionPolicy Bypass -NoProfile -File `"$scriptPath`""
    if ($args.Count -gt 0) { $argList += " " + ($args -join " ") }
    try {
        Start-Process powershell -ArgumentList $argList -Verb RunAs -Wait -ErrorAction Stop
    } catch {
        Write-Host "ERROR: Could not launch elevated process: $_"
        Write-Host "Please re-run this script as Administrator manually."
        if ([Environment]::UserInteractive) { Read-Host -Prompt "Press Enter to close" }
        exit 1
    }
    exit 0
}

$ErrorActionPreference = 'SilentlyContinue'
$ProgressPreference = 'SilentlyContinue'

# Optional mode: master | client | all (default all). Read from $args so it also
# works when piped through 'iex' (param() does not).
$mode = "all"
if ($args.Count -gt 0) {
    $m = $args[0].ToString().ToLower()
    if (@("master","client","all") -contains $m) { $mode = $m }
}

$MasterDir  = "C:\PredictATrade\Master"
$ClientDir  = "C:\PredictATrade\Client"
$LegacyDir  = "C:\PredictATrade\XAUUSD"
$EventSource = "pat-agent"
$TaskName   = "PredictATradeHealthCheck"
$IpcFiles   = @("PAT_ticks.txt","PAT_signals.txt","PAT_license.txt","PAT_init.txt","PAT_commands.txt","PAT_heartbeat.txt","PAT_status.txt")

# Runs a check; returns an object with .Value (result) and .Error (message or $null).
function Invoke-SafeCheck {
    param([ScriptBlock]$Block)
    $err = $null
    $val = $null
    try { $val = & $Block } catch { $err = $_.Exception.Message }
    return [PSCustomObject]@{ Value = $val; Error = $err }
}

$checks = @()
$checkErrors = @()

function Add-Check($Role, $Label, $Found, $Detail) {
    return [PSCustomObject]@{ Role = $Role; Label = $Label; Found = [bool]$Found; Detail = $Detail }
}

# --- Master Node (data-only) ---
if ($mode -eq "master" -or $mode -eq "all") {
    $r = Invoke-SafeCheck { Get-Service -Name "pat-agent-master" -ErrorAction SilentlyContinue }
    if ($r.Error) { $checkErrors += $r.Error } else { $checks += Add-Check "Master Node" "Windows service 'pat-agent-master'" $r.Value "pat-agent-master" }

    $r = Invoke-SafeCheck { Get-Process -Name "pat-master" -ErrorAction SilentlyContinue }
    if ($r.Error) { $checkErrors += $r.Error } else { $checks += Add-Check "Master Node" "Process 'pat-master' running" $r.Value "pat-master" }

    $found = Test-Path $MasterDir
    $checks += Add-Check "Master Node" "Install folder" $found $MasterDir
}

# --- Client Agent (execution) ---
if ($mode -eq "client" -or $mode -eq "all") {
    $r = Invoke-SafeCheck { Get-Service -Name "pat-agent-client" -ErrorAction SilentlyContinue }
    if ($r.Error) { $checkErrors += $r.Error } else { $checks += Add-Check "Client Agent" "Windows service 'pat-agent-client'" $r.Value "pat-agent-client" }

    $r = Invoke-SafeCheck { Get-Process -Name "pat-agent" -ErrorAction SilentlyContinue }
    if ($r.Error) { $checkErrors += $r.Error } else { $checks += Add-Check "Client Agent" "Process 'pat-agent' running" $r.Value "pat-agent" }

    $found = Test-Path $ClientDir
    $checks += Add-Check "Client Agent" "Install folder" $found $ClientDir
}

# --- Shared / legacy items ---
$r = Invoke-SafeCheck { Test-Path $LegacyDir }
if ($r.Error) { $checkErrors += $r.Error } else { $checks += Add-Check "Shared / Legacy" "Old single-folder install (C:\PredictATrade\XAUUSD)" $r.Value $LegacyDir }

$r = Invoke-SafeCheck { Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue }
if ($r.Error) { $checkErrors += $r.Error } else { $checks += Add-Check "Shared / Legacy" "Scheduled task '$TaskName'" $r.Value $TaskName }

$r = Invoke-SafeCheck { [System.Diagnostics.EventLog]::SourceExists($EventSource) }
if ($r.Error) { $checkErrors += $r.Error } else { $checks += Add-Check "Shared / Legacy" "Event-log source '$EventSource'" $r.Value $EventSource }

$ipcFound = @()
$ipcCommon = Join-Path $env:APPDATA "MetaQuotes\Terminal\Common\Files"
foreach ($f in $IpcFiles) {
    if (Test-Path (Join-Path $ipcCommon $f)) { $ipcFound += $f }
}
$checks += Add-Check "Shared / Legacy" "MetaTrader IPC files" ($ipcFound.Count -gt 0) ($ipcFound -join ", ")

$r = Invoke-SafeCheck { $pref = Get-MpPreference -ErrorAction SilentlyContinue; ($pref -and ($pref.ExclusionPath -contains "C:\PredictATrade")) }
if ($r.Error) { $checkErrors += $r.Error } else { $checks += Add-Check "Shared / Legacy" "Windows Defender exclusion for C:\PredictATrade" $r.Value "C:\PredictATrade" }

# --- Build the human-readable report ---
$line = "============================================================"
$Log = Join-Path $env:TEMP "pat_verify_cleanup.log"
$report = @()
$report += $line
$report += "  Predict-A-Trade Agent — Removal Verification"
$report += ("  Mode: " + $mode.ToUpper() + "    " + (Get-Date).ToString("yyyy-MM-dd HH:mm:ss"))
$report += $line
$report += ""

$roles = @("Master Node","Client Agent","Shared / Legacy")
foreach ($role in $roles) {
    $roleChecks = $checks | Where-Object { $_.Role -eq $role }
    if ($roleChecks.Count -eq 0) { continue }
    $report += "  $role"
    $report += "  " + ("-" * 52)
    foreach ($c in $roleChecks) {
        if ($c.Found) {
            $report += ("    [!] LEFT OVER: " + $c.Label)
            $report += ("         -> " + $c.Detail)
        } else {
            $report += ("    [OK] " + $c.Label + " — none found")
        }
    }
    $report += ""
}

if ($checkErrors.Count -gt 0) {
    $report += "  NOTE: Some checks could not run (often just missing admin"
    $report += "        rights). Re-run this script as Administrator."
    $report += "        (" + ($checkErrors -join "; ") + ")"
    $report += ""
}

$report += $line

$anyLeft = ($checks | Where-Object { $_.Found }).Count -gt 0
if (-not $anyLeft -and $checkErrors.Count -eq 0) {
    $report += "  VERDICT:  Your computer is CLEAN."
    $report += "  No Predict-A-Trade agent remnants were found."
    if ($mode -eq "all") {
        $report += "  You can safely install or reinstall BOTH the Master Node"
        $report += "  and the Client Agent."
    } else {
        $report += ("  The " + $mode + " role is clear.")
    }
    $report += $line
} else {
    $report += "  VERDICT:  Leftover agent pieces were found."
    $report += ""
    $report += "  >> To remove them, copy and run this (removes BOTH Master and Client):"
    $report += ""
    $report += "     powershell -NoProfile -ExecutionPolicy Bypass -Command `"`$f=Join-Path `$env:TEMP 'pat_uninstall.ps1'; irm https://downloads.predictatrade.com/windows-agent/uninstall.ps1 -OutFile `$f; & `$f -Mode all`""
    $report += ""
    $report += "  Why this matters:"
    $report += "    An old install that is not fully removed can cause the"
    $report += "    'same-device nssm' conflict when you install both the"
    $report += "    Master Node and the Client Agent."
    $report += ""
    $report += "  Then:"
    $report += "    2. Run this verification check again (same command you just used)."
    $report += "    3. If anything still remains, restart the computer and run"
    $report += "       this check once more (some files/services only unlock"
    $report += "       after a reboot)."
    $report += $line
    $report += ("  Full details saved to: " + $Log)
}

$report | Out-File -FilePath $Log -Encoding utf8

# Print to screen with simple color (Green = clean, Red = leftover)
foreach ($l in $report) {
    if ($l -like "[!]*") { Write-Host $l -ForegroundColor Red }
    elseif ($l -like "[OK]*") { Write-Host $l -ForegroundColor Green }
    elseif ($l -like "*>>*") { Write-Host $l -ForegroundColor Cyan }
    elseif ($l -like "*powershell -Command*") { Write-Host $l -ForegroundColor Cyan }
    elseif ($l -like "  VERDICT:*") { Write-Host $l -ForegroundColor $(if ($anyLeft) { 'Red' } else { 'Green' }) }
    else { Write-Host $l }
}

# Keep the window open if launched by double-click, so the message is read.
if ([Environment]::UserInteractive) {
    Write-Host ""
    Read-Host -Prompt "Press Enter to close"
}

if ($anyLeft -or $checkErrors.Count -gt 0) { exit 1 } else { exit 0 }
