@echo off
echo Starting Bitcoin Sprint + Solana Monitoring Stack...
echo ==================================================

echo.
echo Step 1: Creating Docker network...
docker network create sprint-net 2>nul || echo Network already exists

echo.
echo Step 2: Starting monitoring services...
docker compose -f docker-compose.monitoring.yml up -d

echo.
echo Step 3: Waiting for services to be healthy...
timeout /t 10 /nobreak >nul

echo.
echo Step 4: Checking service status...
docker compose -f docker-compose.monitoring.yml ps

echo.
echo Step 5: Testing endpoints...
echo.
echo Prometheus: http://localhost:9090
echo Grafana:    http://localhost:3000 (admin/sprint123)
echo Solana Exporter: http://localhost:8080/metrics

echo.
echo To import the dashboard:
echo 1. Open Grafana at http://localhost:3000
echo 2. Go to Dashboards → Import
echo 3. Upload: monitoring/grafana/dashboards/bitcoin-sprint-solana-updated.json

echo.
echo Monitoring stack startup complete!
pause
