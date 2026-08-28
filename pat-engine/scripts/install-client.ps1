<# Client Agent (execution) installer wrapper — invokes the shared installer.
# The new pat-engine has no separate "master" role; the central Go engine aggregates
# all agent feeds, so a single client agent per terminal is the only install path.
param(
    [string]$EngineHost = "localhost",
    [int]$GatewayPort   = 80,
    [string]$GatewayUrl = "",
    [string]$BaseUrl    = "",
    [string]$LicenseKey = "",
    [string]$InstallDir = "C:\PredictATrade\Agent"
)
$here = Split-Path -Parent $MyInvocation.MyCommand.Path
$args = @("-ExecutionPolicy","Bypass","-NoProfile","-File","""$here\install-windows-agent.ps1""",
          "-EngineHost",$EngineHost,"-GatewayPort",$GatewayPort)
if ($GatewayUrl) { $args += @("-GatewayUrl",$GatewayUrl) }
if ($BaseUrl)    { $args += @("-BaseUrl",$BaseUrl) }
if ($LicenseKey) { $args += @("-LicenseKey",$LicenseKey) }
if ($InstallDir -ne "C:\PredictATrade\Agent") { $args += @("-InstallDir",$InstallDir) }
Start-Process -FilePath "powershell.exe" -ArgumentList $args -Verb RunAs -Wait
