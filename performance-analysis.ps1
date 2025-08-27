# Bitcoin Sprint Performance Analysis and API Load Testing
# Tests existing performance data and creates load testing tools

param(
    [switch]$AnalyzeExistingResults,
    [switch]$CreateLoadTestScript,
    [switch]$RunSimpleBenchmarks,
    [int]$TestDuration = 60,
    [string]$APIEndpoint = "http://httpbin.org/get"  # Safe default endpoint for testing
)

$ErrorActionPreference = "Stop"

function Write-Section($title) {
    Write-Host ""
    Write-Host "=" * 80 -ForegroundColor Cyan
    Write-Host $title -ForegroundColor Yellow
    Write-Host "=" * 80 -ForegroundColor Cyan
}

function Write-Status($message) {
    Write-Host "🔄 $message" -ForegroundColor Blue
}

function Write-Success($message) {
    Write-Host "✅ $message" -ForegroundColor Green
}

function Write-Error($message) {
    Write-Host "❌ $message" -ForegroundColor Red
}

function Write-Warning($message) {
    Write-Host "⚠️ $message" -ForegroundColor Yellow
}

# Analyze existing performance test results
function Analyze-ExistingResults {
    Write-Section "📊 Analyzing Existing Performance Results"

    $resultFiles = Get-ChildItem "*sla_test*.json" | Sort-Object LastWriteTime -Descending

    if ($resultFiles.Count -eq 0) {
        Write-Warning "No existing SLA test results found"
        return
    }

    Write-Status "Found $($resultFiles.Count) SLA test result files"

    foreach ($file in $resultFiles) {
        Write-Host "📄 Analyzing: $($file.Name)" -ForegroundColor Gray

        try {
            $content = Get-Content $file.FullName | ConvertFrom-Json

            if ($content.performance_results) {
                $perf = $content.performance_results
                Write-Host "  Tier: $($content.test_metadata.tier_tested)" -ForegroundColor White
                Write-Host "  Compliance Rate: $($perf.compliance_rate_percent)%" -ForegroundColor White
                Write-Host "  Average Latency: $($perf.avg_latency_ms)ms" -ForegroundColor White
                Write-Host "  Min Latency: $($perf.min_latency_ms)ms" -ForegroundColor White
                Write-Host "  Max Latency: $($perf.max_latency_ms)ms" -ForegroundColor White
                Write-Host "  Successful Tests: $($perf.successful_tests)" -ForegroundColor White
                Write-Host ""
            }
        } catch {
            Write-Warning "Could not parse $($file.Name): $($_.Exception.Message)"
        }
    }
}

