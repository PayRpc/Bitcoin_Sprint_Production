# Bitcoin Sprint - Entropy Bridge Monitor

This document describes the entropy monitoring system for the Bitcoin Sprint application, providing real-time visibility into the entropy bridge's performance, security, and operational status.

## 🔐 Overview

The Entropy Bridge Monitor provides comprehensive monitoring of the entropy generation system used for secure admin authentication and cryptographic operations. It tracks availability, performance, and quality metrics to ensure the security infrastructure remains robust.

## 📊 Dashboard Features

### Core Metrics Monitored

1. **Entropy Bridge Availability**
   - Real-time status of the entropy bridge
   - Rust vs Node.js fallback mode detection
   - Service availability indicators

2. **Performance Metrics**
   - Secret generation response times
   - Authentication performance
   - Throughput rates

3. **Security Quality**
   - Entropy quality scores
   - Secret generation success rates
   - Authentication attempt patterns

4. **Operational Status**
   - Fallback mode activation
   - Error rates and patterns
   - System health indicators

## 🚀 Setup Instructions

### 1. Start the Application

```bash
cd web
npm run dev
```

### 2. Start Grafana (if using Docker)

```bash
docker-compose up grafana
```

Or for the full stack:
```bash
docker-compose up
```

### 3. Access the Dashboard

- **Grafana URL**: http://localhost:3000
- **Dashboard**: "Bitcoin Sprint - Entropy Bridge Monitor"
- **Prometheus**: http://localhost:9090 (if running)

### 4. Test the Monitoring System

```bash
npm run test:monitor
```

## 📈 Metrics Details

### Prometheus Metrics

| Metric | Description | Type |
|--------|-------------|------|
| `bitcoin_sprint_entropy_bridge_available` | Bridge availability status | Gauge |
| `bitcoin_sprint_entropy_bridge_rust_available` | Rust implementation status | Gauge |
| `bitcoin_sprint_entropy_bridge_fallback_mode` | Fallback mode indicator | Gauge |
| `bitcoin_sprint_entropy_secret_generation_total` | Total secrets generated | Counter |
| `bitcoin_sprint_entropy_secret_generation_duration_seconds` | Generation performance | Histogram |
| `bitcoin_sprint_entropy_quality_score` | Entropy quality metric | Gauge |
| `bitcoin_sprint_entropy_admin_auth_attempts_total` | Admin auth attempts | Counter |
| `bitcoin_sprint_entropy_admin_auth_duration_seconds` | Auth performance | Histogram |

### API Endpoints

- **Metrics Endpoint**: `GET /api/prometheus`
  - Returns all Prometheus metrics in the standard format
  - Used by Prometheus for metric collection

- **Entropy Status**: `GET /api/admin/entropy-status`
  - Provides real-time entropy bridge status
  - Triggers metric collection and updates

## 🔧 Configuration

### Grafana Dashboard

The dashboard is automatically provisioned from:
```
grafana/dashboards/grafana-dashboard-entropy-bridge.json
```

### Prometheus Configuration

Metrics are exposed via the `/api/prometheus` endpoint and collected by Prometheus running at `http://prometheus:9090`.

### Dashboard Refresh

- **Auto-refresh**: 30 seconds
- **Time range**: Last 1 hour (configurable)
- **Update interval**: 30 seconds (configurable)

## 📊 Dashboard Panels

### 1. Entropy Bridge Availability
- **Type**: Gauge
- **Purpose**: Shows real-time availability of the entropy bridge
- **Thresholds**: Green (1) = Available, Red (0) = Unavailable

### 2. Rust Entropy Bridge Status
- **Type**: Gauge
- **Purpose**: Indicates if the native Rust implementation is active
- **Thresholds**: Green (1) = Rust available, Red (0) = Node.js fallback

### 3. Secret Generation Performance
- **Type**: Time Series
- **Purpose**: Monitors response times for entropy secret generation
- **Metrics**: Average generation time over 5-minute windows

### 4. Secret Generation Rate
- **Type**: Bar Chart
- **Purpose**: Shows the rate of entropy secret generation
- **Breakdown**: By encoding type and success status

### 5. Entropy Quality Score
- **Type**: Time Series
- **Purpose**: Tracks the quality of generated entropy
- **Scale**: Higher values indicate better entropy quality

### 6. Admin Authentication Attempts
- **Type**: Bar Chart
- **Purpose**: Monitors admin authentication activity
- **Breakdown**: By success/failure status

### 7. Admin Authentication Performance
- **Type**: Time Series
- **Purpose**: Tracks authentication response times
- **Metrics**: Average authentication time

### 8. Fallback Mode Status
- **Type**: Gauge
- **Purpose**: Indicates when the system is using Node.js fallback
- **Thresholds**: Yellow (0.5) = Mixed mode, Red (1) = Full fallback

### 9. Total Secrets Generated
- **Type**: Time Series
- **Purpose**: Cumulative count of entropy secrets generated
- **Breakdown**: By encoding type and success status

## 🔍 Troubleshooting

### Common Issues

1. **No Metrics Appearing**
   - Ensure the web application is running (`npm run dev`)
   - Check that the entropy status endpoint is being called
   - Verify Prometheus is configured to scrape the metrics endpoint

2. **Dashboard Not Loading**
   - Check Grafana provisioning configuration
   - Ensure dashboard JSON is valid
   - Verify file permissions

3. **Metrics Not Updating**
   - Confirm the metrics collection interval is running
   - Check for errors in the application logs
   - Verify Prometheus can reach the metrics endpoint

### Debug Commands

```bash
# Test entropy bridge directly
npm run test:entropy

# Test monitoring system
npm run test:monitor

# Check Prometheus metrics
curl http://localhost:3002/api/prometheus

# Check entropy status
curl http://localhost:3002/api/admin/entropy-status
```

## 📋 Maintenance

### Regular Tasks

1. **Monitor Alert Thresholds**
   - Review entropy quality scores
   - Check for unusual authentication patterns
   - Monitor bridge availability

2. **Performance Optimization**
   - Analyze generation time trends
   - Optimize secret generation algorithms
   - Review fallback mode usage

3. **Security Audits**
   - Regular entropy quality assessments
   - Authentication attempt pattern analysis
   - Bridge availability monitoring

### Backup and Recovery

- Dashboard configurations are version controlled
- Metrics data is ephemeral (consider long-term storage if needed)
- Entropy bridge status is real-time (no historical persistence)

## 🤝 Contributing

When adding new entropy-related features:

1. Add appropriate Prometheus metrics
2. Update the Grafana dashboard
3. Document new metrics in this README
4. Test the monitoring system with `npm run test:monitor`

## 📞 Support

For issues with the entropy monitoring system:

1. Check the troubleshooting section above
2. Review application logs for errors
3. Verify all services are running and accessible
4. Check network connectivity between services

---

**Note**: This monitoring system provides critical visibility into the entropy generation infrastructure. Regular monitoring ensures the security and reliability of admin authentication and cryptographic operations.
