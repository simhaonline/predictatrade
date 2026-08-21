<#
.SYNOPSIS
    Predict-A-Trade XAUUSD — Windows Agent Installer (with versioning & auto-update)
.DESCRIPTION
    Handles fresh install AND updates seamlessly:
    - Fresh install: downloads everything, creates service, starts it
    - Update: stops running service, downloads new files, restarts service
    - Reinstall: same as update when version is the same

    The subscriber just runs the same command every time:
      irm https://downloads.predictatrade.com/windows-agent/install.ps1 | iex

    Self-elevates to Administrator (UAC prompt appears — user clicks Yes).
    Preserves settings.json on update (does NOT overwrite user's credentials).
#>

# ─── Configuration ───
$BaseUrl       = "https://downloads.predictatrade.com/windows-agent"
$InstallDir    = "C:\Program Files\PredictATrade\XAUUSD"
$ServiceName   = "PredictATradeXAUUSD"
$EventSource   = "PredictATradeXAUUSD"
$TaskName      = "PredictATradeHealthCheck"
$AgentExe      = "agent.exe"
$NssmExe       = "nssm.exe"

# ─── Helper: Write to Event Log ───
function Write-PATEventLog {
    param([string]$Message, [string]$Level = "Information", [int]$EventId = 100)
    try {
        if (-not [System.Diagnostics.EventLog]::SourceExists($EventSource)) {
            [System.Diagnostics.EventLog]::CreateEventSource($EventSource, "Application")
        }
        $log = New-Object System.Diagnostics.EventLog("Application")
        $log.Source = $EventSource
        $entryType = switch ($Level) {
            "Error"   { [System.Diagnostics.EventLogEntryType]::Error }
            "Warning" { [System.Diagnostics.EventLogEntryType]::Warning }
            default   { [System.Diagnostics.EventLogEntryType]::Information }
        }
        $log.WriteEntry($Message, $entryType, $EventId)
    } catch {
        Write-Host "[EventLog fallback] $Message"
    }
}

# ─── Helper: Stop the running service and wait for file lock release ───
function Stop-PATService {
    param([string]$NssmPath)
    $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if (-not $svc) { return }

    if ($svc.Status -eq "Running") {
        Write-Host "  Stopping service..."
        if (Test-Path $NssmPath) {
            & $NssmPath stop $ServiceName 2>&1 | Out-Null
        } else {
            Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
        }

        # Wait for the service to actually stop (max 15 seconds)
        $waited = 0
        while ($waited -lt 15) {
            Start-Sleep -Seconds 1
            $waited++
            $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
            if ($svc -and $svc.Status -eq "Stopped") { break }
        }
    }

    # Extra safety: kill the agent.exe process directly if still running
    $proc = Get-Process -Name "agent" -ErrorAction SilentlyContinue
    if ($proc) {
        Write-Host "  Force-killing agent.exe process..."
        $proc | Stop-Process -Force -ErrorAction SilentlyContinue
        Start-Sleep -Seconds 2
    }

    # Wait for file lock to release (max 5 more seconds)
    $agentPath = Join-Path $InstallDir $AgentExe
    if (Test-Path $agentPath) {
        $lockWaited = 0
        while ($lockWaited -lt 5) {
            try {
                $stream = [System.IO.File]::Open($agentPath, 'Open', 'Read', 'None')
                $stream.Close()
                break  # File is not locked
            } catch {
                Start-Sleep -Seconds 1
                $lockWaited++
            }
        }
    }
}

# ─── Self-elevation: re-download and re-execute as admin ───
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

if (-not $isAdmin) {
    Write-Host "[install] Administrator rights required."
    Write-Host "[install] A UAC prompt will appear — please click Yes to accept elevation."

    $remoteScript = "$BaseUrl/install.ps1"
    try {
        $scriptContent = Invoke-WebRequest -Uri $remoteScript -UseBasicParsing -TimeoutSec 30 | Select-Object -ExpandProperty Content
    } catch {
        Write-Host "[install] FATAL: Failed to download install.ps1 from $remoteScript"
        Write-Host "[install] Error: $_"
        Write-Host "[install] Please open PowerShell as Administrator and run the command again."
        exit 1
    }

    $tempScript = Join-Path $env:TEMP "pat_install_$(Get-Random).ps1"
    Set-Content -Path $tempScript -Value $scriptContent -Encoding UTF8

    try {
        $process = Start-Process -FilePath "powershell.exe" `
            -ArgumentList "-ExecutionPolicy", "Bypass", "-NoProfile", "-File", "`"$tempScript`"" `
            -Verb RunAs `
            -Wait `
            -PassThru
        $exitCode = $process.ExitCode
    } catch {
        Write-Host "[install] Elevation was declined or failed: $_"
        Write-Host "[install] Please open PowerShell as Administrator and run:"
        Write-Host "  irm $remoteScript | iex"
        exit 1
    } finally {
        Remove-Item $tempScript -Force -ErrorAction SilentlyContinue
    }
    exit $exitCode
}

# ─═══════════════════════════════════════════════════════════
# Now running as Administrator
# ─═══════════════════════════════════════════════════════════

Write-Host ""
Write-Host "=========================================="
Write-Host "  Predict-A-Trade XAUUSD — Installer"
Write-Host "=========================================="
Write-Host ""

# ─── Step 0: Check server version & detect update vs fresh install ───
Write-Host "[install] Checking available version..."
$serverVersion = ""
try {
    $serverVersion = (Invoke-WebRequest -Uri "$BaseUrl/version.txt" -UseBasicParsing -TimeoutSec 15).Content.Trim()
    Write-Host "  Server version: v$serverVersion"
} catch {
    Write-Host "  WARN: Could not fetch version.txt — proceeding without version check"
    $serverVersion = "unknown"
}

$installedVersionFile = Join-Path $InstallDir "version.txt"
$previousVersion = ""
$isUpdate = $false
if (Test-Path $installedVersionFile) {
    $previousVersion = (Get-Content $installedVersionFile -Raw).Trim()
    $isUpdate = $true
    if ($serverVersion -ne "unknown" -and $serverVersion -eq $previousVersion) {
        Write-Host "  Installed version: v$previousVersion (same — will reinstall)"
    } elseif ($serverVersion -ne "unknown") {
        Write-Host "  Installed version: v$previousVersion -> updating to v$serverVersion"
    } else {
        Write-Host "  Installed version: v$previousVersion (reinstalling)"
    }
} else {
    Write-Host "  No previous installation found — fresh install"
}

# ─── Step 1: Create installation directory ───
Write-Host "[install] Ensuring installation directory: $InstallDir"
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}
$logsDir = Join-Path $InstallDir "logs"
if (-not (Test-Path $logsDir)) {
    New-Item -ItemType Directory -Path $logsDir -Force | Out-Null
}

