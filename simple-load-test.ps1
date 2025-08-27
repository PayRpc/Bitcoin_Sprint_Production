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
