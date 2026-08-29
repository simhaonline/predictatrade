<#
.SYNOPSIS
    Predict-A-Trade — Windows box diagnostic (run as Administrator)
.DESCRIPTION
    One-shot triage for the round-trip: IPC folders, EA files, service health,
    binary versions, Defender/Zone state. Copy the ENTIRE output back to the
    support engineer (it contains no secrets; the license key is masked).
.USAGE
    PowerShell (Run as Administrator):
      Set-ExecutionPolicy -Scope Process Bypass -Force
      .\pat-diagnose.ps1
#>

$ErrorActionPreference = 'Continue'
$bar = '=' * 78
Write-Host $bar
Write-Host ' Predict-A-Trade Windows diagnostics'
$os = Get-CimInstance Win32_OperatingSystem
Write-Host (" Host: {0}  |  OS: {1} {2}" -f $env:COMPUTERNAME, $os.Caption, $os.BuildNumber)
Write-Host $bar

Write-Host "`n[1] SERVICES"
foreach ($svc in 'pat-agent-client','pat-agent-master') {
    $s = Get-Service -Name $svc -ErrorAction SilentlyContinue
    if ($s) { Write-Host ("  {0,-22} {1}" -f $svc, $s.Status) }
    else    { Write-Host ("  {0,-22} NOT INSTALLED" -f $svc) }
}

Write-Host "`n[2] BINARIES + VERSIONS"
foreach ($d in 'C:\PredictATrade\Client','C:\PredictATrade\Master') {
    foreach ($exe in 'pat-agent.exe','pat-master.exe') {
        $p = Join-Path $d $exe
        if (Test-Path $p) {
            $fi = Get-Item $p
            Write-Host ("  {0}  {1,10:N0} bytes  {2}" -f $p, $fi.Length, $fi.LastWriteTime)
        }
    }
}

Write-Host "`n[3] ENV (machine scope) — what the services actually see"
foreach ($v in 'PAT_LIVE_WS_URL','PAT_DATA_WS_URL','PAT_API_URL','PAT_LOG_DIR','PAT_HEALTH_PORT','PAT_IPC_DIR','PAT_USERS_ROOT') {
    $val = [Environment]::GetEnvironmentVariable($v, 'Machine')
    Write-Host ("  {0,-18} = {1}" -f $v, ($(if ($val) { $val } else { '<unset>' })))
}

Write-Host "`n[4] DEFENDER (Tamper Protection silently blocks scripted exclusions)"
try {
    $st = Get-MpComputerStatus
    Write-Host ("  TamperProtected: {0}   RealTimeProtection: {1}" -f $st.IsTamperProtected, $st.RealTimeProtectionEnabled)
    $mp = Get-MpPreference
    foreach ($want in 'C:\PredictATrade', (Join-Path $env:ProgramData 'PredictATrade')) {
        $hit = @($mp.ExclusionPath) | Where-Object { $_ -and ($_.TrimEnd('\') -ieq $want.TrimEnd('\')) }
        Write-Host ("  Exclusion {0,-40} {1}" -f $want, $(if ($hit) { 'ACTIVE' } else { '*** MISSING ***' }))
    }
    # recent detections mentioning our binaries
    $det = Get-MpThreatDetection -ErrorAction SilentlyContinue | Sort-Object InitialDetectionTime -Descending | Select-Object -First 3
    foreach ($x in $det) {
        Write-Host ("  recent detection: {0}  {1}" -f $x.InitialDetectionTime, ($x.Resources -join ', '))
    }
} catch { Write-Host "  Defender cmdlets unavailable: $_" }

Write-Host "`n[5] EA IPC FILES — every MetaQuotes Common\Files folder"
$anyCommon = $false
Get-ChildItem 'C:\Users' -Directory -ErrorAction SilentlyContinue | ForEach-Object {
    $cf = Join-Path $_.FullName 'AppData\Roaming\MetaQuotes\Terminal\Common\Files'
    if (Test-Path $cf) {
        $anyCommon = $true
        $files = Get-ChildItem $cf -Filter 'PAT_*' -ErrorAction SilentlyContinue
        if ($files) {
            Write-Host ("  {0}" -f $cf)
            foreach ($f in $files) {
                Write-Host ("      {0,-28} {1,8} bytes   {2}" -f $f.Name, $f.Length, $f.LastWriteTime)
            }
        }
        else {
            Write-Host ("  {0}  (exists, no PAT_* files)" -f $cf)
        }
    }
}
if (-not $anyCommon) { Write-Host "  no MetaQuotes Common\Files folders found under C:\Users" }

Write-Host "`n[6] AGENT HEALTH ENDPOINTS"
foreach ($hp in 'http://127.0.0.1:9000/health','http://127.0.0.1:9001/health') {
    try {
        $r = Invoke-WebRequest -Uri $hp -UseBasicParsing -TimeoutSec 3
        Write-Host ("  {0}  → HTTP {1}" -f $hp, $r.StatusCode)
        $j = $r.Content | ConvertFrom-Json
        Write-Host ("      role={0} version={1} backend={2} connected={3}" -f $j.role, $j.version, $j.backend_connected, $j.backend_name)
    } catch {
        Write-Host ("  {0}  → UNREACHABLE ({1})" -f $hp, $_.Exception.Message.Split("`n")[0])
    }
}

Write-Host "`n[7] AGENT LOG TAILS"
foreach ($lg in 'C:\PredictATrade\Client\logs\agent.log','C:\PredictATrade\Master\logs\master_agent.log') {
    Write-Host ("  --- {0} (last 20) ---" -f $lg)
    if (Test-Path $lg) { Get-Content $lg -Tail 20 | ForEach-Object { Write-Host ("      {0}" -f $_) } }
    else { Write-Host '      (missing)' }
}

Write-Host "`n[8] TERMINALS + build info"
Get-ChildItem 'C:\Users\*\AppData\Roaming\MetaQuotes\Terminal\*' -Directory -ErrorAction SilentlyContinue |
    Where-Object { Test-Path (Join-Path $_.FullName 'origin.txt') } | ForEach-Object {
        $o = Get-Content (Join-Path $_.FullName 'origin.txt') -ErrorAction SilentlyContinue
        Write-Host ("  terminal {0}  origin={1}" -f $_.Name.Substring(0,8), $o)
    }

Write-Host "`n$bar"
Write-Host ' DONE — copy this entire output back to support.'
Write-Host $bar