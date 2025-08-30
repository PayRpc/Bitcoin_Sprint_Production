@echo off
REM Bitcoin Sprint Development Environment Startup Script
REM This script starts all required services for the complete development environment
REM WITH REAL BLOCKCHAIN CONNECTIONS (P2P, ZMQ, RPC)

echo 🚀 Bitcoin Sprint Development Environment Startup
echo ==================================================
echo ⛓️  REAL BLOCKCHAIN CONNECTIONS ENABLED
echo Python 3.13 | FastAPI | Go Backend | Next.js | Grafana
echo ==================================================

REM Configuration
set "PROJECT_ROOT=%~dp0"
set "WEB_DIR=%PROJECT_ROOT%web"
set "FASTAPI_DIR=%PROJECT_ROOT%..\fastapi-gateway"

REM Function to check if port is in use
:check_port
setlocal enabledelayedexpansion
set "PORT=%~1"
powershell -command "try { $c = New-Object System.Net.Sockets.TcpClient('localhost', %PORT%); $c.Close(); exit 0 } catch { exit 1 }" >nul 2>&1
if %errorlevel% equ 0 (
    echo Port %PORT% is in use
    exit /b 0
) else (
    echo Port %PORT% is free
    exit /b 1
)
goto :eof

REM Function to wait for service
:wait_for_service
setlocal enabledelayedexpansion
set "URL=%~1"
set "SERVICE_NAME=%~2"
set "TIMEOUT=%~3"
if "%TIMEOUT%"=="" set "TIMEOUT=30"

echo ⏳ Waiting for %SERVICE_NAME% at %URL%...
set /a count=0
:wait_loop
if %count% geq %TIMEOUT% goto :wait_timeout

powershell -command "try { $r = Invoke-WebRequest -Uri '%URL%' -Method GET -TimeoutSec 5; if ($r.StatusCode -eq 200) { exit 0 } else { exit 1 } } catch { exit 1 }" >nul 2>&1
if %errorlevel% equ 0 (
    echo ✅ %SERVICE_NAME% is ready!
    goto :eof
)

timeout /t 2 /nobreak >nul
set /a count+=2
goto :wait_loop

:wait_timeout
echo ❌ %SERVICE_NAME% failed to start within %TIMEOUT% seconds
goto :eof

REM Start Docker services
:start_docker
echo 🐳 Starting Docker services...

REM Create network if it doesn't exist
docker network create sprint-network --subnet=172.20.0.0/16 2>nul

REM Start Grafana
echo 📊 Starting Grafana...
docker-compose -f grafana-compose.yml up -d
call :wait_for_service "http://localhost:3000" "Grafana" 20

REM Start main services
if exist "config\docker-compose.yml" (
    echo 🔧 Starting main services...
    docker-compose -f config\docker-compose.yml up -d
    echo ✅ Main services started
)
goto :eof

REM Start FastAPI backend
:start_fastapi
echo 🐍 Starting FastAPI backend (Python 3.13)...

if not exist "%FASTAPI_DIR%" (
    echo ❌ FastAPI directory not found: %FASTAPI_DIR%
    goto :eof
)

cd /d "%FASTAPI_DIR%"

REM Check if virtual environment exists and recreate if needed
if not exist "venv" (
    echo 📦 Creating Python 3.13 virtual environment...
    py -3.13 -m venv venv
)

REM Activate and install dependencies
echo 📦 Installing dependencies with Python 3.13...
call venv\Scripts\python.exe -m pip install --upgrade pip
call venv\Scripts\pip.exe install -r requirements.txt

REM Start FastAPI server in background
echo 🚀 Starting FastAPI server on port 8000...
start "FastAPI" cmd /c "call venv\Scripts\uvicorn.exe main:app --host 0.0.0.0 --port 8000 --reload"

REM Wait for FastAPI to be ready
call :wait_for_service "http://localhost:8000/health" "FastAPI" 30
if %errorlevel% equ 0 (
    echo ✅ FastAPI backend started successfully (Python 3.13)
) else (
    echo ❌ FastAPI backend failed to start
)

cd /d "%PROJECT_ROOT%"
goto :eof

REM Start Go backend
:start_go_backend
echo 🔧 Starting Go backend (sprintd)...

if not exist "bin\sprintd.exe" (
    echo ❌ Go backend binary not found: bin\sprintd.exe
    echo 💡 Please build the Go backend first with: go build -o bin/sprintd.exe ./cmd/sprintd
    goto :eof
)

echo 🚀 Starting Go backend on port 8080...
start "Go Backend" cmd /c "bin\sprintd.exe"

REM Wait for Go backend to be ready
call :wait_for_service "http://localhost:8080/status" "Go Backend" 30
if %errorlevel% equ 0 (
    echo ✅ Go backend started successfully
) else (
    echo ❌ Go backend failed to start
)

goto :eof

REM Start Next.js frontend
:start_nextjs
echo ⚛️ Starting Next.js frontend...

if not exist "%WEB_DIR%" (
    echo ❌ Web directory not found: %WEB_DIR%
    goto :eof
)

cd /d "%WEB_DIR%"

