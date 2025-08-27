# Bitcoin Sprint - Integrated Storage Verification Startup Script
# Starts both the Rust storage server and Next.js web interface

param(
    [switch]$Production,
    [switch]$StorageOnly,
    [switch]$WebOnly,
    [int]$StoragePort = 8080,
    [int]$WebPort = 3002
)

Write-Host "🚀 Bitcoin Sprint Storage Verification System" -ForegroundColor Green
Write-Host "=============================================" -ForegroundColor Green

$ErrorActionPreference = "Continue"

# Function to check if a port is available
function Test-PortAvailable {
    param([int]$Port)
    try {
        $listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Any, $Port)
        $listener.Start()
        $listener.Stop()
        return $true
    } catch {
        return $false
    }
}

# Function to start Rust storage server
function Start-StorageServer {
    Write-Host "`n📦 Starting Rust Storage Verification Server..." -ForegroundColor Yellow
    
    if (-not (Test-PortAvailable $StoragePort)) {
        Write-Host "   ⚠️  Port $StoragePort is already in use" -ForegroundColor Red
        Write-Host "   🔍 Checking if storage server is already running..." -ForegroundColor Yellow
        
        try {
            $response = Invoke-RestMethod -Uri "http://localhost:$StoragePort/health" -TimeoutSec 5
            if ($response.status -eq "healthy") {
                Write-Host "   ✅ Storage server is already running and healthy" -ForegroundColor Green
                return $true
            }
        } catch {
            Write-Host "   ❌ Port is occupied by another service" -ForegroundColor Red
            return $false
        }
    }
    
    # Check if Rust is available
    try {
        $cargoVersion = cargo --version 2>$null
        Write-Host "   🦀 Cargo found: $cargoVersion" -ForegroundColor Green
    } catch {
        Write-Host "   ❌ Cargo/Rust not found. Please install Rust." -ForegroundColor Red
        return $false
    }
    
    # Navigate to Rust directory
    $rustDir = "secure\rust"
    if (-not (Test-Path $rustDir)) {
        Write-Host "   ❌ Rust directory not found: $rustDir" -ForegroundColor Red
        return $false
    }
    
    Push-Location $rustDir
    
    try {
        Write-Host "   🔨 Building storage server..." -ForegroundColor Cyan
        
        if ($Production) {
            $buildResult = cargo build --release --bin storage_verifier_server --features web-server 2>&1
        } else {
            $buildResult = cargo build --bin storage_verifier_server --features web-server 2>&1
        }
        
        if ($LASTEXITCODE -ne 0) {
            Write-Host "   ❌ Build failed:" -ForegroundColor Red
            Write-Host $buildResult -ForegroundColor Red
            return $false
        }
        
        Write-Host "   ✅ Build successful" -ForegroundColor Green
        Write-Host "   🚀 Starting storage server on port $StoragePort..." -ForegroundColor Cyan
        
        # Start the server in background
        if ($Production) {
            Start-Process -FilePath "cargo" -ArgumentList @("run", "--release", "--bin", "storage_verifier_server", "--features", "web-server") -NoNewWindow
        } else {
            Start-Process -FilePath "cargo" -ArgumentList @("run", "--bin", "storage_verifier_server", "--features", "web-server") -NoNewWindow
        }
        
        # Wait for server to start
        Write-Host "   ⏳ Waiting for server to start..." -ForegroundColor Yellow
        for ($i = 1; $i -le 30; $i++) {
            Start-Sleep -Seconds 1
            try {
                $response = Invoke-RestMethod -Uri "http://localhost:$StoragePort/health" -TimeoutSec 2
                if ($response.status -eq "healthy") {
                    Write-Host "   ✅ Storage server is healthy and ready!" -ForegroundColor Green
                    return $true
                }
            } catch {
                Write-Host "." -NoNewline -ForegroundColor Yellow
            }
        }
        
        Write-Host "`n   ❌ Storage server failed to start within 30 seconds" -ForegroundColor Red
        return $false
        
    } finally {
        Pop-Location
    }
}