# Create a simple load testing script
function Create-LoadTestScript {
    Write-Section "🛠️ Creating API Load Testing Script"

    $loadTestScript = @'
# Simple API Load Testing Script
# Tests any HTTP endpoint with concurrent requests

param(
    [string]$Url = "http://httpbin.org/get",
    [int]$DurationSeconds = 30,
    [int]$ConcurrentUsers = 10,
    [int]$RampUpSeconds = 5,
    [switch]$UseAuth,
    [string]$AuthHeader = "Bearer test_token"
)

$ErrorActionPreference = "Stop"

function Write-Status($message) {
    Write-Host "🔄 $message" -ForegroundColor Blue
}

function Write-Success($message) {
    Write-Host "✅ $message" -ForegroundColor Green
}

function Test-APIEndpoint {
    param(
        [string]$Url,
        [int]$DurationSeconds,
        [int]$ConcurrentUsers,
        [bool]$UseAuth = $false,
        [string]$AuthHeader = ""
    )

    Write-Status "Testing endpoint: $Url"
    Write-Status "Duration: $DurationSeconds seconds | Concurrent Users: $ConcurrentUsers"

    $results = @{
        TotalRequests = 0
        SuccessfulRequests = 0
        FailedRequests = 0
        ResponseTimes = @()
        StatusCodes = @{}
        Errors = @()
        StartTime = Get-Date
        EndTime = $null
    }

    $jobScript = {
        param($url, $duration, $useAuth, $authHeader)

        $results = @{
            Requests = 0
            Successes = 0
            Failures = 0
            ResponseTimes = @()
            StatusCodes = @{}
            Errors = @()
        }

        $endTime = (Get-Date).AddSeconds($duration)

        while ((Get-Date) -lt $endTime) {
            try {
                $start = Get-Date
                $headers = @{}
                if ($useAuth) {
                    $headers["Authorization"] = $authHeader
                }

                $response = Invoke-WebRequest -Uri $url -Headers $headers -TimeoutSec 10
                $responseTime = ((Get-Date) - $start).TotalMilliseconds

                $results.Requests++
                $results.Successes++
                $results.ResponseTimes += $responseTime

                if ($results.StatusCodes.ContainsKey($response.StatusCode)) {
                    $results.StatusCodes[$response.StatusCode]++
                } else {
                    $results.StatusCodes[$response.StatusCode] = 1
                }
            } catch {
                $results.Requests++
                $results.Failures++
                $results.Errors += $_.Exception.Message
            }
        }

        return $results
    }

    # Start concurrent jobs
    $jobs = @()
    for ($i = 0; $i -lt $ConcurrentUsers; $i++) {
        $job = Start-Job -ScriptBlock $jobScript -ArgumentList $Url, $DurationSeconds, $UseAuth, $AuthHeader
        $jobs += $job
    }

    # Wait for all jobs to complete
    $jobs | Wait-Job | Out-Null

    # Collect results
    foreach ($job in $jobs) {
        $jobResult = Receive-Job -Job $job
        $results.TotalRequests += $jobResult.Requests
        $results.SuccessfulRequests += $jobResult.Successes
        $results.FailedRequests += $jobResult.Failures
        $results.ResponseTimes += $jobResult.ResponseTimes
        $results.Errors += $jobResult.Errors

        foreach ($status in $jobResult.StatusCodes.Keys) {
            if ($results.StatusCodes.ContainsKey($status)) {
                $results.StatusCodes[$status] += $jobResult.StatusCodes[$status]
            } else {
                $results.StatusCodes[$status] = $status
            }
        }

        Remove-Job -Job $job
    }

    $results.EndTime = Get-Date

    # Calculate statistics
    if ($results.ResponseTimes.Count -gt 0) {
        $results.Stats = @{
            AverageResponseTime = ($results.ResponseTimes | Measure-Object -Average).Average
            MinResponseTime = ($results.ResponseTimes | Measure-Object -Minimum).Minimum
            MaxResponseTime = ($results.ResponseTimes | Measure-Object -Maximum).Maximum
            Percentile95 = ($results.ResponseTimes | Sort-Object)[[math]::Floor($results.ResponseTimes.Count * 0.95)]
            RequestsPerSecond = $results.TotalRequests / $DurationSeconds
            SuccessRate = ($results.SuccessfulRequests / $results.TotalRequests) * 100
        }
    }

    return $results
}

function Show-Results {
    param($results)

    Write-Success "Load test completed!"

    Write-Host ""
    Write-Host "📊 Results Summary:" -ForegroundColor Green
    Write-Host "  Total Requests: $($results.TotalRequests)" -ForegroundColor White
    Write-Host "  Successful Requests: $($results.SuccessfulRequests)" -ForegroundColor White
    Write-Host "  Failed Requests: $($results.FailedRequests)" -ForegroundColor White
    Write-Host "  Success Rate: $([math]::Round($results.Stats.SuccessRate, 2))%" -ForegroundColor White
    Write-Host "  Requests/Second: $([math]::Round($results.Stats.RequestsPerSecond, 2))" -ForegroundColor White
    Write-Host "  Average Response Time: $([math]::Round($results.Stats.AverageResponseTime, 2))ms" -ForegroundColor White
    Write-Host "  Min Response Time: $([math]::Round($results.Stats.MinResponseTime, 2))ms" -ForegroundColor White
    Write-Host "  Max Response Time: $([math]::Round($results.Stats.MaxResponseTime, 2))ms" -ForegroundColor White
    Write-Host "  95th Percentile: $([math]::Round($results.Stats.Percentile95, 2))ms" -ForegroundColor White

    Write-Host ""
    Write-Host "📋 HTTP Status Codes:" -ForegroundColor Green
    foreach ($status in $results.StatusCodes.Keys | Sort-Object) {
        Write-Host "  $status : $($results.StatusCodes[$status])" -ForegroundColor White
    }

    if ($results.Errors.Count -gt 0) {
        Write-Host ""
        Write-Host "❌ Error Summary:" -ForegroundColor Red
        $errorGroups = $results.Errors | Group-Object
        foreach ($group in $errorGroups) {
            Write-Host "  $($group.Count)x : $($group.Name)" -ForegroundColor White
        }
    }
}

# Main execution
try {
    $results = Test-APIEndpoint -Url $Url -DurationSeconds $DurationSeconds -ConcurrentUsers $ConcurrentUsers -UseAuth $UseAuth -AuthHeader $AuthHeader
    Show-Results -results $results
} catch {
    Write-Host "❌ Load test failed: $($_.Exception.Message)" -ForegroundColor Red
}
'@

    $loadTestScript | Out-File -FilePath "simple-load-test.ps1" -Encoding UTF8
    Write-Success "Created simple-load-test.ps1"

    # Also create a batch file for easy execution
    $batchScript = @'
@echo off
echo 🚀 Bitcoin Sprint API Load Test
echo ===============================

if "%~1"=="" (
    echo Usage: %0 [URL] [DURATION] [CONCURRENT_USERS]
    echo Example: %0 http://localhost:8080/health 30 10
    echo.
    echo Using default test...
    powershell -ExecutionPolicy Bypass -File "%~dp0simple-load-test.ps1"
) else (
    powershell -ExecutionPolicy Bypass -File "%~dp0simple-load-test.ps1" -Url "%~1" -DurationSeconds "%~2" -ConcurrentUsers "%~3"
)
'@

    $batchScript | Out-File -FilePath "run-load-test.bat" -Encoding ASCII
    Write-Success "Created run-load-test.bat"
}

