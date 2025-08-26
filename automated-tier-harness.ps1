#!/usr/bin/env pwsh
#
# Automated Tier Testing Harness
# Switches between FREE/PRO/ENTERPRISE configs and validates performance targets
#

param(
	[switch]$SkipBuild,
	[switch]$StressTest,
	[int]$StressSeconds = 30,
	[switch]$Verbose
)

# Performance targets (ms)
$PERFORMANCE_TARGETS = @{
	"FREE"       = @{ 
		min_avg        = 100; 
		max_avg        = 1000; 
		rate_limit     = 20;
		poll_interval  = 8;
		turbo_expected = $false 
	}
	"PRO"        = @{ 
		min_avg        = 50; 
		max_avg        = 100; 
		rate_limit     = 10;
		poll_interval  = 2;
		turbo_expected = $false 
	}
	"ENTERPRISE" = @{ 
		min_avg        = 10; 
		max_avg        = 50; 
		rate_limit     = 2000;
		poll_interval  = 1;
		turbo_expected = $true 
	}
}

$TIER_CONFIGS = @{
	"FREE"       = "config-free-stable.json"
	"PRO"        = "config.json"  # Default PRO config
	"ENTERPRISE" = "config-enterprise-turbo.json"
}

$TIER_BINARIES = @{
	"FREE"       = "bitcoin-sprint-free.exe"
	"PRO"        = "bitcoin-sprint.exe"
	"ENTERPRISE" = "bitcoin-sprint-turbo.exe"
}

function Write-TestLog {
	param($Message, $Color = "White")
	$timestamp = Get-Date -Format "HH:mm:ss"
	Write-Host "[$timestamp] $Message" -ForegroundColor $Color
}

function Stop-SprintProcess {
	Write-TestLog "Stopping any running Sprint processes..." "Yellow"
	Get-Process -Name "*sprint*" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
	Start-Sleep 2
}

function Start-MockBitcoinCore {
	Write-TestLog "Ensuring Bitcoin Core mock is running..." "Cyan"
    
	$mockRunning = Get-NetTCPConnection -LocalPort 8332 -State Listen -ErrorAction SilentlyContinue
	if (-not $mockRunning) {
		Write-TestLog "Starting Bitcoin Core mock..." "Gray"
		Start-Process -FilePath "python" -ArgumentList "scripts\bitcoin-core-mock.py" -WindowStyle Hidden
		Start-Sleep 3
        
		$mockRunning = Get-NetTCPConnection -LocalPort 8332 -State Listen -ErrorAction SilentlyContinue
		if (-not $mockRunning) {
			throw "Failed to start Bitcoin Core mock on port 8332"
		}
	}
	Write-TestLog "✓ Bitcoin Core mock ready on port 8332" "Green"
}

function Switch-ToTier {
	param($TierName)
    
	Write-TestLog "=== SWITCHING TO $TierName TIER ===" "Cyan"
    
	# Stop current Sprint
	Stop-SprintProcess
    
	# Copy tier config to active config
	$sourceConfig = $TIER_CONFIGS[$TierName]
	if (-not (Test-Path $sourceConfig)) {
		throw "Config file not found: $sourceConfig"
	}
    
	Write-TestLog "Copying $sourceConfig → config.json" "Gray"
	Copy-Item $sourceConfig "config.json" -Force
    
	# Verify config was applied
	$config = Get-Content "config.json" | ConvertFrom-Json
	Write-TestLog "Config applied: tier=$($config.tier) turbo=$($config.turbo_mode) poll=$($config.poll_interval)s" "Gray"
    
	# Set environment for this tier
	$env:RPC_NODES = "http://127.0.0.1:8332"
	$env:RPC_USER = "bitcoin"
	$env:RPC_PASS = "sprint123benchmark"
	$env:API_PORT = "8080"
	$env:TURBO_MODE = if ($config.turbo_mode) { "true" } else { "false" }
    
	# Use tier-specific binary if available
	$binary = $TIER_BINARIES[$TierName]
	if (-not (Test-Path $binary)) {
		$binary = "bitcoin-sprint.exe"  # Fallback to default
	}
    
	Write-TestLog "Starting Sprint with $binary..." "Gray"
    
	# Start Sprint in background
	$sprintJob = Start-Job -ScriptBlock {
		param($BinaryPath, $WorkDir)
		Set-Location $WorkDir
		& $BinaryPath
	} -ArgumentList (Resolve-Path $binary), (Get-Location)
    
	# Wait for Sprint to be ready
	$maxWait = 15
	$waited = 0
	while ($waited -lt $maxWait) {
		Start-Sleep 1
		$waited++
        
		$sprintRunning = Get-NetTCPConnection -LocalPort 8080 -State Listen -ErrorAction SilentlyContinue
		if ($sprintRunning) {
			Write-TestLog "✓ Sprint ($TierName) ready on port 8080" "Green"
			return $sprintJob
		}
	}
    
	throw "Sprint failed to start within $maxWait seconds"
}

