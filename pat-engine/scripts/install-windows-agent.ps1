# Install-PredictATradeAgent.ps1
# Deploys the PAT Windows Agent + MQL EAs (PredictATrade client AND MasterNode) and the
# license file on a Windows trading machine. Run from an ADMINISTRATOR PowerShell.
#
# It does NOT compile anything (cross-compile on Linux/macOS with
# scripts/build-windows-agent.sh, then copy pat-windows-agent.exe next to this script).
# It DOES:
#   1. Copy pat-windows-agent.exe to $InstallPath
#   2. Deploy PredictATrade_MT4/MT5.mq4/.mq5 and the MasterNode variants into every
#      detected MT4/MT5 terminal's Experts folder
#   3. Write PAT_license.txt (status ACTIVE) into the MT common Files folder
#   4. Register the agent to start at boot via Task Scheduler (dependency-free)
#
# Usage:
#   .\Install-PredictATradeAgent.ps1 -GatewayUrl "http://<engine-host>:8080/bar" `
#                                    -LicenseKey "PAT1-CIHT-OBIB-J5VF-SQP4-FQ"

[CmdletBinding()]
param(
    [string]$GatewayUrl  = "http://localhost:8080/bar",
    [string]$LicenseKey  = "",
    [string]$InstallPath = "C:\Program Files\PredictATrade",
    [switch]$NoService
)

$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

function Find-Terminals {
    $mtRoot = Join-Path $env:APPDATA "MetaQuotes\Terminal"
    $found  = @()
    if (-not (Test-Path $mtRoot)) { return $found }
    foreach ($term in (Get-ChildItem $mtRoot -Directory)) {
        foreach ($ver in @(4, 5)) {
            $mql = Join-Path $term.FullName "MQL$ver"
            if (Test-Path $mql) {
                $found += [PSCustomObject]@{
                    Version = $ver
                    Experts = Join-Path $mql "Experts"
                    Files   = Join-Path $mql "Files"
                }
            }
        }
    }
    # FILE_COMMON (shared across terminals) lives here:
    $common = Join-Path $mtRoot "Common\Files"
    if (Test-Path (Split-Path $common)) { $found += [PSCustomObject]@{Version = 0; Experts = $null; Files = $common } }
    return $found
}

# --- 1. Agent binary ---------------------------------------------------------
$agentExe = Join-Path $InstallPath "pat-windows-agent.exe"
if (-not (Test-Path $agentExe)) {
    # fall back to script dir if not yet copied
    $local = Join-Path $ScriptDir "pat-windows-agent.exe"
    if (Test-Path $local) {
        New-Item -ItemType Directory -Force -Path $InstallPath | Out-Null
        Copy-Item $local $agentExe -Force
        Write-Host "Copied agent -> $agentExe"
    } else {
        Write-Error "pat-windows-agent.exe not found in $InstallPath or $ScriptDir. Cross-compile it first (scripts/build-windows-agent.sh)."
    }
}

# wrapper that injects the GATEWAY env the agent reads
$runCmd = Join-Path $InstallPath "run-agent.cmd"
Set-Content -Path $runCmd -Value "@echo off`r`nSET GATEWAY=$GatewayUrl`r`n""$agentExe""""

# --- 2. MQL EAs (client + MasterNode) ---------------------------------------
$terms = Find-Terminals
if ($terms.Count -eq 0) { Write-Warning "No MetaTrader terminal found under $env:APPDATA\MetaQuotes\Terminal. Skipping EA deploy." }
$eas = @(
    @{Src="PredictATrade_MT4.mq4";  Dest="MQL4\Experts"; Match=4},
    @{Src="PredictATrade_MT5.mq5";  Dest="MQL5\Experts"; Match=5},
    @{Src="PredictATrade_MasterNode_MT4.mq4"; Dest="MQL4\Experts"; Match=4},
    @{Src="PredictATrade_MasterNode_MT5.mq5"; Dest="MQL5\Experts"; Match=5}
)
foreach ($t in $terms) {
    if ($null -eq $t.Experts) { continue }
    foreach ($ea in $eas) {
        if ($ea.Match -ne 0 -and $ea.Match -ne $t.Version) { continue }
        $src = Join-Path $ScriptDir $ea.Src
        if (-not (Test-Path $src)) { Write-Warning "EA source missing: $src"; continue }
        $dst = Join-Path $t.Experts (Split-Path $ea.Src -Leaf)
        Copy-Item $src $dst -Force
        Write-Host "Deployed $($ea.Src) -> $dst"
    }
}

# --- 3. License file (status ACTIVE) ----------------------------------------
foreach ($t in $terms) {
    if ($null -eq $t.Files) { continue }
    New-Item -ItemType Directory -Force -Path $t.Files | Out-Null
    $licPath = Join-Path $t.Files "PAT_license.txt"
    $key = if ($LicenseKey) { $LicenseKey } else { "UNSET" }
    Set-Content -Path $licPath -Value (
        "{`"status`":`"ACTIVE`",`"plan`":`"PRO`",`"auth`":`"OK`",`"device`":`"OK`",`"session`":`"OK`",`"trading`":`"ENABLED`",`"key`":`"$key`"}"
    )
    Write-Host "Wrote license -> $licPath"
}

# --- 4. Auto-start the agent (Task Scheduler, no 3rd-party deps) ------------
if (-not $NoService) {
    $taskName = "PredictATradeAgent"
    schtasks /Delete /TN $taskName /F 2>$null
    $res = schtasks /Create /TN $taskName /TR "`"$runCmd`"" /SC ONSTART /RU SYSTEM /RL HIGHEST 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Registered auto-start task '$taskName'. Start it with: schtasks /Run /TN $taskName"
    } else {
        Write-Warning "Could not register task ($res). Start $runCmd manually or at login."
    }
}

Write-Host "`nDone. Agent at $agentExe | Gateway: $GatewayUrl | License keys written."
Write-Host "Next: attach PredictATrade EA + MasterNode EA to XAUUSD charts in MT, set LicenseKey if prompted."
