const crypto = require("crypto");

/**
 * Generate a 256-bit random API key with product prefix.
 */
function generateApiKey(prefix = "sprint") {
  const randomBytes = crypto.randomBytes(32);
  const randomPart = randomBytes.toString("base64url");
  return `${prefix}_${randomPart}`;
}

/**
 * Generate API key with tier-specific prefix for operational visibility.
 */
function generateTierApiKey(tier) {
  const tierPrefixes = {
    'FREE': 'sprint-free',
    'PRO': 'sprint-pro', 
    'ENTERPRISE': 'sprint-ent',
    'ENTERPRISE_PLUS': 'sprint-entplus'
  };
  
  const prefix = tierPrefixes[tier] || 'sprint';
  return generateApiKey(prefix);
}

console.log('=== Enhanced API Key Generation Test ===');

console.log('\n1. Basic API Key Generation:');
console.log('Default:', generateApiKey());
console.log('Custom prefix:', generateApiKey('custom'));

console.log('\n2. Tier-Specific API Keys:');
console.log('FREE:', generateTierApiKey('FREE'));
console.log('PRO:', generateTierApiKey('PRO'));
console.log('ENTERPRISE:', generateTierApiKey('ENTERPRISE'));
console.log('ENTERPRISE_PLUS:', generateTierApiKey('ENTERPRISE_PLUS'));

console.log('\n3. Key Analysis:');
const key = generateTierApiKey('ENTERPRISE');
console.log('Sample key:', key);
console.log('Total length:', key.length);
console.log('Prefix:', key.split('_')[0]);
console.log('Random part length:', key.split('_')[1].length);

console.log('\n4. Operational Benefits:');
console.log('✓ Grep logs: grep "sprint-pro" logs.txt');
console.log('✓ Tier identification: Key prefix shows tier immediately');
console.log('✓ Security: Full 256-bit entropy preserved');
console.log('✓ URL-safe: Works in query params and headers');

console.log('\n5. Consistency Test (Multiple Keys):');
['FREE', 'PRO', 'ENTERPRISE'].forEach(tier => {
  const testKey = generateTierApiKey(tier);
  console.log(`${tier}: ${testKey.substring(0, 25)}... (${testKey.length} chars)`);
});
