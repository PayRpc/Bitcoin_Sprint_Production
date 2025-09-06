# Bitcoin Sprint - Structure Maintenance Script
# Automatically checks and maintains clean enterprise folder structure

param(
    [switch]$Check,      # Only check, don't fix
    [switch]$Fix,        # Check and auto-fix violations
    [switch]$Report      # Generate detailed report
)

Write-Host "🎯 Bitcoin Sprint - Structure Maintenance" -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan

$violations = @()
$rootPath = Get-Location

# Check for loose PowerShell scripts
Write-Host "`n📁 Checking PowerShell Scripts..." -ForegroundColor Yellow
$loosePS1 = Get-ChildItem -Filter "*.ps1" -ErrorAction SilentlyContinue
if ($loosePS1) {
    $violations += "Loose PowerShell scripts found: $($loosePS1.Count) files"
    Write-Host "❌ Found $($loosePS1.Count) loose .ps1 files in root" -ForegroundColor Red
    
    if ($Fix) {
        Write-Host "🔧 Moving to scripts/powershell/..." -ForegroundColor Green
        New-Item -ItemType Directory -Path "scripts\powershell" -Force | Out-Null
        Move-Item -Path "*.ps1" -Destination "scripts\powershell\" -Force
        Write-Host "✅ Fixed: PowerShell scripts organized" -ForegroundColor Green
    }
} else {
    Write-Host "✅ No loose PowerShell scripts" -ForegroundColor Green
}

# Check for loose JavaScript files
Write-Host "`n🟨 Checking JavaScript Files..." -ForegroundColor Yellow
$looseJS = Get-ChildItem -Filter "*.js" -ErrorAction SilentlyContinue | Where-Object {$_.Name -ne "next.config.js"}
$webLooseJS = Get-ChildItem -Path "web\" -Filter "*.js" -ErrorAction SilentlyContinue | Where-Object {$_.Name -ne "next.config.js"}

