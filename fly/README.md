# 🚀 Bitcoin Sprint - Fly.io Production Deployment

This directory contains the production deployment configuration for Bitcoin Sprint on Fly.io, featuring separate containerized services for maximum scalability and maintainability.

## 🏗️ Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   FastAPI       │    │    Grafana      │    │  PostgreSQL     │
│   Backend       │◄──►│   Monitoring    │    │   (Managed)     │
│                 │    │                 │    │                 │
│ • REST API      │    │ • Dashboards    │    │ • Fly Managed   │
│ • WebSocket     │    │ • Metrics       │    │ • Auto-scaling  │
│ • Database      │    │ • Alerts        │    │ • Backups       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 📦 Services

### 1. FastAPI Backend (`fly/fastapi/`)
- **Purpose**: Main application backend with REST API and WebSocket support
- **Port**: 8080 (internal), 80/443 (external)
- **Features**:
  - Multi-chain blockchain integration
  - Real-time metrics collection
  - PostgreSQL database connection
  - Redis caching support
  - Prometheus metrics endpoint

### 2. Grafana Monitoring (`fly/grafana/`)
- **Purpose**: Visualization and monitoring dashboard
- **Port**: 8080 (internal), 80/443 (external)
- **Features**:
  - Pre-configured dashboards
  - Prometheus data source
  - Real-time metrics visualization
  - Alert management

### 3. PostgreSQL Database
- **Managed by Fly.io**
- **Features**:
  - Automatic scaling
  - Daily backups
  - High availability
  - Connection pooling

## 🚀 Deployment Instructions

### Prerequisites
1. Install Fly.io CLI: `curl -L https://fly.io/install.sh | sh`
2. Login to Fly.io: `flyctl auth login`
3. Create a Fly.io organization (if not exists)

### Step 1: Set Environment Variables
```bash
# Set your Fly.io app names
export FASTAPI_APP_NAME="bitcoin-sprint-fastapi"
export GRAFANA_APP_NAME="bitcoin-sprint-grafana"

# Database credentials (set these in Fly.io secrets)
export FLY_PG_USER="your_db_user"
export FLY_PG_PASSWORD="your_secure_password"
export FLY_PG_HOST="your_db_host"
export FLY_PG_DBNAME="bitcoin_sprint"
```

### Step 2: Deploy Services
```bash
# Make deployment script executable
chmod +x fly/deploy.sh

# Run deployment
./fly/deploy.sh
```

### Step 3: Configure Database Connection
```bash
# Set database secrets in FastAPI app
flyctl secrets set DATABASE_URL="postgresql://$FLY_PG_USER:$FLY_PG_PASSWORD@$FLY_PG_HOST:5432/$FLY_PG_DBNAME" --app $FASTAPI_APP_NAME

# Set Grafana admin password
flyctl secrets set GRAFANA_ADMIN_PASSWORD="your_secure_grafana_password" --app $GRAFANA_APP_NAME
```

### Step 4: Update Grafana Data Sources
After deployment, update the Grafana data source URL to match your actual FastAPI service URL.

## 🔧 Configuration Files

### FastAPI Service
- `fly/fastapi/Dockerfile` - Optimized container for FastAPI
- `fly.toml` - Fly.io configuration for FastAPI
- `requirements.txt` - Python dependencies

### Grafana Service
- `fly/grafana/Dockerfile` - Optimized container for Grafana
- `fly/grafana/fly.toml` - Fly.io configuration for Grafana
- `grafana/provisioning/datasources/prometheus.yml` - Data source config
- `grafana/provisioning/dashboards/dashboard.yml` - Dashboard provisioning
- `grafana/dashboards/bitcoin-sprint-dashboard.json` - Sample dashboard

### Database
- `fly/postgres-config.env` - PostgreSQL connection configuration

## 📊 Monitoring & Metrics

### Health Checks
- FastAPI: `GET /health` - Application health
- FastAPI: `GET /metrics` - Prometheus metrics
- Grafana: `GET /api/health` - Grafana health

### Key Metrics to Monitor
- API response times
- Database connection pool usage
- Blockchain node connectivity
- System resource utilization
- Error rates and alerts

## 🔒 Security Considerations

### Environment Variables
- Store sensitive data as Fly.io secrets
- Never commit secrets to version control
- Use strong, unique passwords

### Network Security
- All services use HTTPS by default
- Internal service communication is secure
- Database connections use SSL

### Access Control
- Grafana admin password is secured
- API endpoints can be protected with authentication
- Database access is restricted to application services

## 🚨 Troubleshooting

### Common Issues

1. **Database Connection Failed**
   ```bash
   # Check database secrets
   flyctl secrets list --app $FASTAPI_APP_NAME
   ```

2. **Grafana Data Source Error**
   - Update the data source URL in Grafana UI
   - Ensure FastAPI service is running and accessible

3. **Build Failures**
   ```bash
   # Check build logs
   flyctl logs --app $APP_NAME
   ```

### Logs & Debugging
```bash
# View application logs
flyctl logs --app $FASTAPI_APP_NAME

# SSH into running instance
flyctl ssh console --app $FASTAPI_APP_NAME

# Check service status
flyctl status --app $APP_NAME
```

## 📈 Scaling

### Horizontal Scaling
```bash
# Scale FastAPI service
flyctl scale count 3 --app $FASTAPI_APP_NAME

# Scale Grafana (usually 1 instance is sufficient)
flyctl scale count 1 --app $GRAFANA_APP_NAME
```

### Vertical Scaling
```bash
# Increase CPU/memory
flyctl scale vm shared-cpu-2x --app $FASTAPI_APP_NAME
```

## 🔄 Updates & Maintenance

### Deploying Updates
```bash
# Deploy FastAPI updates
flyctl deploy --app $FASTAPI_APP_NAME

# Deploy Grafana updates
flyctl deploy --app $GRAFANA_APP_NAME
```

### Database Migrations
```bash
# Run database migrations via FastAPI
flyctl ssh console --app $FASTAPI_APP_NAME --command "alembic upgrade head"
```

## 📞 Support

For issues with Fly.io deployment:
- Check Fly.io status: https://status.fly.io
- Review Fly.io documentation: https://fly.io/docs
- Check application logs using `flyctl logs`

---

**🎯 Your Bitcoin Sprint production environment is now ready for deployment on Fly.io!**
