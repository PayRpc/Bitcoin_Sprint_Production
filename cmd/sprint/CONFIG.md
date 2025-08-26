# Configuration Guide

Bitcoin Sprint supports multiple configuration formats with the following precedence (highest to lowest):

1. **Environment variables** - Direct env vars (highest priority)
2. **.env.local** - Local environment file (gitignored)
3. **.env** - Environment file template
4. **config.json** - JSON configuration with ${VAR} placeholders
5. **Built-in defaults** - Fallback values

## Bitcoin Core Integration

Bitcoin Sprint is now fully integrated with Bitcoin Core using standard ports:
- **Bitcoin RPC**: Port 8332 (standard Bitcoin Core RPC port)
- **Bitcoin Sprint API**: Port 8080 (standard HTTP alternative)
- **Peer Network**: Port 8335 (Sprint peer mesh networking)

## Configuration Sources

### Environment Variables (Recommended for Production)
Set these environment variables directly:
```bash
export LICENSE_KEY="DEMO_LICENSE_BYPASS"
export RPC_USER="test_user" 
export RPC_PASS="strong_random_password_here"
export PEER_SECRET="demo_peer_secret_123"
export RPC_NODES="http://localhost:8332"
export TURBO_MODE="false"
export TIER="enterprise"
export API_PORT="8080"
export PEER_LISTEN_PORT="8335"
```

### .env Files (Recommended for Development)
1. Copy `.env.example` to `.env.local`
2. Edit `.env.local` with your actual values
3. The service will automatically load from `.env.local` first, then `.env`

### JSON Configuration with Unified Credentials
All JSON config files now use unified credentials:
```json
{
  "license_key": "DEMO_LICENSE_BYPASS",
  "tier": "enterprise",
  "rpc_user": "test_user",
  "rpc_pass": "strong_random_password_here",
  "peer_secret": "demo_peer_secret_123",
  "rpc_nodes": ["http://localhost:8332"],
  "dashboard_port": 8080,
  "peer_listen_port": 8335
}
```

## Bitcoin Core Setup

Ensure your `bitcoin.conf` includes:
```ini
# Bitcoin Core Configuration for Bitcoin Sprint Integration
server=1
rpcuser=test_user
rpcpassword=strong_random_password_here
rpcport=8332
rpcbind=127.0.0.1
rpcallowip=127.0.0.1
txindex=1
disablewallet=1
```

## Security Notes

- Never commit `.env.local` to version control
- Use placeholder format in `config.json` for shared configurations  
- Sensitive values are automatically masked in logs
- SecureBuffer is used for in-memory protection of secrets
- Bitcoin RPC is restricted to localhost only for security

## Configuration Logging

On startup, Bitcoin Sprint logs:
- Configuration source used (env vars, .env.local, .env, config.json, defaults)
- Safe summary with masked sensitive fields
- Validation results
- Bitcoin Core connection status

Example startup log:
```
INFO Configuration loaded source=.env.local
INFO Config summary tier=enterprise turbo_mode=false license_key=DEMO****PASS rpc_port=8332 api_port=8080
INFO Bitcoin Core connection status=connected rpc_host=localhost:8332
```
