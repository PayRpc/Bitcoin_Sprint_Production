# 📚 Bitcoin Sprint - Complete Documentation
**Version:** 2.1.0 | **Date:** September 5, 2025

## 📖 Table of Contents
1. [Quick Start](#quick-start)
2. [Tier System](#tier-system) 
3. [Port Configuration](#port-configuration)
4. [Web Interface](#web-interface)
5. [Deployment](#deployment)
6. [Security](#security)
7. [Development](#development)

---

## 🚀 Quick Start

### Installation & Setup
```bash
# Clone repository
git clone https://github.com/PayRpc/Bitcoin_Sprint_Production.git
cd Bitcoin_Sprint_Production

# Build optimized version
powershell -ExecutionPolicy Bypass -File ./build-optimized.ps1 -Release

# Start with automatic tier detection
./bitcoin-sprint.exe --tier=auto
```

### Web Dashboard
```bash
cd web
node smart-start.js  # Auto-detects backend tier and starts on correct port
```

---

## 🏆 Tier System

Bitcoin Sprint operates in three tiers with automatic detection:

| **Tier** | **Backend Port** | **Web Port** | **Features** |
|----------|------------------|--------------|--------------|
| **FREE** | 8080 | 3000 | Basic Bitcoin connectivity, 10 req/min |
| **BUSINESS** | 8082 | 3001 | Enhanced features, 100 req/min |
| **ENTERPRISE** | 9000 | 3002 | Full features, 1000 req/min, priority support |

### Tier Configuration Files:
- `.env.free` - FREE tier settings
- `.env.business` - BUSINESS tier settings  
- `.env.enterprise` - ENTERPRISE tier settings

### Starting Specific Tiers:
```bash
# FREE tier
./bitcoin-sprint.exe --config=.env.free --port=8080

# BUSINESS tier  
./bitcoin-sprint.exe --config=.env.business --port=8082

# ENTERPRISE tier
./bitcoin-sprint.exe --config=.env.enterprise --port=9000
```

---

## 🔌 Port Configuration

### Production Port Allocation
| **Service** | **FREE** | **BUSINESS** | **ENTERPRISE** | **Production** |
|-------------|----------|--------------|----------------|----------------|
| **API Server** | 8080 | 8082 | 9000 | - |
| **Web Dashboard** | 3000 | 3001 | 3002 | - |
| **Admin Panel** | 8081 | 8083 | 9001 | - |
| **Prometheus** | - | - | - | 9090 |
| **Grafana** | - | - | - | 3003 |

### Dynamic Backend Detection
The web interface automatically detects which backend tier is running:
1. **Probe Order:** Enterprise (9000) → Business (8082) → Free (8080)
2. **Automatic Failover:** Falls back to lower tier if higher unavailable
3. **Real-time Updates:** Dashboard adapts instantly to backend changes

---

## 🌐 Web Interface

### Smart Startup
```bash
cd web

# Automatic tier detection (recommended)
node smart-start.js

# Force specific tier
node smart-start.js enterprise
node smart-start.js business --production
```

### Manual Startup
```bash
# Development
npm run dev:free        # Port 3000
npm run dev:business    # Port 3001  
npm run dev:enterprise  # Port 3002

# Production
npm run start:free      # Port 3000
npm run start:business  # Port 3001
npm run start:enterprise # Port 3002
```

### Testing Connectivity
```bash
npm run test:connection  # Test with automatic tier detection
npm run test:all        # Comprehensive test suite
```

### Key Files:
- `smart-start.js` - Intelligent startup script
- `lib/tier-detector.js` - Backend detection utility
- `.env.local` - Development configuration
- `.env.production` - Production configuration

---

## 🚀 Deployment

### Docker Compose
```bash
# Start all services
docker-compose up -d

# Start specific tier
docker-compose -f docker-compose.yml -f docker-compose.enterprise.yml up -d
```

### Fly.io Deployment
```bash
# Deploy to Fly.io
fly deploy

# Scale for enterprise
fly scale count 3
```

### Build Optimization
```bash
# Maximum optimization build
powershell -ExecutionPolicy Bypass -File ./build-optimized.ps1 -Release

# Development build with race detection
go build -race -o bitcoin-sprint-dev.exe .
```

### Environment Variables:
- `BITCOIN_SPRINT_TIER` - Set tier: `free`, `business`, `enterprise`, or `auto`
- `AUTO_TIER_DETECTION` - Enable automatic detection: `true`/`false`
- `FALLBACK_TO_FREE` - Fallback to free tier: `true`/`false`

---

## 🔒 Security

### API Authentication
Each tier uses specific API keys:
- **FREE:** `free-api-key-changeme`
- **BUSINESS:** `business-api-key-changeme`
- **ENTERPRISE:** `enterprise-api-key-changeme`

### Security Headers
- Content Security Policy with tier-specific origins
- CORS protection for cross-tier access
- Rate limiting per tier (10/100/1000 req/min)

### TLS/SSL
```bash
# Generate TLS certificates
powershell -ExecutionPolicy Bypass -File ./generate-tls-certs.ps1
```

### Entropy Security
- Hardware entropy sources when available
- Rust-based entropy bridge for enhanced randomness
- Fallback to Node.js crypto for development

---

## 🛠️ Development

### Prerequisites
- Go 1.21+
- Node.js 18+
- Docker & Docker Compose
- PowerShell 7+ (Windows)

### Development Workflow
```bash
# 1. Start backend (any tier)
./bitcoin-sprint.exe --tier=free

# 2. Start web interface (auto-detects backend)
cd web && node smart-start.js

# 3. Run tests
npm run test:all

# 4. Build optimized version
powershell -ExecutionPolicy Bypass -File ./build-optimized.ps1
```

### Key Commands
```bash
# Format code
go fmt ./...
gofmt -s -w .

# Run analysis
go vet ./...

# Run benchmarks
go test ./... -bench . -benchmem

# Clean build artifacts
powershell -Command "Remove-Item -Path '*.exe', '*.log' -ErrorAction SilentlyContinue"
```

### Directory Structure
```
Bitcoin_Sprint_Production/
├── cmd/sprintd/          # Main application entry
├── internal/             # Private packages
├── web/                  # Next.js web interface
├── config/              # Configuration files
├── docs/                # Additional documentation
├── scripts/             # Utility scripts
└── tests/               # Test suites
```

### Testing
```bash
# Backend tests
go test ./...

# Web interface tests
cd web && npm test

# Integration tests
npm run test:entropy
npm run test:connection
```

---

## 📊 Monitoring & Metrics

### Prometheus Metrics
Available at `/metrics` endpoint on each tier:
- `bitcoin_sprint_requests_total` - Total API requests
- `bitcoin_sprint_response_time` - Response time histogram
- `bitcoin_sprint_active_connections` - Active connections
- `bitcoin_sprint_bitcoin_peers` - Connected Bitcoin peers

### Grafana Dashboard
Access at `http://localhost:3003` in production:
- Real-time metrics visualization
- Tier-specific dashboards
- Alert management
- Performance analytics

### Health Checks
```bash
# Check backend health
curl http://localhost:8080/health

# Check specific tier
curl http://localhost:9000/health  # Enterprise
curl http://localhost:8082/health  # Business
curl http://localhost:8080/health  # Free
```

---

## 🆘 Troubleshooting

### Common Issues

**Port Conflicts:**
```bash
# Check what's using a port
netstat -ano | findstr :8080

# Kill process on port
taskkill /PID <PID> /F
```

**Backend Not Detected:**
```bash
# Check tier detection logs
cd web && TIER_DETECTION_LOG=true node smart-start.js

# Manual tier specification
cd web && BITCOIN_SPRINT_TIER=free npm run dev:free
```

**Build Issues:**
```bash
# Clean and rebuild
powershell -Command "Remove-Item -Path '*.exe' -ErrorAction SilentlyContinue"
go clean -cache
go build .
```

**Web Interface Issues:**
```bash
# Reset web configuration
cd web && npm run setup:dev

# Test backend connectivity
npm run test:connection
```

### Support
- **Issues:** [GitHub Issues](https://github.com/PayRpc/Bitcoin_Sprint_Production/issues)
- **Discussions:** [GitHub Discussions](https://github.com/PayRpc/Bitcoin_Sprint_Production/discussions)
- **Enterprise Support:** Contact enterprise tier customers

---

## 📄 License
This project is licensed under the terms specified in the LICENSE file.

---

**🎯 Ready to Sprint with Bitcoin! 🚀**
