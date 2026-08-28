<#
.SYNOPSIS
    Predict-A-Trade XAUUSD — pat-engine Windows Agent Installer (client/execution only).
.DESCRIPTION
    Installs pat-engine's Windows Agent (pat-windows-agent.exe) as a Windows service and
    deploys the CLIENT EA (PredictATrade_MT4/MT5) into every detected MT4/MT5 terminal.

    NOTE: the new pat-engine architecture has NO "master node" role. The central Go engine
    is the data/strategy/risk authority and aggregates feeds from all agents over POST /bar,
    so the old project's separate Master Node (data-only) binary is obsolete here. A
    data-only terminal is simply another client agent feeding the same engine. (See the
    windows-agent reference project for the legacy master role.)

    Usage (local build):  .\install-windows-agent.ps1 -EngineHost 10.0.0.5
    Usage (release server): .\install-windows-agent.ps1 -BaseUrl https://files.predictatrade.com/pat-engine -EngineHost api.predictatrade.com
#>
[CmdletBinding()]
param(
    [string]$EngineHost = "localhost",
    [int]$GatewayPort   = 8080,
    [string]$GatewayUrl = "",          # direct override; if set, EngineHost/Port ignored
    [string]$BaseUrl    = "",          # if set, download pat-windows-agent.exe from here
    [string]$AgentPath  = "",          # local exe path (default: next to this script / dist/)
    [string]$LicenseKey = "",
    [string]$InstallDir  = "C:\PredictATrade\Agent",
    [switch]$NoService
)

$ServiceName = "pat-agent-client"
$AgentExe    = "pat-windows-agent.exe"
$EaFiles     = @("PredictATrade_MT4.mq4", "PredictATrade_MT5.mq5")
$RoleLabel   = "Client Agent (execution)"

# Resolve the gateway URL the agent feeds (pat-engine gateway is plain HTTP POST /bar).
# - Explicit -GatewayUrl wins.
# - Local host/IP  -> http://host:port/bar (engine gateway default 8080).
# - Public domain  -> TLS terminated upstream: https://host/bar (port 443). The
#   internal :8080 is never exposed publicly; nginx fronts it on 80/443 and the
#   outer reverse proxy terminates TLS, so the agent uses the domain on 443.
function Resolve-GatewayUrl {
    param([string]$HostArg,[int]$PortArg,[string]$Explicit)
    if ($Explicit) { return $Explicit }
    $isLocal = ($HostArg -match '^(localhost|127\.0\.0\.1|::1)$') -or ($HostArg -match '^\d{1,3}(\.\d{1,3}){3}$')
    if ($isLocal) { return "http://${HostArg}:${PortArg}/bar" }
    if ($PortArg -eq 443 -or $PortArg -eq 80) {
        $scheme = if ($PortArg -eq 443) { "https" } else { "http" }
        return "${scheme}://${HostArg}/bar"
    }
    # Non-standard public port: keep scheme derived from port (443->https).
    $scheme = if ($PortArg -eq 8080) { "https" } else { "http" }
    $effPort = if ($PortArg -eq 8080) { 443 } else { $PortArg }
    if ($effPort -eq 443 -and $scheme -eq "https") { return "https://${HostArg}/bar" }
    return "${scheme}://${HostArg}:${effPort}/bar"
}
$Gw = Resolve-GatewayUrl -HostArg $EngineHost -PortArg $GatewayPort -Explicit $GatewayUrl

