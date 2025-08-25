import type { NextRequest } from 'next/server';
import { NextResponse } from 'next/server';

export function middleware(request: NextRequest) {
  // Skip maintenance check for health and maintenance endpoints
  const pathname = request.nextUrl.pathname;
  if (pathname.startsWith('/api/health') || 
      pathname.startsWith('/api/maintenance') ||
      pathname.startsWith('/api/update-state') ||
      pathname.startsWith('/_next') ||
      pathname.startsWith('/favicon')) {
    return NextResponse.next();
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
  } catch (error) {
    // If we can't check maintenance mode, allow request to proceed
    console.error('Error checking maintenance mode:', error);
  }

  return NextResponse.next();
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
