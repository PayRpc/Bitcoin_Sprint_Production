/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',
  experimental: {
    // Enable standalone output for Docker
    outputFileTracingRoot: undefined,
  },

  // Environment variables that should be available at build time
  env: {
    NEXT_PUBLIC_APP_ENV: process.env.NEXT_PUBLIC_APP_ENV || 'development',
    NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3002/api',
  },

  images: {
    unoptimized: true, // Required for standalone output
    domains: ['localhost'],
  },

  // Disable telemetry in production
  telemetry: false,

  // API rewrites for development
  async rewrites() {
    return [
      {
        source: '/api/go/:path*',
        destination: `${process.env.GO_API_URL || 'http://localhost:8080'}/:path*`,
      },
    ];
  },

  // Headers for security and CORS
  async headers() {
    return [
      {
        source: '/api/:path*',
        headers: [
          {
            key: 'Access-Control-Allow-Origin',
            value: process.env.CORS_ORIGINS || 'http://localhost:3002',
          },
          {
            key: 'Access-Control-Allow-Methods',
            value: 'GET, POST, PUT, DELETE, OPTIONS',
          },
          {
            key: 'Access-Control-Allow-Headers',
            value: 'Content-Type, Authorization, X-API-Key',
          },
        ],
      },
    ];
  },

  // Webpack configuration for entropy bridge
  webpack: (config, { isServer }) => {
    // Handle FFI modules for entropy bridge
    if (!isServer) {
      config.resolve.fallback = {
        ...config.resolve.fallback,
        fs: false,
        path: false,
        crypto: false,
      };
    }

    // Add support for .node files (Rust FFI)
    config.module.rules.push({
      test: /\.node$/,
      use: 'node-loader',
    });

    return config;
  },
}

module.exports = nextConfig