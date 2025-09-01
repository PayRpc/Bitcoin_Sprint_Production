import type { NextRequest } from 'next/server';
import { NextResponse } from 'next/server';
import { SECURITY_HEADERS } from './lib/security';

export function middleware(request: NextRequest) {
  const response = NextResponse.next();

  // Skip security headers for API routes (they handle their own headers)
  const pathname = request.nextUrl.pathname;
  if (pathname.startsWith('/api/health') ||
    pathname.startsWith('/api/maintenance') ||
    pathname.startsWith('/api/update-state') ||
    pathname.startsWith('/_next') ||
    pathname.startsWith('/favicon')) {
    return response;
  }

  try {
    // Check for maintenance mode via environment variable
    // (Edge runtime compatible - no filesystem operations)
    if (process.env.MAINTENANCE_MODE === 'true') {
      // Return maintenance page for web requests
      if (!pathname.startsWith('/api/')) {
        return NextResponse.redirect(new URL('/maintenance', request.url));
      }

      // Return 503 for API requests
      return new NextResponse(
        JSON.stringify({
          ok: false,
          error: 'Service temporarily unavailable',
          maintenance: {
            enabled: true,
            reason: process.env.MAINTENANCE_REASON || 'System maintenance in progress',
            started_at: process.env.MAINTENANCE_STARTED_AT,
            estimated_duration: process.env.MAINTENANCE_DURATION
          }
        }),
        {
          status: 503,
          headers: {
            'Content-Type': 'application/json',
            'Retry-After': '1800' // 30 minutes
          }
        }
      );
    }

    // Add security headers to all responses
    if (pathname.startsWith('/api/')) {
      // For API routes, add basic security headers
      response.headers.set('X-Content-Type-Options', 'nosniff');
      response.headers.set('X-Frame-Options', 'DENY');
      response.headers.set('X-Request-ID', `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`);
    } else {
      // For web routes, add comprehensive security headers
      Object.entries(SECURITY_HEADERS).forEach(([key, value]) => {
        response.headers.set(key, value);
      });
    }

    // Add HSTS header for HTTPS (only in production)
    if (process.env.NODE_ENV === 'production' && process.env.NEXT_PUBLIC_ENABLE_HTTPS === 'true') {
      response.headers.set('Strict-Transport-Security', 'max-age=31536000; includeSubDomains');
    }

  } catch (error) {
    // If we can't check maintenance mode or add headers, allow request to proceed
    console.error('Error in middleware:', error);
  }

  return response;
}

export const config = {
  matcher: [
    /*
     * Match all request paths except for the ones starting with:
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     */
    '/((?!_next/static|_next/image|favicon.ico).*)',
  ],
};
