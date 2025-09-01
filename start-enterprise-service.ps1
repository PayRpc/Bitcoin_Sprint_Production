#!/usr/bin/env pwsh
# Bitcoin Sprint Enterprise Storage Validation Service Startup Script
# PowerShell script for Windows deployment

param(
    [switch]$Development,
    [switch]$Production,
    [switch]$Clean,
    [int]$Port = 8443
)

$ErrorActionPreference = "Stop"

# Configuration
$ProjectRoot = Split-Path -Parent $PSScriptRoot
$ConfigPath = Join-Path $ProjectRoot "config\enterprise-service.toml"
$LogPath = Join-Path $ProjectRoot "logs"
$CertPath = Join-Path $ProjectRoot "certs"

# Ensure directories exist
@($LogPath, $CertPath) | ForEach-Object {
    if (!(Test-Path $_)) {
        New-Item -ItemType Directory -Path $_ -Force | Out-Null
    }
}

function Write-Log {
    param([string]$Message, [string]$Level = "INFO")
    $Timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $LogMessage = "[$Timestamp] [$Level] $Message"
    Write-Host $LogMessage
    Add-Content -Path (Join-Path $LogPath "enterprise-startup.log") -Value $LogMessage
}

function Test-Dependencies {
    Write-Log "Checking dependencies..."

    # Check if Rust is installed
    try {
        $rustVersion = & rustc --version 2>$null
        Write-Log "Rust found: $rustVersion"
    } catch {
        Write-Log "Rust not found. Please install Rust from https://rustup.rs/" "ERROR"
        exit 1
    }

    # Check if Redis is running (for production)
    if ($Production) {
        try {
            $redis = & redis-cli ping 2>$null
            if ($redis -eq "PONG") {
                Write-Log "Redis is running"
            } else {
                Write-Log "Redis not responding. Starting Redis..." "WARN"
                Start-Process -FilePath "redis-server" -NoNewWindow
                Start-Sleep -Seconds 2
            }
        } catch {
            Write-Log "Redis not found. Installing Redis..." "WARN"
            # Note: In production, Redis should be properly installed
        }
    }
}

function Generate-Certificates {
    if (!(Test-Path (Join-Path $CertPath "server.crt"))) {
        Write-Log "Generating self-signed certificates..."

        # Generate private key
        & openssl genrsa -out (Join-Path $CertPath "server.key") 2048 2>$null

        # Generate certificate
        $subj = "/C=US/ST=State/L=City/O=Bitcoin Sprint/CN=localhost"
        & openssl req -new -x509 -key (Join-Path $CertPath "server.key") -out (Join-Path $CertPath "server.crt") -days 365 -subj $subj 2>$null

        Write-Log "Certificates generated"
    }
}

function Build-Project {
    param([string]$Mode = "release")

    Write-Log "Building project in $Mode mode..."

    Push-Location $ProjectRoot

    try {
        if ($Mode -eq "release") {
            & cargo build --release --features hardened,enterprise
        } else {
            & cargo build --features enterprise
        }

        if ($LASTEXITCODE -ne 0) {
            Write-Log "Build failed" "ERROR"
            exit 1
        }

        Write-Log "Build completed successfully"
    } finally {
        Pop-Location
    }
}

function Start-Service {
    param([int]$Port)

    Write-Log "Starting Enterprise Storage Validation Service on port $Port"

    $exePath = if ($Production) {
        Join-Path $ProjectRoot "target\release\bitcoin-sprint-enterprise.exe"
    } else {
        Join-Path $ProjectRoot "target\debug\bitcoin-sprint-enterprise.exe"
    }

    if (!(Test-Path $exePath)) {
        Write-Log "Executable not found: $exePath" "ERROR"
        exit 1
    }

    # Set environment variables
    $env:RUST_LOG = if ($Production) { "info" } else { "debug" }
    $env:ENTERPRISE_CONFIG = $ConfigPath
    $env:PORT = $Port

    # Start the service
    if ($Production) {
        # Production mode - run in background
        $process = Start-Process -FilePath $exePath -NoNewWindow -PassThru
        Write-Log "Service started with PID: $($process.Id)"

        # Wait a moment for startup
        Start-Sleep -Seconds 3

        # Check if process is still running
        if (!$process.HasExited) {
            Write-Log "Service is running successfully"
            Write-Log "Access the service at: https://localhost:$Port"
            Write-Log "Web interface: https://localhost:$Port/web/enterprise-storage-validation.html"
        } else {
            Write-Log "Service failed to start" "ERROR"
            exit 1
        }
    } else {
        # Development mode - run in foreground
        Write-Log "Starting in development mode (press Ctrl+C to stop)..."
        & $exePath
    }
}

function Stop-Service {
    Write-Log "Stopping Enterprise Storage Validation Service..."

    # Find and stop the process
    $process = Get-Process -Name "bitcoin-sprint-enterprise" -ErrorAction SilentlyContinue
    if ($process) {
        Stop-Process -Id $process.Id -Force
        Write-Log "Service stopped"
    } else {
        Write-Log "Service not running"
    }
}

function Clean-Build {
    Write-Log "Cleaning build artifacts..."

    Push-Location $ProjectRoot

    try {
        & cargo clean
        Remove-Item -Path "target" -Recurse -Force -ErrorAction SilentlyContinue
        Write-Log "Clean completed"
    } finally {
        Pop-Location
    }
}

# Main execution
try {
    Write-Log "=== Bitcoin Sprint Enterprise Service Startup ==="

    if ($Clean) {
        Clean-Build
        exit 0
    }

    Test-Dependencies

    if ($Production) {
        Generate-Certificates
        Build-Project -Mode "release"
        Start-Service -Port $Port
    } elseif ($Development) {
        Build-Project -Mode "debug"
        Start-Service -Port $Port
    } else {
        Write-Log "Please specify -Development or -Production"
        exit 1
    }

} catch {
    Write-Log "Startup failed: $($_.Exception.Message)" "ERROR"
    exit 1
}

Write-Log "=== Startup script completed ==="
