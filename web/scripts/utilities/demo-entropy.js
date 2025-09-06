#!/usr/bin/env node

/**
 * Demo script showing how to use the entropy generator
 * Run with: node demo-entropy.js
 */

const BASE_URL = 'http://localhost:3002';

async function demoEntropyGenerator() {
	console.log('🎲 Bitcoin Sprint Entropy Generator Demo\n');

	// Demo 1: Generate encryption key
	console.log('🔐 Demo 1: Generate 256-bit encryption key');
	try {
		const keyResponse = await fetch(`${BASE_URL}/api/entropy`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ size: 32, format: 'hex' })
		});

		if (keyResponse.ok) {
			const keyData = await keyResponse.json();
			console.log(`✅ Generated encryption key (${keyData.generation_time_ms}ms):`);
			console.log(`   ${keyData.entropy}`);
			console.log(`   Length: ${keyData.entropy.length} characters (${keyData.size * 8} bits)\n`);
		}
	} catch (error) {
		console.log('❌ Failed to generate key:', error.message);
	}

	// Demo 2: Generate session token
	console.log('🎫 Demo 2: Generate secure session token');
	try {
		const tokenResponse = await fetch(`${BASE_URL}/api/entropy`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ size: 16, format: 'base64' })
		});

		if (tokenResponse.ok) {
			const tokenData = await tokenResponse.json();
			console.log(`✅ Generated session token (${tokenData.generation_time_ms}ms):`);
			console.log(`   ${tokenData.entropy}`);
			console.log(`   Length: ${tokenData.entropy.length} characters (${tokenData.size * 8} bits)\n`);
		}
	} catch (error) {
		console.log('❌ Failed to generate token:', error.message);
	}

	// Demo 3: Generate random numbers for gaming
	console.log('🎮 Demo 3: Generate random numbers for gaming');
	try {
		const gameResponse = await fetch(`${BASE_URL}/api/entropy`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ size: 4, format: 'bytes' })
		});

		if (gameResponse.ok) {
			const gameData = await gameResponse.json();
			const bytes = gameData.entropy.split(',').map(Number);
			console.log(`✅ Generated random bytes (${gameData.generation_time_ms}ms):`);
			console.log(`   Raw bytes: [${gameData.entropy}]`);
			console.log(`   As integers: [${bytes.join(', ')}]`);
			console.log(`   Random die roll (1-6): ${bytes[0] % 6 + 1}`);
			console.log(`   Random card (1-52): ${bytes[1] % 52 + 1}\n`);
		}
	} catch (error) {
		console.log('❌ Failed to generate game numbers:', error.message);
	}

	// Demo 4: Performance test
	console.log('⚡ Demo 4: Performance test (10 generations)');
	const startTime = Date.now();
	let totalGenTime = 0;

	for (let i = 0; i < 10; i++) {
		try {
			const perfResponse = await fetch(`${BASE_URL}/api/entropy`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ size: 32 })
			});

			if (perfResponse.ok) {
				const perfData = await perfResponse.json();
				totalGenTime += perfData.generation_time_ms;
			}
		} catch (error) {
			console.log(`❌ Performance test failed on iteration ${i + 1}`);
			break;
		}
	}

	const totalTime = Date.now() - startTime;
	console.log(`✅ Performance test completed:`);
	console.log(`   Total time: ${totalTime}ms`);
	console.log(`   Average generation time: ${(totalGenTime / 10).toFixed(2)}ms`);
	console.log(`   Requests per second: ${(10000 / totalTime).toFixed(1)}\n`);

	console.log('🎯 Demo completed! Visit http://localhost:3002/entropy for the web interface.');
}

// Handle errors gracefully
demoEntropyGenerator().catch(error => {
	console.error('❌ Demo failed:', error.message);
	console.log('\n💡 Make sure:');
	console.log('   1. The web server is running: npm run dev');
	console.log('   2. The main API is running on port 8080');
	console.log('   3. Both services are accessible');
});
