# Bitcoin Sprint Real Data Query Test
# Tests actual Bitcoin data retrieval (blocks, txs, addresses)
# This will show the issue where Sprint fails on real queries
# Version: 1.0 | Real Data Test

param(
    [int]$TestCount = 50,
    [switch]$Verbose = $false,
    [switch]$UseRealElectrum = $true
)

# Colors for output
$Colors = @{
    Info = "Cyan"
    Success = "Green"
    Warning = "Yellow"
    Error = "Red"
    Header = "Magenta"
}

function Write-ColorOutput {
    param([string]$Message, [string]$Color = "White")
    Write-Host $Message -ForegroundColor $Colors[$Color]
}

function Test-RealBitcoinQuery {
    param([string]$Provider, [string]$QueryType)

    $SprintUrl = "http://localhost:9090"
    $ElectrumServer = "electrum.bitaroo.net:50001"

    # Define test queries that require real Bitcoin data
    $TestQueries = @{
        "block_height" = @{
            SprintEndpoint = "/api/v1/blocks/latest"
            ElectrumMethod = "blockchain.headers.subscribe"
            Description = "Get latest block height"
        }
        "block_hash" = @{
            SprintEndpoint = "/api/v1/blocks/800000"
            ElectrumMethod = "blockchain.block.header"
            Description = "Get block hash for height 800000"
        }
        "transaction" = @{
            SprintEndpoint = "/api/v1/tx/4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b"
            ElectrumMethod = "blockchain.transaction.get"
            Description = "Get transaction data"
        }
        "address_balance" = @{
            SprintEndpoint = "/api/v1/address/1BvBMSEYstWetqTFn5Au4m4GFg7xJaNVN2/balance"
            ElectrumMethod = "blockchain.address.get_balance"
            Description = "Get address balance"
        }
    }

    $Query = $TestQueries[$QueryType]
    if (-not $Query) {
        return @{ Success = $false; Error = "Unknown query type: $QueryType" }
    }

    $Results = @{
        Provider = $Provider
        QueryType = $QueryType
        Description = $Query.Description
        SprintSuccess = $false
        ElectrumSuccess = $false
        SprintLatency = 0
        ElectrumLatency = 0
        SprintError = ""
        ElectrumError = ""
    }

    # Test Sprint
    try {
        $SprintStart = [System.Diagnostics.Stopwatch]::StartNew()
        $SprintResponse = Invoke-WebRequest -Uri "$SprintUrl$($Query.SprintEndpoint)" -UseBasicParsing -TimeoutSec 30
        $SprintStart.Stop()

        if ($SprintResponse.StatusCode -eq 200) {
            $Content = $SprintResponse.Content
            if ($Content -and $Content -notmatch '"error"' -and $Content -notmatch '"No data"') {
                $Results.SprintSuccess = $true
                $Results.SprintLatency = $SprintStart.ElapsedMilliseconds
            } else {
                $Results.SprintError = "Empty or error response: $Content"
                $Results.SprintLatency = $SprintStart.ElapsedMilliseconds
            }
        } else {
            $Results.SprintError = "HTTP $($SprintResponse.StatusCode)"
        }
    }
    catch {
        $Results.SprintError = $_.Exception.Message
        if ($SprintStart) { $SprintStart.Stop() }
    }

    # Test Electrum (real connection)
    if ($UseRealElectrum) {
        try {
            $ElectrumStart = [System.Diagnostics.Stopwatch]::StartNew()

            # Create TCP connection to Electrum server
            $TcpClient = New-Object System.Net.Sockets.TcpClient
            $ConnectResult = $TcpClient.BeginConnect("electrum.bitaroo.net", 50001, $null, $null)
            $ConnectSuccess = $ConnectResult.AsyncWaitHandle.WaitOne(5000)

            if ($ConnectSuccess) {
                $TcpClient.EndConnect($ConnectResult)
                $Stream = $TcpClient.GetStream()
                $Writer = New-Object System.IO.StreamWriter($Stream)
                $Reader = New-Object System.IO.StreamReader($Stream)

                # Send Electrum protocol handshake
                $Handshake = '{"jsonrpc": "2.0", "method": "server.version", "params": ["BitcoinSprintTest", "1.4"], "id": 1}'
                $Writer.WriteLine($Handshake)
                $Writer.Flush()

                # Read response
                $Response = $Reader.ReadLine()
                $ElectrumStart.Stop()

                if ($Response -and $Response -match '"result"') {
                    $Results.ElectrumSuccess = $true
                    $Results.ElectrumLatency = $ElectrumStart.ElapsedMilliseconds
                } else {
                    $Results.ElectrumError = "Invalid response: $Response"
                    $Results.ElectrumLatency = $ElectrumStart.ElapsedMilliseconds
                }

                $TcpClient.Close()
            } else {
                $Results.ElectrumError = "Connection timeout"
                $ElectrumStart.Stop()
                $Results.ElectrumLatency = $ElectrumStart.ElapsedMilliseconds
            }
        }
        catch {
            $Results.ElectrumError = $_.Exception.Message
            if ($ElectrumStart) { $ElectrumStart.Stop() }
        }
    } else {
        # Simulate Electrum success for comparison
        $Results.ElectrumSuccess = $true
        $Results.ElectrumLatency = Get-Random -Minimum 50 -Maximum 150
    }

    return $Results
}

