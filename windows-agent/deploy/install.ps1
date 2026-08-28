<#
.SYNOPSIS
    Predict-A-Trade XAUUSD — Windows Agent Installer
.DESCRIPTION
    Downloads and installs the Predict-A-Trade Windows Agent as a Windows service.
    Handles fresh installs and updates seamlessly.
    Usage: irm https://downloads.predictatrade.com/windows-agent/install.ps1 | iex
#>

# ─── Parameters ───
#   -Mode client  → Client Agent (execution). Connects to engine exec port 13081.
#   -Mode master  → Master Node (data-only). Connects to engine data port 13091.
# The two roles ship as SEPARATE binaries (pat-agent.exe for client,
# pat-master.exe for master) so there is no shared --mode flag at runtime.
# Logs: client -> agent.log, master -> master_agent.log.
[CmdletBinding()]
param(
    [ValidateSet("client","master")][string]$Mode = "client",
    [string]$EngineHost = "live.predictatrade.com",
    [string]$BaseUrl = "https://downloads.predictatrade.com/windows-agent"
)

# ─── Config ───
$BaseUrl     = "https://downloads.predictatrade.com/windows-agent"
# Root URL always points at the shared assets (nssm, settings, scripts, version).
# $BaseUrl may be overridden to a role subdir (…/master or …/client) so the
# role-specific binary is fetched from there; shared assets always come from root.
$RootUrl     = "https://downloads.predictatrade.com/windows-agent"
# Role-specific installation directory so a Master Node and a Client Agent can
# coexist on the SAME Windows device without sharing binaries, settings, or logs.
$InstallDir  = if ($Mode -eq "master") { "C:\PredictATrade\Master" } else { "C:\PredictATrade\Client" }

# Build the correct engine WebSocket URL for a given host/port/path.
#  - Public host (domain):  wss://host/path   (TLS terminated by nginx on 443,
#    which proxies /ws/v1/data -> pat-realtime:13091 and /ws/v1/agent -> :13081)
#  - Local host (localhost / IP): ws://host:port/path  (direct plaintext engine)
function Resolve-EngineWsUrl {
    param([string]$EngineHost, [int]$Port, [string]$Path)
    $isLocal = ($EngineHost -match '^(localhost|127\.0\.0\.1|::1)$') -or ($EngineHost -match '^\d{1,3}(\.\d{1,3}){3}$')
    if ($isLocal) { return "ws://${EngineHost}:${Port}${Path}" }
    return "wss://${EngineHost}${Path}"
}

# Mode-specific identity (separate Windows services & ports so a Client and a
# Master Node can run side-by-side on the same machine without conflict).
if ($Mode -eq "master") {
    $ServiceName  = "pat-agent-master"
    $AgentMode    = "data"
    $EngineEnvVar = "PAT_DATA_WS_URL"
    $EngineWsUrl  = Resolve-EngineWsUrl -EngineHost $EngineHost -Port 13091 -Path "/ws/v1/data"
    $RoleLabel    = "Master Node (data-only)"
} else {
    $Mode         = "client"
    $ServiceName  = "pat-agent-client"
    $AgentMode    = "exec"
    $EngineEnvVar = "PAT_LIVE_WS_URL"
    $EngineWsUrl  = Resolve-EngineWsUrl -EngineHost $EngineHost -Port 13081 -Path "/ws/v1/agent"
    $RoleLabel    = "Client Agent (execution)"
}
$AgentExe    = if ($Mode -eq "master") { "pat-master.exe" } else { "pat-agent.exe" }
# The role is fixed by the binary itself (pat-agent.exe = client, pat-master.exe
# = Master Node). No --mode flag is passed at runtime.
$AgentArgs   = ""
$NssmExe     = "nssm.exe"

# ─── Architecture detection (multi-arch support) ───
# MT5/MT4 are x64, but we ship amd64/386/arm64 so the agent runs on any Windows
# and the installer never fails on an unusual architecture.
$RoleDir = if ($Mode -eq "master") { "master" } else { "client" }
$rawArch = $env:PROCESSOR_ARCHITECTURE
if ($rawArch -eq "x86" -and $env:PROCESSOR_ARCHITEW6432 -eq "AMD64") { $rawArch = "AMD64" }
$goArch = switch ($rawArch) {
    "AMD64" { "amd64" }
    "ARM64" { "arm64" }
    "ARM"   { "arm64" }
    "x86"   { "386" }
    default { "amd64" }
}
Write-Host "[arch] Detected Windows architecture: $rawArch → agent build: $goArch"

