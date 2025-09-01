# Comprehensive cleanup script for Bitcoin Sprint project
# Removes build artifacts, logs, and temporary files

Write-Host "🧹 Starting comprehensive cleanup..." -ForegroundColor Green
Write-Host "=====================================" -ForegroundColor Yellow

$cleanupStats = @{
    "exeFiles" = 0
    "logFiles" = 0
    "backupFiles" = 0
    "tempFiles" = 0
    "rustTarget" = 0
    "prometheusData" = 0
    "testFiles" = 0
}

# 1. Remove executable files (*.exe) - build artifacts
Write-Host "`n🔧 Removing executable files (*.exe)..." -ForegroundColor Cyan
Get-ChildItem -Path "." -Recurse -Filter "*.exe" -File | ForEach-Object {
    try {
        Remove-Item $_.FullName -Force
        $cleanupStats.exeFiles++
        Write-Host "   🗑️  Removed: $($_.Name)" -ForegroundColor Gray
    } catch {
        Write-Host "   ⚠️  Failed to remove: $($_.Name)" -ForegroundColor Red
    }
}

# 2. Remove backup files (*~, *.backup)
Write-Host "`n📁 Removing backup files..." -ForegroundColor Cyan
Get-ChildItem -Path "." -Recurse -Include "*~", "*.backup" -File | ForEach-Object {
    try {
        Remove-Item $_.FullName -Force
        $cleanupStats.backupFiles++
        Write-Host "   🗑️  Removed: $($_.Name)" -ForegroundColor Gray
    } catch {
        Write-Host "   ⚠️  Failed to remove: $($_.Name)" -ForegroundColor Red
    }
}

# 3. Remove log files (*.log)
Write-Host "`n📝 Removing log files..." -ForegroundColor Cyan
Get-ChildItem -Path "." -Recurse -Filter "*.log" -File | ForEach-Object {
    try {
        Remove-Item $_.FullName -Force
        $cleanupStats.logFiles++
        Write-Host "   🗑️  Removed: $($_.Name)" -ForegroundColor Gray
    } catch {
        Write-Host "   ⚠️  Failed to remove: $($_.Name)" -ForegroundColor Red
    }
}

# 4. Remove Rust target directory (build artifacts)
Write-Host "`n🦀 Removing Rust target directory..." -ForegroundColor Cyan
$rustTargetPath = ".\secure\rust\target"
if (Test-Path $rustTargetPath) {
    try {
        Remove-Item $rustTargetPath -Recurse -Force
        $cleanupStats.rustTarget++
        Write-Host "   🗑️  Removed: Rust target directory" -ForegroundColor Gray
    } catch {
        Write-Host "   ⚠️  Failed to remove Rust target directory" -ForegroundColor Red
    }
}

# 5. Remove Prometheus data (runtime data)
Write-Host "`n📊 Removing Prometheus data..." -ForegroundColor Cyan
$prometheusDataPath = ".\prometheus-2.45.0.windows-amd64\data"
if (Test-Path $prometheusDataPath) {
    try {
        Remove-Item $prometheusDataPath -Recurse -Force
        $cleanupStats.prometheusData++
        Write-Host "   🗑️  Removed: Prometheus data directory" -ForegroundColor Gray
    } catch {
        Write-Host "   ⚠️  Failed to remove Prometheus data" -ForegroundColor Red
    }
}

# 6. Remove temporary test files we created
Write-Host "`n🧪 Removing temporary test files..." -ForegroundColor Cyan
$testFiles = @(
    ".\test-entropy-proxy-simple.go",
    ".\test-entropy-proxy-mock.go",
    ".\demonstrate-entropy-proxy.go",
    ".\entropy-proxy-server.go"
)

foreach ($file in $testFiles) {
    if (Test-Path $file) {
        try {
            Remove-Item $file -Force
            $cleanupStats.testFiles++
            Write-Host "   🗑️  Removed: $(Split-Path $file -Leaf)" -ForegroundColor Gray
        } catch {
            Write-Host "   ⚠️  Failed to remove: $(Split-Path $file -Leaf)" -ForegroundColor Red
        }
    }
}

# 7. Remove other temporary files
Write-Host "`n🗂️  Removing other temporary files..." -ForegroundColor Cyan
Get-ChildItem -Path "." -Recurse -Include "*.tmp", "*.temp", "*.cache", ".DS_Store", "Thumbs.db" -File | ForEach-Object {
    try {
        Remove-Item $_.FullName -Force
        $cleanupStats.tempFiles++
        Write-Host "   🗑️  Removed: $($_.Name)" -ForegroundColor Gray
    } catch {
        Write-Host "   ⚠️  Failed to remove: $($_.Name)" -ForegroundColor Red
    }
}

# Calculate disk space saved (rough estimate)
Write-Host "`n📊 Cleanup Summary:" -ForegroundColor Green
Write-Host "==================" -ForegroundColor Yellow
Write-Host "Executable files removed: $($cleanupStats.exeFiles)" -ForegroundColor White
Write-Host "Backup files removed: $($cleanupStats.backupFiles)" -ForegroundColor White
Write-Host "Log files removed: $($cleanupStats.logFiles)" -ForegroundColor White
Write-Host "Temporary files removed: $($cleanupStats.tempFiles)" -ForegroundColor White
Write-Host "Rust target directories removed: $($cleanupStats.rustTarget)" -ForegroundColor White
Write-Host "Prometheus data directories removed: $($cleanupStats.prometheusData)" -ForegroundColor White
Write-Host "Test files removed: $($cleanupStats.testFiles)" -ForegroundColor White

$totalRemoved = $cleanupStats.exeFiles + $cleanupStats.backupFiles + $cleanupStats.logFiles + $cleanupStats.tempFiles + $cleanupStats.testFiles

Write-Host "`n✅ Total files/directories removed: $totalRemoved" -ForegroundColor Green
Write-Host "💾 Significant disk space freed!" -ForegroundColor Green
Write-Host "`n🎯 Cleanup completed successfully!" -ForegroundColor Green
