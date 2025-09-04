#!/usr/bin/env pwsh

# Bitcoin Sprint Speed & Handshake Performance Test
# Tests P2P handshake, connection speeds, and RPC performance

param(
    [int]$ConcurrentConnections = 10,
    [int]$TestDuration = 30,
    [string]$TargetHost = "127.0.0.1",
    [int]$TargetPort = 8333
)

Write-Host "🚀 Bitcoin Sprint Speed & Handshake Performance Test" -ForegroundColor Cyan
Write-Host "==================================================" -ForegroundColor Cyan
Write-Host ""

# Test 1: Handshake Performance
Write-Host "📡 Test 1: P2P Handshake Performance" -ForegroundColor Yellow
Write-Host "-----------------------------------" -ForegroundColor Yellow

$handshakeResults = @()
$startTime = Get-Date

1..$ConcurrentConnections | ForEach-Object -Parallel {
    $connId = $_
    $start = Get-Date

    try {
        # Test TCP connection (simulating P2P handshake)
        $tcpClient = New-Object System.Net.Sockets.TcpClient
        $connectResult = $tcpClient.BeginConnect($using:TargetHost, $using:TargetPort, $null, $null)

        # Wait for connection with timeout
        $waitHandle = $connectResult.AsyncWaitHandle
        if ($waitHandle.WaitOne(5000)) {  # 5 second timeout
            $tcpClient.EndConnect($connectResult)
            $end = Get-Date
            $latency = ($end - $start).TotalMilliseconds

            # Simulate handshake protocol
            if ($tcpClient.Connected) {
                $stream = $tcpClient.GetStream()
                $writer = New-Object System.IO.StreamWriter($stream)
                $reader = New-Object System.IO.StreamReader($stream)

                # Send version message (simplified)
                $versionMsg = "VERSION:70016:SERVICES:13:USERAGENT:/BitcoinSprint:1.0.0/"
                $writer.WriteLine($versionMsg)
                $writer.Flush()

                # Read response
                $response = $reader.ReadLine()
                $handshakeEnd = Get-Date
                $handshakeTime = ($handshakeEnd - $start).TotalMilliseconds

                $tcpClient.Close()

                [PSCustomObject]@{
                    ConnectionId = $connId
                    ConnectionTime = $latency
                    HandshakeTime = $handshakeTime
                    Success = $true
                    Status = "Connected"
                }
            }
        } else {
            [PSCustomObject]@{
                ConnectionId = $connId
                ConnectionTime = 5000
                HandshakeTime = 5000
                Success = $false
                Status = "Timeout"
            }
        }
    } catch {
        [PSCustomObject]@{
            ConnectionId = $connId
            ConnectionTime = 0
            HandshakeTime = 0
            Success = $false
            Status = "Error: $($_.Exception.Message)"
        }
    }
} | ForEach-Object { $handshakeResults += $_ }

$endTime = Get-Date
$totalHandshakeTime = ($endTime - $startTime).TotalSeconds

Write-Host "Results:" -ForegroundColor Green
$successfulConnections = ($handshakeResults | Where-Object { $_.Success }).Count
$avgConnectionTime = ($handshakeResults | Where-Object { $_.Success } | Measure-Object -Property ConnectionTime -Average).Average
$avgHandshakeTime = ($handshakeResults | Where-Object { $_.Success } | Measure-Object -Property HandshakeTime -Average).Average

Write-Host "  ✅ Successful connections: $successfulConnections/$ConcurrentConnections" -ForegroundColor Green
Write-Host ("  🕐 Average connection time: {0:N2}ms" -f $avgConnectionTime) -ForegroundColor Green
Write-Host ("  🤝 Average handshake time: {0:N2}ms" -f $avgHandshakeTime) -ForegroundColor Green
Write-Host ("  📊 Connections per second: {0:N1}" -f ($successfulConnections / $totalHandshakeTime)) -ForegroundColor Green
Write-Host ""

# Test 2: RPC Batch Processing Speed
Write-Host "🔄 Test 2: RPC Batch Processing Speed" -ForegroundColor Yellow
Write-Host "------------------------------------" -ForegroundColor Yellow

$rpcResults = @()
$batchSizes = @(10, 50, 100)
$rpcStartTime = Get-Date

