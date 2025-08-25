import type { NextApiResponse } from 'next';
import type { AuthenticatedRequest } from '../../../../lib/auth';
import { withAuth } from '../../_withAuth';

async function handler(req: AuthenticatedRequest, res: NextApiResponse) {
  const { apiKey } = req;
  
  res.json({
    license: {
      tier: apiKey.tier,
      valid: true,
      email: apiKey.email,
      company: apiKey.company,
      key_id: apiKey.id
    },
    limits: {
      rate_limit_per_minute: getRateLimit(apiKey.tier),
      blocks_per_day: getBlockLimit(apiKey.tier),
      endpoints_available: getAvailableEndpoints(apiKey.tier).length,
      mempool_access: ['PRO', 'ENTERPRISE', 'ENTERPRISE_PLUS'].includes(apiKey.tier),
      predictive_features: ['ENTERPRISE', 'ENTERPRISE_PLUS'].includes(apiKey.tier),
      stream_access: ['ENTERPRISE', 'ENTERPRISE_PLUS'].includes(apiKey.tier)
    },
    usage_today: {
      requests: apiKey.requests,
      blocks_delivered: apiKey.blocksToday,
      last_request: new Date().toISOString()
    },
    performance: {
      target_latency_ms: getLatencyTarget(apiKey.tier),
      turbo_mode: ['ENTERPRISE', 'ENTERPRISE_PLUS'].includes(apiKey.tier),
      dedicated_infra: apiKey.tier === 'ENTERPRISE_PLUS'
    },
    api_version: "v1",
    documentation: "https://docs.bitcoin-sprint.com/v1"
  });
}

function getRateLimit(tier: string): number {
  const limits = { FREE: 100, PRO: 2000, ENTERPRISE: 20000, ENTERPRISE_PLUS: 100000 };
  return limits[tier as keyof typeof limits] || 0;
}

function getBlockLimit(tier: string): number {
  const limits = { FREE: 100, PRO: -1, ENTERPRISE: -1, ENTERPRISE_PLUS: -1 };
  return limits[tier as keyof typeof limits] || 0;
}

function getAvailableEndpoints(tier: string): string[] {
  const endpoints = {
    FREE: ['/api/status', '/api/latest'],
    PRO: ['/api/status', '/api/latest', '/api/metrics'],
    ENTERPRISE: ['/api/status', '/api/latest', '/api/metrics', '/api/predictive', '/api/stream', '/api/v1/license/info', '/api/v1/analytics/summary'],
    ENTERPRISE_PLUS: ['*']
  };
  return endpoints[tier as keyof typeof endpoints] || [];
}

function getLatencyTarget(tier: string): number {
  const targets = { FREE: 1000, PRO: 300, ENTERPRISE: 200, ENTERPRISE_PLUS: 100 };
  return targets[tier as keyof typeof targets] || 1000;
}

export default withAuth(handler);
