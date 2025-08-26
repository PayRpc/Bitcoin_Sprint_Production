#!/usr/bin/env pwsh
#
# Real User Speed Summary Report
# Quick assessment of actual user experience
#

Write-Host "📊 REAL USER SPEED SUMMARY REPORT" -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan
Write-Host "Generated: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Gray
Write-Host ""

# Check if Bitcoin Sprint is running
$sprintRunning = Get-NetTCPConnection -LocalPort 8080 -State Listen -ErrorAction SilentlyContinue
if (-not $sprintRunning) {
    Write-Host "❌ Bitcoin Sprint is not running on port 8080" -ForegroundColor Red
    Write-Host "Please start Bitcoin Sprint first to test real user speeds." -ForegroundColor Yellow
    exit 1
}

Write-Host "✅ Bitcoin Sprint detected on port 8080" -ForegroundColor Green
Write-Host ""

# Test different user scenarios with timing
Write-Host "🧪 Testing Real User Scenarios..." -ForegroundColor White
Write-Host ""

$scenarios = @(
    @{ 
        name = "Website Dashboard User"
        description = "User checking status on a website dashboard"
        endpoint = "/status"
        typical_frequency = "Every 30 seconds"
        expectation = "< 200ms for good UX"
    },
    @{ 
        name = "Mobile App User"
        description = "Mobile app getting latest data"
        endpoint = "/latest"
        typical_frequency = "Every 2 minutes"
        expectation = "< 500ms acceptable on mobile"
    },
    @{ 
        name = "Trading Bot"
        description = "Automated system getting rapid updates"
        endpoint = "/latest"
        typical_frequency = "Every 2-5 seconds"
        expectation = "< 100ms for competitive advantage"
    },
    @{ 
        name = "Price Widget"
        description = "Embedded widget showing current price"
        endpoint = "/status"
        typical_frequency = "Every 15 seconds"
        expectation = "< 300ms to not feel sluggish"
    }
)

$allResults = @()

foreach ($scenario in $scenarios) {
    Write-Host "Testing: $($scenario.name)" -ForegroundColor Yellow
    Write-Host "  Use case: $($scenario.description)" -ForegroundColor Gray
    Write-Host "  Frequency: $($scenario.typical_frequency)" -ForegroundColor Gray
    Write-Host "  Expectation: $($scenario.expectation)" -ForegroundColor Gray
    
    # Test this scenario multiple times like a real user would
    $responses = @()
    $errors = 0
    
    for ($test = 1; $test -le 5; $test++) {
        try {
            $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
            $response = Invoke-RestMethod -Uri "http://localhost:8080$($scenario.endpoint)" -TimeoutSec 10
            $stopwatch.Stop()
            $responses += $stopwatch.ElapsedMilliseconds
        } catch {
            $errors++
        }
        
        # Brief pause like a real user
        Start-Sleep -Milliseconds 500
    }
    
    if ($responses.Count -gt 0) {
        $avgTime = ($responses | Measure-Object -Average).Average
        $minTime = ($responses | Measure-Object -Minimum).Minimum
        $maxTime = ($responses | Measure-Object -Maximum).Maximum
        
        # Calculate p95 latency (gold standard for user experience)
        $sorted = $responses | Sort-Object
        $p95Index = [math]::Floor($sorted.Count * 0.95)
        $p95 = $sorted[$p95Index]
        
        # Determine user experience level
        $userExperience = switch ($avgTime) {
            { $_ -lt 50 } { "⚡ Excellent - Lightning fast" }
            { $_ -lt 100 } { "✅ Very Good - Fast and responsive" }
            { $_ -lt 200 } { "👍 Good - Acceptably fast" }
            { $_ -lt 500 } { "🐌 Slow - Noticeable delays" }
            default { "❌ Poor - Frustrating delays" }
        }
        
        $responseColor = if ($avgTime -lt 100) { 'Green' }
                        elseif ($avgTime -lt 200) { 'Yellow' }
                        else { 'Red' }
        
        Write-Host "  Results:" -ForegroundColor White
        Write-Host "    Average Response: $([math]::Round($avgTime, 1))ms" -ForegroundColor $responseColor
        Write-Host "    95th Percentile: $([math]::Round($p95, 1))ms" -ForegroundColor Gray
        Write-Host "    Range: $($minTime) - $($maxTime)ms" -ForegroundColor Gray
        Write-Host "    User Experience: $userExperience" -ForegroundColor $responseColor
        Write-Host "    Success Rate: $(((5-$errors)/5*100))%" -ForegroundColor $(if($errors -eq 0){'Green'}else{'Yellow'})
        if ($errors -gt 0) {
            Write-Host "    Errors: $errors ($([math]::Round($errors/5*100, 1))% of requests)" -ForegroundColor Red
        }
        
        $allResults += [PSCustomObject]@{
            Scenario = $scenario.name
            Description = $scenario.description
            AvgResponseTime = $avgTime
            MinResponseTime = $minTime
            MaxResponseTime = $maxTime
            P95ResponseTime = $p95
            SuccessRate = ((5-$errors)/5*100)
            ErrorCount = $errors
            UserExperience = $userExperience
            MeetsExpectation = switch ($scenario.name) {
                "Website Dashboard User" { $avgTime -lt 200 }
                "Mobile App User" { $avgTime -lt 500 }
                "Trading Bot" { $avgTime -lt 100 }
                "Price Widget" { $avgTime -lt 300 }
            }
        }
    } else {
        Write-Host "  Results: ❌ All requests failed" -ForegroundColor Red
    }
    
    Write-Host ""
}

