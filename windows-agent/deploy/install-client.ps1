<#
.SYNOPSIS
    Predict-A-Trade XAUUSD — Windows Client Agent Installer
.DESCRIPTION
    Installs the Predict-A-Trade Windows Client Agent (execution role) as a
    Windows service. The Client Agent connects to the engine EXEC port 13081 and
    is authorized to place/close XAUUSD orders on behalf of subscribed users.
    Usage (production / hosted):
        irm https://downloads.predictatrade.com/windows-agent/install-client.ps1 | iex
    Usage (self-hosted engine — point at your own engine host):
        $EngineHost="192.168.1.50"; irm https://downloads.predictatrade.com/windows-agent/install-client.ps1 | iex
.PARAMETER EngineHost
    Engine host the agent should connect to. Defaults to live.predictatrade.com.
    Use the LAN/IP of your own docker stack for self-hosted installs
    (the installer builds ws://host:13081 for local/non-TLS hosts).
.PARAMETER BaseUrl
    Override the installer download location (advanced).
#>
[CmdletBinding()]
param(
    [string]$EngineHost,
    [string]$BaseUrl = "https://downloads.predictatrade.com/windows-agent"
)

$tmp = Join-Path $env:TEMP "pat_install_client_$(Get-Random).ps1"
try {
    Invoke-WebRequest -Uri "$BaseUrl/install.ps1" -OutFile $tmp -UseBasicParsing -TimeoutSec 30
} catch {
    Write-Host "[install-client] ERROR: failed to download installer: $_"
    exit 1
}
$args = @("-ExecutionPolicy","Bypass","-NoProfile","-File","""$tmp""","-Mode","client")
if ($EngineHost) { $args += @("-EngineHost", $EngineHost) }
if ($BaseUrl -ne "https://downloads.predictatrade.com/windows-agent") { $args += @("-BaseUrl", $BaseUrl) }
$p = Start-Process -FilePath "powershell.exe" -ArgumentList $args -Verb RunAs -Wait -PassThru
exit $p.ExitCode
