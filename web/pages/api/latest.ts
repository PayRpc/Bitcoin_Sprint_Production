import type { NextApiResponse } from 'next';
import type { AuthenticatedRequest } from '../../lib/auth';
import { withAuth } from './_withAuth';

async function handler(req: AuthenticatedRequest, res: NextApiResponse) {
  const { apiKey } = req;
  
  // Simulate Bitcoin block data - in production this would come from your Bitcoin Core RPC
  const latestBlocks = [
    {
      height: 850123,
      hash: "00000000000000000002a7c4c1e48d76c5a37902165a270156b7a8d72728a054",
      timestamp: Date.now() - 600000, // 10 minutes ago
      transactions: 2847,
      size: 1398420,
      difficulty: 48712405953118.43,
      confirmations: 1
    },
    {
      height: 850122,
      hash: "00000000000000000003b8d5d2f59e87d6b48a13276b381267c8b9e83839b165",
      timestamp: Date.now() - 1200000, // 20 minutes ago
      transactions: 3156,
      size: 1502387,
      difficulty: 48712405953118.43,
      confirmations: 2
    }
  ];

  const response = {
    blocks: latestBlocks,
    network: "mainnet",
    tier: apiKey.tier,
    latency_ms: getLatencyForTier(apiKey.tier),
    cached: false,
    last_updated: new Date().toISOString()
  };

  // Add tier-specific features
  if (['ENTERPRISE', 'ENTERPRISE_PLUS'].includes(apiKey.tier)) {
    (response as any).predictive_data = {
      next_block_eta_seconds: 420,
      mempool_size: 4200,
      fee_recommendation: {
        economy: 12,
        standard: 18,
        priority: 24
      }
    };
  }

  res.json(response);
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
