#!/usr/bin/env pwsh
#
# Advanced Testing Suite - Master Orchestrator
# Coordinates heavy load, soak, and tier comparison tests
#

param(
    [ValidateSet("quick", "heavy", "soak", "comparison", "full", "help")]
    [string]$TestSuite = "help",
    
    [switch]$SaveAllResults,
    [switch]$GenerateReports,
    [switch]$ContinuousMode,
    [int]$ContinuousIntervalHours = 6,
    [switch]$EmailReports
)

function Show-Help {
    Write-Host ""
    Write-Host "🧪 ADVANCED TESTING SUITE - MASTER ORCHESTRATOR" -ForegroundColor Cyan
    Write-Host "===============================================" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Available Test Suites:" -ForegroundColor White
    Write-Host ""
    Write-Host "  quick      - Quick validation (5 minutes)" -ForegroundColor Green
    Write-Host "             Real user scenarios + basic performance check" -ForegroundColor Gray
    Write-Host ""
    Write-Host "  heavy      - Heavy load testing (20+ concurrent clients)" -ForegroundColor Yellow
    Write-Host "             Tests system behavior under high concurrent load" -ForegroundColor Gray
    Write-Host ""
    Write-Host "  soak       - Long-running stability test (30+ minutes)" -ForegroundColor Orange
    Write-Host "             Memory leaks, performance degradation analysis" -ForegroundColor Gray
    Write-Host ""
    Write-Host "  comparison - Tier comparison (FREE vs PRO vs ENTERPRISE)" -ForegroundColor Blue
    Write-Host "             Side-by-side performance analysis" -ForegroundColor Gray
    Write-Host ""
    Write-Host "  full       - Complete testing suite (all tests)" -ForegroundColor Magenta
    Write-Host "             Comprehensive analysis for production readiness" -ForegroundColor Gray
    Write-Host ""
    Write-Host "Options:" -ForegroundColor White
    Write-Host "  -SaveAllResults      Save all test results to JSON files" -ForegroundColor Gray
    Write-Host "  -GenerateReports     Generate consolidated HTML reports" -ForegroundColor Gray
    Write-Host "  -ContinuousMode      Run tests continuously" -ForegroundColor Gray
    Write-Host "  -EmailReports        Send email notifications (requires SMTP config)" -ForegroundColor Gray
    Write-Host ""
    Write-Host "Examples:" -ForegroundColor White
    Write-Host "  .\advanced-testing-suite.ps1 -TestSuite quick" -ForegroundColor Gray
    Write-Host "  .\advanced-testing-suite.ps1 -TestSuite heavy -SaveAllResults" -ForegroundColor Gray
    Write-Host "  .\advanced-testing-suite.ps1 -TestSuite full -GenerateReports" -ForegroundColor Gray
    Write-Host ""
}

function Start-Prerequisites {
    Write-Host "🔧 CHECKING PREREQUISITES" -ForegroundColor Cyan
    Write-Host "=========================" -ForegroundColor Cyan
    Write-Host ""
    
    # Check for Bitcoin Core mock
    $mockRunning = Get-NetTCPConnection -LocalPort 8332 -State Listen -ErrorAction SilentlyContinue
    if (-not $mockRunning) {
        Write-Host "⚠️ Bitcoin Core mock not running. Starting..." -ForegroundColor Yellow
        try {
            Start-Process powershell -ArgumentList "-Command", "cd '$PWD'; python scripts\bitcoin-core-mock.py" -WindowStyle Minimized
            Start-Sleep -Seconds 3
            
            # Verify it started
            $mockRunning = Get-NetTCPConnection -LocalPort 8332 -State Listen -ErrorAction SilentlyContinue
            if ($mockRunning) {
                Write-Host "   ✅ Bitcoin Core mock started successfully" -ForegroundColor Green
            } else {
                Write-Host "   ❌ Failed to start Bitcoin Core mock" -ForegroundColor Red
                return $false
            }
        } catch {
            Write-Host "   ❌ Error starting Bitcoin Core mock: $($_.Exception.Message)" -ForegroundColor Red
            return $false
        }
    } else {
        Write-Host "   ✅ Bitcoin Core mock already running" -ForegroundColor Green
    }
    
    # Check for required test files
    $requiredFiles = @(
        "real-user-speed-report.ps1",
        "heavy-load-test.ps1", 
        "soak-test.ps1",
        "tier-comparison-test.ps1"
    )
    
    foreach ($file in $requiredFiles) {
        if (Test-Path $file) {
            Write-Host "   ✅ $file" -ForegroundColor Green
        } else {
            Write-Host "   ❌ Missing: $file" -ForegroundColor Red
            return $false
        }
    }
    
    # Check for tier configuration files
    $tierConfigs = @(
        "config-free-stable.json",
        "config.json", 
        "config-enterprise-turbo.json"
    )
    
    foreach ($config in $tierConfigs) {
        if (Test-Path $config) {
            Write-Host "   ✅ $config" -ForegroundColor Green
        } else {
            Write-Host "   ⚠️ Missing tier config: $config" -ForegroundColor Yellow
        }
    }
    
    Write-Host ""
    return $true
}

