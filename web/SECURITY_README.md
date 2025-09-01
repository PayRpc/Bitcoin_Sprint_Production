# Bitcoin Sprint Web Security Implementation

## Overview

This document describes the comprehensive security implementation for the Bitcoin Sprint web application, providing enterprise-grade security features using existing backend materials.

## 🛡️ Security Features Implemented

### 1. Authentication System
- **API Key Authentication**: Support for Bearer tokens and X-API-Key headers
- **Tier-Based Access**: Free, Pro, and Enterprise tiers with different capabilities
- **Secure Key Management**: Environment-based configuration with localStorage persistence

### 2. Rate Limiting
- **Tier-Based Limits**:
  - Free: 10 requests/minute
  - Pro: 100 requests/minute
  - Enterprise: 1000 requests/minute
- **In-Memory Store**: Efficient rate limiting with automatic cleanup
- **Request Tracking**: Real-time monitoring of API usage

### 3. Security Headers
- **X-Frame-Options**: Prevents clickjacking attacks
- **X-Content-Type-Options**: Prevents MIME type sniffing
- **X-Request-ID**: Unique request identification for debugging
- **CSP Headers**: Content Security Policy protection
- **HSTS**: HTTP Strict Transport Security

### 4. Input Validation
- **Request Validation**: Comprehensive input sanitization
- **Type Checking**: Runtime type validation for all inputs
- **Size Limits**: Tier-appropriate size restrictions
- **Format Validation**: Supported entropy formats (hex, base64, bytes)

### 5. Error Handling
- **Structured Errors**: Consistent error response format
- **Security Error Codes**: Prevent information leakage
- **Logging**: Comprehensive error logging for monitoring
- **Graceful Degradation**: Fallback handling for edge cases

## 📁 File Structure

```
web/
├── lib/
│   ├── security.ts          # Security utilities and validation
│   ├── api-client.ts        # Secure API client library
│   └── examples.js          # Usage examples and patterns
├── pages/
│   └── api/
│       ├── entropy.ts       # Secure entropy generation endpoint
│       └── docs.ts          # API documentation endpoint
├── middleware.ts            # Security middleware
├── .env.local              # Environment configuration
└── test-security.js        # Comprehensive test suite
```

## 🚀 Quick Start

### 1. Environment Setup

Create or update your `.env.local` file:

```bash
# API Keys for different tiers
FREE_API_KEY=free-api-key-changeme
PRO_API_KEY=pro-api-key-changeme
ENTERPRISE_API_KEY=enterprise-api-key-changeme

# Backend Configuration
BACKEND_URL=http://localhost:8080
API_TIMEOUT=30000

# Security Configuration
RATE_LIMIT_ENABLED=true
SECURITY_HEADERS_ENABLED=true
```

### 2. Start the Services

```bash
# Terminal 1: Start the Go backend
cd /path/to/bitcoin-sprint
go run cmd/sprintd/main.go

# Terminal 2: Start the web application
cd web
npm install
npm run dev
```

### 3. Test the Security Implementation

```bash
# Run the comprehensive test suite
node test-security.js
```

## 📚 API Usage

### Using the Secure API Client

```javascript
import { BitcoinSprintApiClient } from './lib/api-client.js';

const apiClient = new BitcoinSprintApiClient();

// Set your API key
apiClient.setApiKey('your-api-key-here');

// Generate entropy
const result = await apiClient.generateEntropy({
  size: 32,
  format: 'hex'
});

console.log('Entropy:', result.entropy);
console.log('Tier:', result.tier);
console.log('Rate Limit Remaining:', result.rateLimitRemaining);
```

### Direct API Calls

```javascript
// Free tier request
const response = await fetch('/api/entropy', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer free-api-key-changeme',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    size: 32,
    format: 'hex'
  })
});

const data = await response.json();
```

## 🔧 Configuration

### API Key Management

The system supports multiple ways to configure API keys:

1. **Environment Variables** (Recommended for production):
   ```bash
   FREE_API_KEY=your-free-key
   PRO_API_KEY=your-pro-key
   ENTERPRISE_API_KEY=your-enterprise-key
   ```

2. **Programmatic Configuration**:
   ```javascript
   apiClient.setApiKey('your-key');
   ```

3. **localStorage Persistence**:
   ```javascript
   // Keys are automatically saved to localStorage
   apiClient.setApiKey('your-key');
   // Key persists across browser sessions
   ```

