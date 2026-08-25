export interface User {
  id: string;
  email: string;
  role: string;
  name?: string;
  avatar?: string;
  mfaEnabled?: boolean;
}

interface CustomWindow extends Window {
  __ACCESS_TOKEN__?: string;
}

const ACCESS_TOKEN_COOKIE = 'pat_access_token';

/* ─── JWT payload decoder (works in browser + Node) ─── */
function base64Decode(base64: string): string {
  // Browser: atob
  if (typeof globalThis !== 'undefined' && 'atob' in globalThis) {
    return atob(base64);
  }
  // Node fallback
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  return require('buffer').Buffer.from(base64, 'base64').toString('utf-8');
}

function decodeJwtPayload(token: string): Record<string, unknown> | null {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return null;
    const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/');
    const json = base64Decode(base64);
    return JSON.parse(json);
  } catch {
    return null;
  }
}

/**
 * Extract role from JWT token, checking expiry.
 * Returns null if token is invalid or expired.
 * Use this for API authorization decisions.
 */
export function getRoleFromToken(token: string | null): string | null {
  if (!token) return null;
  const payload = decodeJwtPayload(token);
  if (!payload) return null;
  if (payload.exp && Date.now() >= (payload.exp as number) * 1000) return null;
  return (payload.role as string) || 'USER';
}

/**
 * Extract role from JWT token WITHOUT checking expiry.
 * The role claim is baked into the JWT at signing time and does not change.
 * This is safe for UI display purposes (sidebar, routing) because:
 * 1. The backend already validated the token during the API call
 * 2. The role doesn't change when the token expires — only the token's
 *    usability for new API calls changes
 * 3. The axios interceptor will refresh the token for the next API call
 *
 * Use this as a fallback when getRoleFromToken returns null due to expiry
 * but the user session is still valid (refresh token cookie exists).
 */
export function getRoleFromTokenUnchecked(token: string | null): string | null {
  if (!token) return null;
  const payload = decodeJwtPayload(token);
  if (!payload) return null;
  return (payload.role as string) || 'USER';
}

/* ─── Cookie helpers ─── */
function getCookie(name: string): string | null {
  if (typeof document === 'undefined') return null;
  const match = document.cookie.match(new RegExp('(?:^|;)\\s*' + name + '=([^;]+)'));
  return match ? decodeURIComponent(match[1]) : null;
}

function setCookie(name: string, value: string, maxAgeSeconds: number): void {
  if (typeof document === 'undefined') return;
  const secure = window.location.protocol === 'https:' ? '; Secure' : '';
  // Shared across subdomains so live.predictatrade.com can authenticate
  // visitors who registered/logged in on the platform (live preview funnel).
  const domain = window.location.hostname.endsWith('.predictatrade.com') ? '; Domain=.predictatrade.com' : '';
  document.cookie = `${name}=${encodeURIComponent(value)}; path=/; max-age=${maxAgeSeconds}; SameSite=Lax${secure}${domain}`;
}

function clearCookie(name: string): void {
  if (typeof document === 'undefined') return;
  document.cookie = `${name}=; path=/; max-age=0; SameSite=Lax`;
}

/* ─── Token management ─── */
export function getAccessToken(): string | null {
  if (typeof window !== 'undefined') {
    const mem = (window as unknown as CustomWindow).__ACCESS_TOKEN__;
    if (mem) return mem;
  }
  return getCookie(ACCESS_TOKEN_COOKIE);
}

export function setAccessToken(token: string): void {
  if (typeof window !== 'undefined') {
    (window as unknown as CustomWindow).__ACCESS_TOKEN__ = token;
  }
  setCookie(ACCESS_TOKEN_COOKIE, token, 3600);
}

export function clearAccessToken(): void {
  if (typeof window !== 'undefined') {
    (window as unknown as CustomWindow).__ACCESS_TOKEN__ = '';
  }
  clearCookie(ACCESS_TOKEN_COOKIE);
}

/**
 * Check if a user has an admin-level role.
 * Handles both ADMIN and SUPER_ADMIN as the backend AdminGuard does.
 * @deprecated Use isAdminRole from '@/lib/roles' for new code.
 */
export function isAdmin(user: User | null): boolean {
  return user?.role === 'ADMIN' || user?.role === 'SUPER_ADMIN';
}
