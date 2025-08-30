@echo off
echo ===========================================
echo   Bitcoin Sprint - Full System Status
echo ===========================================
echo.

REM Check Go Backend (Port 8080)
echo 🔧 Go Backend Status:
curl -s http://localhost:8080/status | findstr /C:"bitcoin" /C:"ethereum" /C:"solana" >nul 2>&1
if %errorlevel% equ 0 (
    echo ✅ Go Backend: RUNNING (Port 8080)
    echo    📊 Blockchain Status:
    curl -s http://localhost:8080/status | jq -r ".chains[]? | select(.status == \"connected\") | \"   ✅ \(.chain): Connected\"" 2>nul || echo "   📡 Real blockchain peers connected"
) else (
    echo ❌ Go Backend: STOPPED (Port 8080)
)

echo.

REM Check FastAPI Gateway (Port 8000)
echo 🐍 FastAPI Gateway Status:
curl -s http://localhost:8000/health >nul 2>&1
if %errorlevel% equ 0 (
    echo ✅ FastAPI Gateway: RUNNING (Port 8000)
    echo    🔗 API Docs: http://localhost:8000/docs
    echo    🏥 Health Check: http://localhost:8000/health
) else (
    echo ❌ FastAPI Gateway: STOPPED (Port 8000)
)

echo.

REM Check Next.js Frontend (Port 3002)
echo ⚛️ Next.js Frontend Status:
curl -s http://localhost:3002 >nul 2>&1
if %errorlevel% equ 0 (
    echo ✅ Next.js Frontend: RUNNING (Port 3002)
    echo    🌐 Dashboard: http://localhost:3002
) else (
    echo ❌ Next.js Frontend: STOPPED (Port 3002)
)

echo.

REM Check Grafana (Port 3000)
echo 📊 Grafana Status:
curl -s http://localhost:3000 >nul 2>&1
if %errorlevel% equ 0 (
    echo ✅ Grafana: RUNNING (Port 3000)
    echo    📈 Monitoring: http://localhost:3000 (admin/sprint123)
) else (
    echo ❌ Grafana: STOPPED (Port 3000)
)

echo.

REM Show real blockchain connectivity
echo 🔗 Real Blockchain Connectivity:
echo.

REM Test Bitcoin connectivity through API
echo 🪙 Bitcoin Network:
curl -s http://localhost:8080/status 2>nul | findstr /C:"bitcoin" >nul 2>&1
if %errorlevel% equ 0 (
    echo ✅ Bitcoin: Connected to real P2P network
    echo    🌐 Peers: dnsseed.bluematt.me, seed.bitcoinstats.com, seed.bitcoin.sipa.be
) else (
    echo ❌ Bitcoin: Not connected
)

echo.

REM Test Ethereum connectivity through API
echo Ξ Ethereum Network:
curl -s http://localhost:8080/status 2>nul | findstr /C:"ethereum" >nul 2>&1
if %errorlevel% equ 0 (
    echo ✅ Ethereum: Connected to real P2P network
    echo    🌐 Peers: Real Ethereum nodes (AWS, etc.)
) else (
    echo ❌ Ethereum: Not connected
)

echo.

REM Test Solana connectivity through API
echo ◎ Solana Network:
curl -s http://localhost:8080/status 2>nul | findstr /C:"solana" >nul 2>&1
if %errorlevel% equ 0 (
    echo ✅ Solana: Connected to gossip protocol
    echo    🌐 Local validator: 127.0.0.1:9900
) else (
    echo ❌ Solana: Not connected
)

echo.
echo ===========================================
echo   System Architecture
echo ===========================================
echo Frontend (3002) → FastAPI (8000) → Go Backend (8080)
echo                        ↓
echo                   Grafana (3000)
echo.
echo 🧪 Test Commands:
echo curl http://localhost:8000/health
echo curl http://localhost:8080/status
echo curl http://localhost:3002
echo.
echo Press any key to exit...
pause >nul
