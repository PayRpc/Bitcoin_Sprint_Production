#!/usr/bin/env pwsh
#
# Real User API Performance Testing
# Simulates realistic usage patterns and measures actual speeds users experience
#

param(
    [ValidateSet("FREE", "PRO", "ENTERPRISE", "ALL")]
    [string]$Tier = "ALL",
    
    [int]$TestDuration = 300,  # 5 minutes by default
    [int]$Users = 5,           # Concurrent users
    [switch]$ColdStart,        # Test from cold start
    [switch]$WarmupFirst,      # Warmup before measurements
    [switch]$DetailedLogs,     # Show per-request details
    [string]$ReportFile = "real-user-performance-$(Get-Date -Format 'yyyyMMdd-HHmmss').json"
)

# Real-world usage patterns
$USER_SCENARIOS = @{
    "Dashboard_User" = @{
        description = "User checking dashboard every 30 seconds"
        endpoints = @("/status", "/latest")
        interval = 30
        pattern = "regular"
    }
    "Trading_Bot" = @{
        description = "High-frequency trading bot"
        endpoints = @("/latest", "/status", "/latest")
        interval = 2
        pattern = "burst"
    }
    "Mobile_App" = @{
        description = "Mobile app sync every 2 minutes"
        endpoints = @("/latest")
        interval = 120
        pattern = "regular"
    }
    "Analytics_Service" = @{
        description = "Analytics pulling data every minute"
        endpoints = @("/latest", "/status")
        interval = 60
        pattern = "regular"
    }
    "Price_Widget" = @{
        description = "Website price widget updates"
        endpoints = @("/latest")
        interval = 15
        pattern = "regular"
    }
}

# Performance expectations by tier
$TIER_EXPECTATIONS = @{
    "FREE" = @{
        avg_response = 200      # 200ms average expected for free tier
        p95_response = 500      # 95th percentile under 500ms
        max_acceptable = 1000   # Nothing should take more than 1 second
        throughput_rps = 1      # 1 request per second sustained
        rate_limit = 20         # 20 requests per minute
    }
    "PRO" = @{
        avg_response = 100      # 100ms average
        p95_response = 200      # 95th percentile under 200ms
        max_acceptable = 500    # Max 500ms
        throughput_rps = 5      # 5 requests per second sustained
        rate_limit = 300        # 300 requests per minute (5 per second)
    }
    "ENTERPRISE" = @{
        avg_response = 50       # 50ms average with turbo
        p95_response = 100      # 95th percentile under 100ms
        max_acceptable = 200    # Max 200ms
        throughput_rps = 20     # 20 requests per second sustained
        rate_limit = 2000       # 2000 requests per minute
    }
}

function Write-UserTestLog {
    param($Message, $Level = "INFO", $UserID = "", $Scenario = "")
    $timestamp = Get-Date -Format "HH:mm:ss.fff"
    $prefix = "[$timestamp]"
    if ($UserID) { $prefix += " [User-$UserID]" }
    if ($Scenario) { $prefix += " [$Scenario]" }
    
    $color = switch ($Level) {
        "ERROR" { "Red" }
        "WARN"  { "Yellow" }
        "INFO"  { "White" }
        "SUCCESS" { "Green" }
        "PERF" { "Cyan" }
        default { "Gray" }
    }
    
    Write-Host "$prefix $Message" -ForegroundColor $color
}

function Start-BitcoinCoreMock {
    Write-UserTestLog "Ensuring Bitcoin Core mock is running..." "INFO"
    
    $mockRunning = Get-NetTCPConnection -LocalPort 8332 -State Listen -ErrorAction SilentlyContinue
    if (-not $mockRunning) {
        Write-UserTestLog "Starting Bitcoin Core mock..." "INFO"
        $mockJob = Start-Job -ScriptBlock {
            Set-Location $args[0]
            python scripts\bitcoin-core-mock.py
        } -ArgumentList (Get-Location)
        
        Start-Sleep 3
        $mockRunning = Get-NetTCPConnection -LocalPort 8332 -State Listen -ErrorAction SilentlyContinue
        if (-not $mockRunning) {
            throw "Failed to start Bitcoin Core mock"
        }
    }
    Write-UserTestLog "✓ Bitcoin Core mock ready on port 8332" "SUCCESS"
}

