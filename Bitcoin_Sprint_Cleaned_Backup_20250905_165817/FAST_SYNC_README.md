# Bitcoin Sprint - Fast Sync Options
# ===================================

This guide provides multiple strategies to sync Bitcoin Core quickly without downloading the entire blockchain.

## 🚀 Available Sync Methods

### 1. **Pruned Node (RECOMMENDED)**
   - **Disk Usage**: ~550MB
   - **Sync Time**: 2-4 hours
   - **Method**: Keeps only recent blockchain data

```bash
# Use the default pruned configuration
cd C:\Projects\Bitcoin-Sprint-5\BItcoin_Sprint
docker-compose up bitcoin-core
```

### 2. **Lightning Fast Node**
   - **Disk Usage**: ~100MB
   - **Sync Time**: 30-60 minutes
   - **Method**: Extreme pruning + performance optimizations

```bash
# Use lightning fast configuration
cd C:\Projects\Bitcoin-Sprint-5\BItcoin_Sprint
docker-compose -f docker-compose.yml -f docker-compose-fast-sync.yml --profile lightning up bitcoin-core-lightning
```

### 3. **Ultra Fast Node**
   - **Disk Usage**: ~550MB
   - **Sync Time**: 1-2 hours
   - **Method**: Optimized connections + assumptions

```bash
# Use ultra fast configuration
cd C:\Projects\Bitcoin-Sprint-5\BItcoin_Sprint
docker-compose -f docker-compose.yml -f docker-compose-fast-sync.yml --profile ultra-fast up bitcoin-core-ultra
```

### 4. **Snapshot Method (Advanced)**
   - **Disk Usage**: 500GB+ download, then ~550MB
   - **Sync Time**: 2-4 hours after download
   - **Method**: Download pre-synced blockchain

```bash
# Download snapshot first, then use
cd C:\Projects\Bitcoin-Sprint-5\BItcoin_Sprint
docker-compose -f docker-compose.yml -f docker-compose-fast-sync.yml --profile snapshot up bitcoin-core-snapshot
```

## 📊 Configuration Comparison

| Method | Disk Usage | Sync Time | Reliability | Complexity |
|--------|------------|-----------|-------------|------------|
| Pruned | 550MB | 2-4 hours | High | Low |
| Lightning | 100MB | 30-60 min | Medium | Low |
| Ultra Fast | 550MB | 1-2 hours | High | Low |
| Snapshot | 550MB* | 2-4 hours* | High | High |

*After initial download

## ⚙️ Manual Configuration

### Using the Setup Script

```powershell
# Setup pruned node
.\setup-fast-sync.ps1 -UsePrunedNode

# Setup UTXO set (alternative)
.\setup-fast-sync.ps1 -UseUTXOSet

# Setup snapshot (advanced)
.\setup-fast-sync.ps1 -UseSnapshot
```

### Direct Docker Commands

```bash
# Start with custom configuration
docker run -d \
  --name bitcoin-core \
  -p 8332:8332 \
  -p 8333:8333 \
  -p 28332:28332 \
  -p 28333:28333 \
  -v bitcoin-data:/home/bitcoin/.bitcoin \
  -v ./config/bitcoin.conf:/home/bitcoin/.bitcoin/bitcoin.conf:ro \
  bitcoin/bitcoin:latest \
  bitcoind -printtoconsole -conf=/home/bitcoin/.bitcoin/bitcoin.conf
```

## 🔧 Optimization Features

### Current Bitcoin Configuration Includes:
- ✅ **Pruning**: Keeps only 550MB of data
- ✅ **Optimized Cache**: 256MB database cache
- ✅ **Reduced Connections**: 32 max connections
- ✅ **Disabled Wallet**: Faster sync
- ✅ **Block-only Mode**: No transaction filtering
- ✅ **Fast Peers**: Connects to known fast nodes

### Lightning Configuration Adds:
- ✅ **Extreme Pruning**: 100MB only
- ✅ **Large Cache**: 1024MB database cache
- ✅ **Minimal Connections**: 8 max connections
- ✅ **Assumed Valid**: Skips validation for recent blocks
- ✅ **Parallel Processing**: 2 CPU threads

## 📈 Monitoring Sync Progress

```bash
# Check sync status
docker exec bitcoin-core bitcoin-cli -rpcuser=sprint -rpcpassword=sprint_password_2025 getblockchaininfo

# Monitor logs
docker logs -f bitcoin-core

# Check disk usage
docker system df -v
```

## 🎯 Recommended Approach

1. **Start with Pruned Node** (default configuration)
2. **Monitor sync progress** using the commands above
3. **Switch to Lightning** if you need faster sync
4. **Use Snapshot** only if you have fast internet and need instant setup

## ⚠️ Important Notes

- **Security**: Pruned nodes still validate all blocks
- **Functionality**: All methods support RPC and ZMQ
- **Backup**: Always backup your configuration files
- **Network**: Fast sync methods may use more bandwidth initially
- **Compatibility**: All methods work with existing Grafana dashboards

## 🔗 Useful Resources

- [Bitcoin Core Documentation](https://bitcoincore.org/en/doc/)
- [Pruning Guide](https://bitcoincore.org/en/2020/06/14/bitcoin-core-0-20-0/)
- [Fast Sync Options](https://bitcoincore.org/en/releases/25.1/)
- [UTXO Set](https://bitcoincore.org/bin/utxo/)
