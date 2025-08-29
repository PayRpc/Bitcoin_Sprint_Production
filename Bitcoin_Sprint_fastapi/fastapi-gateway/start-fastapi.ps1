# FastAPI Gateway Startup Script for Windows
# This script sets up and starts the FastAPI gateway with all dependencies

param(
    [switch]$Clean,
    [switch]$Build,
    [switch]$Test,
    [switch]$Production,
    [string]$EnvFile = ".env"
)

$ErrorActionPreference = "Stop"

# Configuration
$ProjectRoot = Split-Path -Parent $PSScriptRoot
$GatewayDir = $PSScriptRoot
$PythonVersion = "3.11"

Write-Host "🚀 Bitcoin Sprint FastAPI Gateway Startup" -ForegroundColor Cyan
Write-Host "==============================================" -ForegroundColor Cyan

# Function to check if command exists
function Test-Command {
    param($Command)
    try {
        Get-Command $Command -ErrorAction Stop | Out-Null
        return $true
    } catch {
        return $false
    }
}

# Function to check Python version
function Test-PythonVersion {
    try {
        $version = python --version 2>&1
        if ($version -match "Python (\d+)\.(\d+)") {
            $major = [int]$matches[1]
            $minor = [int]$matches[2]
            if ($major -eq 3 -and $minor -ge 8) {
                Write-Host "✅ Python $version found" -ForegroundColor Green
                return $true
            }
        }
        Write-Host "❌ Python 3.8+ required, found: $version" -ForegroundColor Red
        return $false
    } catch {
        Write-Host "❌ Python not found in PATH" -ForegroundColor Red
        return $false
    }
}

# Function to setup virtual environment
function New-VirtualEnvironment {
    Write-Host "🔧 Setting up Python virtual environment..." -ForegroundColor Yellow

    if (Test-Path "venv") {
        if ($Clean) {
            Remove-Item -Recurse -Force "venv"
            Write-Host "🧹 Cleaned existing virtual environment" -ForegroundColor Yellow
        } else {
            Write-Host "✅ Virtual environment already exists" -ForegroundColor Green
            return
        }
    }

    python -m venv venv
    Write-Host "✅ Virtual environment created" -ForegroundColor Green
}

# Function to activate virtual environment and install dependencies
function Install-Dependencies {
    Write-Host "📦 Installing Python dependencies..." -ForegroundColor Yellow

    & ".\venv\Scripts\Activate.ps1"

    if ($Production) {
        pip install -r requirements.txt --no-dev
    } else {
        pip install -r requirements.txt
    }

    Write-Host "✅ Dependencies installed" -ForegroundColor Green
}

# Function to setup environment file
function New-EnvironmentFile {
    Write-Host "⚙️ Setting up environment configuration..." -ForegroundColor Yellow

    if (!(Test-Path $EnvFile)) {
        Copy-Item ".env.example" $EnvFile
        Write-Host "✅ Environment file created from template" -ForegroundColor Green
        Write-Host "⚠️ Please edit $EnvFile with your configuration" -ForegroundColor Yellow
    } else {
        Write-Host "✅ Environment file already exists" -ForegroundColor Green
    }
}

# Function to check Redis connection
function Test-RedisConnection {
    Write-Host "🔍 Checking Redis connection..." -ForegroundColor Yellow

    try {
        $redisHost = "localhost"
        $redisPort = 6379

        $tcpClient = New-Object System.Net.Sockets.TcpClient
        $tcpClient.Connect($redisHost, $redisPort)
        $tcpClient.Close()

        Write-Host "✅ Redis is running on $redisHost`:$redisPort" -ForegroundColor Green
        return $true
    } catch {
        Write-Host "❌ Redis not accessible on localhost:6379" -ForegroundColor Red
        Write-Host "💡 Make sure Redis is running or update REDIS_URL in $EnvFile" -ForegroundColor Yellow
        return $false
    }
}

# Function to check backend connection
function Test-BackendConnection {
    Write-Host "🔍 Checking backend connection..." -ForegroundColor Yellow

    try {
        $backendUrl = "http://localhost:8080/health"
        $response = Invoke-WebRequest -Uri $backendUrl -TimeoutSec 5 -ErrorAction Stop
        if ($response.StatusCode -eq 200) {
            Write-Host "✅ Backend is healthy" -ForegroundColor Green
            return $true
        } else {
            Write-Host "⚠️ Backend responded with status $($response.StatusCode)" -ForegroundColor Yellow
            return $false
        }
    } catch {
        Write-Host "❌ Backend not accessible on localhost:8080" -ForegroundColor Red
        Write-Host "💡 Make sure your Go backend is running" -ForegroundColor Yellow
        return $false
    }
}

# Function to start the application
function Start-FastAPIGateway {
    Write-Host "🚀 Starting FastAPI Gateway..." -ForegroundColor Yellow

    & ".\venv\Scripts\Activate.ps1"

    if ($Production) {
        Write-Host "🏭 Starting in production mode..." -ForegroundColor Cyan
        $env:PYTHONPATH = $GatewayDir
        python main.py
    } else {
        Write-Host "🔧 Starting in development mode..." -ForegroundColor Cyan
        $env:PYTHONPATH = $GatewayDir
        uvicorn main:app --host 0.0.0.0 --port 8000 --reload --log-level info
    }
}

# Function to run tests
function Start-Tests {
    Write-Host "🧪 Running tests..." -ForegroundColor Yellow

    & ".\venv\Scripts\Activate.ps1"

    if (Test-Path "tests") {
        python -m pytest tests/ -v
    } else {
        Write-Host "⚠️ No tests directory found" -ForegroundColor Yellow
        Write-Host "💡 Create tests/ directory with test files" -ForegroundColor Cyan
    }
}

# Main execution
try {
    Set-Location $GatewayDir

    # Pre-flight checks
    if (!(Test-PythonVersion)) {
        throw "Python version check failed"
    }

    # Setup environment
    New-VirtualEnvironment
    Install-Dependencies
    New-EnvironmentFile

    # Check dependencies
    $redisOk = Test-RedisConnection
    $backendOk = Test-BackendConnection

    if (!$redisOk -or !$backendOk) {
        Write-Host "⚠️ Some dependencies are not available" -ForegroundColor Yellow
        Write-Host "💡 The gateway will still start but some features may not work" -ForegroundColor Cyan
    }

    # Execute requested action
    if ($Test) {
        Start-Tests
    } elseif ($Build) {
        Write-Host "🔨 Build completed successfully!" -ForegroundColor Green
        exit 0
    } else {
        Start-FastAPIGateway
    }

} catch {
    Write-Host "❌ Error: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "💡 Check the error message above and try again" -ForegroundColor Yellow
    exit 1
}
