#!/usr/bin/env pwsh

# Bitcoin Sprint Real Performance Test
# Tests actual running services for speed and reliability

param(
    [int]$ConcurrentRequests = 20,
    [int]$TestDuration = 60,
    [string]$ApiBaseUrl = "http://127.0.0.1:8080"
)

Write-Host "🚀 Bitcoin Sprint Real Performance Test" -ForegroundColor Cyan
Write-Host "=====================================" -ForegroundColor Cyan
Write-Host "Testing against: $ApiBaseUrl" -ForegroundColor Yellow
Write-Host ""

# Test endpoints
$endpoints = @(
    "/health",
    "/bitcoin/status",
    "/network/info",
    "/api/v1/enterprise/entropy/fast"
)

$allResults = @()
$startTime = Get-Date

Write-Host "🔥 Starting concurrent load test..." -ForegroundColor Yellow
Write-Host "=================================" -ForegroundColor Yellow

# Run tests for each endpoint
foreach ($endpoint in $endpoints) {
    Write-Host ""
    Write-Host "📡 Testing endpoint: $endpoint" -ForegroundColor Cyan

    $endpointResults = @()
    $endpointStart = Get-Date

    # Run concurrent requests
    $jobs = 1..$ConcurrentRequests | ForEach-Object -Parallel {
        $requestId = $_
        $url = $using:ApiBaseUrl + $using:endpoint
        $reqStart = Get-Date

        try {
            $response = Invoke-WebRequest -Uri $url -Method GET -TimeoutSec 10
            $reqEnd = Get-Date
            $latency = ($reqEnd - $reqStart).TotalMilliseconds

            [PSCustomObject]@{
                RequestId = $requestId
                Endpoint = $using:endpoint
                StatusCode = $response.StatusCode
                Latency = $latency
                Success = $true
                ContentLength = $response.Content.Length
                Timestamp = $reqStart
            }
        } catch {
            $reqEnd = Get-Date
            $latency = ($reqEnd - $reqStart).TotalMilliseconds

            [PSCustomObject]@{
                RequestId = $requestId
                Endpoint = $using:endpoint
                StatusCode = 0
                Latency = $latency
                Success = $false
                Error = $_.Exception.Message
                ContentLength = 0
                Timestamp = $reqStart
            }
        }
    }

    # Collect results
    foreach ($job in $jobs) {
        $endpointResults += $job
    }

    $endpointEnd = Get-Date
    $endpointDuration = ($endpointEnd - $endpointStart).TotalSeconds

    # Analyze results
    $successful = ($endpointResults | Where-Object { $_.Success }).Count
    $failed = $ConcurrentRequests - $successful
    $avgLatency = ($endpointResults | Where-Object { $_.Success } | Measure-Object -Property Latency -Average).Average
    $minLatency = ($endpointResults | Where-Object { $_.Success } | Measure-Object -Property Latency -Minimum).Minimum
    $maxLatency = ($endpointResults | Where-Object { $_.Success } | Measure-Object -Property Latency -Maximum).Maximum
    $p95Latency = ($endpointResults | Where-Object { $_.Success } | Sort-Object Latency)[$ConcurrentRequests * 0.95].Latency
    $throughput = $successful / $endpointDuration

    Write-Host "  ✅ Successful: $successful/$ConcurrentRequests" -ForegroundColor Green
    Write-Host ("  ❌ Failed: $failed") -ForegroundColor Red
    Write-Host ("  📊 Throughput: {0:N1} req/sec" -f $throughput) -ForegroundColor Green
    Write-Host ("  🕐 Avg Latency: {0:N1}ms" -f $avgLatency) -ForegroundColor Green
    Write-Host ("  📈 P95 Latency: {0:N1}ms" -f $p95Latency) -ForegroundColor Green
    Write-Host ("  📉 Min/Max: {0:N1}ms / {1:N1}ms" -f $minLatency, $maxLatency) -ForegroundColor Green

    $allResults += $endpointResults
}

$endTime = Get-Date
$totalDuration = ($endTime - $startTime).TotalSeconds

Write-Host ""
Write-Host "📈 Overall Performance Summary" -ForegroundColor Cyan
Write-Host "==============================" -ForegroundColor Cyan

$totalRequests = $allResults.Count
$successfulRequests = ($allResults | Where-Object { $_.Success }).Count
$successRate = ($successfulRequests / $totalRequests) * 100
$overallThroughput = $successfulRequests / $totalDuration
$overallAvgLatency = ($allResults | Where-Object { $_.Success } | Measure-Object -Property Latency -Average).Average

Write-Host ("🎯 Total Requests: {0}" -f $totalRequests) -ForegroundColor White
Write-Host ("✅ Success Rate: {0:P1}" -f ($successRate/100)) -ForegroundColor Green
Write-Host ("📊 Overall Throughput: {0:N1} req/sec" -f $overallThroughput) -ForegroundColor Green
Write-Host ("🕐 Overall Avg Latency: {0:N1}ms" -f $overallAvgLatency) -ForegroundColor Green
Write-Host ("⏱️  Test Duration: {0:N1} seconds" -f $totalDuration) -ForegroundColor White

