# Bitcoin Sprint Development Environment Startup Script
# This script starts all required services for the complete development environment

param(
    [switch]$Clean,
    [switch]$SkipFrontend,
    [switch]$SkipBackend,
    [switch]$SkipGrafana
)

$ErrorActionPreference = "Stop"

# Configuration
$ProjectRoot = Split-Path -Parent $PSScriptRoot
$WebDir = Join-Path $ProjectRoot "web"
$FastApiDir = Join-Path $ProjectRoot "Bitcoin_Sprint_fastapi\fastapi-gateway"

Write-Host "[ROCKET] Bitcoin Sprint Development Environment Startup" -ForegroundColor Cyan
Write-Host "==================================================" -ForegroundColor Cyan

# Pre-start validation
Write-Host "[VALIDATION] Running pre-start checks..." -ForegroundColor Yellow
& "$ProjectRoot\scripts\powershell\pre-start-validation.ps1"
if ($LASTEXITCODE -ne 0) {
    Write-Host "[ERROR] Pre-start validation failed!" -ForegroundColor Red
    exit 1
}
Write-Host "[OK] Pre-start validation passed!" -ForegroundColor Green

# Function to check if port is in use
function Test-Port {
    param($Port)
    try {
        $connection = New-Object System.Net.Sockets.TcpClient("localhost", $Port)
        $connection.Close()
        return $true
    } catch {
        return $false
    }
}

# Function to wait for service
function Wait-ForService {
    param($Url, $ServiceName, $Timeout = 30)
    Write-Host "[WAIT] Waiting for $ServiceName at $Url..." -ForegroundColor Yellow
    $count = 0
    while ($count -lt $Timeout) {
        try {
            $response = Invoke-WebRequest -Uri $Url -Method GET -TimeoutSec 5 -ErrorAction Stop
            if ($response.StatusCode -eq 200) {
                Write-Host "[OK] $ServiceName is ready!" -ForegroundColor Green
                return $true
            }
        } catch {
            # Service not ready yet
        }
        Start-Sleep -Seconds 2
        $count += 2
    }
    Write-Host "[ERROR] $ServiceName failed to start within $Timeout seconds" -ForegroundColor Red
    return $false
}

# Function to start Docker services
function Start-DockerServices {
    Write-Host "[DOCKER] Starting Docker services..." -ForegroundColor Blue

    # Create network if it doesn't exist
    try {
        docker network create sprint-network --subnet=172.20.0.0/16 2>$null
    } catch {
        # Network might already exist
    }

    # Start Grafana
    if (!$SkipGrafana) {
        Write-Host "[GRAFANA] Starting Grafana..." -ForegroundColor Blue
        try {
            docker-compose -f grafana-compose.yml up -d
            if (!(Wait-ForService "http://localhost:3000" "Grafana" 20)) {
                Write-Host "[WARN] Grafana may not be fully ready, but continuing..." -ForegroundColor Yellow
            }
        } catch {
            Write-Host "[ERROR] Failed to start Grafana: $_" -ForegroundColor Red
        }
    }

    # Start main services from config/docker-compose.yml
    if (Test-Path "config\docker-compose.yml") {
        Write-Host "[SERVICES] Starting main services..." -ForegroundColor Blue
        try {
            docker-compose -f config\docker-compose.yml up -d
            Write-Host "[OK] Main services started" -ForegroundColor Green
        } catch {
            Write-Host "[ERROR] Failed to start main services: $_" -ForegroundColor Red
        }
    }
}

# Function to start FastAPI backend
function Start-FastApiBackend {
    if ($SkipBackend) { return }

    Write-Host "[PYTHON] Starting FastAPI backend..." -ForegroundColor Blue

    if (!(Test-Path $FastApiDir)) {
        Write-Host "[ERROR] FastAPI directory not found: $FastApiDir" -ForegroundColor Red
        return
    }

    Push-Location $FastApiDir

    try {
        # Check if virtual environment exists
        if (!(Test-Path "venv")) {
            Write-Host "[VENV] Creating virtual environment..." -ForegroundColor Blue
            python -m venv venv
        }

        # Activate virtual environment and install dependencies
        Write-Host "[DEPS] Installing dependencies..." -ForegroundColor Blue
        & ".\venv\Scripts\python.exe" -m pip install --upgrade pip
        & ".\venv\Scripts\pip.exe" install -r requirements.txt

        # Start FastAPI server
        Write-Host "[START] Starting FastAPI server..." -ForegroundColor Blue
        $fastapiJob = Start-Job -ScriptBlock {
            param($FastApiDir)
            Push-Location $FastApiDir
            & ".\venv\Scripts\uvicorn.exe" main:app --host 0.0.0.0 --port 8000 --reload
        } -ArgumentList $FastApiDir

        # Wait for FastAPI to be ready
        if (Wait-ForService "http://localhost:8000/health" "FastAPI" 30) {
            Write-Host "[OK] FastAPI backend started successfully" -ForegroundColor Green
        } else {
            Write-Host "[ERROR] FastAPI backend failed to start" -ForegroundColor Red
        }

        # Store job for cleanup
        $script:FastApiJob = $fastapiJob

    } catch {
        Write-Host "[ERROR] Failed to start FastAPI: $_" -ForegroundColor Red
    } finally {
        Pop-Location
    }
}

