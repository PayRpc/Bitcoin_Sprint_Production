import type { NextApiResponse } from 'next';
import type { AuthenticatedRequest } from '../../lib/auth';
import { withAuth } from './_withAuth';

async function handler(req: AuthenticatedRequest, res: NextApiResponse) {
  const { apiKey } = req;
  
  res.json({
    node_performance: {
      block_latency_ms: getLatencyForTier(apiKey.tier),
      peer_count: 8,
      sync_progress: 1.0,
      network_hashrate: "600.45 EH/s"
    },
    api_usage: {
      requests_today: apiKey.requests,
      blocks_delivered: apiKey.blocksToday,
      tier: apiKey.tier,
      rate_limit_used_percent: 15.3
    },
    mempool_status: {
      size: 4200,
      fee_estimates: {
        fast: 24,
        medium: 18, 
        slow: 12
      }
    },
    system_health: {
      uptime_seconds: Math.floor(process.uptime()),
      cpu_usage_percent: 23.5,
      memory_usage_mb: 2840,
      disk_usage_percent: 67.2
    }
  });
}

function getLatencyForTier(tier: string): number {
  const latencies = {
    FREE: 850,
    PRO: 280,
    ENTERPRISE: 180,
    ENTERPRISE_PLUS: 85
  };
  return latencies[tier as keyof typeof latencies] || 1000;
}

export default withAuth(handler);
