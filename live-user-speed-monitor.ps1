#!/usr/bin/env pwsh
#
# Live Real User Speed Monitor
# Shows what real users are experiencing right now
#

param(
    [int]$Duration = 60,  # How long to monitor
    [int]$UpdateInterval = 5  # Update every N seconds
)

Write-Host "🔍 Live Real User Speed Monitor" -ForegroundColor Cyan
Write-Host "================================" -ForegroundColor Cyan
Write-Host "Monitoring for $Duration seconds, updating every $UpdateInterval seconds" -ForegroundColor Gray
Write-Host "Press Ctrl+C to stop early" -ForegroundColor Gray
Write-Host ""

$endTime = (Get-Date).AddSeconds($Duration)
$iteration = 0

# Initialize arrays to track performance over time
$performanceHistory = @()

while ((Get-Date) -lt $endTime) {
    $iteration++
    Clear-Host
    
    Write-Host "🔍 Live Real User Speed Monitor - Update #$iteration" -ForegroundColor Cyan
    Write-Host "=================================================" -ForegroundColor Cyan
    Write-Host "Current Time: $(Get-Date -Format 'HH:mm:ss')" -ForegroundColor Gray
    Write-Host ""
    
    # Test current performance with realistic user patterns
    $currentPerf = @()
    
    # Simulate different user types
    $userTypes = @(
        @{ name = "Dashboard User"; endpoint = "/status"; description = "Checking service status" },
        @{ name = "Price Checker"; endpoint = "/latest"; description = "Getting latest price" },
        @{ name = "Quick Browser"; endpoint = "/status"; description = "Quick status check" }
    )
    
    foreach ($userType in $userTypes) {
        $responses = @()
        
        # Multiple quick requests like a real user
        for ($i = 0; $i -lt 3; $i++) {
            try {
                $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
                $response = Invoke-RestMethod -Uri "http://localhost:8080$($userType.endpoint)" -TimeoutSec 5
                $stopwatch.Stop()
                $responses += $stopwatch.ElapsedMilliseconds
            } catch {
                $responses += 9999  # Error represented as very slow
            }
            Start-Sleep -Milliseconds 200  # Brief pause between requests
        }
        
        if ($responses.Count -gt 0) {
            $avgTime = ($responses | Measure-Object -Average).Average
            $minTime = ($responses | Measure-Object -Minimum).Minimum
            $maxTime = ($responses | Measure-Object -Maximum).Maximum
            
            $currentPerf += [PSCustomObject]@{
                UserType = $userType.name
                Endpoint = $userType.endpoint
                AvgTime = $avgTime
                MinTime = $minTime
                MaxTime = $maxTime
                Description = $userType.description
            }
        }
    }
    
    # Display current results
    Write-Host "🚀 CURRENT USER EXPERIENCE:" -ForegroundColor White
    Write-Host ""
    
    foreach ($perf in $currentPerf) {
        $color = if ($perf.AvgTime -lt 100) { 'Green' }
                elseif ($perf.AvgTime -lt 200) { 'Yellow' }
                else { 'Red' }
        
        $experience = if ($perf.AvgTime -lt 50) { "⚡ Excellent" }
                     elseif ($perf.AvgTime -lt 100) { "✅ Very Good" }
                     elseif ($perf.AvgTime -lt 200) { "👍 Good" }
                     elseif ($perf.AvgTime -lt 500) { "🐌 Slow" }
                     else { "❌ Poor" }
        
        Write-Host "   $($perf.UserType.PadRight(15)): $([math]::Round($perf.AvgTime, 1))ms ($experience)" -ForegroundColor $color
        Write-Host "      $($perf.Description)" -ForegroundColor Gray
        Write-Host "      Range: $($perf.MinTime) - $($perf.MaxTime)ms" -ForegroundColor Gray
        Write-Host ""
    }
    
    # Calculate overall performance for this iteration
    $overallAvg = ($currentPerf | Measure-Object -Property AvgTime -Average).Average
    $performanceHistory += $overallAvg
    
    # Show trend
    Write-Host "📊 PERFORMANCE TREND:" -ForegroundColor White
    Write-Host "   Current Average: $([math]::Round($overallAvg, 1))ms" -ForegroundColor $(if($overallAvg -lt 100){'Green'}else{'Yellow'})
    
    if ($performanceHistory.Count -gt 1) {
        $trend = $performanceHistory[-1] - $performanceHistory[-2]
        $trendDirection = if ($trend -lt -5) { "📈 Improving" }
                         elseif ($trend -gt 5) { "📉 Degrading" }
                         else { "➡️ Stable" }
        
        Write-Host "   Trend: $trendDirection ($([math]::Round($trend, 1))ms change)" -ForegroundColor Gray
        
        # Show mini-chart of last few measurements
        if ($performanceHistory.Count -gt 5) {
            $recent = $performanceHistory | Select-Object -Last 6
            $chartLine = ""
            foreach ($point in $recent) {
                $chartLine += if ($point -lt 50) { "▁" }
                             elseif ($point -lt 100) { "▂" }
                             elseif ($point -lt 150) { "▃" }
                             elseif ($point -lt 200) { "▅" }
                             elseif ($point -lt 300) { "▆" }
                             else { "▇" }
            }
            Write-Host "   Recent:  $chartLine (last 6 measurements)" -ForegroundColor Gray
        }
    }
    
    # Show what tier this performance suggests
    Write-Host ""
    Write-Host "🎯 TIER PERFORMANCE MATCH:" -ForegroundColor White
    if ($overallAvg -lt 50) {
        Write-Host "   This matches ENTERPRISE tier performance (< 50ms average)" -ForegroundColor Green
    } elseif ($overallAvg -lt 100) {
        Write-Host "   This matches PRO tier performance (50-100ms average)" -ForegroundColor Yellow
    } elseif ($overallAvg -lt 200) {
        Write-Host "   This matches FREE tier performance (100-200ms average)" -ForegroundColor Yellow
    } else {
        Write-Host "   This is below expected tier performance (> 200ms average)" -ForegroundColor Red
    }
    
    # Next update countdown
    Write-Host ""
    Write-Host "Next update in $UpdateInterval seconds..." -ForegroundColor Gray
    Write-Host "Time remaining: $([math]::Round(($endTime - (Get-Date)).TotalSeconds, 0)) seconds" -ForegroundColor Gray
    
    # Wait for next iteration
    Start-Sleep $UpdateInterval
}

