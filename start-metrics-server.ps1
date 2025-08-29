# Ethereum Metrics Server startup script
# Automatically starts the metrics server on port 8081

Set-Location "c:\Projects\Bitcoin-Sprint-5\BItcoin_Sprint"

# Check if metrics_server is already running
$existingProcess = Get-Process -Name "metrics_server" -ErrorAction SilentlyContinue
if ($existingProcess) {
    Write-Host "Metrics server already running (PID: $($existingProcess.Id))"
    exit 0
}

# Build the metrics server as a standalone executable
Write-Host "Building metrics_server executable..."
& go build -o metrics_server.exe metrics_server.go

# Start the metrics server process
Write-Host "Starting metrics_server on port 8081..."
Start-Process -FilePath ".\metrics_server.exe" -WindowStyle Hidden

Write-Host "Waiting for metrics server to start..."
Start-Sleep -Seconds 5

# Verify the server is running
$isRunning = netstat -ano | Select-String ":8081" | Select-String "LISTENING"
if ($isRunning) {
    Write-Host "Metrics server successfully started and listening on port 8081"
} else {
    Write-Host "Failed to start metrics server or verify it's listening on port 8081"
}
