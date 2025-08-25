import { NextApiResponse } from 'next';
import { AuthenticatedRequest, withApiKeyAuth } from '../../lib/apiKeyAuth';

/**
 * Enterprise-only endpoint demonstrating tier-based access control.
 * Only ENTERPRISE and ENTERPRISE_PLUS tiers can access this endpoint.
 */
async function handler(req: AuthenticatedRequest, res: NextApiResponse) {
  if (req.method !== 'GET') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  try {
    // This endpoint is only accessible to Enterprise+ users
    const { apiKey, tier } = req;

    // Check if tier allows advanced features
    const isEnterpriseplus = tier === 'ENTERPRISE_PLUS';
    
    res.status(200).json({
      message: 'Welcome to Enterprise Analytics',
      tier: tier,
      features: {
        basicAnalytics: true,
        advancedMetrics: true,
        customReports: true,
        prioritySupport: true,
        dedicatedInfra: isEnterpriseplus,
        customIntegrations: isEnterpriseplus
      },
      analytics: {
        totalRequests: apiKey?.requests,
        blocksProcessed: apiKey?.blocksToday,
        accountAge: apiKey?.createdAt ? 
          Math.floor((Date.now() - new Date(apiKey.createdAt).getTime()) / (1000 * 60 * 60 * 24)) : 0,
        lastActivity: apiKey?.lastUsedAt
      },
      limits: {
        requestsPerMinute: tier === 'ENTERPRISE_PLUS' ? 100000 : 20000,
        blocksPerDay: 'unlimited',
        dedicatedSupport: isEnterpriseplus
      }
    });

  } catch (error) {
    console.error('[enterprise-analytics] Error:', error);
    res.status(500).json({ 
      error: 'Internal server error' 
    });
  }
}

// Export with authentication middleware requiring ENTERPRISE tier
export default withApiKeyAuth(handler, {
  updateUsage: true,
  incrementBlocks: false,
  requiredTier: 'ENTERPRISE' // This will also allow ENTERPRISE_PLUS
});
