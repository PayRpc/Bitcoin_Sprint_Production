#!/usr/bin/env pwsh
#
# Comprehensive Real User Speed Benchmark
# Tests what actual speeds users get across all tiers
#

param(
    [switch]$QuickTest,  # 30 second test vs 3 minute test
    [switch]$ShowRawData
)

$testDuration = if ($QuickTest) { 30 } else { 180 }
$requestsPerTest = if ($QuickTest) { 10 } else { 30 }

Write-Host "=================================================================" -ForegroundColor Cyan
Write-Host "COMPREHENSIVE REAL USER SPEED BENCHMARK" -ForegroundColor Cyan
Write-Host "=================================================================" -ForegroundColor Cyan
Write-Host "Test Mode: $(if($QuickTest){'Quick (30s)'}else{'Full (3min)'})" -ForegroundColor White
Write-Host "Test Time: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Gray
Write-Host ""

# Ensure Bitcoin Core mock is running
function Start-RequiredServices {
    Write-Host "🔧 Preparing test environment..." -ForegroundColor Yellow
    
    $mockRunning = Get-NetTCPConnection -LocalPort 8332 -State Listen -ErrorAction SilentlyContinue
    if (-not $mockRunning) {
        Write-Host "   Starting Bitcoin Core mock..." -ForegroundColor Gray
        Start-Job -ScriptBlock { 
            Set-Location $args[0]
            python scripts\bitcoin-core-mock.py 
        } -ArgumentList (Get-Location) | Out-Null
        Start-Sleep 3
    }
    Write-Host "   ✓ Bitcoin Core mock ready" -ForegroundColor Green
}

