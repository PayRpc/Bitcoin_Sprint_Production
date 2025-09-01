# Docker Build Cache Management Script
# Manage Docker build cache for optimal performance

param(
    [switch]$Clean,
    [switch]$List,
    [switch]$Prune,
    [switch]$Stats,
    [string]$ImageName = "bitcoin-sprint"
)

Write-Host "🔧 Docker Build Cache Manager" -ForegroundColor Green
Write-Host "================================" -ForegroundColor Yellow

if ($List) {
    Write-Host "`n📋 Docker Images:" -ForegroundColor Cyan
    docker images "$ImageName*" --format "table {{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.Size}}\t{{.CreatedAt}}"

    Write-Host "`n📦 Build Cache:" -ForegroundColor Cyan
    docker buildx ls
}

if ($Stats) {
    Write-Host "`n📊 Docker System Stats:" -ForegroundColor Cyan
    docker system df

    Write-Host "`n🏗️  Build Cache Usage:" -ForegroundColor Cyan
    docker buildx du
}

if ($Clean) {
    Write-Host "`n🧹 Cleaning old images..." -ForegroundColor Cyan

    # Remove dangling images
    $dangling = docker images -f "dangling=true" -q
    if ($dangling) {
        docker rmi $dangling
        Write-Host "✅ Removed dangling images" -ForegroundColor Green
    } else {
        Write-Host "ℹ️  No dangling images found" -ForegroundColor Blue
    }

    # Remove old images (keep last 3 versions)
    Write-Host "`n🗑️  Removing old $ImageName images (keeping latest 3)..." -ForegroundColor Cyan
    $images = docker images "$ImageName*" --format "{{.Repository}}:{{.Tag}}" | Where-Object { $_ -notmatch ":latest$" -and $_ -notmatch ":cache$" }
    if ($images.Count -gt 3) {
        $toRemove = $images | Select-Object -Skip 3
        foreach ($img in $toRemove) {
            docker rmi $img
        }
        Write-Host "✅ Removed $($toRemove.Count) old images" -ForegroundColor Green
    } else {
        Write-Host "ℹ️  No old images to remove" -ForegroundColor Blue
    }
}

if ($Prune) {
    Write-Host "`n🧽 Pruning build cache..." -ForegroundColor Cyan
    docker builder prune -f
    Write-Host "✅ Build cache pruned" -ForegroundColor Green

    Write-Host "`n🧹 Pruning system cache..." -ForegroundColor Cyan
    docker system prune -f
    Write-Host "✅ System cache pruned" -ForegroundColor Green
}

# Default action - show usage
if (-not ($List -or $Stats -or $Clean -or $Prune)) {
    Write-Host "`n📖 Usage:" -ForegroundColor Cyan
    Write-Host "  .\cache-manager.ps1 -List          # List images and cache" -ForegroundColor White
    Write-Host "  .\cache-manager.ps1 -Stats         # Show cache statistics" -ForegroundColor White
    Write-Host "  .\cache-manager.ps1 -Clean         # Clean old images" -ForegroundColor White
    Write-Host "  .\cache-manager.ps1 -Prune         # Prune build and system cache" -ForegroundColor White
    Write-Host "  .\cache-manager.ps1 -List -Stats   # Show everything" -ForegroundColor White

    Write-Host "`n💡 Examples:" -ForegroundColor Blue
    Write-Host "  .\cache-manager.ps1 -Clean -Prune  # Full cleanup" -ForegroundColor Gray
    Write-Host "  .\cache-manager.ps1 -Stats         # Check cache usage" -ForegroundColor Gray
}

Write-Host "`n🎯 Cache management complete!" -ForegroundColor Green
