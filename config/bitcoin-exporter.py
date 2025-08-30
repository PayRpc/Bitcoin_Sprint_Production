#!/usr/bin/env python3
"""
Bitcoin Core Prometheus Exporter
Queries Bitcoin Core RPC and exposes metrics for Prometheus
"""

import json
import time
import requests
from prometheus_client import start_http_server, Gauge, Counter
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Bitcoin RPC configuration
RPC_HOST = "bitcoin-core"  # Use container hostname
RPC_PORT = 8332
RPC_USER = "sprint"
RPC_PASSWORD = "sprint_password_2025"

# Prometheus metrics
BLOCK_HEIGHT = Gauge('bitcoin_block_height', 'Current block height')
BLOCKCHAIN_SIZE = Gauge('bitcoin_blockchain_size_bytes', 'Blockchain size in bytes')
MEMPOOL_SIZE = Gauge('bitcoin_mempool_size', 'Number of transactions in mempool')
PEER_COUNT = Gauge('bitcoin_peer_count', 'Number of connected peers')
VERIFICATION_PROGRESS = Gauge('bitcoin_verification_progress', 'Blockchain verification progress (0-1)')
UPTIME = Gauge('bitcoin_uptime_seconds', 'Bitcoin Core uptime in seconds')
DIFFICULTY = Gauge('bitcoin_difficulty', 'Current mining difficulty')

def rpc_call(method, params=[]):
    """Make RPC call to Bitcoin Core"""
    url = f"http://{RPC_HOST}:{RPC_PORT}"
    headers = {'content-type': 'application/json'}
    payload = {
        "jsonrpc": "2.0",
        "method": method,
        "params": params,
        "id": 1
    }

    try:
        response = requests.post(url, json=payload, auth=(RPC_USER, RPC_PASSWORD), timeout=10)
        response.raise_for_status()
        return response.json()['result']
    except Exception as e:
        logger.error(f"RPC call failed: {e}")
        return None

def collect_metrics():
    """Collect metrics from Bitcoin Core"""
    try:
        # Get blockchain info
        blockchain_info = rpc_call("getblockchaininfo")
        if blockchain_info:
            BLOCK_HEIGHT.set(blockchain_info.get('blocks', 0))
            VERIFICATION_PROGRESS.set(blockchain_info.get('verificationprogress', 0))
            DIFFICULTY.set(blockchain_info.get('difficulty', 0))
            logger.info(f"Block height: {blockchain_info.get('blocks', 0)}")

        # Get network info
        network_info = rpc_call("getnetworkinfo")
        if network_info:
            PEER_COUNT.set(network_info.get('connections', 0))

        # Get mempool info
        mempool_info = rpc_call("getmempoolinfo")
        if mempool_info:
            MEMPOOL_SIZE.set(mempool_info.get('size', 0))

        # Get uptime (approximate)
        uptime_info = rpc_call("uptime")
        if uptime_info:
            UPTIME.set(uptime_info)

    except Exception as e:
        logger.error(f"Error collecting metrics: {e}")

def main():
    """Main function"""
    logger.info("Starting Bitcoin Core Prometheus Exporter")

    # Start Prometheus metrics server
    start_http_server(8080)
    logger.info("Metrics server started on port 8080")

    # Collect metrics every 30 seconds
    while True:
        collect_metrics()
        time.sleep(30)

if __name__ == "__main__":
    main()
