import type { NextApiRequest, NextApiResponse } from 'next';

interface EntropyRequest {
	size?: number;
	format?: 'hex' | 'base64' | 'bytes';
}

interface EntropyResponse {
	entropy: string;
	size: number;
	format: string;
	timestamp: string;
	source: string;
	generation_time_ms: number;
	request_id: string;
	tier: string;
}

// Security middleware for API key validation
function validateApiKey(req: NextApiRequest): { isValid: boolean; tier?: string; apiKey?: string } {
	const authHeader = req.headers.authorization;
	const apiKey = req.headers['x-api-key'] as string;

	// Check Authorization header (Bearer token)
	if (authHeader && authHeader.startsWith('Bearer ')) {
		const token = authHeader.substring(7);

		// Validate against environment-based keys
		const enterpriseKey = process.env.BITCOIN_SPRINT_ENTERPRISE_API_KEY;
		const proKey = process.env.BITCOIN_SPRINT_PRO_API_KEY;
		const freeKey = process.env.BITCOIN_SPRINT_FREE_API_KEY;

		if (token === enterpriseKey) return { isValid: true, tier: 'enterprise', apiKey: token };
		if (token === proKey) return { isValid: true, tier: 'pro', apiKey: token };
		if (token === freeKey) return { isValid: true, tier: 'free', apiKey: token };
	}

	// Check X-API-Key header
	if (apiKey) {
		const enterpriseKey = process.env.BITCOIN_SPRINT_ENTERPRISE_API_KEY;
		const proKey = process.env.BITCOIN_SPRINT_PRO_API_KEY;
		const freeKey = process.env.BITCOIN_SPRINT_FREE_API_KEY;

		if (apiKey === enterpriseKey) return { isValid: true, tier: 'enterprise', apiKey };
		if (apiKey === proKey) return { isValid: true, tier: 'pro', apiKey };
		if (apiKey === freeKey) return { isValid: true, tier: 'free', apiKey };
	}

	return { isValid: false };
}

// Rate limiting storage (in production, use Redis)
const rateLimitStore = new Map<string, { count: number; resetTime: number }>();

function checkRateLimit(apiKey: string, tier: string): { allowed: boolean; remaining: number; resetTime: number } {
	const now = Date.now();
	const key = `ratelimit:${apiKey}`;

	// Tier-based limits
	const limits = {
		free: { requests: 10, windowMs: 60000 }, // 10 per minute
		pro: { requests: 100, windowMs: 60000 }, // 100 per minute
		enterprise: { requests: 1000, windowMs: 60000 } // 1000 per minute
	};

	const limit = limits[tier as keyof typeof limits] || limits.free;
	const record = rateLimitStore.get(key);

	if (!record || now > record.resetTime) {
		// Reset or create new record
		rateLimitStore.set(key, {
			count: 1,
			resetTime: now + limit.windowMs
		});
		return { allowed: true, remaining: limit.requests - 1, resetTime: now + limit.windowMs };
	}

	if (record.count >= limit.requests) {
		return { allowed: false, remaining: 0, resetTime: record.resetTime };
	}

	record.count++;
	return { allowed: true, remaining: limit.requests - record.count, resetTime: record.resetTime };
}

