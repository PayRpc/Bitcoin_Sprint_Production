"""
Pydantic models for Bitcoin Sprint API Gateway
"""

from pydantic import BaseModel
from typing import Dict, Any, Optional
from datetime import datetime

class ChainStatus(BaseModel):
    """Status of a blockchain connection"""
    status: str
    peers: int
    message: str
    ready: Optional[bool] = None
    last_attempt: Optional[str] = None
    connection_status: Optional[str] = None
    protocol_note: Optional[str] = None
    bootstrap_nodes: Optional[list] = None

class SystemStatus(BaseModel):
    """Overall system status"""
    server_status: str
    gateway_version: str
    backend_status: str
    uptime: str
    chains: Dict[str, ChainStatus]
    sla_assessment: Dict[str, Any]
    system_health: Dict[str, Any]
    timestamp: str

class StatusResponse(BaseModel):
    """Response wrapper for status endpoint"""
    success: bool
    data: SystemStatus
    tier: str
    message: Optional[str] = None

class ReadinessResponse(BaseModel):
    """Response wrapper for readiness endpoint"""
    success: bool
    data: Dict[str, Any]
    tier: str
    message: Optional[str] = None

class MetricsResponse(BaseModel):
    """Response wrapper for metrics endpoint"""
    success: bool
    data: str
    tier: str

class APIKey(BaseModel):
    """API key information"""
    key: str
    tier: str
    created: datetime
    expires: Optional[datetime] = None
    active: bool = True

class TierLimits(BaseModel):
    """Rate limits for different tiers"""
    requests_per_minute: int
    requests_per_hour: int
    concurrent_requests: int
    burst_limit: int

class ErrorResponse(BaseModel):
    """Error response model"""
    success: bool = False
    error: str
    code: int
    timestamp: str
    tier: Optional[str] = None
