<#
.SYNOPSIS
    Predict-A-Trade XAUUSD — Windows Master Node Installer (bootstrap)
.DESCRIPTION
    Downloads the shared installer and runs it in Master Node (data-only) mode.
    NOTE: do NOT pass -BaseUrl with a role suffix — the shared installer appends
    the role directory itself ($RoleDir). Passing ".../windows-agent/master"
    caused the 2026-09-01 master/master/ 404 download failure.
    Usage:  irm https://downloads.predictatrade.com/windows-agent/master/install.ps1 | iex
#>

$BaseUrl = "https://downloads.predictatrade.com/windows-agent"
$tmp = Join-Path $env:TEMP ("pat_install_master_" + [guid]::NewGuid().ToString("N") + ".ps1")
try {
    Invoke-WebRequest -Uri "$BaseUrl/install.ps1" -OutFile $tmp -UseBasicParsing -TimeoutSec 30 -Headers @{ "Cache-Control" = "no-cache" }
} catch {
    Write-Host "[install-master] ERROR: failed to download installer: $_"
    exit 1
}

$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if ($isAdmin) {
    # Already elevated — run inline so output is visible in THIS terminal.
    & powershell.exe -ExecutionPolicy Bypass -NoProfile -File "$tmp" -Mode master -BaseUrl "$BaseUrl"
    exit $LASTEXITCODE
}

# Not elevated: request elevation. Output will appear in the new elevated window
# and is also captured to %TEMP%\pat_install_master.log by install.ps1 itself.
$p = Start-Process -FilePath "powershell.exe" -ArgumentList "-ExecutionPolicy","Bypass","-NoProfile","-File","`"$tmp`"","-Mode","master","-BaseUrl","$BaseUrl" -Verb RunAs -Wait -PassThru
exit $p.ExitCode