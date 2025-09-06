# Start Grafana and Prometheus for Bitcoin Sprint monitoring
param(
    [switch]$Force = $false,
    [switch]$WithPostgres = $false
)

$ErrorActionPreference = "Stop"

# Check Docker is running
try {
    docker info > $null
}
catch {
    Write-Host "Error: Docker is not running. Please start Docker Desktop and try again." -ForegroundColor Red
    exit 1
}

# Clean up existing containers if Force is used
if ($Force) {
    Write-Host "Force flag set, stopping and removing existing containers..." -ForegroundColor Yellow
    docker-compose -f docker-compose.grafana.yml down 2>$null
}

# Check if we need to include PostgreSQL and exporters
if ($WithPostgres) {
    Write-Host "Starting monitoring stack with PostgreSQL and exporters..." -ForegroundColor Cyan
    
    # Create the combined docker-compose file
    $composeContent = @"
version: "3.9"
services:
  prometheus:
    image: prom/prometheus:v2.53.0
    command:
      - --config.file=/etc/prometheus/prometheus.yml
      - --storage.tsdb.retention.time=15d
      - --web.enable-lifecycle
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - ./monitoring/alerts.yml:/etc/prometheus/alerts.yml:ro
    ports:
      - "9090:9090"

  grafana:
    image: grafana/grafana:11.1.3
    environment:
      - GF_AUTH_ANONYMOUS_ENABLED=false
      - GF_SECURITY_ADMIN_USER=admin
      - GF_SECURITY_ADMIN_PASSWORD=admin
      - GF_USERS_DEFAULT_THEME=light
    volumes:
      - ./monitoring/grafana/datasources:/etc/grafana/provisioning/datasources:ro
      - ./monitoring/grafana/dashboards:/etc/grafana/provisioning/dashboards:ro
      - ./monitoring/grafana/json:/var/lib/grafana/dashboards:ro
    ports:
      - "3000:3000"
    depends_on:
      - prometheus

  postgres:
    image: postgres:16
    environment:
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=postgres
      - POSTGRES_DB=bitcoin_sprint
    volumes:
      - ./db/init-db.sql:/docker-entrypoint-initdb.d/init-db.sql:ro
    ports:
      - "5432:5432"

  postgres-exporter:
    image: prometheuscommunity/postgres-exporter:v0.15.0
    environment:
      - DATA_SOURCE_NAME=postgresql://postgres:postgres@postgres:5432/bitcoin_sprint?sslmode=disable
    ports:
      - "9187:9187"
    depends_on:
      - postgres

  node-exporter:
    image: prom/node-exporter:v1.7.0
    volumes:
      - /proc:/host/proc:ro
      - /sys:/host/sys:ro
      - /:/rootfs:ro
    command:
      - --path.procfs=/host/proc
      - --path.sysfs=/host/sys
      - --collector.filesystem.mount-points-exclude=^/(sys|proc|dev|host|etc)($$|/)
    ports:
      - "9100:9100"
"@

    $composeContent | Out-File -FilePath "docker-compose.monitoring-full.yml" -Encoding utf8
    docker-compose -f docker-compose.monitoring-full.yml up -d

} else {
    Write-Host "Starting monitoring stack (Grafana and Prometheus only)..." -ForegroundColor Cyan
    docker-compose -f docker-compose.grafana.yml up -d
}

Write-Host "`n✅ Monitoring stack started!" -ForegroundColor Green
Write-Host "📊 Grafana:    http://localhost:3000 (admin/admin)" -ForegroundColor Cyan
Write-Host "📈 Prometheus: http://localhost:9090" -ForegroundColor Cyan
if ($WithPostgres) {
    Write-Host "🛢️  PostgreSQL: localhost:5432 (postgres/postgres)" -ForegroundColor Cyan
}
