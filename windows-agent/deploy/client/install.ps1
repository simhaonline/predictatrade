<#
.SYNOPSIS   Predict-A-Trade XAUUSD — Windows Client Agent Installer
.DESCRIPTION
    Thin role wrapper kept in the client folder for organization. It delegates to
    the canonical installer (windows-agent/deploy/install.ps1) in client mode.
    Usage:  irm https://downloads.predictatrade.com/windows-agent/client/install.ps1 | iex
#>
try {
    irm "https://downloads.predictatrade.com/windows-agent/install-client.ps1" | iex
} catch {
    Write-Host "[client/install] ERROR: failed to start installer: $_"
    exit 1
}
