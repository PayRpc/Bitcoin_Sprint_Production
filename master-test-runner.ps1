#!/usr/bin/env pwsh
#
# Master Test Runner - One script to rule them all
# Entry point for all Bitcoin Sprint testing scenarios
#

param(
    [ValidateSet("quick", "full", "stress", "ci", "e2e", "help")]
    [string]$Mode = "help",
    
    [ValidateSet("FREE", "PRO", "ENTERPRISE", "ALL")]
    [string]$Tier = "ALL",
    
    [int]$Duration = 30,
    [switch]$Verbose,
    [switch]$GenerateReport,
    [string]$OutputFile = "test-results.json"
)

function Show-Help {
    Write-Host ""
    Write-Host "=========================================" -ForegroundColor Cyan
    Write-Host "BITCOIN SPRINT MASTER TEST RUNNER" -ForegroundColor Cyan
    Write-Host "=========================================" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "USAGE:" -ForegroundColor Yellow
    Write-Host "  .\master-test-runner.ps1 -Mode <mode> [options]" -ForegroundColor White
    Write-Host ""
    Write-Host "MODES:" -ForegroundColor Yellow
    Write-Host "  quick   - Fast validation of current setup (< 1 min)" -ForegroundColor White
    Write-Host "  full    - Complete tier testing with performance validation (3-5 min)" -ForegroundColor White
    Write-Host "  stress  - Load testing with bombardier (configurable duration)" -ForegroundColor White
    Write-Host "  ci      - CI/CD validation with build gating (5-10 min)" -ForegroundColor White
    Write-Host "  e2e     - End-to-end monetization pipeline demo (2-3 min)" -ForegroundColor White
    Write-Host "  help    - Show this help message" -ForegroundColor White
    Write-Host ""
    Write-Host "OPTIONS:" -ForegroundColor Yellow
    Write-Host "  -Tier <tier>           Target tier: FREE, PRO, ENTERPRISE, ALL (default: ALL)" -ForegroundColor White
    Write-Host "  -Duration <seconds>    Duration for stress testing (default: 30)" -ForegroundColor White
    Write-Host "  -Verbose              Show detailed output" -ForegroundColor White
    Write-Host "  -GenerateReport       Generate detailed JSON report" -ForegroundColor White
    Write-Host "  -OutputFile <path>    Report output file (default: test-results.json)" -ForegroundColor White
    Write-Host ""
    Write-Host "EXAMPLES:" -ForegroundColor Yellow
    Write-Host "  .\master-test-runner.ps1 -Mode quick" -ForegroundColor Gray
    Write-Host "  .\master-test-runner.ps1 -Mode full -Tier ENTERPRISE -Verbose" -ForegroundColor Gray
    Write-Host "  .\master-test-runner.ps1 -Mode stress -Duration 60" -ForegroundColor Gray
    Write-Host "  .\master-test-runner.ps1 -Mode ci -GenerateReport" -ForegroundColor Gray
    Write-Host "  .\master-test-runner.ps1 -Mode e2e -Tier FREE" -ForegroundColor Gray
    Write-Host ""
}

function Write-MasterLog {
    param($Message, $Level = "INFO")
    $timestamp = Get-Date -Format "HH:mm:ss"
    $color = switch ($Level) {
        "ERROR" { "Red" }
        "WARN"  { "Yellow" }
        "INFO"  { "White" }
        "SUCCESS" { "Green" }
        "HEADER" { "Cyan" }
        default { "Gray" }
    }
    Write-Host "[$timestamp] $Message" -ForegroundColor $color
}

function Test-QuickMode {
    Write-MasterLog "=== QUICK MODE: Basic Validation ===" "HEADER"
    
    try {
        # Check if scripts exist
        $requiredScripts = @("quick-tier-test.ps1", "scripts\bitcoin-core-mock.py")
        foreach ($script in $requiredScripts) {
            if (-not (Test-Path $script)) {
                throw "Required script missing: $script"
            }
        }
        
        Write-MasterLog "Running quick tier test..." "INFO"
        & .\quick-tier-test.ps1
        
        if ($LASTEXITCODE -eq 0) {
            Write-MasterLog "✅ Quick validation passed" "SUCCESS"
            return $true
        } else {
            Write-MasterLog "❌ Quick validation failed" "ERROR"
            return $false
        }
        
    } catch {
        Write-MasterLog "❌ Quick mode error: $($_.Exception.Message)" "ERROR"
        return $false
    }
}

function Test-FullMode {
    Write-MasterLog "=== FULL MODE: Complete Tier Testing ===" "HEADER"
    
    try {
        Write-MasterLog "Running automated tier harness..." "INFO"
        
        $args = @()
        if ($Verbose) { $args += "-Verbose" }
        
        & .\automated-tier-harness.ps1 @args
        
        if ($LASTEXITCODE -eq 0) {
            Write-MasterLog "✅ Full tier testing passed" "SUCCESS"
            return $true
        } else {
            Write-MasterLog "❌ Full tier testing failed" "ERROR"
            return $false
        }
        
    } catch {
        Write-MasterLog "❌ Full mode error: $($_.Exception.Message)" "ERROR"
        return $false
    }
}

