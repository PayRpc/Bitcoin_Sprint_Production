import type { NextApiResponse } from 'next';
import type { AuthenticatedRequest } from '../../lib/auth';
import { withAuth } from './_withAuth';

async function handler(req: AuthenticatedRequest, res: NextApiResponse) {
  const { apiKey } = req;
  
  res.json({
    tier: apiKey.tier,
    license_key: '****' + apiKey.key.slice(-4),
    valid: true,
    uptime_seconds: Math.floor(process.uptime()),
    version: "1.2.0",
    turbo_mode_enabled: ['ENTERPRISE', 'ENTERPRISE_PLUS'].includes(apiKey.tier),
    endpoints_available: getAvailableEndpoints(apiKey.tier),
    rate_limit_remaining: getRateLimitRemaining(apiKey.tier),
    blocks_today: apiKey.blocksToday,
    api_docs: "https://docs.bitcoin-sprint.com"
  });
}

function getAvailableEndpoints(tier: string): string[] {
  const endpoints = {
    FREE: ['/api/status', '/api/latest'],
    PRO: ['/api/status', '/api/latest', '/api/metrics'],
    ENTERPRISE: ['/api/status', '/api/latest', '/api/metrics', '/api/predictive', '/api/stream', '/api/v1/license/info', '/api/v1/analytics/summary'],
    ENTERPRISE_PLUS: ['*'] // All endpoints
  };
  return endpoints[tier as keyof typeof endpoints] || [];
}

function getRateLimitRemaining(tier: string): number {
  const limits = {
    FREE: 100,
    PRO: 2000, 
    ENTERPRISE: 20000,
    ENTERPRISE_PLUS: 100000
  };
  return limits[tier as keyof typeof limits] || 0;
}

export default withAuth(handler);
