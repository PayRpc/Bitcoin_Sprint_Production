# Bitcoin Sprint Package Validation Test
# Validates the production package is ready for deployment

param(
    [string]$PackageDir = "bitcoin-sprint-package",
    [switch]$FullTest = $false
)

Write-Host "🔍 Bitcoin Sprint Package Validation" -ForegroundColor Green
Write-Host "=" * 50

$ErrorCount = 0

# Check package structure
Write-Host "📁 Validating package structure..." -ForegroundColor Cyan

$requiredDirs = @("bin", "config", "docs", "licenses", "scripts")
foreach ($dir in $requiredDirs) {
    if (Test-Path "$PackageDir\$dir") {
        Write-Host "  ✅ $dir/" -ForegroundColor Green
    } else {
        Write-Host "  ❌ $dir/ missing" -ForegroundColor Red
        $ErrorCount++
    }
}

# Check critical files
Write-Host "📄 Validating critical files..." -ForegroundColor Cyan

$criticalFiles = @(
    "bin\bitcoin-sprint-production.exe",
    "config\config-production-optimized.json",
    "licenses\license.json",
    "DEPLOYMENT_GUIDE.md",
    "VERSION.json",
    "install.ps1"
)

foreach ($file in $criticalFiles) {
    if (Test-Path "$PackageDir\$file") {
        $size = (Get-Item "$PackageDir\$file").Length
        Write-Host "  ✅ $file ($([math]::Round($size/1KB, 1))KB)" -ForegroundColor Green
    } else {
        Write-Host "  ❌ $file missing" -ForegroundColor Red
        $ErrorCount++
    }
}

# Test binary execution
Write-Host "🔧 Testing binary execution..." -ForegroundColor Cyan

if (Test-Path "$PackageDir\bin\bitcoin-sprint-production.exe") {
    try {
        # Quick version check
        $versionOutput = & "$PackageDir\bin\bitcoin-sprint-production.exe" --version 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Host "  ✅ Binary executes successfully" -ForegroundColor Green
            Write-Host "  ℹ️ Version: $versionOutput" -ForegroundColor White
        } else {
            Write-Host "  ⚠️ Binary version check failed (exit code: $LASTEXITCODE)" -ForegroundColor Yellow
        }
    } catch {
        Write-Host "  ❌ Binary execution failed: $($_.Exception.Message)" -ForegroundColor Red
        $ErrorCount++
    }
} else {
    Write-Host "  ❌ Production binary not found" -ForegroundColor Red
    $ErrorCount++
}

# Validate configuration files
Write-Host "⚙️ Validating configuration files..." -ForegroundColor Cyan

$configFiles = Get-ChildItem "$PackageDir\config" -Filter "*.json" -ErrorAction SilentlyContinue
foreach ($config in $configFiles) {
    try {
        $configData = Get-Content $config.FullName | ConvertFrom-Json
        Write-Host "  ✅ $($config.Name) - Valid JSON" -ForegroundColor Green
        
        # Check for performance settings
        if ($configData.PSObject.Properties.Name -contains "performance") {
            Write-Host "    ℹ️ Performance settings present" -ForegroundColor White
        }
    } catch {
        Write-Host "  ❌ $($config.Name) - Invalid JSON: $($_.Exception.Message)" -ForegroundColor Red
        $ErrorCount++
    }
}

# Check license files
Write-Host "📜 Validating license files..." -ForegroundColor Cyan

$licenseFiles = Get-ChildItem "$PackageDir\licenses" -Filter "*.json" -ErrorAction SilentlyContinue
foreach ($license in $licenseFiles) {
    try {
        $licenseData = Get-Content $license.FullName | ConvertFrom-Json
        Write-Host "  ✅ $($license.Name) - Valid license" -ForegroundColor Green
        
        if ($licenseData.tier) {
            Write-Host "    ℹ️ Tier: $($licenseData.tier)" -ForegroundColor White
        }
    } catch {
        Write-Host "  ❌ $($license.Name) - Invalid license: $($_.Exception.Message)" -ForegroundColor Red
        $ErrorCount++
    }
}

if ($FullTest) {
    Write-Host "🚀 Running full deployment test..." -ForegroundColor Cyan
    
    # Copy to temp directory and test installation
    $tempDir = "temp-deployment-test"
    if (Test-Path $tempDir) {
        Remove-Item -Recurse -Force $tempDir
    }
    
    Copy-Item -Recurse $PackageDir $tempDir
    
    try {
        # Test installer
        Push-Location $tempDir
        $testInstallPath = "test-install"
        & ".\install.ps1" -InstallPath $testInstallPath -Verbose
        
        if (Test-Path "$testInstallPath\bitcoin-sprint-production.exe") {
            Write-Host "  ✅ Installation test passed" -ForegroundColor Green
        } else {
            Write-Host "  ❌ Installation test failed" -ForegroundColor Red
            $ErrorCount++
        }
        
        Pop-Location
        Remove-Item -Recurse -Force $tempDir -ErrorAction SilentlyContinue
    } catch {
        Write-Host "  ❌ Full deployment test failed: $($_.Exception.Message)" -ForegroundColor Red
        $ErrorCount++
        Pop-Location
    }
}

# Final validation report
Write-Host "`n📋 Validation Summary" -ForegroundColor Green
Write-Host "=" * 50

if ($ErrorCount -eq 0) {
    Write-Host "✅ Package validation PASSED" -ForegroundColor Green
    Write-Host "🚀 Ready for production deployment!" -ForegroundColor Green
    
    # Show package size
    if (Get-ChildItem $PackageDir -Recurse -ErrorAction SilentlyContinue) {
        $totalSize = (Get-ChildItem $PackageDir -Recurse | Measure-Object -Property Length -Sum).Sum
        Write-Host "📦 Total package size: $([math]::Round($totalSize/1MB, 1))MB" -ForegroundColor White
    }
    
    # Show archive if exists
    $archiveFile = Get-ChildItem "bitcoin-sprint-*.zip" | Sort-Object LastWriteTime -Descending | Select-Object -First 1
    if ($archiveFile) {
        $archiveSize = [math]::Round($archiveFile.Length/1MB, 1)
        Write-Host "📁 Archive file: $($archiveFile.Name) ($($archiveSize)MB)" -ForegroundColor White
    }
    
    exit 0
} else {
    Write-Host "❌ Package validation FAILED" -ForegroundColor Red
    Write-Host "🔧 $ErrorCount errors found - fix before deployment" -ForegroundColor Red
    exit 1
}
