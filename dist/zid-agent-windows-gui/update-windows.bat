@echo off
REM ZID Agent - Windows Update Script
REM Updates an existing ZID Agent installation

setlocal enabledelayedexpansion

echo ============================================
echo ZID Agent - Windows Updater
echo ============================================
echo.

REM Check if binary exists
set BINARY=zid-agent-windows-gui.exe
if not exist "%BINARY%" (
    echo ERROR: %BINARY% not found in current directory.
    echo Please run this script from the extracted agent folder.
    pause
    exit /b 1
)

REM Installation directory
set INSTALL_DIR=%LOCALAPPDATA%\ZIDAgent
set INSTALL_PATH=%INSTALL_DIR%\zid-agent.exe

REM Check if agent is installed
if not exist "%INSTALL_PATH%" (
    echo ERROR: ZID Agent is not installed at %INSTALL_PATH%
    echo Please install the agent first using install-windows.bat
    pause
    exit /b 1
)

echo Current installed version:
if exist "%INSTALL_PATH%" (
    "%INSTALL_PATH%" -version 2>nul
)

echo.
echo New version in this bundle:
"%BINARY%" -version 2>nul

echo.
set /p CONTINUE=Continue with update? (Y/n):
if /i "%CONTINUE%"=="n" (
    echo Update cancelled.
    pause
    exit /b 0
)

echo.
echo Stopping running agent...
taskkill /F /IM zid-agent.exe >nul 2>&1
timeout /t 2 /nobreak >nul

echo Updating binary...
copy /Y "%BINARY%" "%INSTALL_PATH%" >nul
if errorlevel 1 (
    echo ERROR: Failed to copy binary. Make sure the agent is stopped.
    pause
    exit /b 1
)

echo Binary updated successfully!
echo.

set /p START_NOW=Start the updated agent now? (Y/n):
if /i not "%START_NOW%"=="n" (
    start "" "%INSTALL_PATH%"
    echo.
    echo Agent started! Look for the ZID icon in your system tray.
) else (
    echo Agent will start on next login.
)

echo.
echo ============================================
echo Update complete!
echo ============================================
echo.
pause
