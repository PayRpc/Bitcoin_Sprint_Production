#!/usr/bin/env node

/**
 * Bitcoin Sprint Web Security Test Suite
 * Comprehensive testing of authentication, rate limiting, and security features
 */

const BASE_URL = 'http://localhost:3002';

class SecurityTestSuite {
	constructor() {
		this.results = [];
		this.startTime = 0;
	}

	async runAllTests() {
		console.log('🔐 Bitcoin Sprint Web Security Test Suite');
		console.log('==========================================\n');

		this.startTime = Date.now();

		// Authentication Tests
		await this.testNoAuth();
		await this.testInvalidAuth();
		await this.testValidFreeTier();
		await this.testValidProTier();
		await this.testValidEnterpriseTier();

		// Rate Limiting Tests
		await this.testRateLimiting();

		// Input Validation Tests
		await this.testInputValidation();

		// Security Headers Tests
		await this.testSecurityHeaders();

		// API Documentation Test
		await this.testApiDocs();

		this.printSummary();
	}

	async makeRequest(endpoint, options = {}) {
		try {
			const response = await fetch(`${BASE_URL}${endpoint}`, {
				headers: {
					'Content-Type': 'application/json',
					...options.headers
				},
				...options
			});

			let data;
			try {
				data = await response.json();
			} catch {
				// Response might not be JSON
			}

			return { response, data };
		} catch (error) {
			return {
				response: new Response(null, { status: 0 }),
				error: error instanceof Error ? error.message : 'Network error'
			};
		}
	}

	async testNoAuth() {
		const start = Date.now();
		const { response } = await this.makeRequest('/api/entropy', {
			method: 'POST',
			body: JSON.stringify({ size: 32 })
		});

		const success = response.status === 401;
		const duration = Date.now() - start;

		this.results.push({
			name: 'No Authentication',
			success,
			message: success ? '✅ Correctly rejected request without auth' : `❌ Expected 401, got ${response.status}`,
			duration,
			details: { status: response.status }
		});
	}

	async testInvalidAuth() {
		const start = Date.now();
		const { response } = await this.makeRequest('/api/entropy', {
			method: 'POST',
			headers: {
				'Authorization': 'Bearer invalid-key'
			},
			body: JSON.stringify({ size: 32 })
		});

		const success = response.status === 401;
		const duration = Date.now() - start;

		this.results.push({
			name: 'Invalid Authentication',
			success,
			message: success ? '✅ Correctly rejected invalid API key' : `❌ Expected 401, got ${response.status}`,
			duration,
			details: { status: response.status }
		});
	}

	async testValidFreeTier() {
		const start = Date.now();
		const { response, data } = await this.makeRequest('/api/entropy', {
			method: 'POST',
			headers: {
				'Authorization': 'Bearer free-api-key-changeme'
			},
			body: JSON.stringify({ size: 32 })
		});

		const success = response.status === 200 && data?.tier === 'free';
		const duration = Date.now() - start;

		this.results.push({
			name: 'Valid Free Tier Authentication',
			success,
			message: success ? '✅ Free tier authentication successful' : `❌ Expected 200 with free tier, got ${response.status}`,
			duration,
			details: { status: response.status, tier: data?.tier }
		});
	}

	async testValidProTier() {
		const start = Date.now();
		const { response, data } = await this.makeRequest('/api/entropy', {
			method: 'POST',
			headers: {
				'Authorization': 'Bearer pro-api-key-changeme'
			},
			body: JSON.stringify({ size: 32 })
		});

		const success = response.status === 200 && data?.tier === 'pro';
		const duration = Date.now() - start;

		this.results.push({
			name: 'Valid Pro Tier Authentication',
			success,
			message: success ? '✅ Pro tier authentication successful' : `❌ Expected 200 with pro tier, got ${response.status}`,
			duration,
			details: { status: response.status, tier: data?.tier }
		});
	}

	async testValidEnterpriseTier() {
		const start = Date.now();
		const { response, data } = await this.makeRequest('/api/entropy', {
			method: 'POST',
			headers: {
				'Authorization': 'Bearer enterprise-api-key-changeme'
			},
			body: JSON.stringify({ size: 64 })
		});

		const success = response.status === 200 && data?.tier === 'enterprise';
		const duration = Date.now() - start;

		this.results.push({
			name: 'Valid Enterprise Tier Authentication',
			success,
			message: success ? '✅ Enterprise tier authentication successful' : `❌ Expected 200 with enterprise tier, got ${response.status}`,
			duration,
			details: { status: response.status, tier: data?.tier }
		});
	}

