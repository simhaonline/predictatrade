@echo off
REM ============================================================
REM  Predict-A-Trade XAUUSD — Windows Agent Installer (BAT)
REM  No PowerShell AMSI blocking — pure batch + bitsadmin
REM  Usage: Download and double-click, or run from CMD.
REM ============================================================

setlocal enabledelayedexpansion

set "INSTALL_DIR=C:\PredictATrade\XAUUSD"
set "BASE_URL=https://downloads.predictatrade.com/windows-agent"
set "SERVICE_NAME=pat-agent"
set "AGENT_EXE=pat-agent.exe"
set "NSSM_EXE=nssm.exe"

echo.
echo ========================================
echo   Predict-A-Trade XAUUSD Installer
echo ========================================
echo.

REM Check admin rights
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo Requesting administrator privileges...
    powershell -Command "Start-Process cmd -ArgumentList '/c %~dp0install.bat' -Verb RunAs"
    exit /b
)

REM Step 1: Create directories
echo [1/7] Creating directories...
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"
if not exist "%INSTALL_DIR%\logs" mkdir "%INSTALL_DIR%\logs"
echo   OK

REM Step 2: Add Defender exclusions (via reg + powershell, minimal)
echo [2/7] Adding Defender exclusions...
powershell -Command "Add-MpPreference -ExclusionPath 'C:\PredictATrade' -ErrorAction SilentlyContinue; Add-MpPreference -ExclusionProcess 'pat-agent.exe' -ErrorAction SilentlyContinue" 2>nul
echo   OK

REM Step 3: Stop old service if running
echo [3/7] Stopping old service...
sc.exe stop %SERVICE_NAME% >nul 2>&1
if exist "%INSTALL_DIR%\%NSSM_EXE%" "%INSTALL_DIR%\%NSSM_EXE%" stop %SERVICE_NAME% >nul 2>&1
timeout /t 2 /nobreak >nul
echo   OK

REM Step 4: Download pat-agent.exe
echo [4/7] Downloading agent...
if exist "%INSTALL_DIR%\%AGENT_EXE%" del /f "%INSTALL_DIR%\%AGENT_EXE%" 2>nul
powershell -Command "Invoke-WebRequest -Uri '%BASE_URL%/%AGENT_EXE%' -OutFile '%INSTALL_DIR%\%AGENT_EXE%' -UseBasicParsing -TimeoutSec 120" 2>nul
if not exist "%INSTALL_DIR%\%AGENT_EXE%" (
    echo   Trying bitsadmin fallback...
    bitsadmin /transfer patdownload %BASE_URL%/%AGENT_EXE% "%INSTALL_DIR%\%AGENT_EXE%" >nul 2>&1
)
if exist "%INSTALL_DIR%\%AGENT_EXE%" (
    echo   OK: Downloaded
) else (
    echo   FATAL: Download failed
    pause
    exit /b 1
)

REM Unblock the file
powershell -Command "Unblock-File -Path '%INSTALL_DIR%\%AGENT_EXE%' -ErrorAction SilentlyContinue" 2>nul

REM Step 5: Download NSSM
echo [5/7] Downloading NSSM...
if exist "%INSTALL_DIR%\%NSSM_EXE%" del /f "%INSTALL_DIR%\%NSSM_EXE%" 2>nul
powershell -Command "Invoke-WebRequest -Uri '%BASE_URL%/nssm/win64/nssm.exe' -OutFile '%INSTALL_DIR%\%NSSM_EXE%' -UseBasicParsing -TimeoutSec 60" 2>nul
if not exist "%INSTALL_DIR%\%NSSM_EXE%" (
    bitsadmin /transfer nssmdownload %BASE_URL%/nssm/win64/nssm.exe "%INSTALL_DIR%\%NSSM_EXE%" >nul 2>&1
)
if exist "%INSTALL_DIR%\%NSSM_EXE%" (
    powershell -Command "Unblock-File -Path '%INSTALL_DIR%\%NSSM_EXE%' -ErrorAction SilentlyContinue" 2>nul
    echo   OK: NSSM downloaded
) else (
    echo   WARN: NSSM not available, will use sc.exe
)

REM Step 6: Kill old process + remove old service
echo [6/7] Removing old service...
taskkill /f /im %AGENT_EXE% >nul 2>&1
timeout /t 1 /nobreak >nul
if exist "%INSTALL_DIR%\%NSSM_EXE%" (
    "%INSTALL_DIR%\%NSSM_EXE%" remove %SERVICE_NAME% confirm >nul 2>&1
) else (
    sc.exe delete %SERVICE_NAME% >nul 2>&1
)
timeout /t 2 /nobreak >nul

REM Step 7: Create and start service
echo [7/7] Creating service...
set "SVC_CREATED=0"

