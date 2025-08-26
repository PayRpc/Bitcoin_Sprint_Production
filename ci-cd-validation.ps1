#!/usr/bin/env pwsh
#
# CI/CD Build Validation Script
# Validates that all tiers meet performance requirements before build passes
#

param(
	[switch]$SkipStress,
	[string]$OutputFormat = "junit",  # junit, json, console
	[string]$OutputFile = "test-results.xml"
)

# Exit codes for CI/CD
$EXIT_SUCCESS = 0
$EXIT_FAILURE = 1
$EXIT_CONFIGURATION_ERROR = 2

# Performance SLAs (Service Level Agreements)
$SLA_TARGETS = @{
	"FREE"       = @{
		max_response_time = 1000    # 1 second max
		max_error_rate    = 5         # 5% max error rate
		min_uptime        = 95            # 95% uptime required
	}
	"PRO"        = @{
		max_response_time = 100    # 100ms max
		max_error_rate    = 2         # 2% max error rate  
		min_uptime        = 99            # 99% uptime required
	}
	"ENTERPRISE" = @{
		max_response_time = 50     # 50ms max (with turbo cache < 10ms bursts)
		max_error_rate    = 1         # 1% max error rate
		min_uptime        = 99.9          # 99.9% uptime required
	}
}

$testResults = @()

function Write-CILog {
	param($Message, $Level = "INFO")
	$timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
	$color = switch ($Level) {
		"ERROR" { "Red" }
		"WARN" { "Yellow" }
		"INFO" { "White" }
		"SUCCESS" { "Green" }
		default { "Gray" }
	}
	Write-Host "[$timestamp] [$Level] $Message" -ForegroundColor $color
}

function Test-Prerequisites {
	Write-CILog "Checking build prerequisites..." "INFO"
    
	$errors = @()
    
	# Check required files
	$requiredFiles = @(
		"bitcoin-sprint.exe",
		"config.json",
		"config-free.json", 
		"config-pro.json",
		"config-enterprise-turbo.json",
		"scripts\bitcoin-core-mock.py"
	)
    
	foreach ($file in $requiredFiles) {
		if (-not (Test-Path $file)) {
			$errors += "Missing required file: $file"
		}
	}
    
	# Check Go installation (for potential tools)
	try {
		$goVersion = & go version
		Write-CILog "Go found: $goVersion" "INFO"
	}
 catch {
		Write-CILog "Go not found - some tools may not be available" "WARN"
	}
    
	# Check Python for mock
	try {
		$pythonVersion = & python --version
		Write-CILog "Python found: $pythonVersion" "INFO"
	}
 catch {
		$errors += "Python not found - required for Bitcoin Core mock"
	}
    
	if ($errors.Count -gt 0) {
		foreach ($error in $errors) {
			Write-CILog $error "ERROR"
		}
		return $false
	}
    
	Write-CILog "✅ All prerequisites met" "SUCCESS"
	return $true
}

function Invoke-TierValidation {
	param($TierName)
    
	Write-CILog "=== VALIDATING $TierName TIER ===" "INFO"
    
	$testResult = [PSCustomObject]@{
		TierName  = $TierName
		TestName  = "$TierName Tier Validation"
		StartTime = Get-Date
		EndTime   = $null
		Duration  = $null
		Status    = "RUNNING"
		Errors    = @()
		Metrics   = @{}
	}
    
	try {
		# Start Bitcoin Core mock if not running
		if (-not (Get-NetTCPConnection -LocalPort 8332 -State Listen -ErrorAction SilentlyContinue)) {
			Write-CILog "Starting Bitcoin Core mock..." "INFO"
			$mockJob = Start-Job -ScriptBlock { 
				Set-Location $args[0]
				python scripts\bitcoin-core-mock.py 
			} -ArgumentList (Get-Location)
			Start-Sleep 3
		}
        
		# Run the automated tier harness for this specific tier
		Write-CILog "Running automated validation for $TierName..." "INFO"
        
		$harnessScript = ".\automated-tier-harness.ps1"
		if (-not (Test-Path $harnessScript)) {
			throw "Automated tier harness script not found"
		}
        
		# Capture harness output
		$harnessOutput = & $harnessScript -SkipBuild 2>&1
		$harnessExitCode = $LASTEXITCODE
        
		if ($harnessExitCode -eq 0) {
			$testResult.Status = "PASSED"
			Write-CILog "✅ $TierName validation passed" "SUCCESS"
		}
		else {
			$testResult.Status = "FAILED"
			$testResult.Errors += "Harness validation failed with exit code $harnessExitCode"
			Write-CILog "❌ $TierName validation failed" "ERROR"
		}
        
		# Extract metrics from output (basic parsing)
		$responseTimeMatch = $harnessOutput | Select-String "(\d+\.?\d*)ms avg"
		if ($responseTimeMatch) {
			$testResult.Metrics["average_response_time"] = [double]$responseTimeMatch.Matches[0].Groups[1].Value
		}
        
		$errorMatch = $harnessOutput | Select-String "(\d+) errors"
		if ($errorMatch) {
			$testResult.Metrics["error_count"] = [int]$errorMatch.Matches[0].Groups[1].Value
		}
        
	}
 catch {
		$testResult.Status = "FAILED"
		$testResult.Errors += $_.Exception.Message
		Write-CILog "❌ Exception during $TierName validation: $($_.Exception.Message)" "ERROR"
	}
    
	$testResult.EndTime = Get-Date
	$testResult.Duration = ($testResult.EndTime - $testResult.StartTime).TotalSeconds
    
	return $testResult
}