# Overall assessment
Write-Host "🎯 OVERALL USER EXPERIENCE ASSESSMENT" -ForegroundColor Cyan
Write-Host "=====================================" -ForegroundColor Cyan
Write-Host ""

if ($allResults.Count -gt 0) {
    $overallAvg = ($allResults | Measure-Object -Property AvgResponseTime -Average).Average
    $successfulScenarios = ($allResults | Where-Object { $_.MeetsExpectation -eq $true }).Count
    $totalScenarios = $allResults.Count
    
    Write-Host "📈 Performance Summary:" -ForegroundColor White
    Write-Host "   Overall Average Response: $([math]::Round($overallAvg, 1))ms" -ForegroundColor $(if($overallAvg -lt 150){'Green'}else{'Yellow'})
    Write-Host "   Scenarios Meeting Expectations: $successfulScenarios/$totalScenarios ($([math]::Round($successfulScenarios/$totalScenarios*100, 1))%)" -ForegroundColor $(if($successfulScenarios -eq $totalScenarios){'Green'}else{'Yellow'})
    
    # Tier recommendation based on performance
    Write-Host ""
    Write-Host "🏆 Performance Tier Assessment:" -ForegroundColor White
    if ($overallAvg -lt 50) {
        Write-Host "   This performance matches ENTERPRISE tier (< 50ms)" -ForegroundColor Green
        Write-Host "   Users experience lightning-fast responses" -ForegroundColor Green
    } elseif ($overallAvg -lt 100) {
        Write-Host "   This performance matches PRO tier (50-100ms)" -ForegroundColor Green
        Write-Host "   Users experience very good responsiveness" -ForegroundColor Green
    } elseif ($overallAvg -lt 200) {
        Write-Host "   This performance matches FREE tier (100-200ms)" -ForegroundColor Yellow
        Write-Host "   Users experience acceptable responsiveness" -ForegroundColor Yellow
    } else {
        Write-Host "   This performance is below FREE tier expectations (> 200ms)" -ForegroundColor Red
        Write-Host "   Users may experience frustrating delays" -ForegroundColor Red
    }
    
    # Business impact assessment
    Write-Host ""
    Write-Host "💼 Business Impact:" -ForegroundColor White
    if ($overallAvg -lt 100) {
        Write-Host "   ✅ Excellent for user retention and satisfaction" -ForegroundColor Green
        Write-Host "   ✅ Suitable for high-frequency trading applications" -ForegroundColor Green
        Write-Host "   ✅ Can support premium pricing models" -ForegroundColor Green
    } elseif ($overallAvg -lt 200) {
        Write-Host "   👍 Good for general web applications" -ForegroundColor Yellow
        Write-Host "   👍 Acceptable for casual users" -ForegroundColor Yellow
        Write-Host "   ⚠️ May need optimization for competitive apps" -ForegroundColor Yellow
    } else {
        Write-Host "   ⚠️ May impact user experience negatively" -ForegroundColor Red
        Write-Host "   ⚠️ Not suitable for time-sensitive applications" -ForegroundColor Red
        Write-Host "   ⚠️ Consider performance optimization" -ForegroundColor Red
    }
    
    # Detailed scenario breakdown
    Write-Host ""
    Write-Host "📋 Detailed Scenario Results:" -ForegroundColor White
    Write-Host "Scenario               | Avg Time | P95 Time | Meets Expectation | User Experience"
    Write-Host "-----------------------|----------|----------|-------------------|------------------"
    
    foreach ($result in $allResults) {
        $meetsExp = if ($result.MeetsExpectation) { "✅ Yes" } else { "❌ No" }
        $meetsExpColor = if ($result.MeetsExpectation) { "Green" } else { "Red" }
        $scenarioShort = $result.Scenario.Substring(0, [Math]::Min(20, $result.Scenario.Length)).PadRight(20)
        $avgTimeStr = "$([math]::Round($result.AvgResponseTime, 1))ms".PadLeft(8)
        $p95TimeStr = "$([math]::Round($result.P95ResponseTime, 1))ms".PadLeft(8)
        $expStr = $meetsExp.PadRight(17)
        $uxStr = $result.UserExperience -replace "⚡|✅|👍|🐌|❌ ", ""
        
        Write-Host "$scenarioShort | $avgTimeStr | $p95TimeStr | " -NoNewline
        Write-Host "$expStr" -ForegroundColor $meetsExpColor -NoNewline
        Write-Host " | $uxStr"
    }
    
} else {
    Write-Host "❌ No successful tests completed" -ForegroundColor Red
}

