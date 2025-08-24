# Bitcoin Sprint - Optimized Build Script
# This script builds Bitcoin Sprint with hot path optimizations for maximum performance

param(
    [string]$OutputName = "bitcoin-sprint-optimized.exe",
    [switch]$Release,
    [switch]$Benchmark,
    [switch]$Turbo,
    [string]$TargetOS = "windows",
    [string]$TargetArch = "amd64",
    [string]$Version = "1.0.3",
    [string]$Commit = ""
)

Write-Host "Bitcoin Sprint - Building with Hot Path Optimizations" -ForegroundColor Cyan
if ($Turbo) {
    Write-Host "TURBO MODE BUILD: Ultra-aggressive optimizations enabled" -ForegroundColor Yellow
}

# Set build environment for maximum performance and reproducible builds
# Fail-safe: ensure CGO is disabled for static, reproducible binaries
if ($env:CGO_ENABLED -ne "0") { $env:CGO_ENABLED = "0" }
$env:CGO_ENABLED = "0"

# Reproducible builds: enforce module download lock
$env:GOFLAGS = "-mod=readonly"

# Cross-compile targets (can be overridden via params)
$env:GOOS = $TargetOS
$env:GOARCH = $TargetArch

# Ultra-optimized build flags for claimed 1.5-4.0s performance advantage
$ldflagsInner = "-s -w -extldflags=-static"
# Embed version and commit if available; prefer explicit -Commit for CI reproducibility
if ($Commit -ne "") {
    $commit = $Commit
} else {
    try {
        $commit = (git rev-parse --short HEAD) -replace "`n",""
    } catch {
        $commit = "unknown"
    }
}

# Add linker flags for versioning (helps trace binaries in CI/prod)
$ldflagsInner = "$ldflagsInner -X main.Version=$Version -X main.Commit=$commit"

$buildFlags = @(
    "-ldflags=$ldflagsInner"  # Strip symbols, reduce binary size, static linking + version embed
    "-trimpath"               # Remove file system paths for deterministic builds
    "-buildmode=exe"          # Optimize for executable
)

if ($Release) {
    Write-Host "RELEASE BUILD: Maximum optimization enabled" -ForegroundColor Green
    $buildFlags += @(
        "-tags=release,netgo"             # Release tags for conditional compilation
        "-a"                              # Force rebuild all packages
        "-installsuffix=netgo"            # Pure Go networking
    )
} else {
    Write-Host "DEVELOPMENT BUILD: Fast compilation with optimizations" -ForegroundColor Yellow
}

if ($Benchmark) {
    Write-Host "BENCHMARK BUILD: Performance testing enabled" -ForegroundColor Magenta
    # In development benchmarks we enable the race detector to catch issues.
    if ($Release) {
        # Release benchmarks should avoid -race for maximal performance
        $buildFlags += "-tags=benchmark"
    } else {
        $buildFlags += "-tags=benchmark"
        $buildFlags += "-race"
    }
}

if ($Turbo) {
    Write-Host "TURBO BUILD: Including turbo mode optimizations" -ForegroundColor Yellow
    $buildFlags += "-tags=turbo"
}

# Execute optimized build
Write-Host "Building with flags: $($buildFlags -join ' ')" -ForegroundColor DarkGray

try {
    $startTime = Get-Date
    
    & go build $buildFlags -o $OutputName .
    
    if ($LASTEXITCODE -eq 0) {
        $buildTime = (Get-Date) - $startTime
        $fileInfo = Get-Item $OutputName

        Write-Host "BUILD SUCCESS!" -ForegroundColor Green
        Write-Host "   Binary: $OutputName" -ForegroundColor White
        Write-Host "   Size: $([math]::Round($fileInfo.Length / 1MB, 2)) MB" -ForegroundColor White
        Write-Host "   Build time: $([math]::Round($buildTime.TotalSeconds, 2))s" -ForegroundColor White
        Write-Host ""
        Write-Host "Optimizations applied:" -ForegroundColor Cyan
        Write-Host "   • Binary size reduction (-s -w)" -ForegroundColor Gray
        Write-Host "   • Static linking for deployment" -ForegroundColor Gray
        Write-Host "   • Path trimming for consistency" -ForegroundColor Gray
        Write-Host "   • Hot path RPC optimizations" -ForegroundColor Gray
        Write-Host "   • Parallel fan-out patterns" -ForegroundColor Gray
        Write-Host "   • Pre-marshaled request buffers" -ForegroundColor Gray
        Write-Host "   • Reduced lock contention" -ForegroundColor Gray
        
        if ($Turbo) {
            Write-Host "   - TURBO: Parallel RPC fan-out" -ForegroundColor Yellow
            Write-Host "   - TURBO: Pre-encoded payloads" -ForegroundColor Yellow
            Write-Host "   - TURBO: 200ms peer deadlines" -ForegroundColor Yellow
            Write-Host "   - TURBO: 500ms mempool predictor" -ForegroundColor Yellow
            Write-Host "   - TURBO: Async logging" -ForegroundColor Yellow
        }

        # Verify binary prints version and exits quickly to avoid spinning
        $versionOK = $false
        try {
            $proc = Start-Process -FilePath (Resolve-Path $OutputName) -ArgumentList "--version" -NoNewWindow -RedirectStandardOutput tmp_ver.txt -PassThru -Wait -ErrorAction Stop
            if ($proc.ExitCode -eq 0) {
                $verOut = Get-Content tmp_ver.txt -Raw
                Write-Host "   Version output: $verOut" -ForegroundColor Gray
                Remove-Item tmp_ver.txt -ErrorAction SilentlyContinue
                $versionOK = $true
            }
        } catch {
            # fallback to existence check only
            Write-Host "   Warning: could not run binary to check --version; skipping runtime check" -ForegroundColor Yellow
        }

        if (-not $versionOK -and (Test-Path $OutputName)) {
            Write-Host "   Note: Binary exists at $OutputName (version check skipped or failed)" -ForegroundColor Yellow
        }
        Write-Host "   Embedded version: $Version" -ForegroundColor Gray
        Write-Host "   Embedded commit:  $commit" -ForegroundColor Gray
        
        if ($Release) {
            Write-Host ""
            Write-Host "Ready for performance testing against claimed advantages:" -ForegroundColor Green
            if ($Turbo) {
                Write-Host "   - TURBO MODE: 2.0-4.6s improvement over Bitcoin Core" -ForegroundColor Yellow
                Write-Host "   - Trading firms: Ultra-aggressive edge" -ForegroundColor White
                Write-Host "   - Mining operations: Maximum detection speed" -ForegroundColor White
                Write-Host "   - Enterprise: Parallel fan-out advantage" -ForegroundColor White
            } else {
                Write-Host "   - SAFE MODE: 1.5-2.3s improvement over Bitcoin Core" -ForegroundColor Cyan
                Write-Host "   - Trading firms: Conservative advantage" -ForegroundColor White
                Write-Host "   - Mining operations: Balanced performance" -ForegroundColor White
                Write-Host "   - Standard: Resource-efficient edge" -ForegroundColor White
            }
        }
        
    } else {
        Write-Host "BUILD FAILED with exit code $LASTEXITCODE" -ForegroundColor Red
        exit $LASTEXITCODE
    }
    
} catch {
    Write-Host "BUILD ERROR: $_" -ForegroundColor Red
    exit 1
}
