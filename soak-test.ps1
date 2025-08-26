#!/usr/bin/env pwsh
#
# Soak Test - Long-Running Stability Test (30+ minutes)
# Tests system stability and memory leaks over extended periods
#

param(
    [int]$DurationMinutes = 35,
    [int]$ConcurrentUsers = 8,
    [int]$RequestsPerMinute = 60,
    [string]$TargetEndpoint = "/status",
    [int]$MemoryCheckInterval = 5,
    [switch]$EnableDetailedLogging,
    [switch]$SaveResults
)

Write-Host "⏰ SOAK TEST - LONG-RUNNING STABILITY" -ForegroundColor Cyan
Write-Host "====================================" -ForegroundColor Cyan
Write-Host "Generated: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Gray
Write-Host ""

Write-Host "📊 Test Configuration:" -ForegroundColor White
Write-Host "   Duration: $DurationMinutes minutes" -ForegroundColor Gray
Write-Host "   Concurrent Users: $ConcurrentUsers" -ForegroundColor Gray
Write-Host "   Requests/Minute: $RequestsPerMinute" -ForegroundColor Gray
Write-Host "   Target Endpoint: $TargetEndpoint" -ForegroundColor Gray
Write-Host "   Memory Checks: Every $MemoryCheckInterval minutes" -ForegroundColor Gray
Write-Host ""

# Check if Bitcoin Sprint is running
$sprintRunning = Get-NetTCPConnection -LocalPort 8080 -State Listen -ErrorAction SilentlyContinue
if (-not $sprintRunning) {
    Write-Host "❌ Bitcoin Sprint is not running on port 8080" -ForegroundColor Red
    Write-Host "Please start Bitcoin Sprint first to run soak test." -ForegroundColor Yellow
    exit 1
}

Write-Host "✅ Bitcoin Sprint detected on port 8080" -ForegroundColor Green

# Get initial system state
$sprintProcess = Get-Process | Where-Object { $_.ProcessName -like "*sprint*" -or $_.ProcessName -like "*bitcoin*" } | Select-Object -First 1
if ($sprintProcess) {
    $initialMemory = $sprintProcess.WorkingSet64 / 1MB
    Write-Host "📈 Initial Sprint Memory: $([math]::Round($initialMemory, 1)) MB" -ForegroundColor Gray
} else {
    Write-Host "⚠️ Could not identify Sprint process for memory monitoring" -ForegroundColor Yellow
    $initialMemory = 0
}

Write-Host ""

# Initialize tracking variables
$global:soakResults = @{
    StartTime = Get-Date
    EndTime = $null
    TotalRequests = 0
    SuccessfulRequests = 0
    FailedRequests = 0
    ResponseTimes = [System.Collections.ArrayList]::new()
    MemoryReadings = [System.Collections.ArrayList]::new()
    ErrorBursts = [System.Collections.ArrayList]::new()
    PerformanceDegradation = @{}
    SystemStability = @{}
}

# Memory monitoring function
function Start-MemoryMonitoring {
    $memoryMonitorScript = {
        param($intervalMinutes, $durationMinutes, $enableLogging)
        
        $endTime = (Get-Date).AddMinutes($durationMinutes)
        $memoryData = @()
        
        while ((Get-Date) -lt $endTime) {
            try {
                $sprintProcess = Get-Process | Where-Object { $_.ProcessName -like "*sprint*" -or $_.ProcessName -like "*bitcoin*" } | Select-Object -First 1
                
                if ($sprintProcess) {
                    $memoryMB = $sprintProcess.WorkingSet64 / 1MB
                    $cpuPercent = $sprintProcess.CPU
                    $handleCount = $sprintProcess.HandleCount
                    $threadCount = $sprintProcess.Threads.Count
                    
                    $reading = @{
                        Timestamp = Get-Date
                        MemoryMB = [math]::Round($memoryMB, 1)
                        CPUTime = $cpuPercent
                        Handles = $handleCount
                        Threads = $threadCount
                    }
                    
                    $memoryData += $reading
                    
                    if ($enableLogging) {
                        $timeStr = Get-Date -Format "HH:mm:ss"
                        Write-Host "[$timeStr] Memory: $([math]::Round($memoryMB, 1))MB, Handles: $handleCount, Threads: $threadCount" -ForegroundColor Cyan
                    }
                }
                
            } catch {
                if ($enableLogging) {
                    Write-Host "[$(Get-Date -Format 'HH:mm:ss')] Memory monitor error: $($_.Exception.Message)" -ForegroundColor Red
                }
            }
            
            Start-Sleep -Seconds ($intervalMinutes * 60)
        }
        
        return $memoryData
    }
    
    return Start-Job -ScriptBlock $memoryMonitorScript -ArgumentList $MemoryCheckInterval, $DurationMinutes, $EnableDetailedLogging
}

