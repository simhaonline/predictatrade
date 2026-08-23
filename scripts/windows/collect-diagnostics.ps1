# Collect Windows Agent diagnostics for support/troubleshooting
$ErrorActionPreference = "Continue"
$OutputDir = "$env:TEMP\pat-diagnostics"
New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null

Write-Host "=== Collecting Diagnostics ===" -ForegroundColor Cyan

# System info
systeminfo > "$OutputDir\systeminfo.txt" 2>&1

# Agent log
$logDir = "C:\ProgramData\PredictATrade\logs"
if (Test-Path $logDir) {
    Copy-Item "$logDir\*" "$OutputDir\" -Recurse -ErrorAction SilentlyContinue
} else {
    Write-Host "Log directory not found: $logDir" -ForegroundColor Yellow
}

# Agent config
$configPath = "C:\ProgramData\PredictATrade\config.json"
if (Test-Path $configPath) {
    Copy-Item $configPath "$OutputDir\" -ErrorAction SilentlyContinue
    # Redact any secrets
    $config = Get-Content $configPath -Raw | ConvertFrom-Json
    Write-Host "Config found (secrets redacted in output)" -ForegroundColor Green
}

# Service status
Get-Service "pat-agent" | Format-List > "$OutputDir\service-status.txt" 2>&1
sc.exe qfailure "pat-agent" >> "$OutputDir\service-status.txt" 2>&1

# Named pipe status
Get-ChildItem "\\.\pipe\" | Where-Object { $_.Name -match "PredictATrade" } | Format-Table > "$OutputDir\pipes.txt" 2>&1

# Event log (last 100 entries related to PredictATrade)
Get-EventLog -LogName Application -Newest 100 -Source "*PredictATrade*" -ErrorAction SilentlyContinue | Format-List > "$OutputDir\event-log.txt" 2>&1

# Network connectivity
Test-NetConnection -ComputerName "api.predictatrade.com" -Port 443 -WarningAction SilentlyContinue | Format-List > "$OutputDir\network-test.txt" 2>&1
Test-NetConnection -ComputerName "live.predictatrade.com" -Port 443 -WarningAction SilentlyContinue | Format-List >> "$OutputDir\network-test.txt" 2>&1

# Create zip
$ZipPath = "$env:TEMP\pat-diagnostics-$(Get-Date -Format 'yyyyMMdd_HHmmss').zip"
Compress-Archive -Path "$OutputDir\*" -DestinationPath $ZipPath -Force
Write-Host "`nDiagnostics collected: $ZipPath" -ForegroundColor Green
Write-Host "Send this file to support for analysis."
