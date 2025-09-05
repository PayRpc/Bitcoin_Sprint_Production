# Bitcoin Sprint Monitoring

This directory contains configurations for monitoring Bitcoin Sprint with Prometheus and Grafana.

## Quick Start

Run the monitoring stack with:

```powershell
# Basic monitoring (Grafana + Prometheus)
./start-monitoring.ps1

# Complete monitoring (including PostgreSQL and exporters)
./start-monitoring.ps1 -WithPostgres
```

## Access Dashboards

- **Grafana**: http://localhost:3000 (default login: admin/admin)
- **Prometheus**: http://localhost:9090

## Components

- **Grafana**: Visualization and dashboards
- **Prometheus**: Metrics collection and storage
- **PostgreSQL Exporter**: Exports database metrics
- **Node Exporter**: Exports system metrics

## Available Dashboards

- **Bitcoin Sprint Overview**: General application metrics
- **PQC Validator Metrics**: Entropy PQC weight and validation metrics
- **Database Performance**: PostgreSQL metrics and query performance

## Adding Custom Dashboards

1. Create your dashboard JSON in `monitoring/grafana/json/`
2. Restart the monitoring stack or use the Grafana UI

## Metrics Available

- `entropy_pqc_weight`: Current PQC entropy weighting (0.0-1.0)
- `go_goroutines`: Number of active goroutines
- Database metrics (via PostgreSQL exporter)
- System metrics (via Node exporter)

## Alerting

Alerts are configured in `monitoring/alerts.yml` and include:

- Service availability
- Memory usage
- Slow database queries

## Integration with CI/CD

For CI/CD, add a step to your pipeline to check metrics:

```yaml
- name: Check Metrics
  run: ./scripts/check-metrics.ps1
```
