# Bitcoin Core Continuous Monitoring Test - 5 Minutes
# Real-time monitoring of Bitcoin Core health and performance

Write-Host "🔍 BITCOIN CORE CONTINUOUS MONITORING TEST" -ForegroundColor Cyan
Write-Host "==========================================="
Write-Host "Duration: 5 minutes"
Write-Host "Monitoring: Block height, connections, sync status, network info"
Write-Host ""

# Configuration
$testDuration = 5 * 60  # 5 minutes in seconds
$checkInterval = 10     # Check every 10 seconds
$bitcoinRpcUrl = "http://localhost:8332"
$rpcUser = "sprint"
$rpcPassword = "1cba94f0a8b1"

# Initialize monitoring data
$startTime = Get-Date
$monitoringData = @()
$checkCount = 0

Write-Host "📊 STARTING MONITORING SESSION" -ForegroundColor Green
Write-Host "================================"
Write-Host "Start Time: $($startTime.ToString('yyyy-MM-dd HH:mm:ss'))" -ForegroundColor Green
Write-Host "Duration: 5 minutes" -ForegroundColor Green
Write-Host "Check Interval: $checkInterval seconds" -ForegroundColor Green
Write-Host ""

# Function to check Bitcoin Core status
function Get-BitcoinCoreStatus {
    try {
        # Create authorization header
        $auth = [Convert]::ToBase64String([Text.Encoding]::ASCII.GetBytes("$rpcUser`:$rpcPassword"))
        $headers = @{
            "Authorization" = "Basic $auth"
            "Content-Type" = "application/json"
        }

        # Get blockchain info
        $blockchainInfo = Invoke-RestMethod -Uri $bitcoinRpcUrl -Method POST -Headers $headers -Body '{"jsonrpc":"1.0","id":"test","method":"getblockchaininfo","params":[]}' -TimeoutSec 5

        # Get network info
        $networkInfo = Invoke-RestMethod -Uri $bitcoinRpcUrl -Method POST -Headers $headers -Body '{"jsonrpc":"1.0","id":"test","method":"getnetworkinfo","params":[]}' -TimeoutSec 5

        # Get mempool info
        $mempoolInfo = Invoke-RestMethod -Uri $bitcoinRpcUrl -Method POST -Headers $headers -Body '{"jsonrpc":"1.0","id":"test","method":"getmempoolinfo","params":[]}' -TimeoutSec 5

        return @{
            Success = $true
            Timestamp = Get-Date
            BlockchainInfo = $blockchainInfo.result
            NetworkInfo = $networkInfo.result
            MempoolInfo = $mempoolInfo.result
        }
    } catch {
        return @{
            Success = $false
            Timestamp = Get-Date
            Error = $_.Exception.Message
        }
    }
}

# Main monitoring loop
$elapsedSeconds = 0
while ($elapsedSeconds -lt $testDuration) {
    $checkCount++
    $currentTime = Get-Date
    $elapsedSeconds = ($currentTime - $startTime).TotalSeconds
    $remainingMinutes = [math]::Max(0, ($testDuration - $elapsedSeconds) / 60)

    Write-Host "🔄 CHECK #$checkCount - $(Get-Date -Format 'HH:mm:ss')" -ForegroundColor Yellow
    Write-Host "Elapsed: $([math]::Round($elapsedSeconds/60, 1))min | Remaining: $([math]::Round($remainingMinutes, 1))min" -ForegroundColor Yellow

    $status = Get-BitcoinCoreStatus

    if ($status.Success) {
        $blockInfo = $status.BlockchainInfo
        $netInfo = $status.NetworkInfo
        $mempoolInfo = $status.MempoolInfo

        Write-Host "  ✅ Bitcoin Core Status:" -ForegroundColor Green
        Write-Host "    • Block Height: $($blockInfo.blocks)" -ForegroundColor Green
        Write-Host "    • Best Block Hash: $($blockInfo.bestblockhash.Substring(0, 16))..." -ForegroundColor Green
        Write-Host "    • Difficulty: $([math]::Round($blockInfo.difficulty, 2))" -ForegroundColor Green
        Write-Host "    • Verification Progress: $([math]::Round($blockInfo.verificationprogress * 100, 4))%" -ForegroundColor Green
        Write-Host "    • Chain: $($blockInfo.chain)" -ForegroundColor Green
        Write-Host "    • Size on Disk: $([math]::Round($blockInfo.size_on_disk / 1GB, 2)) GB" -ForegroundColor Green
        Write-Host ""

        Write-Host "  🌐 Network Status:" -ForegroundColor Cyan
        Write-Host "    • Connections: $($netInfo.connections)" -ForegroundColor Cyan
        Write-Host "    • Network Active: $($netInfo.networkactive)" -ForegroundColor Cyan
        Write-Host "    • Local Addresses: $($netInfo.localaddresses.Count)" -ForegroundColor Cyan
        Write-Host "    • Relay Fee: $($netInfo.relayfee) BTC" -ForegroundColor Cyan
        Write-Host ""

        Write-Host "  📦 Mempool Status:" -ForegroundColor Magenta
        Write-Host "    • Transactions: $($mempoolInfo.size)" -ForegroundColor Magenta
        Write-Host "    • Memory Usage: $([math]::Round($mempoolInfo.bytes / 1MB, 2)) MB" -ForegroundColor Magenta
        Write-Host "    • Min Fee: $($mempoolInfo.mempoolminfee) BTC" -ForegroundColor Magenta
        Write-Host ""

        # Store data for analysis
        $monitoringData += @{
            CheckNumber = $checkCount
            Timestamp = $status.Timestamp
            BlockHeight = $blockInfo.blocks
            Connections = $netInfo.connections
            VerificationProgress = $blockInfo.verificationprogress
            MempoolSize = $mempoolInfo.size
            MempoolBytes = $mempoolInfo.bytes
            Success = $true
        }

    } else {
        Write-Host "  ❌ Bitcoin Core Error:" -ForegroundColor Red
        Write-Host "    • Error: $($status.Error)" -ForegroundColor Red
        Write-Host ""

        # Store error data
        $monitoringData += @{
            CheckNumber = $checkCount
            Timestamp = $status.Timestamp
            Success = $false
            Error = $status.Error
        }
    }

    # Progress indicator
    $progressPercent = [math]::Min(100, ($elapsedSeconds / $testDuration) * 100)
    Write-Host "  📈 Progress: $([math]::Round($progressPercent, 1))% complete" -ForegroundColor White
    Write-Host ""

    # Wait for next check (unless this is the last iteration)
    if ($elapsedSeconds -lt $testDuration) {
        Start-Sleep -Seconds $checkInterval
    }
}

