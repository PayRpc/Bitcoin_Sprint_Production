# Quick Speed Tier Validation Test
# Tests key tier configurations against expected behavior

Write-Host "=== Speed Tier Validation Test ===" -ForegroundColor Cyan
Write-Host ""

# Test Results
$Results = @()

function Test-TierConfig {
	param($ConfigFile, $ExpectedTier, $ExpectedTurbo, $TierName)
    
	Write-Host "Testing $TierName ($ConfigFile):" -ForegroundColor Yellow
    
	if (Test-Path $ConfigFile) {
		try {
			$config = Get-Content $ConfigFile | ConvertFrom-Json
            
			# Tier check
			$tierOK = $config.tier -eq $ExpectedTier
			Write-Host "  Tier: $($config.tier) " -NoNewline -ForegroundColor $(if ($tierOK) { 'Green' }else { 'Red' })
			Write-Host $(if ($tierOK) { '✅' }else { '❌' }) -ForegroundColor $(if ($tierOK) { 'Green' }else { 'Red' })
            
			# Turbo check
			$turboOK = $config.turbo_mode -eq $ExpectedTurbo
			Write-Host "  Turbo Mode: $($config.turbo_mode) " -NoNewline -ForegroundColor $(if ($turboOK) { 'Green' }else { 'Red' })
			Write-Host $(if ($turboOK) { '✅' }else { '❌' }) -ForegroundColor $(if ($turboOK) { 'Green' }else { 'Red' })
            
			# Performance indicators
			Write-Host "  Poll Interval: $($config.poll_interval)s" -ForegroundColor Gray
			if ($config.rate_limits -and $config.rate_limits."/status") {
				Write-Host "  Rate Limit (/status): $($config.rate_limits."/status") req/min" -ForegroundColor Gray
			}
            
			$script:Results += [PSCustomObject]@{
				Tier         = $TierName
				File         = $ConfigFile
				TierCorrect  = $tierOK
				TurboCorrect = $turboOK
				PollInterval = $config.poll_interval
				RateLimit    = if ($config.rate_limits."/status") { $config.rate_limits."/status" } else { "N/A" }
			}
            
		}
		catch {
			Write-Host "  ❌ Failed to parse config: $($_.Exception.Message)" -ForegroundColor Red
		}
	}
 else {
		Write-Host "  ❌ Config file not found" -ForegroundColor Red
	}
	Write-Host ""
}

# Test each tier configuration
Test-TierConfig "config-free-stable.json" "free" $false "FREE Tier"
Test-TierConfig "config.json" "pro" $false "PRO Tier" 
Test-TierConfig "config-turbo.json" "enterprise" $true "ENTERPRISE Tier"
Test-TierConfig "config-enterprise-turbo.json" "enterprise" $true "ENTERPRISE Turbo"

# Check license file
Write-Host "License Configuration:" -ForegroundColor Yellow
if (Test-Path "license-enterprise.json") {
	try {
		$license = Get-Content "license-enterprise.json" | ConvertFrom-Json
		Write-Host "  License Tier: $($license.license.tier)" -ForegroundColor Green
		Write-Host "  Turbo Feature: $($license.license.features.turbo_mode)" -ForegroundColor Green
		Write-Host "  Expires: $($license.license.expires)" -ForegroundColor Gray
	}
 catch {
		Write-Host "  ❌ Failed to parse license file" -ForegroundColor Red
	}
}
else {
	Write-Host "  ❌ license-enterprise.json not found" -ForegroundColor Red
}
Write-Host ""

# Test current running configuration (if Sprint is running)
Write-Host "Live Service Check:" -ForegroundColor Yellow
$sprintRunning = Get-NetTCPConnection -LocalPort 8080 -State Listen -ErrorAction SilentlyContinue
if ($sprintRunning) {
	try {
		$response = Invoke-RestMethod -Uri "http://localhost:8080/status" -TimeoutSec 3
		Write-Host "  ✅ Sprint API responding" -ForegroundColor Green
		Write-Host "  Current Tier: $($response.tier)" -ForegroundColor Gray
		Write-Host "  Turbo Enabled: $($response.turbo_mode_enabled)" -ForegroundColor $(if ($response.turbo_mode_enabled) { 'Green' }else { 'Yellow' })
		Write-Host "  Bitcoin Connected: $($response.bitcoin_connected)" -ForegroundColor $(if ($response.bitcoin_connected) { 'Green' }else { 'Red' })
		Write-Host "  Version: $($response.version)" -ForegroundColor Gray
	}
 catch {
		Write-Host "  ❌ Sprint API call failed: $($_.Exception.Message)" -ForegroundColor Red
	}
}
else {
	Write-Host "  ⚠️  Sprint not running on port 8080" -ForegroundColor Yellow
}
Write-Host ""

# Summary table
Write-Host "Configuration Summary:" -ForegroundColor Cyan
Write-Host "Tier               | File                     | Tier OK | Turbo OK | Poll | Rate Limit" -ForegroundColor White
Write-Host "-" * 85 -ForegroundColor Gray
$Results | ForEach-Object {
	$tierStatus = if ($_.TierCorrect) { "✅" } else { "❌" }
	$turboStatus = if ($_.TurboCorrect) { "✅" } else { "❌" }
	$tierName = $_.Tier.PadRight(18)
	$fileName = $_.File.PadRight(24)
	Write-Host "$tierName | $fileName | $tierStatus      | $turboStatus       | $($_.PollInterval)s   | $($_.RateLimit)" -ForegroundColor Gray
}

# Performance expectations
Write-Host "`nPerformance Expectations:" -ForegroundColor Cyan
Write-Host "• FREE Tier: 8s poll interval, 20 req/min rate limit" -ForegroundColor Gray
Write-Host "• PRO Tier: 2s poll interval, basic rate limits" -ForegroundColor Gray  
Write-Host "• ENTERPRISE: 1s poll interval, turbo mode enabled, high rate limits" -ForegroundColor Gray
Write-Host "• Environment variable TURBO_MODE=true can override config settings" -ForegroundColor Gray

$allCorrect = ($Results | Where-Object { $_.TierCorrect -and $_.TurboCorrect }).Count -eq $Results.Count
if ($allCorrect) {
	Write-Host "`n🎉 All tier configurations are correct!" -ForegroundColor Green
}
else {
	Write-Host "`n⚠️  Some configurations need attention" -ForegroundColor Yellow
}

Write-Host "`nTo test different tiers:" -ForegroundColor Yellow
Write-Host "1. Copy desired config file to config.json" -ForegroundColor Gray
Write-Host "2. Restart Sprint: .\bitcoin-sprint.exe" -ForegroundColor Gray
Write-Host "3. Check /status endpoint for tier and turbo_mode_enabled" -ForegroundColor Gray
