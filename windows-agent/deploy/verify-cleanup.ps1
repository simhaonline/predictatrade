<#
.SYNOPSIS
    Verify that NO Predict-A-Trade Windows Agent (Master or Client) remains.
.DESCRIPTION
    Performs a non-destructive audit for leftover services, processes, directories,
    scheduled tasks, event-log source and MetaTrader IPC files from any prior
    install — including the old single-directory (C:\PredictATrade\XAUUSD) install.
    Run this after uninstall.ps1 to PROVE the agent is completely gone.
    Usage (admin PowerShell):  irm https://downloads.predictatrade.com/windows-agent/verify-cleanup.ps1 | iex
#>

$MasterDir = "C:\PredictATrade\Master"
$ClientDir = "C:\PredictATrade\Client"
$LegacyDir = "C:\PredictATrade\XAUUSD"
$EventSource = "pat-agent"
$TaskName = "PredictATradeHealthCheck"
$ServiceNames = @("pat-agent-client","pat-agent-master","pat-agent","PredictATradeAgent","PredictATradeXAUUSD","agent")
$ProcessNames = @("pat-agent","pat-master")
$IpcFiles = @("PAT_ticks.txt","PAT_signals.txt","PAT_license.txt","PAT_init.txt","PAT_commands.txt","PAT_heartbeat.txt","PAT_status.txt")

Write-Host "=========================================="
Write-Host "  Predict-A-Trade Agent — Remnant Audit"
Write-Host "=========================================="

$remnants = @()

foreach ($svc in $ServiceNames) {
    if (Get-Service -Name $svc -ErrorAction SilentlyContinue) { $remnants += "Service present: $svc" }
}
foreach ($pn in $ProcessNames) {
    if (Get-Process -Name $pn -ErrorAction SilentlyContinue) { $remnants += "Process running: $pn" }
}
foreach ($d in @($MasterDir, $ClientDir, $LegacyDir)) {
    if (Test-Path $d) { $remnants += "Directory remains: $d" }
}
if (Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue) { $remnants += "Scheduled task remains: $TaskName" }
if ([System.Diagnostics.EventLog]::SourceExists($EventSource)) { $remnants += "Event log source remains: $EventSource" }
$ipcCommon = "$env:APPDATA\MetaQuotes\Terminal\Common\Files"
foreach ($f in $IpcFiles) {
    if (Test-Path (Join-Path $ipcCommon $f)) { $remnants += "IPC file remains: $f" }
}
# Defender exclusion leftovers
try {
    $pref = Get-MpPreference -ErrorAction SilentlyContinue
    if ($pref -and $pref.ExclusionPath -contains "C:\PredictATrade") { $remnants += "Defender exclusion remains: C:\PredictATrade" }
} catch {}

Write-Host ""
if ($remnants.Count -eq 0) {
    Write-Host "  PASS: No Predict-A-Trade agent remnants detected."
    Write-Host "  The device is clean — safe to reinstall Master and/or Client."
    Write-Host ""
    exit 0
} else {
    Write-Host "  WARN: The following remnants remain:"
    $remnants | ForEach-Object { Write-Host "   - $_" }
    Write-Host ""
    Write-Host "  Recommended: run uninstall.ps1 (Mode 'all'), then re-run this audit."
    Write-Host ""
    exit 1
}
