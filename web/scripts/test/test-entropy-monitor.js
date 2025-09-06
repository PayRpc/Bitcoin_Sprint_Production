#!/usr/bin/env node

/**
 * Entropy Monitor Test Script
 * Tests the entropy monitoring setup and metrics collection
 */

async function testEntropyMonitoring() {
  console.log('🔐 Bitcoin Sprint Entropy Monitor Test');
  console.log('=====================================');
  console.log('');

  try {
    // Test Prometheus metrics endpoint
    console.log('Testing Prometheus metrics endpoint...');
    const response = await fetch('http://localhost:3002/api/prometheus');
    if (response.ok) {
      const metrics = await response.text();
      console.log('✅ Prometheus metrics endpoint accessible');

      // Check for entropy metrics
      const entropyMetrics = metrics.split('\n').filter(line =>
        line.includes('bitcoin_sprint_entropy')
      );

      console.log(`📊 Found ${entropyMetrics.length} entropy-related metrics`);
      if (entropyMetrics.length > 0) {
        console.log('✅ Entropy metrics are being collected');
        entropyMetrics.slice(0, 5).forEach(metric => {
          console.log(`   ${metric.split('{')[0]}`);
        });
        if (entropyMetrics.length > 5) {
          console.log(`   ... and ${entropyMetrics.length - 5} more`);
        }
      } else {
        console.log('⚠️  No entropy metrics found yet - they will appear after first API call');
      }
    } else {
      console.log('❌ Prometheus metrics endpoint not accessible');
    }

    console.log('');

    // Test entropy status endpoint
    console.log('Testing entropy status endpoint...');
    const statusResponse = await fetch('http://localhost:3002/api/admin/entropy-status');
    if (statusResponse.ok) {
      const status = await statusResponse.json();
      console.log('✅ Entropy status endpoint accessible');
      console.log(`   Bridge Available: ${status.entropy_bridge.available}`);
      console.log(`   Rust Available: ${status.entropy_bridge.rust_available}`);
      console.log(`   Fallback Mode: ${status.entropy_bridge.fallback_mode}`);
      console.log(`   Test Secret Length: ${status.entropy_bridge.test_secret_length}`);
    } else {
      console.log('❌ Entropy status endpoint not accessible');
    }

    console.log('');
    console.log('🎯 Entropy monitoring setup test completed!');
    console.log('');
    console.log('📋 Next Steps:');
    console.log('1. Start Grafana: docker-compose up grafana');
    console.log('2. Access Grafana at: http://localhost:3000');
    console.log('3. Navigate to "Bitcoin Sprint - Entropy Bridge Monitor" dashboard');
    console.log('4. Monitor entropy metrics in real-time');

  } catch (error) {
    console.error('❌ Entropy monitoring test failed:', error.message);
    console.log('');
    console.log('💡 Troubleshooting:');
    console.log('1. Make sure the web application is running: npm run dev');
    console.log('2. Check that Prometheus is running and accessible');
    console.log('3. Verify Grafana configuration and dashboard provisioning');
  }
}

// Run the test
testEntropyMonitoring().catch(console.error);