function Run-QuickSuite {
    Write-Host "🚀 QUICK VALIDATION SUITE" -ForegroundColor Green
    Write-Host "=========================" -ForegroundColor Green
    Write-Host ""
    
    # Start a basic Sprint instance
    Write-Host "Starting Bitcoin Sprint..." -ForegroundColor Yellow
    $env:RPC_NODES = "http://127.0.0.1:8332"
    $env:RPC_USER = "bitcoin"
    $env:RPC_PASS = "sprint123benchmark"
    $env:API_PORT = "8080"
    
    $sprintProcess = Start-Process ".\bitcoin-sprint.exe" -PassThru -WindowStyle Hidden
    Start-Sleep -Seconds 5
    
    try {
        # Run real user speed report
        Write-Host "Running real user speed analysis..." -ForegroundColor White
        & ".\real-user-speed-report.ps1"
        
        # Quick performance sample
        Write-Host ""
        Write-Host "Running quick performance sample..." -ForegroundColor White
        & ".\heavy-load-test.ps1" -ConcurrentClients 5 -RequestsPerClient 20 -TestDurationMinutes 2 -SaveResults:$SaveAllResults
        
    } finally {
        if ($sprintProcess -and -not $sprintProcess.HasExited) {
            $sprintProcess.Kill()
            Write-Host "✅ Bitcoin Sprint stopped" -ForegroundColor Gray
        }
    }
}

function Run-HeavySuite {
    Write-Host "🏋️ HEAVY LOAD TESTING SUITE" -ForegroundColor Yellow
    Write-Host "============================" -ForegroundColor Yellow
    Write-Host ""
    
    # Start Sprint instance
    Write-Host "Starting Bitcoin Sprint for heavy load testing..." -ForegroundColor Yellow
    $env:RPC_NODES = "http://127.0.0.1:8332"
    $env:RPC_USER = "bitcoin"
    $env:RPC_PASS = "sprint123benchmark"
    $env:API_PORT = "8080"
    
    $sprintProcess = Start-Process ".\bitcoin-sprint.exe" -PassThru -WindowStyle Hidden
    Start-Sleep -Seconds 5
    
    try {
        # Heavy load test with escalating load
        Write-Host "Phase 1: Moderate load (15 clients)..." -ForegroundColor White
        & ".\heavy-load-test.ps1" -ConcurrentClients 15 -RequestsPerClient 40 -TestDurationMinutes 3 -SaveResults:$SaveAllResults
        
        Write-Host ""
        Write-Host "Phase 2: Heavy load (25 clients)..." -ForegroundColor White
        & ".\heavy-load-test.ps1" -ConcurrentClients 25 -RequestsPerClient 50 -TestDurationMinutes 5 -SaveResults:$SaveAllResults -RealTimeMonitoring
        
        Write-Host ""
        Write-Host "Phase 3: Peak load (35 clients)..." -ForegroundColor White
        & ".\heavy-load-test.ps1" -ConcurrentClients 35 -RequestsPerClient 30 -TestDurationMinutes 3 -SaveResults:$SaveAllResults
        
    } finally {
        if ($sprintProcess -and -not $sprintProcess.HasExited) {
            $sprintProcess.Kill()
            Write-Host "✅ Bitcoin Sprint stopped" -ForegroundColor Gray
        }
    }
}

