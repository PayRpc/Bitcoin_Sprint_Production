# Comprehensive Bitcoin Sprint API Performance Test (Hardened)
# Tests all API endpoints with memory optimizations, error handling, and performance metrics

param(
    [ValidateSet("turbo", "enterprise", "standard", "lite")]
    [string]$Tier = "turbo",

    [int]$DurationSeconds = 60,
    [int]$ConcurrentUsers = 10,
    [int]$RampUpSeconds = 5,
    [switch]$IncludeMemoryProfiling,
    [switch]$SkipBuild,
    [switch]$Verbose,
    [switch]$UseExistingInstance,
    [switch]$UseWrk  # Optional: run wrk for high-concurrency tests
)

$ErrorActionPreference = "Stop"

# Test configuration
$API_BASE_URL = "http://localhost:9090"
$TEST_RESULTS_DIR = "performance-test-results"
$TIMESTAMP = Get-Date -Format "yyyyMMdd-HHmmss"

# Create results directory
if (!(Test-Path $TEST_RESULTS_DIR)) {
    New-Item -ItemType Directory -Path $TEST_RESULTS_DIR | Out-Null
}

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

# API endpoints to test
$API_ENDPOINTS = @(
    @{ Name = "Health Check"; Path = "/health"; Method = "GET"; Auth = $false },
    @{ Name = "System Status"; Path = "/status"; Method = "GET"; Auth = $false },
    @{ Name = "Version Info"; Path = "/version"; Method = "GET"; Auth = $false },
    @{ Name = "Metrics"; Path = "/metrics"; Method = "GET"; Auth = $true },
    @{ Name = "Turbo Status"; Path = "/turbo-status"; Method = "GET"; Auth = $false },
    @{ Name = "License Info"; Path = "/v1/license/info"; Method = "GET"; Auth = $true },
    @{ Name = "Analytics Summary"; Path = "/v1/analytics/summary"; Method = "GET"; Auth = $true },
    @{ Name = "Cache Status"; Path = "/cache-status"; Method = "GET"; Auth = $true },
    @{ Name = "Predictive Analytics"; Path = "/predictive"; Method = "GET"; Auth = $true },
    # Abuse simulation / fuzz endpoint
    @{ Name = "Malformed JSON"; Path = "/predictive"; Method = "POST"; Auth = $true; Payload = "{ not_json }" }
)

# Performance test results
$TestResults = @{
    StartTime = Get-Date
    EndTime = $null
    Tier = $Tier
    Configuration = @{}
    EndpointResults = @()
    MemoryStats = @()
    SystemStats = @()
    Summary = @{}
}

