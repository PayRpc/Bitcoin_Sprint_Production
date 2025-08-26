#!/usr/bin/env python3
"""
Bitcoin Core RPC Mock Server
Mimics Bitcoin Core RPC interface on port 8332 for testing
"""

import json
import base64
from http.server import ThreadingHTTPServer as HTTPServer, BaseHTTPRequestHandler
from urllib.parse import urlparse
import time
import threading

class BitcoinRPCHandler(BaseHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        self.rpc_user = "bitcoin"
        self.rpc_pass = "sprint123benchmark"
        super().__init__(*args, **kwargs)

    def do_POST(self):
        # Check authentication
        auth_header = self.headers.get('Authorization', '')
        if not auth_header.startswith('Basic '):
            self.send_auth_required()
            return
        
        try:
            encoded_creds = auth_header[6:]  # Remove 'Basic '
            decoded_creds = base64.b64decode(encoded_creds).decode('utf-8')
            username, password = decoded_creds.split(':', 1)
            
            if username != self.rpc_user or password != self.rpc_pass:
                self.send_auth_required()
                return
        except:
            self.send_auth_required()
            return

        # Read request body
        content_length = int(self.headers.get('Content-Length', 0))
        body = self.rfile.read(content_length).decode('utf-8')
        
        try:
            request = json.loads(body)
            response = self.handle_rpc_call(request)
            
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            self.wfile.write(json.dumps(response).encode('utf-8'))
            
        except Exception as e:
            self.send_error(500, f"RPC Error: {str(e)}")

    def send_auth_required(self):
        self.send_response(401)
        self.send_header('WWW-Authenticate', 'Basic realm="Bitcoin RPC"')
        self.end_headers()

    def handle_rpc_call(self, request):
        method = request.get('method', '')
        params = request.get('params', [])
        request_id = request.get('id', 1)

        # Simulate various Bitcoin Core RPC methods
        if method == 'getblockcount':
            return {
                "result": 850000 + int(time.time() % 1000),  # Fake incrementing block height
                "error": None,
                "id": request_id
            }
        
        elif method == 'getblockchaininfo':
            return {
                "result": {
                    "chain": "main",
                    "blocks": 850000 + int(time.time() % 1000),
                    "headers": 850000 + int(time.time() % 1000),
                    "bestblockhash": "0000000000000000000" + "a" * 45,
                    "difficulty": 61030681983.6126,
                    "mediantime": int(time.time()) - 300,
                    "verificationprogress": 0.999999,
                    "initialblockdownload": False,
                    "chainwork": "00000000000000000000000000000000000000007e5dbdad7c8e7e5e1e1e1e1e",
                    "size_on_disk": 500000000000,
                    "pruned": True,
                    "pruneheight": 780000,
                    "softforks": {},
                    "warnings": ""
                },
                "error": None,
                "id": request_id
            }
        
        elif method == 'getnetworkinfo':
            return {
                "result": {
                    "version": 250000,
                    "subversion": "/Satoshi:25.0.0/",
                    "protocolversion": 70016,
                    "localservices": "0000000000000409",
                    "localrelay": True,
                    "timeoffset": 0,
                    "connections": 8,
                    "networkactive": True,
                    "networks": [],
                    "relayfee": 0.00001000,
                    "incrementalfee": 0.00001000,
                    "localaddresses": [],
                    "warnings": ""
                },
                "error": None,
                "id": request_id
            }
        
        elif method == 'getmempoolinfo':
            return {
                "result": {
                    "loaded": True,
                    "size": 2500,
                    "bytes": 1800000,
                    "usage": 5200000,
                    "maxmempool": 300000000,
                    "mempoolminfee": 0.00001000,
                    "minrelaytxfee": 0.00001000
                },
                "error": None,
                "id": request_id
            }
        
        elif method == 'getbestblockhash':
            return {
                "result": "0000000000000000000" + "b" * 45,
                "error": None,
                "id": request_id
            }
        
        elif method == 'ping':
            return {
                "result": None,
                "error": None,
                "id": request_id
            }
        
        elif method == 'uptime':
            return {
                "result": int(time.time() % 86400),  # Fake uptime
                "error": None,
                "id": request_id
            }
        
        else:
            return {
                "result": None,
                "error": {
                    "code": -32601,
                    "message": f"Method not found: {method}"
                },
                "id": request_id
            }

    def log_message(self, format, *args):
        # Suppress default HTTP server logging
        pass

def run_server(port=8332):
    print(f"Starting Bitcoin Core RPC Mock on port {port}")
    print(f"RPC credentials: bitcoin:sprint123benchmark")
    print("Responding to: getblockcount, getblockchaininfo, getnetworkinfo, getmempoolinfo, ping, uptime")
    print("Press Ctrl+C to stop\n")
    
    server = HTTPServer(('127.0.0.1', port), BitcoinRPCHandler)
    
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nShutting down Bitcoin Core mock...")
        server.shutdown()
        server.server_close()

if __name__ == "__main__":
    run_server()