if exist "%INSTALL_DIR%\%NSSM_EXE%" (
    echo   Trying NSSM...
    "%INSTALL_DIR%\%NSSM_EXE%" install %SERVICE_NAME% "%INSTALL_DIR%\%AGENT_EXE%" >nul 2>&1
    if !errorlevel! equ 0 (
        "%INSTALL_DIR%\%NSSM_EXE%" set %SERVICE_NAME% AppDirectory "%INSTALL_DIR%" >nul 2>&1
        "%INSTALL_DIR%\%NSSM_EXE%" set %SERVICE_NAME% AppStdout "%INSTALL_DIR%\logs\stdout.log" >nul 2>&1
        "%INSTALL_DIR%\%NSSM_EXE%" set %SERVICE_NAME% AppStderr "%INSTALL_DIR%\logs\stderr.log" >nul 2>&1
        "%INSTALL_DIR%\%NSSM_EXE%" set %SERVICE_NAME% AppRotateFiles 1 >nul 2>&1
        "%INSTALL_DIR%\%NSSM_EXE%" set %SERVICE_NAME% AppExit Default Restart >nul 2>&1
        "%INSTALL_DIR%\%NSSM_EXE%" set %SERVICE_NAME% AppRestartDelay 5000 >nul 2>&1
        "%INSTALL_DIR%\%NSSM_EXE%" set %SERVICE_NAME% Start SERVICE_AUTO_START >nul 2>&1
        "%INSTALL_DIR%\%NSSM_EXE%" start %SERVICE_NAME% >nul 2>&1
        timeout /t 3 /nobreak >nul
        sc.exe query %SERVICE_NAME% | find "RUNNING" >nul 2>&1
        if !errorlevel! equ 0 (
            echo   OK: Service RUNNING via NSSM
            set "SVC_CREATED=1"
        )
    )
)

if !SVC_CREATED! equ 0 (
    echo   Trying sc.exe...
    sc.exe delete %SERVICE_NAME% >nul 2>&1
    timeout /t 1 /nobreak >nul
    sc.exe create %SERVICE_NAME% binPath= "%INSTALL_DIR%\%AGENT_EXE%" start= auto >nul 2>&1
    if !errorlevel! equ 0 (
        sc.exe description %SERVICE_NAME% "Predict-A-Trade XAUUSD Windows Agent" >nul 2>&1
        sc.exe failure %SERVICE_NAME% reset= 60 actions= restart/5000 >nul 2>&1
        sc.exe start %SERVICE_NAME% >nul 2>&1
        timeout /t 3 /nobreak >nul
        sc.exe query %SERVICE_NAME% | find "RUNNING" >nul 2>&1
        if !errorlevel! equ 0 (
            echo   OK: Service RUNNING via sc.exe
            set "SVC_CREATED=1"
        )
    )
)

if !SVC_CREATED! equ 0 (
    echo   Service failed — trying scheduled task...
    powershell -Command "Register-ScheduledTask -TaskName '%SERVICE_NAME%' -Action (New-ScheduledTaskAction -Execute '%INSTALL_DIR%\%AGENT_EXE%' -WorkingDirectory '%INSTALL_DIR%') -Trigger (New-ScheduledTaskTrigger -AtStartup), (New-ScheduledTaskTrigger -Once -At (Get-Date) -RepetitionInterval (New-TimeSpan -Minutes 5)) -Settings (New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -StartWhenAvailable) -Principal (New-ScheduledTaskPrincipal -UserId SYSTEM -LogonType ServiceAccount -RunLevel Highest) -Force" 2>nul
    powershell -Command "Start-ScheduledTask -TaskName '%SERVICE_NAME%'" 2>nul
    timeout /t 3 /nobreak >nul
    tasklist /fi "imagename eq %AGENT_EXE%" | find "%AGENT_EXE%" >nul 2>&1
    if !errorlevel! equ 0 (
        echo   OK: Agent running via Scheduled Task
        set "SVC_CREATED=1"
    )
)

if !SVC_CREATED! equ 0 (
    echo   All service methods failed — launching directly...
    start "" /b "%INSTALL_DIR%\%AGENT_EXE%"
    timeout /t 3 /nobreak >nul
    tasklist /fi "imagename eq %AGENT_EXE%" | find "%AGENT_EXE%" >nul 2>&1
    if !errorlevel! equ 0 (
        echo   OK: Agent running directly
        set "SVC_CREATED=1"
    )
)

if !SVC_CREATED! equ 0 (
    echo.
    echo   ERROR: Agent could not start.
    echo   Check Windows Security ^> Protection history
    echo   Then run manually: %INSTALL_DIR%\%AGENT_EXE%
    echo.
    pause
    exit /b 1
)

echo.
echo ========================================
echo   Installation Complete!
echo ========================================
echo   Install Dir: %INSTALL_DIR%
echo   Health:      http://127.0.0.1:9000/health
echo   Logs:        %INSTALL_DIR%\logs
echo.
pause
