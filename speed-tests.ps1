#!/usr/bin/env pwsh
<#
.SYNOPSIS
Bitcoin Sprint Speed Tests and Bitcoin Core Connection Verification

.DESCRIPTION
Comprehensive speed testing for Bitcoin Sprint including:
- Standard vs Turbo mode performance comparison
- Bitcoin Core RPC connection testing
- API response time benchmarks
- Configuration validation

.EXAMPLE
.\speed-tests.ps1 -Verbose
#>

param(
	[switch]$Verbose,
	[switch]$SkipBitcoinCore,
	[int]$TestIterations = 10
)

# Test configuration
$TestResults = @()
$Configs = @{
	"Standard" = "config.json"
	"Turbo"    = "config-turbo.json"
}

function Write-TestResult {
	param(
		[string]$TestName,
		[bool]$Passed,
		[string]$Details,
		[double]$ResponseTime = 0,
		[string]$Config = ""
	)
    
	$Result = [PSCustomObject]@{
		TestName     = $TestName
		Config       = $Config
		Passed       = $Passed
		Details      = $Details
		ResponseTime = $ResponseTime
		Timestamp    = Get-Date
	}
    
	$script:TestResults += $Result
    
	$Status = if ($Passed) { "✅ PASS" } else { "❌ FAIL" }
	$TimeInfo = if ($ResponseTime -gt 0) { " ({0:F2}ms)" -f $ResponseTime } else { "" }
	Write-Host "[$Status] $TestName$TimeInfo" -ForegroundColor $(if ($Passed) { "Green" } else { "Red" })
	if ($Details) { Write-Host "    $Details" -ForegroundColor Gray }
}

function Test-BitcoinCoreConnection {
	param([string]$ConfigFile)
    
	Write-Host "`n🔗 Testing Bitcoin Core Connection with $ConfigFile..." -ForegroundColor Cyan
    
	try {
		$config = Get-Content $ConfigFile | ConvertFrom-Json
        
		if ($config.rpc_nodes -and $config.rpc_nodes.Count -gt 0) {
			$rpcUrl = $config.rpc_nodes[0]
			Write-Host "Testing RPC connection to: $rpcUrl" -ForegroundColor Yellow
            
			# Test Bitcoin Core RPC getblockchaininfo
			$authString = [Convert]::ToBase64String([Text.Encoding]::ASCII.GetBytes("$($config.rpc_user):$($config.rpc_pass)"))
			$headers = @{
				"Authorization" = "Basic $authString"
				"Content-Type"  = "application/json"
			}
            
			$rpcBody = @{
				"jsonrpc" = "1.0"
				"id"      = "speed-test"
				"method"  = "getblockchaininfo"
				"params"  = @()
			} | ConvertTo-Json
            
			$stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
            
			try {
				$response = Invoke-WebRequest -Uri $rpcUrl -Method POST -Headers $headers -Body $rpcBody -UseBasicParsing -TimeoutSec 5
				$stopwatch.Stop()
                
				if ($response.StatusCode -eq 200) {
					$result = $response.Content | ConvertFrom-Json
					if ($result.result) {
						Write-TestResult -TestName "Bitcoin Core RPC Connection" -Config $ConfigFile -Passed $true -Details "Chain: $($result.result.chain), Blocks: $($result.result.blocks)" -ResponseTime $stopwatch.ElapsedMilliseconds
						return $true
					}
				}
			}
			catch {
				$stopwatch.Stop()
				Write-TestResult -TestName "Bitcoin Core RPC Connection" -Config $ConfigFile -Passed $false -Details "RPC Error: $($_.Exception.Message)" -ResponseTime $stopwatch.ElapsedMilliseconds
			}
		}
		else {
			Write-TestResult -TestName "Bitcoin Core RPC Connection" -Config $ConfigFile -Passed $false -Details "No RPC nodes configured"
		}
	}
	catch {
		Write-TestResult -TestName "Bitcoin Core Connection Test" -Config $ConfigFile -Passed $false -Details $_.Exception.Message
	}
    
	return $false
}