# ─── Self-elevation ───
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Host "[install] Admin rights required — UAC prompt will appear..."
    $tempScript = Join-Path $env:TEMP "pat_install_$(Get-Random).ps1"
    try {
        $scriptContent = Invoke-WebRequest -Uri "$RootUrl/install.ps1" -UseBasicParsing -TimeoutSec 30 | Select-Object -ExpandProperty Content
        Set-Content -Path $tempScript -Value $scriptContent -Encoding UTF8
        $p = Start-Process -FilePath "powershell.exe" -ArgumentList "-ExecutionPolicy","Bypass","-NoProfile","-File","`"$tempScript`"","-Mode",$Mode,"-EngineHost",$EngineHost,"-BaseUrl",$BaseUrl -Verb RunAs -Wait -PassThru
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
Write-Host "  Predict-A-Trade XAUUSD — Installer v1.2.35"
Write-Host "=========================================="
Write-Host ""

# Step 1: Defender exclusions removed to avoid Killav.VDA false positive
# Users can manually add C:\PredictATrade to exclusions if needed
Write-Host "[1/9] Ready"

# Step 2: Create directories
Write-Host "[2/9] Creating installation directory..."
if (-not (Test-Path $InstallDir)) { New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null }
$logsDir = Join-Path $InstallDir "logs"
if (-not (Test-Path $logsDir)) { New-Item -ItemType Directory -Path $logsDir -Force | Out-Null }
Write-Host "  OK: $InstallDir"

# Step 2b: Persist the engine WebSocket URL for this role as a machine-level
# environment variable (the agent reads PAT_SERVER_URL / PAT_DATA_WS_URL).
[Environment]::SetEnvironmentVariable($EngineEnvVar, $EngineWsUrl, "Machine") | Out-Null
Write-Host "  OK: Engine URL ($RoleLabel) = $EngineWsUrl"

# Step 2c: Unique local health port per role so Client + Master can coexist on
# the same machine (both default to 9000 otherwise → bind conflict).
$HealthPort = if ($Mode -eq "master") { "9001" } else { "9000" }
[Environment]::SetEnvironmentVariable("PAT_HEALTH_PORT", $HealthPort, "Machine") | Out-Null
Write-Host "  OK: Local health port = $HealthPort"

# Step 2d: Pin the control-plane API URL. CRITICAL: a stale PAT_API_URL machine
# env var from a previous install can point at live.predictatrade.com/api/v1,
# which (on the edge host) proxies /api/v1 to the Go realtime engine — NOT the
# NestJS control plane. License/device validation then 404s and the EA reports
# "Access Denied | License: PENDING". The dedicated api.predictatrade.com host
# proxies /api/v1 to control correctly, so pin it explicitly to override any
# stale value and make reinstalls deterministic.
$ApiBaseUrl = "https://api.predictatrade.com/api/v1"
[Environment]::SetEnvironmentVariable("PAT_API_URL", $ApiBaseUrl, "Machine") | Out-Null
Write-Host "  OK: Control API URL = $ApiBaseUrl"

# Step 2e: Tell the auto-updater the exact Windows service name to stop/start when
# it swaps the binary. Must match the service registered below (pat-agent-client /
# pat-agent-master) or the update would target the wrong service and never apply.
[Environment]::SetEnvironmentVariable("PAT_SERVICE_NAME", $ServiceName, "Machine") | Out-Null
Write-Host "  OK: Auto-update service name = $ServiceName"

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

# Step 4: Wait for service to stop (no force-kill to avoid AV triggers)
Write-Host "[4/9] Waiting for processes..."
Start-Sleep -Seconds 2
Write-Host "  OK"

# Step 5: Download the role-specific agent binary
Write-Host "[5/9] Downloading $AgentExe..."
$agentPath = Join-Path $InstallDir $AgentExe
try {
    if (Test-Path $agentPath) { Remove-Item $agentPath -Force -ErrorAction SilentlyContinue }
                Invoke-WebRequest -Uri "$BaseUrl/$RoleDir/$goArch/$AgentExe" -OutFile $agentPath -UseBasicParsing -TimeoutSec 120
    Unblock-File -Path $agentPath -ErrorAction SilentlyContinue

    # Check if Defender quarantined the file immediately after download
    if (-not (Test-Path $agentPath) -or (Get-Item $agentPath).Length -lt 1KB) {
        Write-Host "  WARN: Binary appears quarantined by Defender — attempting restore..."
        try {
            # Try to restore from quarantine
            $threat = Get-MpThreatDetection -ErrorAction SilentlyContinue | Where-Object { $_.Resources -like "*$AgentExe*" } | Select-Object -First 1
            if ($threat) {
                Remove-MpThreat -ErrorAction SilentlyContinue
                Start-Sleep -Seconds 2
                # Re-download after restoring
    Invoke-WebRequest -Uri "$BaseUrl/$RoleDir/$goArch/$AgentExe" -OutFile $agentPath -UseBasicParsing -TimeoutSec 120
                Unblock-File -Path $agentPath -ErrorAction SilentlyContinue
                Write-Host "  OK: Restored and re-downloaded"
            }
        } catch {
            Write-Host "  WARN: Could not auto-restore: $_"
            Write-Host "  Manual fix: Windows Security > Protection history > Allow on device"
        }
    }

    $fileSize = (Get-Item $agentPath).Length
    Write-Host "  OK: Downloaded $AgentExe ($([math]::Round($fileSize/1MB, 1)) MB)"
} catch {
    Write-Host "  FATAL: Download failed: $_"
    Read-Host "Press Enter to close"
    exit 1
}

# Step 5b: Stop Windows Defender from blocking the agent. The binary is shipped
# UNSIGNED (no self-signed or CA signature), so Defender quarantines or blocks it on
# download/run. Add a SCOPED exclusion for the agent's own directories so the service
# can start. Only attempted when the installer is elevated (RunAs). This is a dev/test
# stopgap — for production the binary MUST be Authenticode-signed with a real CA cert.
try {
    $principal = [Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()
    if ($principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        $excl = @($InstallDir, (Join-Path $env:ProgramData 'PredictATrade'))
        foreach ($p in $excl) {
            if (Test-Path $p) { Add-MpPreference -ExclusionPath $p -ErrorAction SilentlyContinue }
        }
        Write-Host "  OK: Added Windows Defender exclusions for agent directories"
    } else {
        Write-Host "  WARN: Not running as Administrator — skipped Defender exclusion."
        Write-Host "        If Defender blocks the agent, re-run the installer as Admin or add a"
        Write-Host "        Windows Security exclusion for $InstallDir manually."
    }
} catch {
    Write-Host "  WARN: Could not configure Defender automatically ($_). Allow the agent in"
    Write-Host "        Windows Security > Virus & threat protection > Exclusions."
}

# ── [5b] Prevent Windows Defender / antivirus from blocking the agent ──
# The agent is shipped UNSIGNED, so we stop Defender from quarantining it by
# (1) Unblock-File (already strips the "downloaded from internet" SmartScreen
# trigger) and (2) adding a Defender exclusion for the install directory. This is
# simple and cert-free. Non-fatal: if it fails, the agent still runs.
try {
    Add-MpPreference -ExclusionPath $InstallDir -ErrorAction Stop
    Write-Host "  OK: Added Windows Defender exclusion for $InstallDir"
} catch {
    Write-Host "  WARN: Could not add Defender exclusion (agent may need a manual allow): $_"
}

# Step 6: Acquire NSSM (service manager — wraps the agent as a Windows service).
# Prefer an EXISTING nssm on the device (PATH, cached common copy, or already in
# this role's dir) so we never overwrite/redeploy a working binary. Re-downloading
# or clobbering nssm on a device that already runs a Master Node + Client Agent is
# exactly what previously caused service conflicts when both shared one directory.
Write-Host "[6/9] Acquiring NSSM (service manager)..."
$is64bit = [Environment]::Is64BitOperatingSystem
$nssmArch = if ($is64bit) { "nssm/win64/nssm.exe" } else { "nssm/win32/nssm.exe" }
$nssmDownloaded = $false

function Get-ExistingNssm {
    $cmd = Get-Command nssm.exe -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }
    $common = "C:\ProgramData\PredictATrade\nssm.exe"
    if (Test-Path $common) { return $common }
    return $null
}

$existingNssm = Get-ExistingNssm
if (Test-Path $nssmDest) {
    Write-Host "  OK: Existing nssm.exe found in install dir — reusing (no download)."
    $nssmDownloaded = $true
} elseif ($existingNssm) {
    try {
        Copy-Item -Path $existingNssm -Destination $nssmDest -Force
        Unblock-File -Path $nssmDest -ErrorAction SilentlyContinue
        $nssmDownloaded = $true
        Write-Host "  OK: Reused existing nssm.exe from $existingNssm"
    } catch {
        Write-Host "  WARN: Could not copy existing nssm ($existingNssm): $_"
    }
}
if (-not $nssmDownloaded) {
    try {
        Invoke-WebRequest -Uri "$RootUrl/$nssmArch" -OutFile $nssmDest -UseBasicParsing -TimeoutSec 60
        Unblock-File -Path $nssmDest -ErrorAction SilentlyContinue
        if (Test-Path $nssmDest) {
            $nssmDownloaded = $true
            Write-Host "  OK: Downloaded nssm.exe"
            # Cache a shared copy so the other role / future installs skip the download.
            try { New-Item -ItemType Directory -Path "C:\ProgramData\PredictATrade" -Force -ErrorAction SilentlyContinue | Out-Null
                  Copy-Item -Path $nssmDest -Destination "C:\ProgramData\PredictATrade\nssm.exe" -Force -ErrorAction SilentlyContinue } catch {}
        }
    } catch {
        Write-Host "  WARN: NSSM download failed: $_"
    }
}

# Step 6b: Download supporting scripts
Write-Host "[6b/9] Downloading supporting scripts..."
$supportFiles = @("health-check.ps1", "status.ps1", "notify.ps1")
foreach ($file in $supportFiles) {
    try {
        $dest = Join-Path $InstallDir $file
        Invoke-WebRequest -Uri "$RootUrl/$file" -OutFile $dest -UseBasicParsing -TimeoutSec 30
        Unblock-File -Path $dest -ErrorAction SilentlyContinue
    } catch {
        Write-Host "  WARN: Could not download ${file}: $_"
    }
}

# Step 6c: Download settings.json if not exists (preserve existing)
$settingsPath = Join-Path $InstallDir "settings.json"
if (-not (Test-Path $settingsPath)) {
    try {
        Invoke-WebRequest -Uri "$RootUrl/settings.json" -OutFile $settingsPath -UseBasicParsing -TimeoutSec 30
        Write-Host "  OK: Downloaded settings.json"
    } catch {
        Write-Host "  WARN: Could not download settings.json: $_"
    }
} else {
    Write-Host "  OK: Preserving existing settings.json"
}

# Step 7: Remove old service and create fresh
Write-Host "[7/9] Creating Windows service..."

# Remove old service if it exists
if ($svc) {
    Write-Host "  Removing old service..."
    if ($nssmDownloaded -and (Test-Path $nssmDest)) {
        & $nssmDest stop $ServiceName 2>&1 | Out-Null
        & $nssmDest remove $ServiceName confirm 2>&1 | Out-Null
    }
    sc.exe stop $ServiceName 2>&1 | Out-Null
    Start-Sleep -Seconds 1
    sc.exe delete $ServiceName 2>&1 | Out-Null
    Start-Sleep -Seconds 2
    Write-Host "  OK: Old service removed"
}

$serviceCreated = $false
# Role-specific log file name so the master and client agents are easy to tell
# apart:  Master Node -> master_agent.log,  Client -> agent.log.
$AgentLog = if ($Mode -eq "master") { Join-Path $logsDir "master_agent.log" } else { Join-Path $logsDir "agent.log" }
$stdoutLog = $AgentLog
$stderrLog = $AgentLog

# Method 1: Try NSSM (best — handles crashes, logs, auto-restart)
if (-not $serviceCreated -and $nssmDownloaded -and (Test-Path $nssmDest)) {
    Write-Host "  Trying NSSM service..."
    $installResult = & $nssmDest install $ServiceName $agentPath $AgentArgs 2>&1
    if ($LASTEXITCODE -eq 0 -or $LASTEXITCODE -eq $null) {
        & $nssmDest set $ServiceName AppDirectory $InstallDir 2>&1 | Out-Null
        & $nssmDest set $ServiceName AppStdout $stdoutLog 2>&1 | Out-Null
        & $nssmDest set $ServiceName AppStderr $stderrLog 2>&1 | Out-Null
        & $nssmDest set $ServiceName AppRotateFiles 1 2>&1 | Out-Null
        & $nssmDest set $ServiceName AppRotateOnline 1 2>&1 | Out-Null
        & $nssmDest set $ServiceName AppExit Default Restart 2>&1 | Out-Null
        & $nssmDest set $ServiceName AppExit 0 Exit 2>&1 | Out-Null
        & $nssmDest set $ServiceName AppRestartDelay 5000 2>&1 | Out-Null
        & $nssmDest set $ServiceName DisplayName "Predict-A-Trade XAUUSD Agent" 2>&1 | Out-Null
        & $nssmDest set $ServiceName Description "Predict-A-Trade XAUUSD Windows Agent" 2>&1 | Out-Null
        & $nssmDest set $ServiceName Start SERVICE_AUTO_START 2>&1 | Out-Null
        $serviceCreated = $true
        Write-Host "  OK: NSSM service created"
    } else {
        Write-Host "  WARN: NSSM install failed: $installResult"
    }
}

# Method 2: Try sc.exe (basic Windows service)
if (-not $serviceCreated) {
    Write-Host "  Trying sc.exe service..."
    $scResult = sc.exe create $ServiceName binPath= "`"$agentPath`" $AgentArgs" start= auto 2>&1
    if ($LASTEXITCODE -eq 0) {
        sc.exe description $ServiceName "Predict-A-Trade XAUUSD Windows Agent" 2>&1 | Out-Null
        sc.exe failure $ServiceName reset= 60 actions= restart/5000 2>&1 | Out-Null
        $serviceCreated = $true
        Write-Host "  OK: sc.exe service created"
    } else {
        Write-Host "  WARN: sc.exe failed: $scResult"
    }
}

