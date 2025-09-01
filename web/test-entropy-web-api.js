#!/usr/bin/env node

/**
 * Test script for the new entropy web API endpoint
 */

const BASE_URL = 'http://localhost:3002'; // Next.js dev server

async function testEntropyAPI() {
	console.log('🧪 Testing Entropy Web API Endpoint\n');

	try {
		// Test 1: Default entropy generation
		console.log('Test 1: Default entropy generation (32 bytes, hex)');
		const response1 = await fetch(`${BASE_URL}/api/entropy`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({})
		});

		if (response1.ok) {
			const data1 = await response1.json();
			console.log('✅ Success!');
			console.log(`   Size: ${data1.size} bytes`);
			console.log(`   Format: ${data1.format}`);
			console.log(`   Generation time: ${data1.generation_time_ms}ms`);
			console.log(`   Sample: ${data1.entropy.substring(0, 32)}...`);
		} else {
			console.log('❌ Failed:', response1.status, response1.statusText);
		}

		console.log('\n' + '='.repeat(50) + '\n');

		// Test 2: Custom size
		console.log('Test 2: Custom size (64 bytes, hex)');
		const response2 = await fetch(`${BASE_URL}/api/entropy`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ size: 64 })
		});

		if (response2.ok) {
			const data2 = await response2.json();
			console.log('✅ Success!');
			console.log(`   Size: ${data2.size} bytes`);
			console.log(`   Generation time: ${data2.generation_time_ms}ms`);
			console.log(`   Sample: ${data2.entropy.substring(0, 32)}...`);
		} else {
			console.log('❌ Failed:', response2.status, response2.statusText);
		}

		console.log('\n' + '='.repeat(50) + '\n');

		// Test 3: Base64 format
		console.log('Test 3: Base64 format (32 bytes)');
		const response3 = await fetch(`${BASE_URL}/api/entropy`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ size: 32, format: 'base64' })
		});

		if (response3.ok) {
			const data3 = await response3.json();
			console.log('✅ Success!');
			console.log(`   Format: ${data3.format}`);
			console.log(`   Sample: ${data3.entropy.substring(0, 32)}...`);
		} else {
			console.log('❌ Failed:', response3.status, response3.statusText);
		}

		console.log('\n' + '='.repeat(50) + '\n');

		// Test 4: Error handling
		console.log('Test 4: Error handling (invalid size)');
		const response4 = await fetch(`${BASE_URL}/api/entropy`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ size: 2000 }) // Too large
		});

		if (!response4.ok) {
			const error4 = await response4.json();
			console.log('✅ Error handling works!');
			console.log(`   Error: ${error4.error}`);
		} else {
			console.log('❌ Should have failed with invalid size');
		}

	} catch (error) {
		console.error('❌ Test failed:', error.message);
	}
}

// Run the test
testEntropyAPI().catch(console.error);