function Start-BitcoinSprint {
    Write-Section "🚀 Starting Bitcoin Sprint for Performance Testing"

    if ($UseExistingInstance) {
        Write-Status "Using existing Bitcoin Sprint instance on port 9090..."
        $global:BitcoinSprintProcess = $null
    } else {
        if (-not $SkipBuild) {
            Write-Status "Building optimized Bitcoin Sprint binary..."
            $buildResult = Start-Process -FilePath "go" -ArgumentList @("build", "-tags", "nozmq", "-o", "bitcoin-sprint-perf.exe", "./cmd/sprintd") -Wait -PassThru -NoNewWindow

            if ($buildResult.ExitCode -ne 0) {
                Write-Warning "Build failed, trying to use existing binary..."
                $existingBinaries = Get-ChildItem "bitcoin-sprint*.exe" | Where-Object { $_.Name -notlike "*perf*" } | Sort-Object LastWriteTime -Descending
                if ($existingBinaries.Count -gt 0) {
                    $binaryToUse = $existingBinaries[0].Name
                    Write-Status "Using existing binary: $binaryToUse"
                    Copy-Item $binaryToUse "bitcoin-sprint-perf.exe" -Force
                    Write-Success "Using existing working binary"
                } else {
                    throw "Build failed and no existing binary found"
                }
            } else {
                Write-Success "Build completed successfully"
            }
        }

        # Binary hash validation
        $binaryHash = Get-FileHash "bitcoin-sprint-perf.exe" -Algorithm SHA256
        Write-Host "Binary SHA256: $($binaryHash.Hash)" -ForegroundColor Gray

        # Configure environment
        $env:TIER = $Tier
        $env:SPRINT_TIER = $Tier
        $env:LICENSE_KEY = "perf_test_license_123"
        $env:SKIP_LICENSE_VALIDATION = "true"
        $env:ZMQ_ENDPOINT = "tcp://127.0.0.1:28332"
        $env:API_HOST = "127.0.0.1"
        $env:API_PORT = "9090"

        # Performance tuning
        $env:GC_PERCENT = "25"
        $env:MAX_CPU_CORES = "0"
        $env:HIGH_PRIORITY = "true"
        $env:LOCK_OS_THREAD = "true"
        $env:PREALLOC_BUFFERS = "true"
        $env:OPTIMIZE_SYSTEM = "true"
        $env:ENABLE_KERNEL_BYPASS = "true"

        Write-Status "Starting Bitcoin Sprint..."
        $global:BitcoinSprintProcess = Start-Process -FilePath "./bitcoin-sprint-perf.exe" -NoNewWindow -PassThru
    }

    Write-Status "Waiting for API to become ready..."
    $maxRetries = 30
    for ($retryCount = 0; $retryCount -lt $maxRetries; $retryCount++) {
        try {
            $response = Invoke-WebRequest -Uri "$API_BASE_URL/health" -TimeoutSec 2
            if ($response.StatusCode -eq 200) {
                Write-Success "API is ready and responding"
                break
            }
        } catch { }
        Start-Sleep -Seconds 1
        if ($retryCount -eq $maxRetries - 1) {
            if ($UseExistingInstance) {
                Write-Warning "API does not seem to be responding at $API_BASE_URL/health"
                Write-Status "Continuing with tests anyway..."
                break
            } else {
                Write-Error "API did not become ready"
                Stop-BitcoinSprint
                exit 1
            }
        }
    }

    # Capture initial configuration
    $TestResults.Configuration = @{
        Tier = $Tier
        APIBaseURL = $API_BASE_URL
        ConcurrentUsers = $ConcurrentUsers
        DurationSeconds = $DurationSeconds
        RampUpSeconds = $RampUpSeconds
        EnvironmentVariables = @{
            GC_PERCENT = $env:GC_PERCENT
            MAX_CPU_CORES = $env:MAX_CPU_CORES
            HIGH_PRIORITY = $env:HIGH_PRIORITY
            OPTIMIZE_SYSTEM = $env:OPTIMIZE_SYSTEM
        }
    }
}

function Stop-BitcoinSprint {
    if (-not $UseExistingInstance -and $global:BitcoinSprintProcess -and !$global:BitcoinSprintProcess.HasExited) {
        Write-Status "Stopping Bitcoin Sprint..."
        Stop-Process -Id $global:BitcoinSprintProcess.Id -Force
        $global:BitcoinSprintProcess = $null
        Write-Success "Bitcoin Sprint stopped"
    } elseif ($UseExistingInstance) {
        Write-Status "Leaving existing Bitcoin Sprint instance running..."
    }
}

function Get-SystemStats {
    $cpu = Get-Counter '\Processor(_Total)\% Processor Time' -ErrorAction SilentlyContinue
    $memory = Get-Counter '\Memory\% Committed Bytes In Use' -ErrorAction SilentlyContinue
    $network = Get-Counter '\Network Interface(*)\Bytes Total/sec' -ErrorAction SilentlyContinue

    return @{
        Timestamp = Get-Date
        CPUUsage = if ($cpu) { $cpu.CounterSamples[0].CookedValue } else { 0 }
        MemoryUsage = if ($memory) { $memory.CounterSamples[0].CookedValue } else { 0 }
        NetworkBytesPerSec = if ($network) { ($network.CounterSamples | Measure-Object -Property CookedValue -Sum).Sum } else { 0 }
    }
}

