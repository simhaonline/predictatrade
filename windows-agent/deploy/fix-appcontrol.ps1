<#
.SYNOPSIS
    Predict-A-Trade — Fix "An Application Control policy has blocked this file"
.DESCRIPTION
    Diagnoses and fixes the Windows layer that is blocking pat-master.exe /
    pat-agent.exe. Three different blockers exist and each needs a different fix:

      1. Smart App Control (SAC)      — Windows 11 consumer policy. Cannot be
                                        disabled programmatically (by design).
                                        Script detects it and prints the exact
                                        UI toggle path. One-time switch.
      2. AppLocker executable rules   — fixable programmatically: prints the
                                        exact rule to add for C:\PredictATrade\*
      3. Windows Defender AV          — folder + process exclusions verified
                                        and re-added here.

    Run in ANY PowerShell (it self-elevates):
      irm https://downloads.predictatrade.com/windows-agent/fix-appcontrol.ps1 | iex
#>
param([switch]$Silent)

$ErrorActionPreference = "Continue"
$Dirs = @("C:\PredictATrade", "$env:ProgramData\PredictATrade")

# ─── Self-elevate ───
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Host "[fix-appcontrol] Administrator rights required — a UAC prompt will appear, click Yes."
    $remoteScript = "https://downloads.predictatrade.com/windows-agent/fix-appcontrol.ps1"
    try { $c = Invoke-WebRequest -Uri $remoteScript -UseBasicParsing -TimeoutSec 30 | Select-Object -ExpandProperty Content } catch {
        Write-Host "[fix-appcontrol] FATAL: download failed: $_"; exit 1
    }
    $t = Join-Path $env:TEMP "pat_fix_appcontrol_$(Get-Random).ps1"
    Set-Content -Path $t -Value $scriptContent -Encoding UTF8
    try {
        $p = Start-Process -FilePath "powershell.exe" -ArgumentList "-ExecutionPolicy","Bypass","-NoProfile","-File","`"$t`"" -Verb RunAs -Wait -PassThru
        exit $p.ExitCode
    } catch { Write-Host "[fix-appcontrol] Elevation declined: $_"; exit 1 }
    finally { Remove-Item $t -Force -ErrorAction SilentlyContinue }
}

Write-Host ""
Write-Host "=============================================="
Write-Host "  Predict-A-Trade — App Control Diagnostic"
Write-Host "=============================================="
Write-Host ""

# ─── 1. Smart App Control state ───
Write-Host "[1] Checking Smart App Control (Windows 11)..."
$sacState = ""
try {
    $sacState = (Get-MpComputerStatus -ErrorAction Stop).SmartAppControlState
} catch {}
Write-Host "    Smart App Control state: $sacState"
if ($sacState -eq "On") {
    Write-Host ""
    Write-Host "  >>> SMART APP CONTROL IS ON — this is what blocks pat-master.exe <<<"
    Write-Host "  SAC cannot be disabled by script (Windows design). Turn it OFF in:"
    Write-Host "    1. Windows Security  (Start > 'Windows Security')"
    Write-Host "    2. App & browser control"
    Write-Host "    3. Smart App Control settings"
    Write-Host "    4. Select OFF  (it will warn; confirm)"
    Write-Host "  Then RE-RUN this script / the installer. SAC stays off permanently."
    Write-Host ""
} elseif ($sacState -eq "Eval") {
    Write-Host "  INFO: SAC is in evaluation mode — it silently blocks unsigned apps."
    Write-Host "  Turn it OFF the same way (Windows Security > App & browser control)."
    Write-Host ""
}

# ─── 2. AppLocker check ───
Write-Host "[2] Checking AppLocker policy..."
$hasAppLocker = $false
try {
    $policy = Get-AppLockerPolicy -Effective -ErrorAction Stop
    $exeRules = $policy.RuleCollections | Where-Object { $_.RuleCollectionType -eq "Exe" }
    if ($exeRules) { $hasAppLocker = $true }
} catch {}
if ($hasAppLocker) {
    $xml = ""
    try { $xml = (Get-AppLockerPolicy -Effective).ToXml() } catch {}
    if ($xml -match "PredictATrade") {
        Write-Host "  OK: an allow rule for PredictATrade already exists"
    } else {
        Write-Host "  AppLocker executable rules FOUND and C:\PredictATrade is NOT allowed."
        Write-Host "  Add the allow rule (2 minutes, UI — reliable on all builds):"
        Write-Host "    1. Win+R > secpol.msc  (or gpedit.msc)"
        Write-Host "    2. Application Control Policies > AppLocker > Executable rules"
        Write-Host "    3. Right-click > Create New Rule..."
        Write-Host "    4. Action: Allow | User or group: Everyone"
        Write-Host "    5. Conditions: Path | Path: C:\PredictATrade\*  > Create"
        Write-Host "  Then also allow: C:\ProgramData\PredictATrade\*"
    }
} else {
    Write-Host "  OK: no AppLocker restrictions detected"
}

# ─── 3. CodeIntegrity block evidence (WDAC / SAC) ───
Write-Host ""
Write-Host "[3] Last Application-Control blocks recorded by Windows:"
try {
    $evs = Get-WinEvent -LogName "Microsoft-Windows-CodeIntegrity/Operational" -MaxEvents 8 -ErrorAction SilentlyContinue
    if ($evs) {
        foreach ($e in $evs) {
            Write-Host ("  {0}  [ID {1}]  {2}" -f $e.TimeCreated, $e.Id, ($e.Message -split "`n")[0])
        }
    } else {
        Write-Host "  (no CodeIntegrity events found)"
    }
} catch {
    Write-Host "  Could not read CodeIntegrity log: $_"
}

