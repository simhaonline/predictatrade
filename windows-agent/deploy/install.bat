@echo off
REM ============================================================
REM  Predict-A-Trade XAUUSD — Windows Agent Installer (Wrapper)
REM  Double-click to install, or run from Command Prompt.
REM ============================================================

echo.
echo ========================================
echo   Predict-A-Trade XAUUSD Installer
echo ========================================
echo.

REM Check if install.ps1 exists locally; if so, run it directly.
REM Otherwise, download and run from the web.

if exist "%~dp0install.ps1" (
    echo Running local install.ps1...
    powershell -ExecutionPolicy Bypass -NoProfile -File "%~dp0install.ps1"
) else (
    echo Downloading and running installer from the web...
    powershell -ExecutionPolicy Bypass -NoProfile -Command "irm https://downloads.predictatrade.com/windows-agent/install.ps1 | iex"
)

echo.
pause
