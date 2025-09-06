#!/usr/bin/env pwsh
# Cleanup script to remove redundant and unnecessary files from Bitcoin Sprint repository

Write-Host "🧹 Bitcoin Sprint Repository Cleanup" -ForegroundColor Cyan
Write-Host "Removing redundant YAML, PowerShell, and other unnecessary files..." -ForegroundColor Yellow

# Files to remove - duplicates and unnecessary configurations
$filesToRemove = @(
    # Duplicate/redundant Docker Compose files (keep docker-compose.override.yml and docker-compose.unified.yml)
    "docker-compose.emergency-limits.yml",
    "docker-compose.exporter.yml", 
    "docker-compose.grafana.yml",
    "docker-compose.monitoring.yml",
    "docker-compose.resource-fix.yml",
    "docker-compose.resource-limits.yml",
    "docker-compose.simple.yml",
    "grafana-compose.yml",
    "monitoring-compose.yml",
    "simple-monitoring.yml",
    
    # Redundant monitoring scripts (keep start-monitoring.ps1)
    "start-monitoring-fixed.ps1",
    "start-monitoring-network.ps1", 
    "start-monitoring-new.ps1",
    "start-standalone-monitoring.ps1",
    "stop-standalone-monitoring.ps1",
    
    # Redundant test scripts (keep core ones)
    "test-axum-server.ps1",
    "test-blockchain-network.ps1", 
    "test-dedupe.ps1",
    "test-relay-connections.ps1",
    "test-relay-stability.ps1",
    "speed-handshake-test.ps1",
    "run-tier-test.ps1",
    
    # Empty or minimal scripts
    "cicd-manager.ps1",
    "stop-rust-api.ps1",
    "test-entropy-speed.ps1",
    
    # Redundant performance scripts (keep analyze-performance.ps1)
    "performance-test.ps1",
    "tier-performance-comparison.ps1",
    
    # Redundant validation scripts (keep validate-deployment.ps1)
    "validate-tier-enforcement.ps1",
    
    # Redundant setup scripts (keep essential ones)
    "emergency-resource-fix.ps1"
)

$removedCount = 0
$failedCount = 0

foreach ($file in $filesToRemove) {
    if (Test-Path $file) {
        try {
            Remove-Item $file -Force
            Write-Host "✅ Removed: $file" -ForegroundColor Green
            $removedCount++
        }
        catch {
            Write-Host "❌ Failed to remove: $file - $($_.Exception.Message)" -ForegroundColor Red
            $failedCount++
        }
    } else {
        Write-Host "⚠️  File not found: $file" -ForegroundColor Yellow
    }
}

# Clean up redundant monitoring directories if they exist
$dirsToClean = @(
    "monitoring\prometheus-backups",
    "monitoring\grafana-old", 
    "monitoring\temp"
)

foreach ($dir in $dirsToClean) {
    if (Test-Path $dir) {
        try {
            Remove-Item $dir -Recurse -Force
            Write-Host "✅ Removed directory: $dir" -ForegroundColor Green
            $removedCount++
        }
        catch {
            Write-Host "❌ Failed to remove directory: $dir - $($_.Exception.Message)" -ForegroundColor Red
            $failedCount++
        }
    }
}

Write-Host "`n📊 Cleanup Summary:" -ForegroundColor Cyan
Write-Host "✅ Files/directories removed: $removedCount" -ForegroundColor Green
Write-Host "❌ Failed removals: $failedCount" -ForegroundColor Red

if ($failedCount -eq 0) {
    Write-Host "`n🎉 Repository cleanup completed successfully!" -ForegroundColor Green
} else {
    Write-Host "`n⚠️  Repository cleanup completed with some failures." -ForegroundColor Yellow
}

# Show remaining key files
Write-Host "`n📁 Key files remaining:" -ForegroundColor Cyan
Write-Host "Docker Compose: docker-compose.override.yml, docker-compose.unified.yml" -ForegroundColor White
Write-Host "Monitoring: prometheus.yml, monitoring-manager.ps1" -ForegroundColor White
Write-Host "Scripts: build-optimized.ps1, start-dev.ps1, validate-deployment.ps1" -ForegroundColor White
