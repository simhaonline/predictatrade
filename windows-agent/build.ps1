# ============================================================================
#  Predict-A-Trade Windows Agent — Build Script (PowerShell / Windows)
#  Produces a normal production Go executable for Windows amd64.
#  Usage:  .\build.ps1 [-Version 1.2.5] [-NoSign]
# ============================================================================
[CmdletBinding()]
param(
    [string]$Version,
    [switch]$NoSign
)

$ErrorActionPreference = "Stop"
$ROOT     = Resolve-Path (Join-Path $PSScriptRoot "..")
$SRC      = Join-Path $ROOT "cmd\client"
$DEPLOY   = Join-Path $ROOT "deploy"
$BINOUT   = Join-Path $DEPLOY "pat-agent.exe"
$MASTERSRC = Join-Path $ROOT "cmd\master"
$MASTERBIN = Join-Path $DEPLOY "pat-master.exe"

# --- 1. Validate Go --------------------------------------------------------
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Error "Go toolchain not found in PATH. Install Go 1.21+ and retry."
    exit 1
}

# --- 2. Determine version --------------------------------------------------
if (-not $Version) {
    $vg = Get-Content (Join-Path $ROOT "internal\version.go") | Where-Object { $_ -match 'const AgentVersion\s*=\s*"([0-9.]+)"' }
    if ($vg -match 'const AgentVersion\s*=\s*"([0-9.]+)"') { $Version = $Matches[1] }
    else { $Version = "0.0.0-dev" }
}

# --- 3. Generate Windows resources (version info + manifest) ---------------
Write-Host "[build] Generating Windows resources..."
$winres = Get-Command winres -ErrorAction SilentlyContinue
$goversioninfo = Get-Command goversioninfo -ErrorAction SilentlyContinue
$resGenerated = $false
if ($winres) {
    Push-Location $SRC
    & winres make --in winres.json --out resource_windows_amd64.syso 2>&1 | Out-Null
    $resGenerated = $LASTEXITCODE -eq 0
    Pop-Location
}
if (-not $resGenerated -and $goversioninfo) {
    Push-Location $SRC
    & goversioninfo -manifest=manifest.xml -o resource_windows_amd64.syso versioninfo.json 2>&1 | Out-Null
    $resGenerated = $LASTEXITCODE -eq 0
    Pop-Location
}
if ($resGenerated) { Write-Host "[build] Resource metadata: OK" } else { Write-Warning "[build] Resource generation skipped (install winres or goversioninfo for version info)" }

# --- 4. Compile Windows executable (Client) -------------------------------
Write-Host "[build] Cross-compiling client (windows/amd64, v$Version)..."
$ldflags = "-s -w -X github.com/predictatrade/windows-agent/internal.buildInfo=$Version"
$env:GOOS = "windows"; $env:GOARCH = "amd64"; $env:CGO_ENABLED = "0"
Push-Location $ROOT
go build -trimpath -ldflags $ldflags -o $BINOUT .\cmd\client\
if ($LASTEXITCODE -ne 0) { Write-Error "Client build failed"; exit 1 }
Pop-Location
Write-Host "[build] Client executable: $BINOUT"

# --- 4b. Compile Windows executable (Master Node / data-only) --------------
Write-Host "[build] Cross-compiling master (windows/amd64, v$Version)..."
Push-Location $ROOT
go build -trimpath -ldflags $ldflags -o $MASTERBIN .\cmd\master\
if ($LASTEXITCODE -ne 0) { Write-Error "Master build failed"; exit 1 }
Pop-Location
Write-Host "[build] Master executable: $MASTERBIN"

# --- 5. Code signing (Authenticode) ----------------------------------------
$signMode = "Unsigned"
if (-not $NoSign -and $env:PAT_SIGN_CERT -and (Get-Command signtool -ErrorAction SilentlyContinue)) {
    Write-Host "[build] Signing with Authenticode (SHA-256 + RFC3161 timestamp)..."
    $ts = if ($env:PAT_TIMESTAMP_URL) { $env:PAT_TIMESTAMP_URL } else { "http://timestamp.digicert.com" }
    & signtool sign /fd SHA256 /tr "$ts" /td SHA256 /f "$env:PAT_SIGN_CERT" /p "$env:PAT_SIGN_CERT_PASSWORD" "$BINOUT" 2>&1
    if ($LASTEXITCODE -eq 0) { $signMode = "Production Signed" } else { Write-Warning "[build] Signing failed" }
} elseif ($env:PAT_SIGN_CERT) {
    Write-Warning "[build] PAT_SIGN_CERT set but signtool.exe not found — skipping signing."
}

# --- 6. Verify signature ---------------------------------------------------
if ($signMode -ne "Unsigned") {
    $sig = Get-AuthenticodeSignature $BINOUT
    Write-Host "[build] Signature status: $($sig.Status)"
}

# --- 7. SHA256 -------------------------------------------------------------
$hash = (Get-FileHash $BINOUT -Algorithm SHA256).Hash
Write-Host "[build] SHA256: $hash"

# --- 7b. Master-node binary already built above (distinct source/role) ------
$MasterBin = $MASTERBIN
if ($signMode -ne "Unsigned") {
    $mSig = Get-AuthenticodeSignature $MasterBin
    Write-Host "[build] Master signature status: $($mSig.Status)"
}
Write-Host "[build] Master-node binary: $MasterBin"

# --- 8. Summary ------------------------------------------------------------
Write-Host ""
Write-Host "================================================="
Write-Host " Predict-A-Trade Windows Build"
Write-Host "================================================="
Write-Host " Version:           $Version"
Write-Host " Architecture:      windows/amd64"
Write-Host " Executable:        $BINOUT"
Write-Host " Resource Metadata: $(if ($resGenerated) {'OK'} else {'SKIPPED'})"
Write-Host " Signature:         $signMode"
Write-Host " SHA256:            $hash"
Write-Host " Master binary:     $MASTERBIN"
Write-Host " Build:             SUCCESS"
Write-Host "================================================="

# Note: a self-signed certificate is acceptable ONLY for labelled local dev/test.
# Production builds MUST use a legitimate organization code-signing certificate.
