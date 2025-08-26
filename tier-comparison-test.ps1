#!/usr/bin/env pwsh
#
# Tier Comparison Test - Side-by-Side Performance Analysis
# Compares FREE vs PRO vs ENTERPRISE tiers simultaneously
#

param(
    [int]$TestDurationMinutes = 10,
    [int]$RequestsPerTier = 100,
    [string[]]$Endpoints = @("/status", "/latest"),
    [switch]$DetailedMetrics,
    [switch]$SaveResults,
    [switch]$GenerateChart
)

Write-Host "⚖️ TIER COMPARISON TEST - SIDE-BY-SIDE ANALYSIS" -ForegroundColor Cyan
Write-Host "===============================================" -ForegroundColor Cyan
Write-Host "Generated: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Gray
Write-Host ""

Write-Host "📊 Test Configuration:" -ForegroundColor White
Write-Host "   Test Duration: $TestDurationMinutes minutes" -ForegroundColor Gray
Write-Host "   Requests per Tier: $RequestsPerTier" -ForegroundColor Gray
Write-Host "   Endpoints: $($Endpoints -join ', ')" -ForegroundColor Gray
Write-Host ""

# Define tier configurations
$tierConfigs = @(
    @{
        Name = "FREE"
        ConfigFile = "config-free-stable.json"
        Binary = "bitcoin-sprint-free.exe"
        Port = 8080
        ExpectedPerformance = @{
            AvgResponseTime = 150  # 100-200ms target
            P95ResponseTime = 250
            SuccessRate = 99
        }
        Color = "Yellow"
    },
    @{
        Name = "PRO"
        ConfigFile = "config.json"
        Binary = "bitcoin-sprint.exe"
        Port = 8081
        ExpectedPerformance = @{
            AvgResponseTime = 75   # 50-100ms target
            P95ResponseTime = 120
            SuccessRate = 99.5
        }
        Color = "Blue"
    },
    @{
        Name = "ENTERPRISE"
        ConfigFile = "config-enterprise-turbo.json"
        Binary = "bitcoin-sprint-turbo.exe"
        Port = 8082
        ExpectedPerformance = @{
            AvgResponseTime = 35   # <50ms target
            P95ResponseTime = 60
            SuccessRate = 99.9
        }
        Color = "Green"
    }
)

# Check prerequisites
Write-Host "🔍 Checking Prerequisites..." -ForegroundColor White

# Check if Bitcoin Core mock is running
$mockRunning = Get-NetTCPConnection -LocalPort 8332 -State Listen -ErrorAction SilentlyContinue
if (-not $mockRunning) {
    Write-Host "❌ Bitcoin Core mock is not running on port 8332" -ForegroundColor Red
    Write-Host "Please start the mock first: python scripts\bitcoin-core-mock.py" -ForegroundColor Yellow
    exit 1
}
Write-Host "   ✅ Bitcoin Core mock running on port 8332" -ForegroundColor Green

# Check tier configurations and binaries
foreach ($tier in $tierConfigs) {
    if (-not (Test-Path $tier.ConfigFile)) {
        Write-Host "   ❌ Missing config file: $($tier.ConfigFile)" -ForegroundColor Red
        exit 1
    }
    if (-not (Test-Path $tier.Binary)) {
        Write-Host "   ❌ Missing binary: $($tier.Binary)" -ForegroundColor Red
        exit 1
    }
    Write-Host "   ✅ $($tier.Name) tier files ready" -ForegroundColor Green
}

Write-Host ""

