# Bitcoin Sprint Backend Manager Script
# This script provides a robust way to start and manage the Bitcoin Sprint backend

param (
    [Parameter(Mandatory=$false)]
    [ValidateSet('start', 'stop', 'restart', 'status')]
    [string]$Command = 'start',
    
    [Parameter(Mandatory=$false)]
    [ValidateSet('free', 'pro', 'enterprise')]
    [string]$Tier = 'enterprise',
    
    [Parameter(Mandatory=$false)]
    [int]$Port = 9090,
    
    [Parameter(Mandatory=$false)]
    [switch]$NoZMQ,
    
    [Parameter(Mandatory=$false)]
    [switch]$Optimized
)

$BackendProcess = "bitcoin-sprint-backend"
$BackendBinary = ".\$BackendProcess.exe"

function Get-BackendStatus {
    $processes = Get-Process -Name $BackendProcess -ErrorAction SilentlyContinue
    $listening = $null
    
    if ($processes) {
        try {
            $netstat = netstat -ano | findstr ":$Port "
            if ($netstat -match 'LISTENING') {
                $listening = $true
            }
        } catch {
            $listening = $false
        }
        
        return @{
            Running = $true
            ProcessCount = $processes.Count
            Processes = $processes
            Listening = $listening
            Port = $Port
        }
    } else {
        return @{
            Running = $false
            ProcessCount = 0
            Processes = $null
            Listening = $false
            Port = $Port
        }
    }
}

function Start-Backend {
    $status = Get-BackendStatus
    
    if ($status.Running) {
        Write-Host "Backend is already running with $($status.ProcessCount) process(es)" -ForegroundColor Yellow
        return
    }
    
    # Make sure the backend is built
    if (-Not (Test-Path $BackendBinary)) {
        Write-Host "Building optimized backend binary..." -ForegroundColor Yellow
        
        $buildFlags = "-ldflags=`"-s -w -extldflags=-static`" -trimpath"
        $buildTags = "cgo"
        
        if ($NoZMQ) {
            $buildTags += " nozmq"
        }
        
        $buildCmd = "go build $buildFlags -tags `"$buildTags`" -o $BackendBinary ./cmd/sprintd"
        Write-Host "Running: $buildCmd" -ForegroundColor Gray
        Invoke-Expression $buildCmd
        
        if (-Not (Test-Path $BackendBinary)) {
            Write-Host "❌ Failed to build backend binary" -ForegroundColor Red
            exit 1
        }
    }
    
    # Set environment variables
    $env:API_PORT = "$Port"
    $env:TIER = "$Tier"
    $env:API_HOST = "127.0.0.1"
    $env:API_KEY = "bitcoin-sprint-secure-key"
    
    if ($Optimized) {
        $env:OPTIMIZE_SYSTEM = "true"
    }
    
    # Start the backend process detached
    Write-Host "Starting Bitcoin Sprint backend ($Tier tier) on port $Port..." -ForegroundColor Green
    Start-Process -FilePath $BackendBinary -NoNewWindow -WorkingDirectory (Get-Location)
    
    # Wait a bit for the process to start
    Start-Sleep -Seconds 3
    
    # Check if the process started successfully
    $status = Get-BackendStatus
    
    if ($status.Running) {
        if ($status.Listening) {
            Write-Host "✅ Backend started successfully and is listening on port $Port" -ForegroundColor Green
        } else {
            Write-Host "⚠️ Backend process is running but not listening on port $Port" -ForegroundColor Yellow
        }
        
        Write-Host "`nRunning Bitcoin Sprint Backend processes:" -ForegroundColor Cyan
        $status.Processes | Format-Table Id, ProcessName, CPU, StartTime
        
        Write-Host "API endpoints available at: http://localhost:$Port/" -ForegroundColor Cyan
        Write-Host " - Health check: http://localhost:$Port/health" -ForegroundColor Cyan
        Write-Host " - Status: http://localhost:$Port/status" -ForegroundColor Cyan
        Write-Host " - Version: http://localhost:$Port/version" -ForegroundColor Cyan
    } else {
        Write-Host "❌ Failed to start backend" -ForegroundColor Red
    }
}

function Stop-Backend {
    $status = Get-BackendStatus
    
    if (-Not $status.Running) {
        Write-Host "Backend is not running" -ForegroundColor Yellow
        return
    }
    
    Write-Host "Stopping Bitcoin Sprint backend ($($status.ProcessCount) processes)..." -ForegroundColor Yellow
    Stop-Process -Name $BackendProcess -Force
    
    # Wait a bit for the process to stop
    Start-Sleep -Seconds 2
    
    $status = Get-BackendStatus
    if (-Not $status.Running) {
        Write-Host "✅ Backend stopped successfully" -ForegroundColor Green
    } else {
        Write-Host "❌ Failed to stop backend" -ForegroundColor Red
    }
}

function Show-Status {
    $status = Get-BackendStatus
    
    Write-Host "`nBitcoin Sprint Backend Status" -ForegroundColor Cyan
    Write-Host "===========================" -ForegroundColor Cyan
    
    if ($status.Running) {
        Write-Host "Status: " -NoNewline
        Write-Host "RUNNING" -ForegroundColor Green
        Write-Host "Process count: $($status.ProcessCount)"
        Write-Host "Listening on port: " -NoNewline
        Write-Host "$($status.Port)" -ForegroundColor Cyan -NoNewline
        Write-Host " - " -NoNewline
        
        if ($status.Listening) {
            Write-Host "YES" -ForegroundColor Green
        } else {
            Write-Host "NO" -ForegroundColor Red
        }
        
        Write-Host "`nRunning processes:" -ForegroundColor Yellow
        $status.Processes | Format-Table Id, ProcessName, CPU, StartTime
        
        # Check API endpoints
        Write-Host "API Endpoint Tests:" -ForegroundColor Yellow
        
        # Use curl directly since Invoke-RestMethod sometimes has issues with certain endpoints
        $healthResult = (curl -s "http://localhost:$($status.Port)/health" 2>$null)
        if ($healthResult) {
            Write-Host "  /health: " -NoNewline
            Write-Host "OK" -ForegroundColor Green -NoNewline
            Write-Host " - $healthResult"
        } else {
            Write-Host "  /health: " -NoNewline
            Write-Host "FAILED" -ForegroundColor Red
        }
        
        $statusResult = (curl -s "http://localhost:$($status.Port)/status" 2>$null)
        if ($statusResult) {
            Write-Host "  /status: " -NoNewline
            Write-Host "OK" -ForegroundColor Green -NoNewline
            Write-Host " - $statusResult"
        } else {
            Write-Host "  /status: " -NoNewline
            Write-Host "FAILED" -ForegroundColor Red
        }
    } else {
        Write-Host "Status: " -NoNewline
        Write-Host "STOPPED" -ForegroundColor Red
        Write-Host "No Bitcoin Sprint backend processes found"
        Write-Host "Use '.\backend-manager.ps1 start' to start the backend"
    }
}

# Main execution
switch ($Command) {
    'start' {
        Start-Backend
    }
    'stop' {
        Stop-Backend
    }
    'restart' {
        Stop-Backend
        Start-Sleep -Seconds 2
        Start-Backend
    }
    'status' {
        Show-Status
    }
}
