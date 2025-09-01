# GitIgnore Maintenance Script for Bitcoin Sprint
# This script helps maintain and optimize the .gitignore file

Write-Host "🔧 GitIgnore Maintenance Script" -ForegroundColor Green
Write-Host "=================================" -ForegroundColor Yellow

# Check for common patterns that might be missing
$missingPatterns = @()

# Check if .gitignore exists
if (-not (Test-Path ".gitignore")) {
    Write-Host "❌ No .gitignore file found!" -ForegroundColor Red
    exit 1
}

$content = Get-Content ".gitignore" -Raw

# Check for common missing patterns
$commonPatterns = @(
    "*.pdb",
    "__pycache__/",
    "*.log",
    "*~",
    "target/",
    ".next/",
    "build/",
    "node_modules/",
    "*.tmp",
    "*.temp",
    "*.bak",
    "*.swp",
    "*.swo"
)

Write-Host "`n📋 Checking for common patterns..." -ForegroundColor Cyan
foreach ($pattern in $commonPatterns) {
    if ($content -notmatch [regex]::Escape($pattern)) {
        $missingPatterns += $pattern
        Write-Host "  ⚠️  Missing: $pattern" -ForegroundColor Yellow
    } else {
        Write-Host "  ✅ Found: $pattern" -ForegroundColor Green
    }
}

if ($missingPatterns.Count -eq 0) {
    Write-Host "`n🎉 All common patterns are present!" -ForegroundColor Green
} else {
    Write-Host "`n💡 Consider adding these missing patterns:" -ForegroundColor Blue
    foreach ($pattern in $missingPatterns) {
        Write-Host "  $pattern" -ForegroundColor Gray
    }
}

# Check file size
$fileSize = (Get-Item ".gitignore").Length
Write-Host "`n📊 .gitignore Statistics:" -ForegroundColor Cyan
Write-Host "  File size: $([math]::Round($fileSize / 1KB, 2)) KB" -ForegroundColor White
Write-Host "  Lines: $((Get-Content ".gitignore").Count)" -ForegroundColor White

# Check for duplicates
$lines = Get-Content ".gitignore"
$duplicates = $lines | Group-Object | Where-Object { $_.Count -gt 1 } | Select-Object -ExpandProperty Name
if ($duplicates) {
    Write-Host "`n⚠️  Duplicate entries found:" -ForegroundColor Yellow
    foreach ($dup in $duplicates) {
        Write-Host "  $dup" -ForegroundColor Gray
    }
} else {
    Write-Host "`n✅ No duplicate entries found" -ForegroundColor Green
}

Write-Host "`n💡 Tips for .gitignore maintenance:" -ForegroundColor Blue
Write-Host "  - Review and remove unused patterns periodically" -ForegroundColor Gray
Write-Host "  - Use blanket patterns (**) for recursive ignores" -ForegroundColor Gray
Write-Host "  - Test with 'git status --ignored' to verify" -ForegroundColor Gray
Write-Host "  - Consider using .gitignore templates for your tech stack" -ForegroundColor Gray
