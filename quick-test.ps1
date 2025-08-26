# Quick Bitcoin Sprint Testing Script
# Tests Bitcoin Sprint with different connection scenarios

param(
	[ValidateSet("mock", "local", "testnet", "regtest")]
	[string]$Mode = "mock",
	[switch]$Verbose
)

$ErrorActionPreference = "Stop"

Write-Host "🚀 Bitcoin Sprint Testing Environment" -ForegroundColor Cyan
Write-Host "Mode: $Mode" -ForegroundColor Green

function Test-BitcoinConnection {
	param($Url, $User, $Pass)
    
	Write-Host "🔍 Testing Bitcoin RPC at $Url..." -ForegroundColor Yellow
    
	try {
		$auth = [Convert]::ToBase64String([Text.Encoding]::ASCII.GetBytes("${User}:${Pass}"))
		$headers = @{ "Authorization" = "Basic $auth" }
		$body = @{
			jsonrpc = "1.0"
			id      = "test"
			method  = "getblockchaininfo"
			params  = @()
		} | ConvertTo-Json
        
		$response = Invoke-RestMethod -Uri $Url -Method POST -Body $body -ContentType "application/json" -Headers $headers -TimeoutSec 5
        
		if ($response.result) {
			Write-Host "✅ Bitcoin Core connected!" -ForegroundColor Green
			Write-Host "   Chain: $($response.result.chain)" -ForegroundColor Cyan
			Write-Host "   Blocks: $($response.result.blocks)" -ForegroundColor Cyan
			return $true
		}
		else {
			Write-Host "❌ Invalid response from Bitcoin Core" -ForegroundColor Red
			return $false
		}
	}
	catch {
		Write-Host "❌ Bitcoin Core not reachable: $($_.Exception.Message)" -ForegroundColor Red
		return $false
	}
}