# Analysis and summary
Write-Host "📊 MONITORING SESSION COMPLETE" -ForegroundColor Green
Write-Host "================================"

$endTime = Get-Date
$totalDuration = $endTime - $startTime

Write-Host "Session Summary:" -ForegroundColor Yellow
Write-Host "  • Total Duration: $([math]::Round($totalDuration.TotalMinutes, 1)) minutes" -ForegroundColor Yellow
Write-Host "  • Total Checks: $checkCount" -ForegroundColor Yellow
Write-Host "  • Check Interval: $checkInterval seconds" -ForegroundColor Yellow
Write-Host "  • Start Time: $($startTime.ToString('yyyy-MM-dd HH:mm:ss'))" -ForegroundColor Yellow
Write-Host "  • End Time: $($endTime.ToString('yyyy-MM-dd HH:mm:ss'))" -ForegroundColor Yellow
Write-Host ""

# Analyze successful checks
$successfulChecks = $monitoringData | Where-Object { $_.Success }
$failedChecks = $monitoringData | Where-Object { -not $_.Success }

Write-Host "Success Rate Analysis:" -ForegroundColor Cyan
Write-Host "  • Successful Checks: $($successfulChecks.Count)/$checkCount" -ForegroundColor Cyan
Write-Host "  • Failed Checks: $($failedChecks.Count)/$checkCount" -ForegroundColor Cyan
Write-Host "  • Success Rate: $([math]::Round(($successfulChecks.Count / $checkCount) * 100, 1))%" -ForegroundColor Cyan
Write-Host ""

