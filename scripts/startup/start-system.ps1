#!/usr/bin/env powershell
# Bitcoin Sprint Full System Startup Script
# Starts Bitcoin Core, Go Backend, and Next.js Frontend

param(
    [switch]$Dev,
    [switch]$Production,
    [switch]$StopAll,
    [switch]$StatusOnly
)

$ErrorActionPreference = "Stop"

# Process management
$bitcoinProcess = $null
$goBackendProcess = $null
$nextjsProcess = $null

function Stop-AllProcesses {
    Write-Host "🛑 Stopping all Bitcoin Sprint processes..." -ForegroundColor Yellow
    
    # Stop processes by name
    Get-Process -Name "bitcoind" -ErrorAction SilentlyContinue | Stop-Process -Force
    Get-Process -Name "bitcoin-sprint*" -ErrorAction SilentlyContinue | Stop-Process -Force
    Get-Process -Name "node" -ErrorAction SilentlyContinue | Where-Object {$_.MainWindowTitle -like "*Next.js*"} | Stop-Process -Force
    
    Write-Host "✅ All processes stopped" -ForegroundColor Green
}

function Get-ProcessStatus {
    Write-Host "📊 Bitcoin Sprint System Status" -ForegroundColor Cyan
    Write-Host "==============================" -ForegroundColor Cyan
    
    $bitcoinRunning = Get-Process -Name "bitcoind" -ErrorAction SilentlyContinue
    $backendRunning = Get-Process -Name "bitcoin-sprint*" -ErrorAction SilentlyContinue
    $frontendRunning = Get-Process -Name "node" -ErrorAction SilentlyContinue | Where-Object {$_.ProcessName -eq "node"}
    
    Write-Host "Bitcoin Core: $(if ($bitcoinRunning) { '🟢 Running' } else { '🔴 Stopped' })" -ForegroundColor $(if ($bitcoinRunning) { 'Green' } else { 'Red' })
    Write-Host "Go Backend:   $(if ($backendRunning) { '🟢 Running' } else { '🔴 Stopped' })" -ForegroundColor $(if ($backendRunning) { 'Green' } else { 'Red' })
    Write-Host "Next.js:      $(if ($frontendRunning) { '🟢 Running' } else { '🔴 Stopped' })" -ForegroundColor $(if ($frontendRunning) { 'Green' } else { 'Red' })
    
    if ($bitcoinRunning) {
        Write-Host "Bitcoin PID:  $($bitcoinRunning.Id)" -ForegroundColor Gray
    }
    if ($backendRunning) {
        Write-Host "Backend PID:  $($backendRunning.Id)" -ForegroundColor Gray
    }
    if ($frontendRunning) {
        Write-Host "Frontend PID: $($frontendRunning.Id)" -ForegroundColor Gray
    }
}

if ($StopAll) {
    Stop-AllProcesses
    exit 0
}

if ($StatusOnly) {
    Get-ProcessStatus
    exit 0
}

Write-Host "🚀 Starting Bitcoin Sprint Full System" -ForegroundColor Cyan

# Stop any existing processes
Stop-AllProcesses
Start-Sleep -Seconds 2

# =============================================================================
# 1. Start Bitcoin Core
# =============================================================================
Write-Host "⛓️  Starting Bitcoin Core (regtest)..." -ForegroundColor Yellow

if (-not (Test-Path "bitcoin.conf")) {
    Write-Error "❌ bitcoin.conf not found. Please run setup first."
    exit 1
}

$bitcoinCmd = "bitcoind"
$bitcoinArgs = @(
    "-regtest",
    "-conf=./bitcoin.conf",
    "-datadir=./bitcoin-data",
    "-daemon"
)

try {
    Start-Process -FilePath $bitcoinCmd -ArgumentList $bitcoinArgs -NoNewWindow
    Write-Host "✅ Bitcoin Core started" -ForegroundColor Green
    Start-Sleep -Seconds 5  # Wait for Bitcoin Core to initialize
} catch {
    Write-Error "❌ Failed to start Bitcoin Core: $_"
    exit 1
}

