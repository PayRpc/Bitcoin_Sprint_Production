#!/usr/bin/env pwsh
#
# Stress Test Runner using Bombardier
# Tests rate limiting and performance under load
#

param(
	[string]$Tier = "ENTERPRISE",
	[int]$Duration = 30,
	[int]$Connections = 10,
	[string]$Endpoint = "http://localhost:8080/status"
)

$TIER_CONFIGS = @{
	"FREE"       = @{ binary = "bitcoin-sprint-free.exe"; config = "config-free.json"; expected_rps = 0.5 }
	"PRO"        = @{ binary = "bitcoin-sprint.exe"; config = "config-pro.json"; expected_rps = 2.0 }
	"ENTERPRISE" = @{ binary = "bitcoin-sprint-turbo.exe"; config = "config-enterprise-turbo.json"; expected_rps = 50.0 }
}

function Write-StressLog {
	param($Message, $Color = "White")
	$timestamp = Get-Date -Format "HH:mm:ss"
	Write-Host "[$timestamp] $Message" -ForegroundColor $Color
}

function Install-Bombardier {
	Write-StressLog "Checking for bombardier..." "Yellow"
    
	try {
		$null = Get-Command bombardier -ErrorAction Stop
		Write-StressLog "✓ bombardier found" "Green"
		return $true
	}
 catch {
		Write-StressLog "bombardier not found, attempting to install..." "Yellow"
        
		try {
			# Try to install via Go
			if (Get-Command go -ErrorAction SilentlyContinue) {
				Write-StressLog "Installing bombardier via Go..." "Gray"
				& go install github.com/codesenberg/bombardier@latest
                
				# Check if it's in PATH now
				$goPath = & go env GOPATH
				$bombardierPath = Join-Path $goPath "bin\bombardier.exe"
				if (Test-Path $bombardierPath) {
					Write-StressLog "✓ bombardier installed to $bombardierPath" "Green"
					$env:PATH += ";$(Join-Path $goPath 'bin')"
					return $true
				}
			}
            
			# Fallback: Download binary directly
			Write-StressLog "Downloading bombardier binary..." "Gray"
			$downloadUrl = "https://github.com/codesenberg/bombardier/releases/download/v1.2.6/bombardier-windows-amd64.exe"
			Invoke-WebRequest -Uri $downloadUrl -OutFile "bombardier.exe"
            
			if (Test-Path "bombardier.exe") {
				Write-StressLog "✓ bombardier downloaded to current directory" "Green"
				return $true
			}
            
		}
		catch {
			Write-StressLog "❌ Failed to install bombardier: $($_.Exception.Message)" "Red"
			return $false
		}
	}
    
	return $false
}

function Start-TierService {
	param($TierName)
    
	$tierConfig = $TIER_CONFIGS[$TierName]
	if (-not $tierConfig) {
		throw "Unknown tier: $TierName"
	}
    
	Write-StressLog "Starting $TierName tier service..." "Cyan"
    
	# Stop existing processes
	Get-Process -Name "*sprint*" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
	Start-Sleep 2
    
	# Copy tier config
	$configFile = $tierConfig.config
	if (-not (Test-Path $configFile)) {
		throw "Config file not found: $configFile"
	}
    
	Copy-Item $configFile "config.json" -Force
	Write-StressLog "Applied config: $configFile" "Gray"
    
	# Set environment
	$env:RPC_NODES = "http://127.0.0.1:8332"
	$env:RPC_USER = "bitcoin"
	$env:RPC_PASS = "sprint123benchmark"
	$env:API_PORT = "8080"
    
	# Start service
	$binary = $tierConfig.binary
	if (-not (Test-Path $binary)) {
		$binary = "bitcoin-sprint.exe"  # Fallback
	}
    
	Write-StressLog "Starting $binary..." "Gray"
	$job = Start-Job -ScriptBlock {
		param($BinaryPath, $WorkDir)
		Set-Location $WorkDir
		& $BinaryPath
	} -ArgumentList (Resolve-Path $binary), (Get-Location)
    
	# Wait for service to be ready
	$maxWait = 15
	for ($i = 0; $i -lt $maxWait; $i++) {
		Start-Sleep 1
		if (Get-NetTCPConnection -LocalPort 8080 -State Listen -ErrorAction SilentlyContinue) {
			Write-StressLog "✓ Service ready on port 8080" "Green"
			return $job
		}
	}
    
	throw "Service failed to start within $maxWait seconds"
}

