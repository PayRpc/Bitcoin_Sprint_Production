#!/usr/bin/env pwsh
#
# Quick Real User Speed Test
# Simple test to see what actual speeds users get from our API
#

param(
    [int]$Requests = 20,
    [switch]$ShowDetails
)

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "QUICK REAL USER SPEED TEST" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan

# Ensure services are running
Write-Host "1. Checking if services are running..." -ForegroundColor Yellow

$mockRunning = Get-NetTCPConnection -LocalPort 8332 -State Listen -ErrorAction SilentlyContinue
$sprintRunning = Get-NetTCPConnection -LocalPort 8080 -State Listen -ErrorAction SilentlyContinue

if (-not $mockRunning) {
    Write-Host "   Starting Bitcoin Core mock..." -ForegroundColor Gray
    Start-Job -ScriptBlock { 
        Set-Location $args[0]
        python scripts\bitcoin-core-mock.py 
    } -ArgumentList (Get-Location) | Out-Null
    Start-Sleep 3
}

if (-not $sprintRunning) {
    Write-Host "   Starting Bitcoin Sprint..." -ForegroundColor Gray
    $env:RPC_NODES = "http://127.0.0.1:8332"
    $env:RPC_USER = "bitcoin"
    $env:RPC_PASS = "sprint123benchmark"
    $env:API_PORT = "8080"
    
    Start-Job -ScriptBlock {
        Set-Location $args[0]
        .\bitcoin-sprint.exe
    } -ArgumentList (Get-Location) | Out-Null
    
    # Wait for startup
    for ($i = 0; $i -lt 15; $i++) {
        Start-Sleep 1
        if (Get-NetTCPConnection -LocalPort 8080 -State Listen -ErrorAction SilentlyContinue) {
            break
        }
    }
}

Write-Host "   ✓ Services ready" -ForegroundColor Green

# Test different endpoints like a real user would
$endpoints = @(
    @{ name = "/status"; description = "Service status check" },
    @{ name = "/latest"; description = "Latest block data" }
)

$allResults = @()

Write-Host ""
Write-Host "2. Testing real user API speeds ($Requests requests per endpoint)..." -ForegroundColor Yellow

foreach ($endpoint in $endpoints) {
    Write-Host ""
    Write-Host "   Testing $($endpoint.name) - $($endpoint.description)" -ForegroundColor White
    
    $responseTimes = @()
    $errors = 0
    
    for ($i = 1; $i -le $Requests; $i++) {
        try {
            $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
            $response = Invoke-RestMethod -Uri "http://localhost:8080$($endpoint.name)" -TimeoutSec 10
            $stopwatch.Stop()
            
            $responseTime = $stopwatch.ElapsedMilliseconds
            $responseTimes += $responseTime
            
            if ($ShowDetails) {
                Write-Host "     Request $i`: $($responseTime)ms" -ForegroundColor Gray
            }
            
        } catch {
            $errors++
            if ($ShowDetails) {
                Write-Host "     Request $i`: ERROR - $($_.Exception.Message)" -ForegroundColor Red
            }
        }
        
        # Small pause between requests (like a real user)
        Start-Sleep -Milliseconds 200
    }
    
    if ($responseTimes.Count -gt 0) {
        $avgTime = ($responseTimes | Measure-Object -Average).Average
        $minTime = ($responseTimes | Measure-Object -Minimum).Minimum
        $maxTime = ($responseTimes | Measure-Object -Maximum).Maximum
        $medianTime = ($responseTimes | Sort-Object)[[math]::Floor($responseTimes.Count / 2)]
        
        Write-Host "     Average: $([math]::Round($avgTime, 1))ms" -ForegroundColor $(if($avgTime -lt 100){'Green'}elseif($avgTime -lt 300){'Yellow'}else{'Red'})
        Write-Host "     Median:  $([math]::Round($medianTime, 1))ms" -ForegroundColor Gray
        Write-Host "     Range:   $($minTime)ms - $($maxTime)ms" -ForegroundColor Gray
        Write-Host "     Errors:  $errors/$Requests" -ForegroundColor $(if($errors -eq 0){'Green'}else{'Red'})
        
        $allResults += [PSCustomObject]@{
            Endpoint = $endpoint.name
            Description = $endpoint.description
            AvgResponseTime = $avgTime
            MedianResponseTime = $medianTime
            MinResponseTime = $minTime
            MaxResponseTime = $maxTime
            ErrorCount = $errors
            TotalRequests = $Requests
            SuccessRate = (($Requests - $errors) / $Requests) * 100
        }
        
    } else {
        Write-Host "     ❌ All requests failed!" -ForegroundColor Red
    }
}

# Overall assessment
Write-Host ""
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "REAL USER SPEED RESULTS" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan

$overallAvg = ($allResults | Measure-Object -Property AvgResponseTime -Average).Average
$overallErrors = ($allResults | Measure-Object -Property ErrorCount -Sum).Sum
$totalRequests = ($allResults | Measure-Object -Property TotalRequests -Sum).Sum

Write-Host ""
Write-Host "📊 Summary for Real Users:" -ForegroundColor White
Write-Host "   Overall Average Response: $([math]::Round($overallAvg, 1))ms" -ForegroundColor $(if($overallAvg -lt 100){'Green'}elseif($overallAvg -lt 300){'Yellow'}else{'Red'})
Write-Host "   Total Requests: $totalRequests" -ForegroundColor Gray
Write-Host "   Total Errors: $overallErrors" -ForegroundColor $(if($overallErrors -eq 0){'Green'}else{'Red'})
Write-Host "   Success Rate: $([math]::Round((($totalRequests - $overallErrors) / $totalRequests) * 100, 1))%" -ForegroundColor $(if($overallErrors -eq 0){'Green'}else{'Yellow'})

Write-Host ""
Write-Host "🎯 User Experience Rating:" -ForegroundColor White
if ($overallAvg -lt 50) {
    Write-Host "   ⚡ EXCELLENT - Users will experience very fast responses" -ForegroundColor Green
} elseif ($overallAvg -lt 100) {
    Write-Host "   ✅ GOOD - Users will experience fast responses" -ForegroundColor Green
} elseif ($overallAvg -lt 200) {
    Write-Host "   ⚠️  ACCEPTABLE - Users will experience reasonable responses" -ForegroundColor Yellow
} elseif ($overallAvg -lt 500) {
    Write-Host "   🐌 SLOW - Users may notice delays" -ForegroundColor Yellow
} else {
    Write-Host "   ❌ POOR - Users will experience frustrating delays" -ForegroundColor Red
}

Write-Host ""
Write-Host "💡 What this means for real users:" -ForegroundColor White

foreach ($result in $allResults) {
    $rating = if ($result.AvgResponseTime -lt 50) { "very responsive" }
              elseif ($result.AvgResponseTime -lt 100) { "responsive" }
              elseif ($result.AvgResponseTime -lt 200) { "acceptable" }
              elseif ($result.AvgResponseTime -lt 500) { "noticeable delay" }
              else { "frustrating delay" }
              
    Write-Host "   $($result.Endpoint): $([math]::Round($result.AvgResponseTime, 1))ms - $rating" -ForegroundColor Gray
}

Write-Host ""
Write-Host "=========================================" -ForegroundColor Cyan

# Save results to file
$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$resultsFile = "quick-user-speed-test-$timestamp.json"
$allResults | ConvertTo-Json -Depth 3 | Out-File $resultsFile -Encoding UTF8
Write-Host "📄 Results saved to: $resultsFile" -ForegroundColor Gray
