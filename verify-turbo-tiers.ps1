#!/usr/bin/env pwsh
<#
.SYNOPSIS
Quick Turbo Mode Verification Script

.DESCRIPTION
Quick verification that turbo mode is properly gated by tiers
#>

Write-Host "🔍 Quick Turbo Mode Tier Check" -ForegroundColor Cyan
Write-Host "==============================" -ForegroundColor Cyan

# Check 1: Configuration Files
Write-Host "`n1. Configuration File Analysis:" -ForegroundColor Yellow

if (Test-Path "config-turbo.json") {
	$turboConfig = Get-Content "config-turbo.json" | ConvertFrom-Json
	Write-Host "   config-turbo.json:" -ForegroundColor White
	Write-Host "     - tier: $($turboConfig.tier)" -ForegroundColor Gray
	Write-Host "     - turbo_mode: $($turboConfig.turbo_mode)" -ForegroundColor Gray
	Write-Host "     - performance_mode: $($turboConfig.performance_mode)" -ForegroundColor Gray
}
else {
	Write-Host "   ❌ config-turbo.json not found" -ForegroundColor Red
}

if (Test-Path "config.json") {
	$regularConfig = Get-Content "config.json" | ConvertFrom-Json
	Write-Host "   config.json:" -ForegroundColor White
	Write-Host "     - tier: $($regularConfig.tier)" -ForegroundColor Gray
	if ($null -eq $regularConfig.turbo_mode) {
		Write-Host "     - turbo_mode: (missing)" -ForegroundColor Red
	}
 else {
		Write-Host "     - turbo_mode: $($regularConfig.turbo_mode)" -ForegroundColor Gray
	}
}
else {
	Write-Host "   ❌ config.json not found" -ForegroundColor Red
}

# Check 2: License Files
Write-Host "`n2. License File Analysis:" -ForegroundColor Yellow

if (Test-Path "license-enterprise.json") {
	$enterpriseLicense = Get-Content "license-enterprise.json" | ConvertFrom-Json
	Write-Host "   license-enterprise.json:" -ForegroundColor White
	Write-Host "     - tier: $($enterpriseLicense.license.tier)" -ForegroundColor Gray
	
	# Check for turbo_mode with fallback paths
	$turboFeature = $enterpriseLicense.license.features.turbo_mode
	if ($null -eq $turboFeature) { 
		$turboFeature = $enterpriseLicense.license.turbo_mode 
	}
	if ($null -eq $turboFeature) {
		$turboFeature = $enterpriseLicense.turbo_mode
	}
	Write-Host "     - turbo_mode feature: $turboFeature" -ForegroundColor Gray
}
else {
	Write-Host "   ❌ license-enterprise.json not found" -ForegroundColor Red
}

# Check 3: Web API Logic
Write-Host "`n3. Web API Turbo Logic Analysis:" -ForegroundColor Yellow

$statusApiPath = "web/pages/api/status.ts"
$licenseInfoPath = "web/pages/api/v1/license/info.ts"

if (Test-Path $statusApiPath) {
	$statusContent = Get-Content $statusApiPath -Raw
	if ($statusContent -match "(?s)turbo_mode_enabled.*ENTERPRISE.*ENTERPRISE_PLUS") {
		Write-Host "   status.ts turbo logic:" -ForegroundColor White
		Write-Host "     - Enabled for ENTERPRISE and ENTERPRISE_PLUS tiers" -ForegroundColor Gray
	}
 else {
		Write-Host "   [X] Could not parse turbo logic in status.ts" -ForegroundColor Red
	}
}
else {
	Write-Host "   [X] $statusApiPath not found" -ForegroundColor Red
}

if (Test-Path $licenseInfoPath) {
	$licenseContent = Get-Content $licenseInfoPath -Raw
	if ($licenseContent -match "(?s)turbo_mode.*ENTERPRISE.*ENTERPRISE_PLUS") {
		Write-Host "   license/info.ts turbo logic:" -ForegroundColor White
		Write-Host "     - Enabled for ENTERPRISE and ENTERPRISE_PLUS tiers" -ForegroundColor Gray
	}
 else {
		Write-Host "   [X] Could not parse turbo logic in license/info.ts" -ForegroundColor Red
	}
}
else {
	Write-Host "   [X] $licenseInfoPath not found" -ForegroundColor Red
}

# Check 4: Go Code Analysis
Write-Host "`n4. Go Code Turbo Implementation:" -ForegroundColor Yellow

