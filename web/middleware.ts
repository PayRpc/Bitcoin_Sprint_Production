import fs from 'fs';
import type { NextRequest } from 'next/server';
import { NextResponse } from 'next/server';
import path from 'path';

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
    // Check if maintenance mode is enabled
    const maintenancePath = path.join(process.cwd(), 'data', 'maintenance.json');
    
    if (fs.existsSync(maintenancePath)) {
      const maintenanceData = fs.readFileSync(maintenancePath, 'utf-8');
      const maintenance = JSON.parse(maintenanceData);
      
      if (maintenance.enabled) {
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
              reason: maintenance.reason || 'System maintenance in progress',
              started_at: maintenance.started_at,
              estimated_duration: maintenance.estimated_duration
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
    }
  } catch (error) {
    // If we can't read maintenance file, allow request to proceed
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
