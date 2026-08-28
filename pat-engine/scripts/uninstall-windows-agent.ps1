<#
.SYNOPSIS
    Predict-A-Trade pat-engine Windows Agent — Uninstaller (client/execution only).
.DESCRIPTION
    Stops & removes the agent service, kills the agent process, removes the install
    directory and cached nssm, and cleans MetaTrader IPC/license files. The new
    pat-engine has no separate "master" role, so only the client agent is handled.
#>
[CmdletBinding()]
param([switch]$Silent)

$InstallDir = "C:\PredictATrade\Agent"
$LegacyDirs = @($InstallDir, (Join-Path "C:\PredictATrade" "Client"), (Join-Path "C:\PredictATrade" "Master"), (Join-Path "C:\PredictATrade" "XAUUSD"))
$Services = @("pat-agent-client", "pat-agent-master")   # master only listed for backward-cleanup
$NssmExe  = "nssm.exe"
$CommonNssm = Join-Path $env:ProgramData "PredictATrade\nssm.exe"

function Get-RoleNssm {
    foreach ($d in $LegacyDirs + @((Join-Path $env:ProgramData "PredictATrade"))) {
        $p = Join-Path $d $NssmExe
        if (Test-Path $p) { return $p }
    }
    $cmd = Get-Command nssm.exe -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }
    return $null
}

$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    $tmp = Join-Path $env:TEMP ("pat_uninstall_" + [guid]::NewGuid().ToString("N") + ".ps1")
    Copy-Item $MyInvocation.MyCommand.Path $tmp -Force
    $p = Start-Process -FilePath "powershell.exe" -ArgumentList "-ExecutionPolicy","Bypass","-NoProfile","-File","""$tmp""","-Silent:`$$Silent" -Verb RunAs -Wait -PassThru
    exit $p.ExitCode
}

Write-Host "Removing Predict-A-Trade pat-engine agent..."
$nssm = Get-RoleNssm
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
Get-Process -Name "pat-windows-agent" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Write-Host "  Killed agent process(es)."

foreach ($d in $LegacyDirs) {
    if (Test-Path $d) { Remove-Item $d -Recurse -Force -ErrorAction SilentlyContinue; Write-Host "  Removed $d" }
}
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
if (Test-Path $CommonNssm) { Remove-Item $CommonNssm -Force -ErrorAction SilentlyContinue }
Write-Host "Uninstall complete."
if (-not $Silent) { Read-Host "Press Enter to close" }