# Function to start tier service
function Start-TierService {
    param($tier)
    
    Write-Host "🚀 Starting $($tier.Name) tier on port $($tier.Port)..." -ForegroundColor $tier.Color
    
    # Set environment variables for this tier
    $env:RPC_NODES = "http://127.0.0.1:8332"
    $env:RPC_USER = "bitcoin"
    $env:RPC_PASS = "sprint123benchmark"
    $env:API_PORT = $tier.Port
    $env:CONFIG_FILE = $tier.ConfigFile
    
    # Start the service in background
    $processInfo = New-Object System.Diagnostics.ProcessStartInfo
    $processInfo.FileName = $tier.Binary
    $processInfo.UseShellExecute = $false
    $processInfo.RedirectStandardOutput = $true
    $processInfo.RedirectStandardError = $true
    $processInfo.CreateNoWindow = $true
    $processInfo.EnvironmentVariables["RPC_NODES"] = $env:RPC_NODES
    $processInfo.EnvironmentVariables["RPC_USER"] = $env:RPC_USER
    $processInfo.EnvironmentVariables["RPC_PASS"] = $env:RPC_PASS
    $processInfo.EnvironmentVariables["API_PORT"] = $tier.Port
    $processInfo.EnvironmentVariables["CONFIG_FILE"] = $tier.ConfigFile
    
    $process = [System.Diagnostics.Process]::Start($processInfo)
    
    # Wait for service to start
    Start-Sleep -Seconds 3
    
    # Verify service is responding
    $maxRetries = 10
    for ($retry = 1; $retry -le $maxRetries; $retry++) {
        try {
            $response = Invoke-RestMethod -Uri "http://localhost:$($tier.Port)/status" -TimeoutSec 5
            Write-Host "   ✅ $($tier.Name) service responding on port $($tier.Port)" -ForegroundColor Green
            return $process
        } catch {
            if ($retry -eq $maxRetries) {
                Write-Host "   ❌ $($tier.Name) service failed to start on port $($tier.Port)" -ForegroundColor Red
                return $null
            }
            Start-Sleep -Seconds 2
        }
    }
}

# Function to test tier performance
function Test-TierPerformance {
    param($tier, $requests, $endpoints)
    
    $results = @{
        TierName = $tier.Name
        Port = $tier.Port
        TotalRequests = 0
        SuccessfulRequests = 0
        FailedRequests = 0
        ResponseTimes = @()
        EndpointResults = @{}
        StartTime = Get-Date
        EndTime = $null
    }
    
    foreach ($endpoint in $endpoints) {
        Write-Host "Testing $($tier.Name) - $endpoint..." -ForegroundColor $tier.Color
        
        $endpointTimes = @()
        $endpointErrors = 0
        
        for ($i = 1; $i -le $requests; $i++) {
            try {
                $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
                $response = Invoke-RestMethod -Uri "http://localhost:$($tier.Port)$endpoint" -TimeoutSec 10
                $stopwatch.Stop()
                
                $responseTime = $stopwatch.ElapsedMilliseconds
                $endpointTimes += $responseTime
                $results.ResponseTimes += $responseTime
                $results.SuccessfulRequests++
                
            } catch {
                $endpointErrors++
                $results.FailedRequests++
            }
            
            $results.TotalRequests++
            
            # Small delay to simulate real usage
            Start-Sleep -Milliseconds (Get-Random -Minimum 100 -Maximum 300)
        }
        
        # Calculate endpoint statistics
        if ($endpointTimes.Count -gt 0) {
            $sorted = $endpointTimes | Sort-Object
            $p95Index = [math]::Floor($sorted.Count * 0.95)
            
            $results.EndpointResults[$endpoint] = @{
                RequestCount = $requests
                SuccessCount = $endpointTimes.Count
                ErrorCount = $endpointErrors
                AverageTime = ($endpointTimes | Measure-Object -Average).Average
                P95Time = $sorted[$p95Index]
                MinTime = ($endpointTimes | Measure-Object -Minimum).Minimum
                MaxTime = ($endpointTimes | Measure-Object -Maximum).Maximum
            }
        }
    }
    
    $results.EndTime = Get-Date
    return $results
}

# Function to stop tier service
function Stop-TierService {
    param($process, $tierName)
    
    if ($process -and -not $process.HasExited) {
        try {
            $process.Kill()
            $process.WaitForExit(5000)
            Write-Host "   ✅ $tierName service stopped" -ForegroundColor Gray
        } catch {
            Write-Host "   ⚠️ Error stopping $tierName service: $($_.Exception.Message)" -ForegroundColor Yellow
        }
    }
}

