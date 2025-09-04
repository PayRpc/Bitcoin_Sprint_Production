# Bitcoin Sprint Performance Analysis Tool
# Analyzes tier performance data and provides insights

param(
    [Parameter(Mandatory=$false)]
    [string]$ResultsFile,

    [Parameter(Mandatory=$false)]
    [switch]$GenerateReport,

    [Parameter(Mandatory=$false)]
    [switch]$CompareWithBaseline,

    [Parameter(Mandatory=$false)]
    [string]$BaselineFile = "baseline-performance.json"
)

$ErrorActionPreference = "Stop"

# Tier configurations
$tierConfig = @{
    "free" = @{
        "Name" = "Free Tier"
        "Color" = "Gray"
        "ExpectedLatency" = "2000-5000ms"
        "RateLimit" = "1 req/sec"
        "Cost" = "$0"
    }
    "pro" = @{
        "Name" = "Pro Tier"
        "Color" = "Blue"
        "ExpectedLatency" = "500-1200ms"
        "RateLimit" = "10 req/sec"
        "Cost" = "$10/month"
    }
    "business" = @{
        "Name" = "Business Tier"
        "Color" = "Yellow"
        "ExpectedLatency" = "200-800ms"
        "RateLimit" = "50 req/sec"
        "Cost" = "$50/month"
    }
    "turbo" = @{
        "Name" = "Turbo Tier"
        "Color" = "Green"
        "ExpectedLatency" = "50-300ms"
        "RateLimit" = "100 req/sec"
        "Cost" = "$100/month"
    }
    "enterprise" = @{
        "Name" = "Enterprise Tier"
        "Color" = "Magenta"
        "ExpectedLatency" = "10-150ms"
        "RateLimit" = "500 req/sec"
        "Cost" = "$500/month"
    }
}

function Write-ColoredOutput {
    param($Message, $Color = "White")
    Write-Host $Message -ForegroundColor $Color
}

function Load-Results {
    param($FilePath)

    if (!(Test-Path $FilePath)) {
        Write-ColoredOutput "❌ Results file not found: $FilePath" "Red"
        return $null
    }

    try {
        $content = Get-Content $FilePath -Raw | ConvertFrom-Json
        Write-ColoredOutput "✅ Loaded results from $FilePath" "Green"
        return $content
    }
    catch {
        Write-ColoredOutput "❌ Error loading results: $($_.Exception.Message)" "Red"
        return $null
    }
}

function Analyze-Performance {
    param($Results)

    Write-ColoredOutput "`n📊 PERFORMANCE ANALYSIS" "Cyan"
    Write-ColoredOutput "=" * 40 "Cyan"

    $analysis = @{
        "Tiers" = @{}
        "Summary" = @{}
        "Recommendations" = @()
    }

    foreach ($tier in $Results.PSObject.Properties.Name) {
        $tierData = $Results.$tier
        $analysis.Tiers[$tier] = @{}

        Write-ColoredOutput "`n🏷️ $($tierConfig[$tier].Name.ToUpper()) ANALYSIS" $tierConfig[$tier].Color
        Write-ColoredOutput "-" * 30 $tierConfig[$tier].Color

        # Calculate key metrics
        $totalRequests = $tierData.Requests
        $successfulRequests = $tierData.Successful
        $failedRequests = $tierData.Failed
        $successRate = if ($totalRequests -gt 0) { [math]::Round(($successfulRequests / $totalRequests) * 100, 2) } else { 0 }

        $analysis.Tiers[$tier]["SuccessRate"] = $successRate
        $analysis.Tiers[$tier]["TotalRequests"] = $totalRequests

        Write-ColoredOutput "📈 Success Rate: $successRate%" "White"

        if ($tierData.Latencies -and $tierData.Latencies.Count -gt 0) {
            $latencies = $tierData.Latencies | ForEach-Object { [double]$_ }
            $avgLatency = [math]::Round(($latencies | Measure-Object -Average).Average, 2)
            $minLatency = [math]::Round(($latencies | Measure-Object -Minimum).Minimum, 2)
            $maxLatency = [math]::Round(($latencies | Measure-Object -Maximum).Maximum, 2)
            $p95Latency = [math]::Round(($latencies | Sort-Object)[[math]::Floor($latencies.Count * 0.95)], 2)
            $p99Latency = [math]::Round(($latencies | Sort-Object)[[math]::Floor($latencies.Count * 0.99)], 2)

            $analysis.Tiers[$tier]["AvgLatency"] = $avgLatency
            $analysis.Tiers[$tier]["P95Latency"] = $p95Latency
            $analysis.Tiers[$tier]["P99Latency"] = $p99Latency

            Write-ColoredOutput "⏱️ Latency Metrics:" "White"
            Write-ColoredOutput "  • Average: $avgLatency ms" "White"
            Write-ColoredOutput "  • P95: $p95Latency ms" "Yellow"
            Write-ColoredOutput "  • P99: $p99Latency ms" "Red"
            Write-ColoredOutput "  • Min/Max: $minLatency/$maxLatency ms" "White"

            # Performance grade
            $grade = Get-PerformanceGrade $avgLatency $tier
            $analysis.Tiers[$tier]["Grade"] = $grade
            Write-ColoredOutput "🎯 Performance Grade: $grade" "White"
        }

        # Error analysis
        if ($tierData.Errors -and $tierData.Errors.Count -gt 0) {
            Write-ColoredOutput "`n❌ Error Analysis:" "Red"
            $errorGroups = $tierData.Errors | Group-Object | Sort-Object Count -Descending | Select-Object -First 3
            foreach ($error in $errorGroups) {
                $percentage = [math]::Round(($error.Count / $totalRequests) * 100, 2)
                Write-ColoredOutput "  • $($error.Name): $($error.Count) ($percentage%)" "Red"
            }
        }
    }

    return $analysis
}

