# API Key Generation - Enhanced Implementation

## Overview

Enhanced API key generation system for Bitcoin Sprint with improved operational visibility, security consistency, and tier-specific identification.

## Key Format

```
{prefix}_{base64url_random_bytes}
```

### Examples

```bash
# Basic keys
sprint_fqz9XjX6ytY0jAqZDjR1OeL12rbmZrF6P0pB5oU3qZ0

# Tier-specific keys
sprint-free_Ab3kL9mN2oP4qR7sT1uV8wX5yZ6aC9dF2gH4jK8mN1oP
sprint-pro_pQ3kL9mN2oP4qR7sT1uV8wX5yZ6aC9dF2gH4jK8mN1oZ
sprint-ent_zX7kL9mN2oP4qR7sT1uV8wX5yZ6aC9dF2gH4jK8mN1oY
sprint-entplus_mM5kL9mN2oP4qR7sT1uV8wX5yZ6aC9dF2gH4jK8mN1oQ
```

## Security Features

### Cryptographic Strength
- **256-bit entropy**: Uses `crypto.randomBytes(32)` for maximum security
- **Uniform distribution**: No bias in random generation
- **Non-predictable**: Each key is cryptographically unique

### Encoding
- **base64url**: URL-safe encoding (no padding issues)
- **Consistent length**: Random portion is always 43 characters
- **No special chars**: Safe for URLs, headers, and config files

## Operational Benefits

### Log Analysis
```bash
# Easy identification in logs
grep "sprint-pro" application.log
grep "sprint-ent" access.log

# Tier-specific monitoring
tail -f logs/api.log | grep "sprint-free"
```

### Monitoring & Alerts
```bash
# Rate limiting by tier
if [[ $api_key == sprint-free* ]]; then
    rate_limit=100
elif [[ $api_key == sprint-pro* ]]; then
    rate_limit=2000
fi
```

### Database Queries
```sql
-- Find all Enterprise keys
SELECT * FROM api_keys WHERE key LIKE 'sprint-ent%';

-- Tier usage analysis  
SELECT 
    CASE 
        WHEN key LIKE 'sprint-free%' THEN 'FREE'
        WHEN key LIKE 'sprint-pro%' THEN 'PRO'
        WHEN key LIKE 'sprint-ent%' THEN 'ENTERPRISE'
        WHEN key LIKE 'sprint-entplus%' THEN 'ENTERPRISE_PLUS'
    END as tier,
    COUNT(*) as count
FROM api_keys GROUP BY tier;
```

## Implementation

### Functions

#### `generateApiKey(prefix = "sprint"): string`
Basic key generation with custom prefix support.

```typescript
// Default
const key = generateApiKey(); 
// Result: sprint_fqz9XjX6ytY0jAqZDjR1OeL12rbmZrF6P0pB5oU3qZ0

// Custom prefix
const key = generateApiKey("custom");
// Result: custom_fqz9XjX6ytY0jAqZDjR1OeL12rbmZrF6P0pB5oU3qZ0
```

#### `generateTierApiKey(tier: string): string`
Tier-aware key generation for operational visibility.

```typescript
const freeKey = generateTierApiKey("FREE");
// Result: sprint-free_fqz9XjX6ytY0jAqZDjR1OeL12rbmZrF6P0pB5oU3qZ0

const proKey = generateTierApiKey("PRO");
// Result: sprint-pro_fqz9XjX6ytY0jAqZDjR1OeL12rbmZrF6P0pB5oU3qZ0
```

### Tier Prefixes

| Tier | Prefix | Use Case |
|------|--------|----------|
| FREE | `sprint-free` | Basic users, monitoring resource usage |
| PRO | `sprint-pro` | Professional users, higher limits |
| ENTERPRISE | `sprint-ent` | Enterprise customers, full features |
| ENTERPRISE_PLUS | `sprint-entplus` | Premium enterprise, dedicated infra |

## Migration Guide

### From Old Format
```typescript
// Old: Variable length, no prefix
const oldKey = crypto.randomBytes(32).toString("base64url");
// Example: fqz9XjX6ytY0jAqZDjR1OeL12rbmZrF6P0pB5oU3qZ0

// New: Consistent format with prefix
const newKey = generateTierApiKey("PRO");
// Example: sprint-pro_fqz9XjX6ytY0jAqZDjR1OeL12rbmZrF6P0pB5oU3qZ0
```

### Database Updates
- Existing keys continue to work (backward compatible)
- New keys use enhanced format
- Update monitoring to recognize both formats during transition

## Best Practices

### Storage
- Store in environment variables: `BITCOIN_SPRINT_API_KEY`
- Keep in secure config files: `config.json`
- Never commit to version control

### Usage
```bash
# In shell scripts
API_KEY="sprint-pro_xyz..."
curl -H "Authorization: Bearer $API_KEY" api.bitcoinsprint.com

# In config files
{
  "license_key": "sprint-ent_abc123...",
  "endpoints": {...}
}
```

### Monitoring
- Set up alerts for tier-specific usage patterns
- Monitor rate limits by key prefix
- Track API adoption by tier

## Technical Specifications

### Length Analysis
```
Total length: prefix + "_" + 43 chars
- sprint-free_: 15 + 43 = 58 characters
- sprint-pro_: 14 + 43 = 57 characters  
- sprint-ent_: 14 + 43 = 57 characters
- sprint-entplus_: 18 + 43 = 61 characters
```

### Character Set
- Prefix: `[a-z-]` (lowercase letters and hyphens)
- Separator: `_` (underscore)
- Random: `[A-Za-z0-9_-]` (base64url alphabet)

### Entropy Distribution
- Total bits: 256 (from 32 random bytes)
- Prefix bits: 0 (non-secret, operational metadata)
- Effective security: 256 bits (unchanged from original)

## Testing

Run the test suite:
```bash
node test-key-generation.js
```

Expected output demonstrates:
- Consistent length across tiers
- Proper prefix application
- Cryptographic randomness
- Operational benefits
