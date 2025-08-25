# Bitcoin Sprint Maintenance System

## Overview

The Bitcoin Sprint maintenance system provides comprehensive tools for managing system updates, maintenance windows, and health monitoring. It includes both programmatic APIs and command-line tools for operational management.

## Features

- **Update State Management**: Track system versions and rollback capabilities
- **Maintenance Mode**: Graceful service degradation during maintenance windows
- **Health Monitoring**: Comprehensive system health checks
- **CLI Tools**: Command-line interface for operational tasks
- **API Endpoints**: RESTful APIs for integration with monitoring systems
- **Automatic Middleware**: Request interception during maintenance mode

## Architecture

### Core Components

1. **`lib/updateState.ts`** - Update state management with caching and validation
2. **`lib/maintenance.ts`** - Maintenance operations and health checks
3. **`middleware.ts`** - Request interception for maintenance mode
4. **`scripts/maintenance.js`** - CLI tool for operations
5. **API endpoints** - RESTful interfaces for external integration

### Data Storage

- **`data/update_state.json`** - Current system version and update history
- **`data/maintenance.json`** - Maintenance mode configuration (created when enabled)

## API Endpoints

### GET `/api/update-state`
Get current system update state.

**Response:**
```json
{
  "ok": true,
  "cached": true,
  "version": "v1.0.0",
  "last_updated": "2025-08-25T10:30:00.000Z",
  "rollback": false
}
```

### GET `/api/health`
Enhanced health check with maintenance awareness.

**Response (Healthy):**
```json
{
  "ok": true,
  "service": "web",
  "timestamp": 1693486200000,
  "status": "healthy",
  "checks": {
    "maintenance": { "status": "pass", "message": "Not in maintenance mode" },
    "state_file": { "status": "pass", "message": "State file accessible" },
    "data_directory": { "status": "pass", "message": "Data directory accessible" },
    "filesystem": { "status": "pass", "message": "Filesystem accessible" }
  }
}
```

**Response (Maintenance):**
```json
{
  "ok": false,
  "service": "web",
  "timestamp": 1693486200000,
  "status": "maintenance",
  "checks": {
    "maintenance": { "status": "fail", "message": "System maintenance in progress" }
  }
}
```

### GET `/api/maintenance`
Get maintenance status.

**Response:**
```json
{
  "ok": true,
  "maintenance": {
    "enabled": true,
    "reason": "Deploying new features",
    "started_at": "2025-08-25T10:30:00.000Z",
    "estimated_duration": "30 minutes"
  }
}
```

### POST `/api/maintenance`
Enable maintenance mode or update system state.

**Enable Maintenance:**
```json
{
  "action": "enable_maintenance",
  "reason": "Deploying new features"
}
```

**Update System State:**
```json
{
  "action": "update_state",
  "version": "v1.2.0",
  "rollback": false
}
```

### DELETE `/api/maintenance`
Disable maintenance mode.

**Response:**
```json
{
  "ok": true,
  "message": "Maintenance mode disabled",
  "maintenance": { "enabled": false }
}
```

## CLI Commands

### Basic Commands

```bash
# Check maintenance status
npm run maintenance:status

# Enable maintenance mode
npm run maintenance:enable "System update in progress"

# Disable maintenance mode
npm run maintenance:disable

# Perform health check
npm run maintenance:health

# Update system version
node scripts/maintenance.js update v1.2.0

# Rollback to previous version
node scripts/maintenance.js update v1.1.0 --rollback
```

### Advanced CLI Usage

```bash
# Full maintenance workflow
node scripts/maintenance.js enable "Deploying critical security updates"
# ... perform deployment ...
node scripts/maintenance.js update v1.2.1
node scripts/maintenance.js disable

# Health monitoring
node scripts/maintenance.js health
```

## Programmatic Usage

### Update State Management

```typescript
import { getUpdateState, UpdateStateSchema } from '@/lib/updateState';

// Get current state
const state = await getUpdateState();
console.log(`Current version: ${state.version}`);

// Validate state data
const validation = UpdateStateSchema.safeParse(data);
if (validation.success) {
  // State is valid
}
```

### Maintenance Operations

```typescript
import {
  createMaintenanceMode,
  disableMaintenanceMode,
  getMaintenanceStatus,
  updateSystemState,
  performSystemHealthCheck
} from '@/lib/maintenance';

// Enable maintenance
await createMaintenanceMode('Scheduled update');

// Update system version
await updateSystemState('v1.2.0', false);

// Check health
const health = await performSystemHealthCheck();

// Disable maintenance
await disableMaintenanceMode();
```

