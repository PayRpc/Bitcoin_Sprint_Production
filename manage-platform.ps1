#!/usr/bin/env pwsh
# Bitcoin Sprint Multi-Chain Platform Management Script
# Comprehensive orchestration for enterprise-grade blockchain infrastructure

param(
    [Parameter(Mandatory = $true)]
    [ValidateSet("start", "stop", "restart", "status", "logs", "health", "scale", "backup", "restore", "update")]
    [string]$Action,
    
    [Parameter(Mandatory = $false)]
    [ValidateSet("all", "bitcoin-sprint", "bitcoin-core", "ethereum", "solana", "cosmos", "polkadot", "monitoring", "cache")]
    [string]$Service = "all",
    
    [Parameter(Mandatory = $false)]
    [switch]$Force,
    
    [Parameter(Mandatory = $false)]
    [string]$BackupPath = "./backups",
    
    [Parameter(Mandatory = $false)]
    [int]$Replicas = 1
)

# Configuration
$ComposeFile = "docker-compose.yml"
$ProjectName = "bitcoin-sprint"
$LogDir = "./logs"
$ConfigDir = "./config"

# Colors for output
$ColorGreen = "`e[32m"
$ColorYellow = "`e[33m"
$ColorRed = "`e[31m"
$ColorBlue = "`e[34m"
$ColorReset = "`e[0m"

function Write-ColorOutput {
    param($Message, $Color = $ColorReset)
    Write-Host "$Color$Message$ColorReset"
}

function Write-Header {
    param($Title)
    Write-ColorOutput "================================" $ColorBlue
    Write-ColorOutput $Title $ColorBlue
    Write-ColorOutput "================================" $ColorBlue
}

function Test-Prerequisites {
    Write-Header "Checking Prerequisites"
    
    # Check Docker
    try {
        $dockerVersion = docker --version
        Write-ColorOutput "✓ Docker: $dockerVersion" $ColorGreen
    }
    catch {
        Write-ColorOutput "✗ Docker not found. Please install Docker Desktop." $ColorRed
        exit 1
    }
    
    # Check Docker Compose
    try {
        $composeVersion = docker compose version
        Write-ColorOutput "✓ Docker Compose: $composeVersion" $ColorGreen
    }
    catch {
        Write-ColorOutput "✗ Docker Compose not found." $ColorRed
        exit 1
    }
    
    # Check compose file
    if (-not (Test-Path $ComposeFile)) {
        Write-ColorOutput "✗ Docker Compose file not found: $ComposeFile" $ColorRed
        exit 1
    }
    Write-ColorOutput "✓ Docker Compose file found" $ColorGreen
    
    # Check available disk space
    $diskSpace = Get-WmiObject -Class Win32_LogicalDisk | Where-Object { $_.DeviceID -eq "C:" }
    $freeSpaceGB = [math]::Round($diskSpace.FreeSpace / 1GB, 2)
    if ($freeSpaceGB -lt 20) {
        Write-ColorOutput "⚠ Warning: Low disk space ($freeSpaceGB GB). Recommended: 20+ GB" $ColorYellow
    } else {
        Write-ColorOutput "✓ Disk space: $freeSpaceGB GB available" $ColorGreen
    }
}

function Start-Services {
    param($ServiceName = "all")
    
    Write-Header "Starting Bitcoin Sprint Multi-Chain Platform"
    
    if ($ServiceName -eq "all") {
        Write-ColorOutput "Starting all services..." $ColorBlue
        docker compose -p $ProjectName up -d
    } else {
        Write-ColorOutput "Starting service: $ServiceName" $ColorBlue
        docker compose -p $ProjectName up -d $ServiceName
    }
    
    if ($LASTEXITCODE -eq 0) {
        Write-ColorOutput "✓ Services started successfully" $ColorGreen
        Start-Sleep 5
        Get-ServiceStatus
    } else {
        Write-ColorOutput "✗ Failed to start services" $ColorRed
        exit 1
    }
}