# Function to start Next.js web interface
function Start-WebInterface {
    Write-Host "`n🌐 Starting Next.js Web Interface..." -ForegroundColor Yellow
    
    if (-not (Test-PortAvailable $WebPort)) {
        Write-Host "   ⚠️  Port $WebPort is already in use" -ForegroundColor Red
        Write-Host "   🔍 Checking if web interface is already running..." -ForegroundColor Yellow
        
        try {
            $response = Invoke-RestMethod -Uri "http://localhost:$WebPort/api/health" -TimeoutSec 5
            if ($response.ok) {
                Write-Host "   ✅ Web interface is already running" -ForegroundColor Green
                return $true
            }
        } catch {
            Write-Host "   ❌ Port is occupied by another service" -ForegroundColor Red
            return $false
        }
    }
    
    # Navigate to web directory
    $webDir = "web"
    if (-not (Test-Path $webDir)) {
        Write-Host "   ❌ Web directory not found: $webDir" -ForegroundColor Red
        return $false
    }
    
    Push-Location $webDir
    
    try {
        # Check if Node.js is available
        try {
            $nodeVersion = node --version 2>$null
            Write-Host "   📦 Node.js found: $nodeVersion" -ForegroundColor Green
        } catch {
            Write-Host "   ❌ Node.js not found. Please install Node.js." -ForegroundColor Red
            return $false
        }
        
        # Check if dependencies are installed
        if (-not (Test-Path "node_modules")) {
            Write-Host "   📦 Installing dependencies..." -ForegroundColor Cyan
            npm install
            if ($LASTEXITCODE -ne 0) {
                Write-Host "   ❌ npm install failed" -ForegroundColor Red
                return $false
            }
        }
        
        Write-Host "   🚀 Starting web interface on port $WebPort..." -ForegroundColor Cyan
        
        # Start the web server in background
        if ($Production) {
            Start-Process -FilePath "npm" -ArgumentList @("run", "build") -NoNewWindow -Wait
            Start-Process -FilePath "npm" -ArgumentList @("start") -NoNewWindow
        } else {
            Start-Process -FilePath "npm" -ArgumentList @("run", "dev") -NoNewWindow
        }
        
        # Wait for web server to start
        Write-Host "   ⏳ Waiting for web interface to start..." -ForegroundColor Yellow
        for ($i = 1; $i -le 45; $i++) {
            Start-Sleep -Seconds 1
            try {
                $response = Invoke-RestMethod -Uri "http://localhost:$WebPort/api/health" -TimeoutSec 2
                if ($response.ok) {
                    Write-Host "   ✅ Web interface is ready!" -ForegroundColor Green
                    return $true
                }
            } catch {
                Write-Host "." -NoNewline -ForegroundColor Yellow
            }
        }
        
        Write-Host "`n   ❌ Web interface failed to start within 45 seconds" -ForegroundColor Red
        return $false
        
    } finally {
        Pop-Location
    }
}

# Main execution
Write-Host "`n🔧 Configuration:" -ForegroundColor Cyan
Write-Host "   Storage Port: $StoragePort" -ForegroundColor White
Write-Host "   Web Port: $WebPort" -ForegroundColor White
Write-Host "   Mode: $(if ($Production) { 'Production' } else { 'Development' })" -ForegroundColor White

$storageSuccess = $false
$webSuccess = $false

if (-not $WebOnly) {
    $storageSuccess = Start-StorageServer
}

if (-not $StorageOnly) {
    $webSuccess = Start-WebInterface
}

Write-Host "`n📋 Startup Summary:" -ForegroundColor Green
Write-Host "===================" -ForegroundColor Green

if (-not $WebOnly) {
    if ($storageSuccess) {
        Write-Host "✅ Rust Storage Server: http://localhost:$StoragePort" -ForegroundColor Green
        Write-Host "   📍 Health: http://localhost:$StoragePort/health" -ForegroundColor Gray
        Write-Host "   📊 Metrics: http://localhost:$StoragePort/metrics" -ForegroundColor Gray
        Write-Host "   🔐 Verify: POST http://localhost:$StoragePort/verify" -ForegroundColor Gray
    } else {
        Write-Host "❌ Rust Storage Server: Failed to start" -ForegroundColor Red
    }
}

if (-not $StorageOnly) {
    if ($webSuccess) {
        Write-Host "✅ Next.js Web Interface: http://localhost:$WebPort" -ForegroundColor Green
        Write-Host "   📍 Health: http://localhost:$WebPort/api/health" -ForegroundColor Gray
        Write-Host "   🔐 Storage API: http://localhost:$WebPort/api/storage/*" -ForegroundColor Gray
    } else {
        Write-Host "❌ Next.js Web Interface: Failed to start" -ForegroundColor Red
    }
}

if (($storageSuccess -or $WebOnly) -and ($webSuccess -or $StorageOnly)) {
    Write-Host "`n🎉 System is ready!" -ForegroundColor Green
    Write-Host "`n🧪 Run tests with:" -ForegroundColor Cyan
    Write-Host "   node test-storage-integration.js" -ForegroundColor White
    Write-Host "`n⏹️  Press Ctrl+C to stop all services" -ForegroundColor Yellow
} else {
    Write-Host "`n❌ System startup incomplete. Check the logs above." -ForegroundColor Red
}
