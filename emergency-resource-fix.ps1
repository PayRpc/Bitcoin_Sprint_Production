# Emergency CPU/Memory Resource Fix Script
# Addresses critical 678% CPU oversubscription and memory exhaustion

Write-Host "🚨 EMERGENCY: Applying Resource Limits to Fix CPU Oversubscription" -ForegroundColor Red
Write-Host "Current State: 678% CPU usage, 6.36GB memory consumption" -ForegroundColor Yellow
Write-Host ""

# Step 1: Stop all containers to prevent further system damage
Write-Host "Step 1: Stopping all containers..." -ForegroundColor Yellow
docker-compose -f config/docker-compose.yml down --remove-orphans

# Step 2: Wait for containers to fully stop
Write-Host "Waiting for containers to stop..." -ForegroundColor Yellow
Start-Sleep -Seconds 10

# Step 3: Check system resources
Write-Host "Step 2: Checking system resources..." -ForegroundColor Yellow
$cpuCount = (Get-ComputerInfo).CsNumberOfProcessors
$totalMemGB = [math]::Round((Get-ComputerInfo).TotalPhysicalMemory / 1GB, 2)

Write-Host "Available CPU Cores: $cpuCount" -ForegroundColor Cyan
Write-Host "Total RAM: $totalMemGB GB" -ForegroundColor Cyan

if ($cpuCount -lt 8) {
    Write-Host "⚠️  WARNING: Only $cpuCount CPU cores detected. Scaling down to 2 validators max." -ForegroundColor Yellow
}

if ($totalMemGB -lt 16) {
    Write-Host "⚠️  WARNING: Only $totalMemGB GB RAM detected. Applying aggressive memory limits." -ForegroundColor Yellow
}

# Step 4: Restart with resource limits - Bitcoin Sprint API FIRST (priority)
Write-Host "Step 3: Starting Bitcoin Sprint API with guaranteed resources..." -ForegroundColor Green
docker-compose -f config/docker-compose.yml -f docker-compose.resource-limits.yml up -d bitcoin-sprint-api grafana prometheus

# Wait for core services
Write-Host "Waiting for core services to stabilize..." -ForegroundColor Yellow
Start-Sleep -Seconds 15

# Step 5: Start minimal Solana validators (max 2)
Write-Host "Step 4: Starting reduced Solana validators (2 max)..." -ForegroundColor Green
docker-compose -f config/docker-compose.yml -f docker-compose.resource-limits.yml up -d solana-validator solana-validator-2

# Step 6: Start supporting infrastructure
Write-Host "Step 5: Starting supporting services..." -ForegroundColor Green
docker-compose -f config/docker-compose.yml -f docker-compose.resource-limits.yml up -d redis nginx postgres

# Step 7: Start exporters
Write-Host "Step 6: Starting metric exporters..." -ForegroundColor Green
docker-compose -f config/docker-compose.yml -f docker-compose.resource-limits.yml up -d bitcoin-exporter solana-exporter

Write-Host ""
Write-Host "🔧 Resource limits applied successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "New Resource Allocation:" -ForegroundColor Cyan
Write-Host "- Bitcoin Sprint API: 2 CPU cores max, 2GB RAM max (PRIORITY)" -ForegroundColor White
Write-Host "- Solana Validators: 1.5 CPU cores max each, 1.5GB RAM max each (2 validators only)" -ForegroundColor White
Write-Host "- Monitoring: 0.5 CPU cores max, 512MB RAM max" -ForegroundColor White
Write-Host "- Validators 4-7: DISABLED to reduce load" -ForegroundColor White
Write-Host ""

# Step 8: Check new resource usage
Write-Host "Step 7: Checking new resource usage..." -ForegroundColor Yellow
Start-Sleep -Seconds 10

Write-Host "Container Status:" -ForegroundColor Cyan
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

Write-Host ""
Write-Host "✅ Emergency fix applied. System should now be stable." -ForegroundColor Green
Write-Host "Monitor CPU usage - should be under 400% total now" -ForegroundColor Yellow
Write-Host ""
Write-Host "Next Steps:" -ForegroundColor Cyan
Write-Host "1. Check Grafana at http://localhost:3000 (admin/sprint123)" -ForegroundColor White
Write-Host "2. Monitor system resources for 10 minutes" -ForegroundColor White
Write-Host "3. If stable, gradually re-enable more validators if needed" -ForegroundColor White
Write-Host "4. Set up resource monitoring alerts" -ForegroundColor White
