"""
Authentication and authorization for Bitcoin Sprint API Gateway
"""

from typing import Optional
import os

# API Keys configuration
API_KEYS = {
    "free": [
        "demo-key-free",
        "bitcoin-sprint-free-2024"
    ],
    "pro": [
        "demo-key-pro",
        "bitcoin-sprint-pro-2024"
    ],
    "enterprise": [
        "demo-key-enterprise",
        "bitcoin-sprint-enterprise-2024"
    ]
}

def verify_api_key(api_key: str) -> Optional[str]:
    """
    Verify API key and return tier

    Args:
        api_key: The API key to verify

    Returns:
        Tier name if valid, None if invalid
    """
    for tier, keys in API_KEYS.items():
        if api_key in keys:
            return tier
    return None

def get_tier_limits(tier: str) -> dict:
    """
    Get rate limits for a given tier

    Args:
        tier: The tier name

    Returns:
        Dictionary with rate limit settings
    """
    limits = {
        "free": {
            "requests_per_minute": 20,
            "requests_per_hour": 100,
            "concurrent_requests": 2,
            "burst_limit": 5
        },
        "pro": {
            "requests_per_minute": 1000,
            "requests_per_hour": 10000,
            "concurrent_requests": 10,
            "burst_limit": 50
        },
        "enterprise": {
            "requests_per_minute": 10000,
            "requests_per_hour": 100000,
            "concurrent_requests": 100,
            "burst_limit": 500
        }
    }
    return limits.get(tier, limits["free"])

def create_api_key(tier: str, prefix: str = "bitcoin-sprint") -> str:
    """
    Generate a new API key for the given tier

    Args:
        tier: The tier for the new key
        prefix: Prefix for the key

    Returns:
        New API key string
    """
    import secrets
    import string

    # Generate random part
    alphabet = string.ascii_letters + string.digits
    random_part = ''.join(secrets.choice(alphabet) for _ in range(16))

    return f"{prefix}-{tier}-{random_part}"

def validate_request_tier(request_tier: str, required_tier: str) -> bool:
    """
    Check if request tier meets the required tier level

    Args:
        request_tier: The tier from the API key
        required_tier: The minimum required tier

    Returns:
        True if access is allowed
    """
    tier_hierarchy = {
        "free": 1,
        "pro": 2,
        "enterprise": 3
    }

    request_level = tier_hierarchy.get(request_tier, 0)
    required_level = tier_hierarchy.get(required_tier, 0)

    return request_level >= required_level

def get_api_key_info(api_key: str) -> Optional[dict]:
    """
    Get information about an API key

    Args:
        api_key: The API key to look up

    Returns:
        Dictionary with key information or None if not found
    """
    tier = verify_api_key(api_key)
    if not tier:
        return None

    return {
        "key": api_key,
        "tier": tier,
        "limits": get_tier_limits(tier),
        "valid": True
    }
