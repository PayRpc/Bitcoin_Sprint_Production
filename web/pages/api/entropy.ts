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
}

export default async function handler(
	req: NextApiRequest,
	res: NextApiResponse<EntropyResponse | { error: string }>
) {
	if (req.method !== 'POST') {
		return res.status(405).json({ error: 'Method not allowed. Use POST.' });
	}

	try {
		const { size = 32, format = 'hex' } = req.body as EntropyRequest;

		// Validate size
		if (size < 1 || size > 1024) {
			return res.status(400).json({ error: 'Size must be between 1 and 1024 bytes' });
		}

		// Validate format
		if (!['hex', 'base64', 'bytes'].includes(format)) {
			return res.status(400).json({ error: 'Format must be hex, base64, or bytes' });
		}

		const startTime = Date.now();

		// Call the main Go API
		const apiResponse = await fetch('http://127.0.0.1:8080/api/v1/enterprise/entropy/fast', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
				'X-API-Key': process.env.BITCOIN_SPRINT_API_KEY || 'turbo-api-key-2024'
			},
			body: JSON.stringify({ size })
		});

		if (!apiResponse.ok) {
			throw new Error(`API request failed: ${apiResponse.status} ${apiResponse.statusText}`);
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
			generation_time_ms: generationTime
		};

		res.status(200).json(response);

	} catch (error: any) {
		console.error('Entropy generation failed:', error);
		res.status(500).json({
			error: error.message || 'Failed to generate entropy'
		});
	}
}
