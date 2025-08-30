# Bitcoin Sprint Web Application

A modern Next.js web application for the Bitcoin Sprint multi-chain blockchain API platform, featuring enterprise-grade security with Rust entropy integration.

## 🏗️ Architecture Overview

```
Bitcoin Sprint Web App
├── Next.js 14 (Frontend + API Routes)
├── TypeScript (Type Safety)
├── Prisma (Database ORM)
├── Rust Entropy Bridge (Security)
├── Tailwind CSS (Styling)
├── Redis (Caching)
└── Authentication System
```

## 🔐 Security Integration

### Rust Entropy Bridge

The application integrates with a **Rust-based entropy bridge** for enterprise-grade security:

- **Admin Secret Generation**: Uses hardware entropy for secure key generation
- **Fallback Support**: Node.js crypto fallback when Rust is unavailable
- **FFI Integration**: Direct C bindings for maximum performance
- **Multi-Platform**: Windows, Linux, and macOS support

### Authentication System

- **Tier-Based Access**: FREE, PRO, ENTERPRISE, ENTERPRISE_PLUS
- **API Key Management**: Secure key validation and rate limiting
- **JWT Integration**: JSON Web Token support
- **Database Persistence**: Prisma-powered user management

## 🚀 Quick Start

### 1. Development Setup

```bash
# Install dependencies
npm install

# Run development setup (database, tests, etc.)
npm run setup:dev

# Start development server
npm run dev
```

### 2. Environment Configuration

The application uses the following environment variables:

```bash
# Database
DATABASE_URL="file:./dev.db"

# Backend Connection
GO_API_URL=http://localhost:8080
API_KEY=bitcoin-sprint-dev-key-2025

# Security
ADMIN_SECRET=bitcoin-sprint-admin-secret-2025

# Redis (optional)
REDIS_URL=redis://localhost:6379
```

### 3. Database Setup

```bash
# Generate Prisma client
npm run prisma:generate

# Run migrations
npm run prisma:migrate

# Open database GUI
npm run prisma:studio
```

## 🧪 Testing

### Test Scripts

```bash
# Test entropy bridge integration
npm run test:entropy

# Test web server endpoints
npm run test:server

# Test backend connection
npm run test:connection

# Run all tests
npm run test:all
```

### Manual Testing

```bash
# Health check
curl http://localhost:3002/api/health

# API status (requires API key)
curl -H "Authorization: Bearer your-api-key" \
     http://localhost:3002/api/status

# Metrics endpoint
curl -H "Authorization: Bearer your-api-key" \
     http://localhost:3002/api/metrics
```

## 📡 API Endpoints

### Public Endpoints

- `GET /api/health` - Service health check
- `GET /` - Landing page
- `GET /signup` - User registration
- `GET /dashboard` - Main dashboard

### Authenticated Endpoints

- `GET /api/status` - API key status and limits
- `GET /api/metrics` - Performance metrics
- `GET /api/latest` - Latest blockchain data
- `GET /api/predictive` - Predictive analytics
- `GET /api/stream` - WebSocket streaming
- `GET /api/v1/license/info` - License information
- `GET /api/v1/analytics/summary` - Analytics summary

### Admin Endpoints

- `GET /api/admin-metrics` - Admin-level metrics
- `POST /api/admin/*` - Administrative functions

## 🔧 Development Commands

```bash
# Development
npm run dev              # Start dev server
npm run build           # Build for production
npm run start           # Start production server
npm run lint            # Run ESLint
npm run typecheck       # TypeScript type checking

# Database
npm run prisma:generate # Generate Prisma client
npm run prisma:migrate  # Run migrations
npm run prisma:studio   # Open database GUI

# Maintenance
npm run maintenance:status    # Check maintenance mode
npm run maintenance:enable    # Enable maintenance mode
npm run maintenance:disable   # Disable maintenance mode

# Security
npm run security:audit  # Run security audit
npm run security:fix    # Fix security issues
```

## 🗂️ Project Structure

```
web/
├── pages/                 # Next.js pages and API routes
│   ├── api/              # API endpoints
│   ├── _app.tsx          # App component
│   ├── index.tsx         # Landing page
│   ├── dashboard.tsx     # Dashboard page
│   └── signup.tsx        # Signup page
├── components/           # React components
│   ├── ui/              # UI components
│   └── ConfigSnippet.tsx # Configuration helper
├── lib/                  # Utility libraries
│   ├── auth.ts          # Authentication middleware
│   ├── goApiClient.ts   # Backend API client
│   └── rust-entropy-bridge.js # Entropy bridge
├── prisma/               # Database schema and migrations
├── scripts/              # Utility scripts
├── styles/               # CSS styles
├── public/               # Static assets
├── test-*.js            # Test scripts
└── setup-dev.js         # Development setup script
```

## 🔗 Integration Points

### Backend Connection

The web app connects to the Go backend through:

1. **Direct API Calls**: Using `goApiClient.ts`
2. **Environment Variables**: `GO_API_URL` configuration
3. **API Key Authentication**: Bearer token authentication
4. **Health Monitoring**: Automatic backend health checks

### Entropy Bridge Integration

```javascript
import { generateAdminSecret } from './lib/rust-entropy-bridge';

// Generate secure admin secret
const secret = await generateAdminSecret('hex');
```

### Database Integration

```javascript
import { PrismaClient } from '@prisma/client';

const prisma = new PrismaClient();

// Query API keys
const apiKey = await prisma.apiKey.findUnique({
  where: { key: 'user-api-key' }
});
```

## 🚢 Deployment

### Local Development

```bash
# Start development server
npm run dev

# Server will be available at http://localhost:3002
```

### Production Build

```bash
# Build for production
npm run build

# Start production server
npm run start
```

### Docker Deployment

```bash
# Build Docker image
docker build -t bitcoin-sprint-web .

# Run container
docker run -p 3002:3002 bitcoin-sprint-web
```

## 🔍 Troubleshooting

### Common Issues

1. **Entropy Bridge Not Working**
   ```bash
   # Check if Rust libraries are built
   npm run test:entropy

   # Install FFI dependencies
   npm install ffi-napi ref-napi
   ```

2. **Database Connection Issues**
   ```bash
   # Check database file
   ls -la prisma/dev.db

   # Reset database
   rm prisma/dev.db
   npm run prisma:migrate
   ```

3. **API Connection Issues**
   ```bash
   # Test backend connection
   npm run test:connection

   # Check environment variables
   cat .env.local
   ```

### Debug Mode

```bash
# Enable debug logging
DEBUG=* npm run dev

# Check server logs
tail -f logs/server.log
```

## 📊 Monitoring

### Health Checks

- **Application Health**: `/api/health`
- **Backend Health**: Automatic monitoring
- **Database Health**: Prisma connection checks

### Metrics

- **API Usage**: Request counts and rates
- **Performance**: Response times and latency
- **Security**: Failed authentication attempts

## 🤝 Contributing

1. **Setup Development Environment**
   ```bash
   npm run setup:dev
   ```

2. **Run Tests**
   ```bash
   npm run test:all
   ```

3. **Code Style**
   ```bash
   npm run lint
   npm run typecheck
   ```

4. **Database Changes**
   ```bash
   # Update schema
   # Run migrations
   npm run prisma:migrate
   ```

## 📝 License

This project is part of the Bitcoin Sprint enterprise platform.

---

**Note**: This web application is designed for enterprise use with multi-chain blockchain support and enterprise-grade security features.
