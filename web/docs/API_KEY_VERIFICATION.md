# API Key Verification System

## Overview

Comprehensive API key verification system for Bitcoin Sprint that validates prefixes, checks database existence, verifies expiration and revocation status, and provides middleware for easy integration.

## Core Functions

### `verifyApiKey(token: string): Promise<ApiKeyValidation>`

Main verification function that performs comprehensive validation:

```typescript
const result = await verifyApiKey('sprint-pro_xyz123...');

if (result.valid) {
  console.log('Key is valid!');
  console.log('User:', result.apiKey?.email);
  console.log('Tier:', result.tier);
} else {
  console.log('Invalid key:', result.reason);
}
```

**Validation Steps:**
1. ✅ **Format validation** - prefix structure, length, character set
2. ✅ **Database lookup** - key exists in database
3. ✅ **Expiration check** - key hasn't expired
4. ✅ **Revocation check** - key hasn't been revoked
5. ✅ **Tier consistency** - prefix matches database tier (warning only)

### `validateApiKeyFormat(token: string)`

Fast prefix and format validation without database calls:

```typescript
const check = validateApiKeyFormat('sprint-ent_abc123xyz...');
// Returns: { valid: true, prefix: 'sprint-ent' }
```

**Checks:**
- Prefix format: `[a-z0-9-]+`
- Structure: `prefix_randompart`
- Random part: exactly 43 characters (base64url)
- Known prefix: must be valid Bitcoin Sprint prefix

### `updateApiKeyUsage(token: string, incrementBlocks?)`

Update usage statistics after successful API calls:

```typescript
// Track request
await updateApiKeyUsage(apiKey);

// Track request + block fetch
await updateApiKeyUsage(apiKey, true);
```

## Middleware Integration

### `withApiKeyAuth()` - Higher-Order Function

Wrap API routes with automatic authentication:

```typescript
import { withApiKeyAuth, AuthenticatedRequest } from '../../lib/apiKeyAuth';

async function handler(req: AuthenticatedRequest, res: NextApiResponse) {
  // req.apiKey and req.tier are now available
  res.json({ 
    user: req.apiKey?.email, 
    tier: req.tier 
  });
}

export default withApiKeyAuth(handler, {
  updateUsage: true,        // Track usage automatically
  incrementBlocks: false,   // Don't count as block request
  requiredTier: 'PRO'      // Require PRO tier or higher
});
```

### `authenticateApiKey()` - Direct Middleware

For more control over the authentication flow:

```typescript
export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  const auth = await authenticateApiKey(req, res, { requiredTier: 'ENTERPRISE' });
  if (!auth.success) return; // Response already sent
  
  // Proceed with authenticated request
  const { apiKey, tier } = auth;
}
```

## Authentication Methods

The middleware supports multiple authentication methods:

### 1. Authorization Header (Recommended)
```bash
curl -H "Authorization: Bearer sprint-pro_xyz123..." /api/endpoint
```

### 2. Query Parameter
```bash
curl "/api/endpoint?api_key=sprint-pro_xyz123..."
```

### 3. POST Body
```bash
curl -X POST "/api/endpoint" -d '{"api_key":"sprint-pro_xyz123..."}'
```

## Tier Hierarchy & Access Control

### Tier Levels
```
FREE (1) < PRO (2) < ENTERPRISE (3) < ENTERPRISE_PLUS (4)
```

### Access Rules
- `requiredTier: 'PRO'` allows PRO, ENTERPRISE, and ENTERPRISE_PLUS
- `requiredTier: 'ENTERPRISE'` allows ENTERPRISE and ENTERPRISE_PLUS only
- `requiredTier: 'FREE'` allows all tiers

### Rate Limits by Tier
```typescript
const limits = getTierRateLimit(tier);
// FREE: 100 req/min, 100 blocks/day
// PRO: 2,000 req/min, unlimited blocks
// ENTERPRISE: 20,000 req/min, unlimited blocks
// ENTERPRISE_PLUS: 100,000 req/min, unlimited blocks
```

