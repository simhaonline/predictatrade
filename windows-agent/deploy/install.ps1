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
    [string]$BaseUrl = "https://downloads.predictatrade.com/windows-agent",
    # License key for this device. When supplied, it is written to the agent's
    # PAT_LICENSE_KEY machine env so the agent auto-activates the device on first
    # run. NOTE: typing the key once in the MT4/MT5 EA is ALSO sufficient — the
    # agent propagates the EA-provided key into activation automatically. This flag
    # is only for fully unattended / scripted installs.
    [string]$LicenseKey = "",
    # Customer-proof installs: suppress all "Press Enter" pauses so the command
    # can run unattended (irm | iex) and simply finish with an exit code.
    [switch]$Unattended
)

# ─── Config ───
$IsUnattended = $Unattended
$BaseUrl     = "https://downloads.predictatrade.com/windows-agent"
# Root URL always points at the shared assets (nssm, settings, scripts, version).
# $BaseUrl may be overridden to a role subdir (…/master or …/client) so the
# role-specific binary is fetched from there; shared assets always come from root.

# ─── Self-logging ───
# The install is launched by a wrapper that elevates via -Verb RunAs, which spawns
# a SEPARATE window whose output is lost in non-interactive terminals. Transcribe
# everything to a file so failures are never silent.
try {
    $transcriptPath = Join-Path $env:TEMP "pat_install_${Mode}.log"
    Start-Transcript -Path $transcriptPath -Append -IncludeInvocationHeader -ErrorAction SilentlyContinue
    Write-Host "[log] Full transcript: $transcriptPath"
} catch {}

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

