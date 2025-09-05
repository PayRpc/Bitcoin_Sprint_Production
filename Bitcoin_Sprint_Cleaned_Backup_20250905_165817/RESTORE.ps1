#!/usr/bin/env pwsh
# Quick Restore Script for Bitcoin Sprint Cleaned Repository
# Created: September 5, 2025

$backupDir = "Bitcoin_Sprint_Cleaned_Backup_20250905_165817"
$scriptLocation = Split-Path -Parent $MyInvocation.MyCommand.Path

Write-Host "🔄 Bitcoin Sprint Cleaned Repository Restore Script" -ForegroundColor Cyan
Write-Host "Backup Location: $backupDir" -ForegroundColor Yellow

if (-not (Test-Path $backupDir)) {
    Write-Host "❌ Backup directory not found: $backupDir" -ForegroundColor Red
    Write-Host "Make sure you're running this from the Bitcoin_Sprint directory" -ForegroundColor Yellow
    exit 1
}

Write-Host "`n🛡️ This will restore your cleaned repository files." -ForegroundColor Green
Write-Host "Files to restore:" -ForegroundColor White
Write-Host "  • ServiceManager implementation (cmd/sprintd/main.go)" -ForegroundColor Gray
Write-Host "  • New internal packages (circuitbreaker, middleware, migrations, ratelimit)" -ForegroundColor Gray
Write-Host "  • Updated Go dependencies (go.mod, go.sum)" -ForegroundColor Gray
Write-Host "  • Filled configuration files" -ForegroundColor Gray
Write-Host "  • Essential scripts only" -ForegroundColor Gray

$confirmation = Read-Host "`nDo you want to continue? (y/N)"
if ($confirmation -ne 'y' -and $confirmation -ne 'Y') {
    Write-Host "Restore cancelled." -ForegroundColor Yellow
    exit 0
}

Write-Host "`n🔄 Restoring files..." -ForegroundColor Green

try {
    # Restore core directories
    Copy-Item "$backupDir\cmd\*" "cmd\" -Recurse -Force
    Write-Host "✅ Restored cmd/ directory" -ForegroundColor Green
    
    Copy-Item "$backupDir\internal\*" "internal\" -Recurse -Force  
    Write-Host "✅ Restored internal/ directory" -ForegroundColor Green
    
    Copy-Item "$backupDir\config\*" "config\" -Recurse -Force
    Write-Host "✅ Restored config/ directory" -ForegroundColor Green
    
    Copy-Item "$backupDir\monitoring\*" "monitoring\" -Recurse -Force
    Write-Host "✅ Restored monitoring/ directory" -ForegroundColor Green
    
    # Restore root files
    Copy-Item "$backupDir\go.mod" "go.mod" -Force
    Copy-Item "$backupDir\go.sum" "go.sum" -Force
    Copy-Item "$backupDir\targets.json" "targets.json" -Force
    Copy-Item "$backupDir\build-optimized.ps1" "build-optimized.ps1" -Force
    Copy-Item "$backupDir\start-dev.ps1" "start-dev.ps1" -Force
    Write-Host "✅ Restored essential root files" -ForegroundColor Green
    
    Write-Host "`n🎉 Restore completed successfully!" -ForegroundColor Green
    Write-Host "Your cleaned repository has been restored." -ForegroundColor White
    
    # Verify build works
    Write-Host "`n🔍 Verifying build..." -ForegroundColor Yellow
    $buildResult = & go build ./cmd/sprintd
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ Build verification successful!" -ForegroundColor Green
    } else {
        Write-Host "⚠️ Build verification failed - check dependencies" -ForegroundColor Yellow
    }
}
catch {
    Write-Host "❌ Restore failed: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "You may need to manually copy files from the backup directory." -ForegroundColor Yellow
}
