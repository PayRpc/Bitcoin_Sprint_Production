#!/usr/bin/env pwsh
# Keep only the essential optimized files and remove everything else

Write-Host "🎯 Keeping only essential optimized files" -ForegroundColor Cyan

# Files to KEEP (everything else will be removed)
$essentialFiles = @(
    "docker-compose.override.yml",  # The CPU-optimized compose file
    "Dockerfile.optimized",         # The healthy optimized Dockerfile  
    "prometheus.yml"                # The healthy Prometheus config
)

Write-Host "📋 Essential files to keep:" -ForegroundColor Yellow
foreach ($file in $essentialFiles) {
    if (Test-Path $file) {
        Write-Host "✅ $file" -ForegroundColor Green
    } else {
        Write-Host "❌ $file (MISSING)" -ForegroundColor Red
    }
}

# Remove the redundant files
$filesToRemove = @(
    "docker-compose.unified.yml",   # Not the CPU-optimized one
    "Dockerfile",                   # Keep only the optimized version
    "Dockerfile.rust"               # Not needed
)

Write-Host "`n🗑️ Removing redundant files:" -ForegroundColor Yellow
$removedCount = 0

foreach ($file in $filesToRemove) {
    if (Test-Path $file) {
        try {
            Remove-Item $file -Force
            Write-Host "✅ Removed: $file" -ForegroundColor Green
            $removedCount++
        }
        catch {
            Write-Host "❌ Failed to remove: $file - $($_.Exception.Message)" -ForegroundColor Red
        }
    } else {
        Write-Host "⚠️  File not found: $file" -ForegroundColor Yellow
    }
}

Write-Host "`n📊 Summary:" -ForegroundColor Cyan
Write-Host "✅ Files removed: $removedCount" -ForegroundColor Green
Write-Host "`n🎉 You now have only the essential optimized files:" -ForegroundColor Green
Write-Host "   • docker-compose.override.yml (CPU overload fix)" -ForegroundColor White
Write-Host "   • Dockerfile.optimized (healthy multi-stage build)" -ForegroundColor White  
Write-Host "   • prometheus.yml (healthy monitoring config)" -ForegroundColor White