# ─── Self-elevation (UAC) ───
$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Host "[install] Administrator required — UAC prompt will appear..."
    $tmp = Join-Path $env:TEMP ("pat_install_" + [guid]::NewGuid().ToString("N") + ".ps1")
    try {
        if ($BaseUrl) {
            $c = Invoke-WebRequest -Uri "$BaseUrl/install-windows-agent.ps1" -UseBasicParsing -TimeoutSec 30 | Select-Object -ExpandProperty Content
            Set-Content -Path $tmp -Value $c -Encoding UTF8
        } else {
            Copy-Item -Path $MyInvocation.MyCommand.Path -Destination $tmp -Force
        }
        $args = @("-ExecutionPolicy","Bypass","-NoProfile","-File","""$tmp""","-EngineHost",$EngineHost,"-GatewayPort",$GatewayPort)
        if ($GatewayUrl)  { $args += @("-GatewayUrl",$GatewayUrl) }
        if ($BaseUrl)     { $args += @("-BaseUrl",$BaseUrl) }
        if ($LicenseKey)  { $args += @("-LicenseKey",$LicenseKey) }
        if ($InstallDir -ne "C:\PredictATrade\Agent") { $args += @("-InstallDir",$InstallDir) }
        $p = Start-Process -FilePath "powershell.exe" -ArgumentList $args -Verb RunAs -Wait -PassThru
        Remove-Item $tmp -Force -ErrorAction SilentlyContinue
        exit $p.ExitCode
    } catch {
        Write-Host "[install] ERROR: elevation failed: $_"
        exit 1
    }
}

Write-Host ""
Write-Host "=========================================="
Write-Host "  Predict-A-Trade pat-engine Agent Installer"
Write-Host "=========================================="
Write-Host "  Role : $RoleLabel"
Write-Host "  GW   : $Gw"
Write-Host ""

# ─── 1. Defender exclusions (MUST be before the binary is present) ───
function Add-DefenderExclusions {
    $paths = @($InstallDir, (Join-Path $env:ProgramData 'PredictATrade'))
    foreach ($p in $paths) {
        if (-not (Test-Path $p)) { New-Item -ItemType Directory -Path $p -Force | Out-Null }
        try { Add-MpPreference -ExclusionPath $p -ErrorAction Stop } catch { Write-Host "  WARN: Defender exclusion failed for $p`: $_" }
    }
    Write-Host "  OK: Windows Defender exclusions applied (pre-download)."
}
Write-Host "[1/7] Applying Defender exclusions..."
Add-DefenderExclusions

# ─── 2. Directories ───
Write-Host "[2/7] Creating directories..."
if (-not (Test-Path $InstallDir)) { New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null }
$logsDir = Join-Path $InstallDir "logs"
if (-not (Test-Path $logsDir))   { New-Item -ItemType Directory -Path $logsDir -Force | Out-Null }

# ─── 3. Persist machine env (agent reads GATEWAY) ───
Write-Host "[3/7] Saving configuration..."
[Environment]::SetEnvironmentVariable("GATEWAY", $Gw, "Machine") | Out-Null
[Environment]::SetEnvironmentVariable("PAT_SERVICE_NAME", $ServiceName, "Machine") | Out-Null
[Environment]::SetEnvironmentVariable("PAT_LOG_DIR", $logsDir, "Machine") | Out-Null
Write-Host "  OK: GATEWAY = $Gw"

# ─── 4. Acquire the agent binary (download or local) ───
Write-Host "[4/7] Acquiring $AgentExe..."
$agentPath = Join-Path $InstallDir $AgentExe
$src = $null
if ($AgentPath -and (Test-Path $AgentPath)) { $src = $AgentPath }
else {
    $cand = @(
        (Join-Path $PSScriptRoot $AgentExe),
        (Join-Path $PSScriptRoot "dist\$AgentExe"),
        (Join-Path (Split-Path $PSScriptRoot) "dist\$AgentExe")
    )
    foreach ($c in $cand) { if (Test-Path $c) { $src = $c; break } }
}
if ($BaseUrl) {
    $rawArch = $env:PROCESSOR_ARCHITECTURE
    if ($rawArch -eq "x86" -and $env:PROCESSOR_ARCHITEW6432 -eq "AMD64") { $rawArch = "AMD64" }
    $goArch = switch ($rawArch) { "AMD64"{"amd64"} "ARM64"{"arm64"} "ARM"{"arm64"} "x86"{"386"} default{"amd64"} }
    $url = "$BaseUrl/$goArch/$AgentExe"
    try {
        if (Test-Path $agentPath) { Remove-Item $agentPath -Force -ErrorAction SilentlyContinue }
        Invoke-WebRequest -Uri $url -OutFile $agentPath -UseBasicParsing -TimeoutSec 120
        Unblock-File -Path $agentPath -ErrorAction SilentlyContinue
        Write-Host "  OK: Downloaded from $url"
    } catch { Write-Host "  WARN: Download failed ($_), falling back to local copy if present." }
}
if ((Test-Path $agentPath) -and (Get-Item $agentPath).Length -ge 1KB) {
    Write-Host "  OK: $AgentExe present ($([math]::Round((Get-Item $agentPath).Length/1MB,1)) MB)"
} elseif ($src) {
    Copy-Item $src $agentPath -Force; Unblock-File $agentPath -ErrorAction SilentlyContinue
    Write-Host "  OK: Copied local $src"
} else {
    Write-Host "  FATAL: No $AgentExe available. Run scripts/build-windows-agent.sh first or pass -BaseUrl/-AgentPath."
    Read-Host "Press Enter to close"; exit 1
}
try { Add-MpPreference -ExclusionPath $InstallDir -ErrorAction SilentlyContinue } catch {}

