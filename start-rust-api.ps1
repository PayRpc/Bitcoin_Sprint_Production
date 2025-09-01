$ErrorActionPreference = 'Stop'

# Resolve working directory and binary path
$workdir = Split-Path -Parent $MyInvocation.MyCommand.Path
$exe = Join-Path $workdir 'secure\\rust\\target\\release\\bitcoin_sprint_api.exe'

# Ensure binary exists; if not, try building it
if (!(Test-Path $exe)) {
    Write-Host "Binary not found, building release..." -ForegroundColor Yellow
    pushd (Join-Path $workdir 'secure\\rust')
    cargo build --release --features axum-only --bin bitcoin_sprint_api | Out-Null
    popd
    if (!(Test-Path $exe)) { throw "Failed to build or locate $exe" }
}

# Environment
$env:API_HOST = $env:API_HOST -as [string]; if (-not $env:API_HOST) { $env:API_HOST = '127.0.0.1' }
$env:API_PORT = $env:API_PORT -as [string]; if (-not $env:API_PORT) { $env:API_PORT = '8443' }
$env:RUST_WEB_SERVER_ENABLED = $env:RUST_WEB_SERVER_ENABLED -as [string]; if (-not $env:RUST_WEB_SERVER_ENABLED) { $env:RUST_WEB_SERVER_ENABLED = 'true' }

$pidfile = Join-Path $workdir '.rust-api.pid'
$logdir  = Join-Path $workdir 'logs'
New-Item -ItemType Directory -Force -Path $logdir | Out-Null

# Start detached with redirected output
$logOut = Join-Path $logdir 'bitcoin_sprint_api.out.log'
$logErr = Join-Path $logdir 'bitcoin_sprint_api.err.log'
# If an old process is running, stop it to avoid file locks
if (Test-Path $pidfile) {
    try {
        $oldPid = Get-Content -Path $pidfile -ErrorAction Stop
        if ($oldPid) {
            $p = Get-Process -Id $oldPid -ErrorAction SilentlyContinue
            if ($p) { Stop-Process -Id $oldPid -Force -ErrorAction SilentlyContinue }
        }
    } catch {}
}

# Ensure env is present in current process so child inherits it
$env:API_HOST = $env:API_HOST
$env:API_PORT = $env:API_PORT
$env:RUST_WEB_SERVER_ENABLED = $env:RUST_WEB_SERVER_ENABLED
# Default protocol flags: enable all unless explicitly disabled
if (-not $env:ENABLE_BITCOIN)  { $env:ENABLE_BITCOIN  = 'true' }
if (-not $env:ENABLE_ETHEREUM) { $env:ENABLE_ETHEREUM = 'true' }
if (-not $env:ENABLE_SOLANA)   { $env:ENABLE_SOLANA   = 'true' }

# Optional seed overrides are already in $env for inheritance

# Launch detached via Start-Process with redirected logs
$proc = Start-Process -FilePath $exe \
    -WorkingDirectory (Split-Path $exe -Parent) \
    -WindowStyle Hidden \
    -RedirectStandardOutput $logOut \
    -RedirectStandardError $logErr \
    -PassThru

Set-Content -Path $pidfile -Value $proc.Id
Write-Host ("Started bitcoin_sprint_api.exe (PID {0}) on {1}:{2}" -f $proc.Id,$env:API_HOST,$env:API_PORT) -ForegroundColor Green
