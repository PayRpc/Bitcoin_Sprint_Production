#!/usr/bin/env pwsh

<#
.SYNOPSIS
    Turbo Mode Performance Benchmark for Bitcoin Sprint
    Validates 1-3ms response time target for turbo tier
.DESCRIPTION
    Comprehensive performance testing suite for Bitcoin Sprint API
    Measures latency, throughput, and validates turbo mode optimizations
.PARAMETER Url
    Base URL of the Bitcoin Sprint API (default: http://localhost:8080)
.PARAMETER DurationSeconds
    Test duration in seconds (default: 60)
.PARAMETER ConcurrentUsers
    Number of concurrent users (default: 100)
.PARAMETER TurboMode
    Enable turbo mode validation (default: $true)
#>

param(
    [string]$Url = "http://localhost:8080",
    [int]$DurationSeconds = 60,
    [int]$ConcurrentUsers = 100,
    [bool]$TurboMode = $true
)

# Configuration
$Config = @{
    Endpoints = @("/status", "/latest", "/health", "/version")
    TurboTarget = @{
        MinLatency = 1    # ms
        MaxLatency = 3    # ms
        TargetP95 = 2.5   # ms
    }
    WarmupSeconds = 10
    CooldownSeconds = 5
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
}

function Write-Header {
    param([string]$Text)
    Write-Host "`n$Text" -ForegroundColor Cyan
    Write-Host ("=" * $Text.Length) -ForegroundColor Cyan
}

function Test-Endpoint {
    param(
        [string]$Endpoint,
        [int]$Duration,
        [int]$Concurrency
    )

    Write-Host "Testing endpoint: $Endpoint" -ForegroundColor Yellow

    $startTime = Get-Date
    $endTime = $startTime.AddSeconds($Duration)
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
                    $response = Invoke-WebRequest -Uri "$url$endpoint" -WebSession $session -TimeoutSec 5
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

        $metrics.RequestsPerSecond = $metrics.SuccessfulRequests / $Duration
        $metrics.ErrorRate = ($metrics.FailedRequests / $metrics.TotalRequests) * 100
    }

    return $metrics
}

function Validate-TurboMode {
    param([PerformanceMetrics]$Metrics, [string]$Endpoint)

    $target = $Config.TurboTarget
    $passed = $true
    $issues = @()

    Write-Host "`nTurbo Mode Validation for $Endpoint" -ForegroundColor Magenta
    Write-Host ("-" * 40) -ForegroundColor Magenta

    # Check P95 latency (most important metric)
    if ($Metrics.P95Latency -gt $target.TargetP95) {
        $passed = $false
        $issues += "P95 latency ($($Metrics.P95Latency.ToString("F2"))ms) exceeds target ($($target.TargetP95)ms)"
    } else {
        Write-Host "✓ P95 latency: $($Metrics.P95Latency.ToString("F2"))ms (target: ≤$($target.TargetP95)ms)" -ForegroundColor Green
    }

    # Check max latency
    if ($Metrics.MaxLatency -gt $target.MaxLatency) {
        $passed = $false
        $issues += "Max latency ($($Metrics.MaxLatency.ToString("F2"))ms) exceeds target ($($target.MaxLatency)ms)"
    } else {
        Write-Host "✓ Max latency: $($Metrics.MaxLatency.ToString("F2"))ms (target: ≤$($target.MaxLatency)ms)" -ForegroundColor Green
    }

    # Check error rate
    if ($Metrics.ErrorRate -gt 1.0) {
        $passed = $false
        $issues += "Error rate ($($Metrics.ErrorRate.ToString("F2"))%) exceeds 1%"
    } else {
        Write-Host "✓ Error rate: $($Metrics.ErrorRate.ToString("F2"))%" -ForegroundColor Green
    }

    # Check throughput
    if ($Metrics.RequestsPerSecond -lt 1000) {
        $issues += "Throughput ($($Metrics.RequestsPerSecond.ToString("F0")) req/s) below 1000 req/s target"
    } else {
        Write-Host "✓ Throughput: $($Metrics.RequestsPerSecond.ToString("F0")) req/s" -ForegroundColor Green
    }

    if (-not $passed) {
        Write-Host "✗ Issues found:" -ForegroundColor Red
        foreach ($issue in $issues) {
            Write-Host "  - $issue" -ForegroundColor Red
        }
    } else {
        Write-Host "✓ All turbo mode targets met!" -ForegroundColor Green
    }

    return $passed
}

function Show-Results {
    param([hashtable]$Results)

    Write-Header "PERFORMANCE TEST RESULTS"

    $allPassed = $true
    foreach ($endpoint in $Config.Endpoints) {
        if ($Results.ContainsKey($endpoint)) {
            $metrics = $Results[$endpoint]

            Write-Host "`nEndpoint: $endpoint" -ForegroundColor Yellow
            Write-Host "  Total Requests: $($metrics.TotalRequests)"
            Write-Host "  Successful: $($metrics.SuccessfulRequests)"
            Write-Host "  Failed: $($metrics.FailedRequests)"
            Write-Host "  Error Rate: $($metrics.ErrorRate.ToString("F2"))%"
            Write-Host "  Requests/sec: $($metrics.RequestsPerSecond.ToString("F0"))"
            Write-Host "  Latency (ms):"
            Write-Host "    Min: $($metrics.MinLatency.ToString("F2"))"
            Write-Host "    Avg: $($metrics.AvgLatency.ToString("F2"))"
            Write-Host "    P50: $($metrics.P50Latency.ToString("F2"))"
            Write-Host "    P95: $($metrics.P95Latency.ToString("F2"))"
            Write-Host "    P99: $($metrics.P99Latency.ToString("F2"))"
            Write-Host "    Max: $($metrics.MaxLatency.ToString("F2"))"

            if ($TurboMode) {
                $passed = Validate-TurboMode -Metrics $metrics -Endpoint $endpoint
                if (-not $passed) { $allPassed = $false }
            }
        }
    }

    Write-Header "SUMMARY"

    if ($TurboMode) {
        if ($allPassed) {
            Write-Host "🎉 ALL TESTS PASSED! Turbo mode is performing within 1-3ms target." -ForegroundColor Green
        } else {
            Write-Host "⚠️  Some tests failed. Turbo mode needs optimization." -ForegroundColor Red
        }
    } else {
        Write-Host "📊 Performance test completed." -ForegroundColor Blue
    }
}

# Main execution
Write-Header "BITCOIN SPRINT TURBO MODE PERFORMANCE TEST"

# Check if API is available
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

# Warmup phase
Write-Host "`nStarting warmup phase ($($Config.WarmupSeconds)s)..." -ForegroundColor Yellow
Start-Sleep -Seconds $Config.WarmupSeconds

# Run tests
$results = @{}
foreach ($endpoint in $Config.Endpoints) {
    $results[$endpoint] = Test-Endpoint -Endpoint $endpoint -Duration $DurationSeconds -Concurrency $ConcurrentUsers
}

# Cooldown phase
Write-Host "`nCooldown phase ($($Config.CooldownSeconds)s)..." -ForegroundColor Yellow
Start-Sleep -Seconds $Config.CooldownSeconds

# Show results
Show-Results -Results $results

# Export results
$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$resultFile = "turbo-benchmark-$timestamp.json"
$results | ConvertTo-Json -Depth 10 | Out-File -FilePath $resultFile
Write-Host "`nResults exported to: $resultFile" -ForegroundColor Blue