function Start-SprintForTier {
    param($TierName)
    
    Write-UserTestLog "Starting Bitcoin Sprint for $TierName tier..." "INFO"
    
    # Stop any running Sprint
    Get-Process -Name "*sprint*" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
    Start-Sleep 2
    
    # Configure for tier
    $configMap = @{
        "FREE" = "config-free-stable.json"
        "PRO" = "config.json"
        "ENTERPRISE" = "config-enterprise-turbo.json"
    }
    
    $configFile = $configMap[$TierName]
    if (-not (Test-Path $configFile)) {
        throw "Config file not found: $configFile"
    }
    
    Copy-Item $configFile "config.json" -Force
    Write-UserTestLog "Applied config: $configFile" "INFO"
    
    # Set environment
    $env:RPC_NODES = "http://127.0.0.1:8332"
    $env:RPC_USER = "bitcoin"
    $env:RPC_PASS = "sprint123benchmark"
    $env:API_PORT = "8080"
    
    # Start Sprint
    $binaryMap = @{
        "FREE" = "bitcoin-sprint-free.exe"
        "PRO" = "bitcoin-sprint.exe"
        "ENTERPRISE" = "bitcoin-sprint-turbo.exe"
    }
    
    $binary = $binaryMap[$TierName]
    if (-not (Test-Path $binary)) {
        $binary = "bitcoin-sprint.exe"  # Fallback
    }
    
    $sprintJob = Start-Job -ScriptBlock {
        param($BinaryPath, $WorkDir)
        Set-Location $WorkDir
        & $BinaryPath
    } -ArgumentList (Resolve-Path $binary), (Get-Location)
    
    # Wait for startup
    $maxWait = 20
    for ($i = 0; $i -lt $maxWait; $i++) {
        Start-Sleep 1
        if (Get-NetTCPConnection -LocalPort 8080 -State Listen -ErrorAction SilentlyContinue) {
            Write-UserTestLog "✓ Sprint ready on port 8080" "SUCCESS"
            return $sprintJob
        }
    }
    
    throw "Sprint failed to start within $maxWait seconds"
}

function Test-ApiWarmup {
    param($TierName)
    
    Write-UserTestLog "Warming up API for $TierName..." "INFO"
    
    # Make several warmup requests
    for ($i = 1; $i -le 10; $i++) {
        try {
            $null = Invoke-RestMethod -Uri "http://localhost:8080/status" -TimeoutSec 5
            $null = Invoke-RestMethod -Uri "http://localhost:8080/latest" -TimeoutSec 5
        } catch {
            # Ignore warmup errors
        }
        Start-Sleep -Milliseconds 100
    }
    
    Write-UserTestLog "✓ API warmup complete" "SUCCESS"
}

function Invoke-UserScenario {
    param($ScenarioName, $ScenarioConfig, $UserID, $Duration, $Results)
    
    $endTime = (Get-Date).AddSeconds($Duration)
    $requestCount = 0
    $errorCount = 0
    $responseTimes = @()
    
    Write-UserTestLog "Starting $($ScenarioConfig.description)" "INFO" $UserID $ScenarioName
    
    while ((Get-Date) -lt $endTime) {
        foreach ($endpoint in $ScenarioConfig.endpoints) {
            try {
                $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
                $response = Invoke-RestMethod -Uri "http://localhost:8080$endpoint" -TimeoutSec 10
                $stopwatch.Stop()
                
                $responseTime = $stopwatch.ElapsedMilliseconds
                $responseTimes += $responseTime
                $requestCount++
                
                if ($DetailedLogs) {
                    Write-UserTestLog "Request to $endpoint`: $($responseTime)ms" "PERF" $UserID $ScenarioName
                }
                
                # Add to thread-safe results
                $Results.Value += [PSCustomObject]@{
                    UserID = $UserID
                    Scenario = $ScenarioName
                    Endpoint = $endpoint
                    ResponseTime = $responseTime
                    Timestamp = Get-Date
                    Success = $true
                }
                
            } catch {
                $errorCount++
                if ($DetailedLogs) {
                    Write-UserTestLog "Request to $endpoint failed: $($_.Exception.Message)" "ERROR" $UserID $ScenarioName
                }
                
                $Results.Value += [PSCustomObject]@{
                    UserID = $UserID
                    Scenario = $ScenarioName
                    Endpoint = $endpoint
                    ResponseTime = -1
                    Timestamp = Get-Date
                    Success = $false
                    Error = $_.Exception.Message
                }
            }
        }
        
        # Wait for next iteration based on scenario pattern
        if ($ScenarioConfig.pattern -eq "burst") {
            Start-Sleep -Milliseconds 500  # Brief pause in burst mode
        } else {
            Start-Sleep -Seconds $ScenarioConfig.interval
        }
    }
    
    if ($responseTimes.Count -gt 0) {
        $avgResponse = ($responseTimes | Measure-Object -Average).Average
        Write-UserTestLog "Completed: $requestCount requests, $errorCount errors, avg $([math]::Round($avgResponse, 1))ms" "SUCCESS" $UserID $ScenarioName
    } else {
        Write-UserTestLog "Completed: All $requestCount requests failed" "ERROR" $UserID $ScenarioName
    }
}

