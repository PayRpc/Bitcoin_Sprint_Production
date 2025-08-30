@echo off
echo ===========================================
echo   Bitcoin Sprint - Unified Startup
echo ===========================================
echo Python 3.13 | FastAPI Gateway | Next.js Frontend
echo.

cd /d "%~dp0"

echo [1/4] Starting FastAPI Gateway (Port 8000)...
cd fastapi-gateway
if exist venv\Scripts\activate.bat (
    call venv\Scripts\activate.bat
    start "FastAPI Gateway" cmd /k "python -m uvicorn main:app --host 127.0.0.1 --port 8000 --reload"
) else (
    echo FastAPI virtual environment not found. Please run setup first.
    pause
    exit /b 1
)
cd ..

echo [2/4] Starting Next.js Frontend (Port 3002)...
cd web
if exist node_modules (
    start "Next.js Frontend" cmd /k "npm run dev"
) else (
    echo Next.js dependencies not installed. Please run 'npm install' first.
    pause
    exit /b 1
)
cd ..

echo [3/4] Starting Go Backend (Port 8080)...
if exist bin\sprintd.exe (
    start "Go Backend" cmd /k "bin\sprintd.exe"
) else (
    echo Go backend not built. Please build it first.
    pause
    exit /b 1
)

echo [4/4] Starting Grafana (Port 3000)...
if exist grafana-compose.yml (
    start "Grafana" cmd /k "docker-compose -f grafana-compose.yml up"
) else (
    echo Grafana configuration not found.
)

echo.
echo ===========================================
echo   All Services Started!
echo ===========================================
echo FastAPI Gateway: http://localhost:8000
echo Next.js Frontend: http://localhost:3002
echo Go Backend: http://localhost:8080
echo Grafana: http://localhost:3000
echo.
echo Press any key to stop all services...
pause >nul

echo Stopping all services...
taskkill /FI "WINDOWTITLE eq FastAPI Gateway*" /T /F >nul 2>&1
taskkill /FI "WINDOWTITLE eq Next.js Frontend*" /T /F >nul 2>&1
taskkill /FI "WINDOWTITLE eq Go Backend*" /T /F >nul 2>&1
taskkill /FI "WINDOWTITLE eq Grafana*" /T /F >nul 2>&1

echo All services stopped.
pause
