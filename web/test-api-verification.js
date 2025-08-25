import { extractTierFromPrefix, validateApiKeyFormat, verifyApiKey } from './lib/generateKey.js';

/**
 * Test the API key verification system
 */
async function testApiKeyVerification() {
  console.log('=== API Key Verification Tests ===\n');

  // Test 1: Format validation
  console.log('1. Format Validation Tests:');
  console.log('----------------------------');
  
  const formatTests = [
    'sprint_validKeyWith43CharsExactly123456789012345', // Valid
    'sprint-pro_validKeyWith43CharsExactly123456789012', // Valid tier-specific
    'invalid-format-without-underscore', // Invalid: no underscore
    'sprint_', // Invalid: empty random part
    'sprint_short', // Invalid: too short
    'SPRINT_validKeyWith43CharsExactly123456789012345', // Invalid: uppercase prefix
    'unknown_validKeyWith43CharsExactly123456789012345', // Invalid: unknown prefix
  ];

  formatTests.forEach((token, i) => {
    const result = validateApiKeyFormat(token);
    console.log(`Test ${i + 1}: ${result.valid ? '✅ PASS' : '❌ FAIL'} - ${token.substring(0, 30)}...`);
    if (!result.valid) {
      console.log(`   Reason: ${result.reason}`);
    } else {
      console.log(`   Prefix: ${result.prefix}`);
    }
  });

  // Test 2: Tier extraction
  console.log('\n2. Tier Extraction Tests:');
  console.log('---------------------------');
  
  const tierTests = ['sprint', 'sprint-free', 'sprint-pro', 'sprint-ent', 'sprint-entplus'];
  tierTests.forEach(prefix => {
    const tier = extractTierFromPrefix(prefix);
    console.log(`${prefix} → ${tier || 'Generic'}`);
  });

  // Test 3: Database verification (using a real key from database)
  console.log('\n3. Database Verification Test:');
  console.log('-------------------------------');
  
  try {
    // This would test with a real key from the database
    const testKey = 'sprint-free_testKeyThatDoesNotExistInDatabase123456';
    const verification = await verifyApiKey(testKey);
    
    console.log(`Test key: ${testKey.substring(0, 30)}...`);
    console.log(`Valid: ${verification.valid}`);
    console.log(`Reason: ${verification.reason || 'Key is valid'}`);
    
    if (verification.valid && verification.apiKey) {
      console.log(`Email: ${verification.apiKey.email}`);
      console.log(`Tier: ${verification.tier}`);
      console.log(`Expires: ${verification.apiKey.expiresAt}`);
      console.log(`Requests: ${verification.apiKey.requests}`);
    }
  } catch (error) {
    console.log(`Database test failed: ${error}`);
  }

  console.log('\n=== Verification System Ready! ===');
  console.log('\nUsage Examples:');
  console.log('----------------');
  console.log('// In your API routes:');
  console.log('const verification = await verifyApiKey(token);');
  console.log('if (!verification.valid) {');
  console.log('  return res.status(401).json({ error: verification.reason });');
  console.log('}');
  console.log('');
  console.log('// Update usage statistics:');
  console.log('await updateApiKeyUsage(token, true); // increment blocks');
  console.log('');
  console.log('// With middleware:');
  console.log('export default withApiKeyAuth(handler, { requiredTier: "PRO" });');
}

// Run tests if this file is executed directly
if (require.main === module) {
  testApiKeyVerification().catch(console.error);
}

export { testApiKeyVerification };
