@echo off
echo 🚀 Starting Bitcoin Sprint + Solana Monitoring Stack
echo ==================================================

echo.
echo Step 1: Creating Docker network...
docker network create sprint-net 2>nul
if %errorlevel% equ 0 (
    echo ✅ Network created
) else (
    echo ℹ️  Network already exists
)

echo.
echo Step 2: Starting monitoring services...
docker compose -f docker-compose.monitoring.yml up -d

echo.
echo Step 3: Waiting for services to start...
timeout /t 15 /nobreak >nul

echo.
echo Step 4: Checking service status...
docker compose -f docker-compose.monitoring.yml ps

echo.
echo Step 5: Testing connectivity...
echo.
echo Testing Solana Exporter:
curl -s http://localhost:8080/metrics | findstr solana_slot_height >nul
if %errorlevel% equ 0 (
    echo ✅ Solana Exporter: Metrics available
) else (
    echo ❌ Solana Exporter: No metrics found
)

echo.
echo Testing Prometheus:
curl -s http://localhost:9090/-/healthy >nul
if %errorlevel% equ 0 (
    echo ✅ Prometheus: Healthy
) else (
    echo ❌ Prometheus: Not responding
)

echo.
echo Testing Grafana:
curl -s http://localhost:3000/api/health >nul
if %errorlevel% equ 0 (
    echo ✅ Grafana: Healthy
) else (
    echo ❌ Grafana: Not responding
)

echo.
echo 📊 Service URLs:
echo    Prometheus: http://localhost:9090
echo    Grafana:    http://localhost:3000 (admin/sprint123)
echo    Solana Exp: http://localhost:8080/metrics

echo.
echo 📋 Next Steps:
echo 1. Check http://localhost:9090/targets
echo 2. Import dashboard in Grafana
echo 3. Verify Solana metrics are flowing

echo.
echo ✨ Setup complete! Press any key to exit...
pause >nul