# ─── Step 2: Detect OS architecture for NSSM ───
$is64bit = [Environment]::Is64BitOperatingSystem
$nssmArchPath = if ($is64bit) { "nssm/win64/nssm.exe" } else { "nssm/win32/nssm.exe" }
Write-Host "[install] OS Architecture: $(if ($is64bit) { '64-bit' } else { '32-bit' })"

# ─── Step 3: STOP the running service BEFORE downloading (release file locks) ───
$nssmDest = Join-Path $InstallDir $NssmExe
if ($isUpdate) {
    Write-Host "[install] Stopping existing service to release file locks..."
    Stop-PATService -NssmPath $nssmDest
    Write-Host "  OK: Service stopped, file locks released"
}

# ─── Step 4: Create Event Log source ───
Write-Host "[install] Creating Event Log source '$EventSource'..."
Write-PATEventLog -Message "Predict-A-Trade XAUUSD installation started$(if ($isUpdate) { " (update from v$previousVersion)" })" -EventId 1

# ─── Step 5: Download all files (agent.exe is no longer locked) ───
$FilesToDownload = @(
    @{ Name = "agent.exe";        Dest = "agent.exe";        Overwrite = $true  }
    @{ Name = "notify.ps1";        Dest = "notify.ps1";        Overwrite = $true  }
    @{ Name = "health-check.ps1"; Dest = "health-check.ps1"; Overwrite = $true  }
    @{ Name = "status.ps1";       Dest = "status.ps1";        Overwrite = $true  }
    @{ Name = "settings.json";    Dest = "settings.json";    Overwrite = $false }
)

$nssmDownloaded = $false

