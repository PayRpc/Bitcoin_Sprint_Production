# Bitcoin Sprint Port Mapping Configuration
# Permanent solution for port conflicts between Docker and local services

## Current Port Assignments (Conflict-Free)

### Docker Services (External Ports)
- **Grafana**: 3000 (unchanged - no local conflict)
- **Bitcoin Sprint Go API**: 8082 → 8080 (internal)
- **Bitcoin Sprint Admin API**: 8083 → 8081 (internal)
- **Bitcoin Sprint Metrics**: 9091 → 9090 (internal)
- **Bitcoin Sprint pprof**: 6061 → 6060 (internal)
- **Bitcoin Sprint Rust API**: 8443 (unchanged - no local conflict)
- **Bitcoin Sprint Rust Admin**: 8444 (unchanged - no local conflict)
- **Bitcoin Sprint Rust Metrics**: 9092 (unchanged - no local conflict)
- **PostgreSQL**: 5433 → 5432 (internal) - CHANGED from 5432 to avoid local conflict
- **Redis**: 6380 → 6379 (internal) - CHANGED from 6379 to avoid local conflict
- **Prometheus**: 9091 (unchanged - no local conflict)

### Local Services (Direct Ports)
- **Go Sprintd Service**: 8080 (when run directly)
- **Local PostgreSQL**: 5432
- **Local Redis**: 6379 (if running)
- **Next.js Dashboard**: 3002
- **FastAPI Gateway**: 8000

## Service URLs

### Docker Services
- Grafana: http://localhost:3000
- Bitcoin Sprint API: http://localhost:8082
- Bitcoin Sprint Admin: http://localhost:8083
- Bitcoin Sprint Metrics: http://localhost:9091
- PostgreSQL: localhost:5433
- Redis: localhost:6380
- Prometheus: http://localhost:9091

### Local Services
- Go Sprintd: http://localhost:8080
- Local PostgreSQL: localhost:5432
- Local Redis: localhost:6379
- Next.js Dashboard: http://localhost:3002
- FastAPI Gateway: http://localhost:8000

## Environment Variables for Local Development

When running services locally (not in Docker), use these ports:

```bash
# Go Service
export API_PORT=8080

# PostgreSQL (local)
export POSTGRES_URL=postgres://user:password@localhost:5432/dbname

# Redis (local)
export REDIS_URL=redis://localhost:6379

# Next.js
export PORT=3002

# FastAPI
export API_PORT=8000
```

## Docker Environment Variables

When running in Docker, services use internal networking:

```bash
# PostgreSQL (Docker internal)
POSTGRES_URL=postgres://sprint:sprint@postgres:5432/sprint_db

# Redis (Docker internal)
REDIS_URL=redis://redis:6379
```

## Conflict Resolution Strategy

1. **Docker services use offset ports**: External ports are offset by +1, +2, etc. to avoid local service conflicts
2. **Internal Docker networking**: Services communicate using internal hostnames and internal ports
3. **Local services use standard ports**: When running services directly, they use their standard/default ports
4. **Environment-driven configuration**: Services read ports from environment variables for flexibility

## Management Commands

### Start Docker Services
```bash
cd config
docker-compose up -d
```

### Stop Conflicting Local Services
```bash
# Stop local PostgreSQL if using Docker PostgreSQL
sudo systemctl stop postgresql

# Stop local Redis if using Docker Redis
sudo systemctl stop redis-server
```

### Check Port Usage
```bash
netstat -ano | findstr LISTENING
```

### Verify Docker Services
```bash
docker ps
docker-compose ps
```

## Testing

After applying these changes:

1. Stop any local PostgreSQL/Redis services that conflict
2. Start Docker services: `docker-compose up -d`
3. Verify services are accessible on their new ports
4. Test local services on their standard ports without conflicts

## Future Port Assignments

When adding new services, follow this pattern:
- Check for existing port usage: `netstat -ano | findstr :PORT`
- Use offset ports for Docker services (+1, +2, +100, etc.)
- Document new assignments in this file
- Update environment variables accordingly
