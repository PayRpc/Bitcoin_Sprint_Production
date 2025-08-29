"""
FastAPI Gateway for Bitcoin Sprint
Enterprise-grade API gateway with authentication, rate limiting, and monitoring
"""

import asyncio
import json
import logging
import time
from contextlib import asynccontextmanager
from datetime import datetime, timedelta
from typing import Any, Dict, List, Optional, Union
from urllib.parse import urljoin

import httpx
import redis.asyncio as redis
import structlog
from fastapi import Depends, FastAPI, HTTPException, Request, Response, WebSocket, WebSocketDisconnect
from fastapi.middleware.cors import CORSMiddleware
from fastapi.middleware.trustedhost import TrustedHostMiddleware
from fastapi.responses import JSONResponse
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
from jose import JWTError, jwt
from passlib.context import CryptContext
from prometheus_client import Counter, Gauge, Histogram, generate_latest, CONTENT_TYPE_LATEST
from pydantic import BaseModel, Field
from pydantic_settings import BaseSettings
from slowapi import Limiter, _rate_limit_exceeded_handler
from slowapi.errors import RateLimitExceeded
from slowapi.middleware import SlowAPIMiddleware
from slowapi.util import get_remote_address
from starlette.responses import StreamingResponse

# Configure structured logging
structlog.configure(
    processors=[
        structlog.stdlib.filter_by_level,
        structlog.stdlib.add_logger_name,
        structlog.stdlib.add_log_level,
        structlog.stdlib.PositionalArgumentsFormatter(),
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.StackInfoRenderer(),
        structlog.processors.format_exc_info,
        structlog.processors.UnicodeDecoder(),
        structlog.processors.JSONRenderer()
    ],
    context_class=dict,
    logger_factory=structlog.stdlib.LoggerFactory(),
    wrapper_class=structlog.stdlib.BoundLogger,
    cache_logger_on_first_use=True,
)

logger = structlog.get_logger()

# Configuration
class Settings(BaseSettings):
    # API Settings
    api_title: str = "Bitcoin Sprint API Gateway"
    api_version: str = "1.0.0"
    api_description: str = "Enterprise-grade API gateway for Bitcoin Sprint blockchain platform"

    # Server Settings
    host: str = "0.0.0.0"
    port: int = 8000
    debug: bool = False

    # Security Settings
    secret_key: str = "your-secret-key-change-in-production"
    algorithm: str = "HS256"
    access_token_expire_minutes: int = 30

    # Backend Settings
    backend_url: str = "http://localhost:8080"
    backend_timeout: float = 30.0

    # Redis Settings
    redis_url: str = "redis://localhost:6379"
    redis_pool_size: int = 10
    redis_cache_url: str = "redis://localhost:6379/1"
    redis_rate_limit_url: str = "redis://localhost:6379/2"

    # Database Settings
    database_url: str = "postgresql://user:password@localhost:5432/bitcoin_sprint"

    # Rate Limiting
    rate_limit_requests: int = 100
    rate_limit_window: int = 60  # seconds

    # CORS Settings
    cors_origins: List[str] = ["http://localhost:3000", "http://localhost:8080"]
    cors_allow_credentials: bool = True
    cors_allow_methods: List[str] = ["*"]
    cors_allow_headers: List[str] = ["*"]

    # Trusted Hosts
    trusted_hosts: List[str] = ["*"]

    # API Key Settings
    api_key_header: str = "X-API-Key"
    api_key_query: str = "api_key"

    # Logging Settings
    log_level: str = "INFO"
    log_format: str = "json"

    # Monitoring Settings
    prometheus_enabled: bool = True
    metrics_port: int = 9090

    class Config:
        env_file = ".env"
        case_sensitive = False

settings = Settings()

# Password hashing
pwd_context = CryptContext(schemes=["bcrypt"], deprecated="auto")

# Redis client
redis_client: Optional[redis.Redis] = None

# Rate limiter
limiter = Limiter(
    key_func=get_remote_address,
    default_limits=[f"{settings.rate_limit_requests} per {settings.rate_limit_window} seconds"]
)

