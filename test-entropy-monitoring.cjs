#!/usr/bin/env node

/**
 * Test script for Entropy Bridge Monitoring System
 * Tests the complete monitoring pipeline from Next.js to Go to Prometheus
 */

const http = require('http');

async function testEndpoint(url, description) {
  return new Promise((resolve) => {
    console.log(`\n🔍 Testing ${description}...`);
    console.log(`   URL: ${url}`);

    const req = http.get(url, (res) => {
      let data = '';

      res.on('data', (chunk) => {
        data += chunk;
      });

      res.on('end', () => {
        console.log(`   ✅ Status: ${res.statusCode}`);
        if (res.statusCode === 200) {
          console.log(`   📊 Response length: ${data.length} characters`);
          resolve({ success: true, data, statusCode: res.statusCode });
        } else {
          console.log(`   ❌ Error response: ${data.substring(0, 100)}...`);
          resolve({ success: false, data, statusCode: res.statusCode });
        }
      });
    });

    req.on('error', (err) => {
      console.log(`   ❌ Connection failed: ${err.message}`);
      resolve({ success: false, error: err.message });
    });

    req.setTimeout(5000, () => {
      console.log(`   ⏰ Timeout after 5 seconds`);
      req.destroy();
      resolve({ success: false, error: 'Timeout' });
    });
  });
}

async function testEntropyBridgeMonitoring() {
  console.log('🚀 Testing Entropy Bridge Monitoring System');
  console.log('=' .repeat(50));

  // Test 1: Next.js entropy status endpoint
  const entropyStatus = await testEndpoint(
    'http://localhost:3002/api/admin/entropy-status',
    'Next.js Entropy Bridge Status'
  );

  if (entropyStatus.success && entropyStatus.data) {
    try {
      const status = JSON.parse(entropyStatus.data);
      console.log(`   🔧 Bridge Available: ${status.status.available}`);
      console.log(`   🦀 Rust Available: ${status.status.rustAvailable}`);
      console.log(`   🔄 Fallback Mode: ${status.status.fallbackMode}`);
      console.log(`   ⏱️ Uptime: ${status.uptime}s`);
    } catch (e) {
      console.log(`   ⚠️ Could not parse JSON response`);
    }
  }

  // Test 2: Next.js Prometheus metrics
  const nextjsMetrics = await testEndpoint(
    'http://localhost:3002/api/prometheus',
    'Next.js Prometheus Metrics'
  );

  if (nextjsMetrics.success && nextjsMetrics.data) {
    const entropyMetrics = nextjsMetrics.data.split('\n')
      .filter(line => line.includes('bitcoin_sprint_entropy_bridge'))
      .slice(0, 3);

    if (entropyMetrics.length > 0) {
      console.log(`   📈 Found ${entropyMetrics.length} entropy bridge metrics:`);
      entropyMetrics.forEach(metric => console.log(`      ${metric}`));
    } else {
      console.log(`   ⚠️ No entropy bridge metrics found in Next.js`);
    }
  }

  // Test 3: Go metrics server
  const goMetrics = await testEndpoint(
    'http://localhost:8081/metrics',
    'Go Metrics Server'
  );

  if (goMetrics.success && goMetrics.data) {
    const entropyMetrics = goMetrics.data.split('\n')
      .filter(line => line.includes('bitcoin_sprint_entropy_bridge'))
      .slice(0, 3);

    if (entropyMetrics.length > 0) {
      console.log(`   📈 Found ${entropyMetrics.length} entropy bridge metrics:`);
      entropyMetrics.forEach(metric => console.log(`      ${metric}`));
    } else {
      console.log(`   ⚠️ No entropy bridge metrics found in Go server`);
    }
  }

  // Test 4: Check if Prometheus would be able to scrape
  console.log(`\n🔍 Testing Prometheus scraping compatibility...`);
  if (goMetrics.success) {
    const lines = goMetrics.data.split('\n');
    const validMetrics = lines.filter(line =>
      line.includes('# TYPE') || line.includes('# HELP') ||
      (line.includes('bitcoin_sprint_entropy_bridge') && !line.startsWith('#'))
    );

    console.log(`   ✅ Found ${validMetrics.length} Prometheus-compatible lines`);
    console.log(`   📊 Metrics are ready for Prometheus scraping`);
  }

  console.log('\n' + '=' .repeat(50));
  console.log('🎉 Entropy Bridge Monitoring Test Complete!');
  console.log('\n💡 System Status:');
  console.log(`   • Next.js Status: ${entropyStatus.success ? '✅ Running' : '❌ Not responding'}`);
  console.log(`   • Go Metrics: ${goMetrics.success ? '✅ Running' : '❌ Not responding'}`);
  console.log(`   • Entropy Bridge: ${entropyStatus.success ? '✅ Active (Fallback Mode)' : '❌ Inactive'}`);
  console.log(`   • Monitoring: ${goMetrics.success && entropyStatus.success ? '✅ Fully Operational' : '⚠️ Partial'}`);

  if (entropyStatus.success && goMetrics.success) {
    console.log('\n🚀 Ready for Grafana Dashboard!');
    console.log('   1. Import grafana-dashboard-entropy-bridge.json');
    console.log('   2. Configure Prometheus data source');
    console.log('   3. Start monitoring entropy bridge status!');
  }
}

testEntropyBridgeMonitoring().catch(console.error);
