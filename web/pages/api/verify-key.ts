import { NextApiResponse } from 'next';
import { AuthenticatedRequest, withApiKeyAuth } from '../../lib/apiKeyAuth';

/**
 * Sample protected endpoint using API key verification.
 * 
 * This endpoint demonstrates:
 * - API key authentication
 * - Usage tracking
 * - Tier-based access control
 * - Rate limiting awareness
 */
async function handler(req: AuthenticatedRequest, res: NextApiResponse) {
  if (req.method !== 'GET') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  try {
    // API key is already verified by the middleware
    const { apiKey, tier } = req;

    // Return user information and API status
    res.status(200).json({
      message: 'Authentication successful',
      user: {
        email: apiKey?.email,
        company: apiKey?.company,
        tier: tier,
        keyId: apiKey?.id
      },
      keyInfo: {
        createdAt: apiKey?.createdAt,
        expiresAt: apiKey?.expiresAt,
        requests: apiKey?.requests,
        blocksToday: apiKey?.blocksToday,
        lastUsedAt: apiKey?.lastUsedAt
      },
      timestamp: new Date().toISOString()
    });

  } catch (error) {
    console.error('[verify-key] Error:', error);
    res.status(500).json({ 
      error: 'Internal server error',
      message: 'Failed to process verification request'
    });
  }
}

// Export with authentication middleware
export default withApiKeyAuth(handler, {
  updateUsage: true,        // Track usage
  incrementBlocks: false,   // Don't count as block request
  requiredTier: undefined   // Allow all tiers
});
