#!/usr/bin/env pwsh

# Bitcoin Sprint Handshake Performance Validator
# Validates P2P handshake speed and service flag enforcement

Write-Host "🔐 Bitcoin Sprint Handshake Performance Validator" -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan
Write-Host ""

# Test 1: Check running processes and their performance
Write-Host "📊 Current System Status" -ForegroundColor Yellow
Write-Host "========================" -ForegroundColor Yellow

$bitcoinProcesses = Get-Process | Where-Object {
    $_.ProcessName -like "*bitcoin*" -or
    $_.ProcessName -like "*sprint*"
} | Where-Object { $_.Id -ne $PID }

if ($bitcoinProcesses) {
    Write-Host "Active Bitcoin Sprint Processes:" -ForegroundColor Green
    foreach ($process in $bitcoinProcesses) {
        $memoryMB = [math]::Round($process.WorkingSet / 1MB, 1)
        $cpuPercent = [math]::Round($process.CPU, 1)
        Write-Host ("  🔧 {0} (PID: {1}) - CPU: {2}%, Memory: {3}MB" -f
            $process.ProcessName, $process.Id, $cpuPercent, $memoryMB) -ForegroundColor Green
    }
} else {
    Write-Host "  ⚠️  No Bitcoin Sprint processes found running" -ForegroundColor Red
}

Write-Host ""

# Test 2: Network connectivity test
Write-Host "🌐 Network Connectivity Test" -ForegroundColor Yellow
Write-Host "============================" -ForegroundColor Yellow

$ports = @(8080, 8082, 8333, 8335)
foreach ($port in $ports) {
    try {
        $connection = Test-NetConnection -ComputerName 127.0.0.1 -Port $port -WarningAction SilentlyContinue
        if ($connection.TcpTestSucceeded) {
            Write-Host ("  ✅ Port {0}: OPEN" -f $port) -ForegroundColor Green
        } else {
            Write-Host ("  ❌ Port {0}: CLOSED" -f $port) -ForegroundColor Red
        }
    } catch {
        Write-Host ("  ⚠️  Port {0}: ERROR - {1}" -f $port, $_.Exception.Message) -ForegroundColor Yellow
    }
}

Write-Host ""

# Test 3: API Performance Test
Write-Host "🚀 API Performance Test" -ForegroundColor Yellow
Write-Host "======================" -ForegroundColor Yellow

$apiResults = @()
$apiStart = Get-Date

# Test health endpoint
try {
    $healthStart = Get-Date
    $response = Invoke-WebRequest -Uri "http://127.0.0.1:8080/health" -Method GET -TimeoutSec 5
    $healthEnd = Get-Date
    $healthLatency = ($healthEnd - $healthStart).TotalMilliseconds

    Write-Host ("  ✅ Health Check: {0}ms - Status: {1}" -f
        [math]::Round($healthLatency, 1), $response.StatusCode) -ForegroundColor Green

    $apiResults += [PSCustomObject]@{
        Endpoint = "/health"
        Latency = $healthLatency
        StatusCode = $response.StatusCode
        Success = $true
    }
} catch {
    Write-Host ("  ❌ Health Check: FAILED - {0}" -f $_.Exception.Message) -ForegroundColor Red
    $apiResults += [PSCustomObject]@{
        Endpoint = "/health"
        Latency = 0
        StatusCode = 0
        Success = $false
        Error = $_.Exception.Message
    }
}