function Test-APIEndpoint {
    param(
        [string]$Name,
        [string]$Path,
        [string]$Method = "GET",
        [bool]$Auth = $false,
        [int]$DurationSeconds = 10,
        [int]$ConcurrentUsers = 5
    )

    Write-Status "Testing $Name endpoint ($Path)..."

    $results = @{
        Endpoint = $Name
        Path = $Path
        Method = $Method
        AuthRequired = $Auth
        StartTime = Get-Date
        EndTime = $null
        TotalRequests = 0
        SuccessfulRequests = 0
        FailedRequests = 0
        ResponseTimes = @()
        Errors = @()
        StatusCodes = @{}
    }

    $jobScript = {
        param($url, $method, $auth, $duration)

        $results = @{
            Requests = 0
            Successes = 0
            Failures = 0
            ResponseTimes = @()
            StatusCodes = @{}
            Errors = @()
        }

        $endTime = (Get-Date).AddSeconds($duration)

        while ((Get-Date) -lt $endTime) {
            try {
                $start = Get-Date
                $headers = @{}
                if ($auth) {
                    $headers["Authorization"] = "Bearer perf_test_key_123"
                }

                $response = Invoke-WebRequest -Uri $url -Method $method -Headers $headers -TimeoutSec 10
                $responseTime = ((Get-Date) - $start).TotalMilliseconds

                $results.Requests++
                $results.Successes++
                $results.ResponseTimes += $responseTime

                if ($results.StatusCodes.ContainsKey($response.StatusCode)) {
                    $results.StatusCodes[$response.StatusCode]++
                } else {
                    $results.StatusCodes[$response.StatusCode] = 1
                }
            } catch {
                $results.Requests++
                $results.Failures++
                $results.Errors += $_.Exception.Message
            }
        }

        return $results
    }

    # Start concurrent jobs
    $jobs = @()
    for ($i = 0; $i -lt $ConcurrentUsers; $i++) {
        $url = "$API_BASE_URL$Path"
        $job = Start-Job -ScriptBlock $jobScript -ArgumentList $url, $Method, $Auth, $DurationSeconds
        $jobs += $job
    }

    # Wait for all jobs to complete
    $jobs | Wait-Job | Out-Null

    # Collect results
    foreach ($job in $jobs) {
        $jobResult = Receive-Job -Job $job
        $results.TotalRequests += $jobResult.Requests
        $results.SuccessfulRequests += $jobResult.Successes
        $results.FailedRequests += $jobResult.Failures
        $results.ResponseTimes += $jobResult.ResponseTimes
        $results.Errors += $jobResult.Errors

        foreach ($status in $jobResult.StatusCodes.Keys) {
            if ($results.StatusCodes.ContainsKey($status)) {
                $results.StatusCodes[$status] += $jobResult.StatusCodes[$status]
            } else {
                $results.StatusCodes[$status] = $jobResult.StatusCodes[$status]
            }
        }

        Remove-Job -Job $job
    }

    $results.EndTime = Get-Date

    # Calculate statistics
    if ($results.ResponseTimes.Count -gt 0) {
        $results.Stats = @{
            AverageResponseTime = ($results.ResponseTimes | Measure-Object -Average).Average
            MinResponseTime = ($results.ResponseTimes | Measure-Object -Minimum).Minimum
            MaxResponseTime = ($results.ResponseTimes | Measure-Object -Maximum).Maximum
            Percentile95 = ($results.ResponseTimes | Sort-Object)[[math]::Floor($results.ResponseTimes.Count * 0.95)]
            RequestsPerSecond = $results.TotalRequests / $DurationSeconds
            SuccessRate = ($results.SuccessfulRequests / $results.TotalRequests) * 100
        }
    }

    Write-Success "$Name completed: $($results.SuccessfulRequests)/$($results.TotalRequests) successful ($([math]::Round($results.Stats.SuccessRate, 2))% success rate)"

    return $results
}