# Function to start Next.js frontend
function Start-NextJsFrontend {
    if ($SkipFrontend) { return }

    Write-Host "[REACT] Starting Next.js frontend..." -ForegroundColor Blue

    if (!(Test-Path $WebDir)) {
        Write-Host "[ERROR] Web directory not found: $WebDir" -ForegroundColor Red
        return
    }

    Push-Location $WebDir

    try {
        # Install dependencies if needed
        if (!(Test-Path "node_modules")) {
            Write-Host "[NPM] Installing Node.js dependencies..." -ForegroundColor Blue
            npm install
        }

        # Start Next.js development server
        Write-Host "[START] Starting Next.js development server..." -ForegroundColor Blue
        $nextjsJob = Start-Job -ScriptBlock {
            param($WebDir)
            Push-Location $WebDir
            node --max-old-space-size=8192 .\node_modules\.bin\next dev -p 3002
        } -ArgumentList $WebDir

        # Wait for Next.js to be ready
        if (Wait-ForService "http://localhost:3002" "Next.js" 30) {
            Write-Host "[OK] Next.js frontend started successfully" -ForegroundColor Green
        } else {
            Write-Host "[ERROR] Next.js frontend failed to start" -ForegroundColor Red
        }

        # Store job for cleanup
        $script:NextJsJob = $nextjsJob

    } catch {
        Write-Host "[ERROR] Failed to start Next.js: $_" -ForegroundColor Red
    } finally {
        Pop-Location
    }
}

# Function to start Rust Web Server
function Start-RustWebServer {
    Write-Host "[RUST] Starting Rust Web Server..." -ForegroundColor Blue

    # Check if Rust binary exists
    $rustBinary = Join-Path $ProjectRoot "bin\bitcoin_sprint_api.exe"
    if (!(Test-Path $rustBinary)) {
        Write-Host "[WARN] Rust binary not found at $rustBinary" -ForegroundColor Yellow
        Write-Host "[INFO] Building Rust web server..." -ForegroundColor Blue

        Push-Location (Join-Path $ProjectRoot "secure\rust")
        try {
            # Build Rust binary with correct features
            & "cargo" build --release --bin bitcoin_sprint_api --features web-server
            if ($LASTEXITCODE -ne 0) {
                Write-Host "[ERROR] Failed to build Rust web server" -ForegroundColor Red
                return
            }

            # Copy binary to bin directory with correct name
            $targetBinary = Join-Path $PSScriptRoot "target\release\bitcoin_sprint_api.exe"
            if (Test-Path $targetBinary) {
                Copy-Item $targetBinary $rustBinary -Force
                Write-Host "[OK] Rust binary built and copied" -ForegroundColor Green
            } else {
                Write-Host "[ERROR] Built binary not found at $targetBinary" -ForegroundColor Red
                return
            }
        } catch {
            Write-Host "[ERROR] Failed to build Rust web server: $_" -ForegroundColor Red
            return
        } finally {
            Pop-Location
        }
    }

    # Set environment variables for Rust server
    $env:RUST_LOG = "info"
    $env:API_HOST = "0.0.0.0"
    $env:API_PORT = "8443"
    $env:ADMIN_PORT = "8444"
    $env:PROMETHEUS_PORT = "9092"
    $env:TLS_CERT_PATH = Join-Path $ProjectRoot "config\tls\cert.pem"
    $env:TLS_KEY_PATH = Join-Path $ProjectRoot "config\tls\key.pem"
    $env:REDIS_URL = "redis://localhost:6379"
    $env:STORAGE_VERIFICATION_ENABLED = "true"
    $env:ENTERPRISE_MODE = "true"

    # Start Rust web server
    Write-Host "[START] Starting Rust web server..." -ForegroundColor Blue
    try {
        $rustJob = Start-Job -ScriptBlock {
            param($BinaryPath)
            & $BinaryPath
        } -ArgumentList $rustBinary

        # Wait for Rust server to be ready
        if (Wait-ForService "https://localhost:8443/health" "Rust Web Server" 30) {
            Write-Host "[OK] Rust web server started successfully" -ForegroundColor Green
        } else {
            Write-Host "[WARN] Rust web server may not be fully ready, but continuing..." -ForegroundColor Yellow
        }

        # Store job for cleanup
        $script:RustJob = $rustJob

    } catch {
        Write-Host "[ERROR] Failed to start Rust web server: $_" -ForegroundColor Red
    }
}

