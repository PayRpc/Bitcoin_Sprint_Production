# Bitcoin Sprint Backend Starter Script
# This script starts the Bitcoin Sprint backend with proper environment settings
# and keeps it running in the background

Write-Host "Starting Bitcoin Sprint Backend (Enterprise Edition)..." -ForegroundColor Green

# Set required environment variables
$env:API_PORT = "9090"
$env:TIER = "enterprise"
$env:API_HOST = "127.0.0.1"
$env:API_KEY = "bitcoin-sprint-secure-key"

# Launch the backend process in a way that it won't terminate with the console
Start-Process -FilePath ".\bitcoin-sprint-backend.exe" -NoNewWindow -PassThru

Write-Host "✅ Backend started successfully on port 9090" -ForegroundColor Green
Write-Host "API endpoints available at: http://localhost:9090/" -ForegroundColor Cyan
Write-Host "To test, run: curl http://localhost:9090/health" -ForegroundColor Yellow
Write-Host "To stop the backend, find and kill the process" -ForegroundColor DarkYellow

# Display process info
Write-Host "`nRunning Bitcoin Sprint Backend processes:" -ForegroundColor Magenta
Get-Process -Name "bitcoin-sprint-backend" | Select-Object Id, ProcessName, StartTime | Format-Table
