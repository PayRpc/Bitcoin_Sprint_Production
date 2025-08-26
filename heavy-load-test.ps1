#!/usr/bin/env pwsh
#
# Heavy Load Test - 20+ Concurrent Clients
# Tests system behavior under high concurrent load
#

param(
    [int]$ConcurrentClients = 25,
    [int]$RequestsPerClient = 50,
    [int]$TestDurationMinutes = 5,
    [string]$TargetEndpoint = "/status",
    [switch]$SaveResults,
    [switch]$RealTimeMonitoring
)

Write-Host "🚀 HEAVY LOAD TEST - CONCURRENT CLIENTS" -ForegroundColor Cyan
Write-Host "=======================================" -ForegroundColor Cyan
Write-Host "Generated: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Gray
Write-Host ""

Write-Host "📊 Test Configuration:" -ForegroundColor White
Write-Host "   Concurrent Clients: $ConcurrentClients" -ForegroundColor Gray
Write-Host "   Requests per Client: $RequestsPerClient" -ForegroundColor Gray
Write-Host "   Test Duration: $TestDurationMinutes minutes" -ForegroundColor Gray
Write-Host "   Target Endpoint: $TargetEndpoint" -ForegroundColor Gray
Write-Host ""

# Check if Bitcoin Sprint is running
$sprintRunning = Get-NetTCPConnection -LocalPort 8080 -State Listen -ErrorAction SilentlyContinue
if (-not $sprintRunning) {
    Write-Host "❌ Bitcoin Sprint is not running on port 8080" -ForegroundColor Red
    Write-Host "Please start Bitcoin Sprint first to test heavy load." -ForegroundColor Yellow
    exit 1
}

Write-Host "✅ Bitcoin Sprint detected on port 8080" -ForegroundColor Green
Write-Host ""

