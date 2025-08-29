# FastAPI Gateway for Bitcoin Sprint

A production-ready FastAPI API gateway for the Bitcoin Sprint multi-chain blockchain platform. This gateway provides enterprise-grade features including authentication, rate limiting, monitoring, and seamless integration with your Go backend.

## 🚀 Features

- **🔐 Authentication**: API key and JWT-based authentication with tier management
- **⏱️ Rate Limiting**: Redis-backed distributed rate limiting with tier-specific limits
- **📊 Monitoring**: Prometheus metrics, structured logging, and health checks
- **🌐 WebSocket Support**: Real-time data streaming proxy
- **🛡️ Security**: CORS protection, trusted hosts, request validation
- **📈 Scalability**: Async processing, connection pooling, and background tasks
- **🐳 Containerization**: Docker support with multi-stage builds
- **📚 Auto-Documentation**: OpenAPI/Swagger documentation

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Client Apps   │────│ FastAPI Gateway │────│   Go Backend    │
│                 │    │                 │    │                 │
│ • Web Apps      │    │ • Auth & Rate   │    │ • Blockchain    │
│ • Mobile Apps   │    │   Limiting      │    │   Logic         │
│ • API Clients   │    │ • Request Proxy │    │ • Core Services │
│ • WebSockets    │    │ • Metrics       │    │ • Data Storage  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │     Redis       │
                       │                 │
                       │ • Rate Limiting │
                       │ • Caching       │
                       │ • API Keys      │
                       └─────────────────┘
```

## 📋 Prerequisites

- Python 3.8+
- Redis (for rate limiting and caching)
- Go backend running on port 8080
- Docker & Docker Compose (optional, for containerized deployment)

## 🛠️ Quick Start

### 1. Clone and Setup

```bash
# Navigate to the gateway directory
cd fastapi-gateway

# Copy environment template
cp .env.example .env

# Edit configuration (see Configuration section below)
notepad .env
```

### 2. Install Dependencies

```bash
# Using PowerShell script (Windows)
.\start-fastapi.ps1 -Build

# Or manually
python -m venv venv
venv\Scripts\activate  # On Windows
pip install -r requirements.txt
```

### 3. Start Services

```bash
# Start Redis (if not already running)
# Using Docker
docker run -d -p 6379:6379 redis:7-alpine

# Start the gateway
.\start-fastapi.ps1
```

### 4. Verify Installation

```bash
# Run integration tests
.\test-integration.ps1

# Check health endpoint
curl http://localhost:8000/health
```

## ⚙️ Configuration

### Environment Variables

Create a `.env` file with the following variables:

```bash
# API Settings
API_TITLE="Bitcoin Sprint API Gateway"
API_VERSION="1.0.0"

# Server Settings
HOST="0.0.0.0"
PORT=8000
DEBUG=false

# Security (CHANGE IN PRODUCTION!)
SECRET_KEY="your-super-secret-key-32-chars-minimum"

# Backend Settings
BACKEND_URL="http://localhost:8080"
BACKEND_TIMEOUT=30.0

# Redis Settings
REDIS_URL="redis://localhost:6379"

# Rate Limiting
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=60

# CORS Settings
CORS_ORIGINS=["http://localhost:3000", "https://yourdomain.com"]
```

### User Tiers

The gateway supports three user tiers with different rate limits:

- **Free**: 10 requests/minute
- **Pro**: 100 requests/minute
- **Enterprise**: 1000 requests/minute

## 🔐 Authentication

### API Key Authentication

```bash
# Using header
curl -H "X-API-Key: your-api-key" http://localhost:8000/api/v1/endpoint

# Using query parameter
curl "http://localhost:8000/api/v1/endpoint?api_key=your-api-key"
```

### JWT Authentication

```bash
# Login to get token
curl -X POST "http://localhost:8000/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username": "demo", "password": "demo"}'

# Use token in requests
curl -H "Authorization: Bearer your-jwt-token" http://localhost:8000/api/v1/endpoint
```

## 🐳 Docker Deployment

### Using Docker Compose

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f fastapi-gateway

# Stop services
docker-compose down
```

### Manual Docker Build

```bash
# Build image
docker build -t bitcoin-sprint-gateway .

# Run container
docker run -d \
  --name fastapi-gateway \
  -p 8000:8000 \
  --env-file .env \
  bitcoin-sprint-gateway
```

## 📊 Monitoring

### Health Checks

```bash
# Gateway health
curl http://localhost:8000/health

# Backend health (proxied)
curl http://localhost:8000/health
```