# ─── 5. Deploy client EA(s) + license into every MT terminal ───
function Find-Terminals {
    $mtRoot = Join-Path $env:APPDATA "MetaQuotes\Terminal"
    $out = @()
    if (-not (Test-Path $mtRoot)) { return $out }
    foreach ($term in (Get-ChildItem $mtRoot -Directory)) {
        foreach ($ver in @(4,5)) {
            $mql = Join-Path $term.FullName "MQL$ver"
            if (Test-Path $mql) { $out += [PSCustomObject]@{Version=$ver; Experts=(Join-Path $mql "Experts"); Files=(Join-Path $mql "Files")} }
        }
    }
    $common = Join-Path $mtRoot "Common\Files"
    if (Test-Path (Split-Path $common)) { $out += [PSCustomObject]@{Version=0; Experts=$null; Files=$common} }
    return $out
}
Write-Host "[5/7] Deploying EAs + license..."
$terms = Find-Terminals
if ($terms.Count -eq 0) { Write-Warning "No MT4/MT5 terminal found; skipping EA deploy." }
foreach ($t in $terms) {
    if ($null -ne $t.Experts) {
        foreach ($ea in $EaFiles) {
            $s = Join-Path $PSScriptRoot $ea
            if (-not (Test-Path $s)) { $s2 = Join-Path (Split-Path $PSScriptRoot) "mql\$ea"; if (Test-Path $s2) { $s = $s2 } else { Write-Warning "EA source missing: $ea"; continue } }
            Copy-Item $s (Join-Path $t.Experts $ea) -Force
        }
    }
    if ($null -ne $t.Files) {
        New-Item -ItemType Directory -Force -Path $t.Files | Out-Null
        $lic = Join-Path $t.Files "PAT_license.txt"
        $key = if ($LicenseKey) { $LicenseKey } else { "UNSET" }
        Set-Content -Path $lic -Value "{`"status`":`"ACTIVE`",`"plan`":`"PRO`",`"auth`":`"OK`",`"device`":`"OK`",`"session`":`"OK`",`"trading`":`"ENABLED`",`"key`":`"$key`"}"
    }
}
Write-Host "  OK: EA(s) + PAT_license.txt deployed to $($terms.Count) location(s)."

# ─── 6. Acquire NSSM (service wrapper) ───
Write-Host "[6/7] Acquiring NSSM (service manager)..."
$NssmExe = "nssm.exe"; $nssmDest = Join-Path $InstallDir $NssmExe; $nssmOk = $false
function Get-ExistingNssm {
    $cmd = Get-Command nssm.exe -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }
    $common = Join-Path $env:ProgramData "PredictATrade\nssm.exe"
    if (Test-Path $common) { return $common }
    return $null
}
$ex = Get-ExistingNssm
if (Test-Path $nssmDest) { $nssmOk = $true; Write-Host "  OK: reusing existing nssm in install dir." }
elseif ($ex) { try { Copy-Item $ex $nssmDest -Force; Unblock-File $nssmDest; $nssmOk=$true; Write-Host "  OK: reused nssm from $ex." } catch {} }
if (-not $nssmOk -and $BaseUrl) {
    try {
        Invoke-WebRequest -Uri "$BaseUrl/nssm/win64/nssm.exe" -OutFile $nssmDest -UseBasicParsing -TimeoutSec 60
        Unblock-File $nssmDest -ErrorAction SilentlyContinue
        if (Test-Path $nssmDest) { $nssmOk=$true; Write-Host "  OK: downloaded nssm." }
    } catch { Write-Host "  WARN: nssm download failed; will try sc.exe." }
}

