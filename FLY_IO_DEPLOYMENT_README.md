# Bitcoin Sprint Fly.io Production Deployment

This guide provides complete instructions for deploying the Bitcoin Sprint application to Fly.io for production use with real metrics monitoring.

## Architecture Overview

The deployment includes:
- **Go Backend (sprintd)**: Core blockchain processing engine
- **FastAPI Gateway**: REST API and WebSocket endpoints
- **Grafana**: Real-time metrics visualization
- **PostgreSQL**: Managed database for persistent storage

## Prerequisites

1. **Fly.io Account**: Sign up at [fly.io](https://fly.io)
2. **flyctl CLI**: Install the Fly.io CLI tool
   ```bash
   # On Linux/Mac
   curl -L https://fly.io/install.sh | sh

   # On Windows (PowerShell)
   iwr https://fly.io/install.sh -useb | iex
   ```

3. **Docker**: Ensure Docker is installed and running
4. **Git**: Version control for deployment

## Quick Start Deployment

### 1. Authenticate with Fly.io
```bash
flyctl auth login
```

### 2. Deploy the Application
```powershell
# Windows PowerShell
.\deploy-fly.ps1

# Or specify options
.\deploy-fly.ps1 -Action deploy
```

### 3. Check Deployment Status
```powershell
.\deploy-fly.ps1 -Action status
```

## Manual Deployment Steps

If you prefer manual control over the deployment process:

### 1. Prepare the Environment
```bash
# Clone or ensure you're in the project directory
cd /path/to/bitcoin-sprint

# Login to Fly.io
flyctl auth login
```

### 2. Configure Database (Optional)
```bash
# Create PostgreSQL database
flyctl postgres create --name bitcoin-sprint-db --region iad

# Attach to your app
flyctl postgres attach bitcoin-sprint-db --app bitcoin-sprint-fastapi
```

### 3. Deploy Application
```bash
# Deploy using the custom Dockerfile
flyctl deploy --dockerfile fly/fastapi/Dockerfile --remote-only
```

### 4. Verify Deployment
```bash
# Check status
flyctl status

# View logs
flyctl logs

# Get application URL
flyctl status --json | jq -r '.hostname'
```

## Application Configuration

### Environment Variables

The application uses the following environment variables:

```bash
# Application Settings
PORT=8080
ENVIRONMENT=production
LOG_LEVEL=info

# Backend Configuration
SPRINTD_PORT=9090
FASTAPI_PORT=8080
BACKEND_URL=http://localhost:9090

# Database (if using PostgreSQL)
DATABASE_URL=postgresql://user:password@host:5432/database
```

### Secrets Management

Set sensitive configuration using Fly.io secrets:

```bash
# Set database credentials
flyctl secrets set DATABASE_URL="postgresql://..."

# Set API keys
flyctl secrets set API_KEY="your-api-key"

# Set JWT secrets
flyctl secrets set JWT_SECRET="your-jwt-secret"
```

## Service Architecture

### Go Backend (sprintd)
- **Port**: 9090 (internal)
- **Health Check**: `/health`
- **Metrics**: `/metrics`
- **API**: `/api/v1/*`

### FastAPI Gateway
- **Port**: 8080 (external)
- **Health Check**: `/health`
- **API Documentation**: `/docs`
- **WebSocket**: `/ws/*`

### Grafana Dashboard
- **Port**: 3000
- **Admin User**: admin
- **Default Password**: admin (change on first login)

## Monitoring and Observability

### Application Metrics
- Real-time blockchain metrics
- API response times
- Error rates and success rates
- Resource utilization

### Health Checks
```bash
# Check application health
curl https://your-app.fly.dev/health

# Check backend status
curl https://your-app.fly.dev/api/v1/backend/status
```

### Logs
```bash
# View application logs
flyctl logs

# View specific service logs
flyctl logs --instance <instance-id>
```

## Scaling and Performance

### Horizontal Scaling
```bash
# Scale to multiple instances
flyctl scale count 3

# Scale based on CPU usage
flyctl autoscale standard
```

### Resource Allocation
```bash
# Set VM size
flyctl scale vm shared-cpu-2x

# Set memory limits
flyctl scale memory 1024mb
```

## Database Management

### PostgreSQL Operations
```bash
# Connect to database
flyctl postgres connect --app bitcoin-sprint-db

# Run migrations
flyctl ssh console --command "python manage.py migrate"

# Backup database
flyctl postgres create-backup
```

### Database Monitoring
```bash
# Check database status
flyctl postgres status

# View database logs
flyctl postgres logs
```

## Troubleshooting

### Common Issues

1. **Deployment Fails**
   ```bash
   # Check build logs
   flyctl logs --build

   # Validate Dockerfile
   docker build -f fly/fastapi/Dockerfile .
   ```

2. **Application Not Starting**
   ```bash
   # Check application logs
   flyctl logs

   # SSH into the instance
   flyctl ssh console

   # Check running processes
   ps aux
   ```

3. **Database Connection Issues**
   ```bash
   # Test database connection
   flyctl postgres connect

   # Check database credentials
   flyctl secrets list
   ```

### Health Checks

```bash
# Manual health check
curl -f https://your-app.fly.dev/health

# Check backend connectivity
curl -f https://your-app.fly.dev/api/v1/backend/status
```

## Backup and Recovery

### Application Backups
```bash
# Create backup
flyctl volumes snapshots create vol_xxx

# List backups
flyctl volumes snapshots list
```

### Database Backups
```bash
# Create database backup
flyctl postgres create-backup

# List database backups
flyctl postgres list-backups
```

## Security Considerations

1. **Network Security**
   - All traffic uses HTTPS/TLS
   - Internal services communicate over private networking
   - Database access is restricted to application instances

2. **Secret Management**
   - Use Fly.io secrets for sensitive data
   - Rotate secrets regularly
   - Never commit secrets to version control

3. **Access Control**
   - Configure proper authentication
   - Use role-based access control
   - Monitor access logs

## Cost Optimization

### Resource Optimization
```bash
# Use shared CPU for development
flyctl scale vm shared-cpu-1x

# Scale down during off-hours
flyctl scale count 1

# Use spot instances for non-critical workloads
flyctl scale vm performance-1x --spot
```

### Monitoring Costs
```bash
# Check current costs
flyctl costs

# Set spending limits
flyctl billing set-limit 50
```

## Development Workflow

### Local Development
```bash
# Run locally with Docker
docker-compose up

# Run with hot reload
docker-compose -f docker-compose.dev.yml up
```

### CI/CD Integration
```yaml
# .github/workflows/deploy.yml
name: Deploy to Fly.io
on:
  push:
    branches: [main]
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: superfly/flyctl-actions/setup-flyctl@master
      - run: flyctl deploy --remote-only
        env:
          FLY_API_TOKEN: ${{ secrets.FLY_API_TOKEN }}
```

## Support and Resources

- **Fly.io Documentation**: https://fly.io/docs
- **Bitcoin Sprint Docs**: See `docs/` directory
- **Community Support**: GitHub Issues
- **Status Page**: https://status.fly.io

## Production Checklist

- [ ] Environment variables configured
- [ ] Secrets properly set
- [ ] Database attached and configured
- [ ] Health checks passing
- [ ] Monitoring and alerting configured
- [ ] SSL/TLS certificates valid
- [ ] Backup strategy implemented
- [ ] Scaling rules configured
- [ ] Security policies applied
- [ ] Performance benchmarks completed

---

**Note**: This deployment configuration is optimized for production use with real metrics collection and monitoring. Ensure all security best practices are followed before deploying to production.