### Prometheus Metrics

```bash
# Metrics endpoint
curl http://localhost:8000/metrics
```

### Structured Logging

The gateway uses structured JSON logging with the following fields:
- `timestamp`: ISO 8601 timestamp
- `level`: Log level (INFO, ERROR, etc.)
- `message`: Log message
- `method`: HTTP method
- `path`: Request path
- `status`: HTTP status code
- `duration`: Request duration in seconds
- `user`: Authenticated user (if available)

## 🔌 API Endpoints

### Gateway Endpoints

- `GET /health` - Health check
- `GET /metrics` - Prometheus metrics
- `POST /auth/login` - JWT login
- `WS /ws` - WebSocket proxy

### Proxied Endpoints

All other endpoints are proxied to your Go backend:

```
GET    /api/v1/*  → http://localhost:8080/api/v1/*
POST   /api/v1/*  → http://localhost:8080/api/v1/*
PUT    /api/v1/*  → http://localhost:8080/api/v1/*
DELETE /api/v1/*  → http://localhost:8080/api/v1/*
```

## 🧪 Testing

### Integration Tests

```bash
# Run all tests
.\test-integration.ps1

# Quick test (skip performance)
.\test-integration.ps1 -Quick

# Test with custom URLs
.\test-integration.ps1 -GatewayUrl "http://prod-gateway:8000" -BackendUrl "http://prod-backend:8080"
```

### Manual Testing

```bash
# Test health
curl http://localhost:8000/health

# Test authentication
curl -H "X-API-Key: test-key" http://localhost:8000/api/v1/test

# Test rate limiting (make multiple requests quickly)
for i in {1..20}; do curl http://localhost:8000/api/v1/test; done
```

## 🚀 Production Deployment

### 1. Security Checklist

- [ ] Change `SECRET_KEY` to a strong random value
- [ ] Set `DEBUG=false`
- [ ] Configure specific `CORS_ORIGINS`
- [ ] Set specific `TRUSTED_HOSTS`
- [ ] Use HTTPS in production
- [ ] Implement proper API key management
- [ ] Set up monitoring and alerting

### 2. Performance Optimization

- [ ] Configure Redis cluster for high availability
- [ ] Set up load balancer
- [ ] Configure connection pooling
- [ ] Implement caching strategies
- [ ] Set up database for user management

### 3. Monitoring Setup

```bash
# Prometheus configuration
scrape_configs:
  - job_name: 'fastapi-gateway'
    static_configs:
      - targets: ['localhost:8000']
    metrics_path: '/metrics'
```

## 🛠️ Development

### Project Structure

```
fastapi-gateway/
├── main.py                 # Main FastAPI application
├── requirements.txt        # Python dependencies
├── .env.example           # Environment template
├── Dockerfile            # Docker configuration
├── docker-compose.yml    # Docker Compose setup
├── start-fastapi.ps1     # Windows startup script
├── test-integration.ps1  # Integration tests
└── README.md             # This file
```

### Adding New Features

1. **New Endpoint**: Add route in `main.py`
2. **New Middleware**: Add to `app.add_middleware()`
3. **New Dependency**: Add to `requirements.txt`
4. **New Configuration**: Add to `.env.example` and `Settings` class

### Code Style

- Use type hints for all function parameters
- Follow PEP 8 style guidelines
- Add docstrings to all functions
- Use structured logging instead of print statements

## 🐛 Troubleshooting

### Common Issues

#### Gateway won't start

- Check Python version (3.8+ required)
- Verify all dependencies are installed
- Check Redis connection
- Review environment variables

#### Authentication fails

- Verify API key format
- Check JWT token expiration
- Ensure Redis is accessible for rate limiting

#### Backend connection fails

- Verify Go backend is running on port 8080
- Check network connectivity
- Review backend URL configuration

#### Rate limiting not working

- Ensure Redis is running and accessible
- Check Redis URL configuration
- Verify rate limit settings

### Debug Mode

Enable debug mode for detailed logging:

```bash
# In .env file
DEBUG=true

# Start with debug logging
uvicorn main:app --reload --log-level debug
```

## 📚 API Documentation

Once running, visit:

- **Swagger UI**: `http://localhost:8000/docs`
- **ReDoc**: `http://localhost:8000/redoc`
- **OpenAPI Schema**: `http://localhost:8000/openapi.json`

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new features
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🆘 Support

For support and questions:

- Check the troubleshooting section
- Review the integration tests
- Check the logs for error messages
- Open an issue on GitHub

---

**Happy coding!** 🚀
