<#
.SYNOPSIS
    Check whether any Predict-A-Trade Windows Agent is still on this computer.
.DESCRIPTION
    Non-destructive audit for leftover services, processes, folders, scheduled
    task, event-log source and MetaTrader IPC files from any prior install —
    including the old single-folder (C:\PredictATrade\XAUUSD) install.

    Run this AFTER uninstall.ps1 to confirm the agent is fully gone, or BEFORE a
    reinstall to make sure nothing old will conflict.

    Usage (admin PowerShell):
      irm https://downloads.predictatrade.com/windows-agent/verify-cleanup.ps1 | iex

    The script ends with a plain-language verdict. (Technically it returns
    exit code 0 = clean, 1 = leftovers found — but you only need to read the
    on-screen message.)
#>

$ErrorActionPreference = 'SilentlyContinue'
$ProgressPreference = 'SilentlyContinue'

$MasterDir  = "C:\PredictATrade\Master"
$ClientDir  = "C:\PredictATrade\Client"
$LegacyDir  = "C:\PredictATrade\XAUUSD"
$EventSource = "pat-agent"
$TaskName   = "PredictATradeHealthCheck"
$ServiceNames = @("pat-agent-client","pat-agent-master","pat-agent","PredictATradeAgent","PredictATradeXAUUSD","agent")
$ProcessNames = @("pat-agent","pat-master")
$IpcFiles   = @("PAT_ticks.txt","PAT_signals.txt","PAT_license.txt","PAT_init.txt","PAT_commands.txt","PAT_heartbeat.txt","PAT_status.txt")

# Runs a check; returns an object with .Value (result) and .Error (message or $null).
function Invoke-SafeCheck {
    param([ScriptBlock]$Block)
    $err = $null
    $val = $null
    try { $val = & $Block } catch { $err = $_.Exception.Message }
    return [PSCustomObject]@{ Value = $val; Error = $err }
}

$results = @()
$checkErrors = @()

# --- Run the checks ---
$svcFound = @()
foreach ($svc in $ServiceNames) {
    $r = Invoke-SafeCheck { Get-Service -Name $svc -ErrorAction SilentlyContinue }
    if ($r.Error) { $checkErrors += $r.Error }
    if ($r.Value) { $svcFound += $svc }
}
$results += [PSCustomObject]@{ Label = "Agent Windows services (Master/Client/old)"; Found = ($svcFound.Count -gt 0); Detail = ($svcFound -join ", ") }

$procFound = @()
foreach ($pn in $ProcessNames) {
    $r = Invoke-SafeCheck { Get-Process -Name $pn -ErrorAction SilentlyContinue }
    if ($r.Error) { $checkErrors += $r.Error }
    if ($r.Value) { $procFound += $pn }
}
$results += [PSCustomObject]@{ Label = "Agent processes running"; Found = ($procFound.Count -gt 0); Detail = ($procFound -join ", ") }

$dirFound = @()
foreach ($d in @($MasterDir, $ClientDir, $LegacyDir)) {
    if (Test-Path $d) { $dirFound += $d }
}
$results += [PSCustomObject]@{ Label = "Agent install folders"; Found = ($dirFound.Count -gt 0); Detail = ($dirFound -join ", ") }

$r = Invoke-SafeCheck { Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue }
if ($r.Error) { $checkErrors += $r.Error }
$results += [PSCustomObject]@{ Label = "Scheduled task '$TaskName'"; Found = [bool]$r.Value; Detail = $TaskName }

$r = Invoke-SafeCheck { [System.Diagnostics.EventLog]::SourceExists($EventSource) }
if ($r.Error) { $checkErrors += $r.Error }
$results += [PSCustomObject]@{ Label = "Event-log source '$EventSource'"; Found = [bool]$r.Value; Detail = $EventSource }

$ipcFound = @()
$ipcCommon = Join-Path $env:APPDATA "MetaQuotes\Terminal\Common\Files"
foreach ($f in $IpcFiles) {
    if (Test-Path (Join-Path $ipcCommon $f)) { $ipcFound += $f }
}
$results += [PSCustomObject]@{ Label = "MetaTrader IPC files"; Found = ($ipcFound.Count -gt 0); Detail = ($ipcFound -join ", ") }

$r = Invoke-SafeCheck {
    $pref = Get-MpPreference -ErrorAction SilentlyContinue
    ($pref -and ($pref.ExclusionPath -contains "C:\PredictATrade"))
}
if ($r.Error) { $checkErrors += $r.Error }
$results += [PSCustomObject]@{ Label = "Windows Defender exclusion for C:\PredictATrade"; Found = [bool]$r.Value; Detail = "C:\PredictATrade" }

# --- Build the human-readable report ---
$line = "============================================================"
$Log = Join-Path $env:TEMP "pat_verify_cleanup.log"
$report = @()
$report += $line
$report += "  Predict-A-Trade Agent — Removal Verification"
$report += ("  " + (Get-Date).ToString("yyyy-MM-dd HH:mm:ss"))
$report += $line
$report += ""
$report += "  Checking this computer for any leftover agent pieces..."
$report += ""

foreach ($row in $results) {
    if ($row.Found) {
        $report += ("  [!] LEFT OVER: " + $row.Label)
        $report += ("        -> " + $row.Detail)
    } else {
        $report += ("  [OK] " + $row.Label + " — none found")
    }
}

if ($checkErrors.Count -gt 0) {
    $report += ""
    $report += "  NOTE: Some checks could not run (often just missing admin"
    $report += "        rights). Re-run this script as Administrator."
    $report += "        (" + ($checkErrors -join "; ") + ")"
}

$report += ""
$report += $line

$anyLeft = ($results | Where-Object { $_.Found }).Count -gt 0
if (-not $anyLeft -and $checkErrors.Count -eq 0) {
    $report += "  VERDICT:  Your computer is CLEAN."
    $report += "  No Predict-A-Trade agent remnants were found."
    $report += "  You can safely install or reinstall the Master Node"
    $report += "  and/or the Client Agent."
    $report += $line
} else {
    $report += "  VERDICT:  Leftover agent pieces were found."
    $report += ""
    $report += "  Why this matters:"
    $report += "    An old install that is not fully removed can cause the"
    $report += "    'same-device nssm' conflict when you install both the"
    $report += "    Master Node and the Client Agent."
    $report += ""
    $report += "  What to do:"
    $report += "    1. Run the uninstaller (removes everything):"
    $report += "         irm https://downloads.predictatrade.com/windows-agent/uninstall.ps1 | iex"
    $report += "       (when asked, choose Mode = all)"
    $report += "    2. Run this verification check again."
    $report += "    3. If anything still remains, restart the computer and"
    $report += "       run this check once more (some files/services only"
    $report += "       unlock after a reboot)."
    $report += $line
    $report += ("  Full details saved to: " + $Log)
}

$report | Out-File -FilePath $Log -Encoding utf8

# Print to screen with simple color (Green = clean, Red = leftover)
foreach ($l in $report) {
    if ($l -like "[!]*") { Write-Host $l -ForegroundColor Red }
    elseif ($l -like "[OK]*") { Write-Host $l -ForegroundColor Green }
    elseif ($l -like "  VERDICT:*") { Write-Host $l -ForegroundColor $(if ($anyLeft) { 'Red' } else { 'Green' }) }
    else { Write-Host $l }
}

# Keep the window open if launched by double-click, so the message is read.
if ([Environment]::UserInteractive) {
    Write-Host ""
    Read-Host -Prompt "Press Enter to close"
}

if ($anyLeft -or $checkErrors.Count -gt 0) { exit 1 } else { exit 0 }
