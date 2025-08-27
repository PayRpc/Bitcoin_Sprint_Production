#!/usr/bin/env pwsh
# Bitcoin Sprint Performance-Optimized Startup
# Implements GC tuning and runtime optimizations for 99.9% SLA compliance

param(
    [ValidateSet("turbo", "enterprise", "standard")]
    [string]$Tier = "turbo",
    [switch]$MaxPerformance,
    [switch]$ShowMetrics
)

Write-Host "🚀 PERFORMANCE-OPTIMIZED BITCOIN SPRINT STARTUP" -ForegroundColor Cyan
Write-Host "=================================================" -ForegroundColor Cyan
Write-Host "Tier: $Tier" -ForegroundColor Yellow
Write-Host "Target: 99.9% SLA compliance at ≤5ms" -ForegroundColor Yellow
Write-Host ""

# Stop any existing processes
Write-Host "🔧 Preparing environment..." -ForegroundColor Blue
Get-Process -Name "*sprint*" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep 2

# Apply tier configuration
$configFile = switch ($Tier) {
    "turbo" { "config-enterprise-turbo.json" }
    "enterprise" { "config-enterprise-turbo.json" }
    "standard" { "config.json" }
}

if (Test-Path $configFile) {
    Copy-Item $configFile "config.json" -Force
    Write-Host "✅ Applied $Tier configuration" -ForegroundColor Green
} else {
    Write-Host "⚠️ Configuration file $configFile not found, using default" -ForegroundColor Yellow
}

# Set performance environment variables
Write-Host ""
Write-Host "⚡ Applying performance optimizations..." -ForegroundColor Blue

# Core performance settings
$env:TIER = $Tier
$env:GOMAXPROCS = [Environment]::ProcessorCount
Write-Host "   GOMAXPROCS: $env:GOMAXPROCS (using all CPU cores)" -ForegroundColor Gray

# GC tuning for ultra-low latency
if ($MaxPerformance) {
    # Maximum performance: disable GC during critical paths
    $env:GOGC = "off"
    Write-Host "   GOGC: OFF (maximum performance, monitor memory)" -ForegroundColor Yellow
} else {
    # Optimized for low latency with controlled memory usage
    $env:GOGC = "25"
    Write-Host "   GOGC: 25% (low-latency tuning)" -ForegroundColor Gray
}

# Runtime debugging for optimization
if ($ShowMetrics) {
    $env:GODEBUG = "gctrace=1,gcpacertrace=1"
    Write-Host "   GODEBUG: GC tracing enabled" -ForegroundColor Gray
} else {
    $env:GODEBUG = "gctrace=0"
    Write-Host "   GODEBUG: Clean runtime (production mode)" -ForegroundColor Gray
}

# API optimization
$env:RPC_NODES = "http://127.0.0.1:8332"
$env:RPC_USER = "bitcoin"
$env:RPC_PASS = "sprint123benchmark"
$env:API_PORT = "8080"

Write-Host ""
Write-Host "🔥 Performance Configuration Summary:" -ForegroundColor Cyan
Write-Host "   Tier: $Tier (ultra-low latency)" -ForegroundColor White
Write-Host "   CPU Cores: $env:GOMAXPROCS" -ForegroundColor White
Write-Host "   GC Setting: $env:GOGC" -ForegroundColor White
Write-Host "   Target SLA: ≤5ms with 99.9% compliance" -ForegroundColor White

# Build optimized binary if needed
$binaryName = "bitcoin-sprint-optimized.exe"
if (-not (Test-Path $binaryName) -or $MaxPerformance) {
    Write-Host ""
    Write-Host "🔨 Building performance-optimized binary..." -ForegroundColor Blue
    
    # Use nozmq build tags to avoid ZMQ compilation issues on Windows
    $buildArgs = @(
        "build"
        "-tags", "nozmq"
        "-ldflags=-s -w -extldflags=-static"
        "-trimpath"
        "-o", $binaryName
        ".\cmd\sprintd"
    )
    
    & go @buildArgs
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ Optimized binary built successfully" -ForegroundColor Green
    } else {
        Write-Host "⚠️ Build failed, using existing binary" -ForegroundColor Yellow
        $binaryName = "bitcoin-sprint-test.exe"
    }
}

# Launch Sprint with performance optimizations
Write-Host ""
Write-Host "🚀 Starting Bitcoin Sprint with performance optimizations..." -ForegroundColor Cyan

$process = Start-Process -FilePath $binaryName -PassThru -WindowStyle Hidden
Write-Host "✅ Bitcoin Sprint started (PID: $($process.Id))" -ForegroundColor Green

# Wait for startup
Write-Host "⏳ Waiting for API to become ready..." -ForegroundColor Blue
Start-Sleep 6

# Verify endpoints are accessible
$endpoints = @("/health", "/version", "/status")
$allReady = $true

foreach ($endpoint in $endpoints) {
    try {
        $response = Invoke-RestMethod "http://127.0.0.1:8080$endpoint" -TimeoutSec 3 -ErrorAction Stop
        Write-Host "   ✅ ${endpoint}: Ready" -ForegroundColor Green
    } catch {
        Write-Host "   ❌ ${endpoint}: Not ready ($($_.Exception.Message))" -ForegroundColor Red
        $allReady = $false
    }
}

Write-Host ""
if ($allReady) {
    Write-Host "🎉 BITCOIN SPRINT READY FOR 99.9% SLA TESTING!" -ForegroundColor Green
    Write-Host "=================================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "🧪 Run ZMQ SLA Test:" -ForegroundColor Cyan
    Write-Host "   Quick test:  .\tests\integration\real_zmq_sla_test.ps1 -Tier $Tier -QuickSeconds 30" -ForegroundColor White
    Write-Host "   Full test:   .\tests\integration\real_zmq_sla_test.ps1 -Tier $Tier" -ForegroundColor White
    Write-Host ""
    Write-Host "📊 Performance Monitoring:" -ForegroundColor Cyan
    if ($ShowMetrics) {
        Write-Host "   GC traces will be shown in console output" -ForegroundColor White
    }
    Write-Host "   Process ID: $($process.Id)" -ForegroundColor White
    Write-Host "   Binary: $binaryName" -ForegroundColor White
} else {
    Write-Host "⚠️ SOME ENDPOINTS NOT READY" -ForegroundColor Yellow
    Write-Host "Sprint may still be starting or there may be configuration issues." -ForegroundColor Yellow
}

Write-Host ""
Write-Host "🎯 TARGET: 99.9% SLA compliance with optimized runtime!" -ForegroundColor Cyan
