# Bitcoin Sprint Port Conflict Resolution Script
# This script manages port conflicts between Docker and local services

param(
    [switch]$CheckConflicts,
    [switch]$StopLocalServices,
    [switch]$StartDocker,
    [switch]$VerifyServices,
    [switch]$ShowPorts
)

function Write-Header {
    param([string]$Message)
    Write-Host "`n=== $Message ===" -ForegroundColor Cyan
}

function Get-ListeningPorts {
    Write-Header "Current Listening Ports"
    $ports = netstat -ano | findstr LISTENING | ForEach-Object {
        $parts = $_ -split '\s+'
        [PSCustomObject]@{
            Protocol = $parts[0]
            LocalAddress = $parts[1]
            ForeignAddress = $parts[2]
            State = $parts[3]
            PID = $parts[4]
        }
    }
    $ports | Format-Table -AutoSize
    return $ports
}

function Check-PortConflicts {
    Write-Header "Port Conflict Analysis"

    $conflicts = @()
    $listening = netstat -ano | findstr LISTENING

    # Check for duplicate ports
    $portCounts = @{}
    foreach ($line in $listening) {
        $parts = $line -split '\s+'
        $address = $parts[1]
        if ($address -match ':(\d+)$') {
            $port = $matches[1]
            if ($portCounts.ContainsKey($port)) {
                $portCounts[$port]++
            } else {
                $portCounts[$port] = 1
            }
        }
    }

    foreach ($port in $portCounts.Keys) {
        if ($portCounts[$port] -gt 1) {
            Write-Host "CONFLICT: Port $port is used by multiple services" -ForegroundColor Red
            $conflicts += $port
        }
    }

    if ($conflicts.Count -eq 0) {
        Write-Host "No port conflicts detected" -ForegroundColor Green
    }

    return $conflicts
}

function Stop-LocalServices {
    Write-Header "Stopping Local Services"

    # Stop local PostgreSQL
    Write-Host "Stopping local PostgreSQL..."
    try {
        Stop-Service -Name "postgresql*" -ErrorAction SilentlyContinue
        Write-Host "Local PostgreSQL stopped" -ForegroundColor Green
    } catch {
        Write-Host "Local PostgreSQL not running or not installed" -ForegroundColor Yellow
    }

    # Stop local Redis
    Write-Host "Stopping local Redis..."
    try {
        Stop-Service -Name "*redis*" -ErrorAction SilentlyContinue
        Write-Host "Local Redis stopped" -ForegroundColor Green
    } catch {
        Write-Host "Local Redis not running or not installed" -ForegroundColor Yellow
    }
}

function Start-DockerServices {
    Write-Header "Starting Docker Services"

    $dockerComposePath = Join-Path $PSScriptRoot "config\docker-compose.yml"

    if (Test-Path $dockerComposePath) {
        Write-Host "Starting Docker services..."
        Push-Location (Join-Path $PSScriptRoot "config")
        try {
            docker-compose down
            docker-compose up -d
            Write-Host "Docker services started" -ForegroundColor Green
        } catch {
            Write-Host "Failed to start Docker services: $_" -ForegroundColor Red
        }
        Pop-Location
    } else {
        Write-Host "docker-compose.yml not found at $dockerComposePath" -ForegroundColor Red
    }
}

function Verify-Services {
    Write-Header "Service Verification"

    $services = @(
        @{Name = "Grafana"; Url = "http://localhost:3000/api/health"; ExpectedCode = 200},
        @{Name = "Bitcoin Sprint API"; Url = "http://localhost:8082/health"; ExpectedCode = 200},
        @{Name = "PostgreSQL"; Url = "localhost:5433"; IsPort = $true},
        @{Name = "Redis"; Url = "localhost:6380"; IsPort = $true}
    )

    foreach ($service in $services) {
        try {
            if ($service.IsPort) {
                $connection = Test-NetConnection -ComputerName "localhost" -Port $service.Url -WarningAction SilentlyContinue
                if ($connection.TcpTestSucceeded) {
                    Write-Host "$($service.Name): CONNECTED" -ForegroundColor Green
                } else {
                    Write-Host "$($service.Name): NOT CONNECTED" -ForegroundColor Red
                }
            } else {
                $response = Invoke-WebRequest -Uri $service.Url -TimeoutSec 5 -ErrorAction SilentlyContinue
                if ($response.StatusCode -eq $service.ExpectedCode) {
                    Write-Host "$($service.Name): OK" -ForegroundColor Green
                } else {
                    Write-Host "$($service.Name): HTTP $($response.StatusCode)" -ForegroundColor Yellow
                }
            }
        } catch {
            Write-Host "$($service.Name): ERROR - $($_.Exception.Message)" -ForegroundColor Red
        }
    }
}

function Show-PortMapping {
    Write-Header "Port Mapping Guide"

    Write-Host "Docker Services (External Ports):" -ForegroundColor Cyan
    Write-Host "  Grafana:              3000"
    Write-Host "  Bitcoin Sprint API:   8082"
    Write-Host "  Bitcoin Sprint Admin: 8083"
    Write-Host "  PostgreSQL:           5433"
    Write-Host "  Redis:                6380"
    Write-Host "  Prometheus:           9091"
    Write-Host ""

    Write-Host "Local Services (Direct Ports):" -ForegroundColor Cyan
    Write-Host "  Go Sprintd:           8080"
    Write-Host "  Local PostgreSQL:     5432"
    Write-Host "  Local Redis:          6379"
    Write-Host "  Next.js Dashboard:    3002"
    Write-Host "  FastAPI Gateway:      8000"
}

# Main execution logic
if ($CheckConflicts) {
    Check-PortConflicts
} elseif ($StopLocalServices) {
    Stop-LocalServices
} elseif ($StartDocker) {
    Start-DockerServices
} elseif ($VerifyServices) {
    Verify-Services
} elseif ($ShowPorts) {
    Show-PortMapping
} else {
    # Default: Show help
    Write-Header "Bitcoin Sprint Port Management"
    Write-Host "Usage: .\resolve-ports.ps1 [options]"
    Write-Host ""
    Write-Host "Options:"
    Write-Host "  -CheckConflicts    Check for port conflicts"
    Write-Host "  -StopLocalServices Stop local PostgreSQL/Redis services"
    Write-Host "  -StartDocker       Start Docker services with new port mapping"
    Write-Host "  -VerifyServices    Verify all services are running correctly"
    Write-Host "  -ShowPorts         Show port mapping guide"
    Write-Host ""
    Write-Host "Examples:"
    Write-Host "  .\resolve-ports.ps1 -CheckConflicts"
    Write-Host "  .\resolve-ports.ps1 -StopLocalServices -StartDocker"
    Write-Host "  .\resolve-ports.ps1 -VerifyServices"
}