foreach ($file in $FilesToDownload) {
    $url  = "$BaseUrl/$($file.Name)"
    $dest = Join-Path $InstallDir $file.Dest

    # Preserve settings.json on update (don't overwrite user's configured credentials)
    if (-not $file.Overwrite -and (Test-Path $dest)) {
        Write-Host "[install] Preserving existing $($file.Name) (user configuration kept)"
        continue
    }

    Write-Host "[install] Downloading $($file.Name)..."
    try {
        # If the file already exists, delete it first (avoids "file in use" on retry)
        if (Test-Path $dest) {
            Remove-Item $dest -Force -ErrorAction SilentlyContinue
        }
        Invoke-WebRequest -Uri $url -OutFile $dest -UseBasicParsing -TimeoutSec 120
        Write-Host "  OK: Downloaded $($file.Name)"
    } catch {
        Write-Host "  FAIL: Could not download $($file.Name): $_"
        Write-PATEventLog -Message "install.ps1: Failed to download $($file.Name): $_" -Level "Error" -EventId 2
        if ($file.Name -eq "agent.exe") {
            Write-Host "[install] FATAL: Cannot install without agent.exe — aborting"
            Write-PATEventLog -Message "install.ps1: FATAL - agent.exe download failed, aborting" -Level "Error" -EventId 3
            Write-Host ""
            Read-Host "Press Enter to close"
            exit 1
        }
    }
}

# Download NSSM (architecture-specific)
Write-Host "[install] Downloading NSSM ($nssmArchPath)..."
$nssmUrl = "$BaseUrl/$nssmArchPath"
try {
    if (Test-Path $nssmDest) { Remove-Item $nssmDest -Force -ErrorAction SilentlyContinue }
    Invoke-WebRequest -Uri $nssmUrl -OutFile $nssmDest -UseBasicParsing -TimeoutSec 60
    if (Test-Path $nssmDest) {
        $nssmDownloaded = $true
        Write-Host "  OK: Downloaded nssm.exe ($(if ($is64bit) { '64-bit' } else { '32-bit' }))"
    }
} catch {
    Write-Host "  FAIL: Could not download NSSM: $_"
    Write-PATEventLog -Message "install.ps1: NSSM download failed: $_" -Level "Warning" -EventId 4
}

# ─── Step 6: Write version file ───
if ($serverVersion -ne "unknown") {
    Set-Content -Path $installedVersionFile -Value $serverVersion -Encoding UTF8
    Write-Host "[install] Wrote version.txt (v$serverVersion)"
}

# ─── Step 7: Remove old service and install fresh ───
Write-Host "[install] Registering service..."
$existingSvc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($existingSvc) {
    try {
        if ($nssmDownloaded) {
            & $nssmDest remove $ServiceName confirm 2>&1 | Out-Null
        } else {
            sc.exe delete $ServiceName 2>&1 | Out-Null
        }
        Start-Sleep -Seconds 1
    } catch {
        sc.exe delete $ServiceName 2>&1 | Out-Null
    }
}

$agentPath = Join-Path $InstallDir $AgentExe

if ($nssmDownloaded -and (Test-Path $nssmDest)) {
    & $nssmDest install $ServiceName $agentPath 2>&1 | Out-Null
    & $nssmDest set $ServiceName AppDirectory $InstallDir 2>&1 | Out-Null
    & $nssmDest set $ServiceName AppStdout (Join-Path $logsDir "stdout.log") 2>&1 | Out-Null
    & $nssmDest set $ServiceName AppStderr (Join-Path $logsDir "stderr.log") 2>&1 | Out-Null
    & $nssmDest set $ServiceName AppRotateFiles 1 2>&1 | Out-Null
    & $nssmDest set $ServiceName AppRotateOnline 1 2>&1 | Out-Null
    & $nssmDest set $ServiceName AppNoConsole 1 2>&1 | Out-Null
    # ─── NSSM Auto-Restart Configuration ───
    # AppExit Default Restart: auto-restart on ANY crash (unhandled exit codes)
    # AppExit 0 Exit: do NOT restart on clean manual stop (exit code 0)
    # AppRestartDelay 5000: wait 5 seconds before restarting
    # The health-check.ps1 (runs every 1 min) handles crash detection,
    # Windows popup alerts, and external notifications (Telegram/Discord/Email).
    & $nssmDest set $ServiceName AppExit Default Restart 2>&1 | Out-Null
    & $nssmDest set $ServiceName AppExit 0 Exit 2>&1 | Out-Null
    & $nssmDest set $ServiceName AppRestartDelay 5000 2>&1 | Out-Null
    & $nssmDest set $ServiceName AppStdout (Join-Path $logsDir "stdout.log") 2>&1 | Out-Null
    & $nssmDest set $ServiceName AppStderr (Join-Path $logsDir "stderr.log") 2>&1 | Out-Null
    & $nssmDest set $ServiceName AppRotateFiles 1 2>&1 | Out-Null
    & $nssmDest set $ServiceName AppRotateOnline 1 2>&1 | Out-Null
    & $nssmDest set $ServiceName AppNoConsole 1 2>&1 | Out-Null
    & $nssmDest set $ServiceName DisplayName "Predict-A-Trade XAUUSD Agent" 2>&1 | Out-Null
    & $nssmDest set $ServiceName Description "Predict-A-Trade XAUUSD Windows Agent - MT4/MT5 bridge and signal delivery" 2>&1 | Out-Null
    & $nssmDest set $ServiceName Start SERVICE_AUTO_START 2>&1 | Out-Null
    Write-Host "  OK: Service registered with NSSM auto-restart (5s delay on crash)"
} else {
    Write-Host "[install] WARNING: NSSM not available - falling back to sc.exe"
    sc.exe create $ServiceName binPath= "$agentPath" start= auto 2>&1 | Out-Null
    sc.exe description $ServiceName "Predict-A-Trade XAUUSD Windows Agent" 2>&1 | Out-Null
}

