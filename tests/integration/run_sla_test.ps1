# Bitcoin Sprint Real-Life Integration Test Runner
# This script sets up the environment and runs comprehensive SLA tests

param(
    [ValidateSet("turbo", "enterprise", "standard", "lite")]
    [string]$Tier = "standard",
    
    [switch]$SkipBuild,
    [switch]$SkipCleanup,
    [string]$Duration = "30s"
)

$ErrorActionPreference = "Stop"

function Write-Section($title) {
    Write-Host ""
    Write-Host "=" * 80 -ForegroundColor Cyan
    Write-Host $title -ForegroundColor Yellow
    Write-Host "=" * 80 -ForegroundColor Cyan
}

function Write-Status($message, $color = "White") {
    Write-Host "🔄 $message" -ForegroundColor $color
}

function Write-Success($message) {
    Write-Host "✅ $message" -ForegroundColor Green
}

function Write-Error($message) {
    Write-Host "❌ $message" -ForegroundColor Red
}

# Test configuration
$testConfig = @{
    turbo = @{
        env_vars = @{
            SPRINT_TIER = "turbo"
            PEER_HMAC_SECRET = "testsecret123"
            LICENSE_KEY = "testlicense123"
            SKIP_LICENSE_VALIDATION = "true"
            ZMQ_ENDPOINT = "tcp://127.0.0.1:28332"
            API_HOST = "127.0.0.1"
            API_PORT = "8080"
        }
        sla_ms = 5
        description = "⚡ Turbo Tier - Ultra-low latency (≤5ms)"
    }
    enterprise = @{
        env_vars = @{
            SPRINT_TIER = "enterprise"
            PEER_HMAC_SECRET = "testsecret123"
            LICENSE_KEY = "testlicense123"
            SKIP_LICENSE_VALIDATION = "true"
            ZMQ_ENDPOINT = "tcp://127.0.0.1:28332"
            API_HOST = "127.0.0.1"
            API_PORT = "8080"
        }
        sla_ms = 20
        description = "🛡️ Enterprise Tier - High performance (≤20ms)"
    }
    standard = @{
        env_vars = @{
            SPRINT_TIER = "standard"
            PEER_HMAC_SECRET = "testsecret123"
            LICENSE_KEY = "testlicense123"
            SKIP_LICENSE_VALIDATION = "true"
            ZMQ_ENDPOINT = "tcp://127.0.0.1:28332"
            API_HOST = "127.0.0.1"
            API_PORT = "8080"
        }
        sla_ms = 300
        description = "📊 Standard Tier - Reliable performance (≤300ms)"
    }
    lite = @{
        env_vars = @{
            SPRINT_TIER = "lite"
            PEER_HMAC_SECRET = "testsecret123"
            LICENSE_KEY = "testlicense123"
            SKIP_LICENSE_VALIDATION = "true"
            ZMQ_ENDPOINT = "tcp://127.0.0.1:28332"
            API_HOST = "127.0.0.1"
            API_PORT = "8080"
        }
        sla_ms = 1000
        description = "🌱 Lite Tier - Basic performance (≤1s)"
    }
}

