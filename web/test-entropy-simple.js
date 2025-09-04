#!/usr/bin/env node

/**
 * Simple entropy generation test
 */

async function testEntropy() {
	console.log('🎲 Testing Entropy Generation...\n');

	try {
		const response = await fetch('http://localhost:3002/api/entropy', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
				'X-API-Key': 'free-api-key-changeme'
			},
			body: JSON.stringify({
				size: 32,
				format: 'hex'
			})
		});

		if (response.ok) {
			const data = await response.json();
			console.log('✅ Entropy Generated Successfully!');
			console.log(`📊 Size: ${data.size} bytes`);
			console.log(`🔢 Format: ${data.format}`);
			console.log(`🎯 Source: ${data.source}`);
			console.log(`⚡ Generation Time: ${data.generation_time_ms}ms`);
			console.log(`🔑 Entropy: ${data.entropy.substring(0, 64)}...`);
			console.log(`📅 Timestamp: ${data.timestamp}`);
		} else {
			console.log('❌ Error:', response.status, response.statusText);
			const errorText = await response.text();
			console.log('Details:', errorText);
		}
	} catch (error) {
		console.log('❌ Network Error:', error.message);
	}
}

testEntropy();
