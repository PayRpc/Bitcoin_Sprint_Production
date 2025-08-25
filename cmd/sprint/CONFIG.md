# Configuration Guide

Bitcoin Sprint supports multiple configuration formats with the following precedence (highest to lowest):

1. **Environment variables** - Direct env vars (highest priority)
2. **.env.local** - Local environment file (gitignored)
3. **.env** - Environment file template
4. **config.json** - JSON configuration with ${VAR} placeholders
5. **Built-in defaults** - Fallback values

## Configuration Sources

### Environment Variables (Recommended for Production)
Set these environment variables directly:
```bash
export LICENSE_KEY="your_license_key"
export RPC_USER="your_username" 
export RPC_PASS="your_password"
export PEER_SECRET="your_secret"
export RPC_NODES="https://node1.com:8332,https://node2.com:8332"
export TURBO_MODE="true"
```

### .env Files (Recommended for Development)
1. Copy `.env.example` to `.env.local`
2. Edit `.env.local` with your actual values
3. The service will automatically load from `.env.local` first, then `.env`

### JSON Configuration with Placeholders
Use `config.json` with environment variable placeholders:
```json
{
  "license_key": "${LICENSE_KEY}",
  "rpc_user": "${RPC_USER}",
  "rpc_pass": "${RPC_PASS}",
  "peer_secret": "${PEER_SECRET}",
  "rpc_nodes": ["${RPC_NODE_1}", "${RPC_NODE_2}"]
}
```

## Security Notes

- Never commit `.env.local` to version control
- Use placeholder format in `config.json` for shared configurations  
- Sensitive values are automatically masked in logs
- SecureBuffer is used for in-memory protection of secrets

## Configuration Logging

On startup, Bitcoin Sprint logs:
- Configuration source used (env vars, .env.local, .env, config.json, defaults)
- Safe summary with masked sensitive fields
- Validation results

Example startup log:
```
INFO Configuration loaded source=.env.local
INFO Config summary tier=pro turbo_mode=true license_key=abcd****wxyz
```
