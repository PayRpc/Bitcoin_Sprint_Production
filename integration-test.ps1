# Bitcoin Sprint Integration Test (PowerShell)
# Tests Rust SecureBuffer integration with Go main application

Write-Host "🧪 Bitcoin Sprint Integration Test" -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan

# Stop any running instances
Write-Host "🔄 Stopping existing processes..." -ForegroundColor Yellow
Get-Process -Name "*bitcoin-sprint*" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue

# Set test environment
$env:LICENSE_KEY = "DEMO_LICENSE_BYPASS"
$env:PEER_SECRET = "demo_peer_secret_123"

Write-Host ""
Write-Host "🏗️  Building with Rust SecureBuffer integration..." -ForegroundColor Yellow

# Build with CGO and Rust
Set-Location "C:\Projects\Bitcoin_Sprint_final_1\BItcoin_Sprint"
$env:Path = "C:\msys64\mingw64\bin;" + $env:Path
$env:CGO_ENABLED = '1'
$env:CC = 'gcc'
$env:CXX = 'g++'

try {
	go build -tags cgo -o bitcoin-sprint-test.exe ./cmd/sprint
	Write-Host "✅ Build successful" -ForegroundColor Green
}
catch {
	Write-Host "❌ Build failed: $($_.Exception.Message)" -ForegroundColor Red
	exit 1
}

Write-Host ""
Write-Host "🔍 Checking Rust library..." -ForegroundColor Yellow
if (Test-Path "secure\rust\target\release\securebuffer.dll") {
	Write-Host "✅ Rust SecureBuffer library found" -ForegroundColor Green
	Get-ChildItem "secure\rust\target\release\securebuffer.*" | Select-Object Name, Length, LastWriteTime
}
else {
	Write-Host "❌ Rust library missing" -ForegroundColor Red
	exit 1
}

Write-Host ""
Write-Host "🚀 Testing application startup..." -ForegroundColor Yellow

# Start in background
$process = Start-Process -FilePath ".\bitcoin-sprint-test.exe" -PassThru
Start-Sleep 3

# Test health endpoint
Write-Host "🌐 Testing health endpoint..." -ForegroundColor Yellow
try {
	$response = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/status" -UseBasicParsing -TimeoutSec 5
	if ($response.Content -like "*memory_protection*") {
		Write-Host "✅ Health endpoint responding" -ForegroundColor Green
		Write-Host "🔒 Memory protection status:" -ForegroundColor Cyan
        
		# Try to parse and display memory protection info
		try {
			$json = $response.Content | ConvertFrom-Json
			Write-Host "   Backend: $($json.memory_protection.secure_backend)" -ForegroundColor White
			Write-Host "   Self-check: $($json.memory_protection.self_check)" -ForegroundColor White
			Write-Host "   Features: $($json.memory_protection.features -join ', ')" -ForegroundColor White
		}
		catch {
			Write-Host "   Raw response received (JSON parse failed)" -ForegroundColor Yellow
		}
	}
 else {
		Write-Host "❌ Health endpoint missing memory protection info" -ForegroundColor Red
	}
}
catch {
	Write-Host "❌ Health endpoint not responding: $($_.Exception.Message)" -ForegroundColor Red
}

# Cleanup
Write-Host ""
Write-Host "🧹 Cleaning up..." -ForegroundColor Yellow
if (-not $process.HasExited) {
	$process | Stop-Process -Force -ErrorAction SilentlyContinue
}
Start-Sleep 1

Write-Host "✅ Integration test complete!" -ForegroundColor Green
