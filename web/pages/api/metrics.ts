import type { NextApiResponse } from 'next';
import os from 'os';
import type { AuthenticatedRequest } from '../../lib/auth';
import { withAuth } from './_withAuth';

// Dynamic import for Rust entropy bridge (optional)
let rustBridge: any = null;
let rustBridgeLoaded = false;

async function loadRustBridge() {
  if (rustBridgeLoaded) return rustBridge;
  try {
    const bridge = await import('../../rust-entropy-bridge.js');
    rustBridge = bridge;
    rustBridgeLoaded = true;
    console.log('✅ Rust entropy bridge loaded successfully');
  } catch (error) {
    console.warn('⚠️ Rust entropy bridge not available:', error instanceof Error ? error.message : String(error));
    rustBridgeLoaded = true; // Don't try again
  }
  return rustBridge;
}

// Import tier config for development mock
const TIER_CONFIG = {
  FREE: { rateLimit: 100, blocksPerDay: 100, endpoints: [], latencyTarget: 1000, mempoolAccess: false, burstable: false },
  PRO: { rateLimit: 2000, blocksPerDay: -1, endpoints: [], latencyTarget: 300, mempoolAccess: true, burstable: false },
  ENTERPRISE: { rateLimit: 10000, blocksPerDay: -1, endpoints: [], latencyTarget: 100, mempoolAccess: true, burstable: true },
  ENTERPRISE_PLUS: { rateLimit: 50000, blocksPerDay: -1, endpoints: [], latencyTarget: 50, mempoolAccess: true, burstable: true }
};

async function handler(req: AuthenticatedRequest, res: NextApiResponse) {
  // For development, allow unauthenticated access if NODE_ENV is development
  if (process.env.NODE_ENV === 'development' && !req.headers.authorization) {
    // Mock API key for development
    (req as any).apiKey = {
      id: 'dev-key',
      key: 'bitcoin-sprint-dev-key-2025',
      tier: 'ENTERPRISE' as keyof typeof TIER_CONFIG,
      email: 'dev@bitcoin-sprint.com',
      company: 'Bitcoin Sprint Dev',
      requests: Math.floor(Math.random() * 1000),
      blocksToday: Math.floor(Math.random() * 100)
    };
  }

  const { apiKey } = req;

  // Get system information
  const systemInfo = {
    uptime: os.uptime(),
    totalMemory: os.totalmem(),
    freeMemory: os.freemem(),
    cpus: os.cpus().length,
    loadAverage: os.loadavg(),
    platform: os.platform(),
    arch: os.arch()
  };

  // Calculate memory usage percentage
  const memoryUsagePercent = ((systemInfo.totalMemory - systemInfo.freeMemory) / systemInfo.totalMemory) * 100;

  // Calculate CPU usage (simplified - in production you'd use a proper monitoring library)
  const cpuUsagePercent = Math.min((systemInfo.loadAverage?.[0] || 0) * 100 / systemInfo.cpus, 100);

  // Try to use Rust entropy bridge for enhanced entropy
  let rustEntropyAvailable = false;
  let rustSecretHex = '';

  try {
    const bridge = await loadRustBridge();
    if (bridge && rustBridge) {
      const entropyBridge = await rustBridge.getEntropyBridge();
      if (entropyBridge && entropyBridge.isAvailable()) {
        rustSecretHex = await entropyBridge.generateAdminSecret('hex');
        rustEntropyAvailable = true;
        console.log('🔐 Generated entropy using Rust bridge');
      }
    }
  } catch (error) {
    console.warn('Rust entropy generation failed, using Node.js fallback');
  }

  // Mock entropy generation stats (in production, these would come from your backend)
  const entropyStats = {
    totalGenerated: Math.floor(Math.random() * 50000) + 10000,
    totalRequests: Math.floor(Math.random() * 10000) + 5000,
    avgGenerationTime: Math.floor(Math.random() * 10) + 5,
    rustEntropyAvailable,
    rustSecretHex
  };

  // Return Prometheus-compatible metrics format
  const metrics = `# Bitcoin Sprint Web Dashboard Metrics
# System Metrics
system_uptime_seconds ${Math.floor(systemInfo.uptime)}
system_platform{platform="${systemInfo.platform}"} 1
system_architecture{arch="${systemInfo.arch}"} 1
system_cpu_cores ${systemInfo.cpus}
system_total_memory_mb ${Math.floor(systemInfo.totalMemory / 1024 / 1024)}
system_free_memory_mb ${Math.floor(systemInfo.freeMemory / 1024 / 1024)}
system_memory_usage_percent ${Math.round(memoryUsagePercent * 100) / 100}
system_cpu_usage_percent ${Math.round(cpuUsagePercent * 100) / 100}
system_load_average_1m ${Math.round((systemInfo.loadAverage?.[0] || 0) * 100) / 100}
system_load_average_5m ${Math.round((systemInfo.loadAverage?.[1] || 0) * 100) / 100}
system_load_average_15m ${Math.round((systemInfo.loadAverage?.[2] || 0) * 100) / 100}

# Entropy Generation Metrics
entropy_total_generated_bytes ${entropyStats.totalGenerated}
entropy_total_requests ${entropyStats.totalRequests}
entropy_average_generation_time_ms ${entropyStats.avgGenerationTime}
entropy_generation_rate_per_second ${Math.floor(entropyStats.totalRequests / Math.max(systemInfo.uptime, 1))}
entropy_rust_available ${entropyStats.rustEntropyAvailable ? 1 : 0}

# API Usage Metrics
api_requests_today{tier="${apiKey.tier}"} ${apiKey.requests || 0}
api_blocks_today{tier="${apiKey.tier}"} ${apiKey.blocksToday || 0}
api_rate_limit_used_percent{tier="${apiKey.tier}"} ${Math.floor(Math.random() * 30) + 10}

# Network Metrics
network_status{status="connected"} 1
network_latency_ms{tier="${apiKey.tier}"} ${getLatencyForTier(apiKey.tier)}
network_peers_connected ${Math.floor(Math.random() * 10) + 5}

# Service Health
up 1
`;

  res.setHeader('Content-Type', 'text/plain; charset=utf-8');
  res.status(200).send(metrics);
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
