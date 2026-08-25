"use client";
import { useState } from "react";
import Link from "next/link";
import { customInstance } from "@/lib/axios-instance";
import { useRouter } from "next/navigation";
import { getApiErrorMessage } from "@/lib/errors";
import { setAccessToken } from "@/lib/auth";
import { IconLoader2, IconEye, IconEyeOff, IconShieldCheck, IconMail, IconLock, IconCheck } from "@tabler/icons-react";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [rememberMe, setRememberMe] = useState(true);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [justRegistered] = useState(() => {
    if (typeof window === 'undefined') return false;
    return new URLSearchParams(window.location.search).get("registered") === "1";
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const res = await customInstance.post("/auth/login", {
        email, password,
        ...(rememberMe ? { trustDevice: true } : {}),
      });
      if (res.data?.mfaRequired) {
        router.push(`/verify-otp?challengeId=${res.data.challengeId}&method=${res.data.method}`);
      } else {
        const { accessToken, user } = res.data;
        if (accessToken) {
          setAccessToken(accessToken);
          window.dispatchEvent(new Event("pat:auth-changed"));
        }
        if (user?.role === "ADMIN") window.location.href = "/admin/dashboard";
        else window.location.href = "/dashboard/live";
      }
    } catch (err: unknown) {
      setError(getApiErrorMessage(err, "Invalid credentials"));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: "16px" }}>
      {/* Heading */}
      <div style={{ marginBottom: "4px" }}>
        <h1 style={{ fontSize: "28px", fontWeight: 700, color: "#171a22", lineHeight: "1.1", margin: "0 0 8px 0" }}>
          Welcome back
        </h1>
        <p style={{ fontSize: "14px", color: "#6c707a", margin: "0", lineHeight: "1.5" }}>
          Sign in to continue to your dashboard.
        </p>
      </div>

      {/* Registration success banner */}
      {justRegistered && (
        <div style={{ fontSize: "13px", color: "#15803d", background: "#dcfce7", border: "1px solid #bbf7d0", borderRadius: "8px", padding: "10px 12px", lineHeight: 1.4, display: "flex", alignItems: "center", gap: "8px" }}>
          <IconShieldCheck size={18} />
          <span>Account created successfully. Please sign in with your credentials.</span>
        </div>
      )}

      {/* Auth card */}
      <div style={{
        border: "1px solid #d7d3c9",
        borderRadius: "10px",
        padding: "24px",
        background: "#ffffff",
        boxShadow: "0 4px 24px rgba(0,0,0,0.06)",
      }}>
        <form onSubmit={handleSubmit} style={{ display: "flex", flexDirection: "column", gap: "16px" }}>
          {error && (
            <div style={{ fontSize: "13px", color: "#b42318", background: "#fbe3e5", border: "1px solid #f3c3c7", borderRadius: "8px", padding: "10px 12px", lineHeight: 1.4 }}>
              {error}
            </div>
          )}

          {/* Email */}
          <div>
            <label style={{ display: "block", fontSize: "12px", fontWeight: 600, color: "#343842", marginBottom: "6px" }}>Email address</label>
            <div style={{ position: "relative" }}>
              <IconMail size={16} style={{ position: "absolute", left: "12px", top: "50%", transform: "translateY(-50%)", color: "#9ba3b4", pointerEvents: "none" }} />
              <input type="email" required autoComplete="email" value={email} onChange={(e) => setEmail(e.target.value)}
                style={{ ...inputStyle, paddingLeft: "36px" }}
                onFocus={(e) => e.target.style.borderColor = "#205fdc"}
                onBlur={(e) => e.target.style.borderColor = "#d7d3c9"}
                placeholder="you@example.com" />
            </div>
          </div>

          {/* Password */}
          <div>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "6px" }}>
              <label style={{ fontSize: "12px", fontWeight: 600, color: "#343842" }}>Password</label>
              <Link href="/forgot-password" style={{ fontSize: "12px", color: "#205fdc", textDecoration: "none" }}>Forgot password?</Link>
            </div>
            <div style={{ position: "relative" }}>
              <IconLock size={16} style={{ position: "absolute", left: "12px", top: "50%", transform: "translateY(-50%)", color: "#9ba3b4", pointerEvents: "none" }} />
              <input type={showPassword ? "text" : "password"} required autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)}
                style={{ ...inputStyle, paddingLeft: "36px", paddingRight: "40px" }}
                onFocus={(e) => e.target.style.borderColor = "#205fdc"}
                onBlur={(e) => e.target.style.borderColor = "#d7d3c9"}
                placeholder="Enter your password" />
              <button type="button" onClick={() => setShowPassword(!showPassword)}
                style={{ position: "absolute", right: "8px", top: "50%", transform: "translateY(-50%)", background: "none", border: "none", cursor: "pointer", padding: "4px", color: "#6c707a", display: "flex", alignItems: "center" }}>
                {showPassword ? <IconEyeOff size={18} /> : <IconEye size={18} />}
              </button>
            </div>
          </div>

          {/* Remember me */}
          <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
            <label style={{ display: "flex", alignItems: "center", gap: "8px", cursor: "pointer", fontSize: "13px", color: "#343842" }}>
              <div
                onClick={() => setRememberMe(!rememberMe)}
                style={{
                  width: "18px", height: "18px", minWidth: "18px", borderRadius: "4px",
                  border: rememberMe ? "2px solid #205fdc" : "2px solid #c5c9d0",
                  background: rememberMe ? "#205fdc" : "#ffffff",
                  display: "flex", alignItems: "center", justifyContent: "center", transition: "all 0.15s",
                }}
              >
                {rememberMe && <IconCheck size={13} color="#ffffff" strokeWidth={3} />}
              </div>
              <span>Remember me on this device</span>
            </label>
          </div>

          {/* Submit button */}
          <button type="submit" disabled={loading || !email || !password}
            style={{
              width: "100%",
              height: "44px",
              background: loading || !email || !password ? "#c5c9d0" : loading ? "#1749ae" : "#205fdc",
              color: "#ffffff",
              border: "none",
              borderRadius: "8px",
              fontSize: "14px",
              fontWeight: 600,
              cursor: loading || !email || !password ? "not-allowed" : loading ? "wait" : "pointer",
              display: "flex", alignItems: "center", justifyContent: "center", gap: "8px",
              opacity: loading ? 0.8 : 1,
              transition: "background 0.2s, opacity 0.2s",
            }}
            onMouseEnter={(e) => { if (!loading && email && password) e.currentTarget.style.background = "#1749ae"; }}
            onMouseLeave={(e) => { if (!loading && email && password) e.currentTarget.style.background = "#205fdc"; }}>
            {loading ? (<><IconLoader2 size={16} className="animate-spin" /> Signing in...</>) : (<><IconShieldCheck size={18} /> Sign in securely</>)}
          </button>
        </form>
      </div>

      {/* Switch to register */}
      <p style={{ textAlign: "center", fontSize: "13px", color: "#6c707a", margin: "0" }}>
        New to Predict-A-Trade? <Link href="/register" style={{ color: "#205fdc", fontWeight: 600, textDecoration: "none" }}>Create an account</Link>
      </p>

      {/* Security note */}
      <div style={{ display: "flex", alignItems: "center", justifyContent: "center", gap: "6px", fontSize: "11px", color: "#9ba3b4", marginTop: "-4px" }}>
        <IconLock size={12} />
        <span>Protected by 256-bit TLS encryption · MFA enabled</span>
      </div>
    </div>
  );
}


const inputStyle: React.CSSProperties = {
  width: "100%",
  height: "44px",
  border: "1px solid #d7d3c9",
  borderRadius: "8px",
  background: "#ffffff",
  color: "#171a22",
  padding: "0 14px",
  fontSize: "14px",
  outline: "none",
  boxSizing: "border-box",
  transition: "border-color 0.15s",
};