function Test-EndToEndFlow {
	Write-CILog "=== TESTING END-TO-END FLOW ===" "INFO"
    
	$testResult = [PSCustomObject]@{
		TierName  = "E2E"
		TestName  = "End-to-End Flow Validation"
		StartTime = Get-Date
		EndTime   = $null
		Duration  = $null
		Status    = "RUNNING"
		Errors    = @()
		Metrics   = @{}
	}
    
	try {
		# 1. Generate/validate license
		Write-CILog "Step 1: License validation..." "INFO"
		if (Test-Path "license-enterprise.json") {
			$license = Get-Content "license-enterprise.json" | ConvertFrom-Json
			if ($license.features.turbo_mode) {
				Write-CILog "✅ Enterprise license with turbo_mode found" "SUCCESS"
			}
			else {
				$testResult.Errors += "Enterprise license missing turbo_mode feature"
			}
		}
		else {
			Write-CILog "⚠️ No enterprise license found - using default config" "WARN"
		}
        
		# 2. Validate with API
		Write-CILog "Step 2: API validation..." "INFO"
		try {
			$response = Invoke-RestMethod -Uri "http://localhost:8080/status" -TimeoutSec 5
			Write-CILog "✅ API responds correctly" "SUCCESS"
            
			# 3. Query Sprint status
			Write-CILog "Step 3: Sprint status check..." "INFO"
			if ($response.bitcoin_connected) {
				Write-CILog "✅ Sprint connected to Bitcoin Core" "SUCCESS"
			}
			else {
				$testResult.Errors += "Sprint not connected to Bitcoin Core"
			}
            
			# 4. Confirm block updates
			Write-CILog "Step 4: Block data validation..." "INFO"
			if ($response.latest_block -and $response.latest_block -gt 0) {
				Write-CILog "✅ Block data available: block $($response.latest_block)" "SUCCESS"
			}
			else {
				$testResult.Errors += "No block data available"
			}
            
			# 5. Measure response time
			Write-CILog "Step 5: Performance measurement..." "INFO"
			$stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
			$null = Invoke-RestMethod -Uri "http://localhost:8080/status" -TimeoutSec 5
			$stopwatch.Stop()
            
			$responseTime = $stopwatch.ElapsedMilliseconds
			$testResult.Metrics["response_time"] = $responseTime
			Write-CILog "✅ Response time: ${responseTime}ms" "SUCCESS"
            
			if ($testResult.Errors.Count -eq 0) {
				$testResult.Status = "PASSED"
				Write-CILog "✅ End-to-end flow validation passed" "SUCCESS"
			}
			else {
				$testResult.Status = "FAILED"
			}
            
		}
		catch {
			$testResult.Status = "FAILED"
			$testResult.Errors += "API validation failed: $($_.Exception.Message)"
		}
        
	}
 catch {
		$testResult.Status = "FAILED"
		$testResult.Errors += $_.Exception.Message
		Write-CILog "❌ Exception during E2E test: $($_.Exception.Message)" "ERROR"
	}
    
	$testResult.EndTime = Get-Date
	$testResult.Duration = ($testResult.EndTime - $testResult.StartTime).TotalSeconds
    
	return $testResult
}

function Export-TestResults {
	param($Results, $Format, $OutputFile)
    
	switch ($Format.ToLower()) {
		"junit" {
			Export-JUnitResults $Results $OutputFile
		}
		"json" {
			$Results | ConvertTo-Json -Depth 5 | Out-File $OutputFile -Encoding UTF8
			Write-CILog "Results exported to $OutputFile (JSON)" "INFO"
		}
		"console" {
			# Already displayed to console
		}
		default {
			Write-CILog "Unknown output format: $Format" "WARN"
		}
	}
}

