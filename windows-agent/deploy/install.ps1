<#
.SYNOPSIS
    Predict-A-Trade XAUUSD — Windows Agent Installer
.DESCRIPTION
    Downloads and installs the Predict-A-Trade Windows Agent as a Windows service.
    Handles fresh installs and updates seamlessly.
    Usage: irm https://downloads.predictatrade.com/windows-agent/install.ps1 | iex
#>

# ─── Config ───
$BaseUrl     = "https://downloads.predictatrade.com/windows-agent"
$InstallDir  = "C:\PredictATrade\XAUUSD"
$ServiceName = "pat-agent"
$EventSource = "pat-agent"
$TaskName    = "PredictATradeHealthCheck"
$AgentExe    = "pat-agent.exe"
$NssmExe     = "nssm.exe"

# ─── Self-elevation ───
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Host "[install] Admin rights required — UAC prompt will appear..."
    $tempScript = Join-Path $env:TEMP "pat_install_$(Get-Random).ps1"
    try {
        $scriptContent = Invoke-WebRequest -Uri "$BaseUrl/install.ps1" -UseBasicParsing -TimeoutSec 30 | Select-Object -ExpandProperty Content
        Set-Content -Path $tempScript -Value $scriptContent -Encoding UTF8
        $p = Start-Process -FilePath "powershell.exe" -ArgumentList "-ExecutionPolicy","Bypass","-NoProfile","-File","`"$tempScript`"" -Verb RunAs -Wait -PassThru
        exit $p.ExitCode
    } catch {
        Write-Host "[install] ERROR: Elevation failed: $_"
        Write-Host "[install] Please open PowerShell as Administrator and run: irm $BaseUrl/install.ps1 | iex"
        exit 1
    } finally {
        Remove-Item $tempScript -Force -ErrorAction SilentlyContinue
    }
}

# ─── NOW RUNNING AS ADMIN ───
Write-Host ""
Write-Host "=========================================="
Write-Host "  Predict-A-Trade XAUUSD — Installer v1.2.32"
Write-Host "=========================================="
Write-Host ""

# Step 1: Add Defender exclusion BEFORE downloading (prevents quarantine)
Write-Host "[1/9] Adding Windows Defender exclusion..."
try {
    Add-MpPreference -ExclusionPath "C:\PredictATrade" -ErrorAction SilentlyContinue
    Add-MpPreference -ExclusionProcess $AgentExe -ErrorAction SilentlyContinue
    Write-Host "  OK"
} catch {
    Write-Host "  WARN: Could not add Defender exclusion (non-fatal): $_"
}

# Step 2: Create directories
Write-Host "[2/9] Creating installation directory..."
if (-not (Test-Path $InstallDir)) { New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null }
$logsDir = Join-Path $InstallDir "logs"
if (-not (Test-Path $logsDir)) { New-Item -ItemType Directory -Path $logsDir -Force | Out-Null }
Write-Host "  OK: $InstallDir"

# Step 3: Stop existing service if running
Write-Host "[3/9] Stopping existing service if running..."
$nssmDest = Join-Path $InstallDir $NssmExe
$svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($svc) {
    if ($svc.Status -eq "Running") {
        if (Test-Path $nssmDest) { & $nssmDest stop $ServiceName 2>&1 | Out-Null }
        else { Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue }
        Start-Sleep -Seconds 3
    }
    Write-Host "  OK: Service stopped"
} else {
    Write-Host "  OK: No existing service"
}

# Step 4: Kill any running pat-agent processes
Write-Host "[4/9] Killing any running pat-agent processes..."
$procs = Get-Process -Name "pat-agent" -ErrorAction SilentlyContinue
if ($procs) {
    $procs | Stop-Process -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 2
    Write-Host "  OK: Killed $($procs.Count) process(es)"
} else {
    Write-Host "  OK: No running processes"
}

# Step 5: Download pat-agent.exe
Write-Host "[5/9] Downloading pat-agent.exe..."
$agentPath = Join-Path $InstallDir $AgentExe
try {
    if (Test-Path $agentPath) { Remove-Item $agentPath -Force -ErrorAction SilentlyContinue }
    Invoke-WebRequest -Uri "$BaseUrl/pat-agent.exe" -OutFile $agentPath -UseBasicParsing -TimeoutSec 120
    Unblock-File -Path $agentPath -ErrorAction SilentlyContinue
    $fileSize = (Get-Item $agentPath).Length
    Write-Host "  OK: Downloaded $AgentExe ($([math]::Round($fileSize/1MB, 1)) MB)"
} catch {
    Write-Host "  FATAL: Download failed: $_"
    Read-Host "Press Enter to close"
    exit 1
}

# Step 6: Download NSSM (service manager — wraps the agent as a Windows service)
Write-Host "[6/9] Downloading NSSM (service manager)..."
$is64bit = [Environment]::Is64BitOperatingSystem
$nssmArch = if ($is64bit) { "nssm/win64/nssm.exe" } else { "nssm/win32/nssm.exe" }
$nssmDownloaded = $false
try {
    if (Test-Path $nssmDest) { Remove-Item $nssmDest -Force -ErrorAction SilentlyContinue }
    Invoke-WebRequest -Uri "$BaseUrl/$nssmArch" -OutFile $nssmDest -UseBasicParsing -TimeoutSec 60
    Unblock-File -Path $nssmDest -ErrorAction SilentlyContinue
    if (Test-Path $nssmDest) {
        $nssmDownloaded = $true
        Write-Host "  OK: Downloaded nssm.exe"
    }
} catch {
    Write-Host "  WARN: NSSM download failed: $_"
}

# Step 6b: Download supporting scripts
Write-Host "[6b/9] Downloading supporting scripts..."
$supportFiles = @("health-check.ps1", "status.ps1", "notify.ps1")
foreach ($file in $supportFiles) {
    try {
        $dest = Join-Path $InstallDir $file
        Invoke-WebRequest -Uri "$BaseUrl/$file" -OutFile $dest -UseBasicParsing -TimeoutSec 30
        Unblock-File -Path $dest -ErrorAction SilentlyContinue
    } catch {
        Write-Host "  WARN: Could not download ${file}: $_"
    }
}

# Step 6c: Download settings.json if not exists (preserve existing)
$settingsPath = Join-Path $InstallDir "settings.json"
if (-not (Test-Path $settingsPath)) {
    try {
        Invoke-WebRequest -Uri "$BaseUrl/settings.json" -OutFile $settingsPath -UseBasicParsing -TimeoutSec 30
        Write-Host "  OK: Downloaded settings.json"
    } catch {
        Write-Host "  WARN: Could not download settings.json: $_"
    }
} else {
    Write-Host "  OK: Preserving existing settings.json"
}

# Step 7: Remove old service and create fresh with NSSM
Write-Host "[7/9] Creating Windows service with NSSM..."
if ($svc) {
    if ($nssmDownloaded) { & $nssmDest remove $ServiceName confirm 2>&1 | Out-Null }
    else { sc.exe delete $ServiceName 2>&1 | Out-Null }
    Start-Sleep -Seconds 2
}

if ($nssmDownloaded -and (Test-Path $nssmDest)) {
    & $nssmDest install $ServiceName $agentPath 2>&1 | Out-Null
    & $nssmDest set $ServiceName AppDirectory $InstallDir 2>&1 | Out-Null
    & $nssmDest set $ServiceName AppStdout (Join-Path $logsDir "stdout.log") 2>&1 | Out-Null
    & $nssmDest set $ServiceName AppStderr (Join-Path $logsDir "stderr.log") 2>&1 | Out-Null
    & $nssmDest set $ServiceName AppRotateFiles 1 2>&1 | Out-Null
    & $nssmDest set $ServiceName AppRotateOnline 1 2>&1 | Out-Null
    & $nssmDest set $ServiceName AppNoConsole 1 2>&1 | Out-Null
    & $nssmDest set $ServiceName AppExit Default Restart 2>&1 | Out-Null
    & $nssmDest set $ServiceName AppExit 0 Exit 2>&1 | Out-Null
    & $nssmDest set $ServiceName AppRestartDelay 5000 2>&1 | Out-Null
    & $nssmDest set $ServiceName DisplayName "Predict-A-Trade XAUUSD Agent" 2>&1 | Out-Null
    & $nssmDest set $ServiceName Description "Predict-A-Trade XAUUSD Windows Agent" 2>&1 | Out-Null
    & $nssmDest set $ServiceName Start SERVICE_AUTO_START 2>&1 | Out-Null
    Write-Host "  OK: Service created with NSSM (auto-restart on crash)"
} else {
    Write-Host "  Fallback: Using sc.exe..."
    sc.exe create $ServiceName binPath= $agentPath start= auto 2>&1 | Out-Null
    sc.exe description $ServiceName "Predict-A-Trade XAUUSD Windows Agent" 2>&1 | Out-Null
}

# Step 8: Start the service
Write-Host "[8/9] Starting service..."
if ($nssmDownloaded) {
    & $nssmDest start $ServiceName 2>&1 | Out-Null
} else {
    try { Start-Service -Name $ServiceName -ErrorAction SilentlyContinue } catch {}
}
Start-Sleep -Seconds 3

# Check if service is actually running
$svcCheck = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($svcCheck -and $svcCheck.Status -eq "Running") {
    Write-Host "  OK: Service is RUNNING"
} else {
    Write-Host "  WARN: Service not running — checking logs..."
    $stderrLog = Join-Path $logsDir "stderr.log"
    $stdoutLog = Join-Path $logsDir "stdout.log"
    if (Test-Path $stderrLog) {
        Write-Host "  --- stderr.log (last 10 lines) ---"
        Get-Content $stderrLog -Tail 10 -ErrorAction SilentlyContinue | ForEach-Object { Write-Host "    $_" }
    }
    if (Test-Path $stdoutLog) {
        Write-Host "  --- stdout.log (last 10 lines) ---"
        Get-Content $stdoutLog -Tail 10 -ErrorAction SilentlyContinue | ForEach-Object { Write-Host "    $_" }
    }
    if (-not (Test-Path $stderrLog) -and -not (Test-Path $stdoutLog)) {
        Write-Host "  No log files found — agent may have been blocked by antivirus"
        Write-Host "  Try manually: Open $InstallDir and double-click pat-agent.exe"
    }
}

# Step 9: Save version + verify health endpoint
Write-Host "[9/9] Finalizing..."
# Fetch the actual version from the server's version.txt (single source of truth)
try {
    $serverVersion = (Invoke-WebRequest -Uri "$BaseUrl/version.txt" -UseBasicParsing -TimeoutSec 10).Content.Trim()
    Write-Host "  Server version: v$serverVersion"
} catch {
    $serverVersion = "1.2.31"
    Write-Host "  WARN: Could not fetch server version — using default v$serverVersion"
}
Set-Content -Path (Join-Path $InstallDir "version.txt") -Value $serverVersion -NoNewline

# Try to verify health endpoint
Start-Sleep -Seconds 2
try {
    $healthResp = Invoke-WebRequest -Uri "http://127.0.0.1:9000/health" -UseBasicParsing -TimeoutSec 5 -ErrorAction Stop
    Write-Host "  OK: Health endpoint responding (HTTP $($healthResp.StatusCode))"
} catch {
    Write-Host "  WARN: Health endpoint not responding yet (service may still be starting)"
}

# ─── Summary ───
Write-Host ""
Write-Host "=========================================="
Write-Host "  Installation Complete! v$serverVersion"
Write-Host "=========================================="
Write-Host "  Service:     $ServiceName"
Write-Host "  Status:      $(if ($svcCheck -and $svcCheck.Status -eq 'Running') { 'Running ✓' } else { 'Check logs above' })"
Write-Host "  Install Dir: $InstallDir"
Write-Host "  Health:      http://127.0.0.1:9000"
Write-Host "  Logs:        $logsDir"
Write-Host ""
Write-Host "  To uninstall: irm $BaseUrl/uninstall.ps1 | iex"
Write-Host "  To update:     irm $BaseUrl/install.ps1 | iex"
Write-Host "=========================================="
Write-Host ""
Read-Host "Press Enter to close"
