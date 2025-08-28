# Multi-Chain Sprint SLA Test
# Tests the multi-chain API endpoints with realistic performance measurement

param(
    [ValidateSet("turbo", "enterprise", "standard", "lite")]
    [string]$Tier = "enterprise",
    
    [int]$TestDurationSeconds = 60,
    [string]$Chain = "bitcoin",
    [int]$Port = 8080,
    [switch]$StartBackend,
    [switch]$Verbose
)

$ErrorActionPreference = "Stop"

function Write-Section($title) {
    Write-Host ""
    Write-Host "=" * 80 -ForegroundColor Cyan
    Write-Host $title -ForegroundColor Yellow
    Write-Host "=" * 80 -ForegroundColor Cyan
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

function Write-Warning($message) {
    Write-Host "⚠️ $message" -ForegroundColor Yellow
}

Write-Section "🚀 Multi-Chain Sprint SLA Performance Test"

# SLA requirements per tier
$slaRequirements = @{
    turbo = @{
        max_latency_ms = 5
        description = "⚡ Turbo Tier - Ultra-low latency across all chains"
        target_compliance = 99.5
    }
    enterprise = @{
        max_latency_ms = 20
        description = "🛡️ Enterprise Tier - High performance with security"
        target_compliance = 99.0
    }
    standard = @{
        max_latency_ms = 300
        description = "📊 Standard Tier - Reliable performance"
        target_compliance = 98.5
    }
    lite = @{
        max_latency_ms = 1000
        description = "🌱 Lite Tier - Basic performance"
        target_compliance = 97.0
    }
}

$config = $slaRequirements[$Tier]

Write-Host $config.description -ForegroundColor Green
Write-Host "SLA Target: ≤$($config.max_latency_ms)ms" -ForegroundColor Gray
Write-Host "Target Compliance: ≥$($config.target_compliance)%" -ForegroundColor Gray
Write-Host "Test Duration: $TestDurationSeconds seconds" -ForegroundColor Gray
Write-Host "Primary Chain: $Chain" -ForegroundColor Gray
Write-Host "API Port: $Port" -ForegroundColor Gray

try {
    # Step 1: Check if backend is running or start it
    Write-Section "🔍 Backend Status Check"
    
    $backendProcess = $null
    $backendRunning = $false
    
    try {
        $healthCheck = Invoke-RestMethod -Uri "http://localhost:$Port/api/v1/sprint/value" -TimeoutSec 2 -ErrorAction SilentlyContinue
        if ($healthCheck) {
            $backendRunning = $true
            Write-Success "Multi-Chain Sprint backend is already running on port $Port"
        }
    } catch {
        Write-Status "Backend not detected on port $Port"
    }
    
    if ($StartBackend -and -not $backendRunning) {
        Write-Section "🔨 Building and Starting Backend"
        
        # Set environment for multi-chain
        $env:TIER = $Tier.ToUpper()
        $env:API_KEY = "sprint-sla-test-2025"
        $env:API_PORT = $Port
        $env:MOCK_FAST_BLOCKS = "true"
        $env:PRIMARY_CHAIN = $Chain.ToLower()
        $env:OPTIMIZE_SYSTEM = "true"
        
        Write-Status "Building Multi-Chain Sprint with mock ZMQ support..."
        $buildResult = Start-Process -FilePath "go" -ArgumentList @(
            "build", "-tags", "nozmq", "-o", "multichain-sprint-test.exe", "./cmd/sprintd"
        ) -Wait -PassThru -NoNewWindow
        
        if ($buildResult.ExitCode -eq 0) {
            Write-Success "Build successful"
            
            Write-Status "Starting Multi-Chain Sprint backend..."
            $startInfo = New-Object System.Diagnostics.ProcessStartInfo
            $startInfo.FileName = ".\multichain-sprint-test.exe"
            $startInfo.UseShellExecute = $false
            $startInfo.RedirectStandardOutput = $true
            $startInfo.RedirectStandardError = $true
            $startInfo.CreateNoWindow = $true
            
            $backendProcess = [System.Diagnostics.Process]::Start($startInfo)
            
            # Wait for startup
            Write-Status "Waiting for backend initialization..."
            $retries = 0
            $maxRetries = 20
            
            while ($retries -lt $maxRetries) {
                Start-Sleep -Seconds 1
                $retries++
                
                try {
                    $response = Invoke-RestMethod -Uri "http://localhost:$Port/api/v1/sprint/value" -TimeoutSec 1 -ErrorAction SilentlyContinue
                    if ($response) {
                        $backendRunning = $true
                        Write-Success "Backend started successfully"
                        break
                    }
                } catch {
                    if ($retries % 5 -eq 0) {
                        Write-Host "." -NoNewline -ForegroundColor Yellow
                    }
                }
            }
            
            if (-not $backendRunning) {
                throw "Backend failed to start within timeout"
            }
        } else {
            throw "Build failed"
        }
    }
    
    if (-not $backendRunning) {
        Write-Warning "Backend is not running. Please start it manually or use -StartBackend flag"
        Write-Host "Example: .\multichain_sla_test.ps1 -StartBackend -Tier enterprise" -ForegroundColor Cyan
        return
    }

    # Step 2: API Endpoint Testing
    Write-Section "🔗 Multi-Chain API Endpoint Testing"
    
    $apiBaseUrl = "http://localhost:$Port"
    $testEndpoints = @(
        "/api/v1/sprint/value",
        "/api/v1/sprint/latency-stats", 
        "/api/v1/sprint/cache-stats",
        "/api/v1/sprint/tier-comparison",
        "/api/v1/universal/$Chain/status"
    )
    
    $endpointResults = @()
    
    foreach ($endpoint in $testEndpoints) {
        Write-Status "Testing endpoint: $endpoint"
        
        try {
            $start = Get-Date
            $response = Invoke-RestMethod -Uri "$apiBaseUrl$endpoint" -Method GET -TimeoutSec 5
            $duration = ((Get-Date) - $start).TotalMilliseconds
            
            $endpointResults += @{
                endpoint = $endpoint
                duration_ms = [math]::Round($duration, 2)
                status = "success"
                response_size = ($response | ConvertTo-Json).Length
            }
            
            Write-Success "  Response time: $([math]::Round($duration, 2))ms"
            
            if ($Verbose -and $response) {
                if ($response.PSObject.Properties.Name -contains "sprint_advantages") {
                    Write-Host "    Sprint advantages: $($response.sprint_advantages.Count)" -ForegroundColor Gray
                }
                if ($response.PSObject.Properties.Name -contains "supported_chains") {
                    Write-Host "    Supported chains: $($response.supported_chains)" -ForegroundColor Gray
                }
            }
        } catch {
            Write-Warning "  Failed: $($_.Exception.Message)"
            $endpointResults += @{
                endpoint = $endpoint
                duration_ms = -1
                status = "failed"
                error = $_.Exception.Message
            }
        }
    }

    # Step 3: SLA Performance Testing
    Write-Section "⚡ SLA Performance Testing"
    
    Write-Status "Running $TestDurationSeconds second SLA compliance test..."
    Write-Host "Target: ≤$($config.max_latency_ms)ms, ≥$($config.target_compliance)% compliance" -ForegroundColor Gray
    
    $testResults = @()
    $startTime = Get-Date
    $endTime = $startTime.AddSeconds($TestDurationSeconds)
    $requestCount = 0
    $successCount = 0
    $slaCompliantCount = 0
    
    # Primary endpoint for SLA testing
    $testUrl = "$apiBaseUrl/api/v1/sprint/value"
    
    while ((Get-Date) -lt $endTime) {
        try {
            $requestStart = Get-Date
            $response = Invoke-RestMethod -Uri $testUrl -Method GET -TimeoutSec 2
            $requestDuration = ((Get-Date) - $requestStart).TotalMilliseconds
            
            $requestCount++
            $successCount++
            
            $testResults += $requestDuration
            
            if ($requestDuration -le $config.max_latency_ms) {
                $slaCompliantCount++
            }
            
            # Show progress every 10 requests
            if ($requestCount % 10 -eq 0) {
                $currentCompliance = [math]::Round(($slaCompliantCount / $requestCount) * 100, 1)
                $avgLatency = [math]::Round(($testResults | Measure-Object -Average).Average, 2)
                Write-Host "  Requests: $requestCount | Avg: ${avgLatency}ms | Compliance: ${currentCompliance}%" -ForegroundColor Gray
            }
            
            # Small delay to avoid overwhelming the system
            Start-Sleep -Milliseconds 50
            
        } catch {
            $requestCount++
            Write-Warning "  Request failed: $($_.Exception.Message)"
            
            # Add penalty for failed requests
            $testResults += 999999
        }
    }

    # Step 4: Results Analysis
    Write-Section "📊 SLA Test Results Analysis"
    
    if ($testResults.Count -gt 0) {
        $avgLatency = [math]::Round(($testResults | Measure-Object -Average).Average, 2)
        $p50Latency = [math]::Round(($testResults | Sort-Object)[[math]::Floor($testResults.Count * 0.5)], 2)
        $p95Latency = [math]::Round(($testResults | Sort-Object)[[math]::Floor($testResults.Count * 0.95)], 2)
        $p99Latency = [math]::Round(($testResults | Sort-Object)[[math]::Floor($testResults.Count * 0.99)], 2)
        $maxLatency = [math]::Round(($testResults | Measure-Object -Maximum).Maximum, 2)
        $compliance = [math]::Round(($slaCompliantCount / $requestCount) * 100, 2)
        
        Write-Host ""
        Write-Host "🏁 Performance Results:" -ForegroundColor Cyan
        Write-Host "  Total Requests: $requestCount" -ForegroundColor White
        Write-Host "  Successful Requests: $successCount" -ForegroundColor White
        Write-Host "  Success Rate: $([math]::Round(($successCount / $requestCount) * 100, 1))%" -ForegroundColor White
        Write-Host ""
        Write-Host "📈 Latency Statistics:" -ForegroundColor Cyan
        Write-Host "  Average: ${avgLatency}ms" -ForegroundColor White
        Write-Host "  P50 (Median): ${p50Latency}ms" -ForegroundColor White
        Write-Host "  P95: ${p95Latency}ms" -ForegroundColor White
        Write-Host "  P99: ${p99Latency}ms" -ForegroundColor White
        Write-Host "  Maximum: ${maxLatency}ms" -ForegroundColor White
        Write-Host ""
        Write-Host "🎯 SLA Compliance:" -ForegroundColor Cyan
        Write-Host "  Target: ≤$($config.max_latency_ms)ms" -ForegroundColor White
        Write-Host "  Compliant Requests: $slaCompliantCount / $requestCount" -ForegroundColor White
        Write-Host "  Compliance Rate: ${compliance}%" -ForegroundColor White
        Write-Host "  Target Compliance: ≥$($config.target_compliance)%" -ForegroundColor White
        
        # SLA Pass/Fail
        Write-Host ""
        if ($compliance -ge $config.target_compliance) {
            Write-Success "🏆 SLA COMPLIANCE: PASSED"
            Write-Host "  Multi-Chain Sprint meets $Tier tier SLA requirements" -ForegroundColor Green
        } else {
            Write-Error "❌ SLA COMPLIANCE: FAILED"
            Write-Host "  Compliance: ${compliance}% < Target: $($config.target_compliance)%" -ForegroundColor Red
        }
        
        # Performance vs competitors insight
        Write-Host ""
        Write-Host "🚀 Competitive Analysis:" -ForegroundColor Cyan
        Write-Host "  Flat P99: ${p99Latency}ms (vs Infura's variable 200-2000ms)" -ForegroundColor White
        Write-Host "  Multi-chain: Single API (vs Alchemy's chain-specific endpoints)" -ForegroundColor White
        Write-Host "  Predictable: ${compliance}% consistency (vs QuickNode's variable performance)" -ForegroundColor White
    } else {
        Write-Error "No valid test results collected"
    }

    # Step 5: Endpoint Summary
    Write-Section "🔗 API Endpoint Test Summary"
    
    $successfulEndpoints = $endpointResults | Where-Object { $_.status -eq "success" }
    $failedEndpoints = $endpointResults | Where-Object { $_.status -eq "failed" }
    
    Write-Host "API Endpoint Results:" -ForegroundColor Cyan
    Write-Host "  Successful: $($successfulEndpoints.Count) / $($endpointResults.Count)" -ForegroundColor White
    
    if ($successfulEndpoints.Count -gt 0) {
        $avgEndpointLatency = [math]::Round(($successfulEndpoints.duration_ms | Measure-Object -Average).Average, 2)
        Write-Host "  Average Response Time: ${avgEndpointLatency}ms" -ForegroundColor White
        
        foreach ($result in $successfulEndpoints) {
            Write-Host "    $($result.endpoint): $($result.duration_ms)ms" -ForegroundColor Gray
        }
    }
    
    if ($failedEndpoints.Count -gt 0) {
        Write-Host ""
        Write-Host "Failed Endpoints:" -ForegroundColor Red
        foreach ($result in $failedEndpoints) {
            Write-Host "  $($result.endpoint): $($result.error)" -ForegroundColor Red
        }
    }

} catch {
    Write-Error "Test failed: $($_.Exception.Message)"
    Write-Host "Stack trace:" -ForegroundColor Red
    Write-Host $_.ScriptStackTrace -ForegroundColor Red
} finally {
    # Cleanup
    if ($backendProcess -and -not $backendProcess.HasExited) {
        Write-Status "Cleaning up backend process..."
        $backendProcess.Kill()
        $backendProcess.WaitForExit(5000)
        Write-Success "Backend process terminated"
    }
    
    # Clean up temporary files
    Remove-Item "multichain-sprint-test.exe" -ErrorAction SilentlyContinue
    
    Write-Section "🏁 Multi-Chain Sprint SLA Test Complete"
}
