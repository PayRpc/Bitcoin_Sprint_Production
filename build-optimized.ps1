# Bitcoin Sprint - Optimized Build Script
# This script builds Bitcoin Sprint with hot path optimizations for maximum performance

param(
    [string]$OutputName = "bitcoin-sprint-optimized.exe",
    [switch]$Release,
    [switch]$Benchmark
)

Write-Host "🚀 Bitcoin Sprint - Building with Hot Path Optimizations" -ForegroundColor Cyan

# Set build environment for maximum performance
$env:CGO_ENABLED = "0"
$env:GOOS = "windows"
$env:GOARCH = "amd64"

# Ultra-optimized build flags for claimed 1.5-4.0s performance advantage
$buildFlags = @(
    "-ldflags=-s -w -extldflags=-static"  # Strip symbols, reduce binary size, static linking
    "-trimpath"                           # Remove file system paths for deterministic builds
    "-buildmode=exe"                      # Optimize for executable
)

if ($Release) {
    Write-Host "🎯 RELEASE BUILD: Maximum optimization enabled" -ForegroundColor Green
    $buildFlags += @(
        "-tags=release,netgo"             # Release tags for conditional compilation
        "-a"                              # Force rebuild all packages
        "-installsuffix=netgo"            # Pure Go networking
    )
} else {
    Write-Host "🔧 DEVELOPMENT BUILD: Fast compilation with optimizations" -ForegroundColor Yellow
}

if ($Benchmark) {
    Write-Host "📊 BENCHMARK BUILD: Performance testing enabled" -ForegroundColor Magenta
    $buildFlags += "-tags=benchmark"
}

# Execute optimized build
Write-Host "Building with flags: $($buildFlags -join ' ')" -ForegroundColor DarkGray

try {
    $startTime = Get-Date
    
    & go build $buildFlags -o $OutputName .
    
    if ($LASTEXITCODE -eq 0) {
        $buildTime = (Get-Date) - $startTime
        $fileInfo = Get-Item $OutputName
        
        Write-Host "✅ BUILD SUCCESS!" -ForegroundColor Green
        Write-Host "   Binary: $OutputName" -ForegroundColor White
        Write-Host "   Size: $([math]::Round($fileInfo.Length / 1MB, 2)) MB" -ForegroundColor White
        Write-Host "   Build time: $([math]::Round($buildTime.TotalSeconds, 2))s" -ForegroundColor White
        Write-Host ""
        Write-Host "⚡ Optimizations applied:" -ForegroundColor Cyan
        Write-Host "   • Binary size reduction (-s -w)" -ForegroundColor Gray
        Write-Host "   • Static linking for deployment" -ForegroundColor Gray
        Write-Host "   • Path trimming for consistency" -ForegroundColor Gray
        Write-Host "   • Hot path RPC optimizations" -ForegroundColor Gray
        Write-Host "   • Parallel fan-out patterns" -ForegroundColor Gray
        Write-Host "   • Pre-marshaled request buffers" -ForegroundColor Gray
        Write-Host "   • Reduced lock contention" -ForegroundColor Gray
        
        if ($Release) {
            Write-Host ""
            Write-Host "🎯 Ready for performance testing against claimed advantages:" -ForegroundColor Green
            Write-Host "   • Trading firms: 1.5-2.3s improvement" -ForegroundColor White
            Write-Host "   • Mining operations: 1.8-2.1s improvement" -ForegroundColor White
            Write-Host "   • Exchange infrastructure: 1.9-2.4s improvement" -ForegroundColor White
        }
        
    } else {
        Write-Host "❌ BUILD FAILED with exit code $LASTEXITCODE" -ForegroundColor Red
        exit $LASTEXITCODE
    }
    
} catch {
    Write-Host "❌ BUILD ERROR: $_" -ForegroundColor Red
    exit 1
}