# Save comprehensive JSON report for CI/CD and historical analysis
if ($allResults.Count -gt 0) {
    Write-Host ""
    Write-Host "📄 Saving JSON Report..." -ForegroundColor Cyan
    
    $timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $reportFile = "real-user-speed-summary-$timestamp.json"
    
    $comprehensiveReport = @{
        test_info = @{
            timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
            test_type = "Real User Speed Summary Report"
            scenarios_tested = $allResults.Count
        }
        overall_metrics = @{
            overall_average_response_ms = [math]::Round($overallAvg, 2)
            scenarios_meeting_expectations = $successfulScenarios
            total_scenarios = $totalScenarios
            success_percentage = [math]::Round($successfulScenarios/$totalScenarios*100, 1)
        }
        tier_assessment = @{
            performance_tier = if ($overallAvg -lt 50) { "ENTERPRISE" }
                              elseif ($overallAvg -lt 100) { "PRO" }
                              elseif ($overallAvg -lt 200) { "FREE" }
                              else { "BELOW_FREE" }
            business_ready = $overallAvg -lt 200
        }
        scenario_results = $allResults
    }
    
    try {
        $comprehensiveReport | ConvertTo-Json -Depth 5 | Out-File $reportFile -Encoding UTF8
        Write-Host "   ✅ Report saved: $reportFile" -ForegroundColor Green
        Write-Host "   📊 Contains detailed metrics for CI/CD integration" -ForegroundColor Gray
    } catch {
        Write-Host "   ⚠️ Failed to save JSON report: $($_.Exception.Message)" -ForegroundColor Yellow
    }
}

Write-Host ""
Write-Host "📄 Report completed at $(Get-Date -Format 'HH:mm:ss')" -ForegroundColor Gray
Write-Host "To get continuous monitoring, run: .\live-user-speed-monitor.ps1" -ForegroundColor Gray
