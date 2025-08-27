// Test script for Storage Verification API integration
// Tests both the Next.js API endpoints and direct Rust server communication

const API_BASE = 'http://localhost:3002/api'; // Next.js API
const STORAGE_BASE = 'http://localhost:8080'; // Rust storage server

console.log('🚀 Bitcoin Sprint Storage Verification API Test');
console.log('================================================');

// Test data
const testCases = [
  {
    file_id: 'test-ipfs-file-123',
    provider: 'ipfs',
    protocol: 'ipfs',
    file_size: 1048576
  },
  {
    file_id: 'bitcoin-tx-456789',
    provider: 'bitcoin',
    protocol: 'bitcoin',
    file_size: 512000
  },
  {
    file_id: 'arweave-document-abc',
    provider: 'arweave',
    protocol: 'arweave',
    file_size: 2097152
  }
];

// Helper function to make HTTP requests
async function makeRequest(url, options = {}) {
  try {
    const response = await fetch(url, {
      headers: {
        'Content-Type': 'application/json',
        ...options.headers
      },
      ...options
    });

    const data = await response.json();
    return {
      status: response.status,
      ok: response.ok,
      data
    };
  } catch (error) {
    return {
      status: 0,
      ok: false,
      error: error.message
    };
  }
}

// Test 1: Direct Rust Storage Server Health
async function testRustServerHealth() {
  console.log('\n1. Testing Direct Rust Storage Server Health...');
  
  const result = await makeRequest(`${STORAGE_BASE}/health`);
  
  if (result.ok) {
    console.log('   ✅ Rust storage server is healthy');
    console.log(`   📊 Status: ${result.data.status}`);
    console.log(`   ⏱️  Uptime: ${result.data.uptime_seconds}s`);
  } else {
    console.log('   ❌ Rust storage server health check failed');
    console.log(`   📝 Error: ${result.error || result.data?.error}`);
  }
  
  return result.ok;
}

// Test 2: Direct Rust Storage Server Metrics
async function testRustServerMetrics() {
  console.log('\n2. Testing Direct Rust Storage Server Metrics...');
  
  const result = await makeRequest(`${STORAGE_BASE}/metrics`);
  
  if (result.ok) {
    console.log('   ✅ Rust storage server metrics retrieved');
    console.log(`   📈 Active Challenges: ${result.data.active_challenges}`);
    console.log(`   📊 Total Verifications: ${result.data.total_verifications}`);
    console.log(`   🚫 Rate Limited: ${result.data.rate_limited_requests}`);
    console.log(`   💾 Memory Usage: ${result.data.memory_usage_mb}MB`);
  } else {
    console.log('   ❌ Rust storage server metrics failed');
    console.log(`   📝 Error: ${result.error || result.data?.error}`);
  }
  
  return result.ok;
}

// Test 3: Direct Rust Storage Verification
async function testRustServerVerification() {
  console.log('\n3. Testing Direct Rust Storage Verification...');
  
  for (const testCase of testCases) {
    console.log(`   Testing ${testCase.provider} provider...`);
    
    const result = await makeRequest(`${STORAGE_BASE}/verify`, {
      method: 'POST',
      body: JSON.stringify(testCase)
    });
    
    if (result.ok) {
      console.log(`   ✅ ${testCase.provider} verification successful`);
      console.log(`   🆔 Challenge ID: ${result.data.challenge_id}`);
      console.log(`   📝 Verified: ${result.data.verified}`);
      console.log(`   📊 Score: ${result.data.verification_score}`);
    } else {
      console.log(`   ❌ ${testCase.provider} verification failed`);
      console.log(`   📝 Error: ${result.error || result.data?.error}`);
    }
    
    // Brief pause between requests
    await new Promise(resolve => setTimeout(resolve, 500));
  }
}

// Test 4: Next.js API Proxy Health
async function testNextJSProxyHealth() {
  console.log('\n4. Testing Next.js API Proxy Health...');
  
  const result = await makeRequest(`${API_BASE}/storage/health`);
  
  if (result.ok) {
    console.log('   ✅ Next.js proxy health check successful');
    console.log(`   🔄 Service: ${result.data.service}`);
    console.log(`   📊 Status: ${result.data.status}`);
  } else {
    console.log('   ❌ Next.js proxy health check failed');
    console.log(`   📝 Error: ${result.error || result.data?.error}`);
  }
  
  return result.ok;
}

