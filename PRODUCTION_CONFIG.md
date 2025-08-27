# Bitcoin Sprint Production Configuration Guide

## Security Best Practices

### RPC Authentication
**Important:** Use `rpcauth` instead of `rpcuser`/`rpcpassword` for production deployments.

#### 🔹 Option 1: If you have Python installed

```bash
# From your Bitcoin Core folder
python .\share\rpcauth\rpcauth.py sprint MyStrongPassw0rd!
```

#### 🔹 Option 2: Windows PowerShell (No Python Required)

Use the included PowerShell script:

```powershell
# Generate secure RPC auth
.\gen-rpcauth.ps1 -User sprint -Password "MyStrongPassw0rd!"
```

This outputs something like:
```
rpcauth=sprint:1cba94f0a8b1$7b56b8df2cf56f02a89493c8b7fbcf9c0e68ec7df91d054ee
```

**Configuration:**
- Add the `rpcauth=...` line to `bitcoin.conf`
- Use `BTC_RPC_USER=sprint` and `BTC_RPC_PASS=MyStrongPassw0rd!` in `.env`
- Remove any `rpcuser`/`rpcpassword` lines from `bitcoin.conf`

## Performance Tuning

### Memory Configuration
```conf
# Laptop/Development (8 GB RAM)
dbcache=2000

# Small Server (16 GB RAM)  
dbcache=8000

# Large Server (32+ GB RAM)
dbcache=16000
```

### CPU Optimization
```conf
# Rule: cores/2 for validation parallelism
# 4-core system
par=2

# 8-core system
par=4

# 16-core system
par=8
```

## ZeroMQ Configuration

**CRITICAL:** These ports are required for Bitcoin Sprint and should NOT be changed:

```conf
# Block updates (REQUIRED for Sprint)
zmqpubrawblock=tcp://127.0.0.1:28332

# Transaction updates (optional, useful for /mempool)
zmqpubrawtx=tcp://127.0.0.1:28333
```

**Note:** The Sprint API depends on block notifications at port 28332. Changing this will break the real-time block detection system.

## Complete Production bitcoin.conf Example

### Windows Configuration (rpcauth)

```conf
# ─────────────── Bitcoin Core Config for Sprint (rpcauth) ───────────────

server=1
rpcauth=sprint:1cba94f0a8b1$7b56b8df2cf56f02a89493c8b7fbcf9c0e68ec7df91d054ee
rpcbind=127.0.0.1
rpcallowip=127.0.0.1

# ZeroMQ (Sprint needs this)
zmqpubhashblock=tcp://127.0.0.1:28332
zmqpubhashtx=tcp://127.0.0.1:28333

# Pruned node (~2 GB)
prune=2000
txindex=0

# Networking
listen=1
maxconnections=40
port=8333

# Performance
dbcache=2000
par=2
```

### Complete Production Example

```conf
# === Network ===
mainnet=1
server=1
daemon=1

# === RPC Security (Production) ===
rpcauth=sprint:7d9ba5ae63c3d4dc30583ff4fe65a67e$9e3634e81c11659e3de036d0bf88f89cd169c1039e6e09607562d54765c649cc
rpcbind=127.0.0.1
rpcallowip=127.0.0.1

# === Performance ===
dbcache=8000
par=4
maxconnections=125

# === ZeroMQ (DO NOT CHANGE) ===
zmqpubhashblock=tcp://127.0.0.1:28332
zmqpubhashtx=tcp://127.0.0.1:28333

# === Logging ===
debug=0
logtimestamps=1
```

## Deployment Checklist

- [ ] Generate secure rpcauth credentials
- [ ] Set appropriate dbcache for your RAM
- [ ] Configure par= based on CPU cores
- [ ] Verify ZMQ ports 28332/28333 are available
- [ ] Update LICENSE_KEY in .env with real key
- [ ] Test Bitcoin Core connectivity before starting Sprint
- [ ] Monitor memory usage during initial sync
- [ ] Set up log rotation for Bitcoin Core logs
