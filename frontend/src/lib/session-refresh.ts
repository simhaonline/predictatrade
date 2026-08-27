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

/**
 * Backoff guard: after a failed/throttled refresh, do not hammer
 * /auth/refresh. A rapid refresh storm (e.g. many concurrent 401s during a
 * rate-limit window) previously exhausted the refresh throttle and produced an
 * infinite 429 loop that bounced users out of their session. We cap refresh
 * attempts to at most one per REFRESH_BACKOFF_MS while in a failed state.
 */
const REFRESH_BACKOFF_MS = 10_000;
let lastRefreshAttempt = 0;
let lastRefreshErrorStatus: number | null = null;

export function getLastRefreshErrorStatus(): number | null {
  return lastRefreshErrorStatus;
}

async function performRefresh(): Promise<string | null> {
  lastRefreshErrorStatus = null;
  try {
    const res = await axios.post<{ accessToken?: string }>(
      `${baseURL}/auth/refresh`,
      {},
      { withCredentials: true }
    );
    const token = res.data?.accessToken ?? null;
    if (token) setAccessToken(token);
    return token;
  } catch (e: unknown) {
    const status = (e as { response?: { status?: number } })?.response?.status ?? null;
    lastRefreshErrorStatus = status;
    return null;
  }
}

type LockManager = {
  request: <T>(name: string, callback: () => Promise<T>) => Promise<T>;
};

export function refreshSession(): Promise<string | null> {
  if (inFlight) return inFlight;

  // Backoff: if a recent attempt failed, don't immediately re-hammer the endpoint.
  const now = Date.now();
  if (lastRefreshErrorStatus !== null && now - lastRefreshAttempt < REFRESH_BACKOFF_MS) {
    return Promise.resolve(null);
  }

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
    lastRefreshAttempt = Date.now();
    inFlight = null;
  });
  return inFlight;
}