### Rate Limiting Configuration

Rate limits are configured per tier in `lib/security.ts`:

```javascript
const RATE_LIMITS = {
  free: { requests: 10, windowMs: 60000 },      // 10/min
  pro: { requests: 100, windowMs: 60000 },     // 100/min
  enterprise: { requests: 1000, windowMs: 60000 } // 1000/min
};
```

### Security Headers Configuration

Security headers are defined in `lib/security.ts`:

```javascript
const SECURITY_HEADERS = {
  'X-Frame-Options': 'DENY',
  'X-Content-Type-Options': 'nosniff',
  'X-XSS-Protection': '1; mode=block',
  'Strict-Transport-Security': 'max-age=31536000; includeSubDomains',
  // ... more headers
};
```

## 🧪 Testing

### Running the Test Suite

```bash
# Run all security tests
node test-security.js
```

### Test Coverage

The test suite covers:

- ✅ **Authentication Tests**: No auth, invalid auth, valid tier auth
- ✅ **Rate Limiting Tests**: Triggering rate limits across tiers
- ✅ **Input Validation Tests**: Invalid sizes and formats
- ✅ **Security Headers Tests**: Presence of security headers
- ✅ **API Documentation Tests**: Documentation accessibility

### Manual Testing

```bash
# Test different authentication methods
curl -X POST http://localhost:3002/api/entropy \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer free-api-key-changeme" \
  -d '{"size": 32}'

# Test rate limiting
for i in {1..15}; do
  curl -X POST http://localhost:3002/api/entropy \
    -H "Authorization: Bearer free-api-key-changeme" \
    -d '{"size": 16}' &
done

# Test API documentation
curl http://localhost:3002/api/docs
```

## 📖 API Documentation

Access the interactive API documentation at:
```
http://localhost:3002/api/docs
```

The documentation includes:
- Endpoint specifications
- Authentication requirements
- Request/response examples
- Error codes and handling
- Rate limiting information

## 🔒 Security Best Practices

### 1. API Key Security
- Never commit API keys to version control
- Use environment variables in production
- Rotate keys regularly
- Monitor key usage patterns

### 2. Rate Limiting
- Implement client-side rate limiting as backup
- Monitor rate limit usage
- Alert on unusual patterns
- Consider per-user limits for enterprise

### 3. Input Validation
- Validate all inputs on both client and server
- Use allowlists for accepted values
- Sanitize user inputs
- Implement proper error messages

### 4. Error Handling
- Don't expose internal system details
- Use consistent error formats
- Log errors securely
- Implement proper monitoring

## 🚨 Troubleshooting

### Common Issues

**401 Unauthorized**
- Check API key configuration
- Verify key format (Bearer token)
- Ensure key matches tier requirements

**429 Rate Limited**
- Wait for rate limit reset
- Check your tier's limits
- Implement exponential backoff

**400 Bad Request**
- Validate input parameters
- Check size limits for your tier
- Verify format parameter

**500 Internal Server Error**
- Check backend service status
- Review server logs
- Verify network connectivity

### Debug Mode

Enable debug logging:

```javascript
// In your browser console
localStorage.setItem('DEBUG', 'true');

// Check API client status
apiClient.getRateLimitStatus();
apiClient.testConnection();
```

## 📈 Monitoring

### Key Metrics to Monitor

1. **Authentication Success Rate**
2. **Rate Limit Hit Rate**
3. **API Response Times**
4. **Error Rates by Type**
5. **Tier Usage Distribution**

### Logging

All security events are logged with:
- Request IDs for tracking
- User tiers for analytics
- Error types for debugging
- Rate limit status for monitoring

## 🔄 Updates and Maintenance

### Updating API Keys

```bash
# Update environment variables
echo "FREE_API_KEY=new-key-here" >> .env.local

# Restart the application
npm run dev
```

### Modifying Rate Limits

Edit `lib/security.ts` and update the `RATE_LIMITS` configuration.

### Adding New Security Features

1. Update `lib/security.ts` with new utilities
2. Modify `middleware.ts` for new middleware
3. Update API endpoints as needed
4. Add tests to `test-security.js`
5. Update documentation

## 🤝 Support

For issues or questions:
1. Check the troubleshooting section
2. Review the API documentation
3. Run the test suite to diagnose issues
4. Check server logs for detailed errors

## 📄 License

This security implementation is part of the Bitcoin Sprint project and follows the project's licensing terms.
