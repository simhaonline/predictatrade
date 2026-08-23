import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

const AUTH_ROUTES = ['/login', '/register', '/forgot-password', '/reset-password'];
const PUBLIC_ROUTES = ['/terms', '/privacy', '/complaints', '/sitemap', '/cookies', '/forbidden', '/preview', '/unsubscribe'];

function getRoleFromToken(token: string | undefined): string | null {
  if (!token) return null;
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return null;
    const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/');
    const payload = JSON.parse(Buffer.from(base64, 'base64').toString('utf-8'));
    if (payload.exp && Date.now() >= payload.exp * 1000) return null;
    return payload.role || 'USER';
  } catch { return null; }
}

function isAdminRole(role: string | null): boolean {
  return role === 'ADMIN' || role === 'SUPER_ADMIN';
}

function homeRouteForRole(role: string | null): string {
  return isAdminRole(role) ? '/admin/dashboard' : '/dashboard/live';
}

export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const token = request.cookies.get('pat_access_token')?.value;
  const role = getRoleFromToken(token);
  const isAuthenticated = role !== null;
  const isAdmin = isAdminRole(role);

  // Root path — redirect to login (not preview!)
  // Preview is ONLY for live.predictatrade.com, not platform.predictatrade.com
  if (pathname === '/') {
    if (isAuthenticated) return NextResponse.redirect(new URL(homeRouteForRole(role), request.url));
    return NextResponse.redirect(new URL('/login', request.url));
  }

  // Public routes — no auth required (preview is accessible directly but not as default)
  if (PUBLIC_ROUTES.some(r => pathname === r)) {
    return NextResponse.next();
  }

  const isAuthRoute = AUTH_ROUTES.some(route => pathname === route);
  const onAdminRoute = pathname === '/admin' || pathname.startsWith('/admin/');
  const onUserRoute = pathname === '/dashboard' || pathname.startsWith('/dashboard/');

  if ((onAdminRoute || onUserRoute) && !isAuthenticated) {
    const loginUrl = new URL('/login', request.url);
    const safeRedirect = pathname.startsWith('/') && !pathname.startsWith('//') ? pathname : '/dashboard/live';
    loginUrl.searchParams.set('redirect', safeRedirect);
    return NextResponse.redirect(loginUrl);
  }

  if (onUserRoute && isAdmin) return NextResponse.redirect(new URL('/admin/dashboard', request.url));
  if (onAdminRoute && isAuthenticated && !isAdmin) return NextResponse.redirect(new URL('/dashboard/live', request.url));

  if (pathname === '/admin') {
    if (isAdmin) return NextResponse.redirect(new URL('/admin/dashboard', request.url));
    return NextResponse.redirect(new URL('/dashboard/live', request.url));
  }
  if (pathname === '/dashboard') {
    if (isAdmin) return NextResponse.redirect(new URL('/admin/dashboard', request.url));
    return NextResponse.redirect(new URL('/dashboard/live', request.url));
  }

  if (isAuthRoute && isAuthenticated) {
    return NextResponse.redirect(new URL(homeRouteForRole(role), request.url));
  }

  // Authenticated users never need the guest preview — skip straight to dashboard.
  if (pathname === '/preview' && isAuthenticated) {
    return NextResponse.redirect(new URL(homeRouteForRole(role), request.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    '/', '/dashboard/:path*', '/dashboard', '/admin/:path*', '/admin',
    '/login', '/register', '/forgot-password', '/reset-password',
    '/terms', '/privacy', '/complaints', '/sitemap', '/cookies', '/preview', '/unsubscribe',
  ],
};