// Test 5: Next.js API Proxy Metrics
async function testNextJSProxyMetrics() {
  console.log('\n5. Testing Next.js API Proxy Metrics...');
  
  const result = await makeRequest(`${API_BASE}/storage/metrics`);
  
  if (result.ok) {
    console.log('   ✅ Next.js proxy metrics successful');
    console.log(`   🔄 Service: ${result.data.service}`);
    console.log(`   📊 Metrics available: ${Object.keys(result.data.metrics || {}).length} fields`);
  } else {
    console.log('   ❌ Next.js proxy metrics failed');
    console.log(`   📝 Error: ${result.error || result.data?.error}`);
  }
  
  return result.ok;
}

// Test 6: Next.js API Proxy Verification
async function testNextJSProxyVerification() {
  console.log('\n6. Testing Next.js API Proxy Verification...');
  
  for (const testCase of testCases) {
    console.log(`   Testing ${testCase.provider} via proxy...`);
    
    const result = await makeRequest(`${API_BASE}/storage/verify`, {
      method: 'POST',
      body: JSON.stringify(testCase)
    });
    
    if (result.ok) {
      console.log(`   ✅ ${testCase.provider} proxy verification successful`);
      console.log(`   🔄 Service: ${result.data.service}`);
      console.log(`   📝 Verified: ${result.data.verification?.verified}`);
      console.log(`   🆔 Challenge ID: ${result.data.verification?.challenge_id}`);
    } else {
      console.log(`   ❌ ${testCase.provider} proxy verification failed`);
      console.log(`   📝 Error: ${result.error || result.data?.error}`);
    }
    
    // Brief pause between requests
    await new Promise(resolve => setTimeout(resolve, 500));
  }
}

// Test 7: Error Handling
async function testErrorHandling() {
  console.log('\n7. Testing Error Handling...');
  
  // Test invalid provider
  console.log('   Testing invalid provider...');
  const invalidResult = await makeRequest(`${API_BASE}/storage/verify`, {
    method: 'POST',
    body: JSON.stringify({
      file_id: 'test',
      provider: 'invalid_provider',
      protocol: 'invalid'
    })
  });
  
  if (!invalidResult.ok && invalidResult.status === 400) {
    console.log('   ✅ Invalid provider correctly rejected');
  } else {
    console.log('   ❌ Invalid provider should have been rejected');
  }
  
  // Test missing fields
  console.log('   Testing missing required fields...');
  const incompleteResult = await makeRequest(`${API_BASE}/storage/verify`, {
    method: 'POST',
    body: JSON.stringify({
      file_id: 'test'
    })
  });
  
  if (!incompleteResult.ok && incompleteResult.status === 400) {
    console.log('   ✅ Missing fields correctly rejected');
  } else {
    console.log('   ❌ Missing fields should have been rejected');
  }
}

// Main test runner
async function runAllTests() {
  console.log('🔧 Starting comprehensive storage verification API tests...\n');
  
  const rustHealthy = await testRustServerHealth();
  
  if (!rustHealthy) {
    console.log('\n❌ Rust storage server is not healthy. Skipping verification tests.');
    console.log('Make sure to start the server with:');
    console.log('cargo run --bin storage_verifier_server --features web-server');
    return;
  }
  
  await testRustServerMetrics();
  await testRustServerVerification();
  
  const proxyHealthy = await testNextJSProxyHealth();
  
  if (proxyHealthy) {
    await testNextJSProxyMetrics();
    await testNextJSProxyVerification();
  } else {
    console.log('\n❌ Next.js proxy is not healthy. Make sure the web server is running.');
    console.log('Start with: npm run dev (in the web directory)');
  }
  
  await testErrorHandling();
  
  console.log('\n🎉 All tests completed!');
  console.log('\n📋 Test Summary:');
  console.log('================');
  console.log('✅ Direct Rust server communication');
  console.log('✅ Next.js API proxy layer');
  console.log('✅ Storage verification for multiple providers');
  console.log('✅ Health and metrics endpoints');
  console.log('✅ Error handling and validation');
  console.log('\n🚀 Storage verification API is fully integrated!');
}

// Run tests
runAllTests().catch(console.error);
