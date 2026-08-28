@echo off
REM ============================================================
REM  Predict-A-Trade XAUUSD — Windows Agent Installer (launcher)
REM  Delegates to the canonical PowerShell installer (install.ps1),
REM  which keeps Master and Client in SEPARATE directories and reuses
REM  an existing nssm instead of overwriting it. This fixes the
REM  "same device nssm conflict" when both roles are installed.
REM  Usage:  install.bat [client|master]   (default: client)
REM ============================================================
setlocal enabledelayedexpansion

set "ROLE=client"
if not "%~1"=="" set "ROLE=%~1"
if /i not "%ROLE%"=="master" if /i not "%ROLE%"=="client" set "ROLE=client"

set "BASE_URL=https://downloads.predictatrade.com/windows-agent"
set "TMP_PS=%TEMP%\pat_install_%RANDOM%.ps1"

echo.
echo ========================================
echo   Predict-A-Trade XAUUSD Installer (%ROLE%)
echo ========================================
echo.

REM Check admin rights
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo Requesting administrator privileges...
    powershell -Command "Start-Process cmd -ArgumentList '/c %~f0 %ROLE%' -Verb RunAs"
    exit /b
)

echo Downloading installer...
powershell -Command "Invoke-WebRequest -Uri '%BASE_URL%/install.ps1' -OutFile '%TMP_PS%' -UseBasicParsing -TimeoutSec 60" 2>nul
if not exist "%TMP_PS%" (
    echo FATAL: Could not download installer.
    pause
    exit /b 1
)

echo Launching %ROLE% installer...
powershell -ExecutionPolicy Bypass -NoProfile -File "%TMP_PS%" -Mode %ROLE% -EngineHost live.predictatrade.com -BaseUrl "%BASE_URL%/%ROLE%"
set "RC=%errorlevel%"

if exist "%TMP_PS%" del /f "%TMP_PS%" 2>nul
exit /b %RC%
