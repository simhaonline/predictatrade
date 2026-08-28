<#
.SYNOPSIS
    Verify that NO Predict-A-Trade Windows Agent (Master or Client) remains.
.DESCRIPTION
    Performs a non-destructive audit for leftover services, processes, directories,
    scheduled tasks, event-log source and MetaTrader IPC files from any prior
    install — including the old single-directory (C:\PredictATrade\XAUUSD) install.
    Run this after uninstall.ps1 to PROVE the agent is completely gone.

    EXIT CODES (by design — not crashes):
      0  = CLEAN: no agent remnants detected.
      1  = NOT CLEAN: one or more remnants were found (see the report printed
           above and saved to the log file), OR an internal check itself errored.
           The report clearly distinguishes "remnants found" from
           "internal script error".

    Usage (admin PowerShell):
      irm https://downloads.predictatrade.com/windows-agent/verify-cleanup.ps1 | iex
#>

# Never let a single failing cmdlet abort the whole audit.
$ErrorActionPreference = 'SilentlyContinue'
$ProgressPreference = 'SilentlyContinue'

$MasterDir = "C:\PredictATrade\Master"
$ClientDir = "C:\PredictATrade\Client"
$LegacyDir = "C:\PredictATrade\XAUUSD"
$EventSource = "pat-agent"
$TaskName = "PredictATradeHealthCheck"
$ServiceNames = @("pat-agent-client","pat-agent-master","pat-agent","PredictATradeAgent","PredictATradeXAUUSD","agent")
$ProcessNames = @("pat-agent","pat-master")
$IpcFiles = @("PAT_ticks.txt","PAT_signals.txt","PAT_license.txt","PAT_init.txt","PAT_commands.txt","PAT_heartbeat.txt","PAT_status.txt")

$scriptErrors = @()

function Invoke-SafeCheck {
    param([ScriptBlock]$Block)
    try { return (& $Block) } catch {
        $script:scriptErrors += ("Check error: " + $_.Exception.Message)
        return $false
    }
}

$remnants = @()

foreach ($svc in $ServiceNames) {
    if (Invoke-SafeCheck { Get-Service -Name $svc -ErrorAction SilentlyContinue }) {
        $remnants += "Service present: $svc"
    }
}
foreach ($pn in $ProcessNames) {
    if (Invoke-SafeCheck { Get-Process -Name $pn -ErrorAction SilentlyContinue }) {
        $remnants += "Process running: $pn"
    }
}
foreach ($d in @($MasterDir, $ClientDir, $LegacyDir)) {
    if (Test-Path $d) { $remnants += "Directory remains: $d" }
}
if (Invoke-SafeCheck { Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue }) {
    $remnants += "Scheduled task remains: $TaskName"
}
if (Invoke-SafeCheck { [System.Diagnostics.EventLog]::SourceExists($EventSource) }) {
    $remnants += "Event log source remains: $EventSource"
}

$ipcCommon = Join-Path $env:APPDATA "MetaQuotes\Terminal\Common\Files"
foreach ($f in $IpcFiles) {
    if (Test-Path (Join-Path $ipcCommon $f)) { $remnants += "IPC file remains: $f" }
}

if (Invoke-SafeCheck {
        $pref = Get-MpPreference -ErrorAction SilentlyContinue
        ($pref -and ($pref.ExclusionPath -contains "C:\PredictATrade"))
    }) {
    $remnants += "Defender exclusion remains: C:\PredictATrade"
}

$Log = Join-Path $env:TEMP "pat_verify_cleanup.log"
$report = @()
$report += "Predict-A-Trade Agent — Remnant Audit"
$report += ("Run: " + (Get-Date).ToString("yyyy-MM-dd HH:mm:ss"))
$report += ""
$report += "Remnants found: $($remnants.Count)"
if ($remnants.Count -gt 0) { $report += "  - " + ($remnants -join "`n  - ") }
$report += ""
$report += "Internal check errors: $($scriptErrors.Count)"
if ($scriptErrors.Count -gt 0) { $report += "  - " + ($scriptErrors -join "`n  - ") }
$report += ""

if ($remnants.Count -eq 0 -and $scriptErrors.Count -eq 0) {
    $report += "RESULT: CLEAN (exit 0) — no agent remnants detected. Safe to reinstall Master and/or Client."
    $report | Out-File -FilePath $Log -Encoding utf8
    Write-Host ($report -join "`n")
    exit 0
} else {
    $report += "RESULT: NOT CLEAN (exit 1)."
    if ($remnants.Count -gt 0) {
        $report += "  -> Remnants detected. Run uninstall.ps1 -Mode all, then re-run this audit."
    }
    if ($scriptErrors.Count -gt 0) {
        $report += "  -> One or more checks errored (see above); the host may lack permissions for that check."
    }
    $report += ("  Full report saved to: " + $Log)
    $report | Out-File -FilePath $Log -Encoding utf8
    Write-Host ($report -join "`n")
    exit 1
}