## Error Responses

### Format Errors (400)
```json
{
  "error": "Invalid API key",
  "message": "Invalid token format: missing prefix separator"
}
```

### Authentication Errors (401)
```json
{
  "error": "Invalid API key", 
  "message": "API key not found in database"
}
```

### Authorization Errors (403)
```json
{
  "error": "Insufficient permissions",
  "message": "This endpoint requires PRO tier or higher. Your tier: FREE"
}
```

## Usage Examples

### Basic Protected Endpoint
```typescript
// pages/api/status.ts
import { withApiKeyAuth } from '../../lib/apiKeyAuth';

export default withApiKeyAuth(async (req, res) => {
  res.json({ status: 'Bitcoin Core connected', tier: req.tier });
}, { updateUsage: true });
```

### Tier-Specific Endpoint
```typescript
// pages/api/enterprise/analytics.ts
export default withApiKeyAuth(async (req, res) => {
  res.json({ 
    analytics: getAdvancedMetrics(),
    user: req.apiKey?.email 
  });
}, { 
  requiredTier: 'ENTERPRISE',
  updateUsage: true,
  incrementBlocks: false 
});
```

### Manual Verification
```typescript
// pages/api/custom-auth.ts
export default async function handler(req, res) {
  const token = req.headers.authorization?.substring(7);
  const verification = await verifyApiKey(token);
  
  if (!verification.valid) {
    return res.status(401).json({ error: verification.reason });
  }
  
  // Custom logic here
  await updateApiKeyUsage(token, true);
  res.json({ success: true });
}
```

## Go Integration

For Go services, you can verify keys by calling the Next.js verification endpoint:

```go
// Verify API key from Go service
func verifyApiKey(token string) (*ApiKeyInfo, error) {
    resp, err := http.Get(fmt.Sprintf(
        "http://localhost:3000/api/verify-key?api_key=%s", token))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("invalid API key")
    }
    
    var result ApiKeyInfo
    json.NewDecoder(resp.Body).Decode(&result)
    return &result, nil
}
```

## Database Schema

The verification system works with this Prisma schema:

```prisma
model ApiKey {
  id          String   @id @default(cuid())
  key         String   @unique
  email       String
  company     String?
  tier        Tier     @default(FREE)
  createdAt   DateTime @default(now())
  expiresAt   DateTime
  revoked     Boolean  @default(false)
  lastUsedAt  DateTime?
  requests    Int      @default(0)
  blocksToday Int      @default(0)
}
```

## Security Features

### ✅ **Cryptographic Validation**
- Keys maintain 256-bit entropy
- Secure random generation
- URL-safe base64url encoding

### ✅ **Database Security**
- Unique key constraint
- Expiration enforcement
- Revocation support
- Usage tracking

### ✅ **Access Control**
- Tier-based authorization
- Rate limiting awareness
- Request/block counting

### ✅ **Operational Security**
- Prefix-based identification
- Comprehensive logging
- Error handling

## Testing

Run the verification test suite:

```bash
cd web
node test-api-verification.js
```

Test with curl:

```bash
# Test authentication
curl -H "Authorization: Bearer sprint-free_xyz..." http://localhost:3000/api/verify-key

# Test tier requirement
curl -H "Authorization: Bearer sprint-pro_xyz..." http://localhost:3000/api/enterprise-analytics
```

## Performance Considerations

- **Format validation**: O(1) operation, very fast
- **Database lookup**: Single indexed query on unique key
- **Usage updates**: Async, non-blocking
- **Middleware overhead**: ~1-2ms per request

## Best Practices

1. **Always use HTTPS** in production
2. **Store keys securely** (environment variables, config files)
3. **Update usage statistics** for analytics
4. **Use tier requirements** to control access
5. **Handle errors gracefully** with proper HTTP status codes
6. **Log authentication events** for security monitoring