function Test-RealUserPerformance {
    param($TierName)
    
    Write-UserTestLog "=== TESTING REAL USER PERFORMANCE FOR $TierName ===" "INFO"
    
    # Thread-safe results collection
    $allResults = [ref]@()
    
    if ($ColdStart) {
        Write-UserTestLog "Cold start test - no warmup" "INFO"
    } elseif ($WarmupFirst) {
        Test-ApiWarmup $TierName
    }
    
    # Start concurrent user simulations
    $jobs = @()
    $scenarios = $USER_SCENARIOS.Keys | Get-Random -Count $Users
    
    for ($userID = 1; $userID -le $Users; $userID++) {
        $scenarioName = $scenarios[($userID - 1) % $scenarios.Count]
        $scenario = $USER_SCENARIOS[$scenarioName]
        
        $job = Start-Job -ScriptBlock {
            param($ScenarioName, $ScenarioConfig, $UserID, $Duration, $DetailedLogs, $Results)
            
            # Re-define the function inside the job
            function Write-UserTestLog {
                param($Message, $Level = "INFO", $UserID = "", $Scenario = "")
                $timestamp = Get-Date -Format "HH:mm:ss.fff"
                $prefix = "[$timestamp]"
                if ($UserID) { $prefix += " [User-$UserID]" }
                if ($Scenario) { $prefix += " [$Scenario]" }
                
                $color = switch ($Level) {
                    "ERROR" { "Red" }
                    "WARN"  { "Yellow" }
                    "INFO"  { "White" }
                    "SUCCESS" { "Green" }
                    "PERF" { "Cyan" }
                    default { "Gray" }
                }
                
                Write-Host "$prefix $Message" -ForegroundColor $color
            }
            
            # Execute scenario
            $endTime = (Get-Date).AddSeconds($Duration)
            $requestCount = 0
            $errorCount = 0
            $responseTimes = @()
            
            Write-UserTestLog "Starting $($ScenarioConfig.description)" "INFO" $UserID $ScenarioName
            
            while ((Get-Date) -lt $endTime) {
                foreach ($endpoint in $ScenarioConfig.endpoints) {
                    try {
                        $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
                        $response = Invoke-RestMethod -Uri "http://localhost:8080$endpoint" -TimeoutSec 10
                        $stopwatch.Stop()
                        
                        $responseTime = $stopwatch.ElapsedMilliseconds
                        $responseTimes += $responseTime
                        $requestCount++
                        
                        if ($DetailedLogs) {
                            Write-UserTestLog "Request to $endpoint`: $($responseTime)ms" "PERF" $UserID $ScenarioName
                        }
                        
                        # Return result
                        [PSCustomObject]@{
                            UserID = $UserID
                            Scenario = $ScenarioName
                            Endpoint = $endpoint
                            ResponseTime = $responseTime
                            Timestamp = Get-Date
                            Success = $true
                        }
                        
                    } catch {
                        $errorCount++
                        if ($DetailedLogs) {
                            Write-UserTestLog "Request to $endpoint failed: $($_.Exception.Message)" "ERROR" $UserID $ScenarioName
                        }
                        
                        [PSCustomObject]@{
                            UserID = $UserID
                            Scenario = $ScenarioName
                            Endpoint = $endpoint
                            ResponseTime = -1
                            Timestamp = Get-Date
                            Success = $false
                            Error = $_.Exception.Message
                        }
                    }
                }
                
                # Wait for next iteration
                if ($ScenarioConfig.pattern -eq "burst") {
                    Start-Sleep -Milliseconds 500
                } else {
                    Start-Sleep -Seconds $ScenarioConfig.interval
                }
            }
            
            Write-UserTestLog "User $UserID completed: $requestCount requests, $errorCount errors" "SUCCESS" $UserID $ScenarioName
            
        } -ArgumentList $scenarioName, $scenario, $userID, $TestDuration, $DetailedLogs, $allResults
        
        $jobs += $job
        Start-Sleep -Milliseconds 500  # Stagger user starts
    }
    
    Write-UserTestLog "Started $($jobs.Count) concurrent user simulations for $TestDuration seconds..." "INFO"
    
    # Wait for all users to complete
    $results = $jobs | Wait-Job | Receive-Job
    $jobs | Remove-Job
    
    return $results
}