try {
    Write-Section "🚀 Bitcoin Sprint Real-Life Integration Test"
    Write-Host $testConfig[$Tier].description -ForegroundColor Green
    Write-Host "Test Duration: $Duration" -ForegroundColor Gray
    Write-Host "SLA Target: ≤$($testConfig[$Tier].sla_ms)ms" -ForegroundColor Gray

    # Step 1: Build Bitcoin Sprint (unless skipped)
    if (-not $SkipBuild) {
        Write-Section "🔨 Building Bitcoin Sprint"
        Write-Status "Compiling optimized Bitcoin Sprint binary..."
        
        $buildResult = Start-Process -FilePath "go" -ArgumentList @(
            "build", 
            "-ldflags=-s -w -extldflags=-static",
            "-trimpath",
            "-o", "bitcoin-sprint-test.exe",
            "./cmd/sprintd"
        ) -Wait -PassThru -NoNewWindow
        
        if ($buildResult.ExitCode -eq 0) {
            Write-Success "Bitcoin Sprint built successfully"
        } else {
            throw "Build failed with exit code $($buildResult.ExitCode)"
        }
    }

    # Step 2: Set environment variables for the tier
    Write-Section "⚙️ Configuring Environment"
    $config = $testConfig[$Tier]
    
    foreach ($envVar in $config.env_vars.GetEnumerator()) {
        [Environment]::SetEnvironmentVariable($envVar.Key, $envVar.Value, "Process")
        Write-Status "Set $($envVar.Key) = $($envVar.Value)"
    }

    # Step 3: Start Bitcoin Sprint in background
    Write-Section "🌟 Starting Bitcoin Sprint"
    Write-Status "Launching Bitcoin Sprint with $Tier tier configuration..."
    
    $sprintProcess = Start-Process -FilePath ".\bitcoin-sprint-test.exe" -PassThru -NoNewWindow
    Start-Sleep -Seconds 3
    
    if ($sprintProcess.HasExited) {
        throw "Bitcoin Sprint failed to start (exited immediately)"
    }
    
    Write-Success "Bitcoin Sprint started (PID: $($sprintProcess.Id))"

    # Step 4: Wait for API to be ready
    Write-Status "Waiting for API to become ready..."
    $maxRetries = 30
    $apiReady = $false
    
    for ($i = 1; $i -le $maxRetries; $i++) {
        try {
            $response = Invoke-RestMethod -Uri "http://127.0.0.1:8080/health" -TimeoutSec 2
            if ($response.status -eq "healthy") {
                $apiReady = $true
                break
            }
        }
        catch {
            # Continue waiting
        }
        Start-Sleep -Seconds 1
        Write-Host "." -NoNewline
    }
    
    if (-not $apiReady) {
        throw "API did not become ready within $maxRetries seconds"
    }
    
    Write-Host ""
    Write-Success "API is ready and responding"

    # Step 5: Run comprehensive tests
    Write-Section "🧪 Running SLA Compliance Tests"
    
    # Test tier information
    Write-Status "Retrieving tier information..."
    $tierInfo = Invoke-RestMethod -Uri "http://127.0.0.1:8080/tier-info" -TimeoutSec 5
    
    Write-Host "Current Configuration:" -ForegroundColor Cyan
    Write-Host "  Service: $($tierInfo.service)" -ForegroundColor Gray
    Write-Host "  Version: $($tierInfo.version)" -ForegroundColor Gray
    Write-Host "  Tier: $($tierInfo.current_tier)" -ForegroundColor Green
    Write-Host "  Latency Target: $($tierInfo.tier_config.latency_target)" -ForegroundColor Green
    Write-Host "  Max Peers: $($tierInfo.tier_config.max_peers)" -ForegroundColor Gray
    Write-Host "  Security: $($tierInfo.tier_config.security_level)" -ForegroundColor Gray

    # Validate tier matches expectation
    if ($tierInfo.current_tier -ne $Tier) {
        Write-Error "Tier mismatch! Expected: $Tier, Got: $($tierInfo.current_tier)"
        throw "Tier configuration error"
    }

    # Test API endpoints
    Write-Status "Testing API endpoints..."
    $endpoints = @("/health", "/tier-info", "/latest", "/metrics")
    $endpointResults = @{}
    
    foreach ($endpoint in $endpoints) {
        try {
            $response = Invoke-RestMethod -Uri "http://127.0.0.1:8080$endpoint" -TimeoutSec 3
            $endpointResults[$endpoint] = @{
                status = "✅ OK"
                response_time_ms = (Measure-Command { 
                    Invoke-RestMethod -Uri "http://127.0.0.1:8080$endpoint" -TimeoutSec 3 
                }).TotalMilliseconds
            }
            Write-Success "Endpoint $endpoint responding"
        }
        catch {
            $endpointResults[$endpoint] = @{
                status = "❌ FAILED"
                error = $_.Exception.Message
            }
            Write-Error "Endpoint $endpoint failed: $($_.Exception.Message)"
        }
    }

    # Step 6: Run load test (simulate sustained traffic)
    Write-Section "⚡ Running Performance Load Test"
    Write-Status "Simulating sustained API load for SLA validation..."
    
    $loadTestResults = @()
    $testIterations = 100
    $passedRequests = 0
    
    for ($i = 1; $i -le $testIterations; $i++) {
        $startTime = Get-Date
        
        try {
            $response = Invoke-RestMethod -Uri "http://127.0.0.1:8080/latest" -TimeoutSec 1
            $responseTime = (Get-Date) - $startTime
            $responseTimeMs = $responseTime.TotalMilliseconds
            
            if ($responseTimeMs -le $config.sla_ms) {
                $passedRequests++
            }
            
            $loadTestResults += @{
                iteration = $i
                response_time_ms = $responseTimeMs
                sla_compliant = $responseTimeMs -le $config.sla_ms
                timestamp = Get-Date -Format "HH:mm:ss.fff"
            }
            
            if ($i % 10 -eq 0) {
                Write-Host "." -NoNewline -ForegroundColor Green
            }
        }
        catch {
            $loadTestResults += @{
                iteration = $i
                error = $_.Exception.Message
                sla_compliant = $false
                timestamp = Get-Date -Format "HH:mm:ss.fff"
            }
            Write-Host "!" -NoNewline -ForegroundColor Red
        }
        
        # Small delay between requests
        Start-Sleep -Milliseconds 10
    }
    
    Write-Host ""
    
    # Calculate load test statistics
    $successfulRequests = $loadTestResults | Where-Object { $_.response_time_ms -ne $null }
    $avgResponseTime = ($successfulRequests | Measure-Object -Property response_time_ms -Average).Average
    $maxResponseTime = ($successfulRequests | Measure-Object -Property response_time_ms -Maximum).Maximum
    $minResponseTime = ($successfulRequests | Measure-Object -Property response_time_ms -Minimum).Minimum
    $slaComplianceRate = ($passedRequests / $testIterations) * 100

    Write-Host ""
    Write-Host "Load Test Results:" -ForegroundColor Cyan
    Write-Host "  Total Requests: $testIterations" -ForegroundColor Gray
    Write-Host "  Successful: $($successfulRequests.Count)" -ForegroundColor Green
    Write-Host "  SLA Compliant: $passedRequests ($($slaComplianceRate.ToString('F1'))%)" -ForegroundColor $(if ($slaComplianceRate -ge 95) { "Green" } else { "Red" })
    Write-Host "  Avg Response Time: $($avgResponseTime.ToString('F2'))ms" -ForegroundColor Gray
    Write-Host "  Min Response Time: $($minResponseTime.ToString('F2'))ms" -ForegroundColor Gray
    Write-Host "  Max Response Time: $($maxResponseTime.ToString('F2'))ms" -ForegroundColor Gray

    # Step 7: Generate final report
    Write-Section "📋 Test Report Generation"
    
    $testReport = @{
        test_timestamp = Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ"
        tier_tested = $Tier
        tier_config = $tierInfo
        sla_requirement_ms = $config.sla_ms
        load_test_results = @{
            total_requests = $testIterations
            successful_requests = $successfulRequests.Count
            sla_compliant_requests = $passedRequests
            sla_compliance_rate = $slaComplianceRate
            avg_response_time_ms = $avgResponseTime
            min_response_time_ms = $minResponseTime
            max_response_time_ms = $maxResponseTime
            detailed_results = $loadTestResults
        }
        endpoint_tests = $endpointResults
        overall_passed = $slaComplianceRate -ge 95
    }
    
    $reportFileName = "bitcoin_sprint_sla_report_$($Tier)_$(Get-Date -Format 'yyyyMMdd_HHmmss').json"
    $testReport | ConvertTo-Json -Depth 10 | Out-File -FilePath $reportFileName -Encoding UTF8
    
    Write-Success "Test report saved to: $reportFileName"

    # Final verdict
    Write-Section "🏁 Final Results"
    
    if ($testReport.overall_passed) {
        Write-Host ""
        Write-Host "🎉 SUCCESS: Bitcoin Sprint $($Tier.ToUpper()) tier PASSED all SLA tests!" -ForegroundColor Green
        Write-Host "   ✅ SLA Compliance: $($slaComplianceRate.ToString('F1'))%" -ForegroundColor Green
        Write-Host "   ✅ Performance Target: ≤$($config.sla_ms)ms (Actual avg: $($avgResponseTime.ToString('F2'))ms)" -ForegroundColor Green
        Write-Host ""
        Write-Host "🚀 This proves Bitcoin Sprint delivers on its performance promises!" -ForegroundColor Yellow
        Write-Host "📊 Use this report for customer demos and marketing materials." -ForegroundColor Cyan
    } else {
        Write-Host ""
        Write-Host "⚠️ PARTIAL SUCCESS: Bitcoin Sprint needs optimization" -ForegroundColor Yellow
        Write-Host "   📊 SLA Compliance: $($slaComplianceRate.ToString('F1'))% (Target: ≥95%)" -ForegroundColor Yellow
        Write-Host "   🎯 Consider tier adjustment or system optimization" -ForegroundColor Yellow
    }

} catch {
    Write-Error "Test failed: $($_.Exception.Message)"
    exit 1
} finally {
    # Cleanup
    if (-not $SkipCleanup) {
        Write-Section "🧹 Cleanup"
        
        if ($sprintProcess -and -not $sprintProcess.HasExited) {
            Write-Status "Stopping Bitcoin Sprint..."
            Stop-Process -Id $sprintProcess.Id -Force
            Start-Sleep -Seconds 2
            Write-Success "Bitcoin Sprint stopped"
        }
    }
}

Write-Host ""
Write-Host "Test completed successfully! 🎯" -ForegroundColor Green
