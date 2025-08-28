# Comprehensive API Test Script
Write-Host "🚀 Starting Bitcoin Sprint API Service for comprehensive testing..."

# Start the service in background
$serviceJob = Start-Job -ScriptBlock {
    Set-Location "c:\Projects\Bitcoin-Sprint-5\BItcoin_Sprint"
    $env:RUST_LOG = "info"
    & cargo run --release
}

# Wait for service to start
Start-Sleep -Seconds 5

Write-Host "🔍 Testing all API endpoints:"
Write-Host "================================="

# Test 1: Health Check
Write-Host "`n1. 🏥 Health Check (/health)"
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/health" -Method GET -TimeoutSec 10
    $json = $response.Content | ConvertFrom-Json
    Write-Host "   ✅ Status: $($json.data.status)"
    Write-Host "   ✅ Uptime: $($json.data.uptime_seconds) seconds"
    Write-Host "   ✅ Version: $($json.data.version)"
} catch {
    Write-Host "   ❌ Failed: $($_.Exception.Message)"
}

# Test 2: Root Endpoint
Write-Host "`n2. 🏠 Root Endpoint (/)"
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/" -Method GET -TimeoutSec 10
    $json = $response.Content | ConvertFrom-Json
    Write-Host "   ✅ Service: $($json.service)"
    Write-Host "   ✅ Version: $($json.version)"
    Write-Host "   ✅ Available endpoints: $($json.endpoints.PSObject.Properties.Name -join ', ')"
} catch {
    Write-Host "   ❌ Failed: $($_.Exception.Message)"
}

# Test 3: API Status
Write-Host "`n3. 📊 API Status (/api/v1/status)"
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/status" -Method GET -TimeoutSec 10
    $json = $response.Content | ConvertFrom-Json
    Write-Host "   ✅ Status: $($json.data.status)"
    Write-Host "   ✅ Request Count: $($json.data.request_count)"
} catch {
    Write-Host "   ❌ Failed: $($_.Exception.Message)"
}

# Test 4: Metrics
Write-Host "`n4. 📈 Metrics (/metrics)"
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/metrics" -Method GET -TimeoutSec 10
    $metrics = $response.Content
    $lines = $metrics -split "`n"
    Write-Host "   ✅ Metrics returned $($lines.Count) lines"
    Write-Host "   ✅ Contains prometheus metrics: $(($lines | Where-Object { $_ -match '# TYPE' }).Count) metric types"
} catch {
    Write-Host "   ❌ Failed: $($_.Exception.Message)"
}

# Test 5: Storage Verification
Write-Host "`n5. 💾 Storage Verification (/api/v1/storage/verify)"
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/storage/verify" -Method GET -TimeoutSec 10
    $json = $response.Content | ConvertFrom-Json
    Write-Host "   ✅ Verification ID: $($json.data.verification_id)"
    Write-Host "   ✅ Status: $($json.data.status)"
} catch {
    Write-Host "   ❌ Failed: $($_.Exception.Message)"
}

Write-Host "`n================================="
Write-Host "🎉 API Testing Complete!"

# Keep service running for manual testing
Write-Host "`n💡 Service is still running on http://localhost:8080"
Write-Host "   You can test endpoints manually or press Ctrl+C to stop"

# Wait for user input to stop
Read-Host "Press Enter to stop the service"

# Stop the service
Write-Host "🛑 Stopping service..."
Stop-Job $serviceJob -ErrorAction SilentlyContinue
Remove-Job $serviceJob -ErrorAction SilentlyContinue

Write-Host "✨ All tests completed successfully!"
