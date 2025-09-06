# Bitcoin Sprint - Pre-Start Environment Validation
# Automatically runs before any service starts to ensure environment integrity

Write-Host "🚀 Bitcoin Sprint Pre-Start Validation" -ForegroundColor Cyan
Write-Host "======================================="

# Check environment protection
Write-Host "🛡️ Validating environment configuration..."
$envCheck = & ".\scripts\powershell\protect-environment.ps1" -Check
if ($LASTEXITCODE -ne 0) {
    Write-Host "🚨 CRITICAL: Environment corruption detected!" -ForegroundColor Red
    Write-Host "🔧 Auto-restoring enterprise configuration..." -ForegroundColor Yellow
    
    & ".\scripts\powershell\protect-environment.ps1" -Restore
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ Enterprise environment restored successfully!" -ForegroundColor Green
    } else {
        Write-Host "❌ FAILED: Could not restore environment!" -ForegroundColor Red
        Write-Host "Manual intervention required." -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "✅ Environment integrity verified" -ForegroundColor Green
}

# Verify critical enterprise settings
Write-Host "`n🔍 Verifying enterprise configuration..."
$content = Get-Content ".env" -Raw

$criticalSettings = @(
    @{ Setting = "TIER=enterprise"; Message = "Enterprise tier active" },
    @{ Setting = "RATE_LIMIT_REQUESTS_PER_SECOND=500"; Message = "Enterprise rate limits" },
    @{ Setting = "ENABLE_KERNEL_BYPASS=true"; Message = "Hardware optimizations enabled" },
    @{ Setting = "DATABASE_URL=./enterprise_tier.db"; Message = "Enterprise database configured" }
)

$allValid = $true
foreach ($setting in $criticalSettings) {
    if ($content -match [regex]::Escape($setting.Setting)) {
        Write-Host "  ✅ $($setting.Message)" -ForegroundColor Green
    } else {
        Write-Host "  ❌ $($setting.Message)" -ForegroundColor Red
        $allValid = $false
    }
}

if ($allValid) {
    Write-Host "`n🎯 All systems ready for enterprise operation!" -ForegroundColor Green
    exit 0
} else {
    Write-Host "`n🚨 Configuration validation failed!" -ForegroundColor Red
    exit 1
}
