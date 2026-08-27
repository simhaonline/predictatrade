import axios, { AxiosError, AxiosHeaders, InternalAxiosRequestConfig } from 'axios';
import { getAccessToken, clearAccessToken } from './auth';
import { refreshSession, getLastRefreshErrorStatus } from './session-refresh';

export const customInstance = axios.create({
  baseURL:
    process.env.NEXT_PUBLIC_API_URL ||
    process.env.NEXT_PUBLIC_API_BASE_URL ||
    '/api/v1',
  withCredentials: true,
  timeout: 15000, // 15s timeout — prevents permanent loading spinners
  headers: {
    'Content-Type': 'application/json',
  },
});

customInstance.interceptors.request.use((config) => {
  const token = getAccessToken();
  if (token) {
    config.headers = AxiosHeaders.from(config.headers);
    config.headers.set('Authorization', `Bearer ${token}`);
  }
  return config;
});

/** Paths that should NEVER trigger a token refresh (avoids loops) */
const NO_REFRESH_PATHS = ['/auth/me', '/auth/refresh', '/auth/login', '/auth/register', '/auth/logout'];

function shouldRetry(config: InternalAxiosRequestConfig): boolean {
  if (!config?.url) return true;
  return !NO_REFRESH_PATHS.some((p) => config.url?.includes(p));
}

customInstance.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config;
    if (!originalRequest) return Promise.reject(error);

    // Only handle 401s for non-auth endpoints
    if (error.response?.status === 401 && shouldRetry(originalRequest) && !(originalRequest as InternalAxiosRequestConfig & { _retry?: boolean })._retry) {
      (originalRequest as InternalAxiosRequestConfig & { _retry?: boolean })._retry = true;

      // Serialized, single-flight refresh shared with AuthProvider (prevents
      // concurrent rotations across tabs).
      const newToken = await refreshSession();

      if (newToken) {
        originalRequest.headers = AxiosHeaders.from(originalRequest.headers);
        originalRequest.headers.set('Authorization', `Bearer ${newToken}`);
        return customInstance(originalRequest);
      }

      // Refresh failed. Distinguish a transient rate-limit (429) from a real
      // auth failure. On 429 we must NOT clear the token / force a logout —
      // doing so during a rate-limit storm bounced users out of their session
      // in a login<->logout loop. The existing token may still be valid; we
      // simply let the original request fail and retry later.
      const refreshStatus = getLastRefreshErrorStatus();
      if (refreshStatus === 429) {
        return Promise.reject(error);
      }

      // Refresh failed for real (e.g. invalid/expired refresh token): clear
      // auth and dispatch logout event.
      clearAccessToken();
      if (typeof window !== 'undefined') {
        window.dispatchEvent(new Event('pat:logout'));
      }
    }

    return Promise.reject(error);
  }
);