function Test-MemoryUsage {
    Write-Section "🧠 Memory Usage Analysis"

    if (!$IncludeMemoryProfiling) {
        Write-Status "Memory profiling skipped (use -IncludeMemoryProfiling to enable)"
        return
    }

    Write-Status "Monitoring memory usage during testing..."

    # Get initial memory stats
    $initialMemory = Get-WmiObject -Class Win32_Process -Filter "ProcessId = $($global:BitcoinSprintProcess.Id)" |
        Select-Object -Property WorkingSetSize, VirtualSize, PrivatePageCount

    $memoryStats = @{
        Initial = @{
            WorkingSetMB = [math]::Round($initialMemory.WorkingSetSize / 1MB, 2)
            VirtualSizeMB = [math]::Round($initialMemory.VirtualSize / 1MB, 2)
            PrivatePageCount = $initialMemory.PrivatePageCount
        }
        Samples = @()
    }

    # Monitor memory during testing
    $monitorDuration = $DurationSeconds + 10
    $endTime = (Get-Date).AddSeconds($monitorDuration)

    while ((Get-Date) -lt $endTime) {
        $currentMemory = Get-WmiObject -Class Win32_Process -Filter "ProcessId = $($global:BitcoinSprintProcess.Id)" |
            Select-Object -Property WorkingSetSize, VirtualSize, PrivatePageCount

        $memoryStats.Samples += @{
            Timestamp = Get-Date
            WorkingSetMB = [math]::Round($currentMemory.WorkingSetSize / 1MB, 2)
            VirtualSizeMB = [math]::Round($currentMemory.VirtualSize / 1MB, 2)
            PrivatePageCount = $currentMemory.PrivatePageCount
        }

        Start-Sleep -Seconds 2
    }

    # Get final memory stats
    $finalMemory = Get-WmiObject -Class Win32_Process -Filter "ProcessId = $($global:BitcoinSprintProcess.Id)" |
        Select-Object -Property WorkingSetSize, VirtualSize, PrivatePageCount

    $memoryStats.Final = @{
        WorkingSetMB = [math]::Round($finalMemory.WorkingSetSize / 1MB, 2)
        VirtualSizeMB = [math]::Round($finalMemory.VirtualSize / 1MB, 2)
        PrivatePageCount = $finalMemory.PrivatePageCount
    }

    # Calculate memory statistics
    $workingSetValues = $memoryStats.Samples | ForEach-Object { $_.WorkingSetMB }
    $memoryStats.Analysis = @{
        AverageWorkingSetMB = ($workingSetValues | Measure-Object -Average).Average
        MinWorkingSetMB = ($workingSetValues | Measure-Object -Minimum).Minimum
        MaxWorkingSetMB = ($workingSetValues | Measure-Object -Maximum).Maximum
        MemoryGrowthMB = $memoryStats.Final.WorkingSetMB - $memoryStats.Initial.WorkingSetMB
        MemoryEfficiency = if ($TestResults.Summary.TotalRequests -gt 0) {
            [math]::Round($memoryStats.AverageWorkingSetMB / ($TestResults.Summary.TotalRequests / 1000), 4)
        } else { 0 }
    }

    $TestResults.MemoryStats = $memoryStats

    Write-Success "Memory analysis completed"
    Write-Host "  Initial Memory: $($memoryStats.Initial.WorkingSetMB) MB" -ForegroundColor Gray
    Write-Host "  Final Memory: $($memoryStats.Final.WorkingSetMB) MB" -ForegroundColor Gray
    Write-Host "  Memory Growth: $([math]::Round($memoryStats.Analysis.MemoryGrowthMB, 2)) MB" -ForegroundColor Gray
    Write-Host "  Average Memory: $([math]::Round($memoryStats.Analysis.AverageWorkingSetMB, 2)) MB" -ForegroundColor Gray
}

function Run-PerformanceTest {
    Write-Section "🧪 Running Comprehensive API Performance Test"

    $TestResults.StartTime = Get-Date

    # Test each endpoint
    foreach ($endpoint in $API_ENDPOINTS) {
        $result = Test-APIEndpoint -Name $endpoint.Name -Path $endpoint.Path -Method $endpoint.Method -Auth $endpoint.Auth -DurationSeconds $DurationSeconds -ConcurrentUsers $ConcurrentUsers
        $TestResults.EndpointResults += $result

        # Brief pause between endpoint tests
        Start-Sleep -Seconds 1
    }

    $TestResults.EndTime = Get-Date

    # Calculate overall summary
    $totalRequests = ($TestResults.EndpointResults | Measure-Object -Property TotalRequests -Sum).Sum
    $totalSuccessful = ($TestResults.EndpointResults | Measure-Object -Property SuccessfulRequests -Sum).Sum
    $allResponseTimes = $TestResults.EndpointResults | ForEach-Object { $_.ResponseTimes } | Where-Object { $_ -ne $null }

    $TestResults.Summary = @{
        TotalEndpoints = $API_ENDPOINTS.Count
        TotalRequests = $totalRequests
        TotalSuccessful = $totalSuccessful
        OverallSuccessRate = if ($totalRequests -gt 0) { ($totalSuccessful / $totalRequests) * 100 } else { 0 }
        AverageResponseTime = if ($allResponseTimes.Count -gt 0) { ($allResponseTimes | Measure-Object -Average).Average } else { 0 }
        MinResponseTime = if ($allResponseTimes.Count -gt 0) { ($allResponseTimes | Measure-Object -Minimum).Minimum } else { 0 }
        MaxResponseTime = if ($allResponseTimes.Count -gt 0) { ($allResponseTimes | Measure-Object -Maximum).Maximum } else { 0 }
        TestDuration = ($TestResults.EndTime - $TestResults.StartTime).TotalSeconds
        RequestsPerSecond = if ($TestResults.Summary.TestDuration -gt 0) { $totalRequests / $TestResults.Summary.TestDuration } else { 0 }
    }
}

