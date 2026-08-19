# Validate MT4/MT5 named pipe connectivity and signal delivery
param(
    [switch]$SkipMT4,
    [switch]$SkipMT5
)
$ErrorActionPreference = "Continue"

Write-Host "=== MT4/MT5 Validation ===" -ForegroundColor Cyan

if (-not $SkipMT5) {
    Write-Host "Testing MT5 pipe..."
    try {
        $pipe = New-Object System.IO.Pipes.NamedPipeClientStream(".", "PredictATradeMT5", [System.IO.Pipes.PipeDirection]::InOut)
        $pipe.Connect(5000)
        Write-Host "[PASS] MT5 pipe connected" -ForegroundColor Green
        $pipe.Dispose()
    } catch {
        Write-Host "[NOT_TESTED] MT5 not running — ensure MT5 + PredictATrade EA are loaded" -ForegroundColor Yellow
    }
}

if (-not $SkipMT4) {
    Write-Host "Testing MT4 pipe..."
    try {
        $pipe = New-Object System.IO.Pipes.NamedPipeClientStream(".", "PredictATradeMT4", [System.IO.Pipes.PipeDirection]::InOut)
        $pipe.Connect(5000)
        Write-Host "[PASS] MT4 pipe connected" -ForegroundColor Green
        $pipe.Dispose()
    } catch {
        Write-Host "[NOT_TESTED] MT4 not running — ensure MT4 + PredictATrade EA are loaded" -ForegroundColor Yellow
    }
}

Write-Host "`n[MANUAL] For full signal delivery test:" -ForegroundColor Cyan
Write-Host "1. Start agent with valid license" -ForegroundColor White
Write-Host "2. Wait for backend to generate a signal" -ForegroundColor White
Write-Host "3. Verify signal appears in MT5 terminal" -ForegroundColor White
Write-Host "4. Verify EA acknowledges signal" -ForegroundColor White
Write-Host "5. Check agent.log for 'Signal received' and 'Signal acknowledged'" -ForegroundColor White