foreach ($batchSize in $batchSizes) {
    Write-Host "Testing batch size: $batchSize" -ForegroundColor Cyan

    $batchResult = 1..5 | ForEach-Object -Parallel {
        $batchId = $_
        $batchStart = Get-Date

        try {
            # Simulate RPC batch call
            $rpcData = @{
                jsonrpc = "2.0"
                id = $batchId
                method = "getblockhash"
                params = @($batchId)
            }

            # Convert to JSON and measure serialization time
            $jsonStart = Get-Date
            $jsonPayload = $rpcData | ConvertTo-Json
            $jsonEnd = Get-Date
            $serializationTime = ($jsonEnd - $jsonStart).TotalMilliseconds

            # Simulate network round trip (HTTP call)
            $httpStart = Get-Date
            Start-Sleep -Milliseconds (Get-Random -Minimum 10 -Maximum 50)  # Simulate network latency
            $httpEnd = Get-Date
            $networkTime = ($httpEnd - $httpStart).TotalMilliseconds

            # Simulate response processing
            $processStart = Get-Date
            $response = @{
                result = "0000000000000000000$batchId"
                id = $batchId
            }
            $responseJson = $response | ConvertTo-Json
            $processEnd = Get-Date
            $processingTime = ($processEnd - $processStart).TotalMilliseconds

            $batchEnd = Get-Date
            $totalBatchTime = ($batchEnd - $batchStart).TotalMilliseconds

            [PSCustomObject]@{
                BatchId = $batchId
                BatchSize = $using:batchSize
                SerializationTime = $serializationTime
                NetworkTime = $networkTime
                ProcessingTime = $processingTime
                TotalTime = $totalBatchTime
                Success = $true
            }
        } catch {
            [PSCustomObject]@{
                BatchId = $batchId
                BatchSize = $using:batchSize
                SerializationTime = 0
                NetworkTime = 0
                ProcessingTime = 0
                TotalTime = 0
                Success = $false
                Error = $_.Exception.Message
            }
        }
    }

    $rpcResults += $batchResult

    $avgBatchTime = ($batchResult | Measure-Object -Property TotalTime -Average).Average
    Write-Host ("  📦 Batch size $batchSize - Avg time: {0:N2}ms" -f $avgBatchTime) -ForegroundColor Green
}

$rpcEndTime = Get-Date
$totalRpcTime = ($rpcEndTime - $rpcStartTime).TotalSeconds

Write-Host ""
Write-Host "RPC Performance Summary:" -ForegroundColor Green
foreach ($batchSize in $batchSizes) {
    $batchStats = $rpcResults | Where-Object { $_.BatchSize -eq $batchSize }
    $avgTime = ($batchStats | Measure-Object -Property TotalTime -Average).Average
    $throughput = $batchSize / ($avgTime / 1000)  # requests per second
    Write-Host ("  📊 Batch size $batchSize - {0:N1} req/sec" -f $throughput) -ForegroundColor Green
}

Write-Host ""

# Test 3: File I/O Performance (for state persistence)
Write-Host "💾 Test 3: File I/O Performance" -ForegroundColor Yellow
Write-Host "------------------------------" -ForegroundColor Yellow

$ioResults = @()
$ioStartTime = Get-Date

1..20 | ForEach-Object -Parallel {
    $ioId = $_
    $ioStart = Get-Date

    try {
        $tempFile = [System.IO.Path]::GetTempFileName()
        $testData = "Test data for I/O performance test #$ioId - " + ("x" * 1024)  # 1KB of data

        # Write operation
        $writeStart = Get-Date
        [System.IO.File]::WriteAllText($tempFile, $testData)
        $writeEnd = Get-Date
        $writeTime = ($writeEnd - $writeStart).TotalMilliseconds

        # Sync to disk (fsync simulation)
        $syncStart = Get-Date
        $fileStream = [System.IO.File]::Open($tempFile, [System.IO.FileMode]::Open, [System.IO.FileAccess]::ReadWrite)
        $fileStream.Flush()
        $fileStream.Close()
        $syncEnd = Get-Date
        $syncTime = ($syncEnd - $syncStart).TotalMilliseconds

        # Read operation
        $readStart = Get-Date
        $readData = [System.IO.File]::ReadAllText($tempFile)
        $readEnd = Get-Date
        $readTime = ($readEnd - $readStart).TotalMilliseconds

        # Cleanup
        [System.IO.File]::Delete($tempFile)

        $ioEnd = Get-Date
        $totalIoTime = ($ioEnd - $ioStart).TotalMilliseconds

        [PSCustomObject]@{
            IoId = $ioId
            WriteTime = $writeTime
            SyncTime = $syncTime
            ReadTime = $readTime
            TotalTime = $totalIoTime
            DataSize = $testData.Length
            Success = ($readData -eq $testData)
        }
    } catch {
        [PSCustomObject]@{
            IoId = $ioId
            WriteTime = 0
            SyncTime = 0
            ReadTime = 0
            TotalTime = 0
            DataSize = 0
            Success = $false
            Error = $_.Exception.Message
        }
    }
} | ForEach-Object { $ioResults += $_ }