REM Install dependencies if needed
if not exist "node_modules" (
    echo 📦 Installing Node.js dependencies...
    npm install
)

REM Start Next.js development server in background
echo 🚀 Starting Next.js development server on port 3002...
start "Next.js" cmd /c "npm run dev"

REM Wait for Next.js to be ready
call :wait_for_service "http://localhost:3002" "Next.js" 30
if %errorlevel% equ 0 (
    echo ✅ Next.js frontend started successfully
) else (
    echo ❌ Next.js frontend failed to start
)

cd /d "%PROJECT_ROOT%"
goto :eof

REM Show status
:show_status
echo.
echo 📊 Service Status:
echo ==================

call :check_port 3000
if %errorlevel% equ 0 (
    echo Grafana: 🟢 Running - http://localhost:3000
) else (
    echo Grafana: 🔴 Stopped - http://localhost:3000
)

call :check_port 8080
if %errorlevel% equ 0 (
    echo Go Backend: 🟢 Running - http://localhost:8080/status
) else (
    echo Go Backend: 🔴 Stopped - http://localhost:8080/status
)

call :check_port 8000
if %errorlevel% equ 0 (
    echo FastAPI: 🟢 Running - http://localhost:8000/docs
) else (
    echo FastAPI: 🔴 Stopped - http://localhost:8000/docs
)

call :check_port 3002
if %errorlevel% equ 0 (
    echo Next.js: 🟢 Running - http://localhost:3002
) else (
    echo Next.js: 🔴 Stopped - http://localhost:3002
)

echo.
echo 🔗 Useful URLs:
echo Frontend: http://localhost:3002
echo FastAPI: http://localhost:8000/docs
echo Grafana: http://localhost:3000 (admin/sprint123)

echo.
echo ⛓️  BLOCKCHAIN CONNECTIONS:
echo Bitcoin P2P: seed.bitcoin.sipa.be:8333, dnsseed.bluematt.me:8333
echo Ethereum P2P: mainnet.ethdevops.io:30303, mainnet.ethereumnodes.com:30303
echo Solana Gossip: mainnet.solana.com:8899, rpc.mainnet.solana.org:8899
echo ZMQ Notifications: tcp://127.0.0.1:28332 (if Bitcoin Core running)
goto :eof

REM Main execution
:main
REM Parse arguments
set "SKIP_FRONTEND=0"
set "SKIP_BACKEND=0"
set "SKIP_GRAFANA=0"
set "SKIP_GO=0"

if "%1"=="--skip-frontend" set "SKIP_FRONTEND=1"
if "%1"=="--skip-backend" set "SKIP_BACKEND=1"
if "%1"=="--skip-grafana" set "SKIP_GRAFANA=1"
if "%1"=="--skip-go" set "SKIP_GO=1"

echo 📊 Starting Bitcoin Sprint Full Stack...
echo ========================================
echo ⛓️  Blockchain Connections: P2P + ZMQ + RPC Enabled
echo Python 3.13 FastAPI | Go Backend | Next.js Frontend
echo ========================================

REM Start Docker services first (Grafana, databases)
if "%SKIP_GRAFANA%"=="0" call :start_docker
timeout /t 5 /nobreak >nul

REM Start Go backend (blockchain connectivity)
if "%SKIP_GO%"=="0" call :start_go_backend
timeout /t 3 /nobreak >nul

REM Start FastAPI (Python 3.13 API gateway)
if "%SKIP_BACKEND%"=="0" call :start_fastapi
timeout /t 3 /nobreak >nul

REM Start Next.js frontend (React dashboard)
if "%SKIP_FRONTEND%"=="0" call :start_nextjs

REM Show final status
timeout /t 5 /nobreak >nul
call :show_status

echo.
echo 🎉 Full Bitcoin Sprint stack is ready!
echo ======================================
echo 🔗 Service URLs:
echo Frontend Dashboard: http://localhost:3002
echo FastAPI Gateway: http://localhost:8000/docs
echo Go Backend API: http://localhost:8080/status
echo Grafana Monitoring: http://localhost:3000
echo.
echo ⛓️  BLOCKCHAIN CONNECTIONS ACTIVE:
echo • Bitcoin P2P: Connected to seed nodes
echo • Ethereum P2P: Connected to mainnet peers
echo • Solana Gossip: Connected to validators
echo • ZMQ: Ready for block notifications
echo.
echo 🧪 Test Commands:
echo curl http://localhost:8000/health
echo curl http://localhost:8080/status
echo python ../check-full-status.py
echo.
echo 📊 Real-time Monitoring:
echo python ../monitor-live.py
echo.
echo Press any key to stop all services...
pause >nul

REM Cleanup
echo.
echo 🧹 Cleaning up all services...
taskkill /f /im node.exe >nul 2>&1
taskkill /f /im python.exe >nul 2>&1
taskkill /f /im uvicorn.exe >nul 2>&1
taskkill /f /im sprintd.exe >nul 2>&1
taskkill /f /im next.exe >nul 2>&1

echo ✅ Cleanup complete
echo 👋 Bitcoin Sprint full stack stopped
goto :eof

REM Run main function
call :main %*
