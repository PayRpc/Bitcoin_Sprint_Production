# Start the Bitcoin Sprint API server with robust HTTP connectivity settings
# PowerShell script
param (
    [int]$Port = 9000,
    [string]$Host = "0.0.0.0",
    [switch]$Debug,
    [switch]$Verbose
)

$ErrorActionPreference = "Stop"

Write-Host "🚀 Bitcoin Sprint API Server Starter" -ForegroundColor Cyan
Write-Host "=================================" -ForegroundColor Cyan

# Set environment variables for HTTP server configuration
$env:BITCOIN_SPRINT_API_HOST = $Host
$env:BITCOIN_SPRINT_API_PORT = $Port
$env:BITCOIN_SPRINT_LOG_LEVEL = if ($Debug) { "debug" } else { "info" }
$env:BITCOIN_SPRINT_VERBOSE = if ($Verbose) { "true" } else { "false" }

# Check if port is already in use
try {
    $tcpTest = New-Object System.Net.Sockets.TcpClient
    $tcpTest.Connect("127.0.0.1", $Port)
    $tcpTest.Close()
    Write-Host "⚠️ WARNING: Port $Port is already in use. The server may fail to start." -ForegroundColor Yellow
} catch {
    Write-Host "✓ Port $Port is available" -ForegroundColor Green
}

# Run network diagnostics first
Write-Host "`nRunning quick network diagnostics..." -ForegroundColor Cyan
try {
    & go run .\tools\network-diagnostics.go
} catch {
    Write-Host "Network diagnostics failed: $_" -ForegroundColor Red
}

# Wait for user confirmation
Write-Host "`nPress any key to start the Bitcoin Sprint API server..." -ForegroundColor Yellow
[void][System.Console]::ReadKey($true)

# Build and run the server
Write-Host "`nBuilding and starting Bitcoin Sprint API server..." -ForegroundColor Cyan
Write-Host "Binding to $Host:$Port" -ForegroundColor White

# Run server with enhanced visibility of errors
if ($Debug) {
    go build -o bitcoin-sprint-dev.exe
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Starting in debug mode with verbose output" -ForegroundColor Magenta
        .\bitcoin-sprint-dev.exe
    } else {
        Write-Host "Build failed" -ForegroundColor Red
    }
} else {
    Invoke-Expression "go run ."
}

# Check server status after startup
if ($LASTEXITCODE -ne 0) {
    Write-Host "Server failed to start or crashed. Exit code: $LASTEXITCODE" -ForegroundColor Red
    
    # Try to diagnose common issues
    Write-Host "`nRunning connection test..." -ForegroundColor Yellow
    & go run .\tools\http-connection-tester.go -host 127.0.0.1 -port $Port -path /health -v -retries 5
}