function Measure-RealWorldMetrics {
    param($Results, $TierName)
    
    Write-UserTestLog "=== ANALYZING REAL WORLD PERFORMANCE ===" "INFO"
    
    $successfulRequests = $Results | Where-Object { $_.Success -eq $true }
    $failedRequests = $Results | Where-Object { $_.Success -eq $false }
    
    if ($successfulRequests.Count -eq 0) {
        Write-UserTestLog "❌ No successful requests to analyze!" "ERROR"
        return $null
    }
    
    # Calculate metrics
    $responseTimes = $successfulRequests.ResponseTime
    $totalRequests = $Results.Count
    $successRate = ($successfulRequests.Count / $totalRequests) * 100
    
    $avgResponse = ($responseTimes | Measure-Object -Average).Average
    $medianResponse = ($responseTimes | Sort-Object)[[math]::Floor($responseTimes.Count / 2)]
    $minResponse = ($responseTimes | Measure-Object -Minimum).Minimum
    $maxResponse = ($responseTimes | Measure-Object -Maximum).Maximum
    
    # Calculate percentiles
    $sortedTimes = $responseTimes | Sort-Object
    $p95Index = [math]::Floor($sortedTimes.Count * 0.95)
    $p99Index = [math]::Floor($sortedTimes.Count * 0.99)
    $p95Response = $sortedTimes[$p95Index]
    $p99Response = $sortedTimes[$p99Index]
    
    # Calculate throughput (requests per second)
    $testDurationActual = ($Results | Measure-Object -Property Timestamp -Maximum).Maximum - ($Results | Measure-Object -Property Timestamp -Minimum).Minimum
    $throughput = $totalRequests / $testDurationActual.TotalSeconds
    
    $metrics = [PSCustomObject]@{
        TierName = $TierName
        TotalRequests = $totalRequests
        SuccessfulRequests = $successfulRequests.Count
        FailedRequests = $failedRequests.Count
        SuccessRate = $successRate
        AverageResponseTime = $avgResponse
        MedianResponseTime = $medianResponse
        MinResponseTime = $minResponse
        MaxResponseTime = $maxResponse
        P95ResponseTime = $p95Response
        P99ResponseTime = $p99Response
        ThroughputRPS = $throughput
        TestDuration = $testDurationActual.TotalSeconds
    }
    
    # Display results
    Write-UserTestLog ""
    Write-UserTestLog "📊 REAL USER PERFORMANCE RESULTS FOR $TierName" "SUCCESS"
    Write-UserTestLog "----------------------------------------" "INFO"
    Write-UserTestLog "Total Requests:    $totalRequests" "INFO"
    Write-UserTestLog "Success Rate:      $([math]::Round($successRate, 2))%" $(if($successRate -gt 95){"SUCCESS"}else{"WARN"})
    Write-UserTestLog "Average Response:  $([math]::Round($avgResponse, 1))ms" $(if($avgResponse -lt $TIER_EXPECTATIONS[$TierName].avg_response){"SUCCESS"}else{"WARN"})
    Write-UserTestLog "Median Response:   $([math]::Round($medianResponse, 1))ms" "INFO"
    Write-UserTestLog "95th Percentile:   $([math]::Round($p95Response, 1))ms" $(if($p95Response -lt $TIER_EXPECTATIONS[$TierName].p95_response){"SUCCESS"}else{"WARN"})
    Write-UserTestLog "99th Percentile:   $([math]::Round($p99Response, 1))ms" "INFO"
    Write-UserTestLog "Min/Max Response:  $minResponse ms / $maxResponse ms" "INFO"
    Write-UserTestLog "Throughput:        $([math]::Round($throughput, 2)) requests/sec" $(if($throughput -gt $TIER_EXPECTATIONS[$TierName].throughput_rps){"SUCCESS"}else{"WARN"})
    Write-UserTestLog ""
    
    return $metrics
}

