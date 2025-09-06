Write-Host "🚀 Starting Bitcoin Sprint + Solana Monitoring Stack" -ForegroundColor Cyan
Write-Host "==================================================" -ForegroundColor White

Write-Host "`n1. Creating Docker network..." -ForegroundColor Yellow
docker network create sprint-net 2>$null
if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Network created" -ForegroundColor Green
} else {
    Write-Host "ℹ️  Network already exists" -ForegroundColor Blue
}

Write-Host "`n2. Starting monitoring services..." -ForegroundColor Yellow
docker compose -f docker-compose.monitoring.yml up -d

Write-Host "`n3. Waiting for services to start..." -ForegroundColor Yellow
Start-Sleep 15

Write-Host "`n4. Checking service status..." -ForegroundColor Yellow
docker compose -f docker-compose.monitoring.yml ps

Write-Host "`n5. Testing connectivity..." -ForegroundColor Yellow

# Test Solana Exporter
Write-Host "`n   Testing Solana Exporter:" -ForegroundColor White
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/metrics" -TimeoutSec 10
    if ($response.Content -match "solana_slot_height") {
        Write-Host "   ✅ Solana Exporter: Metrics available" -ForegroundColor Green
    } else {
        Write-Host "   ❌ Solana Exporter: No Solana metrics found" -ForegroundColor Red
    }
} catch {
    Write-Host "   ❌ Solana Exporter: Not responding" -ForegroundColor Red
}

# Test Prometheus
Write-Host "   Testing Prometheus:" -ForegroundColor White
try {
    $response = Invoke-WebRequest -Uri "http://localhost:9090/-/healthy" -TimeoutSec 5
    Write-Host "   ✅ Prometheus: Healthy" -ForegroundColor Green
} catch {
    Write-Host "   ❌ Prometheus: Not responding" -ForegroundColor Red
}

# Test Grafana
Write-Host "   Testing Grafana:" -ForegroundColor White
try {
    $response = Invoke-WebRequest -Uri "http://localhost:3000/api/health" -TimeoutSec 5
    Write-Host "   ✅ Grafana: Healthy" -ForegroundColor Green
} catch {
    Write-Host "   ❌ Grafana: Not responding" -ForegroundColor Red
}

Write-Host "`n📊 Service URLs:" -ForegroundColor Green
Write-Host "   Prometheus: http://localhost:9090" -ForegroundColor White
Write-Host "   Grafana:    http://localhost:3000 (admin/sprint123)" -ForegroundColor White
Write-Host "   Solana Exp: http://localhost:8080/metrics" -ForegroundColor White

Write-Host "`n📋 Next Steps:" -ForegroundColor Cyan
Write-Host "1. Check http://localhost:9090/targets" -ForegroundColor White
Write-Host "2. Import dashboard in Grafana" -ForegroundColor White
Write-Host "3. Verify Solana metrics are flowing" -ForegroundColor White

Write-Host "`n✨ Setup complete!" -ForegroundColor Green
Read-Host "Press Enter to exit"
