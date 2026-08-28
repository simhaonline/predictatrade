<#
.SYNOPSIS
    Predict-A-Trade pat-engine Windows Agent — Uninstaller (role-aware, adapted from
    the windows-agent reference project). Stops & removes the service(s), kills the
    agent process(es), removes the role install directories and the cached nssm, and
    cleans IPC/license files. Always cleans BOTH roles regardless of -Mode (which only
    affects messaging), so a default uninstall leaves the machine fully clean.
#>
[CmdletBinding()]
param(
    [ValidateSet("client","master","all")][string]$Mode = "client",
    [switch]$Silent
)

$InstallRoot = "C:\PredictATrade"
$DirsToRemove = @(
    (Join-Path $InstallRoot "Client"),
    (Join-Path $InstallRoot "Master"),
    (Join-Path $InstallRoot "XAUUSD")   # legacy single-dir install
)
$Services = @("pat-agent-client", "pat-agent-master")
$NssmExe  = "nssm.exe"
$CommonNssm = Join-Path $env:ProgramData "PredictATrade\nssm.exe"

function Get-RoleNssm {
    foreach ($d in $DirsToRemove + @((Join-Path $env:ProgramData "PredictATrade"))) {
        $p = Join-Path $d $NssmExe
        if (Test-Path $p) { return $p }
    }
    $cmd = Get-Command nssm.exe -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }
    return $null
}

# Self-elevate
$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    $tmp = Join-Path $env:TEMP ("pat_uninstall_" + [guid]::NewGuid().ToString("N") + ".ps1")
    if ($BaseUrl) {
        (Invoke-WebRequest -Uri "$BaseUrl/uninstall-windows-agent.ps1" -UseBasicParsing -TimeoutSec 30).Content | Set-Content -Path $tmp -Encoding UTF8
    } else {
        Copy-Item $MyInvocation.MyCommand.Path $tmp -Force
    }
    $p = Start-Process -FilePath "powershell.exe" -ArgumentList "-ExecutionPolicy","Bypass","-NoProfile","-File","""$tmp""","-Mode",$Mode,"-Silent:`$$Silent" -Verb RunAs -Wait -PassThru
    exit $p.ExitCode
}

Write-Host "Removing Predict-A-Trade pat-engine agent ($Mode)..."
$nssm = Get-RoleNssm

# 1. Stop + remove services (both roles always)
foreach ($svc in $Services) {
    $s = Get-Service -Name $svc -ErrorAction SilentlyContinue
    if ($s) {
        try {
            if ($nssm -and (Test-Path $nssm)) { & $nssm stop $svc 2>&1 | Out-Null; & $nssm remove $svc confirm 2>&1 | Out-Null }
            sc.exe stop $svc 2>&1 | Out-Null; sc.exe delete $svc 2>&1 | Out-Null
            Write-Host "  Removed service $svc"
        } catch { Write-Host "  WARN: could not remove $svc`: $_" }
    }
}

# 2. Kill any lingering agent process
Get-Process -Name "pat-windows-agent" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Write-Host "  Killed agent process(es)."

# 3. Remove install directories
foreach ($d in $DirsToRemove) {
    if (Test-Path $d) { Remove-Item $d -Recurse -Force -ErrorAction SilentlyContinue; Write-Host "  Removed $d" }
}

# 4. Clean MT common Files IPC/license artifacts (best-effort)
$mtRoot = Join-Path $env:APPDATA "MetaQuotes\Terminal"
if (Test-Path $mtRoot) {
    foreach ($term in (Get-ChildItem $mtRoot -Directory -ErrorAction SilentlyContinue)) {
        $common = Join-Path $term.FullName "Common\Files"
        foreach ($f in @("PAT_license.txt","PAT_signals.txt")) {
            $p = Join-Path $common $f
            if (Test-Path $p) { Remove-Item $p -Force -ErrorAction SilentlyContinue }
        }
    }
}
Write-Host "  Cleaned MetaTrader IPC/license files (EAs left in place — remove manually if desired)."

# 5. Remove cached nssm
if (Test-Path $CommonNssm) { Remove-Item $CommonNssm -Force -ErrorAction SilentlyContinue }

Write-Host "Uninstall complete. Remove the MT4/MT5 EAs from each terminal's Experts folder manually."
if (-not $Silent) { Read-Host "Press Enter to close" }