# Prometheus metrics
REQUEST_COUNT = Counter(
    'api_requests_total',
    'Total number of API requests',
    ['method', 'endpoint', 'status']
)

REQUEST_LATENCY = Histogram(
    'api_request_duration_seconds',
    'API request duration in seconds',
    ['method', 'endpoint']
)

ACTIVE_CONNECTIONS = Gauge(
    'api_active_connections',
    'Number of active WebSocket connections'
)

# Pydantic models
class Token(BaseModel):
    access_token: str
    token_type: str = "bearer"

class TokenData(BaseModel):
    username: Optional[str] = None
    scopes: List[str] = []

class User(BaseModel):
    username: str
    email: Optional[str] = None
    full_name: Optional[str] = None
    disabled: Optional[bool] = None
    tier: str = "free"  # free, pro, enterprise

class UserInDB(User):
    hashed_password: str

class APIKey(BaseModel):
    key: str
    name: str
    created_at: datetime
    expires_at: Optional[datetime] = None
    tier: str = "free"
    rate_limit: int = 100
    enabled: bool = True

class HealthResponse(BaseModel):
    status: str
    timestamp: datetime
    version: str
    uptime: float
    backend_status: str

# Global variables
start_time = time.time()
backend_status = "unknown"

# Authentication functions
def verify_password(plain_password: str, hashed_password: str) -> bool:
    return pwd_context.verify(plain_password, hashed_password)

def get_password_hash(password: str) -> str:
    return pwd_context.hash(password)

def create_access_token(data: dict, expires_delta: Optional[timedelta] = None):
    to_encode = data.copy()
    if expires_delta:
        expire = datetime.utcnow() + expires_delta
    else:
        expire = datetime.utcnow() + timedelta(minutes=15)
    to_encode.update({"exp": expire})
    encoded_jwt = jwt.encode(to_encode, settings.secret_key, algorithm=settings.algorithm)
    return encoded_jwt

async def get_current_user(token: str = Depends(HTTPBearer())) -> User:
    credentials_exception = HTTPException(
        status_code=401,
        detail="Could not validate credentials",
        headers={"WWW-Authenticate": "Bearer"},
    )
    try:
        payload = jwt.decode(token, settings.secret_key, algorithms=[settings.algorithm])
        username: str = payload.get("sub")
        if username is None:
            raise credentials_exception
        token_data = TokenData(username=username)
    except JWTError:
        raise credentials_exception

    # In a real application, you would fetch user from database
    # For demo purposes, we'll create a mock user
    user = User(
        username=token_data.username,
        email=f"{token_data.username}@example.com",
        tier="pro"
    )
    return user

async def get_api_key_from_header(request: Request) -> Optional[str]:
    """Extract API key from header or query parameter"""
    api_key = request.headers.get(settings.api_key_header.lower())
    if not api_key:
        api_key = request.query_params.get(settings.api_key_query)
    return api_key

async def validate_api_key(api_key: str) -> Optional[APIKey]:
    """Validate API key against Redis/database"""
    if not redis_client:
        return None

    try:
        key_data = await redis_client.get(f"api_key:{api_key}")
        if key_data:
            data = json.loads(key_data)
            return APIKey(**data)
    except Exception as e:
        logger.error("Error validating API key", error=str(e))

    return None

async def authenticate_request(request: Request) -> Optional[User]:
    """Authenticate request using API key or JWT token"""
    api_key = await get_api_key_from_header(request)
    if api_key:
        key_info = await validate_api_key(api_key)
        if key_info and key_info.enabled:
            # Check if key is expired
            if key_info.expires_at and datetime.utcnow() > key_info.expires_at:
                raise HTTPException(status_code=401, detail="API key expired")

            return User(
                username=f"api_key_{key_info.name}",
                tier=key_info.tier
            )

    # Fallback to JWT authentication
    try:
        auth_header = request.headers.get("authorization")
        if auth_header and auth_header.startswith("Bearer "):
            token = auth_header.split(" ")[1]
            return await get_current_user(token)
    except Exception:
        pass

    return None

