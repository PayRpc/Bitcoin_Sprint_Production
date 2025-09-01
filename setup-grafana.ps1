# Bitcoin Sprint - Grafana Setup Script
Write-Host "🚀 Setting up Bitcoin Sprint Grafana Dashboard..." -ForegroundColor Green

# Stop any existing Grafana containers
Write-Host "Stopping existing containers..." -ForegroundColor Yellow
docker stop grafana-live, bitcoin-sprint-grafana 2>$null
docker rm grafana-live, bitcoin-sprint-grafana 2>$null

# Start fresh Grafana with proper credentials
Write-Host "Starting Grafana with admin/admin123 credentials..." -ForegroundColor Cyan
docker run -d `
    --name bitcoin-sprint-grafana `
    -p 3000:3000 `
    -e GF_SECURITY_ADMIN_USER=admin `
    -e GF_SECURITY_ADMIN_PASSWORD=admin123 `
    -e GF_USERS_ALLOW_SIGN_UP=false `
    -e GF_USERS_DEFAULT_THEME=dark `
    -e GF_INSTALL_PLUGINS=grafana-piechart-panel `
    grafana/grafana:latest

# Wait for startup
Write-Host "Waiting for Grafana to start..." -ForegroundColor Yellow
Start-Sleep -Seconds 20

# Test connection
Write-Host "Testing Grafana connection..." -ForegroundColor Cyan
try {
    $response = Invoke-WebRequest -Uri "http://localhost:3000" -UseBasicParsing -TimeoutSec 10
    if ($response.StatusCode -eq 200) {
        Write-Host "✅ Grafana is running successfully!" -ForegroundColor Green
        Write-Host "📍 URL: http://localhost:3000" -ForegroundColor White
        Write-Host "👤 Username: admin" -ForegroundColor White
        Write-Host "🔑 Password: admin123" -ForegroundColor White
    }
} catch {
    Write-Host "❌ Grafana startup failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Check our API
Write-Host "Testing Bitcoin Sprint API..." -ForegroundColor Cyan
try {
    $apiResponse = Invoke-WebRequest -Uri "http://localhost:8080/health" -UseBasicParsing -TimeoutSec 5
    if ($apiResponse.StatusCode -eq 200) {
        Write-Host "✅ Bitcoin Sprint API is online!" -ForegroundColor Green
    }
} catch {
    Write-Host "⚠️ API not responding - make sure to start the Go server" -ForegroundColor Yellow
}

Write-Host "🎯 Setup Complete! Access Grafana at http://localhost:3000" -ForegroundColor Green