if ($looseJS -or $webLooseJS) {
    $totalJS = ($looseJS + $webLooseJS | Sort-Object Name -Unique).Count
    $violations += "Loose JavaScript files found: $totalJS files"
    Write-Host "❌ Found $totalJS loose JavaScript files" -ForegroundColor Red
    
    if ($Fix) {
        Write-Host "� Organizing JavaScript files..." -ForegroundColor Green
        New-Item -ItemType Directory -Path "web\scripts\test" -Force | Out-Null
        New-Item -ItemType Directory -Path "web\scripts\utilities" -Force | Out-Null
        New-Item -ItemType Directory -Path "web\lib\bridges" -Force | Out-Null
        
        # Move test files
        if (Test-Path "web\test-*.js") { Move-Item -Path "web\test-*.js" -Destination "web\scripts\test\" -Force -ErrorAction SilentlyContinue }
        if (Test-Path "test-*.js") { Move-Item -Path "test-*.js" -Destination "web\scripts\test\" -Force -ErrorAction SilentlyContinue }
        
        # Move utility files
        $utilityFiles = @("demo-*.js", "monitor-*.js", "simple-*.js", "smart-*.js", "setup-*.js")
        foreach ($pattern in $utilityFiles) {
            if (Test-Path "web\$pattern") { Move-Item -Path "web\$pattern" -Destination "web\scripts\utilities\" -Force -ErrorAction SilentlyContinue }
            if (Test-Path $pattern) { Move-Item -Path $pattern -Destination "web\scripts\utilities\" -Force -ErrorAction SilentlyContinue }
        }
        
        # Move bridge files
        $bridgeFiles = @("*bridge*.js", "merkle-*.js", "examples.js", "utils.js")
        foreach ($pattern in $bridgeFiles) {
            if (Test-Path "web\$pattern") { Move-Item -Path "web\$pattern" -Destination "web\lib\bridges\" -Force -ErrorAction SilentlyContinue }
            if (Test-Path $pattern) { Move-Item -Path $pattern -Destination "web\lib\bridges\" -Force -ErrorAction SilentlyContinue }
        }
        
        Write-Host "✅ Fixed: JavaScript files organized" -ForegroundColor Green
    }
} else {
    Write-Host "✅ No loose JavaScript files" -ForegroundColor Green
}

# Check for loose Go files
Write-Host "`n🐹 Checking Go Files..." -ForegroundColor Yellow
$looseGo = Get-ChildItem -Filter "*.go" -ErrorAction SilentlyContinue | Where-Object {$_.Directory.Name -eq "BItcoin_Sprint"}

if ($looseGo) {
    $totalGo = $looseGo.Count
    $violations += "Loose Go files found: $totalGo files"
    Write-Host "❌ Found $totalGo loose Go files in root" -ForegroundColor Red
    
    if ($Fix) {
        Write-Host "🔧 Organizing Go files..." -ForegroundColor Green
        New-Item -ItemType Directory -Path "tests\benchmarks" -Force | Out-Null
        New-Item -ItemType Directory -Path "tests\integration" -Force | Out-Null
        New-Item -ItemType Directory -Path "tests\standalone" -Force | Out-Null
        New-Item -ItemType Directory -Path "internal\runtime\verification" -Force | Out-Null
        
        # Move benchmark files
        if (Test-Path "benchmark*.go") { Move-Item -Path "benchmark*.go" -Destination "tests\benchmarks\" -Force -ErrorAction SilentlyContinue }
        
        # Move test files
        if (Test-Path "test_*.go") { Move-Item -Path "test_*.go" -Destination "tests\integration\" -Force -ErrorAction SilentlyContinue }
        if (Test-Path "standalone_test.go") { Move-Item -Path "standalone_test.go" -Destination "tests\standalone\" -Force -ErrorAction SilentlyContinue }
        
        # Move runtime verification files
        if (Test-Path "runtime*.go") { Move-Item -Path "runtime*.go" -Destination "internal\runtime\verification\" -Force -ErrorAction SilentlyContinue }
        
        Write-Host "✅ Fixed: Go files organized" -ForegroundColor Green
    }
} else {
    Write-Host "✅ No loose Go files" -ForegroundColor Green
}

$looseYML = Get-ChildItem -Filter "*.yml" -ErrorAction SilentlyContinue
$looseDockerCompose = Get-ChildItem -Filter "docker-compose*.yml" -ErrorAction SilentlyContinue

if ($looseYML -or $looseDockerCompose) {
    $totalYML = ($looseYML + $looseDockerCompose | Sort-Object Name -Unique).Count
    $violations += "Loose YAML/Docker files found: $totalYML files"
    Write-Host "❌ Found $totalYML loose YAML/Docker files in root" -ForegroundColor Red
    
    if ($Fix) {
        Write-Host "🔧 Moving to docker/..." -ForegroundColor Green
        New-Item -ItemType Directory -Path "docker" -Force | Out-Null
        if ($looseYML) { Move-Item -Path "*.yml" -Destination "docker\" -Force -ErrorAction SilentlyContinue }
        if ($looseDockerCompose) { Move-Item -Path "docker-compose*.yml" -Destination "docker\" -Force -ErrorAction SilentlyContinue }
        Write-Host "✅ Fixed: Docker/YAML files organized" -ForegroundColor Green
    }
} else {
    Write-Host "✅ No loose YAML/Docker files" -ForegroundColor Green
}

# Check for empty files (corruption indicator)
Write-Host "`n📄 Checking for Empty Files..." -ForegroundColor Yellow
$emptyFiles = Get-ChildItem -Recurse | Where-Object {$_.Length -eq 0 -and !$_.PSIsContainer}
if ($emptyFiles) {
    $violations += "Empty files found: $($emptyFiles.Count) files (potential corruption)"
    Write-Host "❌ Found $($emptyFiles.Count) empty files (corruption indicator)" -ForegroundColor Red
    
    if ($Fix) {
        Write-Host "🔧 Removing empty files..." -ForegroundColor Green
        $emptyFiles | Remove-Item -Force
        Write-Host "✅ Fixed: Empty files removed" -ForegroundColor Green
    }
} else {
    Write-Host "✅ No empty files found" -ForegroundColor Green
}

# Check for legacy simple_api
Write-Host "`n🗂️ Checking for Legacy Components..." -ForegroundColor Yellow
$legacyDirs = @("simple_api", "simple-api", "api-simple")
$foundLegacy = @()

foreach ($dir in $legacyDirs) {
    if (Test-Path $dir) {
        $foundLegacy += $dir
    }
}

if ($foundLegacy) {
    $violations += "Legacy directories found: $($foundLegacy -join ', ')"
    Write-Host "❌ Found legacy directories: $($foundLegacy -join ', ')" -ForegroundColor Red
    
    if ($Fix) {
        Write-Host "🔧 Removing legacy directories..." -ForegroundColor Green
        foreach ($dir in $foundLegacy) {
            Remove-Item -Path $dir -Recurse -Force
        }
        Write-Host "✅ Fixed: Legacy directories removed" -ForegroundColor Green
    }
} else {
    Write-Host "✅ No legacy directories found" -ForegroundColor Green
}

# Check essential enterprise components
Write-Host "`n🏢 Verifying Enterprise Components..." -ForegroundColor Yellow
$essentialPaths = @(
    "internal\api\auth.go",           # CustomerKeyManager
    "internal\securebuf",             # SecureBuf system
    "secure\rust\Cargo.toml",        # Rust FFI
    "web\pages\api\v1",               # Web API
    "config\docker-compose.yml"       # Main config
)

$missingComponents = @()
foreach ($path in $essentialPaths) {
    if (!(Test-Path $path)) {
        $missingComponents += $path
    }
}

if ($missingComponents) {
    Write-Host "⚠️ Missing enterprise components: $($missingComponents -join ', ')" -ForegroundColor Red
    $violations += "Missing enterprise components: $($missingComponents.Count) items"
} else {
    Write-Host "✅ All enterprise components present" -ForegroundColor Green
}

# Summary Report
Write-Host "`n📊 STRUCTURE ANALYSIS SUMMARY" -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan

if ($violations.Count -eq 0) {
    Write-Host "🎉 EXCELLENT: Clean enterprise structure maintained!" -ForegroundColor Green
    Write-Host "   All components properly organized" -ForegroundColor Green
    Write-Host "   No violations detected" -ForegroundColor Green
} else {
    Write-Host "⚠️ VIOLATIONS DETECTED: $($violations.Count) issues" -ForegroundColor Red
    foreach ($violation in $violations) {
        Write-Host "   • $violation" -ForegroundColor Red
    }
    
    if (!$Fix) {
        Write-Host "`n💡 Run with -Fix parameter to auto-resolve issues" -ForegroundColor Yellow
    }
}

Write-Host "`n🎯 Enterprise Quality Standards:" -ForegroundColor Cyan
Write-Host "   • CustomerKeyManager: $(if (Test-Path 'internal\api\auth.go') {'✅ Active'} else {'❌ Missing'})"
Write-Host "   • SecureBuf System: $(if (Test-Path 'internal\securebuf') {'✅ Active'} else {'❌ Missing'})"  
Write-Host "   • Rust FFI Library: $(if (Test-Path 'secure\rust\Cargo.toml') {'✅ Active'} else {'❌ Missing'})"
Write-Host "   • Web API Enterprise: $(if (Test-Path 'web\pages\api\v1') {'✅ Active'} else {'❌ Missing'})"

if ($Report) {
    $reportPath = "structure-report-$(Get-Date -Format 'yyyyMMdd-HHmmss').txt"
    @"
Bitcoin Sprint - Structure Analysis Report
Generated: $(Get-Date)
==========================================

Violations: $($violations.Count)
$(if ($violations) { $violations | ForEach-Object { "• $_" } | Out-String } else { "No violations found" })

Enterprise Components Status:
• CustomerKeyManager: $(if (Test-Path 'internal\api\auth.go') {'Active'} else {'Missing'})
• SecureBuf System: $(if (Test-Path 'internal\securebuf') {'Active'} else {'Missing'})
• Rust FFI Library: $(if (Test-Path 'secure\rust\Cargo.toml') {'Active'} else {'Missing'})
• Web API Enterprise: $(if (Test-Path 'web\pages\api\v1') {'Active'} else {'Missing'})

Recommendation: $(if ($violations.Count -eq 0) {'Structure is properly maintained'} else {'Run maintenance script with -Fix parameter'})
"@ | Out-File $reportPath
    Write-Host "`n📄 Report saved to: $reportPath" -ForegroundColor Green
}

Write-Host "`nStructure check complete! 🎯" -ForegroundColor Cyan