function Test-UserExperienceValidation {
    param($Metrics, $TierName)
    
    Write-UserTestLog "=== USER EXPERIENCE VALIDATION ===" "INFO"
    
    $expectations = $TIER_EXPECTATIONS[$TierName]
    $passed = $true
    
    # Test average response time
    if ($Metrics.AverageResponseTime -le $expectations.avg_response) {
        Write-UserTestLog "✅ Average response time acceptable: $([math]::Round($Metrics.AverageResponseTime, 1))ms ≤ $($expectations.avg_response)ms" "SUCCESS"
    } else {
        Write-UserTestLog "❌ Average response time too slow: $([math]::Round($Metrics.AverageResponseTime, 1))ms > $($expectations.avg_response)ms" "ERROR"
        $passed = $false
    }
    
    # Test 95th percentile
    if ($Metrics.P95ResponseTime -le $expectations.p95_response) {
        Write-UserTestLog "✅ 95th percentile acceptable: $([math]::Round($Metrics.P95ResponseTime, 1))ms ≤ $($expectations.p95_response)ms" "SUCCESS"
    } else {
        Write-UserTestLog "❌ 95th percentile too slow: $([math]::Round($Metrics.P95ResponseTime, 1))ms > $($expectations.p95_response)ms" "ERROR"
        $passed = $false
    }
    
    # Test maximum response time
    if ($Metrics.MaxResponseTime -le $expectations.max_acceptable) {
        Write-UserTestLog "✅ Maximum response time acceptable: $($Metrics.MaxResponseTime)ms ≤ $($expectations.max_acceptable)ms" "SUCCESS"
    } else {
        Write-UserTestLog "❌ Maximum response time unacceptable: $($Metrics.MaxResponseTime)ms > $($expectations.max_acceptable)ms" "ERROR"
        $passed = $false
    }
    
    # Test success rate
    if ($Metrics.SuccessRate -ge 95) {
        Write-UserTestLog "✅ Success rate acceptable: $([math]::Round($Metrics.SuccessRate, 2))% ≥ 95%" "SUCCESS"
    } else {
        Write-UserTestLog "❌ Success rate too low: $([math]::Round($Metrics.SuccessRate, 2))% < 95%" "ERROR"
        $passed = $false
    }
    
    # Test throughput
    if ($Metrics.ThroughputRPS -ge $expectations.throughput_rps) {
        Write-UserTestLog "✅ Throughput acceptable: $([math]::Round($Metrics.ThroughputRPS, 2)) RPS ≥ $($expectations.throughput_rps) RPS" "SUCCESS"
    } else {
        Write-UserTestLog "❌ Throughput too low: $([math]::Round($Metrics.ThroughputRPS, 2)) RPS < $($expectations.throughput_rps) RPS" "ERROR"
        $passed = $false
    }
    
    return $passed
}