# Backend proxy functions
async def proxy_request(
    method: str,
    path: str,
    request: Request,
    user: Optional[User] = None
) -> Response:
    """Proxy request to backend service"""
    backend_url = urljoin(settings.backend_url, path)

    # Prepare headers
    headers = dict(request.headers)
    # Remove hop-by-hop headers
    hop_by_hop_headers = [
        'connection', 'keep-alive', 'proxy-authenticate',
        'proxy-authorization', 'te', 'trailers', 'transfer-encoding', 'upgrade'
    ]
    for header in hop_by_hop_headers:
        headers.pop(header, None)

    # Add user context if available
    if user:
        headers['X-User-Tier'] = user.tier
        headers['X-User-Username'] = user.username

    # Prepare request data
    body = await request.body()

    async with httpx.AsyncClient(timeout=settings.backend_timeout) as client:
        try:
            response = await client.request(
                method=method,
                url=backend_url,
                headers=headers,
                content=body,
                params=request.query_params
            )

            # Create response with backend data
            response_headers = dict(response.headers)
            # Remove hop-by-hop headers from response
            for header in hop_by_hop_headers:
                response_headers.pop(header, None)

            return Response(
                content=response.content,
                status_code=response.status_code,
                headers=response_headers
            )

        except httpx.TimeoutException:
            logger.error("Backend timeout", url=backend_url)
            raise HTTPException(status_code=504, detail="Backend service timeout")
        except httpx.RequestError as e:
            logger.error("Backend request error", url=backend_url, error=str(e))
            raise HTTPException(status_code=502, detail="Backend service unavailable")

# WebSocket proxy
class WebSocketProxy:
    def __init__(self, backend_ws_url: str):
        self.backend_ws_url = backend_ws_url
        self.active_connections: List[WebSocket] = []

    async def connect(self, websocket: WebSocket, user: Optional[User] = None):
        await websocket.accept()
        self.active_connections.append(websocket)
        ACTIVE_CONNECTIONS.inc()

        try:
            async with httpx.AsyncClient() as client:
                async with client.stream(
                    "GET",
                    self.backend_ws_url,
                    headers={"X-User-Tier": user.tier if user else "anonymous"}
                ) as response:
                    if response.status_code != 101:  # WebSocket upgrade
                        await websocket.send_text("WebSocket upgrade failed")
                        return

                    # Handle bidirectional communication
                    await self._handle_websocket_communication(websocket, response)

        except WebSocketDisconnect:
            logger.info("WebSocket disconnected")
        except Exception as e:
            logger.error("WebSocket error", error=str(e))
        finally:
            if websocket in self.active_connections:
                self.active_connections.remove(websocket)
            ACTIVE_CONNECTIONS.dec()

    async def _handle_websocket_communication(self, websocket: WebSocket, backend_response):
        """Handle bidirectional WebSocket communication"""
        # This is a simplified implementation
        # In production, you'd want to handle both directions simultaneously
        pass

# Health check functions
async def check_backend_health() -> str:
    """Check backend service health"""
    try:
        async with httpx.AsyncClient(timeout=5.0) as client:
            response = await client.get(urljoin(settings.backend_url, "/health"))
            if response.status_code == 200:
                return "healthy"
            else:
                return "unhealthy"
    except Exception:
        return "unreachable"

async def update_backend_status():
    """Periodically update backend status"""
    global backend_status
    while True:
        backend_status = await check_backend_health()
        await asyncio.sleep(30)  # Check every 30 seconds

# Lifespan context manager
@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup
    global redis_client
    try:
        redis_client = redis.from_url(settings.redis_url, max_connections=settings.redis_pool_size)
        await redis_client.ping()
        logger.info("Redis connected successfully")
    except Exception as e:
        logger.error("Failed to connect to Redis", error=str(e))
        redis_client = None

    # Start background tasks
    asyncio.create_task(update_backend_status())

    logger.info("FastAPI Gateway started", host=settings.host, port=settings.port)
    yield

    # Shutdown
    if redis_client:
        await redis_client.close()
    logger.info("FastAPI Gateway shutdown")

