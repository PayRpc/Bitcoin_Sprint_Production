# Bitcoin Sprint Tier Performance Comparison
# This script switches between different tiers and benchmarks their performance

param(
    [Parameter(Mandatory=$false)]
    [ValidateSet("free", "pro", "business", "turbo", "enterprise")]
    [string[]]$Tiers = @("free", "pro", "business", "turbo", "enterprise"),

    [Parameter(Mandatory=$false)]
    [int]$TestDuration = 30, # seconds per tier

    [Parameter(Mandatory=$false)]
    [int]$ConcurrentRequests = 10,

    [Parameter(Mandatory=$false)]
    [switch]$SkipBuild,

    [Parameter(Mandatory=$false)]
    [switch]$KeepRunning
)

$ErrorActionPreference = "Stop"

# Tier configurations for display
$tierConfig = @{
    "free" = @{
        "Name" = "Free Tier"
        "Color" = "Gray"
        "ExpectedLatency" = "2000-5000ms"
        "RateLimit" = "1 req/sec"
        "Description" = "Basic tier with limited resources"
    }
    "pro" = @{
        "Name" = "Pro Tier"
        "Color" = "Blue"
        "ExpectedLatency" = "500-1200ms"
        "RateLimit" = "10 req/sec"
        "Description" = "Professional tier with moderate resources"
    }
    "business" = @{
        "Name" = "Business Tier"
        "Color" = "Yellow"
        "ExpectedLatency" = "200-800ms"
        "RateLimit" = "50 req/sec"
        "Description" = "Business tier with higher performance"
    }
    "turbo" = @{
        "Name" = "Turbo Tier"
        "Color" = "Green"
        "ExpectedLatency" = "50-300ms"
        "RateLimit" = "100 req/sec"
        "Description" = "High-performance tier with ultra-low latency"
    }
    "enterprise" = @{
        "Name" = "Enterprise Tier"
        "Color" = "Magenta"
        "ExpectedLatency" = "10-150ms"
        "RateLimit" = "500 req/sec"
        "Description" = "Enterprise tier with maximum performance"
    }
}

$results = @{}

function Write-ColoredOutput {
    param($Message, $Color = "White")
    Write-Host $Message -ForegroundColor $Color
}

function Switch-Tier {
    param($Tier)

    Write-ColoredOutput "`n🔄 Switching to $($tierConfig[$Tier].Name)..." $tierConfig[$Tier].Color

    $envFile = ".env.$Tier"

    if (Test-Path $envFile) {
        try {
            Copy-Item $envFile ".env" -Force
            Write-ColoredOutput "✅ Successfully switched to $Tier tier" "Green"

            # Display tier configuration
            Write-ColoredOutput "📊 Tier Configuration:" "Cyan"
            Write-ColoredOutput "  • Expected Latency: $($tierConfig[$Tier].ExpectedLatency)" "White"
            Write-ColoredOutput "  • Rate Limit: $($tierConfig[$Tier].RateLimit)" "White"
            Write-ColoredOutput "  • Description: $($tierConfig[$Tier].Description)" "White"

            return $true
        }
        catch {
            Write-ColoredOutput "❌ Error switching tier: $($_.Exception.Message)" "Red"
            return $false
        }
    } else {
        Write-ColoredOutput "❌ Error: Tier '$Tier' configuration not found" "Red"
        return $false
    }
}

function Build-Application {
    Write-ColoredOutput "`n🔨 Building Bitcoin Sprint..." "Yellow"

    try {
        $buildOutput = & go build -ldflags="-s -w -extldflags=-static" -trimpath -o bitcoin-sprint.exe ./cmd/sprintd 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-ColoredOutput "✅ Build successful" "Green"
            return $true
        } else {
            Write-ColoredOutput "❌ Build failed:" "Red"
            Write-ColoredOutput $buildOutput "Red"
            return $false
        }
    }
    catch {
        Write-ColoredOutput "❌ Build error: $($_.Exception.Message)" "Red"
        return $false
    }
}

function Start-Application {
    Write-ColoredOutput "`n🚀 Starting Bitcoin Sprint..." "Green"

    try {
        # Start the application in background
        $process = Start-Process -FilePath ".\bitcoin-sprint.exe" -NoNewWindow -PassThru

        # Wait for application to start
        Start-Sleep -Seconds 3

        # Check if process is still running
        if (!$process.HasExited) {
            Write-ColoredOutput "✅ Application started (PID: $($process.Id))" "Green"
            return $process
        } else {
            Write-ColoredOutput "❌ Application failed to start" "Red"
            return $null
        }
    }
    catch {
        Write-ColoredOutput "❌ Error starting application: $($_.Exception.Message)" "Red"
        return $null
    }
}

function Stop-Application {
    param($Process)

    if ($Process -and !$Process.HasExited) {
        Write-ColoredOutput "`n🛑 Stopping application..." "Yellow"
        try {
            Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
            Write-ColoredOutput "✅ Application stopped" "Green"
        }
        catch {
            Write-ColoredOutput "⚠️ Could not stop application gracefully" "Yellow"
        }
    }
}

