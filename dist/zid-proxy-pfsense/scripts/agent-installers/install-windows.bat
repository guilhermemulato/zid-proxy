@echo off
REM ZID Agent - Windows Installation Script
REM This script installs the ZID Agent and configures it to start on login

setlocal

echo ============================================
echo ZID Agent - Windows Installer
echo ============================================
echo.

REM Check if running as administrator
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo WARNING: Not running as administrator.
    echo The agent will be installed for current user only.
    echo.
    pause
)

REM Define installation directory
set "INSTALL_DIR=%LOCALAPPDATA%\ZIDAgent"
set "BINARY_NAME=zid-agent-windows-gui.exe"
set "STARTUP_DIR=%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup"

echo Installation directory: %INSTALL_DIR%
echo Startup directory: %STARTUP_DIR%
echo.

REM Create installation directory
if not exist "%INSTALL_DIR%" (
    echo Creating installation directory...
    mkdir "%INSTALL_DIR%"
)

REM Copy binary
if exist "%BINARY_NAME%" (
    echo Installing ZID Agent...
    copy /Y "%BINARY_NAME%" "%INSTALL_DIR%\zid-agent.exe"
    if %errorLevel% neq 0 (
        echo ERROR: Failed to copy agent binary.
        pause
        exit /b 1
    )
) else (
    echo ERROR: %BINARY_NAME% not found in current directory.
    echo Please run this script from the extracted agent folder.
    pause
    exit /b 1
)

REM Create startup shortcut
echo Creating startup shortcut...
powershell -Command "$WshShell = New-Object -comObject WScript.Shell; $Shortcut = $WshShell.CreateShortcut('%STARTUP_DIR%\ZID Agent.lnk'); $Shortcut.TargetPath = '%INSTALL_DIR%\zid-agent.exe'; $Shortcut.WorkingDirectory = '%INSTALL_DIR%'; $Shortcut.Description = 'ZID Agent - Network Monitoring'; $Shortcut.Save()"

if %errorLevel% neq 0 (
    echo WARNING: Failed to create startup shortcut.
    echo You can manually add the agent to startup.
)

echo.
echo ============================================
echo Installation Complete!
echo ============================================
echo.
echo The ZID Agent has been installed to:
echo   %INSTALL_DIR%\zid-agent.exe
echo.
echo The agent will start automatically on next login.
echo.
echo To start the agent now, run:
echo   "%INSTALL_DIR%\zid-agent.exe"
echo.
echo To uninstall, run:
echo   uninstall-windows.bat
echo.

REM Ask if user wants to start now
set /p START_NOW="Start ZID Agent now? (Y/N): "
if /i "%START_NOW%"=="Y" (
    echo Starting ZID Agent...
    start "" "%INSTALL_DIR%\zid-agent.exe"
    echo.
    echo Agent started! Look for the ZID icon in your system tray.
)

echo.
pause
