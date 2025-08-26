// Turbo Mode Tier Testing Script
const fs = require('fs');
const fetch = require('node-fetch'); // <-- fix: use node-fetch for compatibility

const BASE_URL = 'http://localhost:3002';

// Test tiers and their expected turbo mode status
const TIER_TESTS = [
    { tier: 'FREE', expectedTurbo: false },
    { tier: 'PRO', expectedTurbo: false },
    { tier: 'ENTERPRISE', expectedTurbo: true },
    { tier: 'ENTERPRISE_PLUS', expectedTurbo: true }
];

// Helper with timeout support
async function fetchWithTimeout(url, options = {}, timeout = 5000) {
    const controller = new AbortController();
    const id = setTimeout(() => controller.abort(), timeout);
    try {
        const res = await fetch(url, { ...options, signal: controller.signal });
        return res;
    } finally {
        clearTimeout(id);
    }
}

async function testTier(tierConfig) {
    const { tier, expectedTurbo } = tierConfig;
    
    try {
        console.log(`\n🧪 Testing ${tier} tier...`);
        
        // 1. Generate API key for this tier
        const signupResponse = await fetch(`${BASE_URL}/api/simple-signup`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                email: `test-${tier.toLowerCase()}@bitcoinsprint.test`,
                tier: tier
            })
        });
        
        if (!signupResponse.ok) {
            throw new Error(`Signup failed: ${signupResponse.status}`);
        }
        
        const signupData = await signupResponse.json();
        const apiKey = signupData.key;
        
        console.log(`   📋 Generated API key: ${apiKey.slice(0, 20)}...`);
        
        // 2. Test status endpoint
        const statusResponse = await fetch(`${BASE_URL}/api/status`, {
            headers: { 'Authorization': `Bearer ${apiKey}` }
        });
        
        if (!statusResponse.ok) {
            throw new Error(`Status failed: ${statusResponse.status}`);
        }
        
        const statusData = await statusResponse.json();
        const actualTurbo = statusData.turbo_mode_enabled;
        
        // 3. Test license info endpoint
        const licenseResponse = await fetch(`${BASE_URL}/api/v1/license/info`, {
            headers: { 'Authorization': `Bearer ${apiKey}` }
        });
        
        if (!licenseResponse.ok) {
            throw new Error(`License info failed: ${licenseResponse.status}`);
        }
        
        const licenseData = await licenseResponse.json();
        const actualTurboFromLicense = licenseData.performance?.turbo_mode ?? licenseData.turbo_mode;
        
        // 4. Verify results
        const statusPassed = actualTurbo === expectedTurbo;
        const licensePassed = actualTurboFromLicense === expectedTurbo;
        
        console.log(`   📊 Status API turbo_mode_enabled: ${actualTurbo} (expected: ${expectedTurbo}) ${statusPassed ? '✅' : '❌'}`);
        console.log(`   📊 License API turbo_mode: ${actualTurboFromLicense} (expected: ${expectedTurbo}) ${licensePassed ? '✅' : '❌'}`);
        
        return {
            tier,
            passed: statusPassed && licensePassed,
            statusTurbo: actualTurbo,
            licenseTurbo: actualTurboFromLicense,
            expectedTurbo
        };
        
    } catch (error) {
        console.log(`   ❌ Error testing ${tier}: ${error.message}`);
        return {
            tier,
            passed: false,
            error: error.message
        };
    }
}

async function runTests() {
    console.log('🚀 Bitcoin Sprint Turbo Mode Tier Testing');
    console.log('==========================================');
    
    // Test web server availability
    try {
        const healthResponse = await fetchWithTimeout(`${BASE_URL}/api/health`, {}, 5000);
        if (!healthResponse.ok) {
            throw new Error('Health check failed');
        }
        console.log('✅ Web server is running and accessible');
    } catch (error) {
        console.log('❌ Web server is not accessible:', error.message);
        console.log('   Please ensure the web server is running on port 3002');
        process.exit(1);
    }
    
    const results = [];
    
    // Test each tier
    for (const tierConfig of TIER_TESTS) {
        const result = await testTier(tierConfig);
        results.push(result);
        
        // Add delay between tests
        await new Promise(resolve => setTimeout(resolve, 500));
    }
    
    // Summary
    console.log('\n📊 Test Summary:');
    console.log('================');
    
    const passedTests = results.filter(r => r.passed).length;
    const totalTests = results.length;
    
    results.forEach(result => {
        if (result.passed) {
            console.log(`✅ ${result.tier}: PASS`);
        } else {
            console.log(`❌ ${result.tier}: FAIL${result.error ? ` (${result.error})` : ''}`);
        }
    });
    
    console.log(`\nResults: ${passedTests}/${totalTests} tests passed`);
    
    if (passedTests === totalTests) {
        console.log('🎉 All turbo mode tier tests passed!');
        console.log('   - FREE and PRO tiers correctly have turbo_mode disabled');
        console.log('   - ENTERPRISE and ENTERPRISE_PLUS tiers correctly have turbo_mode enabled');
    } else {
        console.log('⚠️  Some tests failed. Check the tier configuration.');
        process.exit(1);
    }
}

// Run tests
runTests().catch(console.error);