function Stop-Services {
    param($ServiceName = "all", $ForceStop = $false)
    
    Write-Header "Stopping Services"
    
    if ($ForceStop) {
        Write-ColorOutput "Force stopping all services..." $ColorYellow
        docker compose -p $ProjectName kill
    } elseif ($ServiceName -eq "all") {
        Write-ColorOutput "Gracefully stopping all services..." $ColorBlue
        docker compose -p $ProjectName down
    } else {
        Write-ColorOutput "Stopping service: $ServiceName" $ColorBlue
        docker compose -p $ProjectName stop $ServiceName
    }
    
    if ($LASTEXITCODE -eq 0) {
        Write-ColorOutput "✓ Services stopped successfully" $ColorGreen
    } else {
        Write-ColorOutput "✗ Failed to stop services" $ColorRed
    }
}

function Restart-Services {
    param($ServiceName = "all")
    
    Write-Header "Restarting Services"
    Stop-Services $ServiceName
    Start-Sleep 3
    Start-Services $ServiceName
}

function Get-ServiceStatus {
    Write-Header "Service Status"
    
    $services = docker compose -p $ProjectName ps --format json | ConvertFrom-Json
    
    if ($services) {
        foreach ($service in $services) {
            $name = $service.Service
            $status = $service.State
            $health = $service.Health
            
            $statusColor = switch ($status) {
                "running" { $ColorGreen }
                "exited" { $ColorRed }
                "restarting" { $ColorYellow }
                default { $ColorReset }
            }
            
            $healthInfo = if ($health) { " ($health)" } else { "" }
            Write-ColorOutput "  $name`: $status$healthInfo" $statusColor
        }
    } else {
        Write-ColorOutput "No services running" $ColorYellow
    }
    
    # Network status
    Write-ColorOutput "`nNetwork Status:" $ColorBlue
    try {
        $networks = docker network ls --filter "name=$ProjectName" --format "table {{.Name}}\t{{.Driver}}\t{{.Scope}}"
        if ($networks) {
            Write-Host $networks
        } else {
            Write-ColorOutput "No project networks found" $ColorYellow
        }
    } catch {
        Write-ColorOutput "Failed to get network status" $ColorRed
    }
}

function Get-ServiceLogs {
    param($ServiceName = "all")
    
    Write-Header "Service Logs"
    
    if ($ServiceName -eq "all") {
        docker compose -p $ProjectName logs --tail=50 -f
    } else {
        docker compose -p $ProjectName logs --tail=50 -f $ServiceName
    }
}

function Test-ServiceHealth {
    Write-Header "Health Check Results"
    
    $healthChecks = @{
        "Bitcoin Sprint API" = "http://localhost:8080/health"
        "Bitcoin Core RPC" = "http://localhost:8332"
        "Ethereum RPC" = "http://localhost:8545"
        "Solana RPC" = "http://localhost:8899"
        "Grafana" = "http://localhost:3000/api/health"
        "Prometheus" = "http://localhost:9091/-/healthy"
    }
    
    foreach ($check in $healthChecks.GetEnumerator()) {
        try {
            $response = Invoke-RestMethod -Uri $check.Value -TimeoutSec 5 -ErrorAction Stop
            Write-ColorOutput "✓ $($check.Key): Healthy" $ColorGreen
        } catch {
            Write-ColorOutput "✗ $($check.Key): Unhealthy" $ColorRed
        }
    }
}

function Scale-Services {
    param($ServiceName, $ReplicaCount)
    
    Write-Header "Scaling Services"
    
    if ($ServiceName -eq "all") {
        Write-ColorOutput "Scaling all scalable services to $ReplicaCount replicas..." $ColorBlue
        docker compose -p $ProjectName up -d --scale bitcoin-sprint=$ReplicaCount
    } else {
        Write-ColorOutput "Scaling $ServiceName to $ReplicaCount replicas..." $ColorBlue
        docker compose -p $ProjectName up -d --scale $ServiceName=$ReplicaCount
    }
    
    if ($LASTEXITCODE -eq 0) {
        Write-ColorOutput "✓ Services scaled successfully" $ColorGreen
    } else {
        Write-ColorOutput "✗ Failed to scale services" $ColorRed
    }
}