function Test-StressMode {
    Write-MasterLog "=== STRESS MODE: Load Testing ===" "HEADER"
    
    try {
        if ($Tier -eq "ALL") {
            $testTiers = @("FREE", "PRO", "ENTERPRISE")
        } else {
            $testTiers = @($Tier)
        }
        
        $allPassed = $true
        
        foreach ($testTier in $testTiers) {
            Write-MasterLog "Stress testing $testTier tier..." "INFO"
            
            & .\stress-test-runner.ps1 -Tier $testTier -Duration $Duration
            
            if ($LASTEXITCODE -ne 0) {
                Write-MasterLog "❌ Stress test failed for $testTier" "ERROR"
                $allPassed = $false
            } else {
                Write-MasterLog "✅ Stress test passed for $testTier" "SUCCESS"
            }
        }
        
        return $allPassed
        
    } catch {
        Write-MasterLog "❌ Stress mode error: $($_.Exception.Message)" "ERROR"
        return $false
    }
}

function Test-CIMode {
    Write-MasterLog "=== CI MODE: Build Validation ===" "HEADER"
    
    try {
        Write-MasterLog "Running CI/CD validation..." "INFO"
        
        $args = @()
        if ($GenerateReport) { 
            $args += "-OutputFormat", "json"
            $args += "-OutputFile", $OutputFile
        }
        
        & .\ci-cd-validation.ps1 @args
        
        if ($LASTEXITCODE -eq 0) {
            Write-MasterLog "✅ CI/CD validation passed - Build should proceed" "SUCCESS"
            return $true
        } else {
            Write-MasterLog "❌ CI/CD validation failed - Build should be blocked" "ERROR"
            return $false
        }
        
    } catch {
        Write-MasterLog "❌ CI mode error: $($_.Exception.Message)" "ERROR"
        return $false
    }
}

function Test-E2EMode {
    Write-MasterLog "=== E2E MODE: End-to-End Pipeline Demo ===" "HEADER"
    
    try {
        if ($Tier -eq "ALL") {
            $testTier = "ENTERPRISE"  # Default to enterprise for E2E
        } else {
            $testTier = $Tier
        }
        
        Write-MasterLog "Running E2E demo for $testTier tier..." "INFO"
        
        $args = @("-Tier", $testTier, "-GenerateLicense")
        if ($Verbose) { $args += "-Verbose" }
        
        & .\e2e-flow-demo.ps1 @args
        
        if ($LASTEXITCODE -eq 0) {
            Write-MasterLog "✅ E2E pipeline demo completed successfully" "SUCCESS"
            return $true
        } else {
            Write-MasterLog "❌ E2E pipeline demo failed" "ERROR"
            return $false
        }
        
    } catch {
        Write-MasterLog "❌ E2E mode error: $($_.Exception.Message)" "ERROR"
        return $false
    }
}

function New-TestReport {
    param($Results)
    
    if (-not $GenerateReport) {
        return
    }
    
    Write-MasterLog "Generating test report..." "INFO"
    
    $report = @{
        timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
        mode = $Mode
        tier = $Tier
        duration = $Duration
        results = $Results
        environment = @{
            os = $env:OS
            computername = $env:COMPUTERNAME
            user = $env:USERNAME
            powershell_version = $PSVersionTable.PSVersion.ToString()
        }
    }
    
    try {
        $report | ConvertTo-Json -Depth 5 | Out-File $OutputFile -Encoding UTF8
        Write-MasterLog "✅ Report generated: $OutputFile" "SUCCESS"
    } catch {
        Write-MasterLog "⚠️ Failed to generate report: $($_.Exception.Message)" "WARN"
    }
}

# Main execution
try {
    $startTime = Get-Date
    
    Write-MasterLog "=========================================" "HEADER"
    Write-MasterLog "BITCOIN SPRINT MASTER TEST RUNNER" "HEADER"
    Write-MasterLog "=========================================" "HEADER"
    Write-MasterLog "Mode: $Mode, Tier: $Tier, Duration: ${Duration}s" "INFO"
    Write-MasterLog ""
    
    $result = switch ($Mode) {
        "quick" { Test-QuickMode }
        "full"  { Test-FullMode }
        "stress" { Test-StressMode }
        "ci"    { Test-CIMode }
        "e2e"   { Test-E2EMode }
        "help"  { Show-Help; $true }
        default { 
            Write-MasterLog "❌ Unknown mode: $Mode" "ERROR"
            Show-Help
            $false
        }
    }
    
    $endTime = Get-Date
    $totalDuration = ($endTime - $startTime).TotalSeconds
    
    # Generate report if requested
    if ($Mode -ne "help") {
        $testResults = @{
            success = $result
            duration_seconds = $totalDuration
            start_time = $startTime.ToString("yyyy-MM-dd HH:mm:ss")
            end_time = $endTime.ToString("yyyy-MM-dd HH:mm:ss")
        }
        
        New-TestReport $testResults
    }
    
    Write-MasterLog ""
    Write-MasterLog "=========================================" "HEADER"
    if ($Mode -ne "help") {
        if ($result) {
            Write-MasterLog "🎉 TEST RUN COMPLETED SUCCESSFULLY" "SUCCESS"
            Write-MasterLog "Duration: $([math]::Round($totalDuration, 1)) seconds" "INFO"
        } else {
            Write-MasterLog "💥 TEST RUN FAILED" "ERROR"
            Write-MasterLog "Duration: $([math]::Round($totalDuration, 1)) seconds" "INFO"
        }
    }
    Write-MasterLog "=========================================" "HEADER"
    
    exit $(if($result){0}else{1})
    
} catch {
    Write-MasterLog "💥 MASTER TEST RUNNER CRASHED: $($_.Exception.Message)" "ERROR"
    exit 2
} finally {
    # Cleanup any running processes
    Get-Process -Name "*sprint*" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
}
