#!/usr/bin/env pwsh

<#
.SYNOPSIS
    Memory Profiling for Bitcoin Sprint Turbo Mode
    Validates memory efficiency of turbo optimizations
.DESCRIPTION
    Comprehensive memory analysis including heap usage, GC pressure,
    and allocation patterns during turbo mode operation
.PARAMETER Url
    Base URL of the Bitcoin Sprint API (default: http://localhost:8080)
.PARAMETER DurationMinutes
    Test duration in minutes (default: 5)
.PARAMETER ConcurrentUsers
    Number of concurrent users (default: 200)
.PARAMETER EnableGCStats
    Enable detailed GC statistics (default: $true)
#>

param(
    [string]$Url = "http://localhost:8080",
    [int]$DurationMinutes = 5,
    [int]$ConcurrentUsers = 200,
    [bool]$EnableGCStats = $true
)

class MemoryMetrics {
    [long]$HeapAlloc
    [long]$HeapInuse
    [long]$HeapObjects
    [long]$HeapReleased
    [long]$HeapSys
    [long]$StackInuse
    [long]$StackSys
    [int]$GCCycles
    [long]$GCPauseTotalNs
    [double]$GCPauseAvgMs
    [long]$NextGC
    [double]$GCUtilization
}

class MemorySnapshot {
    [DateTime]$Timestamp
    [MemoryMetrics]$Metrics
    [double]$TestLatency
    [int]$ActiveGoroutines
}

function Write-Header {
    param([string]$Text)
    Write-Host "`n$Text" -ForegroundColor Cyan
    Write-Host ("=" * $Text.Length) -ForegroundColor Cyan
}

function Get-MemoryStats {
    param([string]$Url)

    try {
        $response = Invoke-WebRequest -Uri "$Url/debug/vars" -TimeoutSec 5
        $stats = $response.Content | ConvertFrom-Json

        $metrics = [MemoryMetrics]::new()

        if ($stats.memstats) {
            $mem = $stats.memstats
            $metrics.HeapAlloc = $mem.HeapAlloc
            $metrics.HeapInuse = $mem.HeapInuse
            $metrics.HeapObjects = $mem.HeapObjects
            $metrics.HeapReleased = $mem.HeapReleased
            $metrics.HeapSys = $mem.HeapSys
            $metrics.StackInuse = $mem.StackInuse
            $metrics.StackSys = $mem.StackSys
            $metrics.GCCycles = $mem.NumGC
            $metrics.GCPauseTotalNs = $mem.PauseTotalNs
            $metrics.NextGC = $mem.NextGC
            $metrics.GCUtilization = ($mem.GCSys / $mem.HeapSys) * 100
        }

        if ($metrics.GCCycles -gt 0) {
            $metrics.GCPauseAvgMs = ($metrics.GCPauseTotalNs / $metrics.GCCycles) / 1000000
        }

        return $metrics
    }
    catch {
        Write-Host "Warning: Could not retrieve memory stats from $Url/debug/vars" -ForegroundColor Yellow
        return $null
    }
}

function Get-GoroutineCount {
    param([string]$Url)

    try {
        $response = Invoke-WebRequest -Uri "$Url/debug/pprof/goroutine?debug=1" -TimeoutSec 5
        $content = $response.Content

        # Extract goroutine count from pprof output
        $match = [regex]::Match($content, '(\d+) goroutines')
        if ($match.Success) {
            return [int]$match.Groups[1].Value
        }
    }
    catch {
        Write-Host "Warning: Could not retrieve goroutine count" -ForegroundColor Yellow
    }

    return 0
}

function Format-Bytes {
    param([long]$Bytes)

    $sizes = @("B", "KB", "MB", "GB", "TB")
    $i = 0

    while ($Bytes -ge 1024 -and $i -lt $sizes.Count - 1) {
        $Bytes = $Bytes / 1024
        $i++
    }

    return "{0:N2} {1}" -f $Bytes, $sizes[$i]
}

function Invoke-MemoryLoadTest {
    param(
        [string]$Url,
        [int]$DurationSeconds,
        [int]$Concurrency
    )

    $startTime = Get-Date
    $endTime = $startTime.AddSeconds($DurationSeconds)
    $snapshots = @()
    $results = @()

    Write-Host "Starting memory profiling test..." -ForegroundColor Yellow

    # Take initial snapshot
    $initialMemory = Get-MemoryStats -Url $Url
    $initialGoroutines = Get-GoroutineCount -Url $Url

    if ($initialMemory) {
        $initialSnapshot = [MemorySnapshot]::new()
        $initialSnapshot.Timestamp = $startTime
        $initialSnapshot.Metrics = $initialMemory
        $initialSnapshot.ActiveGoroutines = $initialGoroutines
        $initialSnapshot.TestLatency = 0
        $snapshots += $initialSnapshot
    }

    # Start load test jobs
    $jobs = @()
    for ($i = 0; $i -lt $Concurrency; $i++) {
        $job = Start-Job -ScriptBlock {
            param($url, $endTime)

            $localResults = @()
            $session = New-Object Microsoft.PowerShell.Commands.WebRequestSession

            while ((Get-Date) -lt $endTime) {
                $start = Get-Date
                try {
                    $response = Invoke-WebRequest -Uri "$url/latest" -WebSession $session -TimeoutSec 1
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
        } -ArgumentList $Url, $endTime

        $jobs += $job
    }

    # Monitor memory during test
    $snapshotInterval = 10  # seconds
    $nextSnapshot = $startTime.AddSeconds($snapshotInterval)

    while ((Get-Date) -lt $endTime) {
        if ((Get-Date) -ge $nextSnapshot) {
            $currentMemory = Get-MemoryStats -Url $Url
            $currentGoroutines = Get-GoroutineCount -Url $Url

            if ($currentMemory) {
                $snapshot = [MemorySnapshot]::new()
                $snapshot.Timestamp = Get-Date
                $snapshot.Metrics = $currentMemory
                $snapshot.ActiveGoroutines = $currentGoroutines
                $snapshot.TestLatency = 0  # Will be calculated from job results
                $snapshots += $snapshot
            }

            $nextSnapshot = $nextSnapshot.AddSeconds($snapshotInterval)
        }

        Start-Sleep -Milliseconds 100
    }

    # Wait for load test jobs to complete
    $jobs | Wait-Job | Out-Null

    # Collect results
    foreach ($job in $jobs) {
        $results += Receive-Job -Job $job
        Remove-Job -Job $job
    }

    # Take final snapshot
    $finalMemory = Get-MemoryStats -Url $Url
    $finalGoroutines = Get-GoroutineCount -Url $Url

    if ($finalMemory) {
        $finalSnapshot = [MemorySnapshot]::new()
        $finalSnapshot.Timestamp = Get-Date
        $finalSnapshot.Metrics = $finalMemory
        $finalSnapshot.ActiveGoroutines = $finalGoroutines
        $finalSnapshot.TestLatency = 0
        $snapshots += $finalSnapshot
    }

    return @{
        Snapshots = $snapshots
        Results = $results
    }
}

function Analyze-MemoryUsage {
    param([MemorySnapshot[]]$Snapshots)

    Write-Header "MEMORY USAGE ANALYSIS"

    if ($Snapshots.Count -lt 2) {
        Write-Host "Insufficient memory snapshots for analysis" -ForegroundColor Red
        return
    }

    $initial = $Snapshots[0]
    $final = $Snapshots[-1]

    Write-Host "Initial Memory State:" -ForegroundColor Yellow
    Write-Host "  Heap Alloc: $(Format-Bytes $initial.Metrics.HeapAlloc)" -ForegroundColor White
    Write-Host "  Heap Inuse: $(Format-Bytes $initial.Metrics.HeapInuse)" -ForegroundColor White
    Write-Host "  Heap Objects: $($initial.Metrics.HeapObjects)" -ForegroundColor White
    Write-Host "  GC Cycles: $($initial.Metrics.GCCycles)" -ForegroundColor White
    Write-Host "  Goroutines: $($initial.ActiveGoroutines)" -ForegroundColor White

    Write-Host "`nFinal Memory State:" -ForegroundColor Yellow
    Write-Host "  Heap Alloc: $(Format-Bytes $final.Metrics.HeapAlloc)" -ForegroundColor White
    Write-Host "  Heap Inuse: $(Format-Bytes $final.Metrics.HeapInuse)" -ForegroundColor White
    Write-Host "  Heap Objects: $($final.Metrics.HeapObjects)" -ForegroundColor White
    Write-Host "  GC Cycles: $($final.Metrics.GCCycles)" -ForegroundColor White
    Write-Host "  Goroutines: $($final.ActiveGoroutines)" -ForegroundColor White

    # Calculate deltas
    $heapAllocDelta = $final.Metrics.HeapAlloc - $initial.Metrics.HeapAlloc
    $heapObjectsDelta = $final.Metrics.HeapObjects - $initial.Metrics.HeapObjects
    $gcCyclesDelta = $final.Metrics.GCCycles - $initial.Metrics.GCCycles
    $goroutinesDelta = $final.ActiveGoroutines - $initial.ActiveGoroutines

    Write-Host "`nMemory Usage Changes:" -ForegroundColor Magenta
    Write-Host "  Heap Alloc Delta: $(Format-Bytes $heapAllocDelta)" -ForegroundColor White
    Write-Host "  Heap Objects Delta: $heapObjectsDelta" -ForegroundColor White
    Write-Host "  GC Cycles Delta: $gcCyclesDelta" -ForegroundColor White
    Write-Host "  Goroutines Delta: $goroutinesDelta" -ForegroundColor White

    # Memory leak detection
    $memoryLeakThreshold = 50MB  # 50MB growth threshold
    $objectLeakThreshold = 10000  # 10K object growth threshold

    $hasMemoryLeak = [Math]::Abs($heapAllocDelta) -gt $memoryLeakThreshold
    $hasObjectLeak = [Math]::Abs($heapObjectsDelta) -gt $objectLeakThreshold

    if ($hasMemoryLeak -or $hasObjectLeak) {
        Write-Host "`n⚠️  POTENTIAL MEMORY LEAK DETECTED!" -ForegroundColor Red
        if ($hasMemoryLeak) {
            Write-Host "  - Heap allocation changed by $(Format-Bytes $heapAllocDelta)" -ForegroundColor Red
        }
        if ($hasObjectLeak) {
            Write-Host "  - Heap objects changed by $heapObjectsDelta" -ForegroundColor Red
        }
    } else {
        Write-Host "`n✅ No significant memory leaks detected" -ForegroundColor Green
    }

    # GC analysis
    if ($gcCyclesDelta -gt 0) {
        $avgGCPause = ($final.Metrics.GCPauseTotalNs - $initial.Metrics.GCPauseTotalNs) / $gcCyclesDelta / 1000000
        Write-Host "`nGC Analysis:" -ForegroundColor Magenta
        Write-Host "  GC Cycles: $gcCyclesDelta" -ForegroundColor White
        Write-Host "  Avg GC Pause: $($avgGCPause.ToString("F3"))ms" -ForegroundColor White

        if ($avgGCPause -gt 10) {
            Write-Host "  ⚠️  High GC pause time detected (>10ms)" -ForegroundColor Red
        } else {
            Write-Host "  ✅ GC pause times within acceptable range" -ForegroundColor Green
        }
    }
}

function Analyze-MemoryTrends {
    param([MemorySnapshot[]]$Snapshots)

    Write-Header "MEMORY TREND ANALYSIS"

    if ($Snapshots.Count -lt 3) {
        Write-Host "Insufficient snapshots for trend analysis" -ForegroundColor Yellow
        return
    }

    # Calculate trends
    $heapAllocTrend = @()
    $heapObjectsTrend = @()
    $gcPauseTrend = @()

    for ($i = 1; $i -lt $Snapshots.Count; $i++) {
        $prev = $Snapshots[$i-1]
        $curr = $Snapshots[$i]

        $timeDiff = ($curr.Timestamp - $prev.Timestamp).TotalSeconds

        if ($timeDiff -gt 0) {
            $heapAllocRate = ($curr.Metrics.HeapAlloc - $prev.Metrics.HeapAlloc) / $timeDiff / 1024 / 1024  # MB/s
            $heapObjectsRate = ($curr.Metrics.HeapObjects - $prev.Metrics.HeapObjects) / $timeDiff

            $heapAllocTrend += $heapAllocRate
            $heapObjectsTrend += $heapObjectsRate
        }
    }

    if ($heapAllocTrend.Count -gt 0) {
        $avgHeapAllocRate = ($heapAllocTrend | Measure-Object -Average).Average
        $avgHeapObjectsRate = ($heapObjectsTrend | Measure-Object -Average).Average

        Write-Host "Average Memory Allocation Rate:" -ForegroundColor Yellow
        Write-Host "  Heap: $($avgHeapAllocRate.ToString("F2")) MB/s" -ForegroundColor White
        Write-Host "  Objects: $($avgHeapObjectsRate.ToString("F2")) objects/s" -ForegroundColor White

        # Check for concerning allocation rates
        if ([Math]::Abs($avgHeapAllocRate) -gt 10) {  # >10MB/s sustained
            Write-Host "  ⚠️  High sustained heap allocation rate" -ForegroundColor Red
        } else {
            Write-Host "  ✅ Heap allocation rate within normal range" -ForegroundColor Green
        }

        if ([Math]::Abs($avgHeapObjectsRate) -gt 1000) {  # >1000 objects/s sustained
            Write-Host "  ⚠️  High sustained object allocation rate" -ForegroundColor Red
        } else {
            Write-Host "  ✅ Object allocation rate within normal range" -ForegroundColor Green
        }
    }
}

function Show-PerformanceSummary {
    param([array]$Results)

    Write-Header "PERFORMANCE SUMMARY"

    $successfulResults = $Results | Where-Object { $_.Success }
    $totalRequests = $Results.Count
    $successfulRequests = $successfulResults.Count
    $failedRequests = $totalRequests - $successfulRequests

    if ($successfulRequests -gt 0) {
        $latencies = $successfulResults | ForEach-Object { $_.Latency } | Sort-Object
        $avgLatency = ($latencies | Measure-Object -Average).Average
        $p95Latency = $latencies[[math]::Floor($latencies.Count * 0.95)]
        $maxLatency = $latencies[-1]

        Write-Host "Request Statistics:" -ForegroundColor Yellow
        Write-Host "  Total Requests: $totalRequests" -ForegroundColor White
        Write-Host "  Successful: $successfulRequests" -ForegroundColor White
        Write-Host "  Failed: $failedRequests" -ForegroundColor White
        Write-Host "  Success Rate: $(([double]$successfulRequests / $totalRequests * 100).ToString("F2"))%" -ForegroundColor White

        Write-Host "`nLatency Statistics (ms):" -ForegroundColor Yellow
        Write-Host "  Average: $($avgLatency.ToString("F2"))" -ForegroundColor White
        Write-Host "  P95: $($p95Latency.ToString("F2"))" -ForegroundColor White
        Write-Host "  Max: $($maxLatency.ToString("F2"))" -ForegroundColor White

        # Turbo mode validation
        if ($p95Latency -le 2.5 -and $maxLatency -le 5.0) {
            Write-Host "`n✅ TURBO MODE PERFORMANCE ACHIEVED!" -ForegroundColor Green
        } elseif ($p95Latency -le 3.0 -and $maxLatency -le 10.0) {
            Write-Host "`n⚠️  NEAR TURBO MODE PERFORMANCE" -ForegroundColor Yellow
        } else {
            Write-Host "`n❌ TURBO MODE PERFORMANCE NOT ACHIEVED" -ForegroundColor Red
        }
    }
}

# Main execution
Write-Header "BITCOIN SPRINT MEMORY PROFILING"

# Check API availability and debug endpoint
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

# Check debug endpoint availability
$debugAvailable = $false
try {
    $response = Invoke-WebRequest -Uri "$Url/debug/vars" -TimeoutSec 5
    if ($response.StatusCode -eq 200) {
        Write-Host "✓ Debug endpoint available" -ForegroundColor Green
        $debugAvailable = $true
    }
}
catch {
    Write-Host "⚠️  Debug endpoint not available - memory analysis will be limited" -ForegroundColor Yellow
}

# Run memory profiling test
$durationSeconds = $DurationMinutes * 60
$testResults = Invoke-MemoryLoadTest -Url $Url -DurationSeconds $durationSeconds -Concurrency $ConcurrentUsers

# Analyze results
if ($debugAvailable -and $testResults.Snapshots.Count -gt 0) {
    Analyze-MemoryUsage -Snapshots $testResults.Snapshots
    Analyze-MemoryTrends -Snapshots $testResults.Snapshots
} else {
    Write-Host "`nSkipping detailed memory analysis (debug endpoint not available)" -ForegroundColor Yellow
}

Show-PerformanceSummary -Results $testResults.Results

# Export results
$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$resultFile = "memory-profile-$timestamp.json"
$testResults | ConvertTo-Json -Depth 10 | Out-File -FilePath $resultFile
Write-Host "`nDetailed results exported to: $resultFile" -ForegroundColor Blue

Write-Header "MEMORY PROFILING COMPLETE"