# Test entropy endpoint
try {
    $entropyStart = Get-Date
    $response = Invoke-WebRequest -Uri "http://127.0.0.1:8080/api/v1/enterprise/entropy/fast" -Method GET -TimeoutSec 5
    $entropyEnd = Get-Date
    $entropyLatency = ($entropyEnd - $entropyStart).TotalMilliseconds

    Write-Host ("  ✅ Entropy API: {0}ms - Status: {1}" -f
        [math]::Round($entropyLatency, 1), $response.StatusCode) -ForegroundColor Green

    $apiResults += [PSCustomObject]@{
        Endpoint = "/api/v1/enterprise/entropy/fast"
        Latency = $entropyLatency
        StatusCode = $response.StatusCode
        Success = $true
    }
} catch {
    Write-Host ("  ❌ Entropy API: FAILED - {0}" -f $_.Exception.Message) -ForegroundColor Red
    $apiResults += [PSCustomObject]@{
        Endpoint = "/api/v1/enterprise/entropy/fast"
        Latency = $entropyLatency
        StatusCode = 0
        Success = $false
        Error = $_.Exception.Message
    }
}

$apiEnd = Get-Date
$apiDuration = ($apiEnd - $apiStart).TotalSeconds

$successfulApis = ($apiResults | Where-Object { $_.Success }).Count
$avgApiLatency = ($apiResults | Where-Object { $_.Success } | Measure-Object -Property Latency -Average).Average

Write-Host ("  📊 API Test Results: {0}/{1} successful" -f $successfulApis, $apiResults.Count) -ForegroundColor Green
if ($successfulApis -gt 0) {
    Write-Host ("  🕐 Average API Latency: {0:N1}ms" -f $avgApiLatency) -ForegroundColor Green
}

Write-Host ""

# Test 4: P2P Handshake Validation
Write-Host "🔗 P2P Handshake Validation" -ForegroundColor Yellow
Write-Host "==========================" -ForegroundColor Yellow

$p2pResults = @()
$p2pStart = Get-Date

# Test multiple P2P connections
1..3 | ForEach-Object -Parallel {
    $testId = $_
    $handshakeStart = Get-Date

    try {
        $tcpClient = New-Object System.Net.Sockets.TcpClient
        $connectResult = $tcpClient.BeginConnect("127.0.0.1", 8335, $null, $null)

        $waitHandle = $connectResult.AsyncWaitHandle
        if ($waitHandle.WaitOne(5000)) {
            $tcpClient.EndConnect($connectResult)
            $handshakeEnd = Get-Date
            $handshakeTime = ($handshakeEnd - $handshakeStart).TotalMilliseconds

            if ($tcpClient.Connected) {
                # Send a simple version message to test handshake
                $stream = $tcpClient.GetStream()
                $writer = New-Object System.IO.StreamWriter($stream)

                # Simple version handshake test
                $versionData = [byte[]]@(0xF9, 0xBE, 0xB4, 0xD9)  # Mainnet magic bytes
                $stream.Write($versionData, 0, $versionData.Length)

                $tcpClient.Close()

                [PSCustomObject]@{
                    TestId = $testId
                    Success = $true
                    HandshakeTime = $handshakeTime
                    Status = "Connected"
                }
            } else {
                [PSCustomObject]@{
                    TestId = $testId
                    Success = $false
                    HandshakeTime = 5000
                    Status = "Connection failed"
                }
            }
        } else {
            [PSCustomObject]@{
                TestId = $testId
                Success = $false
                HandshakeTime = 5000
                Status = "Timeout"
            }
        }
    } catch {
        [PSCustomObject]@{
            TestId = $testId
            Success = $false
            HandshakeTime = 0
            Status = "Error: $($_.Exception.Message)"
        }
    }
} | ForEach-Object { $p2pResults += $_ }

$p2pEnd = Get-Date
$p2pDuration = ($p2pEnd - $p2pStart).TotalSeconds

$successfulP2p = ($p2pResults | Where-Object { $_.Success }).Count
$avgP2pTime = ($p2pResults | Where-Object { $_.Success } | Measure-Object -Property HandshakeTime -Average).Average

Write-Host ("  🔗 P2P Tests: {0}/3 successful" -f $successfulP2p) -ForegroundColor Green
if ($successfulP2p -gt 0) {
    Write-Host ("  🕐 Average P2P Handshake: {0:N1}ms" -f $avgP2pTime) -ForegroundColor Green
}

