# Bitcoin Sprint Production Installer
# Run as Administrator for optimal performance

param(
    [string]$InstallPath = "C:\Bitcoin-Sprint",
    [string]$ServiceName = "BitcoinSprint",
    [switch]$CreateService = $false
)

Write-Host "🚀 Bitcoin Sprint Production Installer" -ForegroundColor Green

# Create installation directory
if (!(Test-Path $InstallPath)) {
    New-Item -ItemType Directory -Force -Path $InstallPath | Out-Null
    Write-Host "✅ Created installation directory: $InstallPath" -ForegroundColor Green
}

# Copy files
Write-Host "📦 Installing files..." -ForegroundColor Cyan
Copy-Item -Recurse -Force "bin\*" $InstallPath
Copy-Item -Recurse -Force "config\*" $InstallPath  
Copy-Item -Recurse -Force "licenses\*" $InstallPath

# Create default config if none exists
if (!(Test-Path "$InstallPath\config.json")) {
    Copy-Item "$InstallPath\config-production-optimized.json" "$InstallPath\config.json"
    Write-Host "✅ Created default configuration" -ForegroundColor Green
}

# Create default license if none exists  
if (!(Test-Path "$InstallPath\license.json")) {
    if (Test-Path "$InstallPath\license-enterprise.json") {
        Copy-Item "$InstallPath\license-enterprise.json" "$InstallPath\license.json"
        Write-Host "✅ Applied enterprise license" -ForegroundColor Green
    } else {
        Copy-Item "$InstallPath\license-demo-free.json" "$InstallPath\license.json"
        Write-Host "✅ Applied free license" -ForegroundColor Yellow
    }
}

if ($CreateService) {
    Write-Host "⚙️ Creating Windows service..." -ForegroundColor Cyan
    # Service creation logic would go here
    Write-Host "ℹ️ Service creation requires additional configuration" -ForegroundColor Yellow
}

Write-Host "✅ Installation complete!" -ForegroundColor Green
Write-Host "Start Bitcoin Sprint: $InstallPath\bitcoin-sprint-production.exe" -ForegroundColor White
