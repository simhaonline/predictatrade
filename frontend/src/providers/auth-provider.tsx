'use client';

import React, { createContext, useContext, useEffect, useState, useCallback, useSyncExternalStore } from 'react';
import { useRouter } from 'next/navigation';
import { customInstance } from '@/lib/axios-instance';
import { setAccessToken, clearAccessToken, getAccessToken, getRoleFromToken, getRoleFromTokenUnchecked } from '@/lib/auth';
import { refreshSession } from '@/lib/session-refresh';
import { homeRouteForRole, type Role } from '@/lib/roles';

export interface User {
  id: string;
  email: string;
  role: Role;
  name?: string;
  avatar?: string;
  mfaEnabled?: boolean;
  /** Server flag: privileged role without an enabled TOTP — enrollment required. */
  mfaEnrollmentRequired?: boolean;
}

type SessionState = 'LOADING' | 'AUTHENTICATED' | 'UNAUTHENTICATED';

interface AuthContextValue {
  user: User | null;
  loading: boolean;
  sessionState: SessionState;
  login: (email: string, password: string, trustDevice?: boolean) => Promise<void>;
  logout: () => Promise<void>;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

/**
 * Normalize a raw API response into a User, extracting role from the
 * CURRENT (possibly just-refreshed) access token — not a stale captured one.
 */
function normalizeUser(raw: unknown): User | null {
  if (!raw || typeof raw !== 'object') return null;
  const r = raw as Record<string, unknown>;
  if (!r.id || !r.email) return null;

  // Always read the CURRENT token — never a captured one.
  const token = getAccessToken();
  const role = getRoleFromToken(token) || getRoleFromTokenUnchecked(token) || 'USER';

  return {
    id: String(r.id),
    email: String(r.email),
    role,
    name: String(r.full_name || r.displayName || r.email),
    avatar: r.avatar ? String(r.avatar) : undefined,
    mfaEnabled: Boolean(r.mfa_enabled || r.mfaEnabled),
    mfaEnrollmentRequired: Boolean(r.mfa_enrollment_required || r.mfaEnrollmentRequired),
  };
}

/** SSR-safe token subscriber so loading is correct on first paint. */
function useHasToken(): boolean {
  return useSyncExternalStore(
    () => () => {},
    () => !!getAccessToken(),
    () => false,
  );
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const hasToken = useHasToken();
  const [sessionState, setSessionState] = useState<SessionState>(hasToken ? 'LOADING' : 'UNAUTHENTICATED');
  const router = useRouter();

  const fetchMe = useCallback(async (): Promise<User | null> => {
    const token = getAccessToken();
    if (!token) {
      setUser(null);
      setSessionState('UNAUTHENTICATED');
      return null;
    }
    try {
      const res = await customInstance.get('/auth/me');
      // Re-read the token AFTER the request — it may have been refreshed
      const fetchedUser = normalizeUser(res.data);
      if (fetchedUser) {
        setUser(fetchedUser);
        setSessionState('AUTHENTICATED');
        // mfaEnrollmentRequired is advisory now (guard no longer 403-blocks);
        // keep operators on their intended page instead of yanking them to
        // the MFA tab. The settings page still surfaces the MFA tab.
      } else {
        setUser(null);
        setSessionState('UNAUTHENTICATED');
      }
      return fetchedUser;
    } catch {
      setUser(null);
      setSessionState('UNAUTHENTICATED');
      return null;
    }
  }, []);

  useEffect(() => {
    const token = getAccessToken();
    if (!token) {
      // Public/auth pages never need session restoration — skip the refresh
      // attempt there so anonymous visitors don't generate doomed 400s.
      const path = typeof window !== 'undefined' ? window.location.pathname : '';
      if (/^\/(login|register|verify-otp|forgot-password|reset-password|preview|unsubscribe|terms|privacy|complaints|cookies|sitemap|forbidden)(\/|$)/.test(path)) {
        queueMicrotask(() => {
          setSessionState('UNAUTHENTICATED');
        });
        return;
      }
      // No access token in memory/cookie — but the refresh token cookie may still be valid.
      // Try to refresh BEFORE declaring the user unauthenticated. Serialized and
      // single-flight across tabs via the shared helper.
      void (async () => {
        const newToken = await refreshSession();
        if (newToken) {
          await fetchMe();
          return;
        }
        queueMicrotask(() => {
          setSessionState('UNAUTHENTICATED');
        });
      })();
      return;
    }
    void (async () => {
      await fetchMe();
    })();
  }, [fetchMe]);

  // Listen for forced logout from axios interceptor (refresh failure)
  useEffect(() => {
    const handler = () => {
      setUser(null);
      setSessionState('UNAUTHENTICATED');
      window.location.href = '/login';
    };
    window.addEventListener('pat:logout', handler);
    return () => window.removeEventListener('pat:logout', handler);
  }, [router]);

  // Proactive token refresh — refresh the access token every 45 minutes
  // (before the 1h expiry) to prevent any interruption during active use.
  useEffect(() => {
    if (sessionState !== 'AUTHENTICATED') return;
    const interval = setInterval(async () => {
      // Shared, serialized refresh — silent on failure; the axios
      // interceptor will handle 401 on the next request.
      await refreshSession();
    }, 45 * 60 * 1000); // 45 minutes
    return () => clearInterval(interval);
  }, [sessionState, router]);

  const login = async (email: string, password: string, trustDevice = false) => {
    const res = await customInstance.post<{
      accessToken?: string;
      user?: { id: string; email: string; displayName?: string };
      mfaRequired?: boolean;
      challengeId?: string;
      method?: string;
      mfaEnrollmentRequired?: boolean;
    }>('/auth/login', { email, password, trustDevice });
    const data = res.data;

    if (data.mfaRequired) {
      if (typeof window !== 'undefined') {
        (window as unknown as Window & { __MFA_CHALLENGE__?: string }).__MFA_CHALLENGE__ = data.challengeId ?? '';
      }
      router.push('/verify-otp');
      return;
    }

    if (!data.accessToken) {
      throw new Error('Login failed: no access token returned');
    }

    setAccessToken(data.accessToken);

    // AUTH-1 is advisory now (forced-MFA guard removed in b9197b8): admins
    // land on their normal destination (dashboard / requested page). The
    // MFA tab remains available at /admin/settings?tab=mfa for voluntary
    // enrollment, and /auth/me still exposes mfaEnrollmentRequired for a
    // non-blocking banner.

    // Build user from login response + token role
    const token = getAccessToken();
    const role = getRoleFromToken(token) || getRoleFromTokenUnchecked(token) || 'USER';

    let loggedInUser: User | null = null;
    if (data.user) {
      loggedInUser = {
        id: data.user.id,
        email: data.user.email,
        role,
        name: data.user.displayName || data.user.email,
      };
    } else {
      loggedInUser = await fetchMe();
    }

    setUser(loggedInUser);
    setSessionState('AUTHENTICATED');

    // Canonical role-aware redirect — handles SUPER_ADMIN too
    const dest = homeRouteForRole(role);
    window.location.href = dest;
  };

  const logout = async () => {
    try {
      await customInstance.post('/auth/logout');
    } catch {
      /* ignore */
    }
    clearAccessToken();
    setUser(null);
    setSessionState('UNAUTHENTICATED');
    window.location.href = '/login';
  };

  const refreshUser = async () => {
    await fetchMe();
  };

  const loading = sessionState === 'LOADING';

  return (
    <AuthContext.Provider value={{ user, loading, sessionState, login, logout, refreshUser }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be inside AuthProvider');
  return ctx;
}
