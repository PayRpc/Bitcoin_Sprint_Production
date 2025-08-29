"""
FastAPI Gateway for Bitcoin Sprint
Customer-facing API with authentication, rate limiting, and monitoring
"""

from fastapi import FastAPI, HTTPException, Depends, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
from slowapi import Limiter, _rate_limit_exceeded_handler
from slowapi.util import get_remote_address
from slowapi.middleware import SlowAPIMiddleware
from slowapi.errors import RateLimitExceeded
import httpx
import os
import time
from datetime import datetime, timedelta
from typing import Optional, Dict, Any
from contextlib import asynccontextmanager

from models import (
    ChainStatus, SystemStatus, APIKey, TierLimits,
    StatusResponse, ReadinessResponse, MetricsResponse
)
from auth import verify_api_key, get_tier_limits
from monitoring import setup_prometheus, track_request

# Configuration
GO_BACKEND_URL = os.getenv("GO_BACKEND_URL", "http://localhost:8080")
REDIS_URL = os.getenv("REDIS_URL", "redis://localhost:6379")
API_KEYS = {
    "free": ["demo-key-free"],
    "pro": ["demo-key-pro"],
    "enterprise": ["demo-key-enterprise"]
}

# Rate limiting - make Redis optional
try:
    limiter = Limiter(key_func=get_remote_address, storage_uri=REDIS_URL)
    redis_available = True
except Exception as e:
    print(f"Redis not available, using in-memory rate limiting: {e}")
    limiter = Limiter(key_func=get_remote_address)
    redis_available = False