if ($successfulChecks.Count -gt 0) {
    # Block height analysis
    $blockHeights = $successfulChecks | ForEach-Object { $_.BlockHeight } | Sort-Object
    $minBlockHeight = $blockHeights | Measure-Object -Minimum | Select-Object -ExpandProperty Minimum
    $maxBlockHeight = $blockHeights | Measure-Object -Maximum | Select-Object -ExpandProperty Maximum
    $blockHeightChange = $maxBlockHeight - $minBlockHeight

    # Connection analysis
    $connections = $successfulChecks | ForEach-Object { $_.Connections }
    $avgConnections = [math]::Round(($connections | Measure-Object -Average | Select-Object -ExpandProperty Average), 1)
    $minConnections = $connections | Measure-Object -Minimum | Select-Object -ExpandProperty Minimum
    $maxConnections = $connections | Measure-Object -Maximum | Select-Object -ExpandProperty Maximum

    # Mempool analysis
    $mempoolSizes = $successfulChecks | ForEach-Object { $_.MempoolSize }
    $avgMempoolSize = [math]::Round(($mempoolSizes | Measure-Object -Average | Select-Object -ExpandProperty Average), 0)
    $maxMempoolSize = $mempoolSizes | Measure-Object -Maximum | Select-Object -ExpandProperty Maximum

    Write-Host "Bitcoin Core Performance Analysis:" -ForegroundColor Green
    Write-Host "  📊 Block Height:" -ForegroundColor Green
    Write-Host "    • Range: $minBlockHeight → $maxBlockHeight" -ForegroundColor Green
    Write-Host "    • Blocks Added: $blockHeightChange" -ForegroundColor Green
    Write-Host "    • Average Blocks/Minute: $([math]::Round($blockHeightChange / $totalDuration.TotalMinutes, 2))" -ForegroundColor Green
    Write-Host ""

    Write-Host "  🌐 Network Connections:" -ForegroundColor Cyan
    Write-Host "    • Average: $avgConnections connections" -ForegroundColor Cyan
    Write-Host "    • Range: $minConnections - $maxConnections connections" -ForegroundColor Cyan
    Write-Host ""

    Write-Host "  📦 Mempool Activity:" -ForegroundColor Magenta
    Write-Host "    • Average Size: $avgMempoolSize transactions" -ForegroundColor Magenta
    Write-Host "    • Peak Size: $maxMempoolSize transactions" -ForegroundColor Magenta
    Write-Host ""

    # Calculate sync status
    $syncProgresses = $successfulChecks | ForEach-Object { $_.VerificationProgress }
    $avgSyncProgress = [math]::Round(($syncProgresses | Measure-Object -Average | Select-Object -ExpandProperty Average) * 100, 4)
    $minSyncProgress = [math]::Round(($syncProgresses | Measure-Object -Minimum | Select-Object -ExpandProperty Minimum) * 100, 4)
    $maxSyncProgress = [math]::Round(($syncProgresses | Measure-Object -Maximum | Select-Object -ExpandProperty Maximum) * 100, 4)

    Write-Host "  🔄 Synchronization Status:" -ForegroundColor Yellow
    Write-Host "    • Average Progress: $avgSyncProgress%" -ForegroundColor Yellow
    Write-Host "    • Range: $minSyncProgress% - $maxSyncProgress%" -ForegroundColor Yellow
    Write-Host ""

    # Performance insights
    Write-Host "  ⚡ Performance Insights:" -ForegroundColor White
    if ($blockHeightChange -gt 0) {
        Write-Host "    • Bitcoin Core is actively syncing new blocks" -ForegroundColor White
        Write-Host "    • Block processing rate: $([math]::Round($blockHeightChange / $totalDuration.TotalMinutes, 2)) blocks/min" -ForegroundColor White
    } else {
        Write-Host "    • Bitcoin Core is fully synced" -ForegroundColor White
    }

    if ($avgConnections -ge 8) {
        Write-Host "    • Good network connectivity ($avgConnections peers)" -ForegroundColor White
    } elseif ($avgConnections -ge 3) {
        Write-Host "    • Adequate network connectivity ($avgConnections peers)" -ForegroundColor White
    } else {
        Write-Host "    • Limited network connectivity ($avgConnections peers)" -ForegroundColor White
    }

    if ($avgSyncProgress -ge 99.9999) {
        Write-Host "    • Blockchain fully verified and synced" -ForegroundColor White
    } elseif ($avgSyncProgress -ge 99.99) {
        Write-Host "    • Blockchain nearly fully synced" -ForegroundColor White
    } else {
        Write-Host "    • Blockchain synchronization in progress" -ForegroundColor White
    }
    Write-Host ""
}

# Error analysis
if ($failedChecks.Count -gt 0) {
    Write-Host "Error Analysis:" -ForegroundColor Red
    Write-Host "  • Total Errors: $($failedChecks.Count)" -ForegroundColor Red

    # Group errors by type
    $errorGroups = $failedChecks | Group-Object -Property Error
    foreach ($group in $errorGroups) {
        Write-Host "    • $($group.Name): $($group.Count) occurrences" -ForegroundColor Red
    }
    Write-Host ""
}

# Recommendations
Write-Host "🎯 RECOMMENDATIONS" -ForegroundColor Cyan
Write-Host "=================="

if ($successfulChecks.Count -eq $checkCount) {
    Write-Host "  ✅ Bitcoin Core is running reliably" -ForegroundColor Green
    Write-Host "  ✅ All monitoring checks passed" -ForegroundColor Green
} else {
    Write-Host "  ⚠️  Some monitoring checks failed" -ForegroundColor Yellow
    Write-Host "  ⚠️  Consider checking Bitcoin Core configuration" -ForegroundColor Yellow
}

if ($successfulChecks.Count -gt 0) {
    if ($avgConnections -lt 8) {
        Write-Host "  💡 Consider increasing max connections for better network resilience" -ForegroundColor Cyan
    }

    if ($blockHeightChange -eq 0 -and $avgSyncProgress -lt 100) {
        Write-Host "  💡 Bitcoin Core may need more time to complete initial sync" -ForegroundColor Cyan
    }

    Write-Host "  💡 Monitoring shows stable Bitcoin Core operation" -ForegroundColor Cyan
}

Write-Host ""
Write-Host "✨ Bitcoin Core monitoring test completed!" -ForegroundColor Green
Write-Host "Duration: $([math]::Round($totalDuration.TotalMinutes, 1)) minutes" -ForegroundColor Green
Write-Host "Checks performed: $checkCount" -ForegroundColor Green
Write-Host "Success rate: $([math]::Round(($successfulChecks.Count / $checkCount) * 100, 1))%" -ForegroundColor Green
