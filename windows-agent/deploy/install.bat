@echo off
REM ============================================================
REM  Predict-A-Trade XAUUSD — Windows Agent Installer
REM  Clean installer — no Defender modification, no process kill
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
    powershell -Command "Start-Process cmd -ArgumentList '/c %~f0' -Verb RunAs"
    exit /b
)

REM Step 1: Create directories
echo [1/6] Creating directories...
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"
if not exist "%INSTALL_DIR%\logs" mkdir "%INSTALL_DIR%\logs"
echo   OK

REM Step 2: Stop old service gracefully
echo [2/6] Stopping old service...
sc.exe stop %SERVICE_NAME% >nul 2>&1
if exist "%INSTALL_DIR%\%NSSM_EXE%" "%INSTALL_DIR%\%NSSM_EXE%" stop %SERVICE_NAME% >nul 2>&1
timeout /t 3 /nobreak >nul
echo   OK

REM Step 3: Download pat-agent.exe
echo [3/6] Downloading agent...
if exist "%INSTALL_DIR%\%AGENT_EXE%" del /f "%INSTALL_DIR%\%AGENT_EXE%" 2>nul
powershell -Command "Invoke-WebRequest -Uri '%BASE_URL%/%AGENT_EXE%' -OutFile '%INSTALL_DIR%\%AGENT_EXE%' -UseBasicParsing -TimeoutSec 120" 2>nul
if not exist "%INSTALL_DIR%\%AGENT_EXE%" (
    echo   Trying bitsadmin...
    bitsadmin /transfer patdl %BASE_URL%/%AGENT_EXE% "%INSTALL_DIR%\%AGENT_EXE%" >nul 2>&1
)
if not exist "%INSTALL_DIR%\%AGENT_EXE%" (
    echo   FATAL: Download failed
    pause
    exit /b 1
)
powershell -Command "Unblock-File -Path '%INSTALL_DIR%\%AGENT_EXE%' -ErrorAction SilentlyContinue" 2>nul
echo   OK: Downloaded

REM Step 4: Download NSSM
echo [4/6] Downloading NSSM...
if exist "%INSTALL_DIR%\%NSSM_EXE%" del /f "%INSTALL_DIR%\%NSSM_EXE%" 2>nul
powershell -Command "Invoke-WebRequest -Uri '%BASE_URL%/nssm/win64/nssm.exe' -OutFile '%INSTALL_DIR%\%NSSM_EXE%' -UseBasicParsing -TimeoutSec 60" 2>nul
if exist "%INSTALL_DIR%\%NSSM_EXE%" (
    powershell -Command "Unblock-File -Path '%INSTALL_DIR%\%NSSM_EXE%' -ErrorAction SilentlyContinue" 2>nul
    echo   OK
) else (
    echo   WARN: NSSM not available
)

REM Step 5: Remove old service, create new one
echo [5/6] Creating service...
sc.exe delete %SERVICE_NAME% >nul 2>&1
timeout /t 2 /nobreak >nul

set "SVC_OK=0"

if exist "%INSTALL_DIR%\%NSSM_EXE%" (
    echo   Trying NSSM...
    "%INSTALL_DIR%\%NSSM_EXE%" install %SERVICE_NAME% "%INSTALL_DIR%\%AGENT_EXE%" >nul 2>&1
    "%INSTALL_DIR%\%NSSM_EXE%" set %SERVICE_NAME% AppDirectory "%INSTALL_DIR%" >nul 2>&1
    "%INSTALL_DIR%\%NSSM_EXE%" set %SERVICE_NAME% AppStdout "%INSTALL_DIR%\logs\stdout.log" >nul 2>&1
    "%INSTALL_DIR%\%NSSM_EXE%" set %SERVICE_NAME% AppStderr "%INSTALL_DIR%\logs\stderr.log" >nul 2>&1
    "%INSTALL_DIR%\%NSSM_EXE%" set %SERVICE_NAME% AppExit Default Restart >nul 2>&1
    "%INSTALL_DIR%\%NSSM_EXE%" set %SERVICE_NAME% AppRestartDelay 5000 >nul 2>&1
    "%INSTALL_DIR%\%NSSM_EXE%" set %SERVICE_NAME% Start SERVICE_AUTO_START >nul 2>&1
    "%INSTALL_DIR%\%NSSM_EXE%" start %SERVICE_NAME% >nul 2>&1
    timeout /t 3 /nobreak >nul
    sc.exe query %SERVICE_NAME% 2>nul | find "RUNNING" >nul 2>&1
    if !errorlevel! equ 0 (
        echo   OK: Service RUNNING via NSSM
        set "SVC_OK=1"
    )
)

