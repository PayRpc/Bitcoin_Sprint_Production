# Enterprise API Key Generator for Grafana Testing
# Generates and validates enterprise API keys for use in monitoring dashboards

Write-Host "=== ENTERPRISE API KEY GENERATOR ===" -ForegroundColor Cyan

# Generate a enterprise-level API key (simulated for demo)
$enterpriseKey = "ent_" + [System.Guid]::NewGuid().ToString().Replace('-', '').Substring(0, 24)

Write-Host "✅ Generated Enterprise API Key:" -ForegroundColor Green
Write-Host "   $enterpriseKey" -ForegroundColor Yellow
Write-Host ""

# Test the key with available endpoints
Write-Host "🧪 TESTING API KEY WITH AVAILABLE ENDPOINTS:" -ForegroundColor Cyan
$headers = @{ "Authorization" = "Bearer $enterpriseKey" }

# Test metrics endpoint
try {
    Write-Host "Testing metrics endpoint..." -ForegroundColor Gray
    $response = Invoke-WebRequest -Uri "http://localhost:8081/metrics" -Headers $headers -TimeoutSec 10
    Write-Host "✅ Metrics endpoint: Status $($response.StatusCode)" -ForegroundColor Green
    $metricsWorking = $true
} catch {
    Write-Host "❌ Metrics endpoint failed: $($_.Exception.Message)" -ForegroundColor Red
    $metricsWorking = $false
}

# Test if main API server is running
try {
    Write-Host "Testing main API server..." -ForegroundColor Gray
    $response = Invoke-WebRequest -Uri "http://localhost:8080/health" -Headers $headers -TimeoutSec 5 -ErrorAction SilentlyContinue
    Write-Host "✅ Main API server: Status $($response.StatusCode)" -ForegroundColor Green
    $mainApiWorking = $true
} catch {
    Write-Host "⚠️  Main API server not responding (this is normal for metrics-only setup)" -ForegroundColor Yellow
    $mainApiWorking = $false
}

Write-Host ""
Write-Host "📊 GRAFANA INTEGRATION GUIDE:" -ForegroundColor Cyan
Write-Host "1. Copy the API key above" -ForegroundColor White
Write-Host "2. In Grafana dashboard, set the 'api_key' variable to:" -ForegroundColor White
Write-Host "   $enterpriseKey" -ForegroundColor Yellow
Write-Host "3. Use this key in JSON API data source headers:" -ForegroundColor White
Write-Host "   Authorization: Bearer $enterpriseKey" -ForegroundColor Yellow
Write-Host ""

# Create a test configuration file
$config = @{
    enterprise_api_key = $enterpriseKey
    endpoints = @{
        metrics = "http://localhost:8081/metrics"
        health = "http://localhost:8080/health"
        status = "http://localhost:8080/status"
        analytics = "http://localhost:8080/analytics"
    }
    grafana_config = @{
        datasource_url = "http://host.docker.internal:8081"
        auth_header = "Bearer $enterpriseKey"
        refresh_interval = "30s"
    }
    test_results = @{
        metrics_endpoint = $metricsWorking
        main_api_endpoint = $mainApiWorking
        generated_at = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    }
}

# Save configuration
$config | ConvertTo-Json -Depth 3 | Out-File -FilePath "enterprise-api-config.json" -Encoding UTF8
Write-Host "💾 Configuration saved to: enterprise-api-config.json" -ForegroundColor Green

Write-Host ""
Write-Host "🚀 READY FOR GRAFANA TESTING!" -ForegroundColor Green
Write-Host "Use the generated API key in your Grafana dashboard variables." -ForegroundColor White

# Export the key for easy access
Write-Host ""
Write-Host "🔑 EXPORT FOR EASY USE:" -ForegroundColor Magenta
Write-Host "`$env:ENTERPRISE_API_KEY = '$enterpriseKey'" -ForegroundColor Yellow
Write-Host ""
Write-Host "=== ENTERPRISE API KEY GENERATION COMPLETE ===" -ForegroundColor Cyan
