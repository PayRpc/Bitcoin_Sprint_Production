import { NextApiRequest, NextApiResponse } from 'next';

interface TurboStatusResponse {
  tier: string;
  turboModeEnabled: boolean;
  writeDeadline: string;
  useSharedMemory: boolean;
  blockBufferSize: number;
  enableKernelBypass: boolean;
  useDirectP2P: boolean;
  useMemoryChannel: boolean;
  optimizeSystem: boolean;
  features: string[];
  performanceTargets: PerformanceTargets;
  timestamp: string;
}

interface PerformanceTargets {
  blockRelayLatency: string;
  writeDeadline: string;
  bufferStrategy: string;
  peerNotification: string;
}

// Get performance targets based on tier
function getPerformanceTargets(tier: string): PerformanceTargets {
  switch (tier) {
    case 'enterprise':
      return {
        blockRelayLatency: '<5ms (Enterprise)',
        writeDeadline: '200µs',
        bufferStrategy: 'Overwrite old events (never miss)',
        peerNotification: 'Zero-copy with kernel bypass',
      };
    case 'turbo':
      return {
        blockRelayLatency: '<10ms (Turbo)',
        writeDeadline: '500µs',
        bufferStrategy: 'Overwrite old events (never miss)',
        peerNotification: 'Zero-copy shared memory',
      };
    case 'business':
      return {
        blockRelayLatency: '<50ms (Business)',
        writeDeadline: '1s',
        bufferStrategy: 'Best effort delivery',
        peerNotification: 'Standard TCP relay',
      };
    case 'pro':
      return {
        blockRelayLatency: '<100ms (Pro)',
        writeDeadline: '1.5s',
        bufferStrategy: 'Best effort delivery',
        peerNotification: 'Standard TCP relay',
      };
    default: // free
      return {
        blockRelayLatency: '<500ms (Free)',
        writeDeadline: '2s',
        bufferStrategy: 'Drop on full buffer',
        peerNotification: 'Standard TCP relay with limits',
      };
  }
}

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse<TurboStatusResponse>
) {
  if (req.method !== 'GET') {
    return res.status(405).json({} as any);
  }

  try {
    // Get tier from environment or default to free
    const tier = process.env.TIER || 'free';
    const turboEnabled = tier === 'turbo' || tier === 'enterprise';

    // Build feature list based on environment variables
    const features: string[] = [];
    if (process.env.USE_SHARED_MEMORY === 'true') {
      features.push('Shared Memory');
    }
    if (process.env.USE_DIRECT_P2P === 'true') {
      features.push('Direct P2P');
    }
    if (process.env.USE_MEMORY_CHANNEL === 'true') {
      features.push('Memory Channel');
    }
    if (process.env.OPTIMIZE_SYSTEM === 'true') {
      features.push('System Optimizations');
    }
    if (process.env.ENABLE_KERNEL_BYPASS === 'true') {
      features.push('Kernel Bypass');
    }

    // Get buffer size based on tier
    let blockBufferSize = 1024; // default
    switch (tier) {
      case 'enterprise':
        blockBufferSize = 4096;
        break;
      case 'turbo':
        blockBufferSize = 2048;
        break;
      case 'business':
        blockBufferSize = 1536;
        break;
      case 'pro':
        blockBufferSize = 1280;
        break;
      default:
        blockBufferSize = 512;
        break;
    }

    // Get write deadline based on tier
    let writeDeadline = '2s'; // default
    switch (tier) {
      case 'enterprise':
        writeDeadline = '200µs';
        break;
      case 'turbo':
        writeDeadline = '500µs';
        break;
      case 'business':
        writeDeadline = '1s';
        break;
      case 'pro':
        writeDeadline = '1.5s';
        break;
    }

    const response: TurboStatusResponse = {
      tier,
      turboModeEnabled: turboEnabled,
      writeDeadline,
      useSharedMemory: process.env.USE_SHARED_MEMORY === 'true',
      blockBufferSize,
      enableKernelBypass: process.env.ENABLE_KERNEL_BYPASS === 'true',
      useDirectP2P: process.env.USE_DIRECT_P2P === 'true',
      useMemoryChannel: process.env.USE_MEMORY_CHANNEL === 'true',
      optimizeSystem: process.env.OPTIMIZE_SYSTEM === 'true',
      features,
      performanceTargets: getPerformanceTargets(tier),
      timestamp: new Date().toISOString(),
    };

    res.status(200).json(response);
  } catch (error) {
    console.error('Error getting turbo status:', error);
    res.status(500).json({} as any);
  }
}
