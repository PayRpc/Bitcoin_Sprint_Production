# Bitcoin Sprint Deployment Guide

This guide covers the deployment of the Bitcoin Sprint Go application to Fly.io.

## Prerequisites

### Required Software
- Go 1.19 or higher
- Docker
- Fly.io CLI (`flyctl`)
- Git

### Installation
```bash
# Install Fly.io CLI
curl -L https://fly.io/install.sh | sh

# Login to Fly.io
flyctl auth login
```

## Project Structure

```
bitcoin-sprint/
├── cmd/sprintd/           # Main application
│   └── main.go
├── internal/              # Internal packages
├── config/                # Configuration files
├── Dockerfile             # Docker configuration
├── fly.toml              # Fly.io configuration
├── go.mod                 # Go modules
├── go.sum                 # Go dependencies
├── init-db.sql           # Database schema
└── deploy-fly.sh         # Deployment script
```

## Deployment Steps

### 1. Validation
Before deploying, validate your configuration:

**Windows (PowerShell):**
```powershell
.\validate-deployment.ps1
```

**Linux/macOS:**
```bash
./validate-deployment.sh
```

### 2. Build the Application
```bash
# Build optimized binary
go build -ldflags="-s -w" -o sprintd ./cmd/sprintd

# Or use the build task
go run -tags=build . build
```

### 3. Deploy to Fly.io
```bash
# Using the deployment script
./deploy-fly.sh

# Or deploy directly
flyctl deploy
```

## Configuration Files

### fly.toml
The Fly.io configuration includes:
- Docker build settings
- Service configuration (ports 80, 443)
- Health checks
- Database attachment
- VM sizing

### Dockerfile
Multi-stage Docker build:
- Build stage: Compiles Go application
- Runtime stage: Minimal Alpine Linux with binary
- Exposes ports: 8080, 8081, 9090, 6060

## Database Setup

The deployment script will prompt you to create a PostgreSQL database:

1. Choose to attach a database when prompted
2. The script creates a managed PostgreSQL instance
3. Database connection string is automatically configured
4. Schema is applied from `init-db.sql`

## Environment Variables

### Required
- `DATABASE_URL`: PostgreSQL connection string
- `DATABASE_TYPE`: Set to "postgres"

### Optional
- `SPRINT_TIER`: "enterprise" or "free"
- `SPRINT_API_HOST`: API bind address (default: 0.0.0.0)
- `SPRINT_API_PORT`: API port (default: 8080)
- `SPRINT_ADMIN_PORT`: Admin port (default: 8081)
- `SPRINT_METRICS_PORT`: Metrics port (default: 9090)
- `SPRINT_PPROF_PORT`: Profiling port (default: 6060)
- `SPRINT_LICENSE_KEY`: Enterprise license key
- `SPRINT_LOG_LEVEL`: Log level (default: info)

## Health Checks

The application includes health check endpoints:
- `/health`: General health status
- `/version`: API version information

## Monitoring

### Logs
```bash
# View application logs
flyctl logs

# View database logs
flyctl postgres logs
```

### Metrics
- Application metrics available on port 9090
- Prometheus-compatible metrics endpoint

### Scaling
```bash
# Scale to multiple instances
flyctl scale count 2

# View current scaling
flyctl scale show
```

## Troubleshooting

### Common Issues

1. **Build Failures**
   - Ensure Go 1.19+ is installed
   - Run `go mod tidy` to clean dependencies
   - Check for missing CGO dependencies

2. **Database Connection Issues**
   - Verify DATABASE_URL is set correctly
   - Check database firewall rules
   - Ensure PostgreSQL extensions are available

3. **Port Conflicts**
   - Verify ports 8080, 8081, 9090, 6060 are available
   - Check Fly.io port configuration

4. **Memory Issues**
   - Monitor memory usage with `flyctl monitor`
   - Adjust VM size in fly.toml if needed

### Debug Commands
```bash
# SSH into running instance
flyctl ssh console

# Check application status
flyctl status

# View environment variables
flyctl secrets list

# Restart application
flyctl restart
```

## Security Considerations

1. **Secrets Management**
   - Use `flyctl secrets set` for sensitive data
   - Never commit secrets to version control
   - Rotate secrets regularly

2. **Network Security**
   - HTTPS is enforced by default
   - Database connections use SSL
   - Internal ports are not exposed externally

3. **Access Control**
   - Configure proper authentication
   - Use enterprise license for production
   - Implement rate limiting

## Performance Optimization

1. **Build Optimization**
   - Use build tags for optimization
   - Enable CGO for Rust integration
   - Strip debug information

2. **Runtime Optimization**
   - Configure appropriate VM size
   - Enable connection pooling
   - Use enterprise features for high throughput

3. **Monitoring**
   - Set up proper logging levels
   - Monitor resource usage
   - Configure alerts for critical metrics

## Backup and Recovery

1. **Database Backups**
   - Fly.io automatically backs up PostgreSQL
   - Export data regularly for additional safety

2. **Application Backups**
   - Keep deployment scripts versioned
   - Document configuration changes

3. **Disaster Recovery**
   - Test deployment process regularly
   - Have rollback procedures ready
   - Monitor application health continuously

## Support

For issues or questions:
1. Check the logs: `flyctl logs`
2. Review this documentation
3. Check Fly.io status: https://status.fly.io/
4. Contact the development team

## Version Information

- Application Version: Check `/version` endpoint
- Go Version: 1.19+
- Database: PostgreSQL 13+
- Platform: Fly.io