export default async function handler(
	req: NextApiRequest,
	res: NextApiResponse<EntropyResponse | { error: string; details?: any }>
) {
	const requestId = `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

	// Only allow POST requests
	if (req.method !== 'POST') {
		return res.status(405).json({
			error: 'Method not allowed. Use POST.',
			request_id: requestId
		});
	}

	try {
		// Validate API key
		const authResult = validateApiKey(req);
		if (!authResult.isValid) {
			return res.status(401).json({
				error: 'Invalid or missing API key. Use Authorization: Bearer <key> or X-API-Key header.',
				request_id: requestId,
				details: {
					supported_formats: ['Bearer token', 'X-API-Key header'],
					documentation: '/api/docs'
				}
			});
		}

		const { tier, apiKey } = authResult;

		// Check rate limits
		const rateLimitResult = checkRateLimit(apiKey!, tier!);
		if (!rateLimitResult.allowed) {
			res.setHeader('X-RateLimit-Remaining', rateLimitResult.remaining.toString());
			res.setHeader('X-RateLimit-Reset', new Date(rateLimitResult.resetTime).toISOString());
			res.setHeader('Retry-After', Math.ceil((rateLimitResult.resetTime - Date.now()) / 1000).toString());

			return res.status(429).json({
				error: 'Rate limit exceeded. Please try again later.',
				request_id: requestId,
				details: {
					tier,
					reset_time: new Date(rateLimitResult.resetTime).toISOString(),
					retry_after_seconds: Math.ceil((rateLimitResult.resetTime - Date.now()) / 1000)
				}
			});
		}

		// Set rate limit headers
		res.setHeader('X-RateLimit-Remaining', rateLimitResult.remaining.toString());
		res.setHeader('X-RateLimit-Reset', new Date(rateLimitResult.resetTime).toISOString());
		res.setHeader('X-API-Tier', tier!);
		res.setHeader('X-Request-ID', requestId);

		const { size = 32, format = 'hex' } = req.body as EntropyRequest;

		// Validate size based on tier
		const maxSizes = { free: 256, pro: 512, enterprise: 1024 };
		const maxSize = maxSizes[tier as keyof typeof maxSizes] || 256;

		if (size < 1 || size > maxSize) {
			return res.status(400).json({
				error: `Size must be between 1 and ${maxSize} bytes for ${tier} tier.`,
				request_id: requestId
			});
		}

		// Validate format
		if (!['hex', 'base64', 'bytes'].includes(format)) {
			return res.status(400).json({
				error: 'Format must be hex, base64, or bytes',
				request_id: requestId
			});
		}

		const startTime = Date.now();

		// Call the main Go API with proper authentication
		const apiResponse = await fetch('http://127.0.0.1:8080/api/v1/enterprise/entropy/fast', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
				'X-API-Key': apiKey,
				'X-Request-ID': requestId,
				'X-Client-Tier': tier,
				'User-Agent': 'BitcoinSprint-Web/1.0'
			},
			body: JSON.stringify({ size })
		});

		if (!apiResponse.ok) {
			const errorText = await apiResponse.text();
			console.error(`Backend error [${requestId}]:`, apiResponse.status, errorText);

			return res.status(apiResponse.status).json({
				error: `Backend service error: ${apiResponse.statusText}`,
				request_id: requestId,
				details: { backend_status: apiResponse.status }
			});
		}

		const apiData = await apiResponse.json();
		const generationTime = Date.now() - startTime;

		// Format the response based on requested format
		let entropy: string;
		switch (format) {
			case 'base64':
				entropy = Buffer.from(apiData.entropy, 'hex').toString('base64');
				break;
			case 'bytes':
				entropy = apiData.entropy.match(/.{2}/g)?.map((byte: string) => parseInt(byte, 16)).join(',') || '';
				break;
			default:
				entropy = apiData.entropy;
		}

		const response: EntropyResponse = {
			entropy,
			size: apiData.size,
			format,
			timestamp: new Date().toISOString(),
			source: apiData.source || 'hardware',
			generation_time_ms: generationTime,
			request_id: requestId,
			tier: tier!
		};

		// Log successful request (in production, use proper logging)
		console.log(`[${requestId}] Entropy generated: ${generationTime}ms, tier: ${tier}, size: ${size}`);

		res.status(200).json(response);

	} catch (error: any) {
		console.error(`[${requestId}] Entropy generation failed:`, error);

		res.status(500).json({
			error: error.message || 'Failed to generate entropy',
			request_id: requestId,
			details: {
				timestamp: new Date().toISOString(),
				endpoint: '/api/entropy'
			}
		});
	}
}
