<# Master Node (data/coordination) installer wrapper — re-invokes the shared installer in master mode. #>
param(
    [string]$EngineHost = "localhost",
    [int]$GatewayPort   = 8080,
    [string]$GatewayUrl = "",
    [string]$BaseUrl    = "",
    [string]$LicenseKey = "",
    [string]$InstallRoot = "C:\PredictATrade"
)
$here = Split-Path -Parent $MyInvocation.MyCommand.Path
$args = @("-ExecutionPolicy","Bypass","-NoProfile","-File","""$here\install-windows-agent.ps1""","-Mode","master",
          "-EngineHost",$EngineHost,"-GatewayPort",$GatewayPort)
if ($GatewayUrl) { $args += @("-GatewayUrl",$GatewayUrl) }
if ($BaseUrl)    { $args += @("-BaseUrl",$BaseUrl) }
if ($LicenseKey) { $args += @("-LicenseKey",$LicenseKey) }
$args += @("-InstallRoot",$InstallRoot)
Start-Process -FilePath "powershell.exe" -ArgumentList $args -Verb RunAs -Wait
