# Bitcoin Sprint Dashboard Integration

## Overview
The Bitcoin Sprint Dashboard provides real-time monitoring of the Bitcoin network with integration into the Next.js web application.

## Features

### 🔗 **Real Bitcoin Network Data**
- Live block data from Mempool.space and Blockstream APIs
- Real-time transaction counts and network statistics
- Block processing times and network health metrics

### 📊 **Internal System Metrics**
- CPU and memory usage from internal monitoring
- API usage statistics and tier information
- System health and performance indicators

### 🔄 **Dual Monitoring System**
- **External**: Real Bitcoin blockchain data (independent of internal systems)
- **Internal**: System metrics from your Docker/Grafana stack
- **Redundant**: Works even if internal monitoring fails

## Access Points

### Web Application
- **URL**: `/dashboard`
- **Navigation**: "Live Dashboard" button on homepage
- **Authentication**: None required (public dashboard)

### Direct Access
- **File**: `web/dashboard.html`
- **Standalone**: Can be served independently

## API Integration

### External APIs Used
- `https://mempool.space/api/v1/blocks` - Live Bitcoin block data
- `https://blockstream.info/api/blocks` - Backup Bitcoin API
- `https://api.blockchain.com/v3/exchange/tickers/BTC-USD` - Price data

### Internal API
- `/api/metrics` - Internal system metrics proxy
- Requires authentication for detailed metrics
- Provides tier-specific performance data

## Architecture

```
┌─────────────────┐    ┌──────────────────┐
│   Next.js App   │────│  Dashboard Page  │
│                 │    │  (/dashboard)    │
└─────────────────┘    └──────────────────┘
         │                       │
         │                       │
    ┌────▼───────────────────────▼────┐
    │                               │
    │    Bitcoin Sprint Dashboard   │
    │                               │
    └────▲───────────────────────▲────┘
         │                       │
         │                       │
┌────────▼────────┐    ┌─────────▼─────────┐
│ External APIs   │    │ Internal Metrics  │
│ (Mempool, etc.) │    │ (/api/metrics)    │
└─────────────────┘    └───────────────────┘
```

## Benefits

### For Users
- **Real-time Bitcoin data** without running full nodes
- **Network health monitoring** for trading decisions
- **Performance insights** for API usage optimization

### For Operations
- **Backup monitoring** when Grafana is unavailable
- **External validation** of network conditions
- **Public dashboard** for stakeholders

### For Development
- **Independent monitoring** of external dependencies
- **API redundancy testing** with multiple Bitcoin APIs
- **Performance benchmarking** against real network data

## Configuration

### Environment Variables
```bash
# For production deployment
NEXT_PUBLIC_DASHBOARD_API_BASE=https://your-domain.com
```

### Docker Integration
The dashboard can be served alongside your existing Docker stack:

```yaml
# Add to docker-compose.yml
dashboard:
  build: ./web
  ports:
    - "3000:3000"
  environment:
    - NODE_ENV=production
```

## Usage Examples

### Basic Access
```javascript
// Access dashboard
window.location.href = '/dashboard';

// Manual refresh
refreshAllWithInternal();
```

### API Usage
```javascript
// Get internal metrics
const metrics = await fetch('/api/metrics');
const data = await metrics.json();

// Get Bitcoin data
const blocks = await fetch('https://mempool.space/api/v1/blocks');
const blockData = await blocks.json();
```

## Security Considerations

- Dashboard serves public Bitcoin data (no sensitive information)
- Internal metrics API requires authentication
- CORS configured for external API access
- No credentials exposed in client-side code

## Future Enhancements

- [ ] Add historical data charts
- [ ] Implement alerting system
- [ ] Add custom metric dashboards
- [ ] Integrate with WebSocket for real-time updates
- [ ] Add export functionality for reports