function Test-ApiPerformance {
	param($TierName, $RequestCount = 50)
    
	Write-TestLog "Testing $TierName performance ($RequestCount requests)..." "Yellow"
    
	$responseTimes = @()
	$errors = 0
    
	for ($i = 1; $i -le $RequestCount; $i++) {
		try {
			$stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
			$response = Invoke-RestMethod -Uri "http://localhost:8080/status" -TimeoutSec 5
			$stopwatch.Stop()
            
			$responseTimes += $stopwatch.ElapsedMilliseconds
            
			if ($Verbose) {
				Write-Host "  Request $i`: $($stopwatch.ElapsedMilliseconds)ms" -ForegroundColor Gray
			}
		}
		catch {
			$errors++
			if ($Verbose) {
				Write-Host "  Request $i`: ERROR - $($_.Exception.Message)" -ForegroundColor Red
			}
		}
        
		# Brief pause between requests
		Start-Sleep -Milliseconds 50
	}
    
	if ($responseTimes.Count -eq 0) {
		throw "All requests failed for $TierName tier"
	}
    
	$avgTime = ($responseTimes | Measure-Object -Average).Average
	$minTime = ($responseTimes | Measure-Object -Minimum).Minimum
	$maxTime = ($responseTimes | Measure-Object -Maximum).Maximum
    
	Write-TestLog "Performance Results for $TierName`:" "White"
	Write-TestLog "  Requests: $RequestCount, Errors: $errors" "Gray"
	Write-TestLog "  Avg: $([math]::Round($avgTime, 1))ms, Min: $($minTime)ms, Max: $($maxTime)ms" "Gray"
    
	return @{
		TierName     = $TierName
		Average      = $avgTime
		Minimum      = $minTime
		Maximum      = $maxTime
		Errors       = $errors
		RequestCount = $RequestCount
	}
}

function Test-StressLoad {
	param($TierName, $DurationSeconds = 30)
    
	Write-TestLog "Running stress test for $TierName ($DurationSeconds seconds)..." "Yellow"
    
	# Check if bombardier is available
	$bombardierAvailable = $false
	try {
		$null = Get-Command bombardier -ErrorAction Stop
		$bombardierAvailable = $true
	}
 catch {
		Write-TestLog "bombardier not found, using PowerShell stress test" "Yellow"
	}
    
	if ($bombardierAvailable) {
		# Use bombardier for proper load testing
		Write-TestLog "Using bombardier for load testing..." "Gray"
		$output = & bombardier -c 10 -d "$($DurationSeconds)s" -l "http://localhost:8080/status" 2>&1
		Write-TestLog "Bombardier output:" "Gray"
		$output | ForEach-Object { Write-TestLog "  $_" "Gray" }
	}
 else {
		# Fallback to PowerShell concurrent requests
		Write-TestLog "Using PowerShell concurrent stress test..." "Gray"
        
		$jobs = @()
		$startTime = Get-Date
        
		# Start 5 concurrent workers
		for ($worker = 1; $worker -le 5; $worker++) {
			$job = Start-Job -ScriptBlock {
				param($WorkerID, $DurationSeconds)
                
				$endTime = (Get-Date).AddSeconds($DurationSeconds)
				$requests = 0
				$errors = 0
                
				while ((Get-Date) -lt $endTime) {
					try {
						$response = Invoke-RestMethod -Uri "http://localhost:8080/status" -TimeoutSec 2
						$requests++
					}
					catch {
						$errors++
					}
					Start-Sleep -Milliseconds 100
				}
                
				return @{ Worker = $WorkerID; Requests = $requests; Errors = $errors }
			} -ArgumentList $worker, $DurationSeconds
            
			$jobs += $job
		}
        
		# Wait for completion
		$results = $jobs | Wait-Job | Receive-Job
		$jobs | Remove-Job
        
		$totalRequests = ($results | Measure-Object -Property Requests -Sum).Sum
		$totalErrors = ($results | Measure-Object -Property Errors -Sum).Sum
        
		Write-TestLog "Stress test results:" "White"
		Write-TestLog "  Total requests: $totalRequests" "Gray"
		Write-TestLog "  Total errors: $totalErrors" "Gray"
		Write-TestLog "  Error rate: $([math]::Round($totalErrors * 100 / $totalRequests, 2))%" "Gray"
		Write-TestLog "  Requests/sec: $([math]::Round($totalRequests / $DurationSeconds, 1))" "Gray"
	}
}