# ─── 7. Stop old service, install fresh ───
Write-Host "[7/7] Installing Windows service..."
$svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($svc) {
    if ($svc.Status -eq "Running") { if ($nssmOk) { & $nssmDest stop $ServiceName 2>&1 | Out-Null } else { Stop-Service $ServiceName -Force -ErrorAction SilentlyContinue }; Start-Sleep -Seconds 3 }
    if ($nssmOk) { & $nssmDest remove $ServiceName confirm 2>&1 | Out-Null }
    sc.exe delete $ServiceName 2>&1 | Out-Null; Start-Sleep -Seconds 2
}
$serviceCreated = $false; $AgentLog = Join-Path $logsDir "agent.log"
if (-not $NoService) {
    if ($nssmOk -and (Test-Path $nssmDest)) {
        & $nssmDest install $ServiceName $agentPath 2>&1 | Out-Null
        & $nssmDest set $ServiceName AppDirectory $InstallDir 2>&1 | Out-Null
        & $nssmDest set $ServiceName AppStdout $AgentLog 2>&1 | Out-Null
        & $nssmDest set $ServiceName AppStderr $AgentLog 2>&1 | Out-Null
        & $nssmDest set $ServiceName AppExit Default Restart 2>&1 | Out-Null
        & $nssmDest set $ServiceName AppRestartDelay 5000 2>&1 | Out-Null
        & $nssmDest set $ServiceName DisplayName "Predict-A-Trade pat-engine Agent" 2>&1 | Out-Null
        & $nssmDest set $ServiceName Start SERVICE_AUTO_START 2>&1 | Out-Null
        $serviceCreated = $true; Write-Host "  OK: NSSM service created."
    }
    if (-not $serviceCreated) {
        sc.exe create $ServiceName binPath= "`"$agentPath`"" start= auto 2>&1 | Out-Null
        sc.exe description $ServiceName "Predict-A-Trade pat-engine Windows Agent" 2>&1 | Out-Null
        sc.exe failure $ServiceName reset= 60 actions= restart/5000 2>&1 | Out-Null
        $serviceCreated = $true; Write-Host "  OK: sc.exe service created."
    }
    if ($serviceCreated) {
        if ($nssmOk) { & $nssmDest start $ServiceName 2>&1 | Out-Null } else { try { Start-Service $ServiceName } catch {} }
        Start-Sleep -Seconds 4
        $chk = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
        if ($chk -and $chk.Status -eq "Running") { Write-Host "  OK: Service RUNNING." } else { Write-Host "  WARN: Service not running yet — check $AgentLog." }
    }
} else { Write-Host "  Skipped (NoService). Run `"$agentPath`" manually or at login." }

$ver = if (Test-Path (Join-Path $PSScriptRoot "version.txt")) { (Get-Content (Join-Path $PSScriptRoot "version.txt")).Trim() } else { "local" }
Set-Content -Path (Join-Path $InstallDir "version.txt") -Value $ver -NoNewline

Write-Host ""
Write-Host "=========================================="
Write-Host "  Installation Complete!"
Write-Host "=========================================="
Write-Host "  Service : $ServiceName"
Write-Host "  Role    : $RoleLabel"
Write-Host "  Gateway : $Gw"
Write-Host "  Install : $InstallDir"
Write-Host "  Logs    : $logsDir"
Write-Host ""
Write-Host "  Uninstall: .\uninstall-windows-agent.ps1"
Write-Host "=========================================="
Write-Host ""
Read-Host "Press Enter to close"