# Test P2P handshake simulation (if Bitcoin node is available)
Write-Host ""
Write-Host "🔗 Testing P2P Handshake Simulation" -ForegroundColor Yellow
Write-Host "===================================" -ForegroundColor Yellow

try {
    $p2pResults = @()
    $p2pStart = Get-Date

    1..5 | ForEach-Object -Parallel {
        $handshakeId = $_
        $handshakeStart = Get-Date

        try {
            # Test basic TCP connection to Bitcoin port
            $tcpClient = New-Object System.Net.Sockets.TcpClient
            $connectResult = $tcpClient.BeginConnect("127.0.0.1", 8333, $null, $null)

            $waitHandle = $connectResult.AsyncWaitHandle
            if ($waitHandle.WaitOne(3000)) {  # 3 second timeout
                $tcpClient.EndConnect($connectResult)
                $handshakeEnd = Get-Date
                $handshakeTime = ($handshakeEnd - $handshakeStart).TotalMilliseconds

                if ($tcpClient.Connected) {
                    $tcpClient.Close()
                    [PSCustomObject]@{
                        HandshakeId = $handshakeId
                        Success = $true
                        Time = $handshakeTime
                        Status = "Connected"
                    }
                } else {
                    [PSCustomObject]@{
                        HandshakeId = $handshakeId
                        Success = $false
                        Time = 3000
                        Status = "Connection failed"
                    }
                }
            } else {
                [PSCustomObject]@{
                    HandshakeId = $handshakeId
                    Success = $false
                    Time = 3000
                    Status = "Timeout"
                }
            }
        } catch {
            [PSCustomObject]@{
                HandshakeId = $handshakeId
                Success = $false
                Time = 0
                Status = "Error: $($_.Exception.Message)"
            }
        }
    } | ForEach-Object { $p2pResults += $_ }

    $p2pEnd = Get-Date
    $p2pDuration = ($p2pEnd - $p2pStart).TotalSeconds

    $p2pSuccessful = ($p2pResults | Where-Object { $_.Success }).Count
    $p2pAvgTime = ($p2pResults | Where-Object { $_.Success } | Measure-Object -Property Time -Average).Average

    Write-Host "  🔗 P2P Connection Tests: $p2pSuccessful/5 successful" -ForegroundColor Green
    if ($p2pSuccessful -gt 0) {
        Write-Host ("  🕐 Average handshake time: {0:N1}ms" -f $p2pAvgTime) -ForegroundColor Green
    }
    Write-Host ("  📊 P2P test duration: {0:N1} seconds" -f $p2pDuration) -ForegroundColor Green

} catch {
    Write-Host "  ⚠️  P2P test failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Memory and CPU monitoring
Write-Host ""
Write-Host "🧠 System Resource Monitoring" -ForegroundColor Yellow
Write-Host "=============================" -ForegroundColor Yellow

$processes = Get-Process | Where-Object {
    $_.ProcessName -like "*bitcoin*" -or
    $_.ProcessName -like "*sprint*" -or
    $_.ProcessName -like "*cargo*" -or
    $_.ProcessName -like "*node*"
}

if ($processes) {
    Write-Host "Active Processes:" -ForegroundColor Green
    foreach ($process in $processes) {
        $memoryMB = [math]::Round($process.WorkingSet / 1MB, 1)
        Write-Host ("  📊 {0} (PID: {1}) - CPU: {2:N1}%, Memory: {3}MB" -f
            $process.ProcessName, $process.Id, $process.CPU, $memoryMB) -ForegroundColor Green
    }
} else {
    Write-Host "  ℹ️  No relevant processes found running" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "✅ Performance Test Complete!" -ForegroundColor Green
Write-Host "============================" -ForegroundColor Green
Write-Host ""
Write-Host "💡 Key Metrics:" -ForegroundColor Cyan
Write-Host ("   • Success Rate: {0:P1}" -f ($successRate/100)) -ForegroundColor Cyan
Write-Host ("   • Throughput: {0:N1} req/sec" -f $overallThroughput) -ForegroundColor Cyan
Write-Host ("   • Avg Latency: {0:N1}ms" -f $overallAvgLatency) -ForegroundColor Cyan
Write-Host ("   • P2P Handshakes: {0}/5 successful" -f $p2pSuccessful) -ForegroundColor Cyan

# Performance grade
$grade = if ($successRate -ge 99.9 -and $overallAvgLatency -le 100) { "A+ (Excellent)" }
         elseif ($successRate -ge 99 -and $overallAvgLatency -le 200) { "A (Very Good)" }
         elseif ($successRate -ge 95 -and $overallAvgLatency -le 500) { "B (Good)" }
         elseif ($successRate -ge 90) { "C (Fair)" }
         else { "D (Needs Improvement)" }

Write-Host ""
Write-Host "🏆 Performance Grade: $grade" -ForegroundColor Magenta
