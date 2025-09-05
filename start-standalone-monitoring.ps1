# Start standalone Grafana and Prometheus for Bitcoin Sprint monitoring
# This script doesn't require Docker, just downloads and runs the binaries directly
param(
    [switch]$ForceDownload = $false
)

$ErrorActionPreference = "Stop"
$ProgressPreference = 'SilentlyContinue' # Speed up downloads

# Setup directories
$monitoringDir = ".\monitoring"
$binariesDir = "$monitoringDir\binaries"
$grafanaDir = "$binariesDir\grafana"
$prometheusDir = "$binariesDir\prometheus"

if (!(Test-Path $binariesDir)) {
    New-Item -ItemType Directory -Path $binariesDir -Force | Out-Null
}

# Define binaries
$grafanaVersion = "11.1.3"
$grafanaUrl = "https://dl.grafana.com/oss/release/grafana-$grafanaVersion.windows-amd64.zip"
$grafanaZip = "$binariesDir\grafana-$grafanaVersion.zip"

$prometheusVersion = "2.49.0"
$prometheusUrl = "https://github.com/prometheus/prometheus/releases/download/v$prometheusVersion/prometheus-$prometheusVersion.windows-amd64.zip"
$prometheusZip = "$binariesDir\prometheus-$prometheusVersion.zip"

# Download and extract Grafana if needed
if (!(Test-Path $grafanaDir) -or $ForceDownload) {
    Write-Host "Downloading Grafana $grafanaVersion..." -ForegroundColor Cyan
    if (!(Test-Path $grafanaZip) -or $ForceDownload) {
        Invoke-WebRequest -Uri $grafanaUrl -OutFile $grafanaZip
    }
    
    Write-Host "Extracting Grafana..." -ForegroundColor Cyan
    if (Test-Path $grafanaDir) {
        Remove-Item -Path $grafanaDir -Recurse -Force
    }
    
    # Extract and rename directory
    Expand-Archive -Path $grafanaZip -DestinationPath $binariesDir
    Get-ChildItem -Path $binariesDir -Filter "grafana-*-windows-amd64" | 
        Rename-Item -NewName "grafana"
}

# Download and extract Prometheus if needed
if (!(Test-Path $prometheusDir) -or $ForceDownload) {
    Write-Host "Downloading Prometheus $prometheusVersion..." -ForegroundColor Cyan
    if (!(Test-Path $prometheusZip) -or $ForceDownload) {
        Invoke-WebRequest -Uri $prometheusUrl -OutFile $prometheusZip
    }
    
    Write-Host "Extracting Prometheus..." -ForegroundColor Cyan
    if (Test-Path $prometheusDir) {
        Remove-Item -Path $prometheusDir -Recurse -Force
    }
    
    # Extract and rename directory
    Expand-Archive -Path $prometheusZip -DestinationPath $binariesDir
    Get-ChildItem -Path $binariesDir -Filter "prometheus-*-windows-amd64" | 
        Rename-Item -NewName "prometheus"
}

# Configure Grafana
$grafanaConfDir = "$grafanaDir\conf"

# Setup directories for provisioning
$provisioningDir = "$grafanaDir\conf\provisioning"
$datasourcesDir = "$provisioningDir\datasources"
$dashboardsDir = "$provisioningDir\dashboards"

if (!(Test-Path $datasourcesDir)) {
    New-Item -ItemType Directory -Path $datasourcesDir -Force | Out-Null
}

if (!(Test-Path $dashboardsDir)) {
    New-Item -ItemType Directory -Path $dashboardsDir -Force | Out-Null
}

# Copy datasource config
Copy-Item -Path "$monitoringDir\grafana\datasources\datasource.yaml" -Destination "$datasourcesDir\" -Force

# Copy dashboard config
Copy-Item -Path "$monitoringDir\grafana\dashboards\dashboard.yaml" -Destination "$dashboardsDir\" -Force

# Copy dashboards
$dashboardDataDir = "$grafanaDir\data\dashboards"
if (!(Test-Path $dashboardDataDir)) {
    New-Item -ItemType Directory -Path $dashboardDataDir -Force | Out-Null
}
Copy-Item -Path "$monitoringDir\grafana\json\*.json" -Destination "$dashboardDataDir\" -Force

# Copy Prometheus config
Copy-Item -Path "$monitoringDir\prometheus.yml" -Destination "$prometheusDir\" -Force

# Start Prometheus
Write-Host "Starting Prometheus..." -ForegroundColor Cyan
$prometheusProcess = Start-Process -FilePath "$prometheusDir\prometheus.exe" -ArgumentList "--config.file=prometheus.yml" -WorkingDirectory $prometheusDir -PassThru -WindowStyle Hidden

# Start Grafana
Write-Host "Starting Grafana..." -ForegroundColor Cyan
$grafanaProcess = Start-Process -FilePath "$grafanaDir\bin\grafana-server.exe" -WorkingDirectory $grafanaDir -PassThru -WindowStyle Hidden

# Write process IDs to file for later cleanup
$prometheusProcess.Id, $grafanaProcess.Id | Out-File "$binariesDir\monitoring.pid"

# Wait for services to start
Start-Sleep -Seconds 5

Write-Host "`n✅ Standalone monitoring stack started!" -ForegroundColor Green
Write-Host "📊 Grafana:    http://localhost:3000 (admin/admin)" -ForegroundColor Cyan
Write-Host "📈 Prometheus: http://localhost:9090" -ForegroundColor Cyan
Write-Host "`nTo stop the services, run stop-standalone-monitoring.ps1" -ForegroundColor Yellow