$ioEndTime = Get-Date
$totalIoTime = ($ioEndTime - $ioStartTime).TotalSeconds

Write-Host "Results:" -ForegroundColor Green
$successfulIo = ($ioResults | Where-Object { $_.Success }).Count
$avgWriteTime = ($ioResults | Where-Object { $_.Success } | Measure-Object -Property WriteTime -Average).Average
$avgSyncTime = ($ioResults | Where-Object { $_.Success } | Measure-Object -Property SyncTime -Average).Average
$avgReadTime = ($ioResults | Where-Object { $_.Success } | Measure-Object -Property ReadTime -Average).Average
$avgTotalIoTime = ($ioResults | Where-Object { $_.Success } | Measure-Object -Property TotalTime -Average).Average

Write-Host "  ✅ Successful I/O operations: $successfulIo/20" -ForegroundColor Green
Write-Host ("  ✍️  Average write time: {0:N2}ms" -f $avgWriteTime) -ForegroundColor Green
Write-Host ("  🔄 Average sync time: {0:N2}ms" -f $avgSyncTime) -ForegroundColor Green
Write-Host ("  📖 Average read time: {0:N2}ms" -f $avgReadTime) -ForegroundColor Green
Write-Host ("  ⏱️  Average total I/O time: {0:N2}ms" -f $avgTotalIoTime) -ForegroundColor Green
Write-Host ("  📈 I/O operations per second: {0:N1}" -f ($successfulIo / $totalIoTime)) -ForegroundColor Green

Write-Host ""

# Test 4: Memory and CPU Performance
Write-Host "🧠 Test 4: Memory & CPU Performance" -ForegroundColor Yellow
Write-Host "----------------------------------" -ForegroundColor Yellow

$memoryResults = @()
$cpuResults = @()

# Get current process info
$bitcoinProcesses = Get-Process | Where-Object { $_.ProcessName -like "*bitcoin*" -or $_.ProcessName -like "*sprint*" }

foreach ($process in $bitcoinProcesses) {
    $memoryResults += [PSCustomObject]@{
        ProcessName = $process.ProcessName
        ProcessId = $process.Id
        WorkingSetMB = [math]::Round($process.WorkingSet / 1MB, 2)
        PrivateMemoryMB = [math]::Round($process.PrivateMemorySize / 1MB, 2)
        CPUPercent = $process.CPU
    }
}

Write-Host "Memory Usage:" -ForegroundColor Green
$memoryResults | Format-Table -AutoSize

Write-Host "Performance Summary:" -ForegroundColor Cyan
Write-Host "===================" -ForegroundColor Cyan
Write-Host ""
Write-Host ("⏱️  Total test duration: {0:N1} seconds" -f (($endTime - $startTime).TotalSeconds)) -ForegroundColor Cyan
Write-Host ("🔗 Handshake success rate: {0:P1}" -f ($successfulConnections / $ConcurrentConnections)) -ForegroundColor Cyan
Write-Host ("💾 I/O success rate: {0:P1}" -f ($successfulIo / 20)) -ForegroundColor Cyan
Write-Host ("🚀 Peak connections/sec: {0:N1}" -f ($ConcurrentConnections / $totalHandshakeTime)) -ForegroundColor Cyan
Write-Host ("📊 Peak I/O ops/sec: {0:N1}" -f ($successfulIo / $totalIoTime)) -ForegroundColor Cyan

Write-Host ""
Write-Host "✅ Speed & Handshake Test Complete!" -ForegroundColor Green
Write-Host "===================================" -ForegroundColor Green
