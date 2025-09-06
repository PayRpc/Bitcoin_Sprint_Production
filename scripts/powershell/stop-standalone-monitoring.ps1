# Stop standalone Grafana and Prometheus services
$ErrorActionPreference = "Stop"

$monitoringDir = ".\monitoring"
$binariesDir = "$monitoringDir\binaries"
$pidFile = "$binariesDir\monitoring.pid"

if (Test-Path $pidFile) {
    $pids = Get-Content $pidFile
    
    foreach ($pid in $pids) {
        try {
            $process = Get-Process -Id $pid -ErrorAction SilentlyContinue
            if ($process) {
                Write-Host "Stopping process with PID $pid..." -ForegroundColor Yellow
                Stop-Process -Id $pid -Force
                Write-Host "Process stopped." -ForegroundColor Green
            }
        } catch {
            Write-Host "Process with PID $pid not found or already stopped." -ForegroundColor Cyan
        }
    }
    
    # Remove PID file
    Remove-Item -Path $pidFile -Force
    
    Write-Host "All monitoring processes have been stopped." -ForegroundColor Green
} else {
    Write-Host "No monitoring processes found to stop." -ForegroundColor Yellow
}
