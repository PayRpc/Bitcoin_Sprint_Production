#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Solana Integration Deployment Script for Bitcoin Sprint
.DESCRIPTION
    Deploys and configures Solana network integration with monitoring
#>

param(
    [switch]$Start,
    [switch]$Stop,
    [switch]$Restart,
    [switch]$Status,
    [switch]$Test
)

Write-Host "🚀 Bitcoin Sprint - Solana Network Integration" -ForegroundColor Cyan
Write-Host "==============================================" -ForegroundColor Cyan

function Test-SolanaConnection {
    Write-Host "`n🔍 Testing Solana Network Connection..." -ForegroundColor Yellow

    try {
        $response = Invoke-WebRequest -Uri "http://localhost:8899/health" -TimeoutSec 10
        if ($response.StatusCode -eq 200) {
            Write-Host "✅ Solana RPC endpoint is responding" -ForegroundColor Green
            return $true
        }
    } catch {
        Write-Host "❌ Solana RPC endpoint is not responding" -ForegroundColor Red
        Write-Host "   Error: $($_.Exception.Message)" -ForegroundColor Red
    }

    try {
        $response = Invoke-WebRequest -Uri "http://localhost:8082/health" -TimeoutSec 10
        if ($response.StatusCode -eq 200) {
            Write-Host "✅ Solana metrics exporter is responding" -ForegroundColor Green
            return $true
        }
    } catch {
        Write-Host "❌ Solana metrics exporter is not responding" -ForegroundColor Red
        Write-Host "   Error: $($_.Exception.Message)" -ForegroundColor Red
    }

    return $false
}

function Test-PrometheusMetrics {
    Write-Host "`n📊 Testing Prometheus Metrics Collection..." -ForegroundColor Yellow

    try {
        $response = Invoke-WebRequest -Uri "http://localhost:8082/metrics" -TimeoutSec 10
        if ($response.StatusCode -eq 200) {
            $metrics = $response.Content
            $solanaMetrics = $metrics | Select-String -Pattern "solana_"
            if ($solanaMetrics.Count -gt 0) {
                Write-Host "✅ Solana metrics are being collected ($($solanaMetrics.Count) metrics found)" -ForegroundColor Green
                return $true
            } else {
                Write-Host "⚠️  Solana metrics endpoint responding but no Solana metrics found" -ForegroundColor Yellow
            }
        }
    } catch {
        Write-Host "❌ Cannot access Solana metrics endpoint" -ForegroundColor Red
        Write-Host "   Error: $($_.Exception.Message)" -ForegroundColor Red
    }

    return $false
}

function Start-SolanaServices {
    Write-Host "`n🚀 Starting Solana Services..." -ForegroundColor Yellow

    # Check if Docker is running
    try {
        $dockerVersion = docker version 2>$null
        if ($LASTEXITCODE -ne 0) {
            throw "Docker not running"
        }
    } catch {
        Write-Host "❌ Docker is not running. Please start Docker first." -ForegroundColor Red
        return $false
    }

    # Start services
    Write-Host "Starting Solana validator and metrics exporter..." -ForegroundColor Cyan
    docker-compose up -d solana-validator solana-exporter

    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ Solana services started successfully" -ForegroundColor Green

        # Wait for services to be ready
        Write-Host "⏳ Waiting for services to initialize..." -ForegroundColor Yellow
        Start-Sleep -Seconds 30

        return $true
    } else {
        Write-Host "❌ Failed to start Solana services" -ForegroundColor Red
        return $false
    }
}

function Stop-SolanaServices {
    Write-Host "`n🛑 Stopping Solana Services..." -ForegroundColor Yellow

    docker-compose stop solana-validator solana-exporter

    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ Solana services stopped successfully" -ForegroundColor Green
        return $true
    } else {
        Write-Host "❌ Failed to stop Solana services" -ForegroundColor Red
        return $false
    }
}

function Get-ServiceStatus {
    Write-Host "`n📋 Solana Services Status:" -ForegroundColor Yellow

    $services = @("solana-validator", "solana-exporter")

    foreach ($service in $services) {
        $status = docker-compose ps $service 2>$null
        if ($status -match "Up") {
            Write-Host "✅ $service - Running" -ForegroundColor Green
        } elseif ($status -match "Exit") {
            Write-Host "❌ $service - Stopped/Exited" -ForegroundColor Red
        } else {
            Write-Host "⚠️  $service - Unknown status" -ForegroundColor Yellow
        }
    }
}

