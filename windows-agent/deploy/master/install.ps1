<#
.SYNOPSIS
    Predict-A-Trade XAUUSD — Windows Master Node Installer
.DESCRIPTION
    Installs the Predict-A-Trade Windows Master Node (data-only role) as a
    Windows service. The Master Node connects to the engine DATA port 13091 and
    streams market/structure data; it NEVER executes orders (execution is the
    Client Agent's responsibility). A Client and a Master Node can run on the
    same machine as separate services on separate ports.
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
    & powershell.exe -ExecutionPolicy Bypass -NoProfile -File "$tmp" -Mode master -BaseUrl "$BaseUrl/master"
    exit $LASTEXITCODE
}

# Not elevated: request elevation. Output will appear in the new elevated window
# and is also captured to %TEMP%\pat_install_master.log by install.ps1 itself.
$p = Start-Process -FilePath "powershell.exe" -ArgumentList "-ExecutionPolicy","Bypass","-NoProfile","-File","`"$tmp`"","-Mode","master","-BaseUrl","$BaseUrl/master" -Verb RunAs -Wait -PassThru
exit $p.ExitCode
