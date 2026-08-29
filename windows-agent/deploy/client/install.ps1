<#
.SYNOPSIS
    Predict-A-Trade XAUUSD — Windows Client Agent (execution) Installer
.DESCRIPTION
    Role-specific entry point for the Client Agent. It downloads the shared
    installer and runs it in "client" mode, fetching the role binary from the
    /windows-agent/client/ subfolder so each role has a distinct, unambiguous URL.
    Usage:  irm https://downloads.predictatrade.com/windows-agent/client/install.ps1 | iex
#>

param([string]$EngineHost = "live.predictatrade.com", [string]$LicenseKey = "")

$root = "https://downloads.predictatrade.com/windows-agent"
$tmp  = Join-Path $env:TEMP ("pat_install_" + [guid]::NewGuid().ToString("N") + ".ps1")
try {
    Invoke-WebRequest -Uri "$root/install.ps1" -OutFile $tmp -UseBasicParsing -TimeoutSec 30
} catch {
    Write-Host "[install-client] ERROR: failed to download installer: $_"
    exit 1
}
$p = Start-Process -FilePath "powershell.exe" -ArgumentList "-ExecutionPolicy","Bypass","-NoProfile","-File","`"$tmp`"","-Mode","client","-EngineHost",$EngineHost,"-LicenseKey",$LicenseKey,"-Unattended","-BaseUrl","$root/client" -Verb RunAs -Wait -PassThru
exit $p.ExitCode