# =============================================================================
# 2. Start Go Backend
# =============================================================================
Write-Host "🔧 Starting Go backend..." -ForegroundColor Yellow

$backendExe = if ($Production) { "sprintd.exe" } else { "bitcoin-sprint-dev.exe" }

if (-not (Test-Path $backendExe)) {
    Write-Error "❌ Backend executable not found: $backendExe"
    Write-Host "Please run: go build -o $backendExe ./cmd/sprintd" -ForegroundColor Yellow
    exit 1
}

try {
    $goBackendProcess = Start-Process -FilePath ".\$backendExe" -PassThru -NoNewWindow
    Write-Host "✅ Go backend started (PID: $($goBackendProcess.Id))" -ForegroundColor Green
    Start-Sleep -Seconds 3  # Wait for API server to start
} catch {
    Write-Error "❌ Failed to start Go backend: $_"
    exit 1
}

# =============================================================================
# 3. Start Next.js Frontend
# =============================================================================
Write-Host "🌐 Starting Next.js frontend..." -ForegroundColor Yellow

Set-Location "web"

if (-not (Test-Path "node_modules")) {
    Write-Host "📦 Installing dependencies..." -ForegroundColor Yellow
    npm install
}

try {
    if ($Production) {
        # Production mode
        if (-not (Test-Path ".next")) {
            Write-Host "🏗️  Building for production..." -ForegroundColor Yellow
            npm run build
        }
        $nextjsProcess = Start-Process -FilePath "npm" -ArgumentList @("start") -PassThru -NoNewWindow
    } else {
        # Development mode
        $nextjsProcess = Start-Process -FilePath "npm" -ArgumentList @("run", "dev") -PassThru -NoNewWindow
    }
    
    Write-Host "✅ Next.js frontend started (PID: $($nextjsProcess.Id))" -ForegroundColor Green
} catch {
    Write-Error "❌ Failed to start Next.js: $_"
    Set-Location ".."
    exit 1
}

Set-Location ".."

# =============================================================================
# System Ready
# =============================================================================
Write-Host "`n🎉 Bitcoin Sprint System Ready!" -ForegroundColor Green
Write-Host "==============================" -ForegroundColor Green

Write-Host "`n🌐 Access URLs:" -ForegroundColor Cyan
Write-Host "Frontend:     http://localhost:3000" -ForegroundColor White
Write-Host "Backend API:  http://localhost:8080" -ForegroundColor White
Write-Host "Bitcoin RPC:  http://localhost:18332" -ForegroundColor White

Write-Host "`n📊 Health Checks:" -ForegroundColor Cyan
Write-Host "Frontend:     http://localhost:3000/api/health" -ForegroundColor White
Write-Host "Backend:      http://localhost:8080/health" -ForegroundColor White

Write-Host "`n🔧 Management:" -ForegroundColor Cyan
Write-Host "Stop all:     .\start-system.ps1 -StopAll" -ForegroundColor White
Write-Host "Status:       .\start-system.ps1 -StatusOnly" -ForegroundColor White

Write-Host "`nPress Ctrl+C to stop all services..." -ForegroundColor Yellow

# Wait and monitor
try {
    while ($true) {
        Start-Sleep -Seconds 10
        
        # Check if processes are still running
        if ($goBackendProcess -and $goBackendProcess.HasExited) {
            Write-Warning "⚠️  Go backend has stopped"
            break
        }
        if ($nextjsProcess -and $nextjsProcess.HasExited) {
            Write-Warning "⚠️  Next.js frontend has stopped"
            break
        }
    }
} catch {
    Write-Host "`n🛑 Stopping services..." -ForegroundColor Yellow
} finally {
    Stop-AllProcesses
}
