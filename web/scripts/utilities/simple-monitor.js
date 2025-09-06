#!/usr/bin/env node

/**
 * Simple Entropy Monitor - Watches for API activity
 */

console.log('🎯 ENTROPY MONITOR ACTIVE');
console.log('========================');
console.log('Monitoring:');
console.log('  Frontend: http://localhost:3002/api/entropy');
console.log('  Backend:  http://127.0.0.1:8080/api/v1/enterprise/entropy/fast');
console.log('');
console.log('🚀 READY! Click "Generate Entropy" now...');
console.log('========================\n');

// Simple monitoring by periodically testing endpoints
async function monitorEndpoints() {
	let requestCount = 0;

	setInterval(async () => {
		try {
			// Test frontend health
			const frontendResponse = await fetch('http://localhost:3002/api/health').catch(() => null);
			const backendResponse = await fetch('http://127.0.0.1:8080/health').catch(() => null);

			if (frontendResponse?.ok && backendResponse?.ok) {
				process.stdout.write('.');
			} else {
				process.stdout.write('!');
			}
		} catch (error) {
			process.stdout.write('x');
		}
	}, 1000);

	// Listen for actual entropy requests
	console.log('⏳ Waiting for entropy generation requests...\n');
}

monitorEndpoints();

// Also set up a simple HTTP server to capture requests if needed
import http from 'http';

const monitorServer = http.createServer((req, res) => {
	if (req.url.includes('/entropy') && req.method === 'POST') {
		console.log('\n🎯 ENTROPY REQUEST CAPTURED!');
		console.log(`Time: ${new Date().toISOString()}`);
		console.log(`URL: ${req.url}`);
		console.log(`Method: ${req.method}`);

		let body = '';
		req.on('data', chunk => {
			body += chunk.toString();
		});

		req.on('end', () => {
			console.log(`Body: ${body}`);
			console.log('Processing request...\n');
		});
	}

	res.writeHead(200, { 'Content-Type': 'text/plain' });
	res.end('Monitor active');
});

monitorServer.listen(9999, () => {
	console.log('📡 Monitor server running on port 9999');
});