# Start all tier services
Write-Host "🚀 Starting All Tier Services..." -ForegroundColor Cyan
Write-Host ""

$runningServices = @{}
$allStarted = $true

foreach ($tier in $tierConfigs) {
    $process = Start-TierService -tier $tier
    if ($process) {
        $runningServices[$tier.Name] = $process
    } else {
        $allStarted = $false
        break
    }
}

if (-not $allStarted) {
    Write-Host "❌ Failed to start all services. Cleaning up..." -ForegroundColor Red
    foreach ($tierName in $runningServices.Keys) {
        Stop-TierService -process $runningServices[$tierName] -tierName $tierName
    }
    exit 1
}

Write-Host ""
Write-Host "✅ All tier services started successfully!" -ForegroundColor Green
Write-Host ""

# Run performance tests for all tiers
Write-Host "⚡ RUNNING TIER PERFORMANCE TESTS" -ForegroundColor Cyan
Write-Host "=================================" -ForegroundColor Cyan
Write-Host ""

$allTierResults = @()

foreach ($tier in $tierConfigs) {
    Write-Host "🧪 Testing $($tier.Name) Tier Performance..." -ForegroundColor $tier.Color
    $tierResult = Test-TierPerformance -tier $tier -requests $RequestsPerTier -endpoints $Endpoints
    $allTierResults += $tierResult
    Write-Host "   ✅ $($tier.Name) testing completed" -ForegroundColor Green
    Write-Host ""
}

# Stop all services
Write-Host "🛑 Stopping All Services..." -ForegroundColor Yellow
foreach ($tierName in $runningServices.Keys) {
    Stop-TierService -process $runningServices[$tierName] -tierName $tierName
}
Write-Host ""

# Analysis and comparison
Write-Host "📊 TIER COMPARISON RESULTS" -ForegroundColor Cyan
Write-Host "=========================" -ForegroundColor Cyan
Write-Host ""

# Overall performance comparison table
Write-Host "📈 Overall Performance Comparison:" -ForegroundColor White
Write-Host "Tier        | Requests | Success% | Avg(ms) | P95(ms) | Status     | vs Target"
Write-Host "------------|----------|----------|---------|---------|------------|----------"

foreach ($result in $allTierResults) {
    $tier = $tierConfigs | Where-Object { $_.Name -eq $result.TierName }
    
    if ($result.ResponseTimes.Count -gt 0) {
        $avgTime = [math]::Round(($result.ResponseTimes | Measure-Object -Average).Average, 1)
        $sorted = $result.ResponseTimes | Sort-Object
        $p95Index = [math]::Floor($sorted.Count * 0.95)
        $p95Time = [math]::Round($sorted[$p95Index], 1)
        $successRate = [math]::Round($result.SuccessfulRequests / $result.TotalRequests * 100, 1)
        
        # Performance assessment
        $meetsAvgTarget = $avgTime -le $tier.ExpectedPerformance.AvgResponseTime
        $meetsP95Target = $p95Time -le $tier.ExpectedPerformance.P95ResponseTime
        $meetsSuccessTarget = $successRate -ge $tier.ExpectedPerformance.SuccessRate
        
        $status = if ($meetsAvgTarget -and $meetsP95Target -and $meetsSuccessTarget) { "✅ Excellent" }
                  elseif ($meetsAvgTarget -and $meetsSuccessTarget) { "👍 Good" }
                  else { "⚠️ Below Target" }
        
        $vsTarget = if ($meetsAvgTarget) { "✅ Met" } else { "❌ Missed" }
        
        $statusColor = if ($status.StartsWith("✅")) { "Green" }
                       elseif ($status.StartsWith("👍")) { "Yellow" }
                       else { "Red" }
        
        $tierNamePadded = $result.TierName.PadRight(11)
        $requestsPadded = $result.TotalRequests.ToString().PadLeft(8)
        $successPadded = "$successRate%".PadLeft(8)
        $avgPadded = "$avgTime".PadLeft(7)
        $p95Padded = "$p95Time".PadLeft(7)
        $statusPadded = $status.PadRight(10)
        
        Write-Host "$tierNamePadded | $requestsPadded | $successPadded | $avgPadded | $p95Padded | " -NoNewline
        Write-Host "$statusPadded" -ForegroundColor $statusColor -NoNewline
        Write-Host " | $vsTarget"
    }
}

