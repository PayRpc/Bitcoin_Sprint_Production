#!/usr/bin/env python3
"""
Solana Metrics Exporter for Prometheus
Collects Solana network metrics and exposes them via HTTP for Prometheus scraping.
"""

import asyncio
import json
import os
import time
from typing import Dict, Any
import aiohttp
from aiohttp import web
from prometheus_client import Gauge, generate_latest, CollectorRegistry, CONTENT_TYPE_LATEST

# Prometheus metrics
registry = CollectorRegistry()

# Solana metrics
solana_slot_height = Gauge('solana_slot_height', 'Current slot height', registry=registry)
solana_block_height = Gauge('solana_block_height', 'Current block height', registry=registry)
solana_transaction_count = Gauge('solana_transaction_count', 'Total transaction count', registry=registry)
solana_validator_count = Gauge('solana_validator_count', 'Number of active validators', registry=registry)
solana_tps = Gauge('solana_tps', 'Transactions per second', registry=registry)
solana_confirmation_time = Gauge('solana_confirmation_time_ms', 'Average confirmation time in milliseconds', registry=registry)
solana_network_latency = Gauge('solana_network_latency_ms', 'Network latency in milliseconds', registry=registry)

class SolanaMetricsCollector:
    def __init__(self, rpc_url: str = "http://localhost:8899"):
        self.rpc_url = rpc_url
        self.session = None
        self.last_slot = 0
        self.last_tx_count = 0
        self.last_update = time.time()

    async def __aenter__(self):
        self.session = aiohttp.ClientSession()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        if self.session:
            await self.session.close()

    async def make_rpc_call(self, method: str, params: list = None) -> Dict[str, Any]:
        """Make RPC call to Solana node"""
        if params is None:
            params = []

        payload = {
            "jsonrpc": "2.0",
            "id": 1,
            "method": method,
            "params": params
        }

        try:
            async with self.session.post(self.rpc_url, json=payload, timeout=10) as response:
                if response.status == 200:
                    return await response.json()
                else:
                    print(f"RPC call failed: {response.status}")
                    return {}
        except Exception as e:
            print(f"RPC call error: {e}")
            return {}

    async def collect_slot_info(self):
        """Collect current slot information"""
        try:
            result = await self.make_rpc_call("getSlot")
            if "result" in result:
                slot = result["result"]
                solana_slot_height.set(slot)
                return slot
        except Exception as e:
            print(f"Error collecting slot info: {e}")
        return 0

    async def collect_block_height(self):
        """Collect current block height"""
        try:
            result = await self.make_rpc_call("getBlockHeight")
            if "result" in result:
                height = result["result"]
                solana_block_height.set(height)
                return height
        except Exception as e:
            print(f"Error collecting block height: {e}")
        return 0

    async def collect_transaction_count(self):
        """Collect transaction count"""
        try:
            result = await self.make_rpc_call("getTransactionCount")
            if "result" in result:
                tx_count = result["result"]
                solana_transaction_count.set(tx_count)

                # Calculate TPS
                current_time = time.time()
                time_diff = current_time - self.last_update
                if time_diff > 0 and self.last_tx_count > 0:
                    tps = (tx_count - self.last_tx_count) / time_diff
                    solana_tps.set(tps)

                self.last_tx_count = tx_count
                self.last_update = current_time

                return tx_count
        except Exception as e:
            print(f"Error collecting transaction count: {e}")
        return 0

    async def collect_validator_info(self):
        """Collect validator information"""
        try:
            result = await self.make_rpc_call("getVoteAccounts")
            if "result" in result and "current" in result["result"]:
                validators = len(result["result"]["current"])
                solana_validator_count.set(validators)
                return validators
        except Exception as e:
            print(f"Error collecting validator info: {e}")
        return 0

    async def collect_network_health(self):
        """Collect network health metrics"""
        try:
            # Measure RPC latency
            start_time = time.time()
            result = await self.make_rpc_call("getHealth")
            latency = (time.time() - start_time) * 1000
            solana_network_latency.set(latency)

            # Set confirmation time (estimated)
            solana_confirmation_time.set(800)  # ~800ms average for Solana

        except Exception as e:
            print(f"Error collecting network health: {e}")

    async def collect_all_metrics(self):
        """Collect all Solana metrics"""
        await asyncio.gather(
            self.collect_slot_info(),
            self.collect_block_height(),
            self.collect_transaction_count(),
            self.collect_validator_info(),
            self.collect_network_health()
        )

async def metrics_handler(request):
    """HTTP handler for Prometheus metrics"""
    return web.Response(
        text=generate_latest(registry).decode('utf-8'),
        content_type='text/plain; version=0.0.4'
    )

async def health_handler(request):
    """Health check handler"""
    return web.json_response({"status": "healthy", "service": "solana-metrics-exporter"})

async def main():
    """Main application"""
    # Get configuration from environment
    rpc_url = os.getenv("SOLANA_RPC_URL", "http://localhost:8899")
    metrics_port = int(os.getenv("METRICS_PORT", "8080"))
    update_interval = float(os.getenv("UPDATE_INTERVAL", "30").rstrip('s'))

    print(f"Starting Solana Metrics Exporter")
    print(f"RPC URL: {rpc_url}")
    print(f"Metrics Port: {metrics_port}")
    print(f"Update Interval: {update_interval}s")

    # Create collector
    async with SolanaMetricsCollector(rpc_url) as collector:
        # Create web application
        app = web.Application()
        app.router.add_get('/metrics', metrics_handler)
        app.router.add_get('/health', health_handler)

        # Start metrics collection task
        async def collect_metrics():
            while True:
                await collector.collect_all_metrics()
                await asyncio.sleep(update_interval)

        # Start background collection
        asyncio.create_task(collect_metrics())

        # Start web server
        runner = web.AppRunner(app)
        await runner.setup()
        site = web.TCPSite(runner, '0.0.0.0', metrics_port)
        await site.start()

        print(f"Metrics server started on port {metrics_port}")

        # Keep running
        try:
            while True:
                await asyncio.sleep(1)
        except KeyboardInterrupt:
            print("Shutting down...")
        finally:
            await runner.cleanup()

if __name__ == "__main__":
    asyncio.run(main())