function Backup-Data {
    param($BackupPath)
    
    Write-Header "Creating Backup"
    
    $timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
    $backupDir = Join-Path $BackupPath "backup_$timestamp"
    
    if (-not (Test-Path $BackupPath)) {
        New-Item -ItemType Directory -Path $BackupPath -Force | Out-Null
    }
    
    New-Item -ItemType Directory -Path $backupDir -Force | Out-Null
    
    Write-ColorOutput "Creating backup in: $backupDir" $ColorBlue
    
    # Backup database
    Write-ColorOutput "Backing up PostgreSQL database..." $ColorBlue
    docker exec -t sprint-postgres pg_dump -U sprint sprint_db | Out-File -FilePath "$backupDir/database.sql" -Encoding UTF8
    
    # Backup Bitcoin data
    Write-ColorOutput "Backing up Bitcoin Core data..." $ColorBlue
    docker run --rm -v bitcoin-sprint_bitcoin-data:/data -v ${PWD}/${backupDir}:/backup alpine tar czf /backup/bitcoin-data.tar.gz -C /data .
    
    # Backup Ethereum data
    Write-ColorOutput "Backing up Ethereum data..." $ColorBlue
    docker run --rm -v bitcoin-sprint_ethereum-data:/data -v ${PWD}/${backupDir}:/backup alpine tar czf /backup/ethereum-data.tar.gz -C /data .
    
    # Backup configuration
    Write-ColorOutput "Backing up configuration files..." $ColorBlue
    Copy-Item -Path $ComposeFile -Destination $backupDir
    Copy-Item -Path "*.conf" -Destination $backupDir -ErrorAction SilentlyContinue
    Copy-Item -Path "prometheus.yml" -Destination $backupDir -ErrorAction SilentlyContinue
    
    Write-ColorOutput "✓ Backup completed: $backupDir" $ColorGreen
}

function Update-Services {
    Write-Header "Updating Services"
    
    Write-ColorOutput "Pulling latest images..." $ColorBlue
    docker compose -p $ProjectName pull
    
    Write-ColorOutput "Rebuilding custom images..." $ColorBlue
    docker compose -p $ProjectName build --no-cache
    
    Write-ColorOutput "Restarting services with new images..." $ColorBlue
    docker compose -p $ProjectName up -d --force-recreate
    
    Write-ColorOutput "✓ Services updated successfully" $ColorGreen
}

function Show-Usage {
    Write-Header "Bitcoin Sprint Multi-Chain Platform Management"
    
    Write-Host @"
Usage: .\manage-platform.ps1 -Action <action> [-Service <service>] [-Force] [-BackupPath <path>] [-Replicas <count>]

Actions:
  start     - Start services
  stop      - Stop services
  restart   - Restart services
  status    - Show service status
  logs      - Show service logs
  health    - Run health checks
  scale     - Scale services
  backup    - Backup data
  restore   - Restore from backup
  update    - Update service images

Services:
  all           - All services (default)
  bitcoin-sprint - Bitcoin Sprint API
  bitcoin-core  - Bitcoin Core node
  ethereum      - Ethereum Geth node
  solana        - Solana validator
  cosmos        - Cosmos Hub node
  polkadot      - Polkadot node
  monitoring    - Prometheus + Grafana
  cache         - Redis + PostgreSQL

Examples:
  .\manage-platform.ps1 -Action start
  .\manage-platform.ps1 -Action stop -Service bitcoin-core
  .\manage-platform.ps1 -Action logs -Service bitcoin-sprint
  .\manage-platform.ps1 -Action scale -Service bitcoin-sprint -Replicas 3
  .\manage-platform.ps1 -Action backup -BackupPath "./backups"

"@
}

# Main execution
try {
    Test-Prerequisites
    
    switch ($Action) {
        "start" { Start-Services $Service }
        "stop" { Stop-Services $Service $Force }
        "restart" { Restart-Services $Service }
        "status" { Get-ServiceStatus }
        "logs" { Get-ServiceLogs $Service }
        "health" { Test-ServiceHealth }
        "scale" { Scale-Services $Service $Replicas }
        "backup" { Backup-Data $BackupPath }
        "update" { Update-Services }
        default { Show-Usage }
    }
} catch {
    Write-ColorOutput "Error: $($_.Exception.Message)" $ColorRed
    exit 1
}
