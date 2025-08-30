#!/usr/bin/env python3
"""
Solana Core Prometheus Exporter
Queries Solana RPC and exposes metrics for Prometheus
"""

import json
import time
import requests
from prometheus_client import start_http_server, Gauge, Counter
import logging
import os
import glob
import pathlib

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Solana RPC configuration
SOLANA_RPC_URL = os.getenv("SOLANA_RPC_URL", "http://solana-validator:8899")

# Prometheus metrics
SLOT_HEIGHT = Gauge('solana_slot_height', 'Current slot height')
BLOCK_HEIGHT = Gauge('solana_block_height', 'Current block height')
TRANSACTION_COUNT = Gauge('solana_transaction_count', 'Total transaction count')
TPS = Gauge('solana_tps', 'Transactions per second')
VALIDATOR_COUNT = Gauge('solana_validator_count', 'Number of active validators')
NETWORK_LATENCY = Gauge('solana_network_latency_ms', 'Network latency in milliseconds')
CONFIRMATION_TIME = Gauge('solana_confirmation_time_ms', 'Average confirmation time')

# Storage metrics
ACCOUNTS_DB_SIZE = Gauge('solana_accounts_db_size_bytes', 'Size of accounts database in bytes')
LEDGER_SIZE = Gauge('solana_ledger_size_bytes', 'Size of ledger in bytes')
SNAPSHOT_SIZE = Gauge('solana_snapshot_size_bytes', 'Size of snapshots in bytes')

def rpc_call(method, params=[]):
    """Make RPC call to Solana"""
    payload = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": method,
        "params": params
    }

    try:
        start_time = time.time()
        response = requests.post(SOLANA_RPC_URL, json=payload, timeout=10)
        latency = (time.time() - start_time) * 1000  # Convert to milliseconds

        response.raise_for_status()
        result = response.json()

        # Update latency metric
        NETWORK_LATENCY.set(latency)

        return result.get('result')
    except Exception as e:
        logger.error(f"RPC call failed: {e}")
        return None

def collect_storage_metrics():
    """Collect storage-related metrics"""
    try:
        # Try to access Solana data directory
        # Common paths in containerized environments
        possible_paths = [
            "/solana/data",  # Mounted data directory
            "/root/.config/solana",  # Default Solana config
            "/var/solana/data"  # Alternative data path
        ]

        data_path = None
        for path in possible_paths:
            if os.path.exists(path):
                data_path = path
                break

        if data_path:
            logger.info(f"Found Solana data directory: {data_path}")

            # Calculate accounts DB size
            accounts_path = os.path.join(data_path, "accounts")
            if os.path.exists(accounts_path):
                accounts_size = get_directory_size(accounts_path)
                ACCOUNTS_DB_SIZE.set(accounts_size)
                logger.info(f"Accounts DB size: {accounts_size} bytes")
            else:
                ACCOUNTS_DB_SIZE.set(0)

            # Calculate ledger size
            ledger_path = os.path.join(data_path, "ledger")
            if os.path.exists(ledger_path):
                ledger_size = get_directory_size(ledger_path)
                LEDGER_SIZE.set(ledger_size)
                logger.info(f"Ledger size: {ledger_size} bytes")
            else:
                LEDGER_SIZE.set(0)

            # Calculate snapshot size
            snapshot_path = os.path.join(data_path, "snapshots")
            if os.path.exists(snapshot_path):
                snapshot_size = get_directory_size(snapshot_path)
                SNAPSHOT_SIZE.set(snapshot_size)
                logger.info(f"Snapshot size: {snapshot_size} bytes")
            else:
                SNAPSHOT_SIZE.set(0)
        else:
            logger.warning("Solana data directory not found, setting storage metrics to 0")
            ACCOUNTS_DB_SIZE.set(0)
            LEDGER_SIZE.set(0)
            SNAPSHOT_SIZE.set(0)

    except Exception as e:
        logger.error(f"Error collecting storage metrics: {e}")
        ACCOUNTS_DB_SIZE.set(0)
        LEDGER_SIZE.set(0)
        SNAPSHOT_SIZE.set(0)

def get_directory_size(path):
    """Calculate total size of directory recursively"""
    total_size = 0
    try:
        for dirpath, dirnames, filenames in os.walk(path):
            for filename in filenames:
                filepath = os.path.join(dirpath, filename)
                try:
                    total_size += os.path.getsize(filepath)
                except OSError:
                    pass  # Skip files we can't access
    except Exception as e:
        logger.warning(f"Error calculating directory size for {path}: {e}")
    return total_size

def collect_metrics():
    """Collect all Solana metrics"""
    try:
        # Collect storage metrics first
        collect_storage_metrics()

        # Get slot height
        slot_result = rpc_call("getSlot")
        if slot_result is not None:
            SLOT_HEIGHT.set(slot_result)
            logger.info(f"Slot height: {slot_result}")

        # Get block height
        block_result = rpc_call("getBlockHeight")
        if block_result is not None:
            BLOCK_HEIGHT.set(block_result)
            logger.info(f"Block height: {block_result}")

        # Get transaction count (approximate from recent block)
        if slot_result:
            try:
                block_data = rpc_call("getConfirmedBlock", [slot_result - 1])
                if block_data and 'transactions' in block_data:
                    tx_count = len(block_data['transactions'])
                    TRANSACTION_COUNT.set(tx_count)
                    logger.info(f"Transaction count in recent block: {tx_count}")
            except:
                pass

        # Get validator count
        validators_result = rpc_call("getVoteAccounts")
        if validators_result and 'current' in validators_result:
            validator_count = len(validators_result['current'])
            VALIDATOR_COUNT.set(validator_count)
            logger.info(f"Active validators: {validator_count}")

        # Calculate TPS (simplified - would need more sophisticated tracking for accurate TPS)
        # For now, just set a placeholder
        TPS.set(0)

        # Set confirmation time (placeholder - would need more complex tracking)
        CONFIRMATION_TIME.set(0)

    except Exception as e:
        logger.error(f"Error collecting metrics: {e}")

def main():
    """Main function"""
    logger.info("Starting Solana Core Prometheus Exporter")
    logger.info(f"Connecting to Solana RPC at: {SOLANA_RPC_URL}")

    # Start Prometheus metrics server
    start_http_server(8080)
    logger.info("Metrics server started on port 8080")

    # Collect metrics every 30 seconds
    while True:
        collect_metrics()
        time.sleep(30)

if __name__ == "__main__":
    main()
