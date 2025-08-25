import type { NextApiResponse } from 'next';
import type { AuthenticatedRequest } from '../../lib/auth';
import { withAuth } from './_withAuth';

async function handler(req: AuthenticatedRequest, res: NextApiResponse) {
  const { apiKey } = req;
  
  res.writeHead(200, {
    'Content-Type': 'text/event-stream',
    'Cache-Control': 'no-cache',
    'Connection': 'keep-alive',
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Headers': 'Cache-Control'
  });

  // Send initial connection event
  res.write(`data: ${JSON.stringify({ 
    event: "stream_connected", 
    tier: apiKey.tier,
    timestamp: new Date().toISOString(),
    latency_target_ms: getLatencyForTier(apiKey.tier)
  })}\n\n`);

  // Simulate real-time Bitcoin data updates
  const intervals: NodeJS.Timeout[] = [];
  
  // Block updates (rare)
  const blockInterval = setInterval(() => {
    const blockData = {
      event: "new_block",
      data: {
        height: 850000 + Math.floor(Math.random() * 1000),
        hash: generateFakeHash(),
        timestamp: Date.now(),
        transactions: Math.floor(Math.random() * 3000) + 1000,
        size: Math.floor(Math.random() * 500000) + 1000000
      },
      tier: apiKey.tier
    };
    res.write(`data: ${JSON.stringify(blockData)}\n\n`);
  }, 30000); // Every 30 seconds for demo
  
  // Mempool updates (frequent)
  const mempoolInterval = setInterval(() => {
    const mempoolData = {
      event: "mempool_update",
      data: {
        size: Math.floor(Math.random() * 1000) + 3500,
        fee_estimate: Math.floor(Math.random() * 20) + 10,
        pending_transactions: Math.floor(Math.random() * 50) + 10
      },
      tier: apiKey.tier
    };
    res.write(`data: ${JSON.stringify(mempoolData)}\n\n`);
  }, 5000); // Every 5 seconds

  intervals.push(blockInterval, mempoolInterval);

  // Cleanup on disconnect
  req.on('close', () => {
    intervals.forEach(interval => clearInterval(interval));
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

function generateFakeHash(): string {
  const chars = '0123456789abcdef';
  let hash = '00000000000000000';
  for (let i = 17; i < 64; i++) {
    hash += chars[Math.floor(Math.random() * chars.length)];
  }
  return hash;
}

export default withAuth(handler);