function Export-Results {
    Write-Section "📊 Exporting Performance Test Results"

    $resultsFile = "$TEST_RESULTS_DIR\api-performance-test-$TIMESTAMP.json"
    $summaryFile = "$TEST_RESULTS_DIR\api-performance-summary-$TIMESTAMP.txt"

    # Export detailed results
    try {
        # Convert hashtables with non-string keys to proper format
        $jsonResults = $TestResults | ConvertTo-Json -Depth 10 -ErrorAction Stop
        $jsonResults | Out-File -FilePath $resultsFile -Encoding UTF8
        Write-Success "Detailed results exported to: $resultsFile"
    } catch {
        Write-Error "Performance test failed: $_"
        # Save the test results summary in a simpler format
        $simpleResults = @{
            StartTime = $TestResults.StartTime
            EndTime = $TestResults.EndTime
            Tier = $TestResults.Tier
            Summary = $TestResults.Summary
        }
        $simpleResults | ConvertTo-Json -Depth 5 | Out-File -FilePath "$TEST_RESULTS_DIR\api-performance-simple-$TIMESTAMP.json" -Encoding UTF8
    }

    # Create summary report
    $summary = @"
BITCOIN SPRINT API PERFORMANCE TEST REPORT
==========================================

Test Configuration:
- Tier: $($TestResults.Tier)
- Duration: $($TestResults.Summary.TestDuration) seconds
- Concurrent Users: $ConcurrentUsers
- Endpoints Tested: $($TestResults.Summary.TotalEndpoints)

Overall Results:
- Total Requests: $($TestResults.Summary.TotalRequests)
- Successful Requests: $($TestResults.Summary.TotalSuccessful)
- Success Rate: $([math]::Round($TestResults.Summary.OverallSuccessRate, 2))%
- Requests/Second: $([math]::Round($TestResults.Summary.RequestsPerSecond, 2))
- Average Response Time: $([math]::Round($TestResults.Summary.AverageResponseTime, 2))ms
- Min Response Time: $([math]::Round($TestResults.Summary.MinResponseTime, 2))ms
- Max Response Time: $([math]::Round($TestResults.Summary.MaxResponseTime, 2))ms

Endpoint Details:
"@

    foreach ($endpoint in $TestResults.EndpointResults) {
        $summary += @"

$($endpoint.Endpoint) ($($endpoint.Path)):
  - Requests: $($endpoint.TotalRequests)
  - Success Rate: $([math]::Round($endpoint.Stats.SuccessRate, 2))%
  - Avg Response Time: $([math]::Round($endpoint.Stats.AverageResponseTime, 2))ms
  - 95th Percentile: $([math]::Round($endpoint.Stats.Percentile95, 2))ms
  - Requests/Second: $([math]::Round($endpoint.Stats.RequestsPerSecond, 2))
"@
    }

    if ($TestResults.MemoryStats) {
        $summary += @"

Memory Analysis:
- Initial Memory: $($TestResults.MemoryStats.Initial.WorkingSetMB) MB
- Final Memory: $($TestResults.MemoryStats.Final.WorkingSetMB) MB
- Memory Growth: $([math]::Round($TestResults.MemoryStats.Analysis.MemoryGrowthMB, 2)) MB
- Average Memory: $([math]::Round($TestResults.MemoryStats.Analysis.AverageWorkingSetMB, 2)) MB
- Memory Efficiency: $($TestResults.MemoryStats.Analysis.MemoryEfficiency) MB per 1000 requests
"@
    }

    $summary | Out-File -FilePath $summaryFile -Encoding UTF8
    Write-Success "Summary report exported to: $summaryFile"
}