Write-Host ""

# Detailed endpoint analysis
if ($DetailedMetrics) {
    Write-Host "🔍 Detailed Endpoint Analysis:" -ForegroundColor White
    Write-Host ""
    
    foreach ($endpoint in $Endpoints) {
        Write-Host "Endpoint: $endpoint" -ForegroundColor Yellow
        Write-Host "Tier        | Requests | Avg(ms) | P95(ms) | Min | Max | Errors"
        Write-Host "------------|----------|---------|---------|-----|-----|-------"
        
        foreach ($result in $allTierResults) {
            if ($result.EndpointResults.ContainsKey($endpoint)) {
                $epResult = $result.EndpointResults[$endpoint]
                $tierName = $result.TierName.PadRight(11)
                $requests = $epResult.RequestCount.ToString().PadLeft(8)
                $avg = [math]::Round($epResult.AverageTime, 1).ToString().PadLeft(7)
                $p95 = [math]::Round($epResult.P95Time, 1).ToString().PadLeft(7)
                $min = $epResult.MinTime.ToString().PadLeft(3)
                $max = $epResult.MaxTime.ToString().PadLeft(3)
                $errors = $epResult.ErrorCount.ToString().PadLeft(5)
                
                Write-Host "$tierName | $requests | $avg | $p95 | $min | $max | $errors"
            }
        }
        Write-Host ""
    }
}

# Performance ranking
Write-Host "🏆 Performance Ranking:" -ForegroundColor White

$rankedTiers = $allTierResults | 
    Where-Object { $_.ResponseTimes.Count -gt 0 } |
    ForEach-Object {
        $avgTime = ($_.ResponseTimes | Measure-Object -Average).Average
        [PSCustomObject]@{
            TierName = $_.TierName
            AverageResponseTime = $avgTime
            TotalRequests = $_.TotalRequests
            SuccessRate = $_.SuccessfulRequests / $_.TotalRequests * 100
        }
    } |
    Sort-Object AverageResponseTime

for ($i = 0; $i -lt $rankedTiers.Count; $i++) {
    $rank = $i + 1
    $tier = $rankedTiers[$i]
    $medal = switch ($rank) {
        1 { "🥇" }
        2 { "🥈" }
        3 { "🥉" }
        default { "$rank." }
    }
    
    Write-Host "   $medal $($tier.TierName) - $([math]::Round($tier.AverageResponseTime, 1))ms avg" -ForegroundColor $(if($rank -eq 1){'Green'}elseif($rank -eq 2){'Yellow'}else{'Gray'})
}

# Business value analysis
Write-Host ""
Write-Host "💼 Business Value Analysis:" -ForegroundColor White

$fastestTier = $rankedTiers[0]
$slowestTier = $rankedTiers[-1]
$speedImprovement = $slowestTier.AverageResponseTime - $fastestTier.AverageResponseTime

Write-Host "   Fastest Tier: $($fastestTier.TierName) ($([math]::Round($fastestTier.AverageResponseTime, 1))ms)" -ForegroundColor Green
Write-Host "   Speed Improvement: $([math]::Round($speedImprovement, 1))ms ($([math]::Round($speedImprovement/$slowestTier.AverageResponseTime*100, 1))% faster)" -ForegroundColor Cyan

# Recommendations
Write-Host ""
Write-Host "💡 Tier Recommendations:" -ForegroundColor White