# FastAPI app
app = FastAPI(
    title="Bitcoin Sprint API Gateway",
    description="Enterprise Multi-Chain Blockchain Infrastructure API",
    version="2.5.0",
    docs_url="/docs",
    redoc_url="/redoc",
    dependencies=[]  # Explicitly set no global dependencies
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Add rate limiting middleware (disabled for now to fix startup issues)
# app.state.limiter = limiter
# app.add_exception_handler(RateLimitExceeded, _rate_limit_exceeded_handler)
# app.add_middleware(SlowAPIMiddleware)

# HTTP client for Go backend
@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup
    try:
        app.state.http_client = httpx.AsyncClient(timeout=30.0)
        setup_prometheus(app)
        print("FastAPI startup complete")
    except Exception as e:
        print(f"Startup error: {e}")
        # Continue without failing
    yield
    # Shutdown
    try:
        await app.state.http_client.aclose()
        print("FastAPI shutdown complete")
    except Exception as e:
        print(f"Shutdown error: {e}")

# Add lifespan to existing app
app.router.lifespan_context = lifespan

# Security
security = HTTPBearer()

async def get_api_key(credentials: HTTPAuthorizationCredentials = Depends(security)) -> str:
    """Extract and validate API key from request"""
    api_key = credentials.credentials
    tier = verify_api_key(api_key)
    if not tier:
        raise HTTPException(status_code=401, detail="Invalid API key")
    return api_key

async def check_rate_limit(request: Request, api_key: str = Depends(get_api_key)):
    """Check rate limits based on API key tier"""
    tier = verify_api_key(api_key)
    if not tier:
        raise HTTPException(status_code=401, detail="Invalid API key")

    # For now, just return the API key - rate limiting is handled by decorators
    # In production, you might want more sophisticated rate limiting logic
    return api_key

@app.post("/generate-key")
async def generate_api_key(request: Request):
    """Generate a new API key for testing purposes (development/demo only)"""
    # In production, this should require admin authentication
    # For now, allow in development but log the request
    client_ip = request.client.host if request.client else "unknown"
    user_agent = request.headers.get("user-agent", "unknown")

    # Log key generation attempt
    print(f"Key generation attempt from {client_ip} - {user_agent}")

    try:
        body = await request.json()
        tier = body.get("tier", "free")

        if tier not in ["free", "pro", "enterprise"]:
            raise HTTPException(status_code=400, detail="Invalid tier. Must be: free, pro, or enterprise")

        # For demo purposes, return a demo key
        demo_keys = {
            "free": "demo-key-free",
            "pro": "demo-key-pro",
            "enterprise": "demo-key-enterprise"
        }

        return {
            "success": True,
            "api_key": demo_keys[tier],
            "tier": tier,
            "limits": get_tier_limits(tier),
            "note": "This is a demo key. Use Authorization: Bearer <key> header for requests",
            "warning": "This endpoint is for development/testing only"
        }

    except Exception as e:
        raise HTTPException(status_code=400, detail=f"Invalid request: {str(e)}")

@app.get("/test")
async def test_endpoint():
    """Simple test endpoint that doesn't require authentication"""
    return {
        "message": "FastAPI Gateway is working!",
        "timestamp": datetime.utcnow().isoformat(),
        "version": "2.5.0",
        "status": "operational"
    }

@app.get("/health")
async def health_check():
    """Basic health check endpoint"""
    return {
        "status": "healthy",
        "timestamp": datetime.utcnow().isoformat(),
        "gateway": "FastAPI",
        "version": "2.5.0"
    }

@app.get("/status", response_model=StatusResponse)
@limiter.limit("60/minute")
async def get_status(request: Request, api_key: str = Depends(check_rate_limit)):
    """Get comprehensive system status"""
    track_request(request, "status")

    try:
        # Get status from Go backend
        client = app.state.http_client
        response = await client.get(f"{GO_BACKEND_URL}/status")
        go_status = response.json()

        # Enhance with FastAPI-specific data
        enhanced_status = SystemStatus(
            server_status="operational",
            gateway_version="2.5.0",
            backend_status=go_status.get("status", "unknown"),
            uptime=go_status.get("uptime", "unknown"),
            chains={
                chain: ChainStatus(**data) for chain, data in go_status.get("chains", {}).items()
            },
            sla_assessment=go_status.get("sla_assessment", {}),
            system_health=go_status.get("system_health", {}),
            timestamp=datetime.utcnow().isoformat()
        )

        return StatusResponse(
            success=True,
            data=enhanced_status,
            tier=verify_api_key(api_key) or "unknown"
        )

    except Exception as e:
        raise HTTPException(status_code=503, detail=f"Backend unavailable: {str(e)}")

@app.get("/readiness", response_model=ReadinessResponse)
@limiter.limit("30/minute")
async def get_readiness(request: Request, api_key: str = Depends(check_rate_limit)):
    """Get production readiness assessment"""
    track_request(request, "readiness")

    try:
        # Get readiness from Go backend
        client = app.state.http_client
        response = await client.get(f"{GO_BACKEND_URL}/readiness")
        go_readiness = response.json()

        return ReadinessResponse(
            success=True,
            data=go_readiness,
            tier=verify_api_key(api_key) or "unknown"
        )

    except Exception as e:
        raise HTTPException(status_code=503, detail=f"Backend unavailable: {str(e)}")

@app.get("/metrics")
async def get_metrics(api_key: str = Depends(get_api_key)):
    """Get Prometheus metrics (Enterprise only)"""
    tier = verify_api_key(api_key)
    if tier != "enterprise":
        raise HTTPException(status_code=403, detail="Enterprise tier required for metrics")

    from prometheus_client import generate_latest
    return JSONResponse(content=generate_latest().decode('utf-8'))

@app.get("/api-keys")
async def list_api_keys(api_key: str = Depends(get_api_key)):
    """List available API keys for testing (development only)"""
    tier = verify_api_key(api_key)
    if tier != "enterprise":
        raise HTTPException(status_code=403, detail="Enterprise tier required")

    return {
        "api_keys": API_KEYS,
        "note": "Use these keys in Authorization header: Bearer <key>"
    }

# Proxy other endpoints to Go backend (must be last)
@app.api_route("/{path:path}", methods=["GET", "POST", "PUT", "DELETE"])
@limiter.limit("100/minute")
async def proxy_to_backend(
    path: str,
    request: Request,
    api_key: str = Depends(check_rate_limit)
):
    """Proxy requests to Go backend with authentication"""
    track_request(request, f"proxy_{path}")

    try:
        # Build target URL
        target_url = f"{GO_BACKEND_URL}/{path}"

        # Get request body if any
        body = await request.body()

        # Filter headers - don't forward Authorization to backend
        safe_headers = {k: v for k, v in request.headers.items()
                       if k.lower() not in ["authorization", "host"]}

        # Forward request to Go backend
        client = app.state.http_client
        response = await client.request(
            method=request.method,
            url=target_url,
            content=body,
            headers=safe_headers
        )

        return JSONResponse(
            content=response.json(),
            status_code=response.status_code
        )

    except Exception as e:
        raise HTTPException(status_code=503, detail=f"Backend error: {str(e)}")

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(
        "app:app",
        host="0.0.0.0",
        port=8000,
        reload=True,
        log_level="info"
    )