# Continuous load generator
function Start-ContinuousLoad {
    $loadGeneratorScript = {
        param($endpoint, $durationMinutes, $requestsPerMinute, $enableLogging)
        
        $endTime = (Get-Date).AddMinutes($durationMinutes)
        $requestInterval = 60.0 / $requestsPerMinute  # seconds between requests
        $results = @{
            Requests = 0
            Successes = 0
            Failures = 0
            ResponseTimes = @()
            ErrorBursts = @()
        }
        
        $consecutiveErrors = 0
        $burstThreshold = 5
        
        while ((Get-Date) -lt $endTime) {
            $requestStart = Get-Date
            $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
            
            try {
                $response = Invoke-RestMethod -Uri "http://localhost:8080$endpoint" -TimeoutSec 10
                $stopwatch.Stop()
                
                $results.Requests++
                $results.Successes++
                $results.ResponseTimes += $stopwatch.ElapsedMilliseconds
                $consecutiveErrors = 0
                
                if ($enableLogging -and ($results.Requests % 60 -eq 0)) {
                    $timeStr = Get-Date -Format "HH:mm:ss"
                    Write-Host "[$timeStr] Load: $($results.Requests) requests, avg: $([math]::Round(($results.ResponseTimes | Measure-Object -Average).Average, 1))ms" -ForegroundColor Green
                }
                
            } catch {
                $stopwatch.Stop()
                $results.Requests++
                $results.Failures++
                $consecutiveErrors++
                
                # Track error bursts
                if ($consecutiveErrors -eq $burstThreshold) {
                    $results.ErrorBursts += @{
                        Timestamp = Get-Date
                        ConsecutiveErrors = $consecutiveErrors
                    }
                    
                    if ($enableLogging) {
                        Write-Host "[$(Get-Date -Format 'HH:mm:ss')] ERROR BURST: $consecutiveErrors consecutive failures" -ForegroundColor Red
                    }
                }
            }
            
            # Wait for next request interval
            $elapsed = ((Get-Date) - $requestStart).TotalSeconds
            $sleepTime = [Math]::Max(0, $requestInterval - $elapsed)
            if ($sleepTime -gt 0) {
                Start-Sleep -Seconds $sleepTime
            }
        }
        
        return $results
    }
    
    # Start multiple load generators for concurrent users
    $loadJobs = @()
    for ($i = 1; $i -le $ConcurrentUsers; $i++) {
        $job = Start-Job -ScriptBlock $loadGeneratorScript -ArgumentList $TargetEndpoint, $DurationMinutes, ($RequestsPerMinute / $ConcurrentUsers), $EnableDetailedLogging
        $loadJobs += $job
    }
    
    return $loadJobs
}

# Performance monitoring function
function Start-PerformanceMonitoring {
    $perfMonitorScript = {
        param($endpoint, $durationMinutes, $sampleInterval)
        
        $endTime = (Get-Date).AddMinutes($durationMinutes)
        $performanceData = @()
        
        while ((Get-Date) -lt $endTime) {
            $sampleStart = Get-Date
            $sampleTimes = @()
            
            # Take 10 quick samples
            for ($i = 1; $i -le 10; $i++) {
                try {
                    $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
                    Invoke-RestMethod -Uri "http://localhost:8080$endpoint" -TimeoutSec 5 | Out-Null
                    $stopwatch.Stop()
                    $sampleTimes += $stopwatch.ElapsedMilliseconds
                } catch {
                    # Skip failed samples
                }
                Start-Sleep -Milliseconds 200
            }
            
            if ($sampleTimes.Count -gt 0) {
                $avgSample = ($sampleTimes | Measure-Object -Average).Average
                $performanceData += @{
                    Timestamp = $sampleStart
                    AverageResponseTime = $avgSample
                    SampleCount = $sampleTimes.Count
                }
            }
            
            Start-Sleep -Seconds ($sampleInterval * 60)
        }
        
        return $performanceData
    }
    
    return Start-Job -ScriptBlock $perfMonitorScript -ArgumentList $TargetEndpoint, $DurationMinutes, 2
}

