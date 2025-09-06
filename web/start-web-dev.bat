@echo off
REM Bitcoin Sprint - Next.js Development Server
REM Starts the web application with optimized memory settings
REM Port: 3002 (configured for enterprise tier)

cd /d %~dp0

echo ===============================================
echo  Bitcoin Sprint Web Application
echo  Starting Next.js Development Server
echo  Memory: 8GB allocated
echo  Port: 3002
echo ===============================================

REM Check if node_modules exists
if not exist "node_modules" (
    echo ERROR: node_modules not found!
    echo Run 'npm install' first
    pause
    exit /b 1
)

REM Start Next.js with increased memory for enterprise workloads
echo Starting server...
node --max-old-space-size=8192 node_modules\next\dist\bin\next dev -p 3002

echo.
echo Server stopped.
pause
