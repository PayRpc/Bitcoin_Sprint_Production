# Optimized Docker Build Script for Bitcoin Sprint
# Features: BuildKit, layer caching, multi-platform builds, and performance optimizations

param(
    [string]$Tag = "latest",
    [string]$Registry = "",
    [switch]$NoCache,
    [switch]$Push,
    [switch]$MultiPlatform,
    [string]$BuildContext = ".",
    [string]$Dockerfile = "Dockerfile.optimized"
)

Write-Host "Bitcoin Sprint Docker Build Optimizer" -ForegroundColor Green
Write-Host "=========================================" -ForegroundColor Yellow

# Enable BuildKit for better performance
$env:DOCKER_BUILDKIT = 1
$env:COMPOSE_DOCKER_CLI_BUILD = 1

# Build arguments
$buildArgs = @(
    "--build-arg", "BUILD_DATE=$(Get-Date -Format 'yyyy-MM-ddTHH:mm:ssZ')",
    "--build-arg", "BUILD_VERSION=$Tag",
    "--build-arg", "GIT_COMMIT=$(git rev-parse --short HEAD 2>$null)"
)

# Cache configuration
if (-not $NoCache) {
    $cacheImage = if ($Registry) { "$Registry/bitcoin-sprint:cache" } else { "bitcoin-sprint:cache" }
    $buildArgs += @("--cache-from", $cacheImage)
}

# Multi-platform support
if ($MultiPlatform) {
    $buildArgs += @("--platform", "linux/amd64,linux/arm64")
}

# Dockerfile specification
$buildArgs += @("-f", $Dockerfile)

# Image tag
$imageTag = if ($Registry) { "$Registry/bitcoin-sprint:$Tag" } else { "bitcoin-sprint:$Tag" }
$buildArgs += @("-t", $imageTag)

# Build context
$buildArgs += @($BuildContext)

Write-Host "`nBuild Configuration:" -ForegroundColor Cyan
Write-Host "  Image Tag: $imageTag" -ForegroundColor White
Write-Host "  Dockerfile: $Dockerfile" -ForegroundColor White
Write-Host "  Build Context: $BuildContext" -ForegroundColor White
Write-Host "  BuildKit: Enabled" -ForegroundColor White
Write-Host "  Cache: $(if ($NoCache) { 'Disabled' } else { 'Enabled' })" -ForegroundColor White
Write-Host "  Multi-platform: $(if ($MultiPlatform) { 'Enabled' } else { 'Disabled' })" -ForegroundColor White

# Execute build
Write-Host "`nBuilding Docker image..." -ForegroundColor Cyan
$buildCommand = "docker build " + ($buildArgs -join " ")
Write-Host "Command: $buildCommand" -ForegroundColor Gray

try {
    $startTime = Get-Date
    Invoke-Expression $buildCommand

    if ($LASTEXITCODE -eq 0) {
        $endTime = Get-Date
        $duration = $endTime - $startTime

    Write-Host "`nBuild completed successfully!" -ForegroundColor Green
    Write-Host "Build time: $($duration.TotalSeconds) seconds" -ForegroundColor White

        # Save cache image
        if (-not $NoCache) {
            Write-Host "`nSaving build cache..." -ForegroundColor Cyan
            docker tag $imageTag $cacheImage
        }

        # Push if requested
        if ($Push) {
            Write-Host "`nPushing image to registry..." -ForegroundColor Cyan
            docker push $imageTag
            if (-not $NoCache) {
                docker push $cacheImage
            }
        }

        # Show image size
        $imageSize = docker images $imageTag --format "table {{.Repository}}:{{.Tag}}\t{{.Size}}"
    Write-Host "`nImage Details:" -ForegroundColor Cyan
        Write-Host $imageSize -ForegroundColor White

    } else {
    Write-Host "`nBuild failed!" -ForegroundColor Red
        exit 1
    }

} catch {
    Write-Host "`nBuild error: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

Write-Host "`nDocker build optimization complete!" -ForegroundColor Green
Write-Host "`nTips for even faster builds:" -ForegroundColor Blue
Write-Host "  - Use './build-optimized.ps1 -NoCache' for clean builds" -ForegroundColor Gray
Write-Host "  - Use './build-optimized.ps1 -Push' to build and push" -ForegroundColor Gray
Write-Host "  - Use './build-optimized.ps1 -MultiPlatform' for multi-arch builds" -ForegroundColor Gray
Write-Host "  - Keep .dockerignore updated to minimize build context" -ForegroundColor Gray