Write-Host "🚀 Starting Soak Test..." -ForegroundColor Green
Write-Host "This will run for $DurationMinutes minutes. Please be patient..." -ForegroundColor Yellow
Write-Host ""

$testStartTime = Get-Date

# Start monitoring jobs
Write-Host "📊 Starting monitoring systems..." -ForegroundColor White
$memoryJob = Start-MemoryMonitoring
$performanceJob = Start-PerformanceMonitoring
$loadJobs = Start-ContinuousLoad

Write-Host "   ✅ Memory monitoring started" -ForegroundColor Green
Write-Host "   ✅ Performance monitoring started" -ForegroundColor Green
Write-Host "   ✅ Load generation started ($ConcurrentUsers users)" -ForegroundColor Green
Write-Host ""

# Progress tracking
$progressInterval = [Math]::Max(1, [Math]::Floor($DurationMinutes / 10))  # 10 progress updates
$nextProgressUpdate = $progressInterval

Write-Host "⏱️ Test Progress:" -ForegroundColor White

for ($minute = 1; $minute -le $DurationMinutes; $minute++) {
    Start-Sleep -Seconds 60
    
    if ($minute -eq $nextProgressUpdate -or $minute -eq $DurationMinutes) {
        $percentage = [Math]::Round($minute / $DurationMinutes * 100)
        $remaining = $DurationMinutes - $minute
        
        Write-Host "   $minute/$DurationMinutes minutes ($percentage%) - $remaining minutes remaining" -ForegroundColor Gray
        
        # Quick health check
        try {
            $healthStart = [System.Diagnostics.Stopwatch]::StartNew()
            Invoke-RestMethod -Uri "http://localhost:8080$TargetEndpoint" -TimeoutSec 5 | Out-Null
            $healthStart.Stop()
            Write-Host "   Health check: $($healthStart.ElapsedMilliseconds)ms" -ForegroundColor $(if($healthStart.ElapsedMilliseconds -lt 200){'Green'}else{'Yellow'})
        } catch {
            Write-Host "   Health check: FAILED" -ForegroundColor Red
        }
        
        $nextProgressUpdate += $progressInterval
    }
}

$testEndTime = Get-Date
$actualDuration = ($testEndTime - $testStartTime).TotalMinutes

Write-Host ""
Write-Host "✅ Soak test completed!" -ForegroundColor Green
Write-Host "Collecting results..." -ForegroundColor Yellow
Write-Host ""

# Collect results from all monitoring jobs
Write-Host "📊 SOAK TEST RESULTS" -ForegroundColor Cyan
Write-Host "===================" -ForegroundColor Cyan
Write-Host ""

# Memory monitoring results
if ($memoryJob) {
    $memoryData = Receive-Job $memoryJob -Wait
    Remove-Job $memoryJob
    
    if ($memoryData -and $memoryData.Count -gt 0) {
        $finalMemory = $memoryData[-1].MemoryMB
        $peakMemory = ($memoryData | Measure-Object -Property MemoryMB -Maximum).Maximum
        $memoryGrowth = $finalMemory - $initialMemory
        
        Write-Host "💾 Memory Analysis:" -ForegroundColor White
        Write-Host "   Initial Memory: $([math]::Round($initialMemory, 1)) MB" -ForegroundColor Gray
        Write-Host "   Final Memory: $([math]::Round($finalMemory, 1)) MB" -ForegroundColor Gray
        Write-Host "   Peak Memory: $([math]::Round($peakMemory, 1)) MB" -ForegroundColor Gray
        Write-Host "   Memory Growth: $([math]::Round($memoryGrowth, 1)) MB" -ForegroundColor $(if($memoryGrowth -lt 50){'Green'}elseif($memoryGrowth -lt 100){'Yellow'}else{'Red'})
        
        if ($memoryGrowth -lt 10) {
            Write-Host "   ✅ Excellent memory stability" -ForegroundColor Green
        } elseif ($memoryGrowth -lt 50) {
            Write-Host "   👍 Good memory stability" -ForegroundColor Yellow
        } else {
            Write-Host "   ⚠️ Potential memory leak detected" -ForegroundColor Red
        }
        
        $global:soakResults.MemoryReadings = $memoryData
    }
}

