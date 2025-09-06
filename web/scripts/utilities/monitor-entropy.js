#!/usr/bin/env node

/**
 * Real-time Entropy Generation Monitor
 * Monitors API calls between frontend and backend
 */

const http = require('http');

console.log('🎯 ENTROPY GENERATION MONITOR ACTIVE');
console.log('=====================================');
console.log('Monitoring requests to:');
console.log('  Frontend API: http://localhost:3002/api/entropy');
console.log('  Backend API: http://127.0.0.1:8080/api/v1/enterprise/entropy/fast');
console.log('');
console.log('🚀 READY! Click "Generate Entropy" on the frontend...');
console.log('=====================================\n');

// Monitor frontend API calls
const frontendServer = http.createServer((req, res) => {
	if (req.url === '/api/entropy' && req.method === 'POST') {
		console.log('📨 FRONTEND REQUEST DETECTED!');
		console.log(`   Time: ${new Date().toISOString()}`);
		console.log(`   Method: ${req.method}`);
		console.log(`   URL: ${req.url}`);
		console.log(`   Headers:`, JSON.stringify(req.headers, null, 2));

		let body = '';
		req.on('data', chunk => {
			body += chunk.toString();
		});

		req.on('end', () => {
			console.log(`   Body: ${body}`);
			console.log('   Status: Processing...\n');
		});
	}

	// Forward the request to the actual Next.js server
	const options = {
		hostname: 'localhost',
		port: 3002,
		path: req.url,
		method: req.method,
		headers: req.headers
	};

	const proxyReq = http.request(options, (proxyRes) => {
		res.writeHead(proxyRes.statusCode, proxyRes.headers);
		proxyRes.pipe(res);
	});

	req.pipe(proxyReq);
});

frontendServer.listen(3003, () => {
	console.log('📡 Frontend monitor listening on port 3003');
});

// Monitor backend API calls
const backendServer = http.createServer((req, res) => {
	if (req.url === '/api/v1/enterprise/entropy/fast' && req.method === 'POST') {
		console.log('🔧 BACKEND REQUEST DETECTED!');
		console.log(`   Time: ${new Date().toISOString()}`);
		console.log(`   Method: ${req.method}`);
		console.log(`   URL: ${req.url}`);
		console.log(`   Headers:`, JSON.stringify(req.headers, null, 2));

		let body = '';
		req.on('data', chunk => {
			body += chunk.toString();
		});

		req.on('end', () => {
			console.log(`   Body: ${body}`);
			console.log('   Status: Processing entropy generation...\n');
		});
	}

	// Forward to actual backend
	const options = {
		hostname: '127.0.0.1',
		port: 8080,
		path: req.url,
		method: req.method,
		headers: req.headers
	};

	const proxyReq = http.request(options, (proxyRes) => {
		console.log('📤 BACKEND RESPONSE:');
		console.log(`   Status: ${proxyRes.statusCode}`);
		console.log(`   Headers:`, JSON.stringify(proxyRes.headers, null, 2));

		let responseBody = '';
		proxyRes.on('data', chunk => {
			responseBody += chunk.toString();
		});

		proxyRes.on('end', () => {
			console.log(`   Response Body: ${responseBody}`);
			console.log('   ✅ Request completed!\n');
		});

		res.writeHead(proxyRes.statusCode, proxyRes.headers);
		proxyRes.pipe(res);
	});

	req.pipe(proxyReq);
});

backendServer.listen(8081, () => {
	console.log('🔧 Backend monitor listening on port 8081\n');
});

console.log('⚠️  NOTE: To use this monitor, you would need to temporarily change the API URLs in the frontend to point to ports 3003 and 8081');
console.log('   Or use browser dev tools to monitor network traffic\n');