# ─── 4. Defender AV exclusions re-verify ───
Write-Host ""
Write-Host "[4] Windows Defender AV exclusions:"
try {
    $mps = Get-MpPreference -ErrorAction Stop
    foreach ($p in @("C:\PredictATrade", "$env:ProgramData\PredictATrade")) {
        $hit = $mps.ExclusionPath | Where-Object { $_ -and $_.TrimEnd('\') -ieq $p }
        if ($hit) {
            Write-Host "  OK: folder exclusion active: $p"
        } else {
            try {
                Add-MpPreference -ExclusionPath $p -ErrorAction Stop
                Write-Host "  OK: added folder exclusion $p"
            } catch {
                Write-Host "  ACTION: add '$p' manually (Windows Security > Exclusions) — Tamper Protection blocked the script add"
            }
        }
    }
    try {
        Add-MpPreference -ExclusionProcess "pat-master.exe" -ErrorAction SilentlyContinue
        Add-MpPreference -ExclusionProcess "pat-agent.exe"  -ErrorAction SilentlyContinue
        Add-MpPreference -ExclusionProcess "nssm.exe"       -ErrorAction SilentlyContinue
        Write-Host "  OK: process exclusions attempted (pat-master.exe, pat-agent.exe, nssm.exe)"
    } catch {}
} catch {
    Write-Host "  WARN: could not read Defender preferences: $_"
}

# ─── 5. Launch test ───
Write-Host ""
Write-Host "[5] Launch test — C:\PredictATrade\Master\pat-master.exe -version"
$exe = "C:\PredictATrade\Master\pat-master.exe"
if (Test-Path $exe) {
    $out = & $exe -version 2>&1
    $code = $LASTEXITCODE
    if ($code -eq 0) {
        Write-Host "  OK: binary RAN (version $out) — no more blocking."
        Write-Host ""
        Write-Host "=== Next: re-run the master installer ==="
        Write-Host "    irm https://downloads.predictatrade.com/windows-agent/master/install.ps1 | iex"
    } else {
        Write-Host "  BLOCKED still (exit $code): $out"
        Write-Host "  → An App Control policy (SAC or WDAC/AppLocker) is still active — see [1]/[2] above."
        Write-Host "  → If SAC is ON, turning it OFF (Windows Security > App & browser control) is required;"
        Write-Host "    no script can bypass it by design."
    }
} else {
    Write-Host "  INFO: $exe not present — re-run the master installer first."
}

Write-Host ""
if (-not $Silent) { $null = Read-Host "Press Enter to close" }