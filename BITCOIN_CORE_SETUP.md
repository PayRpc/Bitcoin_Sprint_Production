# Bitcoin Sprint - Real Bitcoin Core Connection Guide

## 🎯 Connecting Bitcoin Sprint to Real Bitcoin Core

This guide shows you how to connect Bitcoin Sprint to an actual Bitcoin Core node for real testing.

## Option 1: Quick Test with Public Node (Easiest)

### Using Bitcoin.com Public API
```json
{
    "license_key": "DEMO_LICENSE_BYPASS",
    "tier": "enterprise",
    "block_limit": 9999999,
    "poll_interval": 5,
    "turbo_mode": false,
    
    "rpc_nodes": ["https://api.bitcoin.com/v1/blockchain"],
    "rpc_user": "",
    "rpc_pass": "",
    
    "peer_secret": "demo_peer_secret_123",
    "dashboard_port": 8080,
    "api_base": "http://localhost:8080",
    "log_level": "debug"
}
```

### Using BlockCypher API
```json
{
    "license_key": "DEMO_LICENSE_BYPASS", 
    "tier": "enterprise",
    "block_limit": 9999999,
    "poll_interval": 3,
    "turbo_mode": true,
    
    "rpc_nodes": ["https://api.blockcypher.com/v1/btc/main"],
    "rpc_user": "",
    "rpc_pass": "",
    
    "peer_secret": "demo_peer_secret_123",
    "dashboard_port": 8080,
    "api_base": "http://localhost:8080",
    "log_level": "debug"
}
```

## Option 2: Local Bitcoin Core (Full Control)

### Step 1: Download and Install Bitcoin Core
1. Go to https://bitcoin.org/en/download
2. Download Bitcoin Core for Windows
3. Install it (will take ~500GB for full blockchain)

### Step 2: Configure Bitcoin Core
Create `bitcoin.conf` in `%APPDATA%\Bitcoin\`:
```
# RPC Settings
server=1
rpcallowip=127.0.0.1
rpcuser=test_user
rpcpassword=strong_random_password_here
rpcport=8332

# For faster sync (optional)
dbcache=4096
prune=550  # Keep only last ~500MB of blocks

# Network
listen=1
port=8333
```

### Step 3: Start Bitcoin Core
```powershell
# Start Bitcoin Core daemon
bitcoind.exe

# Or use Bitcoin Core GUI and enable server mode
```

### Step 4: Wait for Sync
Bitcoin Core needs to download the blockchain (~500GB). This takes hours/days.

## Option 3: Bitcoin Testnet (Recommended for Testing)

### Step 1: Configure Testnet
Create `bitcoin-testnet.conf`:
```
testnet=1
server=1
rpcallowip=127.0.0.1
rpcuser=test_user
rpcpassword=strong_random_password_here
rpcport=8332
listen=1
port=8333
dbcache=1024
```

### Step 2: Start Bitcoin Core Testnet
```powershell
bitcoind.exe -testnet -conf=bitcoin-testnet.conf
```

### Step 3: Configure Bitcoin Sprint for Testnet
```json
{
    "license_key": "DEMO_LICENSE_BYPASS",
    "tier": "enterprise", 
    "block_limit": 9999999,
    "poll_interval": 2,
    "turbo_mode": false,
    
    "rpc_nodes": ["http://localhost:8332"],
    "rpc_user": "test_user",
    "rpc_pass": "strong_random_password_here",
    
    "peer_secret": "demo_peer_secret_123",
    "dashboard_port": 8080,
    "api_base": "http://localhost:8080",
    "log_level": "debug"
}
```

## Option 4: Bitcoin Regtest (Instant Testing)

### Step 1: Start Regtest Mode
```powershell
bitcoind.exe -regtest -server -rpcuser=test_user -rpcpassword=strong_random_password_here
```

### Step 2: Generate Test Blocks
```powershell
# Create a new address
bitcoin-cli -regtest -rpcuser=test_user -rpcpassword=strong_random_password_here getnewaddress

# Generate 101 blocks (needed for coinbase maturity)
bitcoin-cli -regtest -rpcuser=test_user -rpcpassword=strong_random_password_here generatetoaddress 101 <address>
```

### Step 3: Configure Bitcoin Sprint for Regtest
```json
{
    "license_key": "DEMO_LICENSE_BYPASS",
    "tier": "enterprise",
    "block_limit": 9999999, 
    "poll_interval": 1,
    "turbo_mode": true,
    
    "rpc_nodes": ["http://localhost:18332"],
    "rpc_user": "test_user", 
    "rpc_pass": "strong_random_password_here",
    
    "peer_secret": "demo_peer_secret_123",
    "dashboard_port": 8080,
    "api_base": "http://localhost:8080",
    "log_level": "debug"
}
```

## 🧪 Testing Commands

Once connected to real Bitcoin Core:

```powershell
# Test Bitcoin Core RPC
curl -X POST http://localhost:8332 \
  -H "Content-Type: application/json" \
  -u test_user:strong_random_password_here \
  -d '{"jsonrpc":"1.0","id":"test","method":"getblockchaininfo","params":[]}'

# Test Bitcoin Sprint API
curl http://localhost:8080/status
curl http://localhost:8080/latest
curl http://localhost:8080/metrics

# Get real-time block data
curl http://localhost:8080/stream
```

## 📊 Monitoring

With real Bitcoin Core connection, you'll see:
- Real block heights
- Actual Bitcoin transactions
- Live network statistics
- Real-time blockchain data

## 🔧 Troubleshooting

### Bitcoin Core Not Syncing
- Check disk space (needs ~500GB)
- Check internet connection
- Wait patiently (can take days)

### RPC Connection Failed
- Verify bitcoin.conf settings
- Check firewall settings
- Ensure bitcoind is running
- Verify credentials

### Bitcoin Sprint Can't Connect
- Check Bitcoin Core RPC is enabled
- Verify credentials match
- Check port numbers (8332 mainnet, 18332 regtest)
- Look at Bitcoin Sprint logs

## 🚀 Quick Start Commands

```powershell
# Start with public API (no Bitcoin Core needed)
.\quick-test.ps1 -Mode mock

# Start with local testnet
.\test-with-bitcoin-core.ps1 -Network testnet

# Start with local regtest (instant blocks)
.\test-with-bitcoin-core.ps1 -Network regtest -GenerateBlocks

# Start with mainnet (requires full sync)
.\test-with-bitcoin-core.ps1 -Network mainnet
```