function Test-ServicePerformance {
	param([string]$ConfigFile, [string]$ServicePort = "9090")
    
	Write-Host "`n⚡ Testing Service Performance with $ConfigFile..." -ForegroundColor Cyan
    
	# Start service with specific config
	$processArgs = if ($ConfigFile -eq "config-turbo.json") {
		"-config $ConfigFile"
	}
 else {
		""
	}
    
	Write-Host "Starting Bitcoin Sprint with $ConfigFile..." -ForegroundColor Yellow
    
	# Kill any existing processes
	Get-Process -Name "*bitcoin-sprint*" -ErrorAction SilentlyContinue | Stop-Process -Force
	Start-Sleep 2
    
	# Start new process
	$env:LICENSE_KEY = "DEMO_LICENSE_BYPASS"
	$env:PEER_SECRET = "demo_peer_secret_123"
    
	if ($ConfigFile -eq "config-turbo.json") {
		Copy-Item $ConfigFile "config.json" -Force
	}
    
	$process = Start-Process -FilePath ".\bitcoin-sprint-test.exe" -PassThru
	Start-Sleep 5  # Give service time to start
    
	try {
		# Test endpoints
		$endpoints = @("/status", "/api/v1/secure-channel/status", "/latest")
        
		foreach ($endpoint in $endpoints) {
			$url = "http://localhost:$ServicePort$endpoint"
            
			# Warm up
			try {
				Invoke-WebRequest -Uri $url -UseBasicParsing -TimeoutSec 2 -ErrorAction SilentlyContinue | Out-Null
			}
			catch { }
            
			# Performance test
			$times = @()
			for ($i = 0; $i -lt $TestIterations; $i++) {
				$stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
				try {
					$response = Invoke-WebRequest -Uri $url -UseBasicParsing -TimeoutSec 10
					$stopwatch.Stop()
                    
					if ($response.StatusCode -eq 200) {
						$times += $stopwatch.ElapsedMilliseconds
					}
				}
				catch {
					$stopwatch.Stop()
					Write-Verbose "Request failed: $($_.Exception.Message)"
				}
			}
            
			if ($times.Count -gt 0) {
				$avgTime = ($times | Measure-Object -Average).Average
				$minTime = ($times | Measure-Object -Minimum).Minimum
				$maxTime = ($times | Measure-Object -Maximum).Maximum
                
				Write-TestResult -TestName "$endpoint Performance" -Config $ConfigFile -Passed $true -Details "Avg: $([math]::Round($avgTime,2))ms, Min: $minTime ms, Max: $maxTime ms" -ResponseTime $avgTime
			}
			else {
				Write-TestResult -TestName "$endpoint Performance" -Config $ConfigFile -Passed $false -Details "No successful requests"
			}
		}
	}
	finally {
		# Stop the service
		if ($process -and !$process.HasExited) {
			$process | Stop-Process -Force -ErrorAction SilentlyContinue
		}
		Get-Process -Name "*bitcoin-sprint*" -ErrorAction SilentlyContinue | Stop-Process -Force
	}
}

function Show-Results {
	Write-Host "`n📊 Speed Test Results Summary" -ForegroundColor Magenta
	Write-Host "=" * 60 -ForegroundColor Magenta
    
	$grouped = $TestResults | Group-Object Config
    
	foreach ($group in $grouped) {
		Write-Host "`n🔧 Configuration: $($group.Name)" -ForegroundColor Cyan
        
		$passed = ($group.Group | Where-Object Passed).Count
		$total = $group.Group.Count
		$passRate = if ($total -gt 0) { [math]::Round(($passed / $total) * 100, 1) } else { 0 }
        
		Write-Host "   Pass Rate: $passed/$total ($passRate%)" -ForegroundColor $(if ($passRate -gt 75) { "Green" } else { "Yellow" })
        
		$performanceTests = $group.Group | Where-Object { $_.ResponseTime -gt 0 }
		if ($performanceTests) {
			$avgResponseTime = ($performanceTests | Measure-Object -Property ResponseTime -Average).Average
			Write-Host "   Avg Response Time: $([math]::Round($avgResponseTime,2))ms" -ForegroundColor $(if ($avgResponseTime -lt 100) { "Green" } elseif ($avgResponseTime -lt 500) { "Yellow" } else { "Red" })
		}
        
		foreach ($test in $group.Group) {
			$status = if ($test.Passed) { "✅" } else { "❌" }
			$timeInfo = if ($test.ResponseTime -gt 0) { " ($([math]::Round($test.ResponseTime,2))ms)" } else { "" }
			Write-Host "   $status $($test.TestName)$timeInfo" -ForegroundColor Gray
			if ($test.Details) {
				Write-Host "      $($test.Details)" -ForegroundColor DarkGray
			}
		}
	}
    
	# Performance comparison
	$standardPerf = $TestResults | Where-Object { $_.Config -like "*config.json*" -and $_.ResponseTime -gt 0 }
	$turboPerf = $TestResults | Where-Object { $_.Config -like "*turbo*" -and $_.ResponseTime -gt 0 }
    
	if ($standardPerf -and $turboPerf) {
		$standardAvg = ($standardPerf | Measure-Object -Property ResponseTime -Average).Average
		$turboAvg = ($turboPerf | Measure-Object -Property ResponseTime -Average).Average
        
		if ($turboAvg -lt $standardAvg) {
			$improvement = [math]::Round((($standardAvg - $turboAvg) / $standardAvg) * 100, 1)
			Write-Host "`n🚀 Turbo Mode Performance: $improvement% faster than Standard" -ForegroundColor Green
		}
		else {
			$degradation = [math]::Round((($turboAvg - $standardAvg) / $standardAvg) * 100, 1)
			Write-Host "`n⚠️ Turbo Mode Performance: $degradation% slower than Standard" -ForegroundColor Yellow
		}
	}
}

# Main execution
Write-Host "🚀 Bitcoin Sprint Speed Tests Starting..." -ForegroundColor Magenta
Write-Host "Test Iterations: $TestIterations" -ForegroundColor Gray

# Test Bitcoin Core connections (if not skipped)
if (!$SkipBitcoinCore) {
	foreach ($config in $Configs.Values) {
		if (Test-Path $config) {
			Test-BitcoinCoreConnection -ConfigFile $config
		}
	}
}

# Test service performance with different configurations
foreach ($configPair in $Configs.GetEnumerator()) {
	if (Test-Path $configPair.Value) {
		Test-ServicePerformance -ConfigFile $configPair.Value
	}
}

Show-Results

Write-Host "`n✅ Speed tests completed!" -ForegroundColor Green