# Function to show status
function Show-Status {
    Write-Host "`n[STATUS] Service Status:" -ForegroundColor Cyan
    Write-Host "==================" -ForegroundColor Cyan

    $services = @(
        @{Name="Grafana"; Url="http://localhost:3000"; Port=3000},
        @{Name="FastAPI"; Url="http://localhost:8000/health"; Port=8000},
        @{Name="Next.js"; Url="http://localhost:3002"; Port=3002},
        @{Name="Rust Web Server"; Url="https://localhost:8443/health"; Port=8443}
    )

    foreach ($service in $services) {
        $status = if (Test-Port $service.Port) { "[RUNNING]" } else { "[STOPPED]" }
        Write-Host "$($service.Name): $status - $($service.Url)" -ForegroundColor (if ($status -like "*RUNNING*") { "Green" } else { "Red" })
    }

    Write-Host "`n[LINKS] Useful URLs:" -ForegroundColor Cyan
    Write-Host "Frontend: http://localhost:3002" -ForegroundColor White
    Write-Host "FastAPI: http://localhost:8000/docs" -ForegroundColor White
    Write-Host "Grafana: http://localhost:3000 (admin/sprint123)" -ForegroundColor White
    Write-Host "Rust Web Server: https://localhost:8443/health" -ForegroundColor White
    Write-Host "Rust Admin API: https://localhost:8444/admin" -ForegroundColor White
}

# Function to cleanup on exit
function Cleanup {
    Write-Host "`n[CLEANUP] Cleaning up..." -ForegroundColor Yellow

    if ($script:FastApiJob) {
        Write-Host "Stopping FastAPI job..." -ForegroundColor Blue
        Stop-Job $script:FastApiJob -ErrorAction SilentlyContinue
        Remove-Job $script:FastApiJob -ErrorAction SilentlyContinue
    }

    if ($script:NextJsJob) {
        Write-Host "Stopping Next.js job..." -ForegroundColor Blue
        Stop-Job $script:NextJsJob -ErrorAction SilentlyContinue
        Remove-Job $script:NextJsJob -ErrorAction SilentlyContinue
    }

    if ($script:RustJob) {
        Write-Host "Stopping Rust web server job..." -ForegroundColor Blue
        Stop-Job $script:RustJob -ErrorAction SilentlyContinue
        Remove-Job $script:RustJob -ErrorAction SilentlyContinue
    }

    Write-Host "[OK] Cleanup complete" -ForegroundColor Green
}

# Main execution
try {
    # Clean if requested
    if ($Clean) {
        Write-Host "[CLEAN] Cleaning up previous instances..." -ForegroundColor Yellow
        Cleanup
        Start-Sleep -Seconds 2
    }

    # Start services in order
    Start-DockerServices
    Start-Sleep -Seconds 5  # Give Docker services time to start

    Start-FastApiBackend
    Start-Sleep -Seconds 3

    Start-NextJsFrontend
    Start-Sleep -Seconds 3

    Start-RustWebServer

    # Show final status
    Start-Sleep -Seconds 5
    Show-Status

    Write-Host "`n[SUCCESS] Development environment is ready!" -ForegroundColor Green
    Write-Host "Press Ctrl+C to stop all services" -ForegroundColor White

    # Keep script running
    try {
        while ($true) {
            Start-Sleep -Seconds 10
            # Could add health checks here
        }
    } catch {
        Write-Host "`n[STOP] Shutdown requested..." -ForegroundColor Yellow
    }

} catch {
    Write-Host "[ERROR] Error during startup: $_" -ForegroundColor Red
} finally {
    Cleanup
}

Write-Host "[BYE] Development environment stopped" -ForegroundColor Cyan