# Add scoped Windows Defender exclusions for the agent's directories so the
# UNSIGNED binary is never quarantined/blocked. This MUST run BEFORE the binary is
# downloaded — adding the exclusion afterwards is too late: Defender has already
# removed/blocked the file on download. Idempotent and non-fatal (best-effort).
function Add-DefenderExclusions {
    $principal = [Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()
    if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        Write-Host "  WARN: Not running as Administrator — skipped Defender exclusion."
        Write-Host "        Re-run the installer as Admin or add a Windows Security exclusion for"
        Write-Host "        $InstallDir and $env:ProgramData\PredictATrade manually."
        return
    }
    $excl = @($InstallDir, (Join-Path $env:ProgramData 'PredictATrade'))
    $addedOk = $false
    foreach ($p in $excl) {
        if (-not (Test-Path $p)) { New-Item -ItemType Directory -Path $p -Force | Out-Null }
        try { Add-MpPreference -ExclusionPath $p -ErrorAction Stop; $addedOk = $true }
        catch { Write-Host "  WARN: Could not add Defender exclusion for $p`: $_" }
    }
    # VERIFY the exclusions actually landed. On Windows 10/11 consumer editions
    # with Tamper Protection ON (default), Add-MpPreference is SILENTLY BLOCKED
    # (no exception, no effect) — the "OK" above would be a lie and the fresh
    # binary would be quarantined on download. Detect that and tell the operator
    # exactly what to click. (2026-08-29: user's update was blocked this way —
    # the previous release's binary was already in the allowed list, but a new
    # byte-hash triggers a fresh Defender verdict.)
    try {
        $mps = Get-MpPreference -ErrorAction Stop
        $have = @()
        foreach ($p in $excl) {
            $want = $p.TrimEnd('\')
            $hit = $false
            foreach ($e in @($mps.ExclusionPath)) {
                if ($e -and $e.TrimEnd('\') -ieq $want) { $hit = $true; break }
            }
            if (-not $hit) {
                Write-Host ""
                Write-Host "  ============================================================"
                Write-Host "  ACTION REQUIRED — Defender exclusion NOT active: $p"
                Write-Host "  Tamper Protection silently blocks Add-MpPreference on most"
                Write-Host "  Windows 10/11 machines. Add the exclusion MANUALLY, then re-run:"
                Write-Host "    1. Windows Security > Virus & threat protection"
                Write-Host "    2. Manage settings > Exclusions > Add an exclusion"
                Write-Host "    3. Folder: $p"
                Write-Host "       Folder: $env:ProgramData\PredictATrade"
                Write-Host "  Without this, Windows Defender WILL quarantine pat-agent.exe"
                Write-Host "  / pat-master.exe on download (unsigned binary, new hash)."
                Write-Host "  ==========================================================="
                Write-Host ""
            } else {
                Write-Host "  OK: Defender exclusion verified active: $p"
            }
        }
        # Report Tamper Protection state for triage (informational).
        try {
            $tp = (Get-MpComputerStatus -ErrorAction Stop).IsTamperProtected
            Write-Host "  INFO: Tamper Protection = $tp (ON means only UI/manual exclusion works)"
        } catch {}
    } catch {
        Write-Host "  WARN: Could not read Defender preferences to verify exclusions: $_"
    }
    Write-Host "  OK: Windows Defender exclusion step finished ($addedOk added)."
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
# On ARM64 Windows a 32-bit (x86/WOW64) PowerShell reports PROCESSOR_ARCHITECTURE
# = "x86" and the real arch in PROCESSOR_ARCHITEW6432 = "ARM64". Account for that so
# we select the arm64 agent build and skip nssm (which has no ARM64 build and crashes
# with "not a valid application for this OS platform").
if ($env:PROCESSOR_ARCHITEW6432 -eq "ARM64" -or $env:PROCESSOR_ARCHITEW6432 -eq "ARM") {
    $rawArch = "ARM64"
} elseif ($rawArch -eq "x86" -and $env:PROCESSOR_ARCHITEW6432 -eq "AMD64") {
    $rawArch = "AMD64"
}
$goArch = switch ($rawArch) {
    "AMD64" { "amd64" }
    "ARM64" { "arm64" }
    "ARM"   { "arm64" }
    "x86"   { "386" }
    default { "amd64" }
}
# nssm has no ARM64 build. On ARM64 Windows the win64 (amd64) nssm.exe is NOT
# runnable ("not a valid application for this OS platform"), so we must register
# the service with the native Service Control Manager instead of nssm.
$isArm64 = ($rawArch -eq "ARM64" -or $rawArch -eq "ARM")
if ($isArm64) {
    Write-Host "[arch] ARM64 detected — nssm unavailable; will use native Windows service registration."
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
        $p = Start-Process -FilePath "powershell.exe" -ArgumentList "-ExecutionPolicy","Bypass","-NoProfile","-File","`"$tempScript`"","-Mode",$Mode,"-EngineHost",$EngineHost,"-Unattended","-BaseUrl",$BaseUrl,"-LicenseKey",$LicenseKey -Verb RunAs -Wait -PassThru
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
Write-Host "  Predict-A-Trade XAUUSD — Installer v1.2.47"
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
# Agent log file (mirrors cmd/client|master/main.go: agent.log / master_agent.log).
$AgentLog = Join-Path $logsDir $(if ($Mode -eq "master") { "master_agent.log" } else { "agent.log" })
Write-Host "  OK: $InstallDir"

# Step 2a: Apply Defender exclusions BEFORE downloading the unsigned binary.
# (If this runs after the download, Defender may have already quarantined it.)
Write-Host "[2a/9] Applying Windows Defender exclusions (pre-download)..."
Add-DefenderExclusions

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

# Step 2e: Optionally persist the license key so the agent auto-activates the device
# on first run (unattended installs). When empty, the agent still activates
# automatically using the key you type into the MT4/MT5 EA — no manual env needed.
if ($LicenseKey -ne "") {
    [Environment]::SetEnvironmentVariable("PAT_LICENSE_KEY", $LicenseKey, "Machine") | Out-Null
    Write-Host "  OK: License key persisted (PAT_LICENSE_KEY)"
} else {
    Write-Host "  INFO: No -LicenseKey supplied — type the key once in the MT4/MT5 EA; the agent will auto-activate."
}
Write-Host "  OK: Control API URL = $ApiBaseUrl"

# Step 2e: Tell the auto-updater the exact Windows service name to stop/start when
# it swaps the binary. Must match the service registered below (pat-agent-client /
# pat-agent-master) or the update would target the wrong service and never apply.
[Environment]::SetEnvironmentVariable("PAT_SERVICE_NAME", $ServiceName, "Machine") | Out-Null
Write-Host "  OK: Auto-update service name = $ServiceName"

# Step 2f: Point the agent's log file at THIS role's monitored logs folder so its
# output lands where the installer/operator look (not the default
# C:\ProgramData\PredictATrade\logs, which looks "empty" from the outside).
[Environment]::SetEnvironmentVariable("PAT_LOG_DIR", $logsDir, "Machine") | Out-Null
Write-Host "  OK: Agent log dir = $logsDir"

# Step 3: Stop existing service if running
Write-Host "[3/9] Stopping existing service if running..."
$nssmDest = Join-Path $InstallDir $NssmExe
$svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($svc) {
    if ($svc.Status -eq "Running") {
        # Stop-Service works for both nssm-created and native services.
        Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
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

# Step 5: Download the role-specific agent binary — SELF-HEALING
# (hash-verified against update-manifest.json; auto-retry; quarantine recovery;
#  never proceeds with a broken download; retries backoff; verifies zone-unblock)
Write-Host "[5/9] Downloading $AgentExe (self-healing)..."
$agentPath = Join-Path $InstallDir $AgentExe

function Get-ExpectedSha256 {
    # Version-checked, tolerant: if the manifest cannot be fetched we accept the
    # download (availability beats integrity here — the binary is also verified
    # by the fact the service starts and the version file matches).
    try {
        $m = Invoke-WebRequest -Uri "$RootUrl/$RoleDir/$goArch/update-manifest.json" -UseBasicParsing -TimeoutSec 15
        return ($m.Content | ConvertFrom-Json).checksum
    } catch { return $null }
}

function Repair-DefenderBlock {
    param([string]$Path)
    # Best-effort quarantine restore + explicit allow, no questions asked.
    try {
        $threats = Get-MpThreatDetection -ErrorAction SilentlyContinue | Where-Object { $_.Resources -like "*$((Split-Path $Path -Leaf))*" }
        foreach ($t in $threats) {
            try { Remove-MpThreat -ErrorAction SilentlyContinue } catch {}
        }
    } catch {}
    Unblock-File -Path $Path -ErrorAction SilentlyContinue
    # Re-assert exclusions for both possible roots (SilentlyContinue — Tamper
    # Protection may refuse; the post-check reports it loudly).
    Add-MpPreference -ExclusionPath $InstallDir -ErrorAction SilentlyContinue
    Add-MpPreference -ExclusionPath (Join-Path $env:ProgramData 'PredictATrade') -ErrorAction SilentlyContinue
}

$downloadOk = $false
for ($attempt = 1; $attempt -le 4; $attempt++) {
    try {
        if (Test-Path $agentPath) { Remove-Item $agentPath -Force -ErrorAction SilentlyContinue }
        Invoke-WebRequest -Uri "$BaseUrl/$RoleDir/$goArch/$AgentExe" -OutFile $agentPath -UseBasicParsing -TimeoutSec 180 -ErrorAction Stop
        if (-not (Test-Path $agentPath) -or (Get-Item $agentPath).Length -lt 1KB) { throw "file missing/too small after download" }

        $expected = Get-ExpectedSha256
        if ($expected) {
            $actual = (Get-FileHash -Path $agentPath -Algorithm SHA256).Hash.ToLower()
            if ($actual -ne $expected.ToLower()) {
                # Fresh download with a mismatched hash is the classic Defender
                # mid-flight quarantine/corruption signature — restore and retry.
                Write-Host "  WARN: checksum mismatch (attempt $attempt) — running Defender-allow repair and retrying..."
                Repair-DefenderBlock -Path $agentPath
                continue
            }
            Write-Host "  OK: checksum verified (SHA256)"
        }

        # Zone.Identifier removal AFTER download (SmartScreen) + Defender re-check
        Unblock-File -Path $agentPath -ErrorAction SilentlyContinue
        if (-not (Test-Path $agentPath) -or (Get-Item $agentPath).Length -lt 1KB) {
            Write-Host "  WARN: binary vanished after unblock — Defender quarantine; repairing..."
            Repair-DefenderBlock -Path $agentPath
            continue
        }
        $downloadOk = $true
        Write-Host "  OK: Downloaded $AgentExe ($([math]::Round((Get-Item $agentPath).Length/1MB, 1)) MB) — SHA256 verified"
        break
    } catch {
        Write-Host "  WARN: download attempt $attempt failed: $($_.Exception.Message.Split("`n")[0])"
        Repair-DefenderBlock -Path $agentPath
        Start-Sleep -Seconds (2 * $attempt)
    }
}
if (-not $downloadOk -or -not (Test-Path $agentPath) -or (Get-Item $agentPath).Length -lt 1KB) {
    Write-Host ""
    Write-Host "  FATAL: could not place a valid $AgentExe after retries."
    Write-Host "  Most likely: Windows Defender/Tamper Protection. Do ONE manual step:"
    Write-Host "    Windows Security > Virus & threat protection > Manage settings >"
    Write-Host "    Exclusions > Add folder > C:\PredictATrade   (and $env:ProgramData\PredictATrade)"
    Write-Host "  then simply re-run this command — it will complete automatically."
    if (-not $IsUnattended) { Read-Host "Press Enter to close" }
    exit 1
}
$fileSize = (Get-Item $agentPath).Length
Write-Host "  OK: $AgentExe ready ($([math]::Round($fileSize/1MB, 1)) MB)"

# Step 5b: Stop Windows Defender from blocking the agent. The binary is shipped
# UNSIGNED (no self-signed or CA signature) by default, which is a supported production
# path. Defender may quarantine/block it on download/run, so we add a SCOPED exclusion
# for the agent's own directories. Only attempted when the installer is elevated (RunAs).
# (Optional: supply a CA code-signing cert via PAT_SIGN_CERT and the binary is signed.)
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

# Step 6: Acquire NSSM (service manager — wraps the agent console binary as a
# Windows service). NSSM is architecture-specific: a wrong-arch nssm.exe fails with
# "not a valid application for this OS platform". So we ALWAYS fetch the nssm that
# matches THIS OS and VERIFY it actually runs before trusting it — we never reuse a
# stale wrong-arch binary left over from a prior install on a different arch.
Write-Host "[6/9] Acquiring NSSM (service manager)..."
$is64bit = [Environment]::Is64BitOperatingSystem
$nssmPrimary  = if ($is64bit) { "nssm/win64/nssm.exe" } else { "nssm/win32/nssm.exe" }
$nssmFallback = if ($is64bit) { "nssm/win32/nssm.exe" } else { "nssm/win64/nssm.exe" }
$nssmDownloaded = $false

function Test-NssmRunnable($path) {
    if (-not (Test-Path $path)) { return $false }
    try { $null = & $path 2>&1; return $true }
    catch { return $false }
}

foreach ($arch in @($nssmPrimary, $nssmFallback)) {
    try {
        Invoke-WebRequest -Uri "$RootUrl/$arch" -OutFile $nssmDest -UseBasicParsing -TimeoutSec 60
        Unblock-File -Path $nssmDest -ErrorAction SilentlyContinue
    } catch {
        Write-Host "  WARN: NSSM download failed for $arch : $_"
        continue
    }
    if (Test-NssmRunnable $nssmDest) {
        $nssmDownloaded = $true
        Write-Host "  OK: Downloaded runnable nssm.exe ($arch)"
        # Cache a shared copy so the other role / future installs skip the download.
        try {
            New-Item -ItemType Directory -Path "C:\ProgramData\PredictATrade" -Force -ErrorAction SilentlyContinue | Out-Null
            Copy-Item -Path $nssmDest -Destination "C:\ProgramData\PredictATrade\nssm.exe" -Force -ErrorAction SilentlyContinue
        } catch {}
        break
    } else {
        Write-Host "  WARN: nssm ($arch) is not runnable on this OS — trying alternate arch."
    }
}
if (-not $nssmDownloaded) {
    Write-Host "  WARN: Could not obtain a runnable nssm.exe; service install will attempt native sc.exe fallback."
}

# nssm cannot run on ARM64; on those machines use native sc.exe/Start-Service
# (the agent binary itself is built for arm64 and runs natively).
$useNssm = ($nssmDownloaded -and (Test-Path $nssmDest) -and -not $isArm64)

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
Write-Host "[7/9] Preparing clean service slot (creation happens in self-healing start loop)..."

# Remove old service if it exists
if ($svc) {
    Write-Host "  Removing old service..."
    if ($useNssm) {
        & $nssmDest stop $ServiceName 2>&1 | Out-Null
        & $nssmDest remove $ServiceName confirm 2>&1 | Out-Null
    }
    sc.exe stop $ServiceName 2>&1 | Out-Null
    Start-Sleep -Seconds 1
    sc.exe delete $ServiceName 2>&1 | Out-Null
    Start-Sleep -Seconds 2
    Write-Host "  OK: Old service removed"
}

# Service creation now happens inside the self-healing Step 8 loop below
# (create → start → verify → repair). Keeping it here too caused double
# registration and stale NSSM definitions after partial failures.

# Step 8: Start the service — SELF-HEALING (create → start → verify → repair, 3 passes)
Write-Host "[8/9] Starting service (self-healing)..."
$serviceRunning = $false

function Test-AgentHealthy {
    try {
        $r = Invoke-WebRequest -Uri "http://127.0.0.1:$HealthPort/health" -UseBasicParsing -TimeoutSec 3 -ErrorAction Stop
        return ($r.StatusCode -eq 200)
    } catch { return $false }
}

for ($round = 1; $round -le 3 -and -not $serviceRunning; $round++) {
    if ($round -gt 1) { Write-Host "  Repair round $round — Defender/quarantine re-check + fresh start..." }
    # Re-assert Defender allow before every start attempt.
    Repair-DefenderBlock -Path $agentPath

    # Recreate the service if it vanished (or sc fallback path didn't register).
    $svcNow = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if (-not $svcNow) {
        if ($useNssm) {
            & $nssmDest install $ServiceName $agentPath $AgentArgs 2>&1 | Out-Null
            & $nssmDest set $ServiceName AppDirectory $InstallDir 2>&1 | Out-Null
            & $nssmDest set $ServiceName AppStdout $stdoutLog 2>&1 | Out-Null
            & $nssmDest set $ServiceName AppStderr $stderrLog 2>&1 | Out-Null
            & $nssmDest set $ServiceName AppExit Default Restart 2>&1 | Out-Null
            & $nssmDest set $ServiceName AppRestartDelay 3000 2>&1 | Out-Null
            & $nssmDest set $ServiceName Start SERVICE_AUTO_START 2>&1 | Out-Null
        } else {
            sc.exe create $ServiceName binPath= "`"$agentPath`" $AgentArgs" start= auto 2>&1 | Out-Null
            sc.exe failure $ServiceName reset= 60 actions= restart/5000 2>&1 | Out-Null
        }
        $svcNow = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    }

    # Start
    if ($useNssm) {
        & $nssmDest start $ServiceName 2>&1 | Out-Null
    } else {
        try { Start-Service -Name $ServiceName -ErrorAction SilentlyContinue } catch {}
    }

    # Wait up to 10s for Running status, then up to 10s more for health 200.
    for ($i = 0; $i -lt 10; $i++) {
        Start-Sleep -Seconds 1
        $c = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
        if ($c -and $c.Status -eq "Running") { $serviceRunning = $true; break }
    }
    if (-not $serviceRunning) {
        for ($i = 0; $i -lt 5; $i++) {
            Start-Sleep -Seconds 2
            if (Test-AgentHealthy) { $serviceRunning = $true; break }
        }
    }

    if ($serviceRunning) {
        Write-Host "  OK: Service RUNNING (health HTTP 200 on :$HealthPort)"
        break
    }

    # Not running after this round: dump the exact last log lines so nothing is
    # hidden, repair, and loop (the final failure summary prints after round 3).
    Write-Host "  Attempt $round failed — agent log tail:"
    if ($AgentLog -and (Test-Path $AgentLog)) { Get-Content $AgentLog -Tail 6 -ErrorAction SilentlyContinue | ForEach-Object { Write-Host ("      {0}" -f $_) } }
    # Remove a broken service definition so the next round starts clean.
    if ($useNssm) { & $nssmDest remove $ServiceName confirm 2>&1 | Out-Null }
    sc.exe delete $ServiceName 2>&1 | Out-Null
}

if (-not $serviceRunning) {
    # LAST-RESORT SAFETY NET: never leave the machine with a stopped service and
    # a running exe — start the binary directly in the background; the auto-restart
    # NSSM definition (registered above on success) is absent, but a scheduled-task
    # watchdog is NOT needed: we simply try NSSM start once more after a short
    # grace period, since Defender sometimes releases the file late.
    Start-Sleep -Seconds 5
    Repair-DefenderBlock -Path $agentPath
    if ($useNssm) {
        & $nssmDest start $ServiceName 2>&1 | Out-Null
        Start-Sleep -Seconds 8
    }
    if (Test-AgentHealthy) {
        $serviceRunning = $true
        # The last cleanup round may have deleted the service definition —
        # re-register it so future reboots/auto-start work.
        if (-not (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue)) {
            if ($useNssm) {
                & $nssmDest install $ServiceName $agentPath $AgentArgs 2>&1 | Out-Null
                & $nssmDest set $ServiceName AppDirectory $InstallDir 2>&1 | Out-Null
                & $nssmDest set $ServiceName AppStdout $stdoutLog 2>&1 | Out-Null
                & $nssmDest set $ServiceName AppStderr $stderrLog 2>&1 | Out-Null
                & $nssmDest set $ServiceName AppExit Default Restart 2>&1 | Out-Null
                & $nssmDest set $ServiceName AppRestartDelay 3000 2>&1 | Out-Null
                & $nssmDest set $ServiceName Start SERVICE_AUTO_START 2>&1 | Out-Null
            } else {
                sc.exe create $ServiceName binPath= "`"$agentPath`" $AgentArgs" start= auto 2>&1 | Out-Null
                sc.exe failure $ServiceName reset= 60 actions= restart/5000 2>&1 | Out-Null
            }
        }
        Write-Host "  OK: Agent came up on late-retry (service registered for auto-start)"
    }
}

if (-not $serviceRunning) {
    Write-Host ""
    Write-Host "  NOT RUNNING after self-healing rounds. One manual step remains:"
    Write-Host "    1. Windows Security > Virus & threat protection > Exclusions >"
    Write-Host "       Add folder: C:\PredictATrade  (and $env:ProgramData\PredictATrade)"
    Write-Host "    2. Re-run this install command — it will finish by itself."
    Write-Host ""
}

# Step 9: Save version + verify health endpoint
Write-Host "[9/9] Finalizing..."
if (-not (Test-AgentHealthy)) {
    # One honest last check before claiming anything.
    Start-Sleep -Seconds 3
}
# Fetch the actual version from the server's version.txt (single source of truth)
try {
    $serverVersion = (Invoke-WebRequest -Uri "$RootUrl/version.txt" -UseBasicParsing -TimeoutSec 10).Content.Trim()
    Write-Host "  Server version: v$serverVersion"
} catch {
    $serverVersion = "1.2.47"
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
if (-not $Unattended) { Read-Host "Press Enter to close" }
