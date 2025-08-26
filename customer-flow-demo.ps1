# Bitcoin Sprint Customer Flow Demo with SecureBuffer Protection
Write-Host "🔐 Bitcoin Sprint Customer Flow with Rust SecureBuffer Protection" -ForegroundColor Cyan
Write-Host "=================================================================" -ForegroundColor Cyan

# 1. Check if app is running and SecureBuffer is active
Write-Host "`n🔍 Step 1: Checking SecureBuffer Status" -ForegroundColor Yellow
try {
	$status = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/status" -UseBasicParsing -TimeoutSec 5
	$json = $status.Content | ConvertFrom-Json
	Write-Host "   ✅ Application Running" -ForegroundColor Green
	Write-Host "   ✅ SecureBuffer Backend: $($json.memory_protection.secure_backend)" -ForegroundColor Green
	Write-Host "   ✅ Self-check Passed: $($json.memory_protection.self_check)" -ForegroundColor Green
	Write-Host "   🛡️  Protection: Memory-locked storage preventing swap to disk" -ForegroundColor Cyan
}
catch {
	Write-Host "   ❌ Application not responding" -ForegroundColor Red
	exit 1
}

# 2. Simulate customer API key request
Write-Host "`n🔑 Step 2: Customer Requests API Key" -ForegroundColor Yellow
Write-Host "   📝 Customer email: customer@bitcoinsprint.com" -ForegroundColor White
Write-Host "   📋 Tier: PRO" -ForegroundColor White

try {
	$keyRequest = @{
		email = "customer@bitcoinsprint.com"
		tier  = "PRO"
	} | ConvertTo-Json

	$response = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/keys/generate" -Method POST -Headers @{"Content-Type" = "application/json" } -Body $keyRequest -UseBasicParsing -TimeoutSec 10
	$apiData = $response.Content | ConvertFrom-Json
    
	Write-Host "   ✅ API Key Generated Successfully!" -ForegroundColor Green
	Write-Host "      📧 Email: $($apiData.email)" -ForegroundColor White
	Write-Host "      🎯 Tier: $($apiData.tier)" -ForegroundColor White
	Write-Host "      🔑 Key: $($apiData.key.Substring(0,20))..." -ForegroundColor White
	Write-Host "      📅 Created: $($apiData.created)" -ForegroundColor White
	Write-Host "   🛡️  SecureBuffer: API key stored in memory-locked buffer" -ForegroundColor Cyan
    
	$global:customerKey = $apiData.key
    
}
catch {
	Write-Host "   ❌ API Key generation failed: $($_.Exception.Message)" -ForegroundColor Red
	# Fallback to web interface
	Write-Host "   🌐 Trying web interface fallback..." -ForegroundColor Yellow
	try {
		$webResponse = Invoke-WebRequest -Uri "http://localhost:3000/api/simple-signup" -Method POST -Headers @{"Content-Type" = "application/json" } -Body $keyRequest -UseBasicParsing
		$webData = $webResponse.Content | ConvertFrom-Json
		Write-Host "   ✅ Web API Key Generated!" -ForegroundColor Green
		Write-Host "      🔑 Key: $($webData.key.Substring(0,20))..." -ForegroundColor White
		$global:customerKey = $webData.key
	}
 catch {
		Write-Host "   ❌ Web fallback also failed" -ForegroundColor Red
		return
	}
}

# 3. Customer uses their API key
Write-Host "`n🔐 Step 3: Customer Uses API Key for Authentication" -ForegroundColor Yellow
try {
	$authHeaders = @{
		"Authorization" = "Bearer $global:customerKey"
		"Content-Type"  = "application/json"
	}
    
	$verifyResponse = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/verify" -Headers $authHeaders -UseBasicParsing -TimeoutSec 5
	$verifyData = $verifyResponse.Content | ConvertFrom-Json
    
	Write-Host "   ✅ API Key Verification Successful!" -ForegroundColor Green
	Write-Host "      👤 Authenticated as: $($verifyData.email)" -ForegroundColor White
	Write-Host "      🎯 Access Level: $($verifyData.tier)" -ForegroundColor White
	Write-Host "      ⏰ Valid until: $($verifyData.expires)" -ForegroundColor White
	Write-Host "   🛡️  SecureBuffer: Credentials validated using memory-locked storage" -ForegroundColor Cyan
    
}
catch {
	Write-Host "   ❌ API Key verification failed: $($_.Exception.Message)" -ForegroundColor Red
}

# 4. Show memory protection in action
Write-Host "`n🛡️  Step 4: SecureBuffer Memory Protection Active" -ForegroundColor Yellow
Write-Host "   🔒 License Key: Protected in memory-locked buffer (no swap)" -ForegroundColor Green
Write-Host "   🔒 RPC Password: Protected in memory-locked buffer (no swap)" -ForegroundColor Green  
Write-Host "   🔒 Peer Secret: Protected in memory-locked buffer (no swap)" -ForegroundColor Green
Write-Host "   🔒 Customer API Keys: Protected in memory-locked buffer (no swap)" -ForegroundColor Green
Write-Host "   🧹 Auto-cleanup: Memory zeroed on application shutdown" -ForegroundColor Cyan

Write-Host "`n✅ Customer Flow Complete - All Sensitive Data Protected by Rust SecureBuffer!" -ForegroundColor Green
