import { NextApiRequest, NextApiResponse } from 'next';
import { ApiKeyValidation, updateApiKeyUsage, verifyApiKey } from './generateKey';

export interface AuthenticatedRequest extends NextApiRequest {
  apiKey?: ApiKeyValidation['apiKey'];
  tier?: string;
}

/**
 * Express/Next.js middleware for API key authentication.
 * Validates API key from Authorization header or query parameter.
 * 
 * Usage:
 * ```typescript
 * export default async function handler(req: NextApiRequest, res: NextApiResponse) {
 *   const authResult = await authenticateApiKey(req, res);
 *   if (!authResult.success) return; // Response already sent
 *   
 *   // API key is valid, proceed with request
 *   const { apiKey, tier } = authResult;
 *   // ... your API logic
 * }
 * ```
 */
export async function authenticateApiKey(
  req: NextApiRequest, 
  res: NextApiResponse,
  options: { 
    updateUsage?: boolean;
    incrementBlocks?: boolean;
    requiredTier?: string;
  } = {}
): Promise<{ 
  success: boolean; 
  apiKey?: ApiKeyValidation['apiKey']; 
  tier?: string;
}> {
  
  try {
    // Extract API key from Authorization header or query parameter
    let token: string | undefined;
    
    // Check Authorization header (Bearer token)
    const authHeader = req.headers.authorization;
    if (authHeader && authHeader.startsWith('Bearer ')) {
      token = authHeader.substring(7);
    }
    
    // Fallback to query parameter
    if (!token && req.query.api_key) {
      token = Array.isArray(req.query.api_key) ? req.query.api_key[0] : req.query.api_key;
    }
    
    // Fallback to POST body
    if (!token && req.body && req.body.api_key) {
      token = req.body.api_key;
    }

    if (!token) {
      res.status(401).json({ 
        error: 'Authentication required',
        message: 'Provide API key in Authorization header (Bearer token) or api_key parameter'
      });
      return { success: false };
    }

    // Verify the API key
    const validation = await verifyApiKey(token);
    
    if (!validation.valid) {
      res.status(401).json({
        error: 'Invalid API key',
        message: validation.reason
      });
      return { success: false };
    }

    // Check tier requirement if specified
    if (options.requiredTier && !isAuthorizedTier(validation.tier!, options.requiredTier)) {
      res.status(403).json({
        error: 'Insufficient permissions',
        message: `This endpoint requires ${options.requiredTier} tier or higher. Your tier: ${validation.tier}`
      });
      return { success: false };
    }

    // Update usage statistics if requested
    if (options.updateUsage) {
      await updateApiKeyUsage(token, options.incrementBlocks);
    }

    // Extend request object with API key info
    (req as AuthenticatedRequest).apiKey = validation.apiKey;
    (req as AuthenticatedRequest).tier = validation.tier;

    return { 
      success: true, 
      apiKey: validation.apiKey, 
      tier: validation.tier 
    };

  } catch (error) {
    console.error('[authenticateApiKey] Error:', error);
    res.status(500).json({
      error: 'Authentication error',
      message: 'Internal server error during authentication'
    });
    return { success: false };
  }
}

/**
 * Higher-order function that wraps API routes with authentication.
 * 
 * Usage:
 * ```typescript
 * export default withApiKeyAuth(async function handler(req: AuthenticatedRequest, res: NextApiResponse) {
 *   // req.apiKey and req.tier are now available
 *   res.json({ user: req.apiKey?.email, tier: req.tier });
 * }, { requiredTier: 'PRO', updateUsage: true });
 * ```
 */
export function withApiKeyAuth(
  handler: (req: AuthenticatedRequest, res: NextApiResponse) => Promise<void> | void,
  options: {
    updateUsage?: boolean;
    incrementBlocks?: boolean;
    requiredTier?: string;
  } = {}
) {
  return async (req: NextApiRequest, res: NextApiResponse) => {
    const authResult = await authenticateApiKey(req, res, options);
    if (!authResult.success) {
      return; // Response already sent by authenticateApiKey
    }

    // Call the original handler with extended request
    return handler(req as AuthenticatedRequest, res);
  };
}

/**
 * Tier-based rate limiting helper.
 * Returns the rate limit for a given tier.
 */
export function getTierRateLimit(tier: string): { requestsPerMinute: number; blocksPerDay: number } {
  const limits = {
    'FREE': { requestsPerMinute: 100, blocksPerDay: 100 },
    'PRO': { requestsPerMinute: 2000, blocksPerDay: Infinity },
    'ENTERPRISE': { requestsPerMinute: 20000, blocksPerDay: Infinity },
    'ENTERPRISE_PLUS': { requestsPerMinute: 100000, blocksPerDay: Infinity }
  };

  return limits[tier as keyof typeof limits] || limits['FREE'];
}

/**
 * Check if a user's tier meets the minimum required tier.
 * Tier hierarchy: FREE < PRO < ENTERPRISE < ENTERPRISE_PLUS
 */
export function isAuthorizedTier(userTier: string, requiredTier: string): boolean {
  const tierLevels = {
    'FREE': 1,
    'PRO': 2,
    'ENTERPRISE': 3,
    'ENTERPRISE_PLUS': 4
  };

  const userLevel = tierLevels[userTier as keyof typeof tierLevels] || 0;
  const requiredLevel = tierLevels[requiredTier as keyof typeof tierLevels] || 0;

  return userLevel >= requiredLevel;
}

/**
 * Check if API key has exceeded rate limits.
 * This is a simple implementation - for production, use Redis or similar.
 */
export function checkRateLimit(apiKey: ApiKeyValidation['apiKey']): { 
  allowed: boolean; 
  reason?: string;
  resetTime?: Date;
} {
  if (!apiKey) {
    return { allowed: false, reason: 'No API key provided' };
  }

  const limits = getTierRateLimit(apiKey.tier);
  
  // Check daily blocks limit (simple implementation)
  if (apiKey.blocksToday >= limits.blocksPerDay) {
    const tomorrow = new Date();
    tomorrow.setDate(tomorrow.getDate() + 1);
    tomorrow.setHours(0, 0, 0, 0);
    
    return { 
      allowed: false, 
      reason: `Daily blocks limit exceeded (${limits.blocksPerDay})`,
      resetTime: tomorrow
    };
  }

  // For requests per minute, you'd need a more sophisticated sliding window
  // This is a simplified check - implement proper rate limiting in production
  
  return { allowed: true };
}