function Export-JUnitResults {
	param($Results, $OutputFile)
    
	$totalTests = $Results.Count
	$failedTests = ($Results | Where-Object { $_.Status -eq "FAILED" }).Count
	$totalTime = ($Results | Measure-Object -Property Duration -Sum).Sum
    
	$xml = @"
<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="Bitcoin Sprint Tier Validation" tests="$totalTests" failures="$failedTests" time="$totalTime">
"@
    
	foreach ($result in $Results) {
		$className = "BitcoinSprint.$($result.TierName)"
		$testName = $result.TestName -replace '\s+', ''
        
		$xml += @"

    <testcase classname="$className" name="$testName" time="$($result.Duration)">
"@
        
		if ($result.Status -eq "FAILED") {
			$errorMessage = $result.Errors -join "; "
			$xml += @"

        <failure message="$errorMessage">$errorMessage</failure>
"@
		}
        
		$xml += @"

    </testcase>
"@
	}
    
	$xml += @"

</testsuite>
"@
    
	$xml | Out-File $OutputFile -Encoding UTF8
	Write-CILog "JUnit results exported to $OutputFile" "INFO"
}

# Main CI/CD validation execution
try {
	Write-CILog "=========================================" "INFO"
	Write-CILog "CI/CD BUILD VALIDATION STARTING" "INFO"
	Write-CILog "=========================================" "INFO"
    
	# Check prerequisites
	if (-not (Test-Prerequisites)) {
		Write-CILog "💥 Prerequisites check failed" "ERROR"
		exit $EXIT_CONFIGURATION_ERROR
	}
    
	# Validate each tier
	$tiers = @("FREE", "PRO", "ENTERPRISE")
	foreach ($tier in $tiers) {
		$result = Invoke-TierValidation $tier
		$testResults += $result
	}
    
	# Run end-to-end flow test
	$e2eResult = Test-EndToEndFlow
	$testResults += $e2eResult
    
	# Optional stress testing
	if (-not $SkipStress) {
		Write-CILog "Running stress tests..." "INFO"
		try {
			& .\stress-test-runner.ps1 -Tier "ENTERPRISE" -Duration 10
			if ($LASTEXITCODE -eq 0) {
				Write-CILog "✅ Stress test passed" "SUCCESS"
			}
			else {
				Write-CILog "❌ Stress test failed" "ERROR"
				$failedTests += 1
			}
		}
		catch {
			Write-CILog "❌ Stress test error: $($_.Exception.Message)" "ERROR"
		}
	}
    
	# Generate test report
	Export-TestResults $testResults $OutputFormat $OutputFile
    
	# Final assessment
	$failedTests = ($testResults | Where-Object { $_.Status -eq "FAILED" }).Count
	$totalTests = $testResults.Count
	$passedTests = $totalTests - $failedTests
    
	Write-CILog ""
	Write-CILog "=========================================" "INFO"
	Write-CILog "CI/CD VALIDATION RESULTS" "INFO"
	Write-CILog "=========================================" "INFO"
	Write-CILog "Total Tests: $totalTests" "INFO"
	Write-CILog "Passed: $passedTests" "SUCCESS"
	Write-CILog "Failed: $failedTests" $(if ($failedTests -gt 0) { "ERROR" }else { "SUCCESS" })
    
	if ($failedTests -eq 0) {
		Write-CILog "🎉 BUILD VALIDATION PASSED - All tiers meet SLA requirements!" "SUCCESS"
		exit $EXIT_SUCCESS
	}
 else {
		Write-CILog "💥 BUILD VALIDATION FAILED - Performance targets not met!" "ERROR"
        
		# Show failed test details
		$failedResults = $testResults | Where-Object { $_.Status -eq "FAILED" }
		foreach ($failed in $failedResults) {
			Write-CILog "❌ $($failed.TestName): $($failed.Errors -join '; ')" "ERROR"
		}
        
		exit $EXIT_FAILURE
	}
    
}
catch {
	Write-CILog "💥 CI/CD VALIDATION CRASHED: $($_.Exception.Message)" "ERROR"
	exit $EXIT_CONFIGURATION_ERROR
}
finally {
	# Cleanup any running processes
	Get-Process -Name "*sprint*" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
	Write-CILog "Cleanup completed" "INFO"
}
