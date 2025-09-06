#!/usr/bin/env pwsh

# Unified Docker Compose Monitoring Stack for Bitcoin Sprint
# This script sets up and manages the monitoring stack

param(
    [switch]$Start,
    [switch]$Stop,
    [switch]$Restart,
    [switch]$Status,
    [switch]$Logs,
    [switch]$Dashboard
)

$composePath = "docker-compose.unified.yml"

function Show-Usage {
    Write-Host "Bitcoin Sprint Monitoring Stack Management" -ForegroundColor Cyan
    Write-Host "----------------------------------------" -ForegroundColor Cyan
    Write-Host "Usage:"
    Write-Host "  ./monitoring-manager.ps1 -Start     # Start the monitoring stack"
    Write-Host "  ./monitoring-manager.ps1 -Stop      # Stop the monitoring stack"
    Write-Host "  ./monitoring-manager.ps1 -Restart   # Restart the monitoring stack"
    Write-Host "  ./monitoring-manager.ps1 -Status    # Check status of containers"
    Write-Host "  ./monitoring-manager.ps1 -Logs      # Show logs from all containers"
    Write-Host "  ./monitoring-manager.ps1 -Dashboard # Open Grafana dashboard in browser"
}

function Check-Docker {
    try {
        docker --version | Out-Null
        return $true
    }
    catch {
        Write-Host "Error: Docker is not installed or not running." -ForegroundColor Red
        Write-Host "Please install Docker Desktop or ensure the Docker service is running." -ForegroundColor Red
        return $false
    }
}

function Start-MonitoringStack {
    Write-Host "Starting Bitcoin Sprint monitoring stack..." -ForegroundColor Green
    docker-compose -f $composePath up -d
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ Monitoring stack started successfully!" -ForegroundColor Green
        Write-Host "🔍 Prometheus available at: http://localhost:9090" -ForegroundColor Cyan
        Write-Host "📊 Grafana available at: http://localhost:3000" -ForegroundColor Cyan
        Write-Host "   Grafana login: admin / admin" -ForegroundColor Yellow
    }
    else {
        Write-Host "❌ Failed to start monitoring stack. Check the docker logs." -ForegroundColor Red
    }
}

function Stop-MonitoringStack {
    Write-Host "Stopping Bitcoin Sprint monitoring stack..." -ForegroundColor Yellow
    docker-compose -f $composePath down
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ Monitoring stack stopped successfully." -ForegroundColor Green
    }
    else {
        Write-Host "❌ Failed to stop monitoring stack." -ForegroundColor Red
    }
}

function Show-Status {
    Write-Host "📊 Bitcoin Sprint Monitoring Stack Status:" -ForegroundColor Cyan
    docker-compose -f $composePath ps
}

function Show-Logs {
    Write-Host "📜 Bitcoin Sprint Monitoring Stack Logs:" -ForegroundColor Cyan
    docker-compose -f $composePath logs --tail=100 -f
}

function Open-Dashboard {
    Write-Host "🌐 Opening Grafana dashboard in your browser..." -ForegroundColor Cyan
    Start-Process "http://localhost:3000"
    Write-Host "   Login with admin/admin credentials" -ForegroundColor Yellow
}

# Main execution
if (-not (Check-Docker)) {
    exit 1
}

if ($Start) {
    Start-MonitoringStack
}
elseif ($Stop) {
    Stop-MonitoringStack
}
elseif ($Restart) {
    Stop-MonitoringStack
    Start-Sleep -Seconds 3
    Start-MonitoringStack
}
elseif ($Status) {
    Show-Status
}
elseif ($Logs) {
    Show-Logs
}
elseif ($Dashboard) {
    Open-Dashboard
}
else {
    Show-Usage
}
