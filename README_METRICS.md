# Bitcoin Sprint Metrics Server

This document describes how to configure and maintain the Bitcoin Sprint Metrics Server, which provides Prometheus-compatible metrics for the Ethereum blockchain and API.

## Overview

The metrics server exposes the following metrics:

- `sprint_chain_block_height{chain="ethereum"}`: Current Ethereum block height
- `sprint_chain_peer_count{chain="ethereum"}`: Number of connected Ethereum peers
- `sprint_chain_health_score{chain="ethereum"}`: Health score of the Ethereum blockchain connection
- `sprint_api_requests_total{chain="ethereum", method="..."}`: Total count of API requests by method
- `sprint_api_request_duration_seconds{chain="ethereum", method="..."}`: Histogram of API request durations

## Port Configuration

The metrics server runs on port 8081 by default. This port is configured in the metrics_server.go file.

## Starting the Server

There are three ways to start the metrics server:

### 1. Manual Start

```powershell
cd c:\Projects\Bitcoin-Sprint-5\BItcoin_Sprint
go run metrics_server.go
```

### 2. Startup Script

```powershell
.\start-metrics-server.ps1
```

This script:
- Checks if the server is already running
- Builds the server executable if needed
- Starts the server as a background process
- Verifies the server is running correctly

### 3. Docker Integration

```powershell
.\start-docker-metrics-server.ps1
```

This script:
- Ensures the metrics server is accessible from Docker containers
- Creates a firewall rule if needed
- Verifies the server is running correctly

## Scheduled Task

A Windows scheduled task can be created to start the metrics server automatically on system startup:

```powershell
.\register-metrics-server-task.ps1
```

## Prometheus Configuration

The Prometheus configuration in `prometheus.yml` includes a job for the metrics server:

```yaml
- job_name: 'sprint-metrics-server'
  static_configs:
    - targets: ['host.docker.internal:8081']
  metrics_path: '/metrics'
  scrape_interval: 15s
```

## Troubleshooting

If metrics aren't appearing in Grafana:

1. Check if the metrics server is running:
   ```powershell
   netstat -ano | findstr :8081
   ```

2. Test the metrics endpoint directly:
   ```powershell
   curl http://localhost:8081/metrics | findstr sprint_chain
   ```

3. Check Prometheus targets in the Prometheus UI (usually at http://localhost:9090/targets)

4. Verify the Grafana dashboard is using the correct Prometheus data source and queries

## Maintenance

- The metrics server generates simulated data for demonstration purposes.
- For production use, modify the `updateMetrics()` function to fetch real data from Ethereum nodes.
- Monitor the server periodically to ensure it's running correctly.

## Restarting the Server

If you need to restart the server:

```powershell
Stop-Process -Name "metrics_server" -Force -ErrorAction SilentlyContinue
Start-Sleep 2
.\start-metrics-server.ps1
```
