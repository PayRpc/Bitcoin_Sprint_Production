#!/usr/bin/env python3
"""
Bitcoin Sprint FastAPI Gateway - Quick Start Guide
"""

import os
import sys

def print_header():
    print("=" * 60)
    print("🚀 BITCOIN SPRINT FASTAPI GATEWAY")
    print("=" * 60)
    print()

def print_setup_instructions():
    print("📦 SETUP INSTRUCTIONS")
    print("-" * 30)
    print("1. Install dependencies:")
    print("   pip install -r requirements.txt")
    print()
    print("2. Start your Go backend:")
    print("   cd ../cmd/sprintd")
    print("   go run main.go")
    print()
    print("3. Start FastAPI gateway:")
    print("   python app.py")
    print("   # Or use the PowerShell script:")
    print("   # .\\start-fastapi.ps1")
    print()

def print_api_endpoints():
    print("🔗 API ENDPOINTS")
    print("-" * 30)
    print("Gateway:     http://localhost:8000")
    print("Docs:        http://localhost:8000/docs")
    print("Health:      http://localhost:8000/health")
    print("Status:      http://localhost:8000/status")
    print("Readiness:   http://localhost:8000/readiness")
    print("Metrics:     http://localhost:8000/metrics (Enterprise)")
    print()

def print_api_keys():
    print("🔑 API KEYS FOR TESTING")
    print("-" * 30)
    print("Free Tier:       demo-key-free")
    print("Pro Tier:        demo-key-pro")
    print("Enterprise:      demo-key-enterprise")
    print()
    print("Usage: curl -H \"Authorization: Bearer demo-key-free\" \\")
    print("            http://localhost:8000/status")
    print()

def print_features():
    print("✨ KEY FEATURES")
    print("-" * 30)
    print("✅ Customer-facing API with clean endpoints")
    print("✅ Authentication & tier management")
    print("✅ Rate limiting (20/min free, 1000/min pro)")
    print("✅ Auto-generated Swagger documentation")
    print("✅ Prometheus metrics & monitoring")
    print("✅ CORS support for web applications")
    print("✅ Proxy gateway to Go backend")
    print("✅ Production-ready with proper error handling")
    print()

def print_architecture():
    print("🏗️  ARCHITECTURE")
    print("-" * 30)
    print("[Client] → [FastAPI Gateway:8000] → [Go Backend:8080]")
    print("    ↓              ↓")
    print("[Auth]         [Bitcoin RPC]")
    print("[Rate Limit]   [ZMQ Streams]")
    print("[Monitoring]   [P2P Networks]")
    print()

def print_testing():
    print("🧪 TESTING")
    print("-" * 30)
    print("Run the test suite:")
    print("python test_fastapi.py")
    print()
    print("Manual testing:")
    print("curl http://localhost:8000/health")
    print("curl -H \"Authorization: Bearer demo-key-free\" \\")
    print("     http://localhost:8000/status | jq .")
    print()

def print_production_notes():
    print("🏭 PRODUCTION DEPLOYMENT")
    print("-" * 30)
    print("1. Set environment variables:")
    print("   export GO_BACKEND_URL='http://your-backend:8080'")
    print("   export REDIS_URL='redis://your-redis:6379'")
    print()
    print("2. Use production API keys")
    print("3. Run with Gunicorn:")
    print("   gunicorn app:app -w 4 -k uvicorn.workers.UvicornWorker")
    print()

def main():
    print_header()
    print_setup_instructions()
    print_api_endpoints()
    print_api_keys()
    print_features()
    print_architecture()
    print_testing()
    print_production_notes()

    print("=" * 60)
    print("🎯 READY TO LAUNCH!")
    print("=" * 60)
    print()
    print("Your FastAPI gateway is now ready to provide a professional,")
    print("customer-facing API for your Bitcoin Sprint infrastructure!")
    print()
    print("📚 For detailed documentation, see: FASTAPI_README.md")

if __name__ == "__main__":
    main()