if !SVC_OK! equ 0 (
    echo   Trying sc.exe...
    sc.exe delete %SERVICE_NAME% >nul 2>&1
    timeout /t 1 /nobreak >nul
    sc.exe create %SERVICE_NAME% binPath= "%INSTALL_DIR%\%AGENT_EXE%" start= auto >nul 2>&1
    if !errorlevel! equ 0 (
        sc.exe description %SERVICE_NAME% "Predict-A-Trade XAUUSD Agent" >nul 2>&1
        sc.exe failure %SERVICE_NAME% reset= 60 actions= restart/5000 >nul 2>&1
        sc.exe start %SERVICE_NAME% >nul 2>&1
        timeout /t 3 /nobreak >nul
        sc.exe query %SERVICE_NAME% 2>nul | find "RUNNING" >nul 2>&1
        if !errorlevel! equ 0 (
            echo   OK: Service RUNNING via sc.exe
            set "SVC_OK=1"
        )
    )
)

if !SVC_OK! equ 0 (
    echo   Trying scheduled task...
    sc.exe delete %SERVICE_NAME% >nul 2>&1
    powershell -Command "Register-ScheduledTask -TaskName '%SERVICE_NAME%' -Action (New-ScheduledTaskAction -Execute '%INSTALL_DIR%\%AGENT_EXE%') -Trigger @((New-ScheduledTaskTrigger -AtStartup), (New-ScheduledTaskTrigger -Once -At (Get-Date) -RepetitionInterval (New-TimeSpan -Minutes 5))) -Settings (New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -StartWhenAvailable) -Principal (New-ScheduledTaskPrincipal -UserId SYSTEM -LogonType ServiceAccount -RunLevel Highest) -Force; Start-ScheduledTask -TaskName '%SERVICE_NAME%'" 2>nul
    timeout /t 3 /nobreak >nul
    tasklist /fi "imagename eq %AGENT_EXE%" 2>nul | find "%AGENT_EXE%" >nul 2>&1
    if !errorlevel! equ 0 (
        echo   OK: Agent running via Scheduled Task
        set "SVC_OK=1"
    )
)

if !SVC_OK! equ 0 (
    echo   Trying direct launch...
    start "" /b "%INSTALL_DIR%\%AGENT_EXE%"
    timeout /t 3 /nobreak >nul
    tasklist /fi "imagename eq %AGENT_EXE%" 2>nul | find "%AGENT_EXE%" >nul 2>&1
    if !errorlevel! equ 0 (
        echo   OK: Agent running
        set "SVC_OK=1"
    )
)

if !SVC_OK! equ 0 (
    echo.
    echo   ERROR: Could not start agent.
    echo   Add exclusion manually in Windows Security:
    echo     Settings ^> Update ^& Security ^> Windows Security
    echo     Virus ^& threat protection ^> Exclusions ^> Add
    echo     Add folder: C:\PredictATrade
    echo   Then run: %INSTALL_DIR%\%AGENT_EXE%
    echo.
    pause
    exit /b 1
)

echo.
echo ========================================
echo   Installation Complete!
echo ========================================
echo   Dir:    %INSTALL_DIR%
echo   Health: http://127.0.0.1:9000/health
echo   Logs:   %INSTALL_DIR%\logs
echo.
pause