function Validate-TierPerformance {
	param($Results)
    
	$tier = $Results.TierName
	$targets = $PERFORMANCE_TARGETS[$tier]
    
	Write-TestLog "=== VALIDATING $tier PERFORMANCE ===" "Cyan"
    
	$passed = $true
	$avgTime = $Results.Average
    
	# Check average response time
	if ($avgTime -lt $targets.min_avg) {
		Write-TestLog "⚠️  Average response time ($([math]::Round($avgTime, 1))ms) is suspiciously fast (< $($targets.min_avg)ms)" "Yellow"
	}
 elseif ($avgTime -gt $targets.max_avg) {
		Write-TestLog "❌ Average response time ($([math]::Round($avgTime, 1))ms) exceeds target (> $($targets.max_avg)ms)" "Red"
		$passed = $false
	}
 else {
		Write-TestLog "✅ Average response time ($([math]::Round($avgTime, 1))ms) within target ($($targets.min_avg)-$($targets.max_avg)ms)" "Green"
	}
    
	# Check error rate
	$errorRate = $Results.Errors * 100 / $Results.RequestCount
	if ($errorRate -gt 5) {
		Write-TestLog "❌ Error rate too high: $([math]::Round($errorRate, 1))%" "Red"
		$passed = $false
	}
 else {
		Write-TestLog "✅ Error rate acceptable: $([math]::Round($errorRate, 1))%" "Green"
	}
    
	return $passed
}

function Test-TierConfiguration {
	param($TierName)
    
	Write-TestLog "Validating $TierName configuration..." "Yellow"
    
	try {
		# Test Sprint API status
		$response = Invoke-RestMethod -Uri "http://localhost:8080/status" -TimeoutSec 5
        
		$targets = $PERFORMANCE_TARGETS[$TierName]
		$configValid = $true
        
		# Check turbo mode
		if ($response.turbo_mode_enabled -ne $targets.turbo_expected) {
			Write-TestLog "❌ Turbo mode mismatch: expected $($targets.turbo_expected), got $($response.turbo_mode_enabled)" "Red"
			$configValid = $false
		}
		else {
			Write-TestLog "✅ Turbo mode correct: $($response.turbo_mode_enabled)" "Green"
		}
        
		# Check Bitcoin connection
		if (-not $response.bitcoin_connected) {
			Write-TestLog "❌ Bitcoin Core not connected" "Red"
			$configValid = $false
		}
		else {
			Write-TestLog "✅ Bitcoin Core connected" "Green"
		}
        
		return $configValid
	}
	catch {
		Write-TestLog "❌ Failed to validate $TierName configuration: $($_.Exception.Message)" "Red"
		return $false
	}
}

# Main execution
try {
	Write-TestLog "=========================================" "Cyan"
	Write-TestLog "AUTOMATED TIER TESTING HARNESS" "Cyan"
	Write-TestLog "=========================================" "Cyan"
    
	# Ensure Bitcoin Core mock is running
	Start-MockBitcoinCore
    
	$allResults = @()
	$allPassed = $true
    
	# Test each tier
	foreach ($tier in @("FREE", "PRO", "ENTERPRISE")) {
		try {
			Write-TestLog ""
            
			# Switch to tier
			$sprintJob = Switch-ToTier $tier
            
			# Wait for stabilization
			Start-Sleep 3
            
			# Validate configuration
			$configValid = Test-TierConfiguration $tier
			if (-not $configValid) {
				Write-TestLog "❌ Configuration validation failed for $tier" "Red"
				$allPassed = $false
				continue
			}
            
			# Performance test
			$perfResults = Test-ApiPerformance $tier 30
			$allResults += $perfResults
            
			# Validate performance against targets
			$perfPassed = Validate-TierPerformance $perfResults
			if (-not $perfPassed) {
				$allPassed = $false
			}
            
			# Optional stress test
			if ($StressTest) {
				Test-StressLoad $tier $StressSeconds
			}
            
			# Clean up
			if ($sprintJob) {
				Stop-Job $sprintJob -ErrorAction SilentlyContinue
				Remove-Job $sprintJob -ErrorAction SilentlyContinue
			}
            
		}
		catch {
			Write-TestLog "❌ TIER $tier FAILED: $($_.Exception.Message)" "Red"
			$allPassed = $false
		}
	}
    
	# Final results summary
	Write-TestLog ""
	Write-TestLog "=========================================" "Cyan"
	Write-TestLog "FINAL RESULTS SUMMARY" "Cyan"
	Write-TestLog "=========================================" "Cyan"
    
	foreach ($result in $allResults) {
		$targets = $PERFORMANCE_TARGETS[$result.TierName]
		$status = if ($result.Average -le $targets.max_avg) { "✅ PASS" } else { "❌ FAIL" }
		Write-TestLog "$($result.TierName): $([math]::Round($result.Average, 1))ms avg, $($result.Errors) errors - $status" $(if ($result.Average -le $targets.max_avg) { "Green" } else { "Red" })
	}
    
	Write-TestLog ""
	if ($allPassed) {
		Write-TestLog "🎉 ALL TIERS PASSED - Ready for CI/CD!" "Green"
		exit 0
	}
 else {
		Write-TestLog "💥 SOME TESTS FAILED - Build should fail!" "Red"
		exit 1
	}
}
catch {
	Write-TestLog "💥 HARNESS FAILED: $($_.Exception.Message)" "Red"
	exit 1
}
finally {
	# Cleanup
	Stop-SprintProcess
}