function Get-PerformanceGrade {
    param($AvgLatency, $Tier)

    # Expected latency ranges in milliseconds
    $expectedRanges = @{
        "free" = @{ "Min" = 2000; "Max" = 5000 }
        "pro" = @{ "Min" = 500; "Max" = 1200 }
        "business" = @{ "Min" = 200; "Max" = 800 }
        "turbo" = @{ "Min" = 50; "Max" = 300 }
        "enterprise" = @{ "Min" = 10; "Max" = 150 }
    }

    $range = $expectedRanges[$Tier]
    if (!$range) { return "Unknown" }

    if ($AvgLatency -le $range.Min) {
        return "Excellent (Better than expected)"
    }
    elseif ($AvgLatency -le $range.Max) {
        return "Good (Within expected range)"
    }
    elseif ($AvgLatency -le ($range.Max * 1.5)) {
        return "Fair (Slightly above expected)"
    }
    else {
        return "Poor (Significantly above expected)"
    }
}

function Generate-Recommendations {
    param($Analysis)

    Write-ColoredOutput "`n💡 RECOMMENDATIONS" "Cyan"
    Write-ColoredOutput "=" * 20 "Cyan"

    $recommendations = @()

    # Find best performing tier
    $bestTier = $null
    $bestLatency = [double]::MaxValue

    foreach ($tier in $Analysis.Tiers.Keys) {
        if ($Analysis.Tiers[$tier].AvgLatency -and $Analysis.Tiers[$tier].AvgLatency -lt $bestLatency) {
            $bestLatency = $Analysis.Tiers[$tier].AvgLatency
            $bestTier = $tier
        }
    }

    if ($bestTier) {
        $recommendations += "🏆 Best Performance: $($tierConfig[$bestTier].Name) with $($Analysis.Tiers[$bestTier].AvgLatency)ms average latency"
    }

    # Cost-benefit analysis
    Write-ColoredOutput "`n💰 Cost-Benefit Analysis:" "Yellow"
    foreach ($tier in @("free", "pro", "business", "turbo", "enterprise")) {
        if ($Analysis.Tiers.ContainsKey($tier) -and $Analysis.Tiers[$tier].AvgLatency) {
            $latency = $Analysis.Tiers[$tier].AvgLatency
            $cost = $tierConfig[$tier].Cost
            Write-ColoredOutput "  • $($tierConfig[$tier].Name): ${latency}ms avg latency - $cost" $tierConfig[$tier].Color
        }
    }

    # Performance improvements
    $performanceImprovements = @()
    $tiers = @("free", "pro", "business", "turbo", "enterprise")

    for ($i = 0; $i -lt ($tiers.Count - 1); $i++) {
        $currentTier = $tiers[$i]
        $nextTier = $tiers[$i + 1]

        if ($Analysis.Tiers.ContainsKey($currentTier) -and $Analysis.Tiers.ContainsKey($nextTier)) {
            $currentLatency = $Analysis.Tiers[$currentTier].AvgLatency
            $nextLatency = $Analysis.Tiers[$nextTier].AvgLatency

            if ($currentLatency -and $nextLatency) {
                $improvement = [math]::Round((($currentLatency - $nextLatency) / $currentLatency) * 100, 2)
                if ($improvement -gt 0) {
                    $performanceImprovements += "$($tierConfig[$nextTier].Name) vs $($tierConfig[$currentTier].Name): ${improvement}% faster"
                }
            }
        }
    }

    if ($performanceImprovements.Count -gt 0) {
        Write-ColoredOutput "`n🚀 Performance Improvements:" "Green"
        foreach ($improvement in $performanceImprovements) {
            Write-ColoredOutput "  • $improvement" "Green"
        }
    }

    # Specific recommendations
    Write-ColoredOutput "`n📋 Specific Recommendations:" "White"

    # Check for high error rates
    foreach ($tier in $Analysis.Tiers.Keys) {
        $successRate = $Analysis.Tiers[$tier].SuccessRate
        if ($successRate -and $successRate -lt 95) {
            $recommendations += "⚠️ $($tierConfig[$tier].Name): High error rate (${successRate}%). Consider investigating stability issues."
        }
    }

    # Check for latency outliers
    foreach ($tier in $Analysis.Tiers.Keys) {
        $p99Latency = $Analysis.Tiers[$tier].P99Latency
        $avgLatency = $Analysis.Tiers[$tier].AvgLatency

        if ($p99Latency -and $avgLatency -and ($p99Latency / $avgLatency) -gt 3) {
            $recommendations += "⚠️ $($tierConfig[$tier].Name): High latency variance detected. P99 is $([math]::Round($p99Latency / $avgLatency, 1))x average."
        }
    }

    foreach ($rec in $recommendations) {
        Write-ColoredOutput "  • $rec" "White"
    }

    return $recommendations
}