function Test-HealthEndpoint {
    param($MaxRetries = 10)

    Write-ColoredOutput "🏥 Testing health endpoint..." "Cyan"

    for ($i = 1; $i -le $MaxRetries; $i++) {
        try {
            $response = Invoke-WebRequest -Uri "http://127.0.0.1:8080/health" -Method GET -TimeoutSec 5
            if ($response.StatusCode -eq 200) {
                Write-ColoredOutput "✅ Health check passed" "Green"
                return $true
            }
        }
        catch {
            Write-ColoredOutput "⏳ Health check failed (attempt $i/$MaxRetries): $($_.Exception.Message)" "Yellow"
        }

        if ($i -lt $MaxRetries) {
            Start-Sleep -Seconds 2
        }
    }

    Write-ColoredOutput "❌ Health check failed after $MaxRetries attempts" "Red"
    return $false
}

function Run-PerformanceTest {
    param($Tier, $Duration, $ConcurrentRequests)

    Write-ColoredOutput "`n📊 Running performance test for $Tier tier..." $tierConfig[$Tier].Color
    Write-ColoredOutput "⏱️ Test duration: $Duration seconds" "White"
    Write-ColoredOutput "🔄 Concurrent requests: $ConcurrentRequests" "White"

    $startTime = Get-Date
    $endTime = $startTime.AddSeconds($Duration)

    $results = @{
        "Tier" = $Tier
        "Requests" = 0
        "Successful" = 0
        "Failed" = 0
        "Latencies" = @()
        "Errors" = @()
        "StartTime" = $startTime
        "EndTime" = $null
    }

    # Create runspace pool for concurrent requests
    $runspacePool = [runspacefactory]::CreateRunspacePool(1, $ConcurrentRequests)
    $runspacePool.Open()

    $jobs = @()

    # Start concurrent request jobs
    for ($i = 1; $i -le $ConcurrentRequests; $i++) {
        $job = [powershell]::Create().AddScript({
            param($EndTime, $JobId)

            $localResults = @{
                "Requests" = 0
                "Successful" = 0
                "Failed" = 0
                "Latencies" = @()
                "Errors" = @()
            }

            while ((Get-Date) -lt $EndTime) {
                $localResults.Requests++

                $requestStart = Get-Date
                try {
                    $response = Invoke-WebRequest -Uri "http://127.0.0.1:8080/health" -Method GET -TimeoutSec 10
                    $requestEnd = Get-Date
                    $latency = ($requestEnd - $requestStart).TotalMilliseconds

                    if ($response.StatusCode -eq 200) {
                        $localResults.Successful++
                        $localResults.Latencies += $latency
                    } else {
                        $localResults.Failed++
                        $localResults.Errors += "HTTP $($response.StatusCode)"
                    }
                }
                catch {
                    $localResults.Failed++
                    $localResults.Errors += $_.Exception.Message
                }

                # Small delay to prevent overwhelming
                Start-Sleep -Milliseconds 100
            }

            return $localResults
        }).AddArgument($endTime).AddArgument($i)

        $job.RunspacePool = $runspacePool
        $jobs += @{ "Job" = $job; "Handle" = $job.BeginInvoke() }
    }

    # Wait for all jobs to complete
    foreach ($jobInfo in $jobs) {
        $result = $jobInfo.Job.EndInvoke($jobInfo.Handle)

        $results.Requests += $result.Requests
        $results.Successful += $result.Successful
        $results.Failed += $result.Failed
        $results.Latencies += $result.Latencies
        $results.Errors += $result.Errors
    }

    $runspacePool.Close()

    $results.EndTime = Get-Date

    return $results
}