function Test-TierPerformance {
    param($TierName, $ConfigFile, $BinaryFile)
    
    Write-Host ""
    Write-Host "🚀 TESTING $TierName TIER PERFORMANCE" -ForegroundColor Cyan
    Write-Host "Configuration: $ConfigFile" -ForegroundColor Gray
    Write-Host "Binary: $BinaryFile" -ForegroundColor Gray
    Write-Host ""
    
    # Stop any running Sprint
    Get-Process -Name "*sprint*" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
    Start-Sleep 2
    
    # Apply tier configuration
    if (Test-Path $ConfigFile) {
        Copy-Item $ConfigFile "config.json" -Force
        Write-Host "   ✓ Applied $TierName configuration" -ForegroundColor Green
    } else {
        Write-Host "   ⚠️ Config file not found: $ConfigFile" -ForegroundColor Yellow
        return $null
    }
    
    # Start Sprint for this tier
    $env:RPC_NODES = "http://127.0.0.1:8332"
    $env:RPC_USER = "bitcoin"
    $env:RPC_PASS = "sprint123benchmark"
    $env:API_PORT = "8080"
    
    $binary = if (Test-Path $BinaryFile) { $BinaryFile } else { "bitcoin-sprint.exe" }
    
    Write-Host "   Starting Sprint with $binary..." -ForegroundColor Gray
    $sprintJob = Start-Job -ScriptBlock {
        Set-Location $args[0]
        & $args[1]
    } -ArgumentList (Get-Location), $binary
    
    # Wait for Sprint to be ready
    $ready = $false
    for ($i = 0; $i -lt 20; $i++) {
        Start-Sleep 1
        if (Get-NetTCPConnection -LocalPort 8080 -State Listen -ErrorAction SilentlyContinue) {
            $ready = $true
            break
        }
    }
    
    if (-not $ready) {
        Write-Host "   ❌ Sprint failed to start" -ForegroundColor Red
        return $null
    }
    
    Write-Host "   ✓ Sprint ready for testing" -ForegroundColor Green
    
    # Warmup requests
    Write-Host "   Warming up..." -ForegroundColor Gray
    for ($i = 0; $i -lt 5; $i++) {
        try {
            $null = Invoke-RestMethod -Uri "http://localhost:8080/status" -TimeoutSec 5
            $null = Invoke-RestMethod -Uri "http://localhost:8080/latest" -TimeoutSec 5
        } catch { }
        Start-Sleep -Milliseconds 100
    }
    
    # Real user testing scenarios
    $scenarios = @(
        @{ name = "Dashboard User"; endpoint = "/status"; interval = 5; description = "Checking service status" },
        @{ name = "Data Consumer"; endpoint = "/latest"; interval = 3; description = "Getting latest block data" },
        @{ name = "Mixed Usage"; endpoints = @("/status", "/latest"); interval = 4; description = "Typical API usage" }
    )
    
    $tierResults = @()
    
    foreach ($scenario in $scenarios) {
        Write-Host ""
        Write-Host "   📊 Testing: $($scenario.description)" -ForegroundColor White
        
        $responses = @()
        $errors = 0
        
        for ($request = 1; $request -le $requestsPerTest; $request++) {
            if ($scenario.endpoints) {
                # Mixed endpoint testing
                $endpoint = $scenario.endpoints[(Get-Random -Maximum $scenario.endpoints.Count)]
            } else {
                $endpoint = $scenario.endpoint
            }
            
            try {
                $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
                $response = Invoke-RestMethod -Uri "http://localhost:8080$endpoint" -TimeoutSec 10
                $stopwatch.Stop()
                
                $responseTime = $stopwatch.ElapsedMilliseconds
                $responses += $responseTime
                
                if ($ShowRawData) {
                    Write-Host "      $endpoint`: $($responseTime)ms" -ForegroundColor Gray
                }
                
            } catch {
                $errors++
                if ($ShowRawData) {
                    Write-Host "      $endpoint`: ERROR" -ForegroundColor Red
                }
            }
            
            # Realistic user pause
            Start-Sleep -Milliseconds $scenario.interval * 100
        }
        
        if ($responses.Count -gt 0) {
            $avgTime = ($responses | Measure-Object -Average).Average
            $minTime = ($responses | Measure-Object -Minimum).Minimum
            $maxTime = ($responses | Measure-Object -Maximum).Maximum
            $medianTime = ($responses | Sort-Object)[[math]::Floor($responses.Count / 2)]
            
            # Calculate percentiles
            $sortedResponses = $responses | Sort-Object
            $p95Index = [math]::Floor($sortedResponses.Count * 0.95)
            $p95Time = $sortedResponses[$p95Index]
            
            $successRate = (($requestsPerTest - $errors) / $requestsPerTest) * 100
            
            Write-Host "      Average: $([math]::Round($avgTime, 1))ms" -ForegroundColor $(if($avgTime -lt 100){'Green'}elseif($avgTime -lt 200){'Yellow'}else{'Red'})
            Write-Host "      Median:  $([math]::Round($medianTime, 1))ms" -ForegroundColor Gray
            Write-Host "      95th %:  $([math]::Round($p95Time, 1))ms" -ForegroundColor Gray
            Write-Host "      Range:   $($minTime) - $($maxTime)ms" -ForegroundColor Gray
            Write-Host "      Success: $([math]::Round($successRate, 1))%" -ForegroundColor $(if($successRate -eq 100){'Green'}else{'Yellow'})
            
            $tierResults += [PSCustomObject]@{
                Tier = $TierName
                Scenario = $scenario.name
                Description = $scenario.description
                AvgResponseTime = $avgTime
                MedianResponseTime = $medianTime
                P95ResponseTime = $p95Time
                MinResponseTime = $minTime
                MaxResponseTime = $maxTime
                SuccessRate = $successRate
                ErrorCount = $errors
                TotalRequests = $requestsPerTest
            }
            
        } else {
            Write-Host "      ❌ All requests failed" -ForegroundColor Red
        }
    }
    
    # Clean up
    if ($sprintJob) {
        Stop-Job $sprintJob -ErrorAction SilentlyContinue
        Remove-Job $sprintJob -ErrorAction SilentlyContinue
    }
    
    return $tierResults
}