function Export-PerformanceReport {
    param($AllMetrics, $AllResults)
    
    $report = @{
        test_timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
        test_duration = $TestDuration
        concurrent_users = $Users
        test_mode = if ($ColdStart) { "cold_start" } elseif ($WarmupFirst) { "warmed_up" } else { "standard" }
        tier_results = $AllMetrics
        detailed_results = $AllResults
        user_scenarios = $USER_SCENARIOS
        performance_expectations = $TIER_EXPECTATIONS
        environment = @{
            os = $env:OS
            computername = $env:COMPUTERNAME
            powershell_version = $PSVersionTable.PSVersion.ToString()
        }
    }
    
    try {
        $report | ConvertTo-Json -Depth 10 | Out-File $ReportFile -Encoding UTF8
        Write-UserTestLog "📄 Performance report saved: $ReportFile" "SUCCESS"
    } catch {
        Write-UserTestLog "⚠️ Failed to save report: $($_.Exception.Message)" "WARN"
    }
}

# Main execution
try {
    Write-UserTestLog "=========================================" "INFO"
    Write-UserTestLog "REAL USER API PERFORMANCE TESTING" "INFO"
    Write-UserTestLog "=========================================" "INFO"
    Write-UserTestLog "Test Duration: $TestDuration seconds" "INFO"
    Write-UserTestLog "Concurrent Users: $Users" "INFO"
    Write-UserTestLog "Target Tier(s): $Tier" "INFO"
    Write-UserTestLog ""
    
    # Start Bitcoin Core mock
    Start-BitcoinCoreMock
    
    $allMetrics = @()
    $allResults = @()
    $allPassed = $true
    
    # Test specified tiers
    $tiersToTest = if ($Tier -eq "ALL") { @("FREE", "PRO", "ENTERPRISE") } else { @($Tier) }
    
    foreach ($tierName in $tiersToTest) {
        try {
            Write-UserTestLog ""
            Write-UserTestLog "🚀 Starting real user test for $tierName tier..." "INFO"
            
            # Start Sprint for this tier
            $sprintJob = Start-SprintForTier $tierName
            
            # Run real user performance test
            $results = Test-RealUserPerformance $tierName
            $allResults += $results
            
            # Analyze metrics
            $metrics = Measure-RealWorldMetrics $results $tierName
            if ($metrics) {
                $allMetrics += $metrics
                
                # Validate user experience
                $tierPassed = Test-UserExperienceValidation $metrics $tierName
                if (-not $tierPassed) {
                    $allPassed = $false
                }
            } else {
                $allPassed = $false
            }
            
            # Clean up
            if ($sprintJob) {
                Stop-Job $sprintJob -ErrorAction SilentlyContinue
                Remove-Job $sprintJob -ErrorAction SilentlyContinue
            }
            
        } catch {
            Write-UserTestLog "❌ Tier $tierName failed: $($_.Exception.Message)" "ERROR"
            $allPassed = $false
        }
    }
    
    # Generate performance report
    Export-PerformanceReport $allMetrics $allResults
    
    # Final summary
    Write-UserTestLog ""
    Write-UserTestLog "=========================================" "INFO"
    Write-UserTestLog "REAL USER TESTING SUMMARY" "INFO"
    Write-UserTestLog "=========================================" "INFO"
    
    foreach ($metric in $allMetrics) {
        $avgTime = [math]::Round($metric.AverageResponseTime, 1)
        $successRate = [math]::Round($metric.SuccessRate, 1)
        $throughput = [math]::Round($metric.ThroughputRPS, 2)
        
        Write-UserTestLog "$($metric.TierName): $avgTime ms avg, $successRate% success, $throughput RPS" "INFO"
    }
    
    Write-UserTestLog ""
    if ($allPassed) {
        Write-UserTestLog "🎉 ALL TIERS MEET REAL USER PERFORMANCE EXPECTATIONS!" "SUCCESS"
        exit 0
    } else {
        Write-UserTestLog "💥 SOME TIERS FAILED TO MEET USER EXPECTATIONS!" "ERROR"
        exit 1
    }
    
} catch {
    Write-UserTestLog "💥 REAL USER TESTING CRASHED: $($_.Exception.Message)" "ERROR"
    exit 2
} finally {
    # Cleanup
    Get-Process -Name "*sprint*" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
}