function Start-BitcoinSprintTest {
	param($Mode)
    
	# Configuration based on mode
	$config = switch ($Mode) {
		"mock" {
			@{
				config_file = "config.json"
				rpc_nodes   = @("http://mock-bitcoin-node:8332")
				description = "Mock mode - no real Bitcoin Core needed"
			}
		}
		"local" {
			@{
				config_file = "config.json" 
				rpc_nodes   = @("http://localhost:8332")
				description = "Local Bitcoin Core on mainnet port"
			}
		}
		"testnet" {
			@{
				config_file = "config-testnet.json"
				rpc_nodes   = @("http://localhost:8332")
				description = "Testnet mode"
			}
		}
		"regtest" {
			@{
				config_file = "config-regtest.json"
				rpc_nodes   = @("http://localhost:18332")
				description = "Regtest mode - private blockchain"
			}
		}
	}
    
	Write-Host "📝 Configuration: $($config.description)" -ForegroundColor Yellow
    
	# Test Bitcoin connection if not mock mode
	if ($Mode -ne "mock") {
		$bitcoinWorking = Test-BitcoinConnection -Url $config.rpc_nodes[0] -User "test_user" -Pass "strong_random_password_here"
		if (!$bitcoinWorking) {
			Write-Host "⚠️  Bitcoin Core not available, but continuing anyway..." -ForegroundColor Yellow
			Write-Host "   Bitcoin Sprint will log connection errors but still start" -ForegroundColor Cyan
		}
	}
 else {
		Write-Host "🎭 Mock mode - skipping Bitcoin Core connection test" -ForegroundColor Cyan
	}
    
	# Build Bitcoin Sprint
	Write-Host "🔨 Building Bitcoin Sprint..." -ForegroundColor Yellow
	$buildArgs = @("build", "-o", "bitcoin-sprint-test.exe", "./cmd/sprint")
	$buildResult = & go @buildArgs
    
	if ($LASTEXITCODE -ne 0) {
		Write-Host "❌ Failed to build Bitcoin Sprint" -ForegroundColor Red
		return $false
	}
    
	Write-Host "✅ Build successful!" -ForegroundColor Green
    
	# Use appropriate config
	if (Test-Path $config.config_file) {
		Copy-Item $config.config_file "config.json" -Force
		Write-Host "📄 Using config: $($config.config_file)" -ForegroundColor Cyan
	}
    
	# Set environment variables
	$env:LICENSE_KEY = "DEMO_LICENSE_BYPASS"
	$env:PEER_SECRET = "demo_peer_secret_123"
    
	# Kill any existing Bitcoin Sprint processes
	Get-Process -Name "*bitcoin-sprint*" -ErrorAction SilentlyContinue | Stop-Process -Force
    
	Write-Host "🚀 Starting Bitcoin Sprint..." -ForegroundColor Cyan
	$process = Start-Process -FilePath ".\bitcoin-sprint-test.exe" -PassThru
    
	Write-Host "⏳ Waiting for startup..." -ForegroundColor Yellow
	Start-Sleep 4
    
	# Test the API
	Write-Host "🔍 Testing Bitcoin Sprint API..." -ForegroundColor Yellow
    
	try {
		$response = Invoke-RestMethod -Uri "http://localhost:8080/status" -TimeoutSec 10
		Write-Host "✅ Bitcoin Sprint is running!" -ForegroundColor Green
        
		if ($Verbose) {
			Write-Host "📊 Status Response:" -ForegroundColor Cyan
			$response | ConvertTo-Json -Depth 3 | Write-Host
		}
		else {
			Write-Host "   Status: OK" -ForegroundColor Cyan
		}
        
		# Test additional endpoints
		Write-Host "🧪 Testing additional endpoints..." -ForegroundColor Yellow
        
		$endpoints = @("/latest", "/metrics")
		foreach ($endpoint in $endpoints) {
			try {
				$testResponse = Invoke-RestMethod -Uri "http://localhost:8080$endpoint" -TimeoutSec 5
				Write-Host "   ✅ $endpoint - OK" -ForegroundColor Green
			}
			catch {
				Write-Host "   ❌ $endpoint - Failed: $($_.Exception.Message)" -ForegroundColor Red
			}
		}
        
		Write-Host ""
		Write-Host "🎉 SUCCESS! Bitcoin Sprint is running and responding!" -ForegroundColor Green
		Write-Host ""
		Write-Host "📊 Access Points:" -ForegroundColor Cyan
		Write-Host "   API Status: http://localhost:8080/status" -ForegroundColor White
		Write-Host "   Latest Block: http://localhost:8080/latest" -ForegroundColor White
		Write-Host "   Metrics: http://localhost:8080/metrics" -ForegroundColor White
		Write-Host "   Dashboard: http://localhost:8080/" -ForegroundColor White
		Write-Host ""
		Write-Host "📝 Process ID: $($process.Id)" -ForegroundColor Yellow
		Write-Host "⏹️  To stop: Stop-Process -Id $($process.Id)" -ForegroundColor Red
		Write-Host ""
		Write-Host "🧪 Test it now with:" -ForegroundColor Cyan
		Write-Host "   curl http://localhost:8080/status" -ForegroundColor White
		Write-Host "   Invoke-RestMethod http://localhost:8080/latest" -ForegroundColor White
        
		return $true
        
	}
	catch {
		Write-Host "❌ Bitcoin Sprint API test failed: $($_.Exception.Message)" -ForegroundColor Red
		Write-Host "🔍 Process status: $(if ($process.HasExited) { 'Exited' } else { 'Running' })" -ForegroundColor Yellow
		return $false
	}
}

# Main execution
try {
	$success = Start-BitcoinSprintTest -Mode $Mode
	if (!$success) {
		Write-Host "❌ Test failed" -ForegroundColor Red
		exit 1
	}
}
catch {
	Write-Host "❌ Error: $($_.Exception.Message)" -ForegroundColor Red
	exit 1
}
