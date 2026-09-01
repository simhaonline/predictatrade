<#
.SYNOPSIS
    Predict-A-Trade XAUUSD — Windows Agent Updater
.DESCRIPTION
    Updates an installed Client Agent or Master Node to the latest version.
    Re-runs the shared installer in the selected role, which:
      - stops the existing Windows service,
      - KILLS any leftover background cmd/agent process (so the port/binary lock
        can't block the restart),
      - downloads the latest role binary (SHA256-verified against the per-arch
        update-manifest),
      - swaps the exe in place, and restarts — preserving settings.json.
    No full re-install is required; the registered service is reused.

    Usage:
      irm https://downloads.predictatrade.com/windows-agent/update.ps1 | iex -Mode client
      irm https://downloads.predictatrade.com/windows-agent/update.ps1 | iex -Mode master
    (or use the per-role wrappers update-client.ps1 / update-master.ps1)
#>
param(
    [ValidateSet("client","master")][string]$Mode = "client"
)

$BaseUrl = "https://downloads.predictatrade.com/windows-agent"
$tmp = Join-Path $env:TEMP ("pat_update_" + [guid]::NewGuid().ToString("N") + ".ps1")
try {
    Invoke-WebRequest -Uri "$BaseUrl/install.ps1" -OutFile $tmp -UseBasicParsing -TimeoutSec 30
} catch {
    Write-Host "[update] ERROR: failed to download installer: $_"
    exit 1
}

$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if ($isAdmin) {
    # Already elevated — run inline so output is visible in THIS terminal.
    & powershell.exe -ExecutionPolicy Bypass -NoProfile -File "$tmp" -Mode $Mode
    exit $LASTEXITCODE
}

# Not elevated: request elevation (UAC). Output appears in the new elevated window.
$p = Start-Process -FilePath "powershell.exe" `
    -ArgumentList "-ExecutionPolicy","Bypass","-NoProfile","-File","`"$tmp`"","-Mode",$Mode `
    -Verb RunAs -Wait -PassThru
exit $p.ExitCode
