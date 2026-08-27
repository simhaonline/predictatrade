<#
.SYNOPSIS
    Predict-A-Trade XAUUSD — Windows Master Node Installer
.DESCRIPTION
    Installs the Predict-A-Trade Windows Master Node (data-only role) as a
    Windows service. The Master Node connects to the engine DATA port 13091 and
    streams market/structure data; it NEVER executes orders (execution is the
    Client Agent's responsibility). A Client and a Master Node can run on the
    same machine as separate services on separate ports.
    Usage:  irm https://downloads.predictatrade.com/windows-agent/install-master.ps1 | iex
#>

$BaseUrl = "https://downloads.predictatrade.com/windows-agent"
$tmp = Join-Path $env:TEMP "pat_install_master_$(Get-Random).ps1"
try {
    Invoke-WebRequest -Uri "$BaseUrl/install.ps1" -OutFile $tmp -UseBasicParsing -TimeoutSec 30
} catch {
    Write-Host "[install-master] ERROR: failed to download installer: $_"
    exit 1
}
$p = Start-Process -FilePath "powershell.exe" -ArgumentList "-ExecutionPolicy","Bypass","-NoProfile","-File","`"$tmp`"","-Mode","master" -Verb RunAs -Wait -PassThru
exit $p.ExitCode
