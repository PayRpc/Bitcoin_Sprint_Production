// ============================================================================
// BITCOIN SPRINT API KEY VERIFICATION - QUICK REFERENCE
// ============================================================================

// 1. IMPORT VERIFICATION FUNCTIONS
// This is an example / quick-reference file. It's not part of the runtime API surface and
// may reference helper types for illustrative purposes. Silence TypeScript here to avoid
// blocking builds while keeping the doc source in-tree.
// @ts-nocheck
import { AuthenticatedRequest, withApiKeyAuth } from '../lib/apiKeyAuth';
import { validateApiKeyFormat, verifyApiKey } from '../lib/generateKey';

// 2. BASIC VERIFICATION (Manual)
async function manualVerification(req: NextApiRequest, res: NextApiResponse) {
  const token = req.headers.authorization?.substring(7); // Remove "Bearer "
  
  const result = await verifyApiKey(token);
  if (!result.valid) {
    return res.status(401).json({ error: result.reason });
  }
  
  // Token is valid - proceed with API logic
  const { apiKey, tier } = result;
  res.json({ user: apiKey?.email, tier });
}

// 3. MIDDLEWARE APPROACH (Recommended)
async function protectedHandler(req: AuthenticatedRequest, res: NextApiResponse) {
  // req.apiKey and req.tier are automatically available
  res.json({ 
    message: 'Success!',
    user: req.apiKey?.email,
    tier: req.tier 
  });
}

export default withApiKeyAuth(protectedHandler, {
  updateUsage: true,        // Track this request
  incrementBlocks: false,   // Don't count as block fetch
  requiredTier: 'PRO'      // Require PRO tier or higher
});

// 4. TIER-SPECIFIC ENDPOINTS
export const freeEndpoint = withApiKeyAuth(handler, { requiredTier: 'FREE' });     // All tiers
export const proEndpoint = withApiKeyAuth(handler, { requiredTier: 'PRO' });       // PRO, ENT, ENT+
export const enterpriseEndpoint = withApiKeyAuth(handler, { requiredTier: 'ENTERPRISE' }); // ENT, ENT+
export const enterprisePlusEndpoint = withApiKeyAuth(handler, { requiredTier: 'ENTERPRISE_PLUS' }); // ENT+ only

// 5. FORMAT VALIDATION (Fast, No Database)
const formatCheck = validateApiKeyFormat('sprint-pro_abc123...');
if (formatCheck.valid) {
  console.log('Format is valid, prefix:', formatCheck.prefix);
} else {
  console.log('Invalid format:', formatCheck.reason);
}

// 6. USAGE TRACKING
import { updateApiKeyUsage } from '../lib/generateKey';

// Track request only
await updateApiKeyUsage(apiKey, false);

// Track request + block fetch
await updateApiKeyUsage(apiKey, true);

// 7. ERROR HANDLING
/*
Format Errors (400):
- "Invalid token format: missing prefix separator"
- "Invalid prefix format: should contain only lowercase letters"
- "Invalid token format: random part should be 43 characters"

Authentication Errors (401):
- "Authentication required"
- "API key not found in database"
- "API key expired on 2025-09-25T00:00:00.000Z"
- "API key has been revoked"

Authorization Errors (403):
- "This endpoint requires PRO tier or higher. Your tier: FREE"
*/

// 8. CLIENT EXAMPLES

// JavaScript/TypeScript
const response = await fetch('/api/protected', {
  headers: {
    'Authorization': `Bearer ${apiKey}`,
    'Content-Type': 'application/json'
  }
});

// curl
// curl -H "Authorization: Bearer sprint-pro_xyz..." http://localhost:3000/api/protected

// Go HTTP Client
/*
req, _ := http.NewRequest("GET", "http://localhost:3000/api/protected", nil)
req.Header.Set("Authorization", "Bearer " + apiKey)
resp, err := client.Do(req)
*/

// 9. RATE LIMITING
import { checkRateLimit, getTierRateLimit } from '../lib/apiKeyAuth';

const limits = getTierRateLimit('PRO'); // { requestsPerMinute: 2000, blocksPerDay: Infinity }
const allowed = checkRateLimit(apiKey); // { allowed: true/false, reason?, resetTime? }

// 10. KEY PREFIXES & TIERS
/*
sprint-free_     → FREE tier
sprint-pro_      → PRO tier  
sprint-ent_      → ENTERPRISE tier
sprint-entplus_  → ENTERPRISE_PLUS tier
sprint_          → Generic (legacy)
*/

// ============================================================================
// COMMON PATTERNS
// ============================================================================

// Pattern 1: Public endpoint (no auth)
export default function publicHandler(req, res) {
  res.json({ message: 'No authentication required' });
}

// Pattern 2: Optional auth (show different data based on tier)
export default async function optionalAuthHandler(req, res) {
  const token = req.headers.authorization?.substring(7);
  let tier = 'ANONYMOUS';
  
  if (token) {
    const verification = await verifyApiKey(token);
    if (verification.valid) {
      tier = verification.tier!;
      await updateApiKeyUsage(token);
    }
  }
  
  res.json({ 
    data: getDataForTier(tier),
    tier 
  });
}

// Pattern 3: Required auth with usage tracking
export default withApiKeyAuth(async (req, res) => {
  // Automatically authenticated and usage tracked
  res.json({ data: getProtectedData() });
}, { updateUsage: true });

// Pattern 4: Enterprise features only
export default withApiKeyAuth(async (req, res) => {
  res.json({ 
    advancedAnalytics: getEnterpriseAnalytics(),
    tier: req.tier 
  });
}, { 
  requiredTier: 'ENTERPRISE',
  updateUsage: true,
  incrementBlocks: false 
});

// Pattern 5: Block fetching endpoint
export default withApiKeyAuth(async (req, res) => {
  const blockData = await fetchBlockData(req.query.blockHash);
  res.json(blockData);
}, { 
  updateUsage: true,
  incrementBlocks: true  // Count as block fetch
});

// ============================================================================
