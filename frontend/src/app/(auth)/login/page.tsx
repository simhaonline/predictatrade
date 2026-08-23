"use client";
import { useState } from "react";
import Link from "next/link";
import { customInstance } from "@/lib/axios-instance";
import { useRouter } from "next/navigation";
import { getApiErrorMessage } from "@/lib/errors";
import { setAccessToken } from "@/lib/auth";
import { IconLoader2 } from "@tabler/icons-react";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const res = await customInstance.post("/auth/login", { email, password });
      if (res.data?.mfaRequired) {
        router.push(`/verify-otp?challengeId=${res.data.challengeId}&method=${res.data.method}`);
      } else {
        const { accessToken, user } = res.data;
        if (accessToken) {
          // Persist via the canonical setter (memory + pat_access_token cookie) —
          // middleware/proxy and axios read the cookie, not localStorage.
          setAccessToken(accessToken);
          window.dispatchEvent(new Event("pat:auth-changed"));
        }
        // Use window.location for hard navigation to avoid RSC 404 issues
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
    <div className="space-y-4" style={{ maxWidth: "440px", margin: "0 auto", padding: "20px 0" }}>
      {/* Heading */}
      <div style={{ marginBottom: "8px" }}>
        <h1 style={{ fontSize: "28px", fontWeight: 700, color: "#171a22", lineHeight: "1.1", margin: "0 0 8px 0" }}>
          Welcome back
        </h1>
        <p style={{ fontSize: "14px", color: "#6c707a", margin: "0", lineHeight: "1.5" }}>
          Sign in to continue to your dashboard.
        </p>
      </div>

      {/* Auth card */}
      <div style={{
        border: "1px solid #d7d3c9",
        borderRadius: "8px",
        padding: "24px",
        background: "#ffffff",
        boxShadow: "0 4px 20px rgba(0,0,0,0.08)"
      }}>
        {/* Blue top accent */}
        <div style={{ position: "absolute", marginTop: "-25px", marginLeft: "-25px", width: "60px", height: "3px", background: "#205fdc" }} />

        <form onSubmit={handleSubmit} style={{ display: "flex", flexDirection: "column", gap: "16px" }}>
          {error && (
            <div style={{ fontSize: "13px", color: "#b42318", background: "#fbe3e5", border: "1px solid #f3c3c7", borderRadius: "6px", padding: "10px 12px" }}>
              {error}
            </div>
          )}

          {/* Email */}
          <div>
            <label style={{ display: "block", fontSize: "12px", fontWeight: 600, color: "#343842", marginBottom: "6px" }}>Email address</label>
            <input type="email" required autoComplete="email" value={email} onChange={(e) => setEmail(e.target.value)}
              style={{ width: "100%", height: "44px", border: "1px solid #d7d3c9", borderRadius: "6px", background: "#ffffff", color: "#171a22", padding: "0 14px", fontSize: "14px", outline: "none", boxSizing: "border-box" }}
              onFocus={(e) => e.target.style.borderColor = "#205fdc"}
              onBlur={(e) => e.target.style.borderColor = "#d7d3c9"}
              placeholder="you@example.com" />
          </div>

          {/* Password */}
          <div>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "6px" }}>
              <label style={{ fontSize: "12px", fontWeight: 600, color: "#343842" }}>Password</label>
              <Link href="/forgot-password" style={{ fontSize: "12px", color: "#205fdc", textDecoration: "none" }}>Forgot password?</Link>
            </div>
            <div style={{ position: "relative" }}>
              <input type={showPassword ? "text" : "password"} required autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)}
                style={{ width: "100%", height: "44px", border: "1px solid #d7d3c9", borderRadius: "6px", background: "#ffffff", color: "#171a22", padding: "0 40px 0 14px", fontSize: "14px", outline: "none", boxSizing: "border-box" }}
                onFocus={(e) => e.target.style.borderColor = "#205fdc"}
                onBlur={(e) => e.target.style.borderColor = "#d7d3c9"}
                placeholder="Enter your password" />
              <button type="button" onClick={() => setShowPassword(!showPassword)}
                style={{ position: "absolute", right: "8px", top: "50%", transform: "translateY(-50%)", background: "none", border: "none", cursor: "pointer", padding: "4px", color: "#6c707a" }}>
                {showPassword ? "🙈" : "👁"}
              </button>
            </div>
          </div>

          {/* Submit button — explicit colors, no CSS variables */}
          <button type="submit" disabled={loading}
            style={{
              width: "100%",
              height: "44px",
              background: loading ? "#1749ae" : "#205fdc",
              color: "#ffffff",
              border: "none",
              borderRadius: "6px",
              fontSize: "14px",
              fontWeight: 600,
              cursor: loading ? "wait" : "pointer",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              gap: "8px",
              opacity: loading ? 0.7 : 1,
              transition: "background 0.2s"
            }}
            onMouseEnter={(e) => { if (!loading) e.currentTarget.style.background = "#1749ae"; }}
            onMouseLeave={(e) => { if (!loading) e.currentTarget.style.background = "#205fdc"; }}>
            {loading ? (<><IconLoader2 size={16} className="animate-spin" /> Signing in...</>) : "Sign in securely"}
          </button>
        </form>
      </div>

      {/* Switch to register */}
      <p style={{ textAlign: "center", fontSize: "13px", color: "#6c707a", margin: "0" }}>
        New to Predict-A-Trade? <Link href="/register" style={{ color: "#205fdc", fontWeight: 600, textDecoration: "none" }}>Create an account</Link>
      </p>
    </div>
  );
}
