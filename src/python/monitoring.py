"""
Monitoring and metrics for Bitcoin Sprint API Gateway
"""

from fastapi import FastAPI, Request
from prometheus_client import Counter, Histogram, Gauge, generate_latest
import time
from typing import Dict, Any

# Prometheus metrics
REQUEST_COUNT = Counter(
    'api_requests_total',
    'Total number of API requests',
    ['method', 'endpoint', 'tier', 'status']
)

REQUEST_LATENCY = Histogram(
    'api_request_duration_seconds',
    'Request duration in seconds',
    ['method', 'endpoint', 'tier']
)

ACTIVE_CONNECTIONS = Gauge(
    'api_active_connections',
    'Number of active connections'
)

RATE_LIMIT_HITS = Counter(
    'api_rate_limit_hits_total',
    'Total number of rate limit hits',
    ['tier']
)

def setup_prometheus(app: FastAPI):
    """Setup Prometheus metrics for the FastAPI app"""

    @app.get("/metrics")
    async def metrics():
        """Prometheus metrics endpoint"""
        return generate_latest()

def track_request(request: Request, endpoint: str):
    """
    Track an API request for monitoring

    Args:
        request: FastAPI request object
        endpoint: The endpoint being accessed
    """
    # This would be called from middleware in a real implementation
    # For now, it's a placeholder for request tracking
    pass

def increment_request_count(method: str, endpoint: str, tier: str, status: int):
    """
    Increment request counter

    Args:
        method: HTTP method
        endpoint: API endpoint
        tier: API key tier
        status: HTTP status code
    """
    REQUEST_COUNT.labels(
        method=method,
        endpoint=endpoint,
        tier=tier,
        status=str(status)
    ).inc()

def observe_request_latency(method: str, endpoint: str, tier: str, duration: float):
    """
    Record request latency

    Args:
        method: HTTP method
        endpoint: API endpoint
        tier: API key tier
        duration: Request duration in seconds
    """
    REQUEST_LATENCY.labels(
        method=method,
        endpoint=endpoint,
        tier=tier
    ).observe(duration)

def increment_rate_limit_hit(tier: str):
    """
    Increment rate limit hit counter

    Args:
        tier: API key tier that hit the rate limit
    """
    RATE_LIMIT_HITS.labels(tier=tier).inc()

def update_active_connections(count: int):
    """
    Update active connections gauge

    Args:
        count: Number of active connections
    """
    ACTIVE_CONNECTIONS.set(count)

class RequestTracker:
    """Middleware for tracking requests"""

    def __init__(self):
        self.active_requests = 0

    async def __call__(self, request: Request, call_next):
        start_time = time.time()
        method = request.method
        url = str(request.url.path)

        # Track active connections
        self.active_requests += 1
        update_active_connections(self.active_requests)

        try:
            response = await call_next(request)
            status_code = response.status_code

            # Track metrics
            duration = time.time() - start_time

            # Extract tier from request (this would need to be implemented)
            tier = getattr(request.state, 'tier', 'unknown')

            increment_request_count(method, url, tier, status_code)
            observe_request_latency(method, url, tier, duration)

            return response

        except Exception as e:
            # Track failed requests
            duration = time.time() - start_time
            tier = getattr(request.state, 'tier', 'unknown')
            increment_request_count(method, url, tier, 500)
            observe_request_latency(method, url, tier, duration)
            raise e

        finally:
            # Decrement active connections
            self.active_requests -= 1
            update_active_connections(self.active_requests)

# Global request tracker
request_tracker = RequestTracker()
