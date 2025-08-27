#!/usr/bin/env pwsh
# run-integration-tests.ps1 - Local integration test runner for Bitcoin Sprint

param(
    [switch]$UseBitcoinCoreDocker = $true,
    [switch]$BuildBinary = $true
)

$ErrorActionPreference = "Stop"

Write-Host "Bitcoin Sprint Integration Test Runner" -ForegroundColor Cyan
Write-Host "=====================================" -ForegroundColor Cyan

# Check if Docker is available (if we're using it)
if ($UseBitcoinCoreDocker) {
    try {
        $dockerVersion = docker --version
        Write-Host "Using Docker: $dockerVersion" -ForegroundColor Green
    }
    catch {
        Write-Error "Docker not found. Install Docker or use -UseBitcoinCoreDocker:`$false if you have Bitcoin Core running locally."
    }

    # Stop any existing container
    docker rm -f bitcoin-sprint-test 2>&1 | Out-Null

    # Start Bitcoin Core container
    Write-Host "Starting Bitcoin Core container..." -ForegroundColor Yellow
    docker run -d --name bitcoin-sprint-test `
        -p 8332:8332 -p 28332:28332 `
        -e BITCOIN_RPCUSER=sprint `
        -e BITCOIN_RPCPASSWORD=integration `
        -e BITCOIN_EXTRA_ARGS="-server=1 -txindex=0 -prune=550 -rpcallowip=0.0.0.0/0 -rpcbind=0.0.0.0:8332 -zmqpubhashblock=tcp://0.0.0.0:28332 -fallbackfee=0.0002 -regtest=1" `
        ruimarinho/bitcoin-core:24.2
    
    Write-Host "Waiting for Bitcoin Core to start..." -ForegroundColor Yellow
    Start-Sleep -Seconds 5

    # Verify Bitcoin Core is running
    try {
        $result = docker exec bitcoin-sprint-test bitcoin-cli -rpcuser=sprint -rpcpassword=integration getblockchaininfo
        Write-Host "Bitcoin Core started successfully" -ForegroundColor Green
    }
    catch {
        Write-Error "Failed to verify Bitcoin Core is running. Error: $_"
    }
}

# Build binary if requested
if ($BuildBinary) {
    Write-Host "Building Bitcoin Sprint..." -ForegroundColor Yellow
    $env:CGO_ENABLED = "1"
    go build -o bitcoin-sprint.exe ./cmd/sprintd
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Build failed with exit code $LASTEXITCODE"
    }
    Write-Host "Build completed successfully" -ForegroundColor Green
}

# Start Bitcoin Sprint in background
Write-Host "Starting Bitcoin Sprint..." -ForegroundColor Yellow
$sprinterProcess = Start-Process -FilePath ".\bitcoin-sprint.exe" -ArgumentList "--debug" -PassThru -WindowStyle Hidden

try {
    Write-Host "Waiting for Bitcoin Sprint to initialize..." -ForegroundColor Yellow
    Start-Sleep -Seconds 5

    # Set environment variables for tests
    $env:BTC_RPC_HOST = "127.0.0.1:8332"
    $env:BTC_RPC_USER = "sprint"
    $env:BTC_RPC_PASS = "integration"
    $env:BTC_RPC_DISABLE_TLS = "true"
    $env:ZMQ_ENDPOINT = "tcp://127.0.0.1:28332"
    $env:API_HOST = "127.0.0.1"
    $env:API_PORT = "8080"
    $env:PEER_LISTEN_PORT = "8335"
    $env:LICENSE_KEY = "test_license_key"

    # Run integration tests
    Write-Host "Running integration tests..." -ForegroundColor Yellow
    go test ./tests/integration -v -timeout=1m
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Integration tests PASSED!" -ForegroundColor Green
    } else {
        Write-Host "Integration tests FAILED!" -ForegroundColor Red
    }
}
finally {
    # Clean up
    Write-Host "Cleaning up..." -ForegroundColor Yellow
    
    # Stop Bitcoin Sprint
    if ($sprinterProcess -ne $null) {
        Stop-Process -Id $sprinterProcess.Id -Force -ErrorAction SilentlyContinue
    }

    # Stop Bitcoin Core container
    if ($UseBitcoinCoreDocker) {
        docker rm -f bitcoin-sprint-test 2>&1 | Out-Null
    }
    
    Write-Host "Done!" -ForegroundColor Cyan
}
