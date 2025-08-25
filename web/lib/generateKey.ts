import { PrismaClient } from "@prisma/client";
import crypto from "crypto";

// Prisma client singleton for Next.js dev hot-reload safety
declare global {
  // eslint-disable-next-line no-var
  var prisma: PrismaClient | undefined
}

const prisma: PrismaClient = global.prisma || new PrismaClient()
if (process.env.NODE_ENV !== "production") global.prisma = prisma

/**
 * Generate a 256-bit random API key with product prefix.
 * 
 * @param prefix - Product identifier prefix (default: "sprint")
 * @returns API key in format: {prefix}_{base64url_random_bytes}
 * 
 * Benefits:
 * - Security: Full 256-bit entropy in the random portion
 * - Ops visibility: Easy to identify Bitcoin Sprint keys in logs
 * - URL-safe: Uses base64url encoding (no padding issues)
 * - Consistent: Always 43 chars + prefix length
 * 
 * Example: sprint_fqz9XjX6ytY0jAqZDjR1OeL12rbmZrF6P0pB5oU3qZ0
 */
export function generateApiKey(prefix = "sprint"): string {
  // Generate 256-bit (32 bytes) of cryptographically secure random data
  const randomBytes = crypto.randomBytes(32)
  
  // Convert to base64url (URL-safe, no padding)
  // 32 bytes -> 43 characters (no padding needed for 32 bytes)
  const randomPart = randomBytes.toString("base64url")
  
  return `${prefix}_${randomPart}`
}

/**
 * Generate API key with tier-specific prefix for operational visibility.
 * 
 * @param tier - API tier (FREE, PRO, ENTERPRISE, ENTERPRISE_PLUS)
 * @returns API key with tier-aware prefix
 */
export function generateTierApiKey(tier: string): string {
  const tierPrefixes = {
    'FREE': 'sprint-free',
    'PRO': 'sprint-pro', 
    'ENTERPRISE': 'sprint-ent',
    'ENTERPRISE_PLUS': 'sprint-entplus'
  };
  
  const prefix = tierPrefixes[tier as keyof typeof tierPrefixes] || 'sprint';
  return generateApiKey(prefix);
}

// ============================================================================
// API KEY VERIFICATION
// ============================================================================

export interface ApiKeyValidation {
  valid: boolean;
  reason?: string;
  apiKey?: {
    id: string;
    key: string;
    email: string;
    company?: string | null;
    tier: string;
    createdAt: Date;
    expiresAt: Date;
    revoked: boolean;
    lastUsedAt?: Date | null;
    requests: number;
    blocksToday: number;
  };
  tier?: string;
  prefix?: string;
}

/**
 * Extract and validate API key prefix.
 * 
 * @param token - API key to validate
 * @returns Object with validation status and extracted info
 */
export function validateApiKeyFormat(token: string): { valid: boolean; prefix?: string; reason?: string } {
  if (!token || typeof token !== 'string') {
    return { valid: false, reason: 'Token is required and must be a string' };
  }

  // Check for underscore separator
  if (!token.includes('_')) {
    return { valid: false, reason: 'Invalid token format: missing prefix separator' };
  }

  const parts = token.split('_');
  if (parts.length !== 2) {
    return { valid: false, reason: 'Invalid token format: should be prefix_randompart' };
  }

  const [prefix, randomPart] = parts;

  // Validate prefix format
  if (!prefix || !/^[a-z0-9-]+$/.test(prefix)) {
    return { valid: false, reason: 'Invalid prefix format: should contain only lowercase letters, numbers, and hyphens' };
  }

  // Validate random part length (should be 43 chars for base64url of 32 bytes)
  if (!randomPart || randomPart.length !== 43) {
    return { valid: false, reason: 'Invalid token format: random part should be 43 characters' };
  }

  // Validate base64url characters
  if (!/^[A-Za-z0-9_-]+$/.test(randomPart)) {
    return { valid: false, reason: 'Invalid token format: random part contains invalid characters' };
  }

  // Check if it's a known Bitcoin Sprint prefix
  const validPrefixes = ['sprint', 'sprint-free', 'sprint-pro', 'sprint-ent', 'sprint-entplus'];
  if (!validPrefixes.includes(prefix)) {
    return { valid: false, reason: `Unknown prefix: ${prefix}. Expected one of: ${validPrefixes.join(', ')}` };
  }

  return { valid: true, prefix };
}

/**
 * Comprehensive API key verification with database lookup.
 * 
 * @param token - API key to verify
 * @returns Promise with validation result including database record if valid
 */