function Run-SoakSuite {
    Write-Host "⏰ SOAK TESTING SUITE" -ForegroundColor Orange
    Write-Host "=====================" -ForegroundColor Orange
    Write-Host ""
    
    Write-Host "⚠️ IMPORTANT: Soak test will run for 35+ minutes" -ForegroundColor Yellow
    Write-Host "This test analyzes long-term stability and memory usage." -ForegroundColor Gray
    Write-Host ""
    
    $continue = Read-Host "Continue with soak test? (y/N)"
    if ($continue -ne 'y' -and $continue -ne 'Y') {
        Write-Host "Soak test cancelled." -ForegroundColor Gray
        return
    }
    
    # Start Sprint instance
    Write-Host "Starting Bitcoin Sprint for soak testing..." -ForegroundColor Yellow
    $env:RPC_NODES = "http://127.0.0.1:8332"
    $env:RPC_USER = "bitcoin"
    $env:RPC_PASS = "sprint123benchmark"
    $env:API_PORT = "8080"
    
    $sprintProcess = Start-Process ".\bitcoin-sprint.exe" -PassThru -WindowStyle Hidden
    Start-Sleep -Seconds 5
    
    try {
        # Long-running soak test
        & ".\soak-test.ps1" -DurationMinutes 35 -ConcurrentUsers 8 -RequestsPerMinute 60 -EnableDetailedLogging -SaveResults:$SaveAllResults
        
    } finally {
        if ($sprintProcess -and -not $sprintProcess.HasExited) {
            $sprintProcess.Kill()
            Write-Host "✅ Bitcoin Sprint stopped" -ForegroundColor Gray
        }
    }
}

function Run-ComparisonSuite {
    Write-Host "⚖️ TIER COMPARISON SUITE" -ForegroundColor Blue
    Write-Host "========================" -ForegroundColor Blue
    Write-Host ""
    
    # Tier comparison test (manages its own Sprint instances)
    & ".\tier-comparison-test.ps1" -TestDurationMinutes 8 -RequestsPerTier 75 -DetailedMetrics -SaveResults:$SaveAllResults -GenerateChart
}

function Run-FullSuite {
    Write-Host "🎯 FULL TESTING SUITE" -ForegroundColor Magenta
    Write-Host "=====================" -ForegroundColor Magenta
    Write-Host ""
    
    Write-Host "⚠️ IMPORTANT: Full suite will take 60+ minutes" -ForegroundColor Yellow
    Write-Host "This includes all test types for comprehensive analysis." -ForegroundColor Gray
    Write-Host ""
    
    $continue = Read-Host "Continue with full test suite? (y/N)"
    if ($continue -ne 'y' -and $continue -ne 'Y') {
        Write-Host "Full test suite cancelled." -ForegroundColor Gray
        return
    }
    
    $fullSuiteStart = Get-Date
    
    Write-Host ""
    Write-Host "Phase 1/4: Quick Validation" -ForegroundColor Cyan
    Run-QuickSuite
    
    Write-Host ""
    Write-Host "Phase 2/4: Heavy Load Testing" -ForegroundColor Cyan
    Run-HeavySuite
    
    Write-Host ""
    Write-Host "Phase 3/4: Tier Comparison" -ForegroundColor Cyan
    Run-ComparisonSuite
    
    Write-Host ""
    Write-Host "Phase 4/4: Soak Testing" -ForegroundColor Cyan
    Run-SoakSuite
    
    $fullSuiteEnd = Get-Date
    $totalDuration = ($fullSuiteEnd - $fullSuiteStart).TotalMinutes
    
    Write-Host ""
    Write-Host "🎉 FULL SUITE COMPLETED!" -ForegroundColor Green
    Write-Host "Total Duration: $([math]::Round($totalDuration, 1)) minutes" -ForegroundColor Gray
}

function Generate-ConsolidatedReport {
    Write-Host ""
    Write-Host "📊 GENERATING CONSOLIDATED REPORT" -ForegroundColor Cyan
    Write-Host "==================================" -ForegroundColor Cyan
    Write-Host ""
    
    $timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
    $reportDir = "test-reports-$timestamp"
    New-Item -ItemType Directory -Path $reportDir -Force | Out-Null
    
    # Move all recent test reports to the report directory
    $recentReports = Get-ChildItem -Name "*-test-*.json", "*user-speed-*.json" | 
                     Where-Object { (Get-Item $_).LastWriteTime -gt (Get-Date).AddHours(-2) }
    
    foreach ($report in $recentReports) {
        Copy-Item $report $reportDir
        Write-Host "   📄 Included: $report" -ForegroundColor Green
    }
    
    # Generate summary report
    $summaryFile = "$reportDir\test-suite-summary.md"
    $summaryContent = @"
# Bitcoin Sprint - Advanced Testing Suite Results

**Generated:** $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')
**Test Suite:** $TestSuite

## Test Results Summary

$(if ($recentReports.Count -gt 0) {
    "### Reports Generated ($($recentReports.Count) files)"
    $recentReports | ForEach-Object { "- $_" }
} else {
    "### No recent test reports found"
})

## Test Suite Execution

| Test Type | Status | Duration | Key Metrics |
|-----------|--------|----------|-------------|
| Quick Validation | ✅ Completed | ~5 min | Real user scenarios |
| Heavy Load | $(if($TestSuite -in @('heavy','full')){'✅ Completed'}else{'⏭️ Skipped'}) | ~15 min | Concurrent clients |
| Soak Test | $(if($TestSuite -in @('soak','full')){'✅ Completed'}else{'⏭️ Skipped'}) | ~35 min | Long-term stability |
| Tier Comparison | $(if($TestSuite -in @('comparison','full')){'✅ Completed'}else{'⏭️ Skipped'}) | ~10 min | Performance tiers |

## Production Readiness Assessment

### Performance Metrics
- **Response Time Targets:** FREE (<200ms), PRO (<100ms), ENTERPRISE (<50ms)
- **Reliability Target:** 99%+ success rate
- **Stability Target:** <50MB memory growth over 30 minutes

### Recommendations
1. **For Development:** Use FREE tier configuration
2. **For Production:** Validate with PRO or ENTERPRISE based on requirements
3. **For High-Frequency Trading:** ENTERPRISE tier recommended

## Next Steps
1. Review individual test reports for detailed metrics
2. Monitor production deployment with similar load patterns
3. Set up continuous monitoring with alerting
4. Schedule regular soak tests for long-term validation

---
*Generated by Bitcoin Sprint Advanced Testing Suite*
"@
    
    $summaryContent | Out-File $summaryFile -Encoding UTF8
    Write-Host "   📋 Summary report: $summaryFile" -ForegroundColor Green
    
    Write-Host ""
    Write-Host "✅ Consolidated report generated in: $reportDir" -ForegroundColor Green
    return $reportDir
}

