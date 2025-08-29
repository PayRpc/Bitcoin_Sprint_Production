# Simple Go Backend Starter for Manual Testing
# This script starts the Go backend with mock ZMQ for testing

Write-Host "Starting Bitcoin Sprint Go Backend (Mock Mode)..." -ForegroundColor Green
Write-Host "=================================================" -ForegroundColor Green

# Set environment variables for testing
$env:ZMQ_ENDPOINT = "tcp://127.0.0.1:28332"
$env:API_PORT = "8080"
$env:TIER = "enterprise"

# Start the backend
Write-Host "Starting sprintd.exe..." -ForegroundColor Yellow
$process = Start-Process -FilePath ".\sprintd.exe" -NoNewWindow -PassThru

Write-Host "✅ Backend started (PID: $($process.Id))" -ForegroundColor Green
Write-Host "🌐 API available at: http://localhost:8080" -ForegroundColor Cyan
Write-Host "📊 Health check: http://localhost:8080/health" -ForegroundColor Cyan
Write-Host "" -ForegroundColor White
Write-Host "To stop the backend, press Ctrl+C or run:" -ForegroundColor Yellow
Write-Host "Stop-Process -Id $($process.Id)" -ForegroundColor Yellow

# Wait for the process to finish
Wait-Process -Id $process.Id
