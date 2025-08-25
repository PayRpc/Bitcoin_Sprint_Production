import type { NextApiResponse } from 'next';
import type { AuthenticatedRequest } from '../../lib/auth';
import { withAuth } from './_withAuth';

async function handler(req: AuthenticatedRequest, res: NextApiResponse) {
  const { apiKey } = req;
  
  res.json({
    next_block_prediction: {
      eta_seconds: 420,
      probability_60s: 0.72,
      probability_300s: 0.95,
      confidence_score: 0.87
    },
    mempool_analytics: {
      size: 4200,
      growth_rate_per_minute: 12.3,
      fee_pressure: "moderate",
      congestion_level: 0.42,
      priority_segments: {
        high_fee: { count: 340, min_fee: 25 },
        medium_fee: { count: 1200, min_fee: 15 },
        low_fee: { count: 2660, min_fee: 8 }
      }
    },
    network_trends: {
      hashrate_trend: "rising",
      difficulty_adjustment_eta: "3 days 14 hours",
      next_difficulty_change_percent: 2.4
    },
    tier: apiKey.tier,
    last_updated: new Date().toISOString(),
    turbo_mode: true
  });
}

export default withAuth(handler);
