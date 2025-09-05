# Bitcoin Sprint FastAPI Gateway

A modern, production-ready API gateway for the Bitcoin Sprint multi-chain blockchain infrastructure.

## 🚀 Features

- **Customer-Facing API**: Clean, professional API for external customers
- **Authentication & Authorization**: API key-based authentication with tier management
- **Rate Limiting**: Configurable rate limits per tier (Free/Pro/Enterprise)
- **Monitoring**: Prometheus metrics and health checks
- **Documentation**: Auto-generated Swagger UI at `/docs`
- **CORS Support**: Cross-origin resource sharing enabled
- **Proxy Gateway**: Seamlessly proxies requests to Go backend

## 📋 Prerequisites

- Python 3.8+
- Go backend running on port 8080
- Redis (optional, for distributed rate limiting)

## 🛠 Installation

1. **Install Dependencies**
   ```bash
   pip install -r requirements.txt
   ```

2. **Configure Environment**
   ```bash
   cp .env.example .env
   # Edit .env with your settings
   ```

3. **Start the Gateway**
   ```bash
   # Using PowerShell script
   .\start-fastapi.ps1

   # Or directly with Python
   python app.py
   ```

## 🔑 API Keys

The gateway uses API key authentication. Include the key in the `Authorization` header:

```
Authorization: Bearer your-api-key-here
```

### Demo API Keys

- **Free Tier**: `demo-key-free` (20 requests/minute)
- **Pro Tier**: `demo-key-pro` (1000 requests/minute)
- **Enterprise Tier**: `demo-key-enterprise` (unlimited)

## 📚 API Endpoints

### Core Endpoints

- `GET /health` - Basic health check
- `GET /status` - Comprehensive system status
- `GET /readiness` - Production readiness assessment
- `GET /docs` - Interactive API documentation
- `GET /redoc` - Alternative API documentation

### Proxied Endpoints

All other endpoints are proxied to the Go backend:
- `/mempool` - Mempool information
- `/blocks` - Block data
- `/peers` - Peer connections
- `/metrics` - Prometheus metrics (Enterprise only)

## 🏗 Architecture

```
[Client] → [FastAPI Gateway] → [Go Backend]
    ↓              ↓
[Auth]         [Bitcoin RPC]
[Rate Limit]   [ZMQ Streams]
[Monitoring]   [P2P Networks]
```

## ⚙️ Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GO_BACKEND_URL` | `http://localhost:8080` | Go backend URL |
| `REDIS_URL` | `redis://localhost:6379` | Redis for rate limiting |
| `HOST` | `0.0.0.0` | Server host |
| `PORT` | `8000` | Server port |
| `RATE_LIMIT_FREE` | `20/minute` | Free tier rate limit |
| `RATE_LIMIT_PRO` | `1000/minute` | Pro tier rate limit |

## 📊 Monitoring

### Health Checks

- `GET /health` - Basic connectivity check
- `GET /status` - Detailed system status
- `GET /readiness` - Production readiness

### Metrics

- `GET /metrics` - Prometheus metrics (Enterprise only)
- Request counts, latency, rate limit hits
- Active connections gauge

## 🔒 Security

- API key authentication
- Rate limiting per tier
- CORS configuration
- Input validation with Pydantic
- Security headers

## 🧪 Testing

### Test the API

```bash
# Health check
curl http://localhost:8000/health

# Status with API key
curl -H "Authorization: Bearer demo-key-free" \
     http://localhost:8000/status

# View API documentation
open http://localhost:8000/docs
```

### Rate Limiting Test

```bash
# This will hit rate limits for free tier
for i in {1..25}; do
  curl -H "Authorization: Bearer demo-key-free" \
       http://localhost:8000/status
done
```

## 🚀 Production Deployment

1. **Set Environment Variables**
   ```bash
   export GO_BACKEND_URL="http://your-backend:8080"
   export REDIS_URL="redis://your-redis:6379"
   export SECRET_KEY="your-production-secret"
   ```

2. **Use Production API Keys**
   ```bash
   export API_KEY_FREE="your-free-key"
   export API_KEY_PRO="your-pro-key"
   export API_KEY_ENTERPRISE="your-enterprise-key"
   ```

3. **Run with Gunicorn**
   ```bash
   pip install gunicorn
   gunicorn app:app -w 4 -k uvicorn.workers.UvicornWorker -b 0.0.0.0:8000
   ```

## 📈 Benefits

### For Customers
- **Clean API**: Professional, well-documented endpoints
- **Easy Onboarding**: Swagger UI for testing
- **Tier Management**: Clear rate limits and features per tier

### For Operations
- **Monitoring**: Comprehensive metrics and health checks
- **Security**: Centralized authentication and rate limiting
- **Scalability**: Lightweight gateway that can handle high traffic

### For Development
- **FastAPI**: Modern, async Python framework
- **Type Safety**: Pydantic models for all data
- **Auto Docs**: Automatic API documentation generation

## 🔧 Troubleshooting

### Common Issues

1. **Backend Connection Failed**
   - Ensure Go backend is running on port 8080
   - Check `GO_BACKEND_URL` environment variable

2. **Rate Limiting Issues**
   - Verify API key is correct
   - Check tier limits in configuration

3. **CORS Errors**
   - Configure `ALLOW_ORIGINS` in environment
   - Check browser console for details

### Logs

```bash
# Enable debug logging
export LOG_LEVEL=debug
python app.py
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.
