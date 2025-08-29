# turbo_api_integration.py - FastAPI Integration for Turbo Validation
# Wire turbo validation into your production API gateway

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import Optional, List
import requests
import time
import logging

app = FastAPI(title="Sprint Turbo API Gateway", version="1.0.0")

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Turbo validation service URL (your Rust backend)
TURBO_VALIDATOR_URL = "http://127.0.0.1:8082"

class TurboResults(BaseModel):
    avg_latency_ns: float
    throughput: float
    iterations: int
    safety_factor: float
    passed: bool
    timestamp: int
    execution_count: int

class TurboStatus(BaseModel):
    turbo_validation: TurboResults
    status: str
    validation_score: str

class ValidationHistory(BaseModel):
    executions: List[str]
    validation_history: List[str]
    status: str
    validation_score: str

@app.get("/turbo-status")
async def get_turbo_status():
    """
    Get current turbo validation status
    Returns real-time validation results from the Rust backend
    """
    try:
        # Fetch from Rust backend
        response = requests.get(f"{TURBO_VALIDATOR_URL}/turbo-validation", timeout=5)
        response.raise_for_status()

        data = response.json()
        return TurboStatus(**data)

    except requests.RequestException as e:
        logger.error(f"Failed to fetch turbo status: {e}")
        raise HTTPException(status_code=503, detail="Turbo validator unavailable")

@app.get("/turbo-validation")
async def get_turbo_validation():
    """
    Get comprehensive turbo validation results
    """
    try:
        # Fetch from Rust backend
        response = requests.get(f"{TURBO_VALIDATOR_URL}/turbo", timeout=5)
        response.raise_for_status()

        data = response.json()
        return ValidationHistory(**data)

    except requests.RequestException as e:
        logger.error(f"Failed to fetch turbo validation: {e}")
        raise HTTPException(status_code=503, detail="Turbo validator unavailable")

@app.get("/health")
async def health_check():
    """
    Health check that includes turbo validation status
    """
    try:
        # Check turbo validator health
        turbo_response = requests.get(f"{TURBO_VALIDATOR_URL}/turbo-status", timeout=2)

        if turbo_response.status_code == 200:
            return {
                "status": "healthy",
                "turbo_validator": "active",
                "timestamp": int(time.time())
            }
        else:
            return {
                "status": "degraded",
                "turbo_validator": "inactive",
                "timestamp": int(time.time())
            }

    except requests.RequestException:
        return {
            "status": "unhealthy",
            "turbo_validator": "unreachable",
            "timestamp": int(time.time())
        }

@app.get("/metrics")
async def get_metrics():
    """
    Proxy Prometheus metrics from the Rust backend
    """
    try:
        response = requests.get(f"{TURBO_VALIDATOR_URL}/metrics", timeout=5)
        response.raise_for_status()

        return response.text

    except requests.RequestException as e:
        logger.error(f"Failed to fetch metrics: {e}")
        raise HTTPException(status_code=503, detail="Metrics unavailable")

# PRODUCTION: Startup validation
@app.on_event("startup")
async def startup_validation():
    """
    Run turbo validation on startup to ensure system is ready
    """
    logger.info("🚀 Starting Sprint Turbo API Gateway...")

    try:
        # Wait for turbo validator to be ready
        max_retries = 10
        for i in range(max_retries):
            try:
                response = requests.get(f"{TURBO_VALIDATOR_URL}/turbo-status", timeout=2)
                if response.status_code == 200:
                    logger.info("✅ Turbo validator is active")
                    break
            except requests.RequestException:
                if i == max_retries - 1:
                    logger.error("❌ Turbo validator failed to start")
                    raise
                logger.warning(f"Waiting for turbo validator... ({i+1}/{max_retries})")
                time.sleep(1)

        # Log startup validation
        response = requests.get(f"{TURBO_VALIDATOR_URL}/turbo-validation")
        if response.status_code == 200:
            data = response.json()
            turbo = data.get("turbo_validation", {})
            logger.info(f"🚀 Turbo Validation: {turbo.get('avg_latency_ns', 0):.2f}ns latency, "
                       f"{turbo.get('throughput', 0):.0f} ops/sec, "
                       f"Status: {'PASSED' if turbo.get('validation_passed', False) else 'FAILED'}")

    except Exception as e:
        logger.error(f"❌ Startup validation failed: {e}")
        # In production, you might want to exit here
        # sys.exit(1)

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
