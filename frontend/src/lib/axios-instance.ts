import axios, { AxiosError, AxiosHeaders, InternalAxiosRequestConfig } from 'axios';
import { getAccessToken, setAccessToken, clearAccessToken } from './auth';

export const customInstance = axios.create({
  baseURL:
    typeof window !== 'undefined'
      ? (process.env.NEXT_PUBLIC_API_BASE_URL || process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3000/api/v1')
      : (process.env.NEXT_PUBLIC_API_BASE_URL || process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3000/api/v1'),
  withCredentials: true,
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

let isRefreshing = false;
let refreshPromise: Promise<string | null> | null = null;

customInstance.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config;
    if (!originalRequest) return Promise.reject(error);

    // Only handle 401s for non-auth endpoints
    if (error.response?.status === 401 && shouldRetry(originalRequest) && !(originalRequest as InternalAxiosRequestConfig & { _retry?: boolean })._retry) {
      (originalRequest as InternalAxiosRequestConfig & { _retry?: boolean })._retry = true;

      if (!isRefreshing) {
        isRefreshing = true;
        refreshPromise = (async (): Promise<string | null> => {
          try {
            const res = await axios.post<{ accessToken?: string }>(
              `${customInstance.defaults.baseURL}/auth/refresh`,
              {},
              { withCredentials: true }
            );
            const token = res.data?.accessToken ?? null;
            if (token) {
              setAccessToken(token);
            }
            return token;
          } catch {
            return null;
          }
        })();
      }

      const newToken = await refreshPromise;
      isRefreshing = false;
      refreshPromise = null;

      if (newToken) {
        originalRequest.headers = AxiosHeaders.from(originalRequest.headers);
        originalRequest.headers.set('Authorization', `Bearer ${newToken}`);
        return customInstance(originalRequest);
      }

      // Refresh failed: clear auth and dispatch logout event
      clearAccessToken();
      if (typeof window !== 'undefined') {
        window.dispatchEvent(new Event('pat:logout'));
      }
    }

    return Promise.reject(error);
  }
);