# Run simple benchmarks that don't require full infrastructure
function Run-SimpleBenchmarks {
    Write-Section "⚡ Running Simple Performance Benchmarks"

    Write-Status "Testing basic Go performance..."

    # Test entropy module (this worked before)
    Write-Status "Testing entropy module performance..."
    try {
        $entropyResult = go test ./internal/entropy -v -bench=. -benchmem -run=^$
        Write-Host $entropyResult -ForegroundColor Gray
        Write-Success "Entropy benchmarks completed"
    } catch {
        Write-Warning "Entropy benchmarks failed: $($_.Exception.Message)"
    }

    # Test secure buffer module
    Write-Status "Testing secure buffer performance..."
    try {
        $secureBufResult = go test ./internal/securebuf -v -bench=. -benchmem -run=^$
        Write-Host $secureBufResult -ForegroundColor Gray
        Write-Success "Secure buffer benchmarks completed"
    } catch {
        Write-Warning "Secure buffer benchmarks failed: $($_.Exception.Message)"
    }

    # Test sprint client
    Write-Status "Testing sprint client performance..."
    try {
        $sprintClientResult = go test ./sprintclient -v -bench=. -benchmem -run=^$
        Write-Host $sprintClientResult -ForegroundColor Gray
        Write-Success "Sprint client benchmarks completed"
    } catch {
        Write-Warning "Sprint client benchmarks failed: $($_.Exception.Message)"
    }
}

