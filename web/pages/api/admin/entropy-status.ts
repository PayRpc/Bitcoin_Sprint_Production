import type { NextApiRequest, NextApiResponse } from 'next';

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse
) {
  if (req.method !== 'GET') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  try {
    // Dynamically import the entropy bridge
    const entropyModule = await import('../../../rust-entropy-bridge.js');
    const bridge = await entropyModule.getEntropyBridge();
    const status = bridge.getStatus();

    // Generate a test secret to verify functionality
    const startTime = Date.now();
    const testSecret = await bridge.generateAdminSecret('hex');
    const generationTime = (Date.now() - startTime) / 1000;

    // Record metrics
    const { recordEntropySecretGeneration, recordEntropyQualityScore } = await import('../../../lib/prometheus');
    recordEntropySecretGeneration('hex', true, generationTime);
    recordEntropyQualityScore(testSecret.length * 2); // Simple quality score based on length

    res.status(200).json({
      status: 'operational',
      entropy_bridge: {
        available: status.available,
        rust_available: status.rustAvailable,
        fallback_mode: status.fallbackMode,
        test_secret_length: testSecret.length,
        test_secret_preview: testSecret.substring(0, 16) + '...',
        generation_time_seconds: generationTime
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