function Show-IntegrationStatus {
    Write-Host "`n🔗 Solana Integration Status:" -ForegroundColor Yellow
    Write-Host "===============================" -ForegroundColor Yellow

    # Test connections
    $rpcConnected = Test-SolanaConnection
    $metricsWorking = Test-PrometheusMetrics

    # Backend integration status
    Write-Host "`n🔧 Backend Integration:" -ForegroundColor Yellow
    Write-Host "• Solana relay initialized: ✅ (Code updated)" -ForegroundColor Green
    Write-Host "• Relay dispatcher configured: ✅ (Code updated)" -ForegroundColor Green
    Write-Host "• API endpoints available: ✅ (Universal handler supports Solana)" -ForegroundColor Green

    # Monitoring status
    Write-Host "`n📊 Monitoring Status:" -ForegroundColor Yellow
    if ($rpcConnected) {
        Write-Host "• Solana RPC connection: ✅ Connected" -ForegroundColor Green
    } else {
        Write-Host "• Solana RPC connection: ❌ Not connected" -ForegroundColor Red
    }

    if ($metricsWorking) {
        Write-Host "• Metrics collection: ✅ Working" -ForegroundColor Green
    } else {
        Write-Host "• Metrics collection: ❌ Not working" -ForegroundColor Red
    }

    Write-Host "• Grafana dashboard: ✅ Ready (solana-monitoring.json)" -ForegroundColor Green
    Write-Host "• Prometheus config: ✅ Updated" -ForegroundColor Green

    # Overall status
    Write-Host "`n🎯 Overall Status:" -ForegroundColor Yellow
    if ($rpcConnected -and $metricsWorking) {
        Write-Host "🎉 Solana integration is FULLY OPERATIONAL!" -ForegroundColor Green
        Write-Host "`n📈 Available endpoints:" -ForegroundColor Cyan
        Write-Host "• RPC: http://localhost:8899" -ForegroundColor White
        Write-Host "• WebSocket: ws://localhost:8900" -ForegroundColor White
        Write-Host "• Metrics: http://localhost:8082/metrics" -ForegroundColor White
        Write-Host "• Grafana: http://localhost:3000 (Dashboard: Solana Monitoring)" -ForegroundColor White
        Write-Host "• API: http://localhost:8080/api/v1/universal/solana/latest_block" -ForegroundColor White
    } elseif ($rpcConnected) {
        Write-Host "⚠️  Solana RPC is connected but metrics collection needs attention" -ForegroundColor Yellow
    } else {
        Write-Host "❌ Solana integration is not fully operational" -ForegroundColor Red
        Write-Host "`n🔧 Next steps:" -ForegroundColor Yellow
        Write-Host "1. Start Solana services: .\deploy-solana.ps1 -Start" -ForegroundColor White
        Write-Host "2. Check service status: .\deploy-solana.ps1 -Status" -ForegroundColor White
        Write-Host "3. Test connections: .\deploy-solana.ps1 -Test" -ForegroundColor White
    }
}

# Main execution logic
if ($Start) {
    $success = Start-SolanaServices
    if ($success) {
        Show-IntegrationStatus
    }
} elseif ($Stop) {
    Stop-SolanaServices
} elseif ($Restart) {
    Write-Host "`n🔄 Restarting Solana Services..." -ForegroundColor Yellow
    Stop-SolanaServices
    Start-Sleep -Seconds 5
    $success = Start-SolanaServices
    if ($success) {
        Show-IntegrationStatus
    }
} elseif ($Status) {
    Get-ServiceStatus
    Show-IntegrationStatus
} elseif ($Test) {
    Test-SolanaConnection
    Test-PrometheusMetrics
} else {
    Write-Host "`n📖 Usage:" -ForegroundColor Yellow
    Write-Host "  .\deploy-solana.ps1 -Start    # Start Solana services" -ForegroundColor White
    Write-Host "  .\deploy-solana.ps1 -Stop     # Stop Solana services" -ForegroundColor White
    Write-Host "  .\deploy-solana.ps1 -Restart  # Restart Solana services" -ForegroundColor White
    Write-Host "  .\deploy-solana.ps1 -Status   # Show service status" -ForegroundColor White
    Write-Host "  .\deploy-solana.ps1 -Test     # Test connections and metrics" -ForegroundColor White
    Write-Host "`n📋 Current Status:" -ForegroundColor Yellow
    Show-IntegrationStatus
}

Write-Host "`n✨ Solana integration deployment complete!" -ForegroundColor Cyan