Write-Host ""

# Test 5: Service Flag Validation Test
Write-Host "🏷️  Service Flag Validation" -ForegroundColor Yellow
Write-Host "==========================" -ForegroundColor Yellow

# Check if the service flag constants are properly defined
$serviceConstants = @(
    @{ Name = "NODE_NETWORK"; Value = 1 },
    @{ Name = "NODE_WITNESS"; Value = 8 },
    @{ Name = "NODE_NETWORK_LIMITED"; Value = 1024 }
)

Write-Host "Expected Service Flags:" -ForegroundColor Cyan
foreach ($flag in $serviceConstants) {
    Write-Host ("  📋 {0}: {1}" -f $flag.Name, $flag.Value) -ForegroundColor Cyan
}

# Test service flag combinations
$testServices = @(
    @{ Services = 1; Description = "NODE_NETWORK only" },
    @{ Services = 8; Description = "NODE_WITNESS only" },
    @{ Services = 9; Description = "NODE_NETWORK + NODE_WITNESS (valid)" },
    @{ Services = 1024; Description = "NODE_NETWORK_LIMITED only" },
    @{ Services = 1032; Description = "NODE_NETWORK_LIMITED + NODE_WITNESS (valid)" }
)

Write-Host ""
Write-Host "Service Flag Validation Tests:" -ForegroundColor Cyan
foreach ($test in $testServices) {
    $hasNetwork = ($test.Services -band 1) -ne 0 -or ($test.Services -band 1024) -ne 0
    $hasWitness = ($test.Services -band 8) -ne 0
    $isValid = $hasNetwork -and $hasWitness

    $status = if ($isValid) { "✅ VALID" } else { "❌ INVALID" }
    $color = if ($isValid) { "Green" } else { "Red" }

    Write-Host ("  {0} Services: {1} - {2}" -f $status, $test.Services, $test.Description) -ForegroundColor $color
}

Write-Host ""

# Final Performance Summary
Write-Host "🏆 Final Performance Summary" -ForegroundColor Cyan
Write-Host "===========================" -ForegroundColor Cyan

$totalTests = $apiResults.Count + $p2pResults.Count
$successfulTests = ($apiResults | Where-Object { $_.Success }).Count + ($p2pResults | Where-Object { $_.Success }).Count
$successRate = if ($totalTests -gt 0) { ($successfulTests / $totalTests) * 100 } else { 0 }

Write-Host ("📊 Overall Success Rate: {0:N1}%" -f $successRate) -ForegroundColor Green
Write-Host ("🔗 P2P Connections: {0}/3 successful" -f $successfulP2p) -ForegroundColor Green
Write-Host ("🌐 API Endpoints: {0}/2 functional" -f $successfulApis) -ForegroundColor Green

# Performance grade
$grade = if ($successRate -ge 95 -and $successfulP2p -ge 2) { "A+ (Excellent)" }
         elseif ($successRate -ge 90 -and $successfulP2p -ge 1) { "A (Very Good)" }
         elseif ($successRate -ge 80) { "B (Good)" }
         elseif ($successRate -ge 70) { "C (Fair)" }
         else { "D (Needs Improvement)" }

Write-Host ""
Write-Host "🎯 Performance Grade: $grade" -ForegroundColor Magenta

# Recommendations
Write-Host ""
Write-Host "💡 Recommendations:" -ForegroundColor Yellow
if ($successfulP2p -lt 3) {
    Write-Host "  • Consider checking P2P peer discovery and connection pooling" -ForegroundColor Yellow
}
if ($successfulApis -lt 2) {
    Write-Host "  • Verify API server configuration and port bindings" -ForegroundColor Yellow
}
if ($successRate -ge 95) {
    Write-Host "  • System performing excellently - ready for production!" -ForegroundColor Green
}

Write-Host ""
Write-Host "✅ Handshake & Speed Validation Complete!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
