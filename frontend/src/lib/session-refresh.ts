import axios from 'axios';
import { setAccessToken } from './auth';

/**
 * Single shared session-refresh entry point.
 *
 * Guarantees:
 * - Single-flight per tab: concurrent callers share one request.
 * - Cross-tab serialization via Web Locks: two tabs can never rotate the
 *   refresh-token family concurrently (prevents reuse-detection revocation
 *   when Admin + User dashboards are open side by side).
 * - Canonical token persistence via setAccessToken (cookie + memory).
 */

const baseURL =
  process.env.NEXT_PUBLIC_API_BASE_URL ||
  process.env.NEXT_PUBLIC_API_URL ||
  (typeof window !== 'undefined' ? '/api/v1' : 'http://localhost:3000/api/v1');

let inFlight: Promise<string | null> | null = null;

async function performRefresh(): Promise<string | null> {
  try {
    const res = await axios.post<{ accessToken?: string }>(
      `${baseURL}/auth/refresh`,
      {},
      { withCredentials: true }
    );
    const token = res.data?.accessToken ?? null;
    if (token) setAccessToken(token);
    return token;
  } catch {
    return null;
  }
}

type LockManager = {
  request: <T>(name: string, callback: () => Promise<T>) => Promise<T>;
};

export function refreshSession(): Promise<string | null> {
  if (inFlight) return inFlight;

  const run = (): Promise<string | null> => {
    // Serialize across tabs of the same origin when the API is available.
    const locks = typeof navigator !== 'undefined'
      ? (navigator as Navigator & { locks?: LockManager }).locks
      : undefined;
    if (locks && typeof locks.request === 'function') {
      return locks.request<string | null>('pat_refresh_lock', () => performRefresh());
    }
    return performRefresh();
  };

  inFlight = run().finally(() => {
    inFlight = null;
  });
  return inFlight;
}