## Configuration

### Environment Variables

```bash
# Update state file location (optional)
SPRINT_STATE_FILE=/var/lib/sprint/update_state.json

# Cache TTL for update state (seconds)
UPDATE_CACHE_TTL=300

# Logging level
LOG_LEVEL=info
```

### Default Paths

- **State file**: `{cwd}/data/update_state.json`
- **Maintenance file**: `{cwd}/data/maintenance.json`
- **Data directory**: `{cwd}/data/`

## Security

### Path Validation
- State files must be within the current working directory or `/var/lib/sprint`
- Prevents directory traversal attacks
- Validates file paths before operations

### Access Control
- API endpoints can be protected with authentication middleware
- CLI tools require filesystem access to data directory
- Environment-based configuration for production security

## Monitoring Integration

### Prometheus Metrics
The system provides structured data suitable for Prometheus monitoring:

```typescript
// Health check endpoint returns structured data
const health = await fetch('/api/health');
// Use health.status for alerting rules
```

### Log Integration
Structured logging with Pino for centralized log analysis:

```json
{
  "level": "info",
  "service": "sprint-update-state",
  "version": "v1.2.0",
  "msg": "Successfully retrieved update state"
}
```

## Maintenance Workflow

### Planned Maintenance

1. **Pre-maintenance**:
   ```bash
   npm run maintenance:enable "Scheduled system update"
   ```

2. **Perform updates**:
   ```bash
   # Deploy new version
   # Update configurations
   ```

3. **Update state**:
   ```bash
   node scripts/maintenance.js update v1.2.0
   ```

4. **Health check**:
   ```bash
   npm run maintenance:health
   ```

5. **Exit maintenance**:
   ```bash
   npm run maintenance:disable
   ```

### Emergency Rollback

1. **Enable maintenance**:
   ```bash
   npm run maintenance:enable "Emergency rollback in progress"
   ```

2. **Rollback**:
   ```bash
   node scripts/maintenance.js update v1.1.0 --rollback
   ```

3. **Verify health**:
   ```bash
   npm run maintenance:health
   ```

4. **Exit maintenance**:
   ```bash
   npm run maintenance:disable
   ```

## Integration with Bitcoin Sprint

### Middleware Integration
The maintenance system automatically intercepts requests:

- **Web requests**: Redirected to maintenance page
- **API requests**: Return 503 with maintenance information
- **Health endpoints**: Always accessible for monitoring

### Service Integration
The maintenance system integrates with the main Bitcoin Sprint service:

```go
// In Go service, check maintenance status
curl http://localhost:3000/api/health
```

## Troubleshooting

### Common Issues

**State file not found**:
```bash
# Create initial state file
node scripts/maintenance.js update v1.0.0
```

**Permission denied**:
```bash
# Ensure data directory is writable
chmod 755 data/
```

**Cache issues**:
```bash
# Clear cache by restarting service or waiting for TTL expiry
```

### Debugging

Enable debug logging:
```bash
LOG_LEVEL=debug node scripts/maintenance.js health
```

Check file permissions:
```bash
ls -la data/
```

Validate JSON format:
```bash
cat data/update_state.json | jq .
```

## Best Practices

1. **Always enable maintenance mode before updates**
2. **Update system state after successful deployments**
3. **Perform health checks after maintenance**
4. **Use descriptive maintenance reasons**
5. **Monitor health endpoints continuously**
6. **Test rollback procedures regularly**
7. **Keep maintenance windows short**
8. **Document maintenance procedures**

## Production Deployment

### Required Setup

1. **Create data directory**:
   ```bash
   mkdir -p /var/lib/sprint
   chown app:app /var/lib/sprint
   ```

2. **Set environment variables**:
   ```bash
   export SPRINT_STATE_FILE=/var/lib/sprint/update_state.json
   export LOG_LEVEL=info
   ```

3. **Initialize state**:
   ```bash
   node scripts/maintenance.js update v1.0.0
   ```

4. **Configure monitoring**:
   - Set up health check alerts
   - Monitor maintenance API endpoints
   - Log aggregation for maintenance events

This maintenance system provides comprehensive tools for managing Bitcoin Sprint deployments and ensuring smooth operational procedures with minimal downtime.
