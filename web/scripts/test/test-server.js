#!/usr/bin/env node

/**
 * Bitcoin Sprint Web Server Test
 * Tests the Next.js application server and its endpoints
 */

const http = require('http');

const SERVER_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3002';

console.log('🚀 Bitcoin Sprint Web Server Test');
console.log('==================================');
console.log(`Server URL: ${SERVER_URL}`);
console.log('');

async function testWebEndpoint(endpoint, method = 'GET') {
  return new Promise((resolve) => {
    const url = new URL(endpoint, SERVER_URL);

    const options = {
      hostname: url.hostname,
      port: url.port,
      path: url.pathname + url.search,
      method: method,
      headers: {
        'Content-Type': 'application/json',
        'User-Agent': 'Bitcoin-Sprint-Test/1.0'
      }
    };

    const req = http.request(options, (res) => {
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
            success: res.statusCode >= 200 && res.statusCode < 300,
            headers: res.headers
          });
        } catch (e) {
          resolve({
            status: res.statusCode,
            data: data,
            success: res.statusCode >= 200 && res.statusCode < 300,
            parseError: true,
            headers: res.headers
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

    req.setTimeout(10000, () => {
      req.destroy();
      resolve({
        status: null,
        error: 'Timeout after 10 seconds',
        success: false
      });
    });

    req.end();
  });
}

async function runWebTests() {
  const tests = [
    { name: 'Home Page', endpoint: '/', method: 'GET' },
    { name: 'API Health', endpoint: '/api/health', method: 'GET' },
    { name: 'Dashboard', endpoint: '/dashboard', method: 'GET' },
    { name: 'Signup Page', endpoint: '/signup', method: 'GET' },
    { name: 'Maintenance Status', endpoint: '/api/maintenance', method: 'GET' },
  ];

  console.log('Testing web server endpoints...\n');

  for (const test of tests) {
    process.stdout.write(`Testing ${test.name}... `);

    const result = await testWebEndpoint(test.endpoint, test.method);

    if (result.success) {
      console.log('✅ PASS');
      console.log(`   Status: ${result.status}`);
      if (result.headers['content-type']) {
        console.log(`   Content-Type: ${result.headers['content-type']}`);
      }
    } else {
      console.log('❌ FAIL');
      console.log(`   Status: ${result.status || 'N/A'}`);
      if (result.error) console.log(`   Error: ${result.error}`);
    }
    console.log('');
  }

  // Test API endpoints that require authentication
  console.log('Testing authenticated endpoints...\n');

  const authTests = [
    { name: 'API Status', endpoint: '/api/status', method: 'GET' },
    { name: 'API Metrics', endpoint: '/api/metrics', method: 'GET' },
    { name: 'Latest Data', endpoint: '/api/latest', method: 'GET' },
  ];

  for (const test of authTests) {
    process.stdout.write(`Testing ${test.name}... `);

    const result = await testWebEndpoint(test.endpoint, test.method);

    if (result.success) {
      console.log('✅ PASS (may require API key)');
      console.log(`   Status: ${result.status}`);
    } else if (result.status === 401) {
      console.log('⚠️  AUTH REQUIRED');
      console.log(`   Status: ${result.status} (Expected for unauthenticated request)`);
    } else {
      console.log('❌ FAIL');
      console.log(`   Status: ${result.status || 'N/A'}`);
      if (result.error) console.log(`   Error: ${result.error}`);
    }
    console.log('');
  }

  console.log('🎯 Web server test completed!');
  console.log('\nNext steps:');
  console.log('1. Start the Next.js server: npm run dev');
  console.log('2. Ensure backend is running for API endpoints');
  console.log('3. Set API_KEY for authenticated endpoints');
  console.log('4. Check browser at http://localhost:3002');
}

runWebTests().catch(console.error);
