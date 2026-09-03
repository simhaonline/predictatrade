"use client";
import { useState } from "react";
import { customInstance } from "@/lib/axios-instance";
import { getApiErrorMessage } from "@/lib/errors";
import { setAccessToken, getRoleFromToken, getRoleFromTokenUnchecked } from "@/lib/auth";
import { homeRouteForRole } from "@/lib/roles";

export default function VerifyOtpPage() {
  const [code, setCode] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const challengeId = typeof window !== "undefined"
        ? (window as unknown as Window & { __MFA_CHALLENGE__?: string }).__MFA_CHALLENGE__
          || new URLSearchParams(window.location.search).get("challengeId")
          || ""
        : "";
      const res = await customInstance.post("/auth/verify-otp", { challengeId, code });
      if (res.data?.accessToken) {
        setAccessToken(res.data.accessToken);
        // Role-aware redirect via FULL page load: the auth provider's session
        // bootstrap effect only runs once per mount, and on this page the
        // provider skipped initialization (auth-skip list). A client-side
        // router.push left user/session state unresolved, so admins landing
        // on any /admin route saw "Access Denied" in the sidebar.
        const token = res.data.accessToken as string;
        const role = getRoleFromToken(token) || getRoleFromTokenUnchecked(token);
        window.location.href = homeRouteForRole(role);
      }
    } catch (err: unknown) {
      setError(getApiErrorMessage(err, "Verification failed"));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex flex-col items-center justify-center" style={{ gap: "clamp(0.5rem, 1.5vh, 1rem)" }}>
      <div className="text-center">
        <h1 className="font-bold text-pat-text-primary" style={{ fontSize: "clamp(1.1rem, 2.5vh, 1.5rem)" }}>MFA Verification</h1>
        <p className="text-sm text-pat-text-secondary">Enter your 6-digit authentication code</p>
      </div>
      <form onSubmit={handleSubmit} className="space-y-4">
        {error && <div className="text-sm text-pat-danger">{error}</div>}
        <div>
          <label htmlFor="code" className="block text-sm font-medium mb-1 text-pat-text-primary">Authentication Code</label>
          <input id="code" type="text" inputMode="numeric" pattern="\d{6}" maxLength={6} required value={code}
            onChange={(e) => setCode(e.target.value.replace(/\D/g, ""))}
            placeholder="000000"
            className="w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-center font-mono tracking-widest text-pat-input-text focus:outline-none focus:ring-2 focus:ring-pat-primary" />
        </div>
        <button type="submit" disabled={loading || code.length !== 6}
          className="w-full rounded-md bg-pat-primary px-4 py-2 text-sm font-semibold text-pat-primary-foreground hover:bg-pat-primary-hover disabled:opacity-50">
          {loading ? "Verifying..." : "Verify"}
        </button>
      </form>
    </div>
  );
}