foreach ($result in $allTierResults) {
    $tier = $tierConfigs | Where-Object { $_.Name -eq $result.TierName }
    $avgTime = if ($result.ResponseTimes.Count -gt 0) { ($result.ResponseTimes | Measure-Object -Average).Average } else { 999 }
    
    Write-Host "   $($result.TierName):" -ForegroundColor $tier.Color
    
    if ($avgTime -le $tier.ExpectedPerformance.AvgResponseTime) {
        Write-Host "     ✅ Meets performance targets" -ForegroundColor Green
        Write-Host "     ✅ Suitable for production use" -ForegroundColor Green
    } else {
        Write-Host "     ⚠️ Below performance targets" -ForegroundColor Yellow
        Write-Host "     ⚠️ Consider optimization or higher tier" -ForegroundColor Yellow
    }
    
    # Use case recommendations
    $useCase = switch ($result.TierName) {
        "FREE" { "Ideal for: Basic websites, low-traffic applications, development/testing" }
        "PRO" { "Ideal for: Business applications, moderate traffic, mobile apps" }
        "ENTERPRISE" { "Ideal for: Trading platforms, high-frequency apps, mission-critical systems" }
    }
    Write-Host "     $useCase" -ForegroundColor Gray
}

# Save comprehensive results
if ($SaveResults) {
    Write-Host ""
    Write-Host "💾 Saving Tier Comparison Results..." -ForegroundColor Cyan
    
    $timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $reportFile = "tier-comparison-$timestamp.json"
    
    $comparisonReport = @{
        test_info = @{
            timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
            test_type = "Tier Comparison Test"
            test_duration_minutes = $TestDurationMinutes
            requests_per_tier = $RequestsPerTier
            endpoints_tested = $Endpoints
        }
        tier_results = $allTierResults
        performance_ranking = $rankedTiers
        business_analysis = @{
            fastest_tier = $fastestTier.TierName
            fastest_avg_response_ms = [math]::Round($fastestTier.AverageResponseTime, 2)
            speed_improvement_ms = [math]::Round($speedImprovement, 2)
            speed_improvement_percent = [math]::Round($speedImprovement/$slowestTier.AverageResponseTime*100, 1)
        }
        tier_configurations = $tierConfigs
    }
    
    try {
        $comparisonReport | ConvertTo-Json -Depth 6 | Out-File $reportFile -Encoding UTF8
        Write-Host "   ✅ Report saved: $reportFile" -ForegroundColor Green
        Write-Host "   📊 Contains comprehensive tier comparison analysis" -ForegroundColor Gray
    } catch {
        Write-Host "   ⚠️ Failed to save report: $($_.Exception.Message)" -ForegroundColor Yellow
    }
}

# Generate simple chart if requested
if ($GenerateChart -and $allTierResults.Count -gt 0) {
    Write-Host ""
    Write-Host "📊 PERFORMANCE CHART" -ForegroundColor Cyan
    Write-Host "===================" -ForegroundColor Cyan
    Write-Host ""
    
    $maxTime = ($allTierResults | ForEach-Object { 
        if ($_.ResponseTimes.Count -gt 0) { ($_.ResponseTimes | Measure-Object -Average).Average } else { 0 }
    } | Measure-Object -Maximum).Maximum
    
    foreach ($result in $allTierResults) {
        if ($result.ResponseTimes.Count -gt 0) {
            $avgTime = ($result.ResponseTimes | Measure-Object -Average).Average
            $barLength = [math]::Floor(($avgTime / $maxTime) * 40)
            $bar = "█" * $barLength
            
            $tier = $tierConfigs | Where-Object { $_.Name -eq $result.TierName }
            Write-Host "$($result.TierName.PadRight(12)) | " -NoNewline
            Write-Host $bar -ForegroundColor $tier.Color -NoNewline
            Write-Host " $([math]::Round($avgTime, 1))ms"
        }
    }
}

Write-Host ""
Write-Host "📄 Tier comparison completed at $(Get-Date -Format 'HH:mm:ss')" -ForegroundColor Gray
Write-Host "🚀 For heavy load testing, use: .\heavy-load-test.ps1" -ForegroundColor Gray
Write-Host "⏰ For long-running tests, use: .\soak-test.ps1" -ForegroundColor Gray
