<#
.SYNOPSIS
    Predict-A-Trade XAUUSD — Windows Master Node (data-only) Installer
.DESCRIPTION
    Role-specific entry point for the Master Node. It downloads the shared
    installer and runs it in "master" mode, fetching the role binary from the
    /windows-agent/master/ subfolder so each role has a distinct, unambiguous URL.
    Usage:  irm https://downloads.predictatrade.com/windows-agent/master/install.ps1 | iex
#>

param([string]$EngineHost = "live.predictatrade.com")

$root = "https://downloads.predictatrade.com/windows-agent"
$tmp  = Join-Path $env:TEMP ("pat_install_" + [guid]::NewGuid().ToString("N") + ".ps1")
try {
    Invoke-WebRequest -Uri "$root/install.ps1" -OutFile $tmp -UseBasicParsing -TimeoutSec 30
} catch {
    Write-Host "[install-master] ERROR: failed to download installer: $_"
    exit 1
}
$p = Start-Process -FilePath "powershell.exe" -ArgumentList "-ExecutionPolicy","Bypass","-NoProfile","-File","`"$tmp`"","-Mode","master","-EngineHost",$EngineHost,"-BaseUrl","$root/master" -Verb RunAs -Wait -PassThru
exit $p.ExitCode
