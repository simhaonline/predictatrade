<#
.SYNOPSIS
    Predict-A-Trade XAUUSD — Windows Client Agent Installer
.DESCRIPTION
    Installs the Predict-A-Trade Windows Client Agent (execution role) as a
    Windows service. The Client Agent connects to the engine EXEC port 13081 and
    is authorized to place/close XAUUSD orders on behalf of subscribed users.
    Usage:  irm https://downloads.predictatrade.com/windows-agent/install-client.ps1 | iex
#>

$BaseUrl = "https://downloads.predictatrade.com/windows-agent"
$tmp = Join-Path $env:TEMP "pat_install_client_$(Get-Random).ps1"
try {
    Invoke-WebRequest -Uri "$BaseUrl/install.ps1" -OutFile $tmp -UseBasicParsing -TimeoutSec 30 -Headers @{ "Cache-Control" = "no-cache" }
} catch {
    Write-Host "[install-client] ERROR: failed to download installer: $_"
    exit 1
}
$p = Start-Process -FilePath "powershell.exe" -ArgumentList "-ExecutionPolicy","Bypass","-NoProfile","-File","`"$tmp`"","-Mode","client" -Verb RunAs -Wait -PassThru
exit $p.ExitCode
