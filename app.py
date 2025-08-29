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

# Rate limiting
limiter = Limiter(key_func=get_remote_address, storage_uri=REDIS_URL)

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

# Add rate limiting middleware
app.state.limiter = limiter
app.add_exception_handler(RateLimitExceeded, _rate_limit_exceeded_handler)
app.add_middleware(SlowAPIMiddleware)

# HTTP client for Go backend
@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup
    app.state.http_client = httpx.AsyncClient(timeout=30.0)
    setup_prometheus(app)
    yield
    # Shutdown
    await app.state.http_client.aclose()

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

    limits = get_tier_limits(tier)

    # Apply rate limiting based on tier
    if tier == "free":
        limiter.limit("20/minute")(request)
    elif tier == "pro":
        limiter.limit("1000/minute")(request)
    # Enterprise has no rate limiting

    return api_key

@app.get("/test")
async def test_endpoint():
    """Simple test endpoint without authentication"""
    return {"message": "Test endpoint works!", "timestamp": datetime.utcnow().isoformat()}

@app.post("/generate-key")
async def generate_api_key(request: Request):
    """Generate a new API key for testing purposes (no auth required for demo)"""
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
            "note": "This is a demo key. Use Authorization: Bearer <key> header for requests"
        }

    except Exception as e:
        raise HTTPException(status_code=400, detail=f"Invalid request: {str(e)}")

@app.get("/health")
async def health_check():
    """Basic health check endpoint"""
    return {"status": "healthy", "timestamp": datetime.utcnow().isoformat()}

@app.get("/status", response_model=StatusResponse)
@limiter.limit("60/minute")
async def get_status(request: Request, api_key: str = Depends(check_rate_limit)):
    """Get comprehensive system status"""
    track_request(request, "status")

    try:
        # Get status from Go backend
        async with app.state.http_client as client:
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
        async with app.state.http_client as client:
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

# Proxy other endpoints to Go backend
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

        # Forward request to Go backend
        async with app.state.http_client as client:
            response = await client.request(
                method=request.method,
                url=target_url,
                content=body,
                headers=dict(request.headers)
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
