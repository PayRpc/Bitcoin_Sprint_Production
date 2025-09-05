#!/usr/bin/env pwsh
# Comprehensive cleanup of loose script files - keep only essentials

Write-Host "🧹 Cleaning up loose script files" -ForegroundColor Cyan
Write-Host "Keeping only essential build, start, and core functionality scripts..." -ForegroundColor Yellow

# Essential files to KEEP (everything else gets removed)
$essentialScripts = @(
    # Core build and development
    "build-optimized.ps1",        # Essential build script
    "start-dev.ps1",              # Essential development server
    
    # Core functionality
    "go.mod",                     # Go module file
    "go.sum",                     # Go dependencies
    "Makefile",                   # Build automation
    "Makefile.win"                # Windows build automation
)

# All script files to potentially remove
$allScripts = @(
    # PowerShell scripts
    "analyze-performance.ps1",
    "cache-manager.ps1", 
    "check-ports.ps1",
    "check-status.ps1",
    "cleanup-docker-builds.ps1",
    "cleanup-project.ps1",
    "cleanup-redundant-files.ps1",
    "cleanup-to-essentials.ps1",
    "deploy-fly.ps1",
    "entropy-performance-test.ps1",
    "generate-tls-certs.ps1",
    "handshake-validator.ps1",
    "maintain-gitignore.ps1",
    "monitoring-manager.ps1",
    "resolve-ports.ps1",
    "setup-fast-sync.ps1",
    "setup-grafana.ps1",
    "start-api-server.ps1",
    "start-enterprise-service.ps1",
    "start-monitoring.ps1",
    "start-rust-api.ps1",
    "switch-tier.ps1",
    "test-enterprise-service.ps1",
    "test-env-loading.ps1",
    "test-ethereum-connectivity.ps1",
    "test-monitoring-stack.ps1",
    "test-rust-integration.ps1",
    "test-tier-enforcement.ps1",
    "validate-deployment.ps1",
    
    # Shell scripts
    "deploy-fly.sh",
    "test-env-loading.sh", 
    "validate-deployment.sh",
    
    # Python scripts
    "solana-exporter.py",
    
    # Batch files
    "start-monitoring-fixed.bat",
    "start-monitoring.bat"
)

# Remove scripts that are not essential
$removedCount = 0
$keptCount = 0

Write-Host "`n🗑️  Removing unnecessary scripts:" -ForegroundColor Yellow

foreach ($script in $allScripts) {
    if ($essentialScripts -contains $script) {
        Write-Host "✅ Keeping: $script (essential)" -ForegroundColor Green
        $keptCount++
    }
    elseif (Test-Path $script) {
        try {
            Remove-Item $script -Force
            Write-Host "🗑️  Removed: $script" -ForegroundColor Red
            $removedCount++
        }
        catch {
            Write-Host "❌ Failed to remove: $script - $($_.Exception.Message)" -ForegroundColor DarkRed
        }
    }
}

# Show what essential files remain
Write-Host "`n📋 Essential files kept:" -ForegroundColor Green
foreach ($essential in $essentialScripts) {
    if (Test-Path $essential) {
        Write-Host "✅ $essential" -ForegroundColor Green
    }
}

Write-Host "`n📊 Cleanup Summary:" -ForegroundColor Cyan
Write-Host "🗑️  Scripts removed: $removedCount" -ForegroundColor Red
Write-Host "✅ Essential files kept: $keptCount" -ForegroundColor Green

Write-Host "`n🎉 Repository is now streamlined with only essential scripts!" -ForegroundColor Green
Write-Host "   • build-optimized.ps1 (for building)" -ForegroundColor White
Write-Host "   • start-dev.ps1 (for development)" -ForegroundColor White
Write-Host "   • Core Go files (go.mod, go.sum, Makefiles)" -ForegroundColor White
