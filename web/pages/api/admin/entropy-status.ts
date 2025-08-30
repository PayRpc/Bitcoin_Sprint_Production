import type { NextApiRequest, NextApiResponse } from 'next';

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse
) {
  if (req.method !== 'GET') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  try {
    // Simple stub for build testing
    res.status(200).json({
      status: 'operational',
      entropy_bridge: {
        available: true,
        rust_available: false,
        fallback_mode: true,
        test_secret_length: 64,
        test_secret_preview: 'a1b2c3d4e5f6g7h8...'
      },
      timestamp: new Date().toISOString(),
      service: 'bitcoin-sprint-entropy'
    });
  } catch (error: any) {
    console.error('Entropy status check failed:', error);
    res.status(500).json({
      status: 'error',
      error: error.message || 'Entropy bridge status check failed',
      timestamp: new Date().toISOString()
    });
  }
}
