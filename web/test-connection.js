#!/usr/bin/env node

/**
 * Bitcoin Sprint Web API Connection Test
 * Tests connectivity to the Go backend and validates API endpoints
 */

const https = require('https');
const http = require('http');

const BASE_URL = process.env.GO_API_URL || 'http://localhost:8080';
const API_KEY = process.env.API_KEY || 'test-key';

console.log('🔍 Bitcoin Sprint Web API Connection Test');
console.log('==========================================');
console.log(`Target URL: ${BASE_URL}`);
console.log(`API Key: ${API_KEY ? 'Set' : 'Not Set'}`);
console.log('');

async function testEndpoint(endpoint, method = 'GET', expectAuth = false) {
  return new Promise((resolve) => {
    const url = new URL(endpoint, BASE_URL);
    const protocol = url.protocol === 'https:' ? https : http;

    const options = {
      hostname: url.hostname,
      port: url.port,
      path: url.pathname + url.search,
      method: method,
      headers: {
        'Content-Type': 'application/json',
        ...(expectAuth && API_KEY && { 'Authorization': `Bearer ${API_KEY}` })
      }
    };

    const req = protocol.request(options, (res) => {
      let data = '';

      res.on('data', (chunk) => {
        data += chunk;
      });

      res.on('end', () => {
        try {
          const jsonData = data ? JSON.parse(data) : null;
          resolve({
            status: res.statusCode,
            data: jsonData,
            success: res.statusCode >= 200 && res.statusCode < 300
          });
        } catch (e) {
          resolve({
            status: res.statusCode,
            data: data,
            success: res.statusCode >= 200 && res.statusCode < 300,
            parseError: true
          });
        }
      });
    });

    req.on('error', (err) => {
      resolve({
        status: null,
        error: err.message,
        success: false
      });
    });

    req.setTimeout(5000, () => {
      req.destroy();
      resolve({
        status: null,
        error: 'Timeout after 5 seconds',
        success: false
      });
    });

    req.end();
  });
}

async function runTests() {
  const tests = [
    { name: 'Health Check', endpoint: '/health', method: 'GET', auth: false },
    { name: 'API Status', endpoint: '/api/status', method: 'GET', auth: true },
    { name: 'API Metrics', endpoint: '/api/metrics', method: 'GET', auth: true },
    { name: 'Latest Data', endpoint: '/api/latest', method: 'GET', auth: true },
    { name: 'Predictive Analytics', endpoint: '/api/predictive', method: 'GET', auth: true },
  ];

  console.log('Running connection tests...\n');

  for (const test of tests) {
    process.stdout.write(`Testing ${test.name}... `);

    const result = await testEndpoint(test.endpoint, test.method, test.auth);

    if (result.success) {
      console.log('✅ PASS');
      if (result.data && typeof result.data === 'object') {
        console.log(`   Status: ${result.status}`);
        if (result.data.service) console.log(`   Service: ${result.data.service}`);
        if (result.data.version) console.log(`   Version: ${result.data.version}`);
      }
    } else {
      console.log('❌ FAIL');
      console.log(`   Status: ${result.status || 'N/A'}`);
      if (result.error) console.log(`   Error: ${result.error}`);
      if (result.parseError) console.log(`   Response: ${result.data}`);
    }
    console.log('');
  }

  // Test entropy bridge
  console.log('Testing entropy bridge...');
  try {
    const { isEntropyBridgeAvailable, generateAdminSecret } = await import('./rust-entropy-bridge.js');

    if (isEntropyBridgeAvailable()) {
      console.log('✅ Entropy bridge available');
      const secret = await generateAdminSecret('hex');
      console.log(`   Generated secret: ${secret.substring(0, 16)}...`);
    } else {
      console.log('⚠️  Entropy bridge not available (using Node.js fallback)');
    }
  } catch (error) {
    console.log('❌ Entropy bridge test failed');
    console.log(`   Error: ${error.message}`);
  }

  console.log('\n🎯 Test completed!');
  console.log('\nNext steps:');
  console.log('1. Ensure Go backend is running on the configured URL');
  console.log('2. Set API_KEY environment variable for authenticated endpoints');
  console.log('3. Check network connectivity if tests are failing');
}

runTests().catch(console.error);