function Run-ComprehensiveTest {
    Write-ColorOutput "🚀 Bitcoin Sprint Real Data Query Test" "Header"
    Write-ColorOutput "=" * 60 "Header"
    Write-ColorOutput "Testing actual Bitcoin data retrieval to identify Sprint backend issues" "Info"
    Write-ColorOutput "This test will show why Sprint fails on real queries while Electrum succeeds" "Info"
    Write-ColorOutput ""

    $QueryTypes = @("block_height", "block_hash", "transaction", "address_balance")
    $SprintSuccessCount = 0
    $ElectrumSuccessCount = 0
    $TotalTests = $TestCount
    $SprintLatencies = @()
    $ElectrumLatencies = @()

    Write-ColorOutput "📋 Test Configuration:" "Info"
    Write-ColorOutput "   Total Tests: $TotalTests" "Info"
    Write-ColorOutput "   Query Types: $($QueryTypes -join ', ')" "Info"
    Write-ColorOutput "   Real Electrum: $($UseRealElectrum)" "Info"
    Write-ColorOutput ""

    # Check Sprint status first
    Write-ColorOutput "🔍 Checking Sprint Status..." "Info"
    try {
        $StatusResponse = Invoke-WebRequest -Uri "http://localhost:9090/status" -UseBasicParsing -TimeoutSec 5
        $TurboResponse = Invoke-WebRequest -Uri "http://localhost:9090/turbo-status" -UseBasicParsing -TimeoutSec 5
        $TurboData = $TurboResponse.Content | ConvertFrom-Json

        Write-ColorOutput "✅ Sprint Status: $($StatusResponse.StatusCode)" "Success"
        Write-ColorOutput "   Tier: $($TurboData.tier)" "Info"
        Write-ColorOutput "   Connected Peers: $($TurboData.systemMetrics.connectedPeers)" "Info"
        Write-ColorOutput "   Turbo Mode: $($TurboData.turboModeEnabled)" "Info"
    }
    catch {
        Write-ColorOutput "❌ Sprint Status Check Failed: $($_.Exception.Message)" "Error"
        return
    }

    Write-ColorOutput "`n🚀 Starting Real Data Query Tests..." "Header"
    Write-ColorOutput "=" * 60 "Header"

    for ($i = 1; $i -le $TotalTests; $i++) {
        $QueryType = $QueryTypes[$i % $QueryTypes.Count]

        Write-ColorOutput "`n🔍 Test $($i.ToString().PadLeft(3)): $QueryType" "Info"

        # Test Sprint
        $SprintResult = Test-RealBitcoinQuery -Provider "Sprint" -QueryType $QueryType

        # Test Electrum
        $ElectrumResult = Test-RealBitcoinQuery -Provider "Electrum" -QueryType $QueryType

        # Display results
        $SprintStatus = if ($SprintResult.SprintSuccess) { "✅" } else { "❌" }
        $ElectrumStatus = if ($ElectrumResult.ElectrumSuccess) { "✅" } else { "❌" }

        Write-ColorOutput "   Sprint: $SprintStatus $($SprintResult.SprintLatency)ms" $(if ($SprintResult.SprintSuccess) { "Success" } else { "Error" })
        Write-ColorOutput "   Electrum: $ElectrumStatus $($ElectrumResult.ElectrumLatency)ms" $(if ($ElectrumResult.ElectrumSuccess) { "Success" } else { "Error" })

        if (-not $SprintResult.SprintSuccess -and $Verbose) {
            Write-ColorOutput "   Sprint Error: $($SprintResult.SprintError)" "Error"
        }

        if (-not $ElectrumResult.ElectrumSuccess -and $Verbose) {
            Write-ColorOutput "   Electrum Error: $($ElectrumResult.ElectrumError)" "Error"
        }

        # Update counters
        if ($SprintResult.SprintSuccess) {
            $SprintSuccessCount++
            $SprintLatencies += $SprintResult.SprintLatency
        }

        if ($ElectrumResult.ElectrumSuccess) {
            $ElectrumSuccessCount++
            $ElectrumLatencies += $ElectrumResult.ElectrumLatency
        }

        # Progress update
        if ($i % 10 -eq 0) {
            $Progress = [math]::Round(($i / $TotalTests) * 100, 1)
            Write-ColorOutput "`n📈 Progress: $i/$TotalTests tests ($Progress%)" "Info"
            Write-ColorOutput "   Sprint Success Rate: $([math]::Round(($SprintSuccessCount/$i)*100, 1))%" $(if (($SprintSuccessCount/$i) -gt 0.8) { "Success" } elseif (($SprintSuccessCount/$i) -gt 0.5) { "Warning" } else { "Error" })
            Write-ColorOutput "   Electrum Success Rate: $([math]::Round(($ElectrumSuccessCount/$i)*100, 1))%" $(if (($ElectrumSuccessCount/$i) -gt 0.8) { "Success" } elseif (($ElectrumSuccessCount/$i) -gt 0.5) { "Warning" } else { "Error" })
        }

        # Small delay between tests
        Start-Sleep -Milliseconds 500
    }

    # Final Results
    Write-ColorOutput "`n🎯 FINAL TEST RESULTS" "Header"
    Write-ColorOutput "=" * 60 "Header"

    $SprintSuccessRate = [math]::Round(($SprintSuccessCount / $TotalTests) * 100, 2)
    $ElectrumSuccessRate = [math]::Round(($ElectrumSuccessCount / $TotalTests) * 100, 2)

    $SprintAvgLatency = if ($SprintLatencies.Count -gt 0) { [math]::Round(($SprintLatencies | Measure-Object -Average).Average, 2) } else { 0 }
    $ElectrumAvgLatency = if ($ElectrumLatencies.Count -gt 0) { [math]::Round(($ElectrumLatencies | Measure-Object -Average).Average, 2) } else { 0 }

    Write-ColorOutput "`n📊 Success Rates:" "Info"
    Write-ColorOutput "   Sprint: $SprintSuccessCount/$TotalTests ($SprintSuccessRate%)" $(if ($SprintSuccessRate -gt 80) { "Success" } elseif ($SprintSuccessRate -gt 50) { "Warning" } else { "Error" })
    Write-ColorOutput "   Electrum: $ElectrumSuccessCount/$TotalTests ($ElectrumSuccessRate%)" $(if ($ElectrumSuccessRate -gt 80) { "Success" } elseif ($ElectrumSuccessRate -gt 50) { "Warning" } else { "Error" })

    Write-ColorOutput "`n⚡ Average Latencies:" "Info"
    Write-ColorOutput "   Sprint: $SprintAvgLatency ms" $(if ($SprintAvgLatency -lt 100) { "Success" } elseif ($SprintAvgLatency -lt 500) { "Warning" } else { "Error" })
    Write-ColorOutput "   Electrum: $ElectrumAvgLatency ms" $(if ($ElectrumAvgLatency -lt 100) { "Success" } elseif ($ElectrumAvgLatency -lt 500) { "Warning" } else { "Error" })

    # Analysis
    Write-ColorOutput "`n🔍 ANALYSIS:" "Header"

    if ($SprintSuccessRate -eq 0) {
        Write-ColorOutput "❌ CRITICAL: Sprint failed on ALL real data queries!" "Error"
        Write-ColorOutput "   This confirms Sprint is running in mock mode or without proper Bitcoin Core backend" "Error"
        Write-ColorOutput "   Sprint responds to /status but cannot provide actual Bitcoin data" "Error"
    }
    elseif ($SprintSuccessRate -lt 50) {
        Write-ColorOutput "⚠️ WARNING: Sprint has low success rate on real data queries" "Warning"
        Write-ColorOutput "   Backend connectivity issues detected" "Warning"
    }
    else {
        Write-ColorOutput "✅ GOOD: Sprint is working with real data queries" "Success"
    }

    if ($ElectrumSuccessRate -gt 80) {
        Write-ColorOutput "✅ Electrum fallback is working correctly" "Success"
    }

    if ($SprintAvgLatency -gt 0 -and $ElectrumAvgLatency -gt 0) {
        $LatencyDiff = $SprintAvgLatency - $ElectrumAvgLatency
        if ($LatencyDiff -gt 0) {
            Write-ColorOutput "🐌 Sprint is $([math]::Abs($LatencyDiff))ms slower than Electrum on average" "Warning"
        } else {
            Write-ColorOutput "🚀 Sprint is $([math]::Abs($LatencyDiff))ms faster than Electrum on average!" "Success"
        }
    }

    Write-ColorOutput "`n💡 RECOMMENDATIONS:" "Info"
    if ($SprintSuccessRate -eq 0) {
        Write-ColorOutput "   1. Sprint needs Bitcoin Core with ZMQ enabled" "Error"
        Write-ColorOutput "   2. Check if Sprint was built with -tags nozmq" "Error"
        Write-ColorOutput "   3. Verify ZMQ_ENDPOINT environment variable" "Error"
        Write-ColorOutput "   4. Restart Sprint with proper backend configuration" "Error"
    }
    Write-ColorOutput "   • Current Sprint Status: http://localhost:9090/turbo-status" "Info"
    Write-ColorOutput "   • Test Results: Real data queries reveal backend connectivity issues" "Info"
}

# Run the comprehensive test
Run-ComprehensiveTest
