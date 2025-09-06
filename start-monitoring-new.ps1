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

Write-Host "`n3. Waiting for services to be healthy..." -ForegroundColor Yellow
Start-Sleep 10

Write-Host "`n4. Checking service status..." -ForegroundColor Yellow
docker compose -f docker-compose.monitoring.yml ps

Write-Host "`n5. Service URLs:" -ForegroundColor Green
Write-Host "   Prometheus: http://localhost:9090" -ForegroundColor White
Write-Host "   Grafana:    http://localhost:3000 (admin/sprint123)" -ForegroundColor White
Write-Host "   Solana Exporter: http://localhost:8080/metrics" -ForegroundColor White

Write-Host "`n📋 Next Steps:" -ForegroundColor Cyan
Write-Host "1. Open Grafana: http://localhost:3000" -ForegroundColor White
Write-Host "2. Import dashboard: monitoring/grafana/dashboards/bitcoin-sprint-solana-updated.json" -ForegroundColor White
Write-Host "3. Check targets: http://localhost:9090/targets" -ForegroundColor White

Write-Host "`n✨ Monitoring stack startup complete!" -ForegroundColor Green
Read-Host "Press Enter to exit"
