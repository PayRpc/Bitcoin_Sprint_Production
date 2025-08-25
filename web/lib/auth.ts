import { PrismaClient } from "@prisma/client";
import type { NextApiRequest, NextApiResponse } from 'next';

// Tier configuration with realistic Bitcoin infrastructure limits
const TIER_CONFIG = {
  FREE: {
    rateLimit: 100, // req/minute
    blocksPerDay: 100,
    endpoints: ['/api/status', '/api/latest'] as string[],
    latencyTarget: 1000, // ms
    mempoolAccess: false,
    burstable: false
  },
  PRO: {
    rateLimit: 2000, // req/minute  
    blocksPerDay: -1, // unlimited
    endpoints: ['/api/status', '/api/latest', '/api/metrics'] as string[],
    latencyTarget: 300, // ms
    mempoolAccess: true,
    burstable: false
  },
  ENTERPRISE: {
    rateLimit: 20000, // req/minute
    blocksPerDay: -1, // unlimited
    endpoints: ['/api/status', '/api/latest', '/api/metrics', '/api/predictive', '/api/stream', '/api/v1/license/info', '/api/v1/analytics/summary'] as string[],
    latencyTarget: 200, // ms
    mempoolAccess: true,
    burstable: true
  },
  ENTERPRISE_PLUS: {
    rateLimit: 100000, // negotiated
    blocksPerDay: -1, // unlimited
    endpoints: ['*'] as string[], // all endpoints
    latencyTarget: 100, // ms
    mempoolAccess: true,
    burstable: true,
    dedicatedInfra: true
  }
};

// Rate limiting store (in production, use Redis)
const rateLimitStore = new Map<string, { count: number; resetTime: number }>();
const dailyBlockStore = new Map<string, { count: number; resetTime: number }>();

// Prisma client singleton
declare global {
  var prisma: PrismaClient | undefined;
}

const prisma: PrismaClient = global.prisma || new PrismaClient();
if (process.env.NODE_ENV !== "production") global.prisma = prisma;

interface AuthenticatedRequest extends NextApiRequest {
  apiKey: {
    id: string;
    key: string;
    tier: keyof typeof TIER_CONFIG;
    email: string;
    company?: string;
    requests: number;
    blocksToday: number;
  };
}

export async function authMiddleware(
  req: NextApiRequest, 
  res: NextApiResponse, 
  next: () => void
) {
  try {
    // Extract API key from Authorization header
    const authHeader = req.headers.authorization;
    if (!authHeader || !authHeader.startsWith('Bearer ')) {
      return res.status(401).json({
        error: 'Unauthorized',
        message: 'API key required. Include "Authorization: Bearer <your-api-key>" header.'
      });
    }

    const apiKey = authHeader.substring(7);
    
    // Validate API key in database
    const keyRecord = await prisma.apiKey.findUnique({
      where: { key: apiKey }
    });

    if (!keyRecord || keyRecord.revoked) {
      return res.status(401).json({
        error: 'Invalid API Key',
        message: 'API key not found or has been revoked.'
      });
    }

    // Check expiration
    if (keyRecord.expiresAt < new Date()) {
      return res.status(401).json({
        error: 'Expired API Key',
        message: 'API key has expired. Please generate a new one.'
      });
    }

    const tierConfig = TIER_CONFIG[keyRecord.tier as keyof typeof TIER_CONFIG];
    if (!tierConfig) {
      return res.status(500).json({
        error: 'Invalid Tier',
        message: 'API key has invalid tier configuration.'
      });
    }

    // Check endpoint access
    const requestPath = req.url || '';
    const hasEndpointAccess = tierConfig.endpoints.includes('*') || 
      tierConfig.endpoints.some(endpoint => requestPath.startsWith(endpoint));

    if (!hasEndpointAccess) {
      return res.status(403).json({
        error: 'Insufficient Tier',
        message: `Your ${keyRecord.tier} tier does not have access to this endpoint. Upgrade to access more features.`,
        currentTier: keyRecord.tier,
        allowedEndpoints: tierConfig.endpoints
      });
    }

    // Rate limiting check
    const now = Date.now();
    const minuteWindow = Math.floor(now / 60000);
    const rateLimitKey = `${apiKey}:${minuteWindow}`;
    
    const currentRateLimit = rateLimitStore.get(rateLimitKey) || { count: 0, resetTime: minuteWindow };
    
    if (currentRateLimit.count >= tierConfig.rateLimit) {
      const resetIn = ((minuteWindow + 1) * 60000 - now) / 1000;
      return res.status(429).json({
        error: 'Rate Limit Exceeded',
        message: `Rate limit of ${tierConfig.rateLimit} requests per minute exceeded.`,
        retryAfter: Math.ceil(resetIn),
        tier: keyRecord.tier
      });
    }

    // Daily block limit check (for non-unlimited tiers)
    if (tierConfig.blocksPerDay > 0) {
      const dayWindow = Math.floor(now / (24 * 60 * 60 * 1000));
      const dailyKey = `${apiKey}:blocks:${dayWindow}`;
      const currentDailyBlocks = dailyBlockStore.get(dailyKey) || { count: 0, resetTime: dayWindow };
      
      if (currentDailyBlocks.count >= tierConfig.blocksPerDay) {
        return res.status(429).json({
          error: 'Daily Block Limit Exceeded',
          message: `Daily limit of ${tierConfig.blocksPerDay} blocks exceeded.`,
          tier: keyRecord.tier,
          upgradeRequired: true
        });
      }
    }

    // Update usage counters
    currentRateLimit.count++;
    rateLimitStore.set(rateLimitKey, currentRateLimit);

    // Update database usage stats (async, don't block request) - simplified for now
    // TODO: Add proper usage tracking once Prisma is regenerated

    // Attach authenticated API key data to request
    (req as AuthenticatedRequest).apiKey = {
      id: keyRecord.id,
      key: keyRecord.key,
      tier: keyRecord.tier as keyof typeof TIER_CONFIG,
      email: keyRecord.email,
      company: keyRecord.company || undefined,
      requests: 0, // Will be properly tracked after schema update
      blocksToday: 0 // Will be properly tracked after schema update
    };

    // Continue to protected handler
    next();

  } catch (error) {
    console.error('[AUTH] Middleware error:', error);
    return res.status(500).json({
      error: 'Authentication Error',
      message: 'Internal authentication error. Please try again.'
    });
  }
}

// Export authenticated request type for handlers
export type { AuthenticatedRequest };
