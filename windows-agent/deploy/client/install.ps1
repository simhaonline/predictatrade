<#
.SYNOPSIS
    Predict-A-Trade XAUUSD — Windows Client Agent Installer
.DESCRIPTION
    Installs the Predict-A-Trade Windows Client Agent (execution role) as a
    Windows service. The Client Agent connects to the engine EXEC port 13081 and
    is authorized to place/close XAUUSD orders on behalf of subscribed users.
    Usage:  irm https://downloads.predictatrade.com/windows-agent/client/install.ps1 | iex
#>

$BaseUrl = "https://downloads.predictatrade.com/windows-agent"
$tmp = Join-Path $env:TEMP ("pat_install_client_" + [guid]::NewGuid().ToString("N") + ".ps1")
try {
    Invoke-WebRequest -Uri "$BaseUrl/install.ps1" -OutFile $tmp -UseBasicParsing -TimeoutSec 30 -Headers @{ "Cache-Control" = "no-cache" }
} catch {
    Write-Host "[install-client] ERROR: failed to download installer: $_"
    exit 1
}

$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if ($isAdmin) {
    # Already elevated — run inline so output is visible in THIS terminal.
    & powershell.exe -ExecutionPolicy Bypass -NoProfile -File "$tmp" -Mode client -BaseUrl "$BaseUrl"
    exit $LASTEXITCODE
}

# Not elevated: request elevation. Output will appear in the new elevated window
# and is also captured to %TEMP%\pat_install_client.log by install.ps1 itself.
$p = Start-Process -FilePath "powershell.exe" -ArgumentList "-ExecutionPolicy","Bypass","-NoProfile","-File","`"$tmp`"","-Mode","client","-BaseUrl","$BaseUrl" -Verb RunAs -Wait -PassThru
exit $p.ExitCode
