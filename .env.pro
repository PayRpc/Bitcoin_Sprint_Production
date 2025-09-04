# Bitcoin Sprint Configuration - PRO TIER
# Professional tier with moderate resources and performance

# Activate Pro Tier
TIER=pro

# API Configuration
API_HOST=0.0.0.0
API_PORT=8082

# Web Dashboard Configuration
WEB_HOST=0.0.0.0
WEB_PORT=3002

# Bitcoin Node Configuration
BITCOIN_NODE=127.0.0.1:8333
ZMQ_NODE=127.0.0.1:28332

# P2P Configuration
PEER_LISTEN_PORT=8335
PEER_HMAC_SECRET=bitcoin-sprint-pro-secret-key-2025

# Security & Authentication
LICENSE_KEY=PRO-TIER-STANDARD-FEATURES
API_KEY=pro-api-key-changeme

# Performance Settings (Pro Tier)
USE_DIRECT_P2P=true
USE_MEMORY_CHANNEL=false
OPTIMIZE_SYSTEM=true
ENABLE_KERNEL_BYPASS=false

# Rate Limiting (Pro Tier)
RATE_LIMIT_REQUESTS_PER_SECOND=10
RATE_LIMIT_REQUESTS_PER_HOUR=10000
CONCURRENT_STREAMS=5
DATA_SIZE_LIMIT_MB=100

# Monitoring (Enhanced)
ENABLE_PROMETHEUS=true
PROMETHEUS_PORT=9090
ENABLE_ENTROPY_MONITORING=true
ENTROPY_METRICS_INTERVAL=60s

# Database (Standard)
DATABASE_TYPE=sqlite
DATABASE_URL=./pro_tier.db
ENABLE_PERSISTENCE=true

# Logging
LOG_LEVEL=info
LOG_FILE=logs/pro_tier.log
