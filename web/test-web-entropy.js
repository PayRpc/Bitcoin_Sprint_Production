#!/usr/bin/env node

/**
 * Test entropy generation from web API
 */

async function testWebEntropy() {
	console.log('🎲 Testing Web Entropy Generation...\n');

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
			console.log('✅ Web Entropy Generated Successfully!');
			console.log(`📊 Size: ${data.size} bytes`);
			console.log(`🔢 Format: ${data.format}`);
			console.log(`🎯 Source: ${data.source}`);
			console.log(`⚡ Generation Time: ${data.generation_time_ms}ms`);
			console.log(`🔑 Entropy: ${data.entropy}`);
			console.log(`📅 Timestamp: ${data.timestamp}`);
			console.log(`🏷️  Tier: ${data.tier}`);
			console.log(`🆔 Request ID: ${data.request_id}`);
		} else {
			console.log('❌ Error:', response.status, response.statusText);
			const errorText = await response.text();
			console.log('Details:', errorText);
		}
	} catch (error) {
		console.log('❌ Network Error:', error.message);
	}
}

testWebEntropy();
