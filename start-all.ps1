param(
    [switch]$Stop
)

$ErrorActionPreference = "Stop"

Write-Host "==========================================="
Write-Host "  Bitcoin Sprint - Unified Startup"
Write-Host "==========================================="
Write-Host "Python 3.13 | FastAPI Gateway | Next.js Frontend"
Write-Host ""

$rootDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $rootDir

if ($Stop) {
    Write-Host "Stopping all services..."
    Get-Process | Where-Object { $_.MainWindowTitle -like "*Bitcoin Sprint*" } | Stop-Process -Force
    Write-Host "All services stopped."
    exit
}

Write-Host "[1/4] Starting FastAPI Gateway (Port 8000)..."
Push-Location "fastapi-gateway"
if (Test-Path "venv\Scripts\Activate.ps1") {
    & "venv\Scripts\Activate.ps1"
    Start-Process -FilePath "python" -ArgumentList "-m uvicorn main:app --host 127.0.0.1 --port 8000 --reload" -WindowStyle Normal
    Write-Host "FastAPI Gateway started in new window"
} else {
    Write-Error "FastAPI virtual environment not found. Please run setup first."
    exit 1
}
Pop-Location

Write-Host "[2/4] Starting Next.js Frontend (Port 3002)..."
Push-Location "web"
if (Test-Path "node_modules") {
    Start-Process -FilePath "npm" -ArgumentList "run dev" -WindowStyle Normal
    Write-Host "Next.js Frontend started in new window"
} else {
    Write-Error "Next.js dependencies not installed. Please run 'npm install' first."
    exit 1
}
Pop-Location

Write-Host "[3/4] Starting Go Backend (Port 8080)..."
if (Test-Path "bin\sprintd.exe") {
    Start-Process -FilePath "bin\sprintd.exe" -WindowStyle Normal
    Write-Host "Go Backend started in new window"
} else {
    Write-Warning "Go backend not built. Please build it first."
}

Write-Host "[4/4] Starting Grafana (Port 3000)..."
if (Test-Path "grafana-compose.yml") {
    Start-Process -FilePath "docker-compose" -ArgumentList "-f grafana-compose.yml up" -WindowStyle Normal
    Write-Host "Grafana started in new window"
} else {
    Write-Warning "Grafana configuration not found."
}

Write-Host ""
Write-Host "==========================================="
Write-Host "  All Services Started!"
Write-Host "==========================================="
Write-Host "FastAPI Gateway: http://localhost:8000"
Write-Host "Next.js Frontend: http://localhost:3002"
Write-Host "Go Backend: http://localhost:8080"
Write-Host "Grafana: http://localhost:3000"
Write-Host ""
Write-Host "Run this script with -Stop to stop all services"
Write-Host ""

# Keep the script running
try {
    while ($true) {
        Start-Sleep -Seconds 1
    }
} finally {
    Write-Host "Stopping all services..."
    Get-Process | Where-Object { $_.MainWindowTitle -like "*Bitcoin Sprint*" } | Stop-Process -Force
    Write-Host "All services stopped."
}
