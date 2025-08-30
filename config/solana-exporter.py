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
import sys

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Solana RPC configuration 
# Use localhost for direct testing
SOLANA_RPC_URLS = [
    os.getenv("SOLANA_RPC_URL", "http://localhost:8899"),
    os.getenv("SOLANA_RPC_URL_2", "http://localhost:8901"),
    os.getenv("SOLANA_RPC_URL_3", "http://localhost:8903"),
    os.getenv("SOLANA_RPC_URL_4", "http://localhost:8905"),
    os.getenv("SOLANA_RPC_URL_5", "http://localhost:8907"),
    os.getenv("SOLANA_RPC_URL_6", "http://localhost:8909"),
    os.getenv("SOLANA_RPC_URL_7", "http://localhost:8911"),
]

# Prometheus metrics
SLOT_HEIGHT = Gauge('solana_slot_height', 'Current slot height')
BLOCK_HEIGHT = Gauge('solana_block_height', 'Current block height')
TRANSACTION_COUNT = Gauge('solana_transaction_count', 'Total transaction count')
TPS = Gauge('solana_tps', 'Transactions per second')
VALIDATOR_COUNT = Gauge('solana_validator_count', 'Number of active validators')
NETWORK_LATENCY = Gauge('solana_network_latency_ms', 'Network latency in milliseconds')
CONFIRMATION_TIME = Gauge('solana_confirmation_time_ms', 'Average confirmation time')

# Multi-validator metrics
VALIDATOR_HEALTH = Gauge('solana_validator_health', 'Health status of individual validators', ['validator_id'])
VALIDATOR_SLOT_HEIGHT = Gauge('solana_validator_slot_height', 'Slot height per validator', ['validator_id'])
VALIDATOR_BLOCK_HEIGHT = Gauge('solana_validator_block_height', 'Block height per validator', ['validator_id'])

# Storage metrics
ACCOUNTS_DB_SIZE = Gauge('solana_accounts_db_size_bytes', 'Size of accounts database in bytes')
LEDGER_SIZE = Gauge('solana_ledger_size_bytes', 'Size of ledger in bytes')
SNAPSHOT_SIZE = Gauge('solana_snapshot_size_bytes', 'Size of snapshots in bytes')

def rpc_call(method, params=[], rpc_url=None):
    """Make RPC call to Solana"""
    if rpc_url is None:
        rpc_url = SOLANA_RPC_URLS[0]  # Default to first validator
    
    payload = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": method,
        "params": params
    }

    try:
        start_time = time.time()
        response = requests.post(rpc_url, json=payload, timeout=10)
        latency = (time.time() - start_time) * 1000  # Convert to milliseconds

        response.raise_for_status()
        result = response.json()

        # Update latency metric
        NETWORK_LATENCY.set(latency)

        return result.get('result')
    except Exception as e:
        logger.error(f"RPC call failed for {rpc_url}: {e}")
        return None

