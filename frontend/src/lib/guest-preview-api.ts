/**
 * Guest Preview API client + React hook.
 *
 * The guest preview timer is enforced SERVER-SIDE. The client only:
 *   1. asks the server to issue a short-lived anonymous session (POST /guest/session),
 *   2. periodically polls the server-authoritative status (GET /guest/status),
 *   3. shows a display-only countdown from the server's remainingSeconds.
 *
 * The browser cookie/localStorage countdown is NEVER the source of truth —
 * clearing cookies / incognito still locks at PREVIEW_SECONDS because the
 * server validates the signed guest JWT's `exp` claim on every status check.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { customInstance } from "@/lib/axios-instance";
import { setAccessToken } from "@/lib/auth";

export interface GuestSession {
  guestToken: string;
  expiresAt: number;
  previewSeconds: number;
}

export interface GuestStatus {
  locked: boolean;
  expiresAt: number | null;
  remainingSeconds: number;
}

export interface GuestRegisterPayload {
  fullName: string;
  email: string;
  phone?: string;
  broker?: string;
  termsAccepted: boolean;
  riskAcknowledged: boolean;
  marketingOptIn: boolean;
}

export interface GuestRegisterResponse {
  message: string;
  challengeId: string;
}

export interface GuestVerifyResponse {
  accessToken: string;
  user: { id: string; email: string; displayName: string };
}

/** Issue a new server-side guest session (sets an HttpOnly cookie). */
export async function issueGuestSession(): Promise<GuestSession> {
  const res = await customInstance.post<GuestSession>("/guest/session");
  return res.data;
}

/** Fetch the server-authoritative guest status. */
export async function getGuestStatus(): Promise<GuestStatus> {
  const res = await customInstance.get<GuestStatus>("/guest/status");
  return res.data;
}

/** Submit the registration form (sends OTP email). Generic response. */
export async function registerGuest(payload: GuestRegisterPayload): Promise<GuestRegisterResponse> {
  const res = await customInstance.post<GuestRegisterResponse>("/guest/register", payload);
  return res.data;
}

/** Resend the OTP (60s cooldown enforced server-side). */
export async function resendGuestOtp(email: string): Promise<{ message: string }> {
  const res = await customInstance.post<{ message: string }>("/guest/otp/resend", { email });
  return res.data;
}

/** Verify the OTP. On success, sets the access token (refresh is an HttpOnly cookie). */
export async function verifyGuestOtp(email: string, code: string): Promise<GuestVerifyResponse> {
  const res = await customInstance.post<GuestVerifyResponse>("/guest/otp/verify", { email, code });
  if (res.data?.accessToken) {
    setAccessToken(res.data.accessToken);
  }
  return res.data;
}

/**
 * useGuestPreview — manages the server-authoritative guest preview lifecycle.
 *
 * Returns the display-only countdown, locked state, and actions to register /
 * verify OTP. Polls /guest/status every `pollMs` (default 5s) so the lock fires
 * immediately when the server-side expiry passes.
 */
export function useGuestPreview(pollMs = 5000) {
  const [status, setStatus] = useState<GuestStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const refreshStatus = useCallback(async () => {
    try {
      const s = await getGuestStatus();
      setStatus(s);
      setError(null);
    } catch {
      // Network error — keep last known status, don't crash the dashboard.
    } finally {
      setLoading(false);
    }
  }, []);

  // Issue a session on mount (if none exists), then poll.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        // Always (re)issue a session if the server says we're locked and have no
        // token. The server is the authority; issuing just bootstraps the cookie.
        const existing = await getGuestStatus();
        if (existing.expiresAt === null && !cancelled) {
          await issueGuestSession();
        }
      } catch {
        // ignore — refreshStatus will handle
      }
      if (!cancelled) {
        await refreshStatus();
        timerRef.current = setInterval(refreshStatus, pollMs);
      }
    })();
    return () => {
      cancelled = true;
      if (timerRef.current) clearInterval(timerRef.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const register = useCallback(async (payload: GuestRegisterPayload) => {
    setError(null);
    return registerGuest(payload);
  }, []);

  const resend = useCallback(async (email: string) => {
    setError(null);
    return resendGuestOtp(email);
  }, []);

  const verify = useCallback(async (email: string, code: string) => {
    setError(null);
    return verifyGuestOtp(email, code);
  }, []);

  return {
    status,
    loading,
    error,
    setError,
    refreshStatus,
    register,
    resend,
    verify,
  };
}
