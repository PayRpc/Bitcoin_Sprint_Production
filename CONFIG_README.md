# Bitcoin Sprint Configuration

## Environment Variables

### Address Configuration
- `API_ADDR`: Combined API host and port (e.g., `127.0.0.1:8080` or `0.0.0.0:8080`)
  - Overrides `API_HOST` and `API_PORT` if set
- `METRICS_ADDR`: Combined metrics host and port (e.g., `127.0.0.1:9090`)
  - Overrides `PROMETHEUS_PORT` if set

### Legacy Individual Settings
- `API_HOST`: API server host (default: `0.0.0.0`)
- `API_PORT`: API server port (default: `8080`)
- `PROMETHEUS_PORT`: Metrics server port (default: `9090`)

## Port Management

Use the included `check-ports.ps1` script to verify port availability:

```powershell
# Check if ports are available
.\check-ports.ps1

# Kill processes using the ports and check again
.\check-ports.ps1 -Kill

# Quiet mode (only show errors)
.\check-ports.ps1 -Quiet
```

## Startup Banner

When Bitcoin Sprint starts successfully, you'll see:

```
Bitcoin Sprint starting…
 API:      http://0.0.0.0:8080
 Metrics:  http://127.0.0.1:9090/metrics
 PProf:    disabled
 P2P:      enabled (min proto 70016, witness only)
 Workers:  16
```

## Examples

### Using combined address variables
```bash
export API_ADDR="127.0.0.1:8080"
export METRICS_ADDR="127.0.0.1:9090"
./bitcoin-sprint.exe
```

### Using individual settings
```bash
export API_HOST="127.0.0.1"
export API_PORT="8080"
export PROMETHEUS_PORT="9090"
./bitcoin-sprint.exe
```

### PowerShell
```powershell
$env:API_ADDR="127.0.0.1:8080"
$env:METRICS_ADDR="127.0.0.1:9090"
.\bitcoin-sprint.exe
```