export async function verifyApiKey(token: string): Promise<ApiKeyValidation> {
  try {
    // Step 1: Format validation
    const formatCheck = validateApiKeyFormat(token);
    if (!formatCheck.valid) {
      return {
        valid: false,
        reason: formatCheck.reason,
        prefix: formatCheck.prefix
      };
    }

    // Step 2: Database lookup
    const apiKeyRecord = await prisma.apiKey.findUnique({
      where: { key: token }
    });

    if (!apiKeyRecord) {
      return {
        valid: false,
        reason: 'API key not found in database',
        prefix: formatCheck.prefix
      };
    }

    // Step 3: Expiration check
    const now = new Date();
    if (apiKeyRecord.expiresAt < now) {
      return {
        valid: false,
        reason: `API key expired on ${apiKeyRecord.expiresAt.toISOString()}`,
        prefix: formatCheck.prefix,
        apiKey: {
          ...apiKeyRecord,
          requests: (apiKeyRecord as any).requests ?? 0,
          blocksToday: (apiKeyRecord as any).blocksToday ?? 0
        }
      };
    }

    // Step 4: Revocation check
    if (apiKeyRecord.revoked) {
      return {
        valid: false,
        reason: 'API key has been revoked',
        prefix: formatCheck.prefix,
        apiKey: {
          ...apiKeyRecord,
          requests: (apiKeyRecord as any).requests ?? 0,
          blocksToday: (apiKeyRecord as any).blocksToday ?? 0
        }
      };
    }

    // Step 5: Extract tier from prefix for validation
    const tierFromPrefix = extractTierFromPrefix(formatCheck.prefix!);
    
    // Optional: Verify tier consistency (prefix should match database tier)
    if (tierFromPrefix && tierFromPrefix !== apiKeyRecord.tier) {
      console.warn(`Tier mismatch: prefix suggests ${tierFromPrefix}, database has ${apiKeyRecord.tier}`);
    }

    // All checks passed - valid key
    return {
      valid: true,
      apiKey: {
        ...apiKeyRecord,
        requests: (apiKeyRecord as any).requests ?? 0,
        blocksToday: (apiKeyRecord as any).blocksToday ?? 0
      },
      tier: apiKeyRecord.tier,
      prefix: formatCheck.prefix
    };

  } catch (error) {
    console.error('[verifyApiKey] Database error:', error);
    return {
      valid: false,
      reason: 'Database error during verification'
    };
  }
}

/**
 * Extract tier information from API key prefix.
 * 
 * @param prefix - API key prefix
 * @returns Tier string or null if not tier-specific
 */
export function extractTierFromPrefix(prefix: string): string | null {
  const tierMap: Record<string, string | null> = {
    'sprint-free': 'FREE',
    'sprint-pro': 'PRO',
    'sprint-ent': 'ENTERPRISE', 
    'sprint-entplus': 'ENTERPRISE_PLUS',
    'sprint': null // Generic prefix, no tier info
  };

  return tierMap[prefix] || null;
}

/**
 * Update API key usage statistics (call this after successful verification).
 * 
 * @param token - API key that was used
 * @param incrementBlocks - Whether to increment blocks count (default: false)
 */
export async function updateApiKeyUsage(token: string, incrementBlocks = false): Promise<void> {
  try {
    const updateData: any = {
      lastUsedAt: new Date(),
      requests: { increment: 1 },
      requestsToday: { increment: 1 }
    };

    if (incrementBlocks) {
      updateData.blocksToday = { increment: 1 };
    }

    await prisma.apiKey.update({
      where: { key: token },
      data: updateData
    });
  } catch (error) {
    console.error('[updateApiKeyUsage] Failed to update usage:', error);
    // Don't throw - usage tracking shouldn't break API functionality
  }
}

/**
 * Reset daily counters (requestsToday, blocksToday) at midnight UTC.
 * This is a helper - in production use a scheduled job (cron) or DB trigger.
 */
export async function resetDailyCountersIfNeeded() {
  try {
    // For simplicity, if any key has lastUsedAt before today UTC, reset its counters.
    // This is not perfect but acceptable for demo/local setups.
    const now = new Date();
    const today = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate()));
    await prisma.apiKey.updateMany({
      where: { lastUsedAt: { lt: today } },
      data: { requestsToday: 0, blocksToday: 0 }
    });
  } catch (err) {
    console.error('[resetDailyCountersIfNeeded] Failed:', err);
  }
}

/**
 * Renew an API key by extending its expiry date.
 * Returns the updated record or null on failure.
 */
export async function renewApiKey(token: string, extendDays = 30) {
  try {
    const rec = await prisma.apiKey.findUnique({ where: { key: token } });
    if (!rec) return null;
    const now = new Date();
    const base = rec.expiresAt && rec.expiresAt > now ? rec.expiresAt : now;
    const newExpiry = new Date(base.getTime() + extendDays * 24 * 60 * 60 * 1000);
    const updated = await prisma.apiKey.update({ where: { key: token }, data: { expiresAt: newExpiry } });
    return updated;
  } catch (err) {
    console.error('[renewApiKey] Failed:', err);
    return null;
  }
}

/**
 * Revoke an API key (mark as revoked in database).
 * 
 * @param token - API key to revoke
 * @returns Success status
 */
export async function revokeApiKey(token: string): Promise<{ success: boolean; reason?: string }> {
  try {
    const updated = await prisma.apiKey.update({
      where: { key: token },
      data: { revoked: true }
    });

    return { success: true };
  } catch (error) {
    console.error('[revokeApiKey] Failed to revoke key:', error);
    return { 
      success: false, 
      reason: error instanceof Error ? error.message : 'Database error'
    };
  }
}