function Show-ComparisonResults {
    param($AllResults)
    
    Write-Host ""
    Write-Host "=================================================================" -ForegroundColor Cyan
    Write-Host "REAL USER SPEED COMPARISON ACROSS TIERS" -ForegroundColor Cyan
    Write-Host "=================================================================" -ForegroundColor Cyan
    
    # Group by tier
    $tierGroups = $AllResults | Group-Object Tier
    
    foreach ($tierGroup in $tierGroups) {
        $tierName = $tierGroup.Name
        $tierData = $tierGroup.Group
        
        $avgOfAvgs = ($tierData | Measure-Object -Property AvgResponseTime -Average).Average
        $overallP95 = ($tierData | Measure-Object -Property P95ResponseTime -Average).Average
        $overallSuccess = ($tierData | Measure-Object -Property SuccessRate -Average).Average
        
        Write-Host ""
        Write-Host "🎯 $tierName TIER SUMMARY:" -ForegroundColor White
        Write-Host "   Overall Average Response: $([math]::Round($avgOfAvgs, 1))ms" -ForegroundColor $(if($avgOfAvgs -lt 100){'Green'}elseif($avgOfAvgs -lt 200){'Yellow'}else{'Red'})
        Write-Host "   Overall 95th Percentile:  $([math]::Round($overallP95, 1))ms" -ForegroundColor Gray
        Write-Host "   Overall Success Rate:     $([math]::Round($overallSuccess, 1))%" -ForegroundColor $(if($overallSuccess -eq 100){'Green'}else{'Yellow'})
        
        # User experience rating
        if ($avgOfAvgs -lt 50) {
            Write-Host "   User Experience: ⚡ EXCELLENT - Lightning fast" -ForegroundColor Green
        } elseif ($avgOfAvgs -lt 100) {
            Write-Host "   User Experience: ✅ VERY GOOD - Fast and responsive" -ForegroundColor Green
        } elseif ($avgOfAvgs -lt 200) {
            Write-Host "   User Experience: 👍 GOOD - Acceptably fast" -ForegroundColor Yellow
        } elseif ($avgOfAvgs -lt 500) {
            Write-Host "   User Experience: 🐌 SLOW - Noticeable delays" -ForegroundColor Yellow
        } else {
            Write-Host "   User Experience: ❌ POOR - Frustrating delays" -ForegroundColor Red
        }
    }
    
    # Side-by-side comparison
    Write-Host ""
    Write-Host "📊 SIDE-BY-SIDE TIER COMPARISON:" -ForegroundColor White
    Write-Host "Tier       | Avg Response | 95th %ile | User Experience"
    Write-Host "-----------|--------------|-----------|------------------"
    
    foreach ($tierGroup in $tierGroups | Sort-Object Name) {
        $tierName = $tierGroup.Name
        $tierData = $tierGroup.Group
        
        $avgOfAvgs = ($tierData | Measure-Object -Property AvgResponseTime -Average).Average
        $overallP95 = ($tierData | Measure-Object -Property P95ResponseTime -Average).Average
        
        $experience = if ($avgOfAvgs -lt 50) { "Excellent" }
                     elseif ($avgOfAvgs -lt 100) { "Very Good" }
                     elseif ($avgOfAvgs -lt 200) { "Good" }
                     elseif ($avgOfAvgs -lt 500) { "Slow" }
                     else { "Poor" }
        
        $avgStr = "$([math]::Round($avgOfAvgs, 1))ms".PadLeft(10)
        $p95Str = "$([math]::Round($overallP95, 1))ms".PadLeft(8)
        
        Write-Host "$($tierName.PadRight(10)) | $avgStr | $p95Str | $experience"
    }
}

function Save-BenchmarkReport {
    param($AllResults)
    
    $timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $reportFile = "real-user-speed-benchmark-$timestamp.json"
    
    $report = @{
        test_timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
        test_mode = if ($QuickTest) { "quick" } else { "full" }
        test_duration = $testDuration
        requests_per_scenario = $requestsPerTest
        results = $AllResults
        summary = @{
            tiers_tested = ($AllResults | Select-Object -Unique Tier).Tier
            total_scenarios = $AllResults.Count
            overall_avg_response = [math]::Round(($AllResults | Measure-Object -Property AvgResponseTime -Average).Average, 2)
        }
    }
    
    try {
        $report | ConvertTo-Json -Depth 5 | Out-File $reportFile -Encoding UTF8
        Write-Host ""
        Write-Host "📄 Benchmark report saved: $reportFile" -ForegroundColor Green
    } catch {
        Write-Host ""
        Write-Host "⚠️ Failed to save report: $($_.Exception.Message)" -ForegroundColor Yellow
    }
}

# Main execution
try {
    Start-RequiredServices
    
    # Define tier configurations
    $tierConfigs = @(
        @{ name = "FREE"; config = "config-free-stable.json"; binary = "bitcoin-sprint-free.exe" },
        @{ name = "PRO"; config = "config.json"; binary = "bitcoin-sprint.exe" },
        @{ name = "ENTERPRISE"; config = "config-enterprise-turbo.json"; binary = "bitcoin-sprint-turbo.exe" }
    )
    
    $allResults = @()
    
    # Test each tier
    foreach ($tierConfig in $tierConfigs) {
        $results = Test-TierPerformance $tierConfig.name $tierConfig.config $tierConfig.binary
        if ($results) {
            $allResults += $results
        }
    }
    
    # Show comparison results
    Show-ComparisonResults $allResults
    
    # Save detailed report
    Save-BenchmarkReport $allResults
    
    Write-Host ""
    Write-Host "=================================================================" -ForegroundColor Cyan
    Write-Host "✅ Real User Speed Benchmark Complete!" -ForegroundColor Green
    Write-Host "=================================================================" -ForegroundColor Cyan
    
} catch {
    Write-Host "❌ Benchmark failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
} finally {
    # Cleanup
    Get-Process -Name "*sprint*" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
}
