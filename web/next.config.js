/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',
  experimental: {
    // Enable standalone output for Docker
  },
  env: {
    // Environment variables that should be available at build time
  },
  images: {
    unoptimized: true, // Required for standalone output
  },
  // Disable telemetry in production
  telemetry: false,
}

module.exports = nextConfig