function Show-LiveResults {
    Write-Section "📈 Live Performance Monitoring"

    $endTime = (Get-Date).AddSeconds($DurationSeconds)
    $startTime = Get-Date

    while ((Get-Date) -lt $endTime) {
        $elapsed = (Get-Date) - $startTime
        $progress = ($elapsed.TotalSeconds / $DurationSeconds) * 100

        # Get current system stats
        $systemStats = Get-SystemStats
        $TestResults.SystemStats += $systemStats

        Write-Host ("Progress: [{0}] {1:F1}%" -f ("#" * [math]::Floor($progress / 2)).PadRight(50), $progress) -ForegroundColor Green
        Write-Host ("  CPU: {0:F1}% | Memory: {1:F1}% | Network: {2:F0} B/s" -f $systemStats.CPUUsage, $systemStats.MemoryUsage, $systemStats.NetworkBytesPerSec) -ForegroundColor Gray

        Start-Sleep -Seconds 5
    }
}

# Main execution
try {
    Write-Section "🚀 Bitcoin Sprint API Performance Test Suite"
    Write-Host "Tier: $Tier | Duration: $DurationSeconds seconds | Concurrent Users: $ConcurrentUsers" -ForegroundColor Green
    Write-Host "Memory Profiling: $($IncludeMemoryProfiling ? 'Enabled' : 'Disabled')" -ForegroundColor Gray

    # Start Bitcoin Sprint
    Start-BitcoinSprint

    # Run performance tests
    if ($IncludeMemoryProfiling) {
        # Run memory monitoring in background
        $memoryJob = Start-Job -ScriptBlock { Test-MemoryUsage }
    }

    # Show live monitoring
    $monitorJob = Start-Job -ScriptBlock { Show-LiveResults }

    # Run the actual performance tests
    Run-PerformanceTest

    # Wait for background jobs
    if ($IncludeMemoryProfiling) {
        Wait-Job -Job $memoryJob | Out-Null
        Remove-Job -Job $memoryJob
    }

    Wait-Job -Job $monitorJob | Out-Null
    Remove-Job -Job $monitorJob

    # Export results
    Export-Results

    # Display final summary
    Write-Section "🎯 Performance Test Summary"

    Write-Host "Overall Results:" -ForegroundColor Green
    Write-Host "  Total Requests: $($TestResults.Summary.TotalRequests)" -ForegroundColor White
    Write-Host "  Success Rate: $([math]::Round($TestResults.Summary.OverallSuccessRate, 2))%" -ForegroundColor White
    Write-Host "  Requests/Second: $([math]::Round($TestResults.Summary.RequestsPerSecond, 2))" -ForegroundColor White
    Write-Host "  Average Response Time: $([math]::Round($TestResults.Summary.AverageResponseTime, 2))ms" -ForegroundColor White

    if ($TestResults.MemoryStats) {
        Write-Host "Memory Usage:" -ForegroundColor Green
        Write-Host "  Initial: $($TestResults.MemoryStats.Initial.WorkingSetMB) MB" -ForegroundColor White
        Write-Host "  Final: $($TestResults.MemoryStats.Final.WorkingSetMB) MB" -ForegroundColor White
        Write-Host "  Growth: $([math]::Round($TestResults.MemoryStats.Analysis.MemoryGrowthMB, 2)) MB" -ForegroundColor White
    }

    Write-Success "Performance testing completed successfully!"

} catch {
    Write-Error "Performance test failed: $($_.Exception.Message)"
    Write-Error $_.ScriptStackTrace
} finally {
    # Clean up
    Stop-BitcoinSprint
}

Write-Section "✅ Performance Test Complete"
Write-Host "Results saved to: $TEST_RESULTS_DIR" -ForegroundColor Green