def collect_storage_metrics():
    """Collect storage-related metrics from all validators"""
    try:
        # Data paths for all validators (mounted volumes in containers)
        # Since all validators mount to the same internal path, we need to check
        # if we're running inside a validator container or the exporter container
        # For now, let's check the common Solana data paths
        data_paths_to_check = [
            "/root/.config/solana",  # Container internal path
            "/solana/data",          # Alternative path
        ]

        total_accounts_size = 0
        total_ledger_size = 0
        total_snapshot_size = 0

        for data_path in data_paths_to_check:
            if os.path.exists(data_path):
                logger.info(f"Found Solana data directory: {data_path}")

                # Calculate accounts DB size
                accounts_path = os.path.join(data_path, "accounts")
                if os.path.exists(accounts_path):
                    accounts_size = get_directory_size(accounts_path)
                    total_accounts_size += accounts_size
                    logger.info(f"Accounts DB size: {accounts_size} bytes")
                else:
                    logger.warning(f"Accounts path not found: {accounts_path}")

                # Calculate ledger size
                ledger_path = os.path.join(data_path, "ledger")
                if os.path.exists(ledger_path):
                    ledger_size = get_directory_size(ledger_path)
                    total_ledger_size += ledger_size
                    logger.info(f"Ledger size: {ledger_size} bytes")
                else:
                    logger.warning(f"Ledger path not found: {ledger_path}")

                # Calculate snapshot size
                snapshot_path = os.path.join(data_path, "snapshots")
                if os.path.exists(snapshot_path):
                    snapshot_size = get_directory_size(snapshot_path)
                    total_snapshot_size += snapshot_size
                    logger.info(f"Snapshot size: {snapshot_size} bytes")
                else:
                    logger.warning(f"Snapshot path not found: {snapshot_path}")

        # Set total metrics
        ACCOUNTS_DB_SIZE.set(total_accounts_size)
        LEDGER_SIZE.set(total_ledger_size)
        SNAPSHOT_SIZE.set(total_snapshot_size)

        logger.info(f"Total storage metrics - Accounts: {total_accounts_size}, Ledger: {total_ledger_size}, Snapshots: {total_snapshot_size}")

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
    """Collect all Solana metrics from multiple validators"""
    try:
        # Collect storage metrics first
        collect_storage_metrics()

        max_slot_height = 0
        max_block_height = 0
        total_validator_count = 0
        total_transaction_count = 0

        # Query each validator
        for i, rpc_url in enumerate(SOLANA_RPC_URLS):
            validator_id = f"validator_{i+1}"

            try:
                # Get slot height
                slot_result = rpc_call("getSlot", rpc_url=rpc_url)
                if slot_result is not None:
                    slot_height = int(slot_result)
                    VALIDATOR_SLOT_HEIGHT.labels(validator_id=validator_id).set(slot_height)
                    max_slot_height = max(max_slot_height, slot_height)
                    VALIDATOR_HEALTH.labels(validator_id=validator_id).set(1)  # Healthy
                    logger.info(f"Validator {validator_id} slot height: {slot_height}")
                else:
                    VALIDATOR_HEALTH.labels(validator_id=validator_id).set(0)  # Unhealthy

                # Get block height
                block_result = rpc_call("getBlockHeight", rpc_url=rpc_url)
                if block_result is not None:
                    block_height = int(block_result)
                    VALIDATOR_BLOCK_HEIGHT.labels(validator_id=validator_id).set(block_height)
                    max_block_height = max(max_block_height, block_height)
                    logger.info(f"Validator {validator_id} block height: {block_height}")

                # Get validator count from this validator
                validators_result = rpc_call("getVoteAccounts", rpc_url=rpc_url)
                if validators_result and 'current' in validators_result:
                    validator_count = len(validators_result['current'])
                    total_validator_count = max(total_validator_count, validator_count)  # Use max to avoid double counting

                # Get transaction count (approximate from recent block)
                if slot_result:
                    try:
                        block_data = rpc_call("getConfirmedBlock", [slot_result - 1], rpc_url=rpc_url)
                        if block_data and 'transactions' in block_data:
                            tx_count = len(block_data['transactions'])
                            total_transaction_count += tx_count
                    except:
                        pass

            except Exception as e:
                logger.error(f"Error querying validator {validator_id}: {e}")
                VALIDATOR_HEALTH.labels(validator_id=validator_id).set(0)

        # Set aggregated metrics
        SLOT_HEIGHT.set(max_slot_height)
        BLOCK_HEIGHT.set(max_block_height)
        VALIDATOR_COUNT.set(total_validator_count)
        TRANSACTION_COUNT.set(total_transaction_count)

        logger.info(f"Aggregated - Slot: {max_slot_height}, Block: {max_block_height}, Validators: {total_validator_count}")

        # Calculate TPS (simplified - would need more sophisticated tracking for accurate TPS)
        TPS.set(0)

        # Set confirmation time (placeholder - would need more complex tracking)
        CONFIRMATION_TIME.set(0)

    except Exception as e:
        logger.error(f"Error collecting metrics: {e}")

def main():
    """Main function"""
    logger.info("Starting Solana Core Prometheus Exporter")
    logger.info(f"Connecting to Solana RPC endpoints: {', '.join(SOLANA_RPC_URLS)}")

    try:
        # Start Prometheus metrics server
        start_http_server(8080)
        logger.info("Metrics server started on port 8080")
        
        # First collection immediately
        collect_metrics()
        
        # Output diagnostic info to help debug
        logger.info(f"Successfully completed first metrics collection")
        logger.info(f"System: {sys.platform}, Python: {sys.version}")
        
        # Collect metrics every 30 seconds
        while True:
            time.sleep(30)
            collect_metrics()
            
    except Exception as e:
        logger.error(f"Fatal error in main loop: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