# Step 8: Start the service
Write-Host "[8/9] Starting service..."
$serviceRunning = $false

if ($serviceCreated) {
    if ($nssmDownloaded -and (Test-Path $nssmDest)) {
        & $nssmDest start $ServiceName 2>&1 | Out-Null
    } else {
        try { Start-Service -Name $ServiceName -ErrorAction SilentlyContinue } catch {}
    }

    # Wait up to 15 seconds for service to start
    Write-Host "  Waiting for service to start..."
    for ($i = 0; $i -lt 15; $i++) {
        Start-Sleep -Seconds 1
        $svcCheck = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
        if ($svcCheck -and $svcCheck.Status -eq "Running") {
            $serviceRunning = $true
            Write-Host "  OK: Service is RUNNING"
            break
        }
    }

    # If service status not Running yet, check health endpoint
    if (-not $serviceRunning) {
        Write-Host "  Service starting — verifying health endpoint..."
        for ($i = 0; $i -lt 5; $i++) {
            Start-Sleep -Seconds 2
            try {
                $healthResp = Invoke-WebRequest -Uri "http://127.0.0.1:$HealthPort/health" -UseBasicParsing -TimeoutSec 3 -ErrorAction Stop
                if ($healthResp.StatusCode -eq 200) {
                    $serviceRunning = $true
                    Write-Host "  OK: Agent is RUNNING (health: HTTP 200)"
                    break
                }
            } catch {}
        }
    }
}

