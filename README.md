# 🚀 Bitcoin Sprint
**Multi-Chain Enterprise Sprint Platform** | Version 2.1.0

[![Go](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Production%20Ready-brightgreen.svg)]()

## 🎯 Quick Start

### 1. **Build & Run Backend**
```bash
# Build optimized version
powershell -ExecutionPolicy Bypass -File ./build-optimized.ps1 -Release

# Start (auto-detects best tier)
./bitcoin-sprint.exe --tier=auto
```

### 2. **Start Web Dashboard**
```bash
cd web
node smart-start.js  # Auto-detects backend and starts on correct port
```

### 3. **Access Interface**
- **Dashboard:** http://localhost:PORT (auto-detected)
- **API:** http://localhost:PORT/api
- **Metrics:** http://localhost:PORT/metrics

## 🏆 Tier System

| **Tier** | **Port** | **Features** |
|----------|----------|--------------|
| **FREE** | 8080 | Basic Bitcoin connectivity |
| **BUSINESS** | 8082 | Enhanced features + analytics |
| **ENTERPRISE** | 9000 | Full features + priority support |

## 📖 Full Documentation

**👉 See [DOCUMENTATION.md](DOCUMENTATION.md) for complete setup, configuration, and deployment guide.**

## 🔧 Development

```bash
# Format & test
go fmt ./... && go test ./...

# Web development
cd web && npm run dev:auto  # Auto-detects backend tier
```

## 🚀 Production Deployment

```bash
# Docker
docker-compose up -d

# Fly.io
fly deploy
```

## 📊 Key Features

- ✅ **Automatic Tier Detection** - Web interface adapts to backend
- ✅ **Zero Port Conflicts** - Smart port allocation per tier  
- ✅ **Real-time Monitoring** - Prometheus + Grafana integration
- ✅ **Multi-Chain Support** - Bitcoin, Ethereum, Solana
- ✅ **Enterprise Security** - Tier-based authentication & rate limiting

---

**🎯 Ready to Sprint with Bitcoin!** 🚀