# Final summary
Clear-Host
Write-Host "📋 FINAL MONITORING SUMMARY" -ForegroundColor Cyan
Write-Host "============================" -ForegroundColor Cyan
Write-Host ""

if ($performanceHistory.Count -gt 0) {
    $finalAvg = ($performanceHistory | Measure-Object -Average).Average
    $finalMin = ($performanceHistory | Measure-Object -Minimum).Minimum
    $finalMax = ($performanceHistory | Measure-Object -Maximum).Maximum
    
    Write-Host "Overall Session Performance:" -ForegroundColor White
    Write-Host "   Average Response Time: $([math]::Round($finalAvg, 1))ms" -ForegroundColor $(if($finalAvg -lt 100){'Green'}else{'Yellow'})
    Write-Host "   Best Performance:      $([math]::Round($finalMin, 1))ms" -ForegroundColor Green
    Write-Host "   Worst Performance:     $([math]::Round($finalMax, 1))ms" -ForegroundColor Red
    Write-Host "   Total Measurements:    $($performanceHistory.Count)" -ForegroundColor Gray
    
    # User experience assessment
    Write-Host ""
    if ($finalAvg -lt 50) {
        Write-Host "🎉 User Experience: EXCELLENT - Users will love this speed!" -ForegroundColor Green
    } elseif ($finalAvg -lt 100) {
        Write-Host "✅ User Experience: VERY GOOD - Fast and responsive for users" -ForegroundColor Green
    } elseif ($finalAvg -lt 200) {
        Write-Host "👍 User Experience: GOOD - Acceptable speed for most users" -ForegroundColor Yellow
    } else {
        Write-Host "⚠️ User Experience: NEEDS IMPROVEMENT - Users may notice delays" -ForegroundColor Red
    }
} else {
    Write-Host "No performance data collected during monitoring period." -ForegroundColor Red
}

Write-Host ""
Write-Host "Monitoring completed at $(Get-Date -Format 'HH:mm:ss')" -ForegroundColor Gray
