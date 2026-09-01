<#
.SYNOPSIS   Predict-A-Trade XAUUSD — Windows Master Node Installer
.DESCRIPTION
    Thin role wrapper kept in the master folder for organization. It delegates to
    the canonical installer (windows-agent/deploy/install.ps1) in master mode.
    Usage:  irm https://downloads.predictatrade.com/windows-agent/master/install.ps1 | iex
#>
try {
    irm "https://downloads.predictatrade.com/windows-agent/install-master.ps1" | iex
} catch {
    Write-Host "[master/install] ERROR: failed to start installer: $_"
    exit 1
}