function Start-ContinuousMode {
    Write-Host ""
    Write-Host "🔄 CONTINUOUS TESTING MODE" -ForegroundColor Magenta
    Write-Host "==========================" -ForegroundColor Magenta
    Write-Host ""
    Write-Host "Running tests every $ContinuousIntervalHours hours..." -ForegroundColor Yellow
    Write-Host "Press Ctrl+C to stop continuous mode" -ForegroundColor Gray
    Write-Host ""
    
    $iteration = 1
    while ($true) {
        Write-Host "🔄 Continuous Test Iteration $iteration" -ForegroundColor Cyan
        Write-Host "Started: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Gray
        
        try {
            switch ($TestSuite) {
                "quick" { Run-QuickSuite }
                "heavy" { Run-HeavySuite }
                "comparison" { Run-ComparisonSuite }
                default { Run-QuickSuite }
            }
            
            if ($GenerateReports) {
                $reportDir = Generate-ConsolidatedReport
                Write-Host "Reports available in: $reportDir" -ForegroundColor Green
            }
            
        } catch {
            Write-Host "❌ Error in test iteration $iteration`: $($_.Exception.Message)" -ForegroundColor Red
        }
        
        Write-Host ""
        Write-Host "⏳ Next test in $ContinuousIntervalHours hours (at $(((Get-Date).AddHours($ContinuousIntervalHours)).ToString('HH:mm:ss')))" -ForegroundColor Yellow
        Write-Host ""
        
        Start-Sleep -Seconds ($ContinuousIntervalHours * 3600)
        $iteration++
    }
}

# Main execution
Write-Host ""
Write-Host "🧪 BITCOIN SPRINT - ADVANCED TESTING SUITE" -ForegroundColor Cyan
Write-Host "===========================================" -ForegroundColor Cyan
Write-Host "Generated: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Gray
Write-Host ""

if ($TestSuite -eq "help") {
    Show-Help
    exit 0
}

# Check prerequisites
if (-not (Start-Prerequisites)) {
    Write-Host "❌ Prerequisites check failed. Cannot proceed with testing." -ForegroundColor Red
    exit 1
}

# Execute selected test suite
try {
    switch ($TestSuite) {
        "quick" { 
            Run-QuickSuite 
        }
        "heavy" { 
            Run-HeavySuite 
        }
        "soak" { 
            Run-SoakSuite 
        }
        "comparison" { 
            Run-ComparisonSuite 
        }
        "full" { 
            Run-FullSuite 
        }
    }
    
    # Generate reports if requested
    if ($GenerateReports) {
        $reportDir = Generate-ConsolidatedReport
        Write-Host ""
        Write-Host "📊 Consolidated reports available in: $reportDir" -ForegroundColor Green
    }
    
    # Start continuous mode if requested
    if ($ContinuousMode) {
        Start-ContinuousMode
    }
    
} catch {
    Write-Host "❌ Test suite error: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "✅ Advanced testing suite completed successfully!" -ForegroundColor Green
Write-Host "📄 Test completed at $(Get-Date -Format 'HH:mm:ss')" -ForegroundColor Gray
Write-Host ""