function Display-Results {
    param($TestResults)

    Write-ColoredOutput "`n📈 PERFORMANCE RESULTS SUMMARY" "Cyan"
    Write-ColoredOutput "=" * 50 "Cyan"

    foreach ($tier in $TestResults.Keys | Sort-Object) {
        $result = $TestResults[$tier]

        Write-ColoredOutput "`n🏷️ $($tierConfig[$tier].Name.ToUpper())" $tierConfig[$tier].Color
        Write-ColoredOutput "-" * 30 $tierConfig[$tier].Color

        Write-ColoredOutput "📊 Request Statistics:" "White"
        Write-ColoredOutput "  • Total Requests: $($result.Requests)" "White"
        Write-ColoredOutput "  • Successful: $($result.Successful)" "Green"
        Write-ColoredOutput "  • Failed: $($result.Failed)" "Red"

        if ($result.Successful -gt 0) {
            $successRate = [math]::Round(($result.Successful / $result.Requests) * 100, 2)
            Write-ColoredOutput "  • Success Rate: $successRate%" "White"

            $avgLatency = [math]::Round(($result.Latencies | Measure-Object -Average).Average, 2)
            $minLatency = [math]::Round(($result.Latencies | Measure-Object -Minimum).Minimum, 2)
            $maxLatency = [math]::Round(($result.Latencies | Measure-Object -Maximum).Maximum, 2)

            Write-ColoredOutput "`n⏱️ Latency Statistics:" "White"
            Write-ColoredOutput "  • Average: $avgLatency ms" "White"
            Write-ColoredOutput "  • Minimum: $minLatency ms" "Green"
            Write-ColoredOutput "  • Maximum: $maxLatency ms" "Red"

            $requestsPerSecond = [math]::Round($result.Requests / $TestDuration, 2)
            Write-ColoredOutput "`n⚡ Throughput:" "White"
            Write-ColoredOutput "  • Requests/second: $requestsPerSecond" "White"
        }

        if ($result.Errors.Count -gt 0) {
            Write-ColoredOutput "`n❌ Top Errors:" "Red"
            $errorGroups = $result.Errors | Group-Object | Sort-Object Count -Descending | Select-Object -First 5
            foreach ($error in $errorGroups) {
                Write-ColoredOutput "  • $($error.Name): $($error.Count) times" "Red"
            }
        }
    }

    # Performance comparison
    Write-ColoredOutput "`n🏁 PERFORMANCE COMPARISON" "Cyan"
    Write-ColoredOutput "=" * 30 "Cyan"

    $comparisonData = @()
    foreach ($tier in $TestResults.Keys | Sort-Object) {
        $result = $TestResults[$tier]
        if ($result.Successful -gt 0) {
            $avgLatency = [math]::Round(($result.Latencies | Measure-Object -Average).Average, 2)
            $requestsPerSecond = [math]::Round($result.Requests / $TestDuration, 2)
            $successRate = [math]::Round(($result.Successful / $result.Requests) * 100, 2)

            $comparisonData += @{
                "Tier" = $tier
                "AvgLatency" = $avgLatency
                "RequestsPerSecond" = $requestsPerSecond
                "SuccessRate" = $successRate
            }
        }
    }

    if ($comparisonData.Count -gt 1) {
        # Sort by average latency (lower is better)
        $sortedByLatency = $comparisonData | Sort-Object AvgLatency
        Write-ColoredOutput "`n🏆 By Average Latency (lower is better):" "Yellow"
        foreach ($data in $sortedByLatency) {
            Write-ColoredOutput "  $($data.Tier): $($data.AvgLatency)ms" $tierConfig[$data.Tier].Color
        }

        # Sort by throughput (higher is better)
        $sortedByThroughput = $comparisonData | Sort-Object RequestsPerSecond -Descending
        Write-ColoredOutput "`n🚀 By Throughput (higher is better):" "Yellow"
        foreach ($data in $sortedByThroughput) {
            Write-ColoredOutput "  $($data.Tier): $($data.RequestsPerSecond) req/sec" $tierConfig[$data.Tier].Color
        }
    }
}

# Main execution
Write-ColoredOutput "🚀 BITCOIN SPRINT TIER PERFORMANCE COMPARISON" "Cyan"
Write-ColoredOutput "=" * 60 "Cyan"
Write-ColoredOutput "Testing tiers: $($Tiers -join ', ')" "White"
Write-ColoredOutput "Test duration per tier: $TestDuration seconds" "White"
Write-ColoredOutput "Concurrent requests: $ConcurrentRequests" "White"

$allResults = @{}

foreach ($tier in $Tiers) {
    Write-ColoredOutput "`n`n🎯 TESTING TIER: $($tierConfig[$tier].Name.ToUpper())" $tierConfig[$tier].Color
    Write-ColoredOutput "=" * 50 $tierConfig[$tier].Color

    # Switch to tier
    if (!(Switch-Tier $tier)) {
        Write-ColoredOutput "⏭️ Skipping $tier tier due to configuration error" "Yellow"
        continue
    }

    # Build application (unless skipped)
    if (!$SkipBuild) {
        if (!(Build-Application)) {
            Write-ColoredOutput "⏭️ Skipping $tier tier due to build failure" "Yellow"
            continue
        }
    }

    # Start application
    $process = Start-Application
    if (!$process) {
        Write-ColoredOutput "⏭️ Skipping $tier tier due to startup failure" "Yellow"
        continue
    }

    # Wait for application to be ready
    if (!(Test-HealthEndpoint)) {
        Write-ColoredOutput "⏭️ Skipping $tier tier due to health check failure" "Yellow"
        Stop-Application $process
        continue
    }

    # Run performance test
    $testResults = Run-PerformanceTest $tier $TestDuration $ConcurrentRequests
    $allResults[$tier] = $testResults

    # Stop application (unless keep running is specified)
    if (!$KeepRunning) {
        Stop-Application $process
    }
}

# Display final results
if ($allResults.Count -gt 0) {
    Display-Results $allResults

    Write-ColoredOutput "`n✅ Tier performance comparison completed!" "Green"
    Write-ColoredOutput "💡 Tip: Enterprise tier typically shows 10-100x better performance than Free tier" "Cyan"
} else {
    Write-ColoredOutput "`n❌ No successful tests completed" "Red"
}
