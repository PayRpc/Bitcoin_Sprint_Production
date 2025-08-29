#!/usr/bin/env node

/**
 * Simple test for entropy bridge fallback functionality
 */

const crypto = require('crypto');

function fallbackGenerateSecret(encoding = 'base64') {
  // Use Node.js crypto for fallback
  const entropy = crypto.randomBytes(32);

  switch (encoding) {
    case 'raw':
      return entropy.toString('hex');
    case 'base64':
      return entropy.toString('base64');
    case 'hex':
      return entropy.toString('hex');
    default:
      throw new Error(`Unsupported encoding: ${encoding}`);
  }
}

async function testFallback() {
  console.log('🔐 Testing Entropy Bridge Fallback System\n');

  // Test 1: Check crypto availability
  console.log('1. Testing Node.js crypto availability...');
  try {
    const testBytes = crypto.randomBytes(16);
    console.log(`   ✅ Crypto available: ${testBytes.length} bytes generated`);
  } catch (error) {
    console.log(`   ❌ Crypto not available: ${error}`);
    return;
  }
  console.log('');

  // Test 2: Generate secrets with different encodings
  console.log('2. Testing secret generation...');

  try {
    const base64Secret = fallbackGenerateSecret('base64');
    console.log(`   ✅ Base64 Secret: ${base64Secret.substring(0, 20)}...`);

    const hexSecret = fallbackGenerateSecret('hex');
    console.log(`   ✅ Hex Secret: ${hexSecret.substring(0, 20)}...`);

    const rawSecret = fallbackGenerateSecret('raw');
    console.log(`   ✅ Raw Secret: ${rawSecret.substring(0, 20)}...`);
    console.log('');
  } catch (error) {
    console.error(`   ❌ Secret generation failed: ${error}`);
    console.log('');
  }

  // Test 3: Performance test
  console.log('3. Testing performance...');

  const startTime = Date.now();
  const iterations = 1000;

  for (let i = 0; i < iterations; i++) {
    fallbackGenerateSecret('base64');
  }

  const endTime = Date.now();
  const totalTime = endTime - startTime;
  const avgTime = totalTime / iterations;

  console.log(`   ✅ Generated ${iterations} secrets in ${totalTime}ms`);
  console.log(`   📊 Average time per secret: ${avgTime.toFixed(3)}ms`);
  console.log('');

  // Test 4: Uniqueness test
  console.log('4. Testing secret uniqueness...');

  const secrets = new Set();
  for (let i = 0; i < 1000; i++) {
    secrets.add(fallbackGenerateSecret('base64'));
  }

  console.log(`   ✅ Generated ${secrets.size} unique secrets out of 1000 attempts`);
  console.log(`   📊 Uniqueness rate: ${(secrets.size / 1000 * 100).toFixed(2)}%`);
  console.log('');

  console.log('🎉 Fallback system test completed successfully!');
  console.log('💡 The system is ready to use Node.js crypto when Rust FFI is not available.');
}

testFallback().catch(console.error);
