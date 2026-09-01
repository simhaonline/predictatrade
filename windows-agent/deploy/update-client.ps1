<#
.SYNOPSIS
    Predict-A-Trade XAUUSD — Windows Client Agent Updater
.DESCRIPTION
    Updates the installed Client Agent (execution role) to the latest version.
    Usage:  irm https://downloads.predictatrade.com/windows-agent/update-client.ps1 | iex
#>
$BaseUrl = "https://downloads.predictatrade.com/windows-agent"
$tmp = Join-Path $env:TEMP ("pat_update_client_" + [guid]::NewGuid().ToString("N") + ".ps1")
try {
    Invoke-WebRequest -Uri "$BaseUrl/update.ps1" -OutFile $tmp -UseBasicParsing -TimeoutSec 30
} catch {
    Write-Host "[update-client] ERROR: failed to download updater: $_"
    exit 1
}
$p = Start-Process -FilePath "powershell.exe" -ArgumentList "-ExecutionPolicy","Bypass","-NoProfile","-File","`"$tmp`"","-Mode","client" -Verb RunAs -Wait -PassThru
exit $p.ExitCode
