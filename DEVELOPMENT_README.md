# Bitcoin Sprint Development Environment

This guide explains how to start the complete Bitcoin Sprint development environment with all services running.

## 🚀 Quick Start

### Option 1: PowerShell Script (Recommended)
```powershell
.\start-dev.ps1
```

### Option 2: Batch Script
```batch
start-dev.bat
```

## 📋 What Gets Started

The startup script automatically starts all required services in the correct order:

1. **🐳 Docker Services**
   - Grafana (http://localhost:3000)
   - Redis, PostgreSQL, and other infrastructure services

2. **🐍 FastAPI Backend**
   - Main API server (http://localhost:8000)
   - Health endpoint: http://localhost:8000/health
   - API documentation: http://localhost:8000/docs

3. **⚛️ Next.js Frontend**
   - Web dashboard (http://localhost:3002)
   - Real-time monitoring and management interface

## 🎯 Service URLs

Once everything is running, you can access:

- **Frontend Dashboard**: http://localhost:3002
- **FastAPI API**: http://localhost:8000/docs
- **Grafana Monitoring**: http://localhost:3000 (admin/sprint123)
- **Health Check**: http://localhost:8000/health

## 🔧 Command Line Options

### PowerShell Script
```powershell
# Start everything
.\start-dev.ps1

# Skip specific services
.\start-dev.ps1 -SkipFrontend
.\start-dev.ps1 -SkipBackend
.\start-dev.ps1 -SkipGrafana

# Clean restart (stops previous instances)
.\start-dev.ps1 -Clean
```

### Batch Script
```batch
# Start everything
start-dev.bat

# Skip specific services
start-dev.bat --skip-frontend
start-dev.bat --skip-backend
start-dev.bat --skip-grafana
```

## 🛠️ Manual Startup (Alternative)

If you prefer to start services manually:

### 1. Start Docker Services
```bash
# Start Grafana
docker-compose -f grafana-compose.yml up -d

# Start infrastructure services
docker-compose -f config/docker-compose.yml up -d
```

### 2. Start FastAPI Backend
```bash
cd Bitcoin_Sprint_fastapi/fastapi-gateway
python -m venv venv
venv\Scripts\pip install -r requirements.txt
venv\Scripts\uvicorn main:app --host 0.0.0.0 --port 8000 --reload
```

### 3. Start Next.js Frontend
```bash
cd web
npm install
npm run dev:memory
```

## 🔍 Troubleshooting

### Common Issues

1. **Port conflicts**: Make sure ports 3000, 3002, and 8000 are available
2. **Docker not running**: Ensure Docker Desktop is running
3. **Python version**: Requires Python 3.13+ for FastAPI
4. **Node.js version**: Requires Node.js 18+ for Next.js

### Checking Service Status
```powershell
# Check if services are running
Get-NetTCPConnection | Where-Object { $_.LocalPort -in 3000,3002,8000,6379,5432 }
```

### Stopping Services
```powershell
# Stop all Docker services
docker-compose -f grafana-compose.yml down
docker-compose -f config/docker-compose.yml down

# Kill Node.js and Python processes
taskkill /f /im node.exe
taskkill /f /im python.exe
```

## 📊 Monitoring

- **Grafana**: Access at http://localhost:3000 with admin/sprint123
- **FastAPI Metrics**: Available at http://localhost:8000/metrics
- **Health Checks**: http://localhost:8000/health

## 🔐 Environment Variables

Key environment variables are configured in:
- `.env` (FastAPI configuration)
- `web/.env.local` (Next.js configuration)

## 📝 Development Notes

- The startup script waits for services to be ready before proceeding
- All services run in the background and can be stopped with Ctrl+C
- Logs are available in the respective service directories
- The environment is optimized for development with hot reloading enabled
