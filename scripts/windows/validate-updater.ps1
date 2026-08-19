# Validate agent update process: version discovery, download, verify, apply, rollback
param([string]$AgentPath = "C:\Program Files\PredictATrade\PredictATradeAgent.exe")
$ErrorActionPreference = "Continue"

Write-Host "=== Updater Validation ===" -ForegroundColor Cyan

# Check if update command exists
Write-Host "Step 1: Version discovery..."
Write-Host "[MANUAL] Run: $AgentPath -check-update" -ForegroundColor Yellow
Write-Host "[MANUAL] Verify agent queries backend for new version" -ForegroundColor Yellow

Write-Host "Step 2: Download + checksum verification..."
Write-Host "[MANUAL] If update available, verify:" -ForegroundColor Yellow
Write-Host "[MANUAL]   - Download uses HTTPS" -ForegroundColor White
Write-Host "[MANUAL]   - SHA-256 checksum is verified before staging" -ForegroundColor White
Write-Host "[MANUAL]   - Unverified binary is never executed" -ForegroundColor White

Write-Host "Step 3: Atomic replacement..."
Write-Host "[MANUAL] Verify old binary is backed up to .bak" -ForegroundColor Yellow
Write-Host "[MANUAL] Verify new binary replaces old atomically" -ForegroundColor Yellow

Write-Host "Step 4: Service restart..."
Write-Host "[MANUAL] Verify service restarts with new version" -ForegroundColor Yellow
Write-Host "[MANUAL] Verify new version is reported in heartbeat" -ForegroundColor Yellow

Write-Host "Step 5: Rollback test..."
Write-Host "[MANUAL] To test rollback:" -ForegroundColor Yellow
Write-Host "[MANUAL]   1. Stop service" -ForegroundColor White
Write-Host "[MANUAL]   2. Replace binary with invalid one" -ForegroundColor White
Write-Host "[MANUAL]   3. Start service (should fail)" -ForegroundColor White
Write-Host "[MANUAL]   4. Verify .bak is restored" -ForegroundColor White
Write-Host "[MANUAL]   5. Service should start with old version" -ForegroundColor White
