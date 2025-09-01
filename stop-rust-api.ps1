$ErrorActionPreference = 'Stop'

$workdir = Split-Path -Parent $MyInvocation.MyCommand.Path
$pidfile = Join-Path $workdir '.rust-api.pid'

if (!(Test-Path $pidfile)) {
    Write-Host 'No pidfile found; attempting best-effort shutdown...' -ForegroundColor Yellow
    # Try stopping by name if running from known path
    Get-Process -Name bitcoin_sprint_api -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
    exit 0
}

$pid = Get-Content $pidfile | Select-Object -First 1
try {
    if ($pid) {
        Stop-Process -Id $pid -Force -ErrorAction Stop
    }
} catch {
    Write-Host "Process $pid not running" -ForegroundColor Yellow
}

Remove-Item $pidfile -Force -ErrorAction SilentlyContinue
Write-Host "Stopped bitcoin_sprint_api.exe (PID $pid)" -ForegroundColor Green