$mainGoPath = "cmd/sprint/main.go"
if (Test-Path $mainGoPath) {
	$goContent = Get-Content $mainGoPath -Raw
    
	# Track validation results
	$goValidations = @{
		TurboStruct    = $false
		EnvVar         = $false
		RateLimit      = $false
		StatusResponse = $false
	}
    
	# Check for turbo mode struct field
	if ($goContent -match "TurboMode\s+bool.*json:.*turbo_mode") {
		Write-Host "   [✓] TurboMode field found in config struct" -ForegroundColor Green
		$goValidations.TurboStruct = $true
	}
 else {
		Write-Host "   [X] TurboMode field not found" -ForegroundColor Red
	}
    
	# Check for environment variable override
	if ($goContent -match 'os\.Getenv\("TURBO_MODE"\)') {
		Write-Host "   [✓] TURBO_MODE environment variable support found" -ForegroundColor Green
		$goValidations.EnvVar = $true
	}
 else {
		Write-Host "   [X] TURBO_MODE environment variable support not found" -ForegroundColor Red
	}
    
	# Check for turbo mode usage in rate limiting
	if ($goContent -match 'rateLimiter\.Allow.*s\.config\.TurboMode') {
		Write-Host "   [✓] Turbo mode integrated with rate limiting" -ForegroundColor Green
		$goValidations.RateLimit = $true
	}
 else {
		Write-Host "   [X] Turbo mode not integrated with rate limiting" -ForegroundColor Red
	}
    
	# Check for turbo mode in status response
	if ($goContent -match 'TurboModeEnabled:\s*s\.config\.TurboMode') {
		Write-Host "   [✓] Turbo mode included in status response" -ForegroundColor Green
		$goValidations.StatusResponse = $true
	}
 else {
		Write-Host "   [X] Turbo mode not included in status response" -ForegroundColor Red
	}
    
	# Overall Go integration check
	$allGoChecks = $goValidations.Values | ForEach-Object { $_ }
	if ($allGoChecks -notcontains $false) {
		Write-Host "   [✓] Turbo mode fully integrated in Go code" -ForegroundColor Green
	}
 else {
		$failedChecks = $goValidations.GetEnumerator() | Where-Object { -not $_.Value } | ForEach-Object { $_.Key }
		Write-Host "   [!] Turbo integration incomplete - missing: $($failedChecks -join ', ')" -ForegroundColor Yellow
	}
}
else {
	Write-Host "   ❌ $mainGoPath not found" -ForegroundColor Red
}

# Check 5: Current Service Status (if running)
Write-Host "`n5. Live Service Check:" -ForegroundColor Yellow

try {
	$response = Invoke-RestMethod -Uri "http://localhost:8080/status" -TimeoutSec 3 -ErrorAction Stop
	Write-Host "   Go Service (port 8080):" -ForegroundColor White
	Write-Host "     - turbo_mode_enabled: $($response.turbo_mode_enabled)" -ForegroundColor Gray
	Write-Host "     - license_key: $($response.license_key)" -ForegroundColor Gray
	Write-Host "     - version: $($response.version)" -ForegroundColor Gray
}
catch {
	Write-Host "   ⚠️ Go service not responding on port 8080" -ForegroundColor Yellow
}

try {
	$response = Invoke-RestMethod -Uri "http://localhost:3000/api/status" -TimeoutSec 3 -ErrorAction Stop
	Write-Host "   Web API (port 3000):" -ForegroundColor White
	Write-Host "     - turbo_mode_enabled: $($response.turbo_mode_enabled)" -ForegroundColor Gray
	Write-Host "     - version: $($response.version)" -ForegroundColor Gray
}
catch {
	Write-Host "   ⚠️ Web API not responding on port 3000" -ForegroundColor Yellow
}

Write-Host "`n📋 Summary:" -ForegroundColor Cyan
Write-Host "- Turbo mode should be enabled for ENTERPRISE and ENTERPRISE_PLUS tiers only" -ForegroundColor White
Write-Host "- FREE and PRO tiers should NOT have turbo mode enabled" -ForegroundColor White
Write-Host "- Environment variable TURBO_MODE=true can override config settings" -ForegroundColor White
Write-Host "- Run .\test-turbo-tiers.ps1 for comprehensive testing" -ForegroundColor White

# Final validation check
Write-Host "`n🚦 Turbo Gating Validation: " -ForegroundColor Cyan
try {
	$response = Invoke-RestMethod -Uri "http://localhost:8080/status" -TimeoutSec 3 -ErrorAction Stop
	$currentTier = $response.tier
	$turboEnabled = $response.turbo_mode_enabled
	
	if ($turboEnabled -and ($currentTier -notin @("ENTERPRISE", "ENTERPRISE_PLUS"))) {
		Write-Host "❌ Turbo wrongly enabled for $currentTier tier" -ForegroundColor Red
		Write-Host "   Current tier: $currentTier, Turbo enabled: $turboEnabled" -ForegroundColor Yellow
	}
 elseif (-not $turboEnabled -and ($currentTier -in @("ENTERPRISE", "ENTERPRISE_PLUS"))) {
		Write-Host "⚠️  Turbo should be enabled for $currentTier tier but is disabled" -ForegroundColor Yellow
		Write-Host "   Check TURBO_MODE environment variable or config settings" -ForegroundColor Gray
	}
 else {
		Write-Host "✅ Turbo mode gating appears correct" -ForegroundColor Green
		Write-Host "   Tier: $currentTier, Turbo: $turboEnabled" -ForegroundColor Gray
	}
}
catch {
	Write-Host "⚠️  Cannot validate live service - service not responding" -ForegroundColor Yellow
	Write-Host "   Start the service with: .\bitcoin-sprint.exe" -ForegroundColor Gray
}
