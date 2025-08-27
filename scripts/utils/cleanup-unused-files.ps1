#!/usr/bin/env pwsh
# Bitcoin Sprint - Cleanup Unused Files
# Removes development artifacts, old test reports, and duplicate binaries

param(
    [switch]$DryRun,  # Show what would be deleted without actually deleting
    [switch]$Force    # Skip confirmation prompts
)

$ErrorActionPreference = "Stop"

function Write-CleanupStatus($message, $color = "White") {
    Write-Host "[CLEANUP] $message" -ForegroundColor $color
}

function Remove-FilesSafely($files, $description) {
    if ($files.Count -eq 0) {
        Write-CleanupStatus "No $description found" -color "Gray"
        return
    }
    
    Write-CleanupStatus "Found $($files.Count) $description" -color "Yellow"
    
    foreach ($file in $files) {
        $size = if ($file.Length) { " ($([math]::Round($file.Length / 1MB, 2)) MB)" } else { "" }
        
        if ($DryRun) {
            Write-CleanupStatus "Would remove: $($file.Name)$size" -color "Cyan"
        } else {
            try {
                Remove-Item $file.FullName -Force
                Write-CleanupStatus "Removed: $($file.Name)$size" -color "Green"
            } catch {
                Write-CleanupStatus "Failed to remove: $($file.Name) - $($_.Exception.Message)" -color "Red"
            }
        }
    }
}

Write-CleanupStatus "=== Bitcoin Sprint Cleanup Utility ===" -color "Cyan"
Write-CleanupStatus "Working directory: $(Get-Location)" -color "Gray"

if ($DryRun) {
    Write-CleanupStatus "DRY RUN MODE - No files will be deleted" -color "Yellow"
}

# 1. Remove old test executable files (keep only the latest working ones)
Write-CleanupStatus "`n1. Cleaning up old executable files..." -color "Cyan"
$keepExes = @(
    "bitcoin-sprint.exe",           # Main production binary
    "bitcoin-sprint-test.exe"       # Main test binary
)

$oldExes = Get-ChildItem *.exe | Where-Object { 
    $_.Name -notin $keepExes -and (
        $_.Name -like "*test-config*" -or
        $_.Name -like "*integration*" -or
        $_.Name -like "*api*" -or
        $_.Name -like "*production*" -or
        $_.Name -like "*zmq*" -or
        $_.Name -like "*sla*" -or
        $_.Name -like "*tiers*" -or
        $_.Name -like "*turbo*"
    )
}
Remove-FilesSafely $oldExes "old executable files"

# 2. Remove test report JSON files (keep latest one of each type)
Write-CleanupStatus "`n2. Cleaning up test report files..." -color "Cyan"
$testReports = Get-ChildItem *.json | Where-Object { 
    $_.Name -like "*report*" -or 
    $_.Name -like "*sla_test*" -or
    $_.Name -like "*demo_report*"
}

# Keep only the latest SLA test report
$slaReports = $testReports | Where-Object { $_.Name -like "*sla_test*" } | Sort-Object LastWriteTime
$oldSlaReports = $slaReports | Select-Object -SkipLast 1

# Keep only the latest demo report
$demoReports = $testReports | Where-Object { $_.Name -like "*demo_report*" } | Sort-Object LastWriteTime  
$oldDemoReports = $demoReports | Select-Object -SkipLast 1

$allOldReports = @($oldSlaReports) + @($oldDemoReports)
Remove-FilesSafely $allOldReports "old test report files"

# 3. Remove batch files (replaced by PowerShell scripts)
Write-CleanupStatus "`n3. Cleaning up legacy batch files..." -color "Cyan"
$batchFiles = Get-ChildItem *.bat
Remove-FilesSafely $batchFiles "legacy batch files"

# 4. Remove CMake build artifacts
Write-CleanupStatus "`n4. Cleaning up CMake artifacts..." -color "Cyan"
$cmakeFiles = @()
if (Test-Path "_cmake_build") {
    $cmakeFiles += Get-Item "_cmake_build"
}
if (Test-Path "CMakeLists.txt") {
    $cmakeFiles += Get-Item "CMakeLists.txt"
}
Remove-FilesSafely $cmakeFiles "CMake artifacts"

# 5. Remove duplicate config files (regtest/testnet are not used in production)
Write-CleanupStatus "`n5. Cleaning up unused config files..." -color "Cyan"
$unusedConfigs = Get-ChildItem *config*.json | Where-Object {
    $_.Name -like "*regtest*" -or 
    $_.Name -like "*testnet*" -or
    $_.Name -like "*minimal*"
}
Remove-FilesSafely $unusedConfigs "unused config files"

# 6. Remove old Bitcoin Core config files
Write-CleanupStatus "`n6. Cleaning up old Bitcoin Core configs..." -color "Cyan"
$oldBitcoinConfigs = Get-ChildItem bitcoin*.conf | Where-Object {
    $_.Name -like "*regtest*" -or 
    $_.Name -like "*testnet*" -or
    $_.Name -eq "bitcoin-test.conf"
}
Remove-FilesSafely $oldBitcoinConfigs "old Bitcoin Core config files"

# 7. Remove logs directory if it exists and is empty or contains only old logs
Write-CleanupStatus "`n7. Cleaning up log files..." -color "Cyan"
if (Test-Path "logs") {
    $logFiles = Get-ChildItem "logs" -Recurse -File | Where-Object {
        $_.LastWriteTime -lt (Get-Date).AddDays(-7)  # Older than 7 days
    }
    Remove-FilesSafely $logFiles "old log files"
    
    # Remove logs directory if empty
    if ((Get-ChildItem "logs" -Force | Measure-Object).Count -eq 0) {
        if ($DryRun) {
            Write-CleanupStatus "Would remove empty logs directory" -color "Cyan"
        } else {
            Remove-Item "logs" -Force
            Write-CleanupStatus "Removed empty logs directory" -color "Green"
        }
    }
}

# 8. Calculate space saved
Write-CleanupStatus "`n8. Calculating space saved..." -color "Cyan"
$totalSize = 0
$allCleanedFiles = @($oldExes) + @($allOldReports) + @($batchFiles) + @($cmakeFiles) + @($unusedConfigs) + @($oldBitcoinConfigs)
foreach ($file in $allCleanedFiles) {
    if ($file.Length) {
        $totalSize += $file.Length
    }
}

$spaceSaved = [math]::Round($totalSize / 1MB, 2)
Write-CleanupStatus "Total space to be freed: $spaceSaved MB" -color "Green"

# Summary
Write-CleanupStatus "`n=== Cleanup Summary ===" -color "Cyan"
Write-CleanupStatus "Files to remove:" -color "White"
Write-CleanupStatus "  - Old executables: $($oldExes.Count)" -color "Gray"
Write-CleanupStatus "  - Old test reports: $($allOldReports.Count)" -color "Gray"
Write-CleanupStatus "  - Batch files: $($batchFiles.Count)" -color "Gray"
Write-CleanupStatus "  - CMake artifacts: $($cmakeFiles.Count)" -color "Gray"
Write-CleanupStatus "  - Unused configs: $($unusedConfigs.Count + $oldBitcoinConfigs.Count)" -color "Gray"
Write-CleanupStatus "Total space saved: $spaceSaved MB" -color "Green"

if ($DryRun) {
    Write-CleanupStatus "`nRun without -DryRun to actually delete these files" -color "Yellow"
} else {
    Write-CleanupStatus "`nCleanup completed!" -color "Green"
}
