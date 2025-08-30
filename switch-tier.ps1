# Bitcoin Sprint Tier Switcher
# Usage: .\switch-tier.ps1 [free|pro|business|turbo|enterprise]

param(
    [Parameter(Mandatory=$false)]
    [ValidateSet("free", "pro", "business", "turbo", "enterprise")]
    [string]$Tier
)

if (-not $Tier) {
    Write-Host "Usage: .\switch-tier.ps1 [free|pro|business|turbo|enterprise]" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Available tiers:" -ForegroundColor Cyan
    Write-Host "  free       - Basic tier with limited resources" -ForegroundColor White
    Write-Host "  pro        - Professional tier with moderate resources" -ForegroundColor White
    Write-Host "  business   - Business tier with higher performance" -ForegroundColor White
    Write-Host "  turbo      - High-performance tier with ultra-low latency" -ForegroundColor Green
    Write-Host "  enterprise - Enterprise tier with maximum performance" -ForegroundColor Magenta
    exit 1
}

$envFile = ".env.$Tier"

if (Test-Path $envFile) {
    Write-Host "🔄 Switching to $Tier tier..." -ForegroundColor Yellow

    try {
        Copy-Item $envFile ".env" -Force
        Write-Host "✅ Successfully switched to $Tier tier" -ForegroundColor Green
        Write-Host ""
        Write-Host "Current configuration:" -ForegroundColor Cyan
        Write-Host "TIER=$Tier" -ForegroundColor White
        Write-Host ""
        Write-Host "To apply changes, restart the application:" -ForegroundColor Yellow
        Write-Host "  .\start-dev.bat" -ForegroundColor White
        Write-Host "  # or" -ForegroundColor Gray
        Write-Host "  .\start-dev.ps1" -ForegroundColor White
    }
    catch {
        Write-Host "❌ Error switching tier: $($_.Exception.Message)" -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "❌ Error: Tier '$Tier' configuration not found" -ForegroundColor Red
    Write-Host "Available tiers: free, pro, business, turbo, enterprise" -ForegroundColor Yellow
    exit 1
}
