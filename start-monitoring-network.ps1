# Ensure sprint-net network exists
Write-Host "Creating sprint-net network if it doesn't exist..." -ForegroundColor Cyan
docker network create sprint-net 2>$null

# Remove any existing containers to avoid conflicts
Write-Host "Stopping existing monitoring stack..." -ForegroundColor Cyan
docker compose -f monitoring-compose.yml down

# Start the monitoring stack with clean volumes
Write-Host "Starting monitoring stack with clean configuration..." -ForegroundColor Cyan
docker compose -f monitoring-compose.yml up -d --force-recreate --remove-orphans

# Check if the containers are running
Write-Host "Checking container status..." -ForegroundColor Cyan
docker ps --filter "network=sprint-net"

# Wait for Prometheus to start
Write-Host "Waiting for Prometheus to initialize (15 seconds)..." -ForegroundColor Cyan
Start-Sleep -Seconds 15

# Check Prometheus targets
Write-Host "Testing Prometheus targets..." -ForegroundColor Cyan
$prometheusContainer = docker ps --filter "name=sprint-prometheus" --format "{{.Names}}"
docker exec -it $prometheusContainer sh -c "curl -s http://localhost:9090/api/v1/targets | grep -E 'endpoint|state'"

# Test resolution from inside Prometheus container
Write-Host "Testing DNS resolution from Prometheus container..." -ForegroundColor Cyan
docker exec -it $prometheusContainer sh -c "getent hosts bitcoin_sprint; getent hosts solana-exporter"

Write-Host "Monitoring stack setup complete. Access Grafana at http://localhost:3000" -ForegroundColor Green
Write-Host "Default login: admin / sprint123" -ForegroundColor Yellow