function Save-AnalysisReport {
    param($Analysis, $FilePath)

    $report = @{
        "GeneratedAt" = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
        "Analysis" = $Analysis
        "Summary" = @{
            "TotalTiersTested" = $Analysis.Tiers.Count
            "BestPerformingTier" = $null
            "WorstPerformingTier" = $null
            "AverageSuccessRate" = 0
        }
    }

    # Calculate summary statistics
    $totalSuccessRate = 0
    $bestLatency = [double]::MaxValue
    $worstLatency = 0
    $bestTier = $null
    $worstTier = $null

    foreach ($tier in $Analysis.Tiers.Keys) {
        $tierData = $Analysis.Tiers[$tier]

        if ($tierData.SuccessRate) {
            $totalSuccessRate += $tierData.SuccessRate
        }

        if ($tierData.AvgLatency) {
            if ($tierData.AvgLatency -lt $bestLatency) {
                $bestLatency = $tierData.AvgLatency
                $bestTier = $tier
            }

            if ($tierData.AvgLatency -gt $worstLatency) {
                $worstLatency = $tierData.AvgLatency
                $worstTier = $tier
            }
        }
    }

    $report.Summary.BestPerformingTier = $bestTier
    $report.Summary.WorstPerformingTier = $worstTier
    $report.Summary.AverageSuccessRate = [math]::Round($totalSuccessRate / $Analysis.Tiers.Count, 2)

    try {
        $report | ConvertTo-Json -Depth 10 | Out-File $FilePath -Encoding UTF8
        Write-ColoredOutput "✅ Analysis report saved to $FilePath" "Green"
    }
    catch {
        Write-ColoredOutput "❌ Error saving report: $($_.Exception.Message)" "Red"
    }
}

# Main execution
Write-ColoredOutput "🔍 BITCOIN SPRINT PERFORMANCE ANALYSIS TOOL" "Cyan"
Write-ColoredOutput "=" * 50 "Cyan"

$results = $null

if ($ResultsFile) {
    $results = Load-Results $ResultsFile
} else {
    # Look for recent results file
    $recentResults = Get-ChildItem "*.json" | Where-Object { $_.Name -match "tier.*results|performance.*results" } | Sort-Object LastWriteTime -Descending | Select-Object -First 1

    if ($recentResults) {
        Write-ColoredOutput "📁 Found recent results file: $($recentResults.Name)" "Yellow"
        $results = Load-Results $recentResults.FullName
    } else {
        Write-ColoredOutput "❌ No results file found. Please run tier-performance-comparison.ps1 first or specify -ResultsFile" "Red"
        exit 1
    }
}

if ($results) {
    $analysis = Analyze-Performance $results
    $recommendations = Generate-Recommendations $analysis

    if ($GenerateReport) {
        $reportFile = "tier-performance-analysis-$(Get-Date -Format 'yyyyMMdd-HHmmss').json"
        Save-AnalysisReport $analysis $reportFile
    }

    Write-ColoredOutput "`n✅ Analysis completed!" "Green"
} else {
    Write-ColoredOutput "❌ Could not load results for analysis" "Red"
    exit 1
}