# Load generation results
$allLoadResults = @{
    TotalRequests = 0
    TotalSuccesses = 0
    TotalFailures = 0
    AllResponseTimes = @()
    AllErrorBursts = @()
}

foreach ($job in $loadJobs) {
    $loadData = Receive-Job $job -Wait
    Remove-Job $job
    
    if ($loadData) {
        $allLoadResults.TotalRequests += $loadData.Requests
        $allLoadResults.TotalSuccesses += $loadData.Successes
        $allLoadResults.TotalFailures += $loadData.Failures
        $allLoadResults.AllResponseTimes += $loadData.ResponseTimes
        $allLoadResults.AllErrorBursts += $loadData.ErrorBursts
    }
}

Write-Host ""
Write-Host "⚡ Load Test Results:" -ForegroundColor White
Write-Host "   Duration: $([math]::Round($actualDuration, 1)) minutes" -ForegroundColor Gray
Write-Host "   Total Requests: $($allLoadResults.TotalRequests)" -ForegroundColor Gray
Write-Host "   Successful: $($allLoadResults.TotalSuccesses) ($([math]::Round($allLoadResults.TotalSuccesses/$allLoadResults.TotalRequests*100, 1))%)" -ForegroundColor $(if($allLoadResults.TotalSuccesses -eq $allLoadResults.TotalRequests){'Green'}else{'Yellow'})
Write-Host "   Failed: $($allLoadResults.TotalFailures) ($([math]::Round($allLoadResults.TotalFailures/$allLoadResults.TotalRequests*100, 1))%)" -ForegroundColor $(if($allLoadResults.TotalFailures -eq 0){'Green'}else{'Red'})
Write-Host "   Requests/Minute: $([math]::Round($allLoadResults.TotalRequests/$actualDuration, 1))" -ForegroundColor Gray

if ($allLoadResults.AllResponseTimes.Count -gt 0) {
    $avgResponse = ($allLoadResults.AllResponseTimes | Measure-Object -Average).Average
    $sorted = $allLoadResults.AllResponseTimes | Sort-Object
    $p95Index = [math]::Floor($sorted.Count * 0.95)
    $p95Response = $sorted[$p95Index]
    
    Write-Host "   Average Response: $([math]::Round($avgResponse, 1))ms" -ForegroundColor $(if($avgResponse -lt 100){'Green'}elseif($avgResponse -lt 200){'Yellow'}else{'Red'})
    Write-Host "   95th Percentile: $([math]::Round($p95Response, 1))ms" -ForegroundColor $(if($p95Response -lt 200){'Green'}elseif($p95Response -lt 500){'Yellow'}else{'Red'})
}

# Error burst analysis
if ($allLoadResults.AllErrorBursts.Count -gt 0) {
    Write-Host ""
    Write-Host "🚨 Error Burst Analysis:" -ForegroundColor Red
    Write-Host "   Error Bursts Detected: $($allLoadResults.AllErrorBursts.Count)" -ForegroundColor Red
    foreach ($burst in $allLoadResults.AllErrorBursts | Select-Object -First 5) {
        Write-Host "   - $(Get-Date $burst.Timestamp -Format 'HH:mm:ss'): $($burst.ConsecutiveErrors) consecutive errors" -ForegroundColor Yellow
    }
} else {
    Write-Host ""
    Write-Host "✅ No error bursts detected - excellent stability!" -ForegroundColor Green
}

# Performance monitoring results
if ($performanceJob) {
    $performanceData = Receive-Job $performanceJob -Wait
    Remove-Job $performanceJob
    
    if ($performanceData -and $performanceData.Count -gt 1) {
        Write-Host ""
        Write-Host "📈 Performance Degradation Analysis:" -ForegroundColor White
        
        $earlyPerformance = ($performanceData | Select-Object -First 3 | Measure-Object -Property AverageResponseTime -Average).Average
        $latePerformance = ($performanceData | Select-Object -Last 3 | Measure-Object -Property AverageResponseTime -Average).Average
        $degradation = $latePerformance - $earlyPerformance
        
        Write-Host "   Early Performance: $([math]::Round($earlyPerformance, 1))ms" -ForegroundColor Gray
        Write-Host "   Late Performance: $([math]::Round($latePerformance, 1))ms" -ForegroundColor Gray
        Write-Host "   Degradation: $([math]::Round($degradation, 1))ms" -ForegroundColor $(if($degradation -lt 10){'Green'}elseif($degradation -lt 50){'Yellow'}else{'Red'})
        
        if ($degradation -lt 5) {
            Write-Host "   ✅ Excellent performance stability" -ForegroundColor Green
        } elseif ($degradation -lt 20) {
            Write-Host "   👍 Good performance stability" -ForegroundColor Yellow
        } else {
            Write-Host "   ⚠️ Performance degradation over time" -ForegroundColor Red
        }
    }
}

