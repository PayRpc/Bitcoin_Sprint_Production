import type { NextApiResponse } from 'next';
import type { AuthenticatedRequest } from '../../../../lib/auth';
import { withAuth } from '../../_withAuth';

async function handler(req: AuthenticatedRequest, res: NextApiResponse) {
  const { apiKey } = req;
  
  res.json({
    network_summary: {
      current_block_height: 850123,
      network_hashrate: "600.45 EH/s", 
      difficulty: 48712405953118.43,
      total_peers: 8,
      mempool_size: 4200
    },
    node_performance: {
      uptime_seconds: Math.floor(process.uptime()),
      block_sync_status: "synchronized",
      last_block_received: new Date(Date.now() - 420000).toISOString(),
      average_block_time: 587 // seconds
    },
    api_analytics: {
      your_requests_today: apiKey.requests,
      blocks_sent_today: apiKey.blocksToday,
      tier: apiKey.tier,
      performance_tier_latency: getLatencyTarget(apiKey.tier),
      turbo_mode: ['ENTERPRISE', 'ENTERPRISE_PLUS'].includes(apiKey.tier)
    },
    system_health: {
      cpu_usage_percent: 23.5,
      memory_usage_mb: 2840,
      disk_usage_percent: 67.2,
      network_io_kbps: 450
    },
    recent_activity: {
      last_10_requests: generateRecentActivity(),
      error_rate_percent: 0.02,
      average_response_time_ms: getLatencyTarget(apiKey.tier)
    },
    api_version: "v1",
    timestamp: new Date().toISOString()
  });
}

function getLatencyTarget(tier: string): number {
  const targets = { FREE: 1000, PRO: 300, ENTERPRISE: 200, ENTERPRISE_PLUS: 100 };
  return targets[tier as keyof typeof targets] || 1000;
}

function generateRecentActivity() {
  const endpoints = ['/api/status', '/api/latest', '/api/metrics', '/api/predictive'];
  const activities = [];
  
  for (let i = 0; i < 10; i++) {
    activities.push({
      timestamp: new Date(Date.now() - i * 60000).toISOString(),
      endpoint: endpoints[Math.floor(Math.random() * endpoints.length)],
      response_time_ms: Math.floor(Math.random() * 200) + 100,
      status: 200
    });
  }
  
  return activities;
}

export default withAuth(handler);