	async testRateLimiting() {
		const start = Date.now();
		let rateLimited = false;
		let requestCount = 0;

		// Make multiple requests to trigger rate limiting
		for (let i = 0; i < 15; i++) {
			const { response } = await this.makeRequest('/api/entropy', {
				method: 'POST',
				headers: {
					'Authorization': 'Bearer free-api-key-changeme'
				},
				body: JSON.stringify({ size: 16 })
			});

			requestCount++;
			if (response.status === 429) {
				rateLimited = true;
				break;
			}

			// Small delay to avoid overwhelming
			await new Promise(resolve => setTimeout(resolve, 100));
		}

		const success = rateLimited;
		const duration = Date.now() - start;

		this.results.push({
			name: 'Rate Limiting',
			success,
			message: success ? `✅ Rate limiting triggered after ${requestCount} requests` : '❌ Rate limiting not working',
			duration,
			details: { requestsMade: requestCount, rateLimited }
		});
	}

	async testInputValidation() {
		const start = Date.now();

		// Test invalid size
		const { response: sizeResponse } = await this.makeRequest('/api/entropy', {
			method: 'POST',
			headers: {
				'Authorization': 'Bearer free-api-key-changeme'
			},
			body: JSON.stringify({ size: 1000 }) // Too large for free tier
		});

		// Test invalid format
		const { response: formatResponse } = await this.makeRequest('/api/entropy', {
			method: 'POST',
			headers: {
				'Authorization': 'Bearer free-api-key-changeme'
			},
			body: JSON.stringify({ size: 32, format: 'invalid' })
		});

		const success = sizeResponse.status === 400 && formatResponse.status === 400;
		const duration = Date.now() - start;

		this.results.push({
			name: 'Input Validation',
			success,
			message: success ? '✅ Input validation working correctly' : '❌ Input validation failed',
			duration,
			details: {
				sizeValidation: sizeResponse.status,
				formatValidation: formatResponse.status
			}
		});
	}

	async testSecurityHeaders() {
		const start = Date.now();
		const { response } = await this.makeRequest('/api/entropy', {
			method: 'POST',
			headers: {
				'Authorization': 'Bearer free-api-key-changeme'
			},
			body: JSON.stringify({ size: 32 })
		});

		const hasSecurityHeaders =
			response.headers.get('X-Frame-Options') === 'DENY' &&
			response.headers.get('X-Content-Type-Options') === 'nosniff' &&
			response.headers.get('X-Request-ID');

		const success = hasSecurityHeaders;
		const duration = Date.now() - start;

		this.results.push({
			name: 'Security Headers',
			success,
			message: success ? '✅ Security headers present' : '❌ Missing security headers',
			duration,
			details: {
				frameOptions: response.headers.get('X-Frame-Options'),
				contentTypeOptions: response.headers.get('X-Content-Type-Options'),
				requestId: response.headers.get('X-Request-ID')
			}
		});
	}

	async testApiDocs() {
		const start = Date.now();
		const { response, data } = await this.makeRequest('/api/docs');

		const success = response.status === 200 && data?.title === 'Bitcoin Sprint Web API';
		const duration = Date.now() - start;

		this.results.push({
			name: 'API Documentation',
			success,
			message: success ? '✅ API documentation accessible' : `❌ API docs failed: ${response.status}`,
			duration,
			details: { status: response.status, title: data?.title }
		});
	}

	printSummary() {
		const totalTime = Date.now() - this.startTime;
		const passed = this.results.filter(r => r.success).length;
		const failed = this.results.length - passed;

		console.log('\n📊 Test Results Summary');
		console.log('========================');

		this.results.forEach(result => {
			const icon = result.success ? '✅' : '❌';
			const duration = result.duration ? ` (${result.duration}ms)` : '';
			console.log(`${icon} ${result.name}${duration}: ${result.message}`);
		});

		console.log('\n🎯 Final Score:');
		console.log(`   Passed: ${passed}/${this.results.length}`);
		console.log(`   Failed: ${failed}/${this.results.length}`);
		console.log(`   Success Rate: ${((passed / this.results.length) * 100).toFixed(1)}%`);
		console.log(`   Total Time: ${totalTime}ms`);

		if (failed === 0) {
			console.log('\n🎉 All security tests passed! Your web application is secure.');
		} else {
			console.log('\n⚠️  Some tests failed. Please review the security implementation.');
		}

		console.log('\n🔧 Next Steps:');
		console.log('1. Start the web application: npm run dev');
		console.log('2. Start the Go backend: go run cmd/sprintd/main.go');
		console.log('3. Run this test: node test-security.js');
		console.log('4. Access API docs: http://localhost:3002/api/docs');
	}
}

// Run the test suite
async function main() {
	const testSuite = new SecurityTestSuite();
	await testSuite.runAllTests();
}

if (require.main === module) {
	main().catch(console.error);
}

module.exports = SecurityTestSuite;
