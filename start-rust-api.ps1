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

$psi = New-Object System.Diagnostics.ProcessStartInfo
$psi.FileName = $exe
$psi.WorkingDirectory = (Split-Path $exe -Parent)
$psi.UseShellExecute = $false
$psi.RedirectStandardOutput = $true
$psi.RedirectStandardError = $true
$psi.Environment['API_HOST'] = $env:API_HOST
$psi.Environment['API_PORT'] = $env:API_PORT
$psi.Environment['RUST_WEB_SERVER_ENABLED'] = $env:RUST_WEB_SERVER_ENABLED
# Default protocol flags to avoid noisy localhost connections
if (-not $env:ENABLE_BITCOIN)  { $env:ENABLE_BITCOIN  = 'true' }
if (-not $env:ENABLE_ETHEREUM) { $env:ENABLE_ETHEREUM = 'false' }
if (-not $env:ENABLE_SOLANA)   { $env:ENABLE_SOLANA   = 'false' }

$psi.Environment['ENABLE_BITCOIN']  = $env:ENABLE_BITCOIN
$psi.Environment['ENABLE_ETHEREUM'] = $env:ENABLE_ETHEREUM
$psi.Environment['ENABLE_SOLANA']   = $env:ENABLE_SOLANA

$proc = New-Object System.Diagnostics.Process
$proc.StartInfo = $psi
[void]$proc.Start()

# Async log handlers (attach handlers before starting read)
$stdOutHandler = [System.Diagnostics.DataReceivedEventHandler]{ param($s,$e) if($e.Data){ Add-Content -Path $logOut -Value $e.Data } }
$stdErrHandler = [System.Diagnostics.DataReceivedEventHandler]{ param($s,$e) if($e.Data){ Add-Content -Path $logErr -Value $e.Data } }
$proc.add_OutputDataReceived($stdOutHandler)
$proc.add_ErrorDataReceived($stdErrHandler)
$proc.BeginOutputReadLine()
$proc.BeginErrorReadLine()

Set-Content -Path $pidfile -Value $proc.Id
Write-Host ("Started bitcoin_sprint_api.exe (PID {0}) on {1}:{2}" -f $proc.Id,$env:API_HOST,$env:API_PORT) -ForegroundColor Green
