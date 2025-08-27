# file-check.ps1
# A utility to check for potential file corruption

param(
    [string]$Path = "internal/p2p",
    [switch]$Fix
)

Write-Host "Bitcoin Sprint File Integrity Check" -ForegroundColor Cyan
Write-Host "====================================" -ForegroundColor Cyan

$files = Get-ChildItem -Path $Path -Filter "*.go" -Recurse -ErrorAction SilentlyContinue

Write-Host "Checking ${files.Count} files in $Path" -ForegroundColor Yellow

$corruptedFiles = @()
$emptyFiles = @()

foreach ($file in $files) {
    $content = Get-Content -Path $file.FullName -Raw -ErrorAction SilentlyContinue
    $size = (Get-Item -Path $file.FullName).Length
    
    Write-Host "Checking $($file.Name) - Size: $size bytes" -ForegroundColor Gray
    
    if ($size -eq 0) {
        Write-Host "  WARNING: Empty file detected: $($file.FullName)" -ForegroundColor Red
        $emptyFiles += $file
        continue
    }
    
    # Check if file content matches what's on disk
    $tempFile = [System.IO.Path]::GetTempFileName()
    $content | Set-Content -Path $tempFile -NoNewline
    $tempSize = (Get-Item -Path $tempFile).Length
    
    if ($tempSize -ne $size) {
        Write-Host "  WARNING: Possible corruption in $($file.FullName)" -ForegroundColor Red
        Write-Host "  Content size ($tempSize) != File size ($size)" -ForegroundColor Red
        $corruptedFiles += $file
    }
    
    Remove-Item -Path $tempFile -Force
}

if ($corruptedFiles.Count -gt 0 -or $emptyFiles.Count -gt 0) {
    Write-Host "`nIssues detected:" -ForegroundColor Red
    Write-Host "  Empty files: $($emptyFiles.Count)" -ForegroundColor Yellow
    Write-Host "  Corrupted files: $($corruptedFiles.Count)" -ForegroundColor Yellow
    
    if ($Fix) {
        Write-Host "`nAttempting to fix issues..." -ForegroundColor Yellow
        
        foreach ($file in $emptyFiles) {
            Write-Host "  Backing up empty file: $($file.Name)" -ForegroundColor Cyan
            Copy-Item -Path $file.FullName -Destination "$($file.FullName).empty.bak" -ErrorAction SilentlyContinue
        }
        
        foreach ($file in $corruptedFiles) {
            Write-Host "  Backing up corrupted file: $($file.Name)" -ForegroundColor Cyan
            Copy-Item -Path $file.FullName -Destination "$($file.FullName).corrupt.bak" -ErrorAction SilentlyContinue
            
            # Get VS Code buffer content if possible
            # (Just a placeholder - VS Code API would need to be used here)
            Write-Host "  Attempting to recover from editor buffer for $($file.Name)" -ForegroundColor Yellow
        }
    }
} else {
    Write-Host "`nAll files passed integrity check." -ForegroundColor Green
}

Write-Host "`nAdvice to prevent corruption:" -ForegroundColor Cyan
Write-Host "1. Save files before running build commands" -ForegroundColor White
Write-Host "2. Avoid simultaneous writes from multiple programs" -ForegroundColor White
Write-Host "3. Check for disk errors or antivirus interference" -ForegroundColor White
Write-Host "4. Make sure adequate disk space is available" -ForegroundColor White
Write-Host "5. Consider creating backups before each build" -ForegroundColor White