# Pre-test system baseline
Write-Host "📈 System Baseline (Pre-Test):" -ForegroundColor Yellow
$preTestTime = Get-Date
try {
    $baselineStart = [System.Diagnostics.Stopwatch]::StartNew()
    $baseline = Invoke-RestMethod -Uri "http://localhost:8080$TargetEndpoint" -TimeoutSec 5
    $baselineStart.Stop()
    Write-Host "   Baseline Response: $($baselineStart.ElapsedMilliseconds)ms" -ForegroundColor Green
} catch {
    Write-Host "   Baseline Test Failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Initialize results tracking
$global:completedClients = 0
$global:totalRequests = 0
$global:totalErrors = 0
$global:allResponseTimes = [System.Collections.Concurrent.ConcurrentBag[double]]::new()
$global:clientResults = [System.Collections.Concurrent.ConcurrentBag[PSCustomObject]]::new()

# Real-time monitoring function
function Start-RealTimeMonitoring {
    if (-not $RealTimeMonitoring) { return }
    
    $monitorJob = Start-Job -ScriptBlock {
        param($duration, $endpoint)
        $endTime = (Get-Date).AddMinutes($duration)
        
        while ((Get-Date) -lt $endTime) {
            try {
                $start = [System.Diagnostics.Stopwatch]::StartNew()
                Invoke-RestMethod -Uri "http://localhost:8080$endpoint" -TimeoutSec 2 | Out-Null
                $start.Stop()
                
                $timestamp = Get-Date -Format "HH:mm:ss"
                Write-Host "[$timestamp] Monitor: $($start.ElapsedMilliseconds)ms" -ForegroundColor Cyan
            } catch {
                $timestamp = Get-Date -Format "HH:mm:ss"
                Write-Host "[$timestamp] Monitor: ERROR" -ForegroundColor Red
            }
            Start-Sleep -Seconds 5
        }
    } -ArgumentList $TestDurationMinutes, $TargetEndpoint
    
    return $monitorJob
}

# Client simulation function
$clientScript = {
    param($clientId, $requestCount, $endpoint, $durationMinutes)
    
    $results = @{
        ClientId = $clientId
        RequestsSent = 0
        SuccessfulRequests = 0
        FailedRequests = 0
        ResponseTimes = @()
        StartTime = Get-Date
        EndTime = $null
        AverageResponse = 0
        MinResponse = 0
        MaxResponse = 0
        P95Response = 0
    }
    
    $endTime = (Get-Date).AddMinutes($durationMinutes)
    
    try {
        for ($i = 1; $i -le $requestCount -and (Get-Date) -lt $endTime; $i++) {
            $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
            
            try {
                $response = Invoke-RestMethod -Uri "http://localhost:8080$endpoint" -TimeoutSec 10
                $stopwatch.Stop()
                
                $results.ResponseTimes += $stopwatch.ElapsedMilliseconds
                $results.SuccessfulRequests++
            } catch {
                $stopwatch.Stop()
                $results.FailedRequests++
            }
            
            $results.RequestsSent++
            
            # Brief pause to simulate realistic client behavior
            Start-Sleep -Milliseconds (Get-Random -Minimum 100 -Maximum 500)
        }
        
        $results.EndTime = Get-Date
        
        # Calculate statistics
        if ($results.ResponseTimes.Count -gt 0) {
            $results.AverageResponse = ($results.ResponseTimes | Measure-Object -Average).Average
            $results.MinResponse = ($results.ResponseTimes | Measure-Object -Minimum).Minimum
            $results.MaxResponse = ($results.ResponseTimes | Measure-Object -Maximum).Maximum
            
            # Calculate P95
            $sorted = $results.ResponseTimes | Sort-Object
            $p95Index = [math]::Floor($sorted.Count * 0.95)
            $results.P95Response = $sorted[$p95Index]
        }
        
    } catch {
        Write-Error "Client $clientId error: $($_.Exception.Message)"
    }
    
    return $results
}

Write-Host "🚀 Starting Heavy Load Test..." -ForegroundColor Green
Write-Host "Launching $ConcurrentClients concurrent clients..." -ForegroundColor Yellow
Write-Host ""

# Start real-time monitoring
$monitorJob = Start-RealTimeMonitoring

# Launch concurrent clients
$clientJobs = @()
$testStartTime = Get-Date

for ($i = 1; $i -le $ConcurrentClients; $i++) {
    $job = Start-Job -ScriptBlock $clientScript -ArgumentList $i, $RequestsPerClient, $TargetEndpoint, $TestDurationMinutes
    $clientJobs += $job
    
    Write-Host "Client $i launched..." -ForegroundColor Gray
    
    # Brief stagger to avoid thundering herd
    Start-Sleep -Milliseconds 50
}

Write-Host ""
Write-Host "⏱️ All clients launched. Test duration: $TestDurationMinutes minutes" -ForegroundColor Green
Write-Host "Waiting for completion..." -ForegroundColor Yellow
Write-Host ""

# Wait for all clients to complete
$completedClients = 0
$spinner = @('|', '/', '-', '\')
$spinnerIndex = 0

while ($completedClients -lt $ConcurrentClients) {
    $completedClients = ($clientJobs | Where-Object { $_.State -eq 'Completed' }).Count
    $runningClients = $ConcurrentClients - $completedClients
    
    Write-Host "`r$($spinner[$spinnerIndex]) Clients: $completedClients completed, $runningClients running..." -NoNewline -ForegroundColor Yellow
    $spinnerIndex = ($spinnerIndex + 1) % 4
    
    Start-Sleep -Seconds 1
}

Write-Host "`r✅ All clients completed!                                           " -ForegroundColor Green
Write-Host ""

$testEndTime = Get-Date
$totalTestDuration = ($testEndTime - $testStartTime).TotalSeconds

# Stop monitoring
if ($monitorJob) {
    Stop-Job $monitorJob -ErrorAction SilentlyContinue
    Remove-Job $monitorJob -ErrorAction SilentlyContinue
}

# Collect and analyze results
Write-Host "📊 HEAVY LOAD TEST RESULTS" -ForegroundColor Cyan
Write-Host "==========================" -ForegroundColor Cyan
Write-Host ""

$allClientResults = @()
$allResponseTimes = @()
$totalRequests = 0
$totalSuccessful = 0
$totalFailed = 0

foreach ($job in $clientJobs) {
    $result = Receive-Job $job
    Remove-Job $job
    
    if ($result) {
        $allClientResults += $result
        $allResponseTimes += $result.ResponseTimes
        $totalRequests += $result.RequestsSent
        $totalSuccessful += $result.SuccessfulRequests
        $totalFailed += $result.FailedRequests
    }
}

# Overall statistics
Write-Host "🎯 Overall Performance:" -ForegroundColor White
Write-Host "   Test Duration: $([math]::Round($totalTestDuration, 1)) seconds" -ForegroundColor Gray
Write-Host "   Total Requests: $totalRequests" -ForegroundColor Gray
Write-Host "   Successful: $totalSuccessful ($([math]::Round($totalSuccessful/$totalRequests*100, 1))%)" -ForegroundColor $(if($totalSuccessful -eq $totalRequests){'Green'}else{'Yellow'})
Write-Host "   Failed: $totalFailed ($([math]::Round($totalFailed/$totalRequests*100, 1))%)" -ForegroundColor $(if($totalFailed -eq 0){'Green'}else{'Red'})
Write-Host "   Requests/Second: $([math]::Round($totalRequests/$totalTestDuration, 1))" -ForegroundColor Gray
Write-Host ""

if ($allResponseTimes.Count -gt 0) {
    $avgResponse = ($allResponseTimes | Measure-Object -Average).Average
    $minResponse = ($allResponseTimes | Measure-Object -Minimum).Minimum
    $maxResponse = ($allResponseTimes | Measure-Object -Maximum).Maximum
    
    # Calculate percentiles
    $sorted = $allResponseTimes | Sort-Object
    $p50Index = [math]::Floor($sorted.Count * 0.50)
    $p95Index = [math]::Floor($sorted.Count * 0.95)
    $p99Index = [math]::Floor($sorted.Count * 0.99)
    
    $p50 = $sorted[$p50Index]
    $p95 = $sorted[$p95Index]
    $p99 = $sorted[$p99Index]
    
    Write-Host "⚡ Response Time Analysis:" -ForegroundColor White
    Write-Host "   Average: $([math]::Round($avgResponse, 1))ms" -ForegroundColor $(if($avgResponse -lt 100){'Green'}elseif($avgResponse -lt 200){'Yellow'}else{'Red'})
    Write-Host "   Median (P50): $([math]::Round($p50, 1))ms" -ForegroundColor Gray
    Write-Host "   95th Percentile: $([math]::Round($p95, 1))ms" -ForegroundColor $(if($p95 -lt 200){'Green'}elseif($p95 -lt 500){'Yellow'}else{'Red'})
    Write-Host "   99th Percentile: $([math]::Round($p99, 1))ms" -ForegroundColor $(if($p99 -lt 500){'Green'}elseif($p99 -lt 1000){'Yellow'}else{'Red'})
    Write-Host "   Min/Max: $minResponse - $maxResponse ms" -ForegroundColor Gray
    Write-Host ""
    
    # Load test assessment
    Write-Host "🏋️ Load Test Assessment:" -ForegroundColor White
    
    $successRate = $totalSuccessful / $totalRequests * 100
    $rps = $totalRequests / $totalTestDuration
    
    if ($successRate -ge 99 -and $p95 -lt 200 -and $rps -ge $ConcurrentClients * 2) {
        Write-Host "   ✅ EXCELLENT - System handles heavy load very well" -ForegroundColor Green
        Write-Host "   ✅ High throughput with low latency maintained" -ForegroundColor Green
        $loadGrade = "A"
    } elseif ($successRate -ge 95 -and $p95 -lt 500 -and $rps -ge $ConcurrentClients) {
        Write-Host "   👍 GOOD - System performs well under load" -ForegroundColor Yellow
        Write-Host "   👍 Acceptable performance degradation" -ForegroundColor Yellow
        $loadGrade = "B"
    } elseif ($successRate -ge 90 -and $p95 -lt 1000) {
        Write-Host "   ⚠️ FAIR - System shows strain under load" -ForegroundColor Orange
        Write-Host "   ⚠️ Consider optimization for production use" -ForegroundColor Orange
        $loadGrade = "C"
    } else {
        Write-Host "   ❌ POOR - System struggles with heavy load" -ForegroundColor Red
        Write-Host "   ❌ Optimization required before production" -ForegroundColor Red
        $loadGrade = "D"
    }
    
    Write-Host ""
    
    # Client performance distribution
    Write-Host "👥 Client Performance Distribution:" -ForegroundColor White
    Write-Host "Client | Requests | Success% | Avg(ms) | P95(ms) | Status"
    Write-Host "-------|----------|----------|---------|---------|--------"
    
    foreach ($client in $allClientResults | Sort-Object ClientId) {
        $successPercent = if ($client.RequestsSent -gt 0) { [math]::Round($client.SuccessfulRequests/$client.RequestsSent*100, 1) } else { 0 }
        $status = if ($successPercent -ge 95 -and $client.P95Response -lt 300) { "✅ Good" } 
                  elseif ($successPercent -ge 90) { "⚠️ Fair" } 
                  else { "❌ Poor" }
        
        $statusColor = if ($successPercent -ge 95) { "Green" } elseif ($successPercent -ge 90) { "Yellow" } else { "Red" }
        
        Write-Host "$($client.ClientId.ToString().PadLeft(6)) | $($client.RequestsSent.ToString().PadLeft(8)) | $($successPercent.ToString().PadLeft(7))% | $([math]::Round($client.AverageResponse, 1).ToString().PadLeft(7)) | $([math]::Round($client.P95Response, 1).ToString().PadLeft(7)) | " -NoNewline
        Write-Host $status -ForegroundColor $statusColor
    }
    
    Write-Host ""
}

# Save results if requested
if ($SaveResults) {
    Write-Host "💾 Saving Heavy Load Test Results..." -ForegroundColor Cyan
    
    $timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $reportFile = "heavy-load-test-$timestamp.json"
    
    $heavyLoadReport = @{
        test_info = @{
            timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
            test_type = "Heavy Load Test"
            concurrent_clients = $ConcurrentClients
            requests_per_client = $RequestsPerClient
            test_duration_minutes = $TestDurationMinutes
            target_endpoint = $TargetEndpoint
            actual_duration_seconds = [math]::Round($totalTestDuration, 1)
        }
        overall_metrics = @{
            total_requests = $totalRequests
            successful_requests = $totalSuccessful
            failed_requests = $totalFailed
            success_rate_percent = [math]::Round($successRate, 2)
            requests_per_second = [math]::Round($rps, 2)
            average_response_time_ms = [math]::Round($avgResponse, 2)
            median_response_time_ms = [math]::Round($p50, 2)
            p95_response_time_ms = [math]::Round($p95, 2)
            p99_response_time_ms = [math]::Round($p99, 2)
            min_response_time_ms = $minResponse
            max_response_time_ms = $maxResponse
        }
        load_assessment = @{
            grade = $loadGrade
            system_ready_for_production = ($loadGrade -in @("A", "B"))
            recommendations = switch ($loadGrade) {
                "A" { @("System performs excellently under load", "Ready for production deployment") }
                "B" { @("Good performance under load", "Monitor for potential optimization opportunities") }
                "C" { @("Performance degradation under load", "Consider optimization before production", "Review resource allocation") }
                "D" { @("Poor performance under load", "Significant optimization required", "Not ready for production") }
            }
        }
        client_results = $allClientResults
    }
    
    try {
        $heavyLoadReport | ConvertTo-Json -Depth 6 | Out-File $reportFile -Encoding UTF8
        Write-Host "   ✅ Report saved: $reportFile" -ForegroundColor Green
        Write-Host "   📊 Contains detailed heavy load analysis" -ForegroundColor Gray
    } catch {
        Write-Host "   ⚠️ Failed to save report: $($_.Exception.Message)" -ForegroundColor Yellow
    }
}

Write-Host ""
Write-Host "📄 Heavy Load Test completed at $(Get-Date -Format 'HH:mm:ss')" -ForegroundColor Gray
Write-Host "🚀 For long-running tests, use: .\soak-test.ps1" -ForegroundColor Gray
Write-Host "📊 For tier comparison, use: .\tier-comparison-test.ps1" -ForegroundColor Gray
