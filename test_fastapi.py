#!/usr/bin/env python3
"""
Test script for Bitcoin Sprint FastAPI Gateway
"""

import requests
import json
import time
from typing import Dict, Any

class FastAPIGatewayTester:
    def __init__(self, base_url: str = "http://localhost:8000"):
        self.base_url = base_url
        self.api_keys = {
            "free": "demo-key-free",
            "pro": "demo-key-pro",
            "enterprise": "demo-key-enterprise"
        }

    def make_request(self, endpoint: str, api_key: str) -> Dict[str, Any]:
        """Make a request to the FastAPI gateway"""
        headers = {"Authorization": f"Bearer {api_key}"}
        url = f"{self.base_url}{endpoint}"

        try:
            response = requests.get(url, headers=headers, timeout=10)
            return {
                "status_code": response.status_code,
                "success": response.status_code == 200,
                "data": response.json() if response.headers.get('content-type') == 'application/json' else response.text
            }
        except requests.exceptions.RequestException as e:
            return {
                "status_code": 0,
                "success": False,
                "error": str(e)
            }

    def test_health_check(self) -> None:
        """Test basic health check (no auth required)"""
        print("🩺 Testing Health Check...")
        result = self.make_request("/health", "")

        if result["success"]:
            print("✅ Health check passed")
            print(f"   Response: {result['data']}")
        else:
            print("❌ Health check failed")
            print(f"   Error: {result.get('error', 'Unknown error')}")

    def test_status_endpoint(self, tier: str) -> None:
        """Test status endpoint with authentication"""
        print(f"📊 Testing Status Endpoint ({tier} tier)...")
        api_key = self.api_keys[tier]
        result = self.make_request("/status", api_key)

        if result["success"]:
            print("✅ Status endpoint accessible")
            data = result["data"]
            if isinstance(data, dict) and "data" in data:
                chains = data["data"].get("chains", {})
                for chain, info in chains.items():
                    status = info.get("status", "unknown")
                    peers = info.get("peers", 0)
                    print(f"   {chain}: {status} ({peers} peers)")
        else:
            print("❌ Status endpoint failed")
            print(f"   Status: {result['status_code']}")
            if "error" in result:
                print(f"   Error: {result['error']}")

    def test_rate_limiting(self) -> None:
        """Test rate limiting with free tier"""
        print("⏱️  Testing Rate Limiting...")
        api_key = self.api_keys["free"]

        # Make multiple requests quickly
        rate_limited = False
        for i in range(25):  # Free tier allows 20/minute
            result = self.make_request("/status", api_key)
            if result["status_code"] == 429:  # Too Many Requests
                rate_limited = True
                print(f"✅ Rate limiting working (hit limit at request {i+1})")
                break
            time.sleep(0.1)  # Small delay between requests

        if not rate_limited:
            print("⚠️  Rate limiting may not be working properly")

    def test_invalid_auth(self) -> None:
        """Test invalid authentication"""
        print("🔐 Testing Invalid Authentication...")
        result = self.make_request("/status", "invalid-key")

        if result["status_code"] == 401:
            print("✅ Authentication properly rejecting invalid keys")
        else:
            print("❌ Authentication not working properly")
            print(f"   Expected 401, got {result['status_code']}")

    def run_all_tests(self) -> None:
        """Run all tests"""
        print("=" * 50)
        print("🚀 Bitcoin Sprint FastAPI Gateway Tests")
        print("=" * 50)
        print()

        # Test basic connectivity
        self.test_health_check()
        print()

        # Test different tiers
        for tier in ["free", "pro", "enterprise"]:
            self.test_status_endpoint(tier)
            print()

        # Test security features
        self.test_invalid_auth()
        print()

        # Test rate limiting
        self.test_rate_limiting()
        print()

        print("=" * 50)
        print("✨ Test suite completed!")
        print("=" * 50)

if __name__ == "__main__":
    tester = FastAPIGatewayTester()
    tester.run_all_tests()
