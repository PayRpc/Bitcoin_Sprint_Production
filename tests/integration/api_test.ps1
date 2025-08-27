# Bitcoin Sprint API Integration Test (Mock Mode)
# Demonstrates API functionality without requiring ZMQ setup

param(
    [ValidateSet("turbo", "enterprise", "standard", "lite")]
    [string]$Tier = "standard"
)

$ErrorActionPreference = "Stop"

function Write-Section($title) {
    Write-Host ""
    Write-Host "=" * 60 -ForegroundColor Cyan
    Write-Host $title -ForegroundColor Yellow
    Write-Host "=" * 60 -ForegroundColor Cyan
}

function Write-Status($message) {
    Write-Host "🔄 $message" -ForegroundColor Blue
}

function Write-Success($message) {
    Write-Host "✅ $message" -ForegroundColor Green
}

function Write-Error($message) {
    Write-Host "❌ $message" -ForegroundColor Red
}

try {
    Write-Section "🚀 Bitcoin Sprint API Integration Test"
    Write-Host "Testing tier: $($Tier.ToUpper())" -ForegroundColor Green

    # Set environment variables
    Write-Status "Setting up environment for $Tier tier..."
    $env:SPRINT_TIER = $Tier
    $env:PEER_HMAC_SECRET = "testsecret123"
    $env:LICENSE_KEY = "testlicense123"
    $env:SKIP_LICENSE_VALIDATION = "true"
    $env:ZMQ_ENDPOINT = "tcp://127.0.0.1:28332"
    $env:API_HOST = "127.0.0.1"
    $env:API_PORT = "8080"

    # Build Bitcoin Sprint (try without ZMQ first)
    Write-Status "Building Bitcoin Sprint..."
    
    # Use go build without ZMQ dependencies for demo
    $buildResult = Start-Process -FilePath "go" -ArgumentList @(
        "build", 
        "-tags", "nozmq",
        "-o", "bitcoin-sprint-demo.exe",
        "./cmd/sprintd"
    ) -Wait -PassThru -NoNewWindow -RedirectStandardError "build_error.log"
    
    if ($buildResult.ExitCode -ne 0) {
        Write-Status "ZMQ build failed, trying regular build..."
        # If that fails, just demonstrate with existing binary
        if (Test-Path "bitcoin-sprint.exe") {
            Copy-Item "bitcoin-sprint.exe" "bitcoin-sprint-demo.exe"
            Write-Success "Using existing Bitcoin Sprint binary"
        } else {
            throw "No Bitcoin Sprint binary available"
        }
    } else {
        Write-Success "Bitcoin Sprint built successfully"
    }

    # Start Bitcoin Sprint
    Write-Status "Starting Bitcoin Sprint API server..."
    $sprintProcess = Start-Process -FilePath ".\bitcoin-sprint-demo.exe" -PassThru -WindowStyle Hidden
    Start-Sleep -Seconds 5

    if ($sprintProcess.HasExited) {
        Write-Error "Bitcoin Sprint exited immediately, checking logs..."
        Get-Content "*.log" -ErrorAction SilentlyContinue | Select-Object -Last 10
        throw "Bitcoin Sprint failed to start"
    }

    Write-Success "Bitcoin Sprint started (PID: $($sprintProcess.Id))"

    # Test API endpoints
    Write-Section "🧪 Testing API Endpoints"
    
    $apiBase = "http://127.0.0.1:8080"
    $testResults = @{}

    # Test health endpoint
    Write-Status "Testing /health endpoint..."
    try {
        $healthResponse = Invoke-RestMethod -Uri "$apiBase/health" -TimeoutSec 5
        Write-Success "Health check passed"
        Write-Host "  Status: $($healthResponse.status)" -ForegroundColor Gray
        Write-Host "  Service: $($healthResponse.service)" -ForegroundColor Gray
        $testResults.health = $true
    }
    catch {
        Write-Error "Health check failed: $($_.Exception.Message)"
        $testResults.health = $false
    }

    # Test tier-info endpoint
    Write-Status "Testing /tier-info endpoint..."
    try {
        $tierResponse = Invoke-RestMethod -Uri "$apiBase/tier-info" -TimeoutSec 5
        Write-Success "Tier info retrieved"
        Write-Host "  Current Tier: $($tierResponse.current_tier)" -ForegroundColor Green
        Write-Host "  Latency Target: $($tierResponse.tier_config.latency_target)" -ForegroundColor Green
        Write-Host "  Max Peers: $($tierResponse.tier_config.max_peers)" -ForegroundColor Gray
        Write-Host "  Security Level: $($tierResponse.tier_config.security_level)" -ForegroundColor Gray
        $testResults.tier_info = $true
        
        # Validate tier matches
        if ($tierResponse.current_tier -eq $Tier) {
            Write-Success "Tier configuration matches request"
        } else {
            Write-Error "Tier mismatch: Expected $Tier, got $($tierResponse.current_tier)"
        }
    }
    catch {
        Write-Error "Tier info failed: $($_.Exception.Message)"
        $testResults.tier_info = $false
    }

    # Test latest endpoint  
    Write-Status "Testing /latest endpoint..."
    try {
        $latestResponse = Invoke-RestMethod -Uri "$apiBase/latest" -TimeoutSec 5
        if ($latestResponse.msg -eq "no block yet") {
            Write-Success "Latest endpoint responding (no blocks yet - expected in mock mode)"
        } else {
            Write-Success "Latest endpoint returned block data"
            Write-Host "  Block Hash: $($latestResponse.hash)" -ForegroundColor Gray
            Write-Host "  Relay Time: $($latestResponse.relay_time_ms)ms" -ForegroundColor Green
        }
        $testResults.latest = $true
    }
    catch {
        Write-Error "Latest endpoint failed: $($_.Exception.Message)"
        $testResults.latest = $false
    }

    # Test metrics endpoint
    Write-Status "Testing /metrics endpoint..."
    try {
        $metricsResponse = Invoke-RestMethod -Uri "$apiBase/metrics" -TimeoutSec 5
        Write-Success "Metrics endpoint responding"
        $testResults.metrics = $true
    }
    catch {
        Write-Error "Metrics endpoint failed: $($_.Exception.Message)"
        $testResults.metrics = $false
    }

    # Performance test
    Write-Section "⚡ Performance Testing"
    Write-Status "Running rapid API requests to test responsiveness..."
    
    $performanceResults = @()
    $requestCount = 20
    
    for ($i = 1; $i -le $requestCount; $i++) {
        $startTime = Get-Date
        try {
            $response = Invoke-RestMethod -Uri "$apiBase/health" -TimeoutSec 1
            $responseTime = (Get-Date) - $startTime
            $performanceResults += $responseTime.TotalMilliseconds
            Write-Host "." -NoNewline -ForegroundColor Green
        }
        catch {
            Write-Host "!" -NoNewline -ForegroundColor Red
        }
    }
    
    Write-Host ""
    
    if ($performanceResults.Count -gt 0) {
        $avgResponseTime = ($performanceResults | Measure-Object -Average).Average
        $maxResponseTime = ($performanceResults | Measure-Object -Maximum).Maximum
        $minResponseTime = ($performanceResults | Measure-Object -Minimum).Minimum
        
        Write-Success "Performance test completed"
        Write-Host "  Requests: $($performanceResults.Count)/$requestCount" -ForegroundColor Gray
        Write-Host "  Average: $($avgResponseTime.ToString('F2'))ms" -ForegroundColor Green
        Write-Host "  Min: $($minResponseTime.ToString('F2'))ms" -ForegroundColor Gray
        Write-Host "  Max: $($maxResponseTime.ToString('F2'))ms" -ForegroundColor Gray
        
        # Check if performance meets tier expectations
        $tierLimits = @{
            turbo = 50
            enterprise = 100
            standard = 500
            lite = 1000
        }
        
        if ($avgResponseTime -le $tierLimits[$Tier]) {
            Write-Success "Performance meets $Tier tier expectations (≤$($tierLimits[$Tier])ms for API)"
        } else {
            Write-Error "Performance needs optimization for $Tier tier"
        }
    }

    # Generate report
    Write-Section "📋 Test Report"
    
    $passedTests = ($testResults.Values | Where-Object { $_ -eq $true }).Count
    $totalTests = $testResults.Count
    
    Write-Host "Test Results Summary:" -ForegroundColor Cyan
    foreach ($test in $testResults.GetEnumerator()) {
        $status = if ($test.Value) { "✅ PASSED" } else { "❌ FAILED" }
        Write-Host "  $($test.Key): $status" -ForegroundColor $(if ($test.Value) { "Green" } else { "Red" })
    }
    
    $overallSuccess = $passedTests -eq $totalTests
    Write-Host ""
    Write-Host "Overall: $passedTests/$totalTests tests passed" -ForegroundColor $(if ($overallSuccess) { "Green" } else { "Red" })
    
    if ($overallSuccess) {
        Write-Success "🎉 All API integration tests PASSED!"
        Write-Host "Bitcoin Sprint $($Tier.ToUpper()) tier is ready for production" -ForegroundColor Green
    } else {
        Write-Error "⚠️ Some tests failed - review configuration"
    }

    # Save report
    $report = @{
        timestamp = Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ"
        tier = $Tier
        test_results = $testResults
        performance = if ($performanceResults.Count -gt 0) {
            @{
                avg_response_time_ms = $avgResponseTime
                min_response_time_ms = $minResponseTime
                max_response_time_ms = $maxResponseTime
                total_requests = $performanceResults.Count
            }
        } else { $null }
        overall_passed = $overallSuccess
    }
    
    $reportFile = "bitcoin_sprint_api_test_$($Tier)_$(Get-Date -Format 'yyyyMMdd_HHmmss').json"
    $report | ConvertTo-Json -Depth 3 | Out-File -FilePath $reportFile -Encoding UTF8
    
    Write-Host ""
    Write-Host "📁 Test report saved to: $reportFile" -ForegroundColor Blue

} catch {
    Write-Error "Test failed: $($_.Exception.Message)"
    exit 1
} finally {
    # Cleanup
    if ($sprintProcess -and -not $sprintProcess.HasExited) {
        Write-Status "Stopping Bitcoin Sprint..."
        Stop-Process -Id $sprintProcess.Id -Force -ErrorAction SilentlyContinue
        Write-Success "Cleanup completed"
    }
}

Write-Host ""
Write-Host "API integration test completed! 🎯" -ForegroundColor Green
