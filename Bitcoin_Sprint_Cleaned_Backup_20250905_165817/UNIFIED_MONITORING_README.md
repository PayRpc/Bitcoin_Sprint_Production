# Unified Monitoring Stack for Bitcoin Sprint

This directory has been updated with a new unified monitoring solution that includes Prometheus, Grafana, and a sample metrics exporter for Solana blockchain metrics.

## Quick Start with Unified Stack

Use the monitoring manager script to control the unified monitoring stack:

```powershell
# Start the unified stack
./monitoring-manager.ps1 -Start

# Check status
./monitoring-manager.ps1 -Status

# Open Grafana dashboard
./monitoring-manager.ps1 -Dashboard

# View logs
./monitoring-manager.ps1 -Logs

# Stop the stack
./monitoring-manager.ps1 -Stop
```

## Legacy Monitoring

The original monitoring configuration is still available:

```powershell
# Basic monitoring (Grafana + Prometheus)
./start-monitoring.ps1

# Complete monitoring (including PostgreSQL and exporters)
./start-monitoring.ps1 -WithPostgres
```

## Unified Stack Components

- **Solana Exporter**: A custom metrics exporter that provides Solana blockchain metrics
- **Prometheus**: Time series database for storing metrics
- **Grafana**: Data visualization platform with pre-configured dashboards

## Accessing Components

- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (login with admin/admin)

## New Unified Dashboards

The new unified monitoring stack comes with additional pre-configured dashboards:

1. **Solana Metrics Dashboard**: Shows Solana blockchain metrics including:
   - Transaction count
   - Transactions per second (TPS)
   - Block height
   - Active validators
   - Total stake

## Extending the Unified Monitoring Stack

To add new metrics:

1. Add new metric collectors to the sample-metrics.py file
2. Update the Prometheus configuration in prometheus-simple.yml
3. Create or modify dashboards in Grafana

## Configuration Files

- **docker-compose.unified.yml**: Main compose file for the unified stack
- **monitoring/prometheus-simple.yml**: Unified Prometheus configuration
- **monitoring/sample-metrics.py**: Sample metrics exporter for Solana
- **monitoring/grafana/dashboards/**: Pre-configured Grafana dashboards
