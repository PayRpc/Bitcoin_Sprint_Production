import type { NextApiRequest, NextApiResponse } from 'next';

interface ApiDocsResponse {
	title: string;
	version: string;
	description: string;
	endpoints: {
		[key: string]: {
			method: string;
			path: string;
			description: string;
			authentication: string;
			parameters: any;
			responses: any;
			examples: any;
		};
	};
	authentication: {
		methods: string[];
		tiers: {
			[key: string]: {
				description: string;
				limits: string;
				features: string[];
			};
		};
	};
	security: {
		headers: string[];
		rate_limiting: string;
		cors: string;
	};
}

export default function handler(
	req: NextApiRequest,
	res: NextApiResponse<ApiDocsResponse>
) {
	if (req.method !== 'GET') {
		return res.status(405).json({
			title: 'Method Not Allowed',
			version: '1.0.0',
			description: 'Use GET method',
			endpoints: {},
			authentication: { methods: [], tiers: {} },
			security: { headers: [], rate_limiting: '', cors: '' }
		} as ApiDocsResponse);
	}

	const docs: ApiDocsResponse = {
		title: 'Bitcoin Sprint Web API',
		version: '2.5.0',
		description: 'Secure entropy generation and blockchain infrastructure API',
		endpoints: {
			entropy: {
				method: 'POST',
				path: '/api/entropy',
				description: 'Generate cryptographically secure random entropy',
				authentication: 'Required - Bearer token or X-API-Key header',
				parameters: {
					body: {
						size: {
							type: 'number',
							description: 'Number of bytes to generate (1-1024)',
							default: 32,
							required: false
						},
						format: {
							type: 'string',
							description: 'Output format',
							enum: ['hex', 'base64', 'bytes'],
							default: 'hex',
							required: false
						}
					}
				},
				responses: {
					200: {
						description: 'Success',
						schema: {
							entropy: 'string',
							size: 'number',
							format: 'string',
							timestamp: 'string',
							source: 'string',
							generation_time_ms: 'number',
							request_id: 'string',
							tier: 'string'
						}
					},
					400: {
						description: 'Bad Request - Invalid parameters'
					},
					401: {
						description: 'Unauthorized - Invalid or missing API key'
					},
					429: {
						description: 'Rate limit exceeded'
					},
					500: {
						description: 'Internal server error'
					}
				},
				examples: {
					request: {
						curl: `curl -X POST http://localhost:3002/api/entropy \\
  -H "Authorization: Bearer your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{"size": 32, "format": "hex"}'`,
						javascript: `fetch('/api/entropy', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer your-api-key',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({ size: 32, format: 'hex' })
})`
					},
					response: {
						success: {
							entropy: "a1b2c3d4e5f678901234567890abcdef...",
							size: 32,
							format: "hex",
							timestamp: "2025-01-15T10:30:00.000Z",
							source: "hardware",
							generation_time_ms: 15,
							request_id: "req_1736937000000_abc123",
							tier: "enterprise"
						}
					}
				}
			}
		},
		authentication: {
			methods: [
				'Authorization: Bearer <api-key>',
				'X-API-Key: <api-key>'
			],
			tiers: {
				free: {
					description: 'Basic tier for development and testing',
					limits: '10 requests per minute, max 256 bytes',
					features: [
						'Basic entropy generation',
						'Hex and base64 formats',
						'Community support'
					]
				},
				pro: {
					description: 'Professional tier for production applications',
					limits: '100 requests per minute, max 512 bytes',
					features: [
						'All free features',
						'Higher rate limits',
						'Priority support',
						'Advanced monitoring'
					]
				},
				enterprise: {
					description: 'Enterprise tier for high-volume applications',
					limits: '1000 requests per minute, max 1024 bytes',
					features: [
						'All pro features',
						'Maximum rate limits',
						'Dedicated support',
						'Custom integrations',
						'SLA guarantees'
					]
				}
			}
		},
		security: {
			headers: [
				'X-Frame-Options: DENY',
				'X-Content-Type-Options: nosniff',
				'X-XSS-Protection: 1; mode=block',
				'Content-Security-Policy: strict policy',
				'X-RateLimit-Remaining: <count>',
				'X-RateLimit-Reset: <timestamp>',
				'X-API-Tier: <tier>',
				'X-Request-ID: <uuid>'
			],
			rate_limiting: 'Tier-based rate limiting with automatic reset windows',
			cors: 'Configured for localhost origins in development'
		}
	};

	res.setHeader('X-API-Docs-Version', '2.5.0');
	res.setHeader('Cache-Control', 'public, max-age=3600'); // Cache for 1 hour

	res.status(200).json(docs);
}
