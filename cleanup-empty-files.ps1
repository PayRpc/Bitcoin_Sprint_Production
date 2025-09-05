#!/usr/bin/env pwsh
# Remove all 0-size files and fill up essential ones with basic content

Write-Host "🧹 Cleaning up empty files and filling essential ones" -ForegroundColor Cyan

# Files to fill with basic content instead of removing
$filesToFill = @{
    "config\service-config.toml" = @"
# Bitcoin Sprint Service Configuration
[service]
name = "bitcoin-sprint"
version = "1.0.0"
port = 8080

[logging]
level = "info"
format = "json"

[database]
type = "sqlite"
path = "data/bitcoin-sprint.db"
"@

    "targets.json" = @"
{
  "targets": [],
  "version": "1.0.0",
  "description": "Bitcoin Sprint build targets configuration"
}
"@
}

# Files to remove (empty placeholders not needed)
$filesToRemove = @(
    "test_config.go",
    "test_integration.go", 
    "test_tier_enforcement.go",
    "cmd\p2p\diagnostics\example.go",
    "cmd\p2p\diagnostics\integration_example.go"
)

Write-Host "`n📝 Filling essential files with basic content:" -ForegroundColor Green
$filledCount = 0

foreach ($file in $filesToFill.Keys) {
    if (Test-Path $file) {
        $fileInfo = Get-Item $file
        if ($fileInfo.Length -eq 0) {
            try {
                Set-Content -Path $file -Value $filesToFill[$file] -Encoding UTF8
                Write-Host "✅ Filled: $file" -ForegroundColor Green
                $filledCount++
            }
            catch {
                Write-Host "❌ Failed to fill: $file - $($_.Exception.Message)" -ForegroundColor Red
            }
        } else {
            Write-Host "⚠️  Skipped: $file (not empty)" -ForegroundColor Yellow
        }
    } else {
        Write-Host "⚠️  Not found: $file" -ForegroundColor Yellow
    }
}

Write-Host "`n🗑️  Removing unnecessary empty files:" -ForegroundColor Red
$removedCount = 0

foreach ($file in $filesToRemove) {
    if (Test-Path $file) {
        $fileInfo = Get-Item $file
        if ($fileInfo.Length -eq 0) {
            try {
                Remove-Item $file -Force
                Write-Host "🗑️  Removed: $file" -ForegroundColor Red
                $removedCount++
            }
            catch {
                Write-Host "❌ Failed to remove: $file - $($_.Exception.Message)" -ForegroundColor DarkRed
            }
        } else {
            Write-Host "⚠️  Skipped: $file (not empty)" -ForegroundColor Yellow
        }
    } else {
        Write-Host "⚠️  Not found: $file" -ForegroundColor Yellow
    }
}

Write-Host "`n📊 Summary:" -ForegroundColor Cyan
Write-Host "📝 Files filled with content: $filledCount" -ForegroundColor Green
Write-Host "🗑️  Empty files removed: $removedCount" -ForegroundColor Red

Write-Host "`n🎉 Empty file cleanup completed!" -ForegroundColor Green
Write-Host "Essential config files now have basic content, unnecessary empty files removed." -ForegroundColor White