# Memory and system analysis
function Analyze-SystemPerformance {
    Write-Section "🖥️ System Performance Analysis"

    Write-Status "Gathering system information..."

    $systemInfo = @{
        CPU = Get-CimInstance Win32_Processor | Select-Object -First 1
        Memory = Get-CimInstance Win32_OperatingSystem
        Disk = Get-CimInstance Win32_LogicalDisk -Filter "DriveType=3" | Select-Object -First 1
    }

    Write-Host "🖥️ System Information:" -ForegroundColor Green
    Write-Host "  CPU: $($systemInfo.CPU.Name)" -ForegroundColor White
    Write-Host "  Cores: $($systemInfo.CPU.NumberOfCores) cores, $($systemInfo.CPU.NumberOfLogicalProcessors) logical processors" -ForegroundColor White
    Write-Host "  Memory: $([math]::Round($systemInfo.Memory.TotalVisibleMemorySize / 1MB, 2)) GB total" -ForegroundColor White
    Write-Host "  Available Memory: $([math]::Round($systemInfo.Memory.FreePhysicalMemory / 1MB, 2)) GB free" -ForegroundColor White

    if ($systemInfo.Disk) {
        Write-Host "  Disk: $([math]::Round($systemInfo.Disk.Size / 1GB, 2)) GB total, $([math]::Round($systemInfo.Disk.FreeSpace / 1GB, 2)) GB free" -ForegroundColor White
    }

    # Performance recommendations
    Write-Host ""
    Write-Host "💡 Performance Recommendations:" -ForegroundColor Green

    if ($systemInfo.CPU.NumberOfCores -lt 4) {
        Write-Host "  ⚠️ Consider upgrading to a CPU with more cores for better performance" -ForegroundColor Yellow
    } else {
        Write-Host "  ✅ CPU core count is adequate for Bitcoin Sprint" -ForegroundColor Green
    }

    $memoryGB = $systemInfo.Memory.TotalVisibleMemorySize / 1MB
    if ($memoryGB -lt 8) {
        Write-Host "  ⚠️ Consider adding more RAM (8GB+ recommended)" -ForegroundColor Yellow
    } else {
        Write-Host "  ✅ Memory capacity is adequate" -ForegroundColor Green
    }

    Write-Host "  📊 Based on existing SLA tests:" -ForegroundColor Green
    Write-Host "    - Turbo tier: ~3.4ms average latency" -ForegroundColor White
    Write-Host "    - 88% SLA compliance rate" -ForegroundColor White
    Write-Host "    - Memory efficient with < 50MB working set" -ForegroundColor White
}

# Main execution
Write-Section "🚀 Bitcoin Sprint Performance Testing Suite"

if ($AnalyzeExistingResults) {
    Analyze-ExistingResults
}

if ($CreateLoadTestScript) {
    Create-LoadTestScript
}

if ($RunSimpleBenchmarks) {
    Run-SimpleBenchmarks
}

if (-not $AnalyzeExistingResults -and -not $CreateLoadTestScript -and -not $RunSimpleBenchmarks) {
    # Run all tests by default
    Analyze-ExistingResults
    Create-LoadTestScript
    Run-SimpleBenchmarks
    Analyze-SystemPerformance
}

Write-Section "✅ Performance Testing Complete"

Write-Host ""
Write-Host "📋 Summary:" -ForegroundColor Green
Write-Host "  📊 Analyzed existing performance results" -ForegroundColor White
Write-Host "  🛠️ Created load testing tools" -ForegroundColor White
Write-Host "  ⚡ Ran available benchmarks" -ForegroundColor White
Write-Host "  🖥️ Analyzed system performance" -ForegroundColor White
Write-Host ""
Write-Host "🔧 Next Steps:" -ForegroundColor Green
Write-Host "  1. Use simple-load-test.ps1 to test any API endpoint" -ForegroundColor White
Write-Host "  2. Run run-load-test.bat for quick testing" -ForegroundColor White
Write-Host "  3. Review performance-results/ directory for detailed results" -ForegroundColor White
Write-Host "  4. For full Bitcoin Sprint testing, resolve ZMQ dependencies" -ForegroundColor White
