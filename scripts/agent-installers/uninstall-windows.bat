@echo off
REM ZID Agent - Windows Uninstallation Script

setlocal

echo ============================================
echo ZID Agent - Windows Uninstaller
echo ============================================
echo.

set "INSTALL_DIR=%LOCALAPPDATA%\ZIDAgent"
set "STARTUP_LINK=%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup\ZID Agent.lnk"

echo This will remove the ZID Agent from your computer.
echo Installation directory: %INSTALL_DIR%
echo.

set /p CONFIRM="Are you sure you want to uninstall? (Y/N): "
if /i not "%CONFIRM%"=="Y" (
    echo Uninstall cancelled.
    pause
    exit /b 0
)

REM Stop running agent process
echo Stopping ZID Agent...
taskkill /F /IM zid-agent.exe >nul 2>&1
if %errorLevel% equ 0 (
    echo Agent process stopped.
) else (
    echo No running agent process found.
)

timeout /t 2 /nobreak >nul

REM Remove startup shortcut
if exist "%STARTUP_LINK%" (
    echo Removing startup shortcut...
    del /F /Q "%STARTUP_LINK%"
)

REM Remove installation directory
if exist "%INSTALL_DIR%" (
    echo Removing installation directory...
    rmdir /S /Q "%INSTALL_DIR%"
    if %errorLevel% neq 0 (
        echo WARNING: Could not remove installation directory.
        echo Please delete manually: %INSTALL_DIR%
    )
)

echo.
echo ============================================
echo Uninstall Complete!
echo ============================================
echo.
echo ZID Agent has been removed from your system.
echo.
pause
