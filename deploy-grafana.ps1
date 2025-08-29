# Grafana Dashboard Deployment Script
# This script helps deploy Bitcoin Sprint dashboards to Grafana

Write-Host "🚀 Bitcoin Sprint Grafana Dashboard Deployment" -ForegroundColor Green
Write-Host "==============================================" -ForegroundColor Green

# Check if Grafana is running
Write-Host "`n1. Checking Grafana status..." -ForegroundColor Yellow
try {
    Invoke-WebRequest -Uri "http://localhost:3000/api/health" -TimeoutSec 5 | Out-Null
    Write-Host "✅ Grafana is running" -ForegroundColor Green
} catch {
    Write-Host "❌ Grafana is not accessible at http://localhost:3000" -ForegroundColor Red
    Write-Host "Please ensure Grafana is running and accessible" -ForegroundColor Yellow
    exit 1
}

# Check dashboard files
Write-Host "`n2. Checking dashboard files..." -ForegroundColor Yellow
$dashboardDir = ".\grafana\dashboards"
$dashboards = Get-ChildItem -Path $dashboardDir -Filter "*.json"

if ($dashboards.Count -eq 0) {
    Write-Host "❌ No dashboard files found in $dashboardDir" -ForegroundColor Red
    exit 1
}

Write-Host "✅ Found $($dashboards.Count) dashboard files:" -ForegroundColor Green
foreach ($dashboard in $dashboards) {
    Write-Host "   - $($dashboard.Name)" -ForegroundColor Cyan
}

# Validate JSON syntax
Write-Host "`n3. Validating dashboard JSON syntax..." -ForegroundColor Yellow
$validCount = 0
foreach ($dashboard in $dashboards) {
    try {
        $jsonContent = Get-Content -Path $dashboard.FullName -Raw
        ConvertFrom-Json $jsonContent | Out-Null
        Write-Host "✅ $($dashboard.Name) - Valid JSON" -ForegroundColor Green
        $validCount++
    } catch {
        Write-Host "❌ $($dashboard.Name) - Invalid JSON: $($_.Exception.Message)" -ForegroundColor Red
    }
}

if ($validCount -eq $dashboards.Count) {
    Write-Host "`n🎉 All dashboards are valid!" -ForegroundColor Green
} else {
    Write-Host "`n⚠️  Some dashboards have validation errors" -ForegroundColor Yellow
}

# Deployment instructions
Write-Host "`n4. Deployment Instructions:" -ForegroundColor Yellow
Write-Host "For Docker deployment, ensure your docker-compose.yml includes:" -ForegroundColor White
Write-Host "  - Volume mount: ./grafana/dashboards:/var/lib/grafana/dashboards" -ForegroundColor Cyan
Write-Host "  - Dashboard provisioning enabled" -ForegroundColor Cyan

Write-Host "`nFor manual deployment:" -ForegroundColor White
Write-Host "  1. Copy dashboard files to Grafana's dashboard directory" -ForegroundColor Cyan
Write-Host "  2. Restart Grafana service" -ForegroundColor Cyan
Write-Host "  3. Access dashboards at http://localhost:3000" -ForegroundColor Cyan

Write-Host "`n📊 Available Dashboards:" -ForegroundColor Green
Write-Host "  - Sprint Overview: System-wide metrics and performance" -ForegroundColor White
Write-Host "  - Solana Monitoring: Comprehensive Solana network metrics" -ForegroundColor White
Write-Host "  - API Testing: Request/response monitoring and testing" -ForegroundColor White

Write-Host "`n✅ Dashboard deployment preparation complete!" -ForegroundColor Green