# ─── Step 8: Create/update Scheduled Task for health check ───
Write-Host "[install] Setting up health-check Scheduled Task..."
$existingTask = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
if ($existingTask) {
    Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false -ErrorAction SilentlyContinue
}

$healthCheckScript = Join-Path $InstallDir "health-check.ps1"
$action = New-ScheduledTaskAction -Execute "powershell.exe" -Argument "-ExecutionPolicy Bypass -NoProfile -NonInteractive -WindowStyle Hidden -File `"$healthCheckScript`""

$settingsFile = Join-Path $InstallDir "settings.json"
$intervalMin = 1
if (Test-Path $settingsFile) {
    try {
        $cfg = Get-Content $settingsFile -Raw | ConvertFrom-Json
        if ($cfg.health_check_interval_minutes) {
            $intervalMin = [int]$cfg.health_check_interval_minutes
        }
    } catch {}
}

$trigger = New-ScheduledTaskTrigger -Once -At (Get-Date) -RepetitionInterval (New-TimeSpan -Minutes $intervalMin)
$principal = New-ScheduledTaskPrincipal -UserId "SYSTEM" -LogonType ServiceAccount -RunLevel Highest
$taskSettings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable
Register-ScheduledTask -TaskName $TaskName -Action $action -Trigger $trigger -Principal $principal -Settings $taskSettings -Force | Out-Null
Write-Host "  OK: Health-check task runs every $intervalMin min as SYSTEM"

# ─── Step 9: Start the service ───
Write-Host "[install] Starting service..."
if ($nssmDownloaded) {
    & $nssmDest start $ServiceName 2>&1 | Out-Null
} else {
    Start-Service -Name $ServiceName -ErrorAction SilentlyContinue
}
Start-Sleep -Seconds 2

$svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($svc -and $svc.Status -eq "Running") {
    Write-Host "  OK: Service started successfully"
    Write-PATEventLog -Message "Predict-A-Trade XAUUSD installation completed - service running (v$serverVersion)" -EventId 5
} else {
    Write-Host "  WARN: Service may not have started - check logs at $logsDir"
    Write-PATEventLog -Message "install.ps1: Service failed to start - check logs" -Level "Warning" -EventId 6
}

# ─── Summary ───
Write-Host ""
Write-Host "=========================================="
if ($isUpdate) {
    if ($serverVersion -ne "unknown" -and $serverVersion -ne $previousVersion) {
        Write-Host "  Update Complete! v$previousVersion -> v$serverVersion"
    } else {
        Write-Host "  Reinstall Complete! v$serverVersion"
    }
} else {
    Write-Host "  Installation Complete! v$serverVersion"
}
Write-Host "=========================================="
Write-Host "  Service:     $ServiceName"
Write-Host "  Status:      $(if ($svc -and $svc.Status -eq 'Running') { 'Running' } else { 'Check logs' })"
Write-Host "  Install Dir: $InstallDir"
Write-Host "  Version:     v$serverVersion"
Write-Host "  NSSM:        $(if ($is64bit) { '64-bit' } else { '32-bit' })"
Write-Host "  Logs:        $logsDir"
Write-Host "  Health Task: $TaskName (every $intervalMin min)"
Write-Host "  Event Log:   Application / $EventSource"
Write-Host ""
Write-Host "  To uninstall: irm $BaseUrl/uninstall.ps1 | iex"
Write-Host "  To update:     irm $BaseUrl/install.ps1 | iex"
Write-Host "=========================================="
Write-Host ""

# Pause so the user can read the summary
Write-Host "Press Enter to close this window..."
Read-Host
