# Start metrics server for Docker integration
# This script ensures the metrics server is accessible from Docker containers

param (
    [int]$Port = 8081
)

$metricsServerPath = Join-Path $PSScriptRoot "metrics_server.exe"

# Check if metrics_server.exe exists, if not, build it
if (-not (Test-Path $metricsServerPath)) {
    Write-Host "Building metrics_server.exe..."
    Set-Location $PSScriptRoot
    & go build -o metrics_server.exe metrics_server.go
    if (-not $?) {
        Write-Error "Failed to build metrics_server.exe"
        exit 1
    }
}

# Check if metrics server is already running
$existingProcess = Get-Process -Name "metrics_server" -ErrorAction SilentlyContinue
if ($existingProcess) {
    Write-Host "Metrics server already running (PID: $($existingProcess.Id))"
} else {
    # Start the metrics server
    Write-Host "Starting metrics server on port $Port..."
    Start-Process -FilePath $metricsServerPath -WindowStyle Hidden
    Start-Sleep -Seconds 3
}

# Verify the server is running
$isRunning = netstat -ano | Select-String ":$Port" | Select-String "LISTENING"
if (-not $isRunning) {
    Write-Error "Failed to start metrics server on port $Port"
    exit 1
}

# Ensure Windows firewall allows this port for Docker access
$firewallRule = Get-NetFirewallRule -DisplayName "Bitcoin-Sprint-Metrics-$Port" -ErrorAction SilentlyContinue
if (-not $firewallRule) {
    Write-Host "Creating firewall rule to allow Docker containers to access metrics server..."
    New-NetFirewallRule -DisplayName "Bitcoin-Sprint-Metrics-$Port" -Direction Inbound -Action Allow -Protocol TCP -LocalPort $Port
}

Write-Host "Metrics server is running and accessible to Docker containers on port $Port"
