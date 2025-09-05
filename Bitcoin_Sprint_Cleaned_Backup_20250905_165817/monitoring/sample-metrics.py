#!/usr/bin/env python3
"""
Simple Prometheus metrics exporter script that simulates a Solana exporter.
Run this to provide test metrics for Grafana dashboards.
"""
import time
import random
from http.server import HTTPServer, BaseHTTPRequestHandler

# Sample metrics text that matches what's expected by the dashboard
METRICS_TEXT = """# HELP python_gc_objects_collected_total Objects collected during gc
# TYPE python_gc_objects_collected_total counter
python_gc_objects_collected_total{generation="0"} 58793.0
python_gc_objects_collected_total{generation="1"} 6800.0
python_gc_objects_collected_total{generation="2"} 310.0

# HELP solana_slot_height Current slot height
# TYPE solana_slot_height gauge
solana_slot_height %d

# HELP solana_block_height Current block height
# TYPE solana_block_height gauge
solana_block_height %d

# HELP solana_validator_count Number of active validators
# TYPE solana_validator_count gauge
solana_validator_count 981.0

# HELP solana_network_latency_ms Network latency in milliseconds
# TYPE solana_network_latency_ms gauge
solana_network_latency_ms %f

# HELP solana_validator_health Health status of individual validators
# TYPE solana_validator_health gauge
solana_validator_health{validator_id="validator_1"} 1.0
solana_validator_health{validator_id="validator_2"} 1.0
solana_validator_health{validator_id="validator_5"} 1.0

# HELP solana_validator_slot_height Slot height per validator
# TYPE solana_validator_slot_height gauge
solana_validator_slot_height{validator_id="validator_1"} %d
solana_validator_slot_height{validator_id="validator_5"} %d

# HELP solana_tps Transactions per second
# TYPE solana_tps gauge
solana_tps %f
"""

class MetricsHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/metrics':
            # Generate slightly varying values to create a nice graph
            slot_height = 364774181 + int(time.time() % 100) 
            block_height = 342949681 + int(time.time() % 50)
            latency = 50 + random.random() * 50
            tps = 1000 + random.random() * 500
            
            # Format the metrics text with current values
            metrics = METRICS_TEXT % (
                slot_height, 
                block_height, 
                latency,
                slot_height - 100,
                slot_height,
                tps
            )
            
            self.send_response(200)
            self.send_header('Content-Type', 'text/plain')
            self.end_headers()
            self.wfile.write(metrics.encode())
        else:
            self.send_response(404)
            self.end_headers()
            self.wfile.write(b'Not Found')

if __name__ == '__main__':
    server = HTTPServer(('0.0.0.0', 8080), MetricsHandler)
    print('Starting metrics server on port 8080...')
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print('Stopping server...')
        server.server_close()
