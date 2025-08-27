#!/usr/bin/env pwsh

<#
.SYNOPSIS
    Turbo Mode Load Test for Bitcoin Sprint API
    Stress tests the optimized turbo mode implementation
.DESCRIPTION
    Advanced load testing with ramp-up, sustained load, and cooldown phases
    Validates turbo mode under various load conditions
.PARAMETER Url
    Base URL of the Bitcoin Sprint API (default: http://localhost:8080)
.PARAMETER MaxConcurrency
    Maximum concurrent users (default: 500)
.PARAMETER RampUpMinutes
    Ramp-up time in minutes (default: 2)
.PARAMETER SustainedMinutes
    Sustained load time in minutes (default: 5)
.PARAMETER TurboValidation
    Enable strict turbo mode validation (default: $true)
#>

param(
    [string]$Url = "http://localhost:8080",
    [int]$MaxConcurrency = 500,
    [int]$RampUpMinutes = 2,
    [int]$SustainedMinutes = 5,
    [bool]$TurboValidation = $true
)

class LoadTestResult {
    [string]$Phase
    [int]$Concurrency
    [double]$DurationSeconds
    [PerformanceMetrics]$Metrics
}

class PerformanceMetrics {
    [double]$MinLatency
    [double]$MaxLatency
    [double]$AvgLatency
    [double]$P50Latency
    [double]$P95Latency
    [double]$P99Latency
    [int]$TotalRequests
    [int]$SuccessfulRequests
    [int]$FailedRequests
    [double]$RequestsPerSecond
    [double]$ErrorRate
    [double]$Throughput
}

function Write-Header {
    param([string]$Text)
    Write-Host "`n$Text" -ForegroundColor Cyan
    Write-Host ("=" * $Text.Length) -ForegroundColor Cyan
}

function Invoke-LoadTest {
    param(
        [string]$Endpoint,
        [int]$Concurrency,
        [int]$DurationSeconds
    )

    $startTime = Get-Date
    $endTime = $startTime.AddSeconds($DurationSeconds)
    $results = @()
    $jobs = @()

    # Start concurrent jobs
    for ($i = 0; $i -lt $Concurrency; $i++) {
        $job = Start-Job -ScriptBlock {
            param($url, $endpoint, $endTime)

            $localResults = @()
            $session = New-Object Microsoft.PowerShell.Commands.WebRequestSession

            while ((Get-Date) -lt $endTime) {
                $start = Get-Date
                try {
                    $response = Invoke-WebRequest -Uri "$url$endpoint" -WebSession $session -TimeoutSec 1
                    $latency = ((Get-Date) - $start).TotalMilliseconds

                    $localResults += @{
                        Latency = $latency
                        StatusCode = $response.StatusCode
                        Success = $true
                    }
                }
                catch {
                    $latency = ((Get-Date) - $start).TotalMilliseconds
                    $localResults += @{
                        Latency = $latency
                        StatusCode = 0
                        Success = $false
                        Error = $_.Exception.Message
                    }
                }
            }

            return $localResults
        } -ArgumentList $Url, $Endpoint, $endTime

        $jobs += $job
    }

    # Wait for all jobs to complete
    $jobs | Wait-Job | Out-Null

    # Collect results
    foreach ($job in $jobs) {
        $results += Receive-Job -Job $job
        Remove-Job -Job $job
    }

    # Calculate metrics
    $metrics = [PerformanceMetrics]::new()
    $metrics.TotalRequests = $results.Count

    $successfulResults = $results | Where-Object { $_.Success }
    $metrics.SuccessfulRequests = $successfulResults.Count
    $metrics.FailedRequests = $metrics.TotalRequests - $metrics.SuccessfulRequests

    if ($metrics.SuccessfulRequests -gt 0) {
        $latencies = $successfulResults | ForEach-Object { $_.Latency } | Sort-Object

        $metrics.MinLatency = $latencies[0]
        $metrics.MaxLatency = $latencies[-1]
        $metrics.AvgLatency = ($latencies | Measure-Object -Average).Average

        $p50Index = [math]::Floor($latencies.Count * 0.5)
        $p95Index = [math]::Floor($latencies.Count * 0.95)
        $p99Index = [math]::Floor($latencies.Count * 0.99)

        $metrics.P50Latency = $latencies[$p50Index]
        $metrics.P95Latency = $latencies[$p95Index]
        $metrics.P99Latency = $latencies[$p99Index]

        $metrics.RequestsPerSecond = $metrics.SuccessfulRequests / $DurationSeconds
        $metrics.ErrorRate = ($metrics.FailedRequests / $metrics.TotalRequests) * 100
        $metrics.Throughput = $metrics.RequestsPerSecond
    }

    return $metrics
}

function New-LoadTestResult {
    param(
        [string]$Phase,
        [int]$Concurrency,
        [PerformanceMetrics]$Metrics
    )

    $result = [LoadTestResult]::new()
    $result.Phase = $Phase
    $result.Concurrency = $Concurrency
    $result.DurationSeconds = 30  # Each phase is 30 seconds
    $result.Metrics = $Metrics

    return $result
}

function Test-RampUpPhase {
    param([string]$Endpoint)

    Write-Header "RAMP-UP PHASE"

    $rampUpResults = @()
    $concurrencyLevels = @(10, 25, 50, 100, 200, 300, 400, 500)

    foreach ($concurrency in $concurrencyLevels) {
        Write-Host "Testing with $concurrency concurrent users..." -ForegroundColor Yellow

        $metrics = Invoke-LoadTest -Endpoint $Endpoint -Concurrency $concurrency -DurationSeconds 30
        $result = New-LoadTestResult -Phase "RampUp" -Concurrency $concurrency -Metrics $metrics
        $rampUpResults += $result

        Write-Host "  P95 Latency: $($metrics.P95Latency.ToString("F2"))ms" -ForegroundColor Blue
        Write-Host "  Throughput: $($metrics.RequestsPerSecond.ToString("F0")) req/s" -ForegroundColor Blue
        Write-Host "  Error Rate: $($metrics.ErrorRate.ToString("F2"))%" -ForegroundColor Blue

        # Check if latency is degrading significantly
        if ($metrics.P95Latency -gt 5.0) {
            Write-Host "  ⚠️  High latency detected at $concurrency users" -ForegroundColor Yellow
        }
    }

    return $rampUpResults
}

function Test-SustainedLoadPhase {
    param([string]$Endpoint, [int]$Concurrency)

    Write-Header "SUSTAINED LOAD PHASE"

    Write-Host "Testing sustained load with $Concurrency concurrent users..." -ForegroundColor Yellow

    $sustainedResults = @()
    $phaseDuration = $SustainedMinutes * 60

    $metrics = Invoke-LoadTest -Endpoint $Endpoint -Concurrency $Concurrency -DurationSeconds $phaseDuration
    $result = New-LoadTestResult -Phase "Sustained" -Concurrency $Concurrency -Metrics $metrics
    $sustainedResults += $result

    Write-Host "  Duration: $phaseDuration seconds" -ForegroundColor Blue
    Write-Host "  P95 Latency: $($metrics.P95Latency.ToString("F2"))ms" -ForegroundColor Blue
    Write-Host "  Throughput: $($metrics.RequestsPerSecond.ToString("F0")) req/s" -ForegroundColor Blue
    Write-Host "  Error Rate: $($metrics.ErrorRate.ToString("F2"))%" -ForegroundColor Blue

    return $sustainedResults
}

function Test-CooldownPhase {
    param([string]$Endpoint)

    Write-Header "COOLDOWN PHASE"

    Write-Host "Testing cooldown with reduced load..." -ForegroundColor Yellow

    $cooldownResults = @()
    $concurrencyLevels = @(100, 50, 25, 10)

    foreach ($concurrency in $concurrencyLevels) {
        $metrics = Invoke-LoadTest -Endpoint $Endpoint -Concurrency $concurrency -DurationSeconds 15
        $result = New-LoadTestResult -Phase "Cooldown" -Concurrency $concurrency -Metrics $metrics
        $cooldownResults += $result

        Write-Host "  $concurrency users - P95: $($metrics.P95Latency.ToString("F2"))ms" -ForegroundColor Blue
    }

    return $cooldownResults
}

function Validate-TurboPerformance {
    param([LoadTestResult[]]$Results)

    Write-Header "TURBO MODE VALIDATION"

    $turboTargets = @{
        P95Latency = 2.5  # ms
        MaxLatency = 5.0  # ms
        ErrorRate = 1.0   # %
        MinThroughput = 1000  # req/s
    }

    $validationResults = @()

    foreach ($result in $Results) {
        $metrics = $result.Metrics
        $validation = @{
            Phase = $result.Phase
            Concurrency = $result.Concurrency
            P95LatencyOK = $metrics.P95Latency -le $turboTargets.P95Latency
            MaxLatencyOK = $metrics.MaxLatency -le $turboTargets.MaxLatency
            ErrorRateOK = $metrics.ErrorRate -le $turboTargets.ErrorRate
            ThroughputOK = $metrics.RequestsPerSecond -ge $turboTargets.MinThroughput
        }

        $validationResults += $validation

        Write-Host "`n$($result.Phase) Phase - $($result.Concurrency) users:" -ForegroundColor Yellow

        if ($validation.P95LatencyOK) {
            Write-Host "  ✓ P95 Latency: $($metrics.P95Latency.ToString("F2"))ms (≤$($turboTargets.P95Latency)ms)" -ForegroundColor Green
        } else {
            Write-Host "  ✗ P95 Latency: $($metrics.P95Latency.ToString("F2"))ms (>$($turboTargets.P95Latency)ms)" -ForegroundColor Red
        }

        if ($validation.MaxLatencyOK) {
            Write-Host "  ✓ Max Latency: $($metrics.MaxLatency.ToString("F2"))ms (≤$($turboTargets.MaxLatency)ms)" -ForegroundColor Green
        } else {
            Write-Host "  ✗ Max Latency: $($metrics.MaxLatency.ToString("F2"))ms (>$($turboTargets.MaxLatency)ms)" -ForegroundColor Red
        }

        if ($validation.ErrorRateOK) {
            Write-Host "  ✓ Error Rate: $($metrics.ErrorRate.ToString("F2"))% (≤$($turboTargets.ErrorRate)%))" -ForegroundColor Green
        } else {
            Write-Host "  ✗ Error Rate: $($metrics.ErrorRate.ToString("F2"))% (>$($turboTargets.ErrorRate)%))" -ForegroundColor Red
        }

        if ($validation.ThroughputOK) {
            Write-Host "  ✓ Throughput: $($metrics.RequestsPerSecond.ToString("F0")) req/s (≥$($turboTargets.MinThroughput))" -ForegroundColor Green
        } else {
            Write-Host "  ✗ Throughput: $($metrics.RequestsPerSecond.ToString("F0")) req/s (<$($turboTargets.MinThroughput))" -ForegroundColor Red
        }
    }

    # Overall assessment
    $allPassed = ($validationResults | Where-Object { -not ($_.P95LatencyOK -and $_.MaxLatencyOK -and $_.ErrorRateOK -and $_.ThroughputOK) }).Count -eq 0

    Write-Header "OVERALL ASSESSMENT"

    if ($allPassed) {
        Write-Host "🎉 TURBO MODE VALIDATION PASSED!" -ForegroundColor Green
        Write-Host "All performance targets met under load conditions." -ForegroundColor Green
    } else {
        Write-Host "⚠️  TURBO MODE VALIDATION FAILED!" -ForegroundColor Red
        Write-Host "Some performance targets not met under load conditions." -ForegroundColor Red
    }

    return $allPassed
}

function Show-ComprehensiveResults {
    param([LoadTestResult[]]$AllResults)

    Write-Header "COMPREHENSIVE LOAD TEST RESULTS"

    $groupedResults = $AllResults | Group-Object -Property Phase

    foreach ($group in $groupedResults) {
        Write-Host "`n$($group.Name.ToUpper()) PHASE RESULTS" -ForegroundColor Magenta
        Write-Host ("-" * 30) -ForegroundColor Magenta

        foreach ($result in $group.Group) {
            $metrics = $result.Metrics
            Write-Host "Concurrency: $($result.Concurrency) users" -ForegroundColor Yellow
            Write-Host "  Requests: $($metrics.TotalRequests) total, $($metrics.SuccessfulRequests) successful" -ForegroundColor White
            Write-Host "  Latency (ms): Min=$($metrics.MinLatency.ToString("F2")), Avg=$($metrics.AvgLatency.ToString("F2")), P95=$($metrics.P95Latency.ToString("F2")), Max=$($metrics.MaxLatency.ToString("F2"))" -ForegroundColor White
            Write-Host "  Performance: $($metrics.RequestsPerSecond.ToString("F0")) req/s, $($metrics.ErrorRate.ToString("F2"))% errors" -ForegroundColor White
        }
    }
}

# Main execution
Write-Header "BITCOIN SPRINT TURBO MODE LOAD TEST"

# Check API availability
Write-Host "Checking API availability..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "$Url/status" -TimeoutSec 10
    if ($response.StatusCode -eq 200) {
        Write-Host "✓ API is available" -ForegroundColor Green
    } else {
        Write-Host "✗ API returned status $($response.StatusCode)" -ForegroundColor Red
        exit 1
    }
}
catch {
    Write-Host "✗ Cannot connect to API: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Test the most critical endpoint
$testEndpoint = "/latest"

# Phase 1: Ramp-up
$rampUpResults = Test-RampUpPhase -Endpoint $testEndpoint

# Phase 2: Sustained load (find optimal concurrency from ramp-up)
$optimalConcurrency = 200  # Default
$bestP95 = [double]::MaxValue

foreach ($result in $rampUpResults) {
    if ($result.Metrics.P95Latency -lt $bestP95 -and $result.Metrics.P95Latency -le 3.0) {
        $bestP95 = $result.Metrics.P95Latency
        $optimalConcurrency = $result.Concurrency
    }
}

Write-Host "`nOptimal concurrency for sustained test: $optimalConcurrency users" -ForegroundColor Green

$sustainedResults = Test-SustainedLoadPhase -Endpoint $testEndpoint -Concurrency $optimalConcurrency

# Phase 3: Cooldown
$cooldownResults = Test-CooldownPhase -Endpoint $testEndpoint

# Combine all results
$allResults = $rampUpResults + $sustainedResults + $cooldownResults

# Validate turbo performance
if ($TurboValidation) {
    $validationPassed = Validate-TurboPerformance -Results $allResults
}

# Show comprehensive results
Show-ComprehensiveResults -Results $allResults

# Export results
$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$resultFile = "turbo-loadtest-$timestamp.json"
$allResults | ConvertTo-Json -Depth 10 | Out-File -FilePath $resultFile
Write-Host "`nDetailed results exported to: $resultFile" -ForegroundColor Blue

# Final summary
Write-Header "FINAL SUMMARY"

if ($TurboValidation) {
    if ($validationPassed) {
        Write-Host "✅ TURBO MODE LOAD TEST PASSED" -ForegroundColor Green
        Write-Host "The Bitcoin Sprint API successfully handles turbo mode loads within 1-3ms targets." -ForegroundColor Green
    } else {
        Write-Host "❌ TURBO MODE LOAD TEST FAILED" -ForegroundColor Red
        Write-Host "Further optimization required to meet 1-3ms turbo mode targets under load." -ForegroundColor Red
    }
} else {
    Write-Host "📊 Load test completed successfully." -ForegroundColor Blue
}
