@echo off
echo Bitcoin Sprint Service Installer
echo -----------------------------

REM Check for admin privileges
NET SESSION >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo Error: Administrator privileges required.
    echo Please run this script as Administrator.
    pause
    exit /b 1
)

REM Create the installation directory
echo Creating installation directory...
if not exist "C:\Program Files\Bitcoin Sprint" (
    mkdir "C:\Program Files\Bitcoin Sprint"
    if %ERRORLEVEL% neq 0 (
        echo Failed to create installation directory.
        pause
        exit /b 1
    )
)

REM Copy the executable
echo Copying executable...
copy /y "bitcoin-sprint.exe" "C:\Program Files\Bitcoin Sprint\"
if %ERRORLEVEL% neq 0 (
    echo Failed to copy executable.
    pause
    exit /b 1
)

REM Install the service
echo Installing service...
sc stop BitcoinSprint >nul 2>&1
sc delete BitcoinSprint >nul 2>&1
sc create BitcoinSprint binPath= "\"C:\Program Files\Bitcoin Sprint\bitcoin-sprint.exe\"" DisplayName= "Bitcoin Sprint" start= auto
if %ERRORLEVEL% neq 0 (
    echo Failed to create service.
    pause
    exit /b 1
)

REM Start the service
echo Starting service...
sc start BitcoinSprint
if %ERRORLEVEL% neq 0 (
    echo Failed to start service.
    echo Check Windows Event Viewer for details.
)

echo.
echo Installation completed successfully.
echo Service Name: BitcoinSprint
echo Executable: C:\Program Files\Bitcoin Sprint\bitcoin-sprint.exe
echo.
pause