if (-not $serviceRunning) {
    Write-Host "  WARN: Agent not responding — checking logs..."
    if (Test-Path $AgentLog) {
        Write-Host "  --- $($AgentLog | Split-Path -Leaf) ---"
        Get-Content $AgentLog -Tail 15 -ErrorAction SilentlyContinue | ForEach-Object { Write-Host "    $_" }
    } else {
        Write-Host "  No logs — agent may be blocked by antivirus"
        Write-Host "  Manual fix: Windows Security > Virus & threat protection > Exclusions > Add > C:\\PredictATrade"
    }
}

# Step 9: Save version + verify health endpoint
Write-Host "[9/9] Finalizing..."
# Fetch the actual version from the server's version.txt (single source of truth)
try {
    $serverVersion = (Invoke-WebRequest -Uri "$RootUrl/version.txt" -UseBasicParsing -TimeoutSec 10).Content.Trim()
    Write-Host "  Server version: v$serverVersion"
} catch {
    $serverVersion = "1.2.35"
    Write-Host "  WARN: Could not fetch server version — using default v$serverVersion"
}
Set-Content -Path (Join-Path $InstallDir "version.txt") -Value $serverVersion -NoNewline

# Try to verify health endpoint
Start-Sleep -Seconds 2
try {
    $healthResp = Invoke-WebRequest -Uri "http://127.0.0.1:$HealthPort/health" -UseBasicParsing -TimeoutSec 5 -ErrorAction Stop
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
Write-Host "  Role:        $RoleLabel"
Write-Host "  Status:      $(if ($serviceRunning) { 'Running ✓' } else { 'Check logs above' })"
Write-Host "  Install Dir: $InstallDir"
Write-Host "  Health:      http://127.0.0.1:$HealthPort"
Write-Host "  Logs:        $logsDir"
Write-Host ""
$roleName = if ($Mode -eq "master") { "master" } else { "client" }
Write-Host "  To uninstall: irm $RootUrl/uninstall.ps1 | iex   (use: -Mode $roleName)"
Write-Host "  To update:    irm $RootUrl/install-$roleName.ps1 | iex"
Write-Host "=========================================="
Write-Host ""
Read-Host "Press Enter to close"