# Overall soak test assessment
Write-Host ""
Write-Host "🎯 SOAK TEST ASSESSMENT" -ForegroundColor Cyan
Write-Host "=======================" -ForegroundColor Cyan

$successRate = if ($allLoadResults.TotalRequests -gt 0) { $allLoadResults.TotalSuccesses / $allLoadResults.TotalRequests * 100 } else { 0 }
$memoryStable = $memoryGrowth -lt 50
$performanceStable = $degradation -lt 20
$noErrorBursts = $allLoadResults.AllErrorBursts.Count -eq 0

if ($successRate -ge 99 -and $memoryStable -and $performanceStable -and $noErrorBursts) {
    Write-Host "✅ EXCELLENT - System demonstrates exceptional long-term stability" -ForegroundColor Green
    Write-Host "✅ Ready for production deployment with confidence" -ForegroundColor Green
    $soakGrade = "A"
} elseif ($successRate -ge 95 -and $memoryStable -and $performanceStable) {
    Write-Host "👍 GOOD - System shows good long-term stability" -ForegroundColor Yellow
    Write-Host "👍 Suitable for production with monitoring" -ForegroundColor Yellow
    $soakGrade = "B"
} elseif ($successRate -ge 90 -and $memoryGrowth -lt 100) {
    Write-Host "⚠️ FAIR - System shows some stability concerns" -ForegroundColor Orange
    Write-Host "⚠️ Consider optimization before long-term deployment" -ForegroundColor Orange
    $soakGrade = "C"
} else {
    Write-Host "❌ POOR - System shows significant stability issues" -ForegroundColor Red
    Write-Host "❌ Not suitable for long-term production use" -ForegroundColor Red
    $soakGrade = "D"
}

# Save results if requested
if ($SaveResults) {
    Write-Host ""
    Write-Host "💾 Saving Soak Test Results..." -ForegroundColor Cyan
    
    $timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $reportFile = "soak-test-$timestamp.json"
    
    $soakReport = @{
        test_info = @{
            timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
            test_type = "Soak Test"
            planned_duration_minutes = $DurationMinutes
            actual_duration_minutes = [math]::Round($actualDuration, 2)
            concurrent_users = $ConcurrentUsers
            target_requests_per_minute = $RequestsPerMinute
            target_endpoint = $TargetEndpoint
        }
        stability_metrics = @{
            success_rate_percent = [math]::Round($successRate, 2)
            total_requests = $allLoadResults.TotalRequests
            failed_requests = $allLoadResults.TotalFailures
            error_bursts_detected = $allLoadResults.AllErrorBursts.Count
            memory_growth_mb = [math]::Round($memoryGrowth, 2)
            performance_degradation_ms = [math]::Round($degradation, 2)
        }
        soak_assessment = @{
            grade = $soakGrade
            memory_stable = $memoryStable
            performance_stable = $performanceStable
            production_ready = ($soakGrade -in @("A", "B"))
        }
        detailed_data = @{
            memory_readings = $memoryData
            performance_samples = $performanceData
            error_bursts = $allLoadResults.AllErrorBursts
        }
    }
    
    try {
        $soakReport | ConvertTo-Json -Depth 6 | Out-File $reportFile -Encoding UTF8
        Write-Host "   ✅ Report saved: $reportFile" -ForegroundColor Green
        Write-Host "   📊 Contains detailed stability analysis" -ForegroundColor Gray
    } catch {
        Write-Host "   ⚠️ Failed to save report: $($_.Exception.Message)" -ForegroundColor Yellow
    }
}

Write-Host ""
Write-Host "📄 Soak test completed at $(Get-Date -Format 'HH:mm:ss')" -ForegroundColor Gray
Write-Host "🚀 For heavy load testing, use: .\heavy-load-test.ps1" -ForegroundColor Gray
Write-Host "📊 For tier comparison, use: .\tier-comparison-test.ps1" -ForegroundColor Gray