# Create FastAPI app
app = FastAPI(
    title=settings.api_title,
    version=settings.api_version,
    description=settings.api_description,
    lifespan=lifespan
)

# Add middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.cors_origins,
    allow_credentials=settings.cors_allow_credentials,
    allow_methods=settings.cors_allow_methods,
    allow_headers=settings.cors_allow_headers,
)

app.add_middleware(TrustedHostMiddleware, allowed_hosts=settings.trusted_hosts)
app.add_middleware(SlowAPIMiddleware)

# Rate limiting exception handler
app.add_exception_handler(RateLimitExceeded, _rate_limit_exceeded_handler)

# WebSocket proxy instance
ws_proxy = WebSocketProxy(urljoin(settings.backend_url.replace("http", "ws"), "/ws"))

# Routes
@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint"""
    return HealthResponse(
        status="healthy",
        timestamp=datetime.utcnow(),
        version=settings.api_version,
        uptime=time.time() - start_time,
        backend_status=backend_status
    )

@app.get("/metrics")
async def metrics():
    """Prometheus metrics endpoint"""
    return Response(
        content=generate_latest(),
        media_type=CONTENT_TYPE_LATEST
    )

@app.post("/auth/login", response_model=Token)
async def login(username: str, password: str):
    """Login endpoint (demo implementation)"""
    # In production, validate against database
    if username == "demo" and password == "demo":
        access_token = create_access_token(
            data={"sub": username},
            expires_delta=timedelta(minutes=settings.access_token_expire_minutes)
        )
        return Token(access_token=access_token)
    else:
        raise HTTPException(status_code=401, detail="Invalid credentials")

# Proxy all other requests to backend
@app.api_route("/{path:path}", methods=["GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"])
async def proxy_all_requests(
    path: str,
    request: Request,
    user: Optional[User] = Depends(authenticate_request)
):
    """Proxy all requests to backend service"""
    start_time_req = time.time()

    try:
        # Check rate limits based on user tier
        if user:
            # Apply tier-specific rate limiting
            tier_limits = {
                "free": 10,
                "pro": 100,
                "enterprise": 1000
            }
            limit = tier_limits.get(user.tier, 10)

            if redis_client:
                # Implement custom rate limiting per user
                key = f"rate_limit:{user.username}"
                current = await redis_client.incr(key)
                if current == 1:
                    await redis_client.expire(key, 60)  # 1 minute window

                if current > limit:
                    REQUEST_COUNT.labels(
                        method=request.method,
                        endpoint=path,
                        status="429"
                    ).inc()
                    raise HTTPException(status_code=429, detail="Rate limit exceeded")

        # Proxy the request
        response = await proxy_request(request.method, path, request, user)

        # Record metrics
        duration = time.time() - start_time_req
        REQUEST_COUNT.labels(
            method=request.method,
            endpoint=path,
            status=str(response.status_code)
        ).inc()
        REQUEST_LATENCY.labels(
            method=request.method,
            endpoint=path
        ).observe(duration)

        # Log request
        logger.info(
            "Request processed",
            method=request.method,
            path=path,
            status=response.status_code,
            duration=duration,
            user=user.username if user else "anonymous"
        )

        return response

    except HTTPException:
        raise
    except Exception as e:
        logger.error(
            "Request processing error",
            method=request.method,
            path=path,
            error=str(e)
        )
        REQUEST_COUNT.labels(
            method=request.method,
            endpoint=path,
            status="500"
        ).inc()
        raise HTTPException(status_code=500, detail="Internal server error")

@app.websocket("/ws")
async def websocket_endpoint(websocket: WebSocket, user: Optional[User] = Depends(authenticate_request)):
    """WebSocket proxy endpoint"""
    await ws_proxy.connect(websocket, user)

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(
        "main:app",
        host=settings.host,
        port=settings.port,
        reload=settings.debug,
        log_level="info" if not settings.debug else "debug"
    )
