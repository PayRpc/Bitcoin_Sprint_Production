# Quick Solana Load Test Demo
Write-Host "🚀 Quick Solana Load Testing Demo" -ForegroundColor Cyan
Write-Host "=" * 40 -ForegroundColor Cyan

# Check current status
Write-Host "`n📊 Current Solana Status:" -ForegroundColor Yellow
curl -s "http://localhost:9091/api/v1/query?query=solana_slot_height" | findstr "value"
curl -s "http://localhost:9091/api/v1/query?query=solana_tps" | findstr "value"

# Start load testing tool
Write-Host "`n⚡ Starting Load Testing Tool..." -ForegroundColor Green
docker-compose --profile load-testing up -d solana-bench-tps

# Run a quick load test
Write-Host "`n🔥 Running Quick Load Test (10 seconds, 100 tx)..." -ForegroundColor Yellow
docker exec solana-bench-tps solana-bench-tps --entrypoint http://solana-validator:8899 --duration 10 --tx-count 100 --threads 2

# Check results
Write-Host "`n📈 Post-Load Test Metrics:" -ForegroundColor Green
curl -s "http://localhost:9091/api/v1/query?query=solana_slot_height" | findstr "value"
curl -s "http://localhost:9091/api/v1/query?query=solana_tps" | findstr "value"

Write-Host "`n✅ Load test completed!" -ForegroundColor Green
Write-Host "Check your Grafana dashboard at: http://localhost:3000" -ForegroundColor Cyan
Write-Host "You should see TPS spikes during the test!" -ForegroundColor Yellow