function Run-BombardierTest {
	param($Duration, $Connections, $Endpoint)
    
	Write-StressLog "Running bombardier stress test..." "Yellow"
	Write-StressLog "Duration: ${Duration}s, Connections: $Connections, Endpoint: $Endpoint" "Gray"
    
	# Check for local bombardier first
	$bombardierCmd = "bombardier"
	if (Test-Path "bombardier.exe") {
		$bombardierCmd = ".\bombardier.exe"
	}
    
	try {
		$output = & $bombardierCmd -c $Connections -d "${Duration}s" -l --print r --format json $Endpoint 2>&1
        
		# Try to parse JSON output
		try {
			$jsonOutput = $output | Where-Object { $_.StartsWith("{") } | ConvertFrom-Json
            
			Write-StressLog "=== BOMBARDIER RESULTS ===" "Cyan"
			Write-StressLog "Requests completed: $($jsonOutput.req.total)" "White"
			Write-StressLog "Requests/sec: $([math]::Round($jsonOutput.req.rps, 2))" "White"
			Write-StressLog "Average latency: $([math]::Round($jsonOutput.latencies.mean / 1000, 2))ms" "White"
			Write-StressLog "95th percentile: $([math]::Round($jsonOutput.latencies.p95 / 1000, 2))ms" "White"
			Write-StressLog "99th percentile: $([math]::Round($jsonOutput.latencies.p99 / 1000, 2))ms" "White"
			Write-StressLog "Errors: $($jsonOutput.errors.total)" "$(if($jsonOutput.errors.total -gt 0){'Red'}else{'Green'})"
            
			return $jsonOutput
		}
		catch {
			# Fallback to text parsing
			Write-StressLog "Raw bombardier output:" "Gray"
			$output | ForEach-Object { Write-StressLog "  $_" "Gray" }
			return $null
		}
        
	}
 catch {
		Write-StressLog "❌ Bombardier failed: $($_.Exception.Message)" "Red"
		return $null
	}
}

function Validate-StressResults {
	param($Results, $TierName)
    
	if (-not $Results) {
		Write-StressLog "❌ No results to validate" "Red"
		return $false
	}
    
	$tierConfig = $TIER_CONFIGS[$TierName]
	$expectedRps = $tierConfig.expected_rps
	$actualRps = $Results.req.rps
    
	Write-StressLog "=== VALIDATION FOR $TierName ===" "Cyan"
	Write-StressLog "Expected RPS: >= $expectedRps" "Gray"
	Write-StressLog "Actual RPS: $([math]::Round($actualRps, 2))" "Gray"
    
	if ($actualRps -ge $expectedRps) {
		Write-StressLog "✅ Performance target met" "Green"
		return $true
	}
 else {
		Write-StressLog "❌ Performance target not met" "Red"
		return $false
	}
}

# Main execution
try {
	Write-StressLog "=========================================" "Cyan"
	Write-StressLog "STRESS TEST RUNNER" "Cyan"
	Write-StressLog "=========================================" "Cyan"
    
	# Install bombardier if needed
	$bombardierReady = Install-Bombardier
	if (-not $bombardierReady) {
		throw "bombardier is required for stress testing"
	}
    
	# Ensure Bitcoin Core mock is running
	if (-not (Get-NetTCPConnection -LocalPort 8332 -State Listen -ErrorAction SilentlyContinue)) {
		Write-StressLog "Starting Bitcoin Core mock..." "Yellow"
		Start-Process -FilePath "python" -ArgumentList "scripts\bitcoin-core-mock.py" -WindowStyle Hidden
		Start-Sleep 3
	}
    
	# Start tier service
	$serviceJob = Start-TierService $Tier
    
	# Run stress test
	$results = Run-BombardierTest $Duration $Connections $Endpoint
    
	# Validate results
	$passed = Validate-StressResults $results $Tier
    
	if ($passed) {
		Write-StressLog "🎉 STRESS TEST PASSED" "Green"
		exit 0
	}
 else {
		Write-StressLog "💥 STRESS TEST FAILED" "Red"
		exit 1
	}
    
}
catch {
	Write-StressLog "💥 STRESS TEST ERROR: $($_.Exception.Message)" "Red"
	exit 1
}
finally {
	# Cleanup
	if ($serviceJob) {
		Stop-Job $serviceJob -ErrorAction SilentlyContinue
		Remove-Job $serviceJob -ErrorAction SilentlyContinue
	}
	Get-Process -Name "*sprint*" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
}
