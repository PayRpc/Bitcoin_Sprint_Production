# Docker Build Cleanup Script for Bitcoin Sprint
# This script removes temporary files, build artifacts, and cache directories
# to clean up the workspace after Docker builds

Write-Host "🧹 Starting Docker Build Cleanup..." -ForegroundColor Green
Write-Host "=====================================" -ForegroundColor Yellow

# Track cleanup statistics
$filesRemoved = 0
$spaceSaved = 0

# Function to calculate directory size
function Get-DirectorySize {
    param([string]$path)
    if (Test-Path $path) {
        return (Get-ChildItem $path -Recurse -File | Measure-Object -Property Length -Sum).Sum
    }
    return 0
}

# Function to format size
function Format-Size {
    param([long]$bytes)
    if ($bytes -gt 1GB) { return "$([math]::Round($bytes / 1GB, 2)) GB" }
    if ($bytes -gt 1MB) { return "$([math]::Round($bytes / 1MB, 2)) MB" }
    if ($bytes -gt 1KB) { return "$([math]::Round($bytes / 1KB, 2)) KB" }
    return "$bytes bytes"
}

# 1. Remove debug symbol files (.pdb files)
Write-Host "`n🔧 Removing debug symbol files..." -ForegroundColor Cyan
$pdbFiles = Get-ChildItem -Recurse -File -Filter "*.pdb"
foreach ($file in $pdbFiles) {
    $size = $file.Length
    Remove-Item $file.FullName -Force
    $filesRemoved++
    $spaceSaved += $size
    Write-Host "  Deleted: $($file.Name)" -ForegroundColor Gray
}

# 2. Remove backup files (ending with ~)
Write-Host "`n📁 Removing backup files..." -ForegroundColor Cyan
$backupFiles = Get-ChildItem -Recurse -File | Where-Object { $_.Name -match '~$' }
foreach ($file in $backupFiles) {
    $size = $file.Length
    Remove-Item $file.FullName -Force
    $filesRemoved++
    $spaceSaved += $size
    Write-Host "  Deleted: $($file.Name)" -ForegroundColor Gray
}

# 3. Remove Python cache directories
Write-Host "`n🐍 Removing Python cache directories..." -ForegroundColor Cyan
$pythonCacheDirs = Get-ChildItem -Recurse -Directory -Filter "__pycache__"
foreach ($dir in $pythonCacheDirs) {
    $size = Get-DirectorySize $dir.FullName
    Remove-Item $dir.FullName -Recurse -Force
    $filesRemoved++
    $spaceSaved += $size
    Write-Host "  Deleted: $($dir.Name) ($(Format-Size $size))" -ForegroundColor Gray
}

# 4. Remove test output logs (except important ones)
Write-Host "`n📝 Removing temporary test logs..." -ForegroundColor Cyan
$tempLogs = Get-ChildItem -File -Filter "test_output.log"
foreach ($file in $tempLogs) {
    $size = $file.Length
    Remove-Item $file.FullName -Force
    $filesRemoved++
    $spaceSaved += $size
    Write-Host "  Deleted: $($file.Name)" -ForegroundColor Gray
}

# 5. Remove temporary test builds
Write-Host "`n🔨 Removing temporary test builds..." -ForegroundColor Cyan
$tempBuilds = Get-ChildItem -File -Filter "sprintd-test"
foreach ($file in $tempBuilds) {
    $size = $file.Length
    Remove-Item $file.FullName -Force
    $filesRemoved++
    $spaceSaved += $size
    Write-Host "  Deleted: $($file.Name)" -ForegroundColor Gray
}

# 6. Clean Rust target directory (optional - ask user)
Write-Host "`n🦀 Rust build artifacts found:" -ForegroundColor Yellow
$rustTargetPath = "secure\rust\target"
if (Test-Path $rustTargetPath) {
    $rustSize = Get-DirectorySize $rustTargetPath
    Write-Host "  $rustTargetPath : $(Format-Size $rustSize)" -ForegroundColor Red
    Write-Host "  ⚠️  This will remove all Rust build artifacts!" -ForegroundColor Yellow
    $cleanRust = Read-Host "  Clean Rust target directory? (y/N)"
    if ($cleanRust -eq 'y' -or $cleanRust -eq 'Y') {
        Remove-Item $rustTargetPath -Recurse -Force
        $filesRemoved++
        $spaceSaved += $rustSize
        Write-Host "  ✅ Rust target directory cleaned" -ForegroundColor Green
    }
}

# 7. Clean Next.js cache (optional - ask user)
Write-Host "`n⚛️  Next.js cache found:" -ForegroundColor Yellow
$nextCachePath = "web\.next"
if (Test-Path $nextCachePath) {
    $nextSize = Get-DirectorySize $nextCachePath
    Write-Host "  $nextCachePath : $(Format-Size $nextSize)" -ForegroundColor Red
    Write-Host "  ⚠️  This will clear Next.js build cache!" -ForegroundColor Yellow
    $cleanNext = Read-Host "  Clean Next.js cache? (y/N)"
    if ($cleanNext -eq 'y' -or $cleanNext -eq 'Y') {
        Remove-Item $nextCachePath -Recurse -Force
        $filesRemoved++
        $spaceSaved += $nextSize
        Write-Host "  ✅ Next.js cache cleaned" -ForegroundColor Green
    }
}

# 8. Clean build directory (optional - ask user)
Write-Host "`n🏗️  Build directory found:" -ForegroundColor Yellow
$buildPath = "build"
if (Test-Path $buildPath) {
    $buildSize = Get-DirectorySize $buildPath
    $buildFiles = Get-ChildItem $buildPath -File
    Write-Host "  $buildPath : $(Format-Size $buildSize) ($($buildFiles.Count) files)" -ForegroundColor Red
    Write-Host "  Files:" -ForegroundColor Gray
    foreach ($file in $buildFiles) {
        Write-Host "    - $($file.Name)" -ForegroundColor Gray
    }
    Write-Host "  ⚠️  This will remove all build artifacts!" -ForegroundColor Yellow
    $cleanBuild = Read-Host "  Clean build directory? (y/N)"
    if ($cleanBuild -eq 'y' -or $cleanBuild -eq 'Y') {
        Remove-Item "$buildPath\*" -Force
        $filesRemoved++
        $spaceSaved += $buildSize
        Write-Host "  ✅ Build directory cleaned" -ForegroundColor Green
    }
}

# Summary
Write-Host "`n🎉 Cleanup Complete!" -ForegroundColor Green
Write-Host "=====================================" -ForegroundColor Yellow
Write-Host "Files/Directories removed: $filesRemoved" -ForegroundColor Cyan
Write-Host "Space saved: $(Format-Size $spaceSaved)" -ForegroundColor Cyan
Write-Host "`n💡 Tips:" -ForegroundColor Blue
Write-Host "  - Run 'go clean' to remove Go build cache" -ForegroundColor Gray
Write-Host "  - Run 'npm cache clean --force' in web/ directory for Node.js cache" -ForegroundColor Gray
Write-Host "  - Consider adding these patterns to .gitignore:" -ForegroundColor Gray
Write-Host "    *.pdb" -ForegroundColor Gray
Write-Host "    __pycache__/" -ForegroundColor Gray
Write-Host "    *.log" -ForegroundColor Gray
Write-Host "    *~" -ForegroundColor Gray
