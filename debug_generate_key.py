#!/usr/bin/env python3
"""
Debug the generate-key endpoint
"""

import requests
import json

def main():
    print('🔍 Debugging /generate-key endpoint')

    try:
        response = requests.post('http://localhost:8000/generate-key',
                               json={'tier': 'free'}, timeout=5)
        print(f'Status Code: {response.status_code}')
        print(f'Content-Type: {response.headers.get("content-type")}')
        print(f'Raw Response: {repr(response.text)}')

        if response.text:
            try:
                data = response.json()
                print(f'Parsed JSON: {data}')
            except Exception as e:
                print(f'JSON Parse Error: {e}')
        else:
            print('Empty response body')

    except Exception as e:
        print(f'Request Error: {e}')

if __name__ == '__main__':
    main()
