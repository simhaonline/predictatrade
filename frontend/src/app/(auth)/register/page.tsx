"use client";
import { useState, useEffect, Suspense } from "react";
import Link from "next/link";
import { customInstance } from "@/lib/axios-instance";
import { useRouter, useSearchParams } from "next/navigation";
import { getApiErrorMessage } from "@/lib/errors";
import { IconLoader2 } from "@tabler/icons-react";

function RegisterForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [referralCode, setReferralCode] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const ref = searchParams.get("ref");
    if (ref) setReferralCode(ref);
  }, [searchParams]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    if (password !== confirmPassword) { setError("Passwords do not match"); return; }
    if (password.length < 8) { setError("Password must be at least 8 characters"); return; }
    setLoading(true);
    try {
      await customInstance.post("/auth/register", {
        email, password,
        ...(referralCode.trim() ? { referralCode: referralCode.trim() } : {}),
      });
      window.location.href = "/login";
    } catch (err: unknown) {
      setError(getApiErrorMessage(err, "Registration failed"));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: "16px" }}>
      {/* Heading */}
      <div>
        <h1 style={{ fontSize: "28px", fontWeight: 700, color: "#171a22", lineHeight: "1.1", margin: "0 0 8px 0" }}>
          Create account
        </h1>
        <p style={{ fontSize: "14px", color: "#6c707a", margin: "0", lineHeight: "1.5" }}>
          Join Predict-A-Trade — it only takes a moment.
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
            <label style={{ display: "block", fontSize: "12px", fontWeight: 600, color: "#343842", marginBottom: "6px" }}>Create password</label>
            <div style={{ position: "relative" }}>
              <input type={showPassword ? "text" : "password"} required autoComplete="new-password" value={password} onChange={(e) => setPassword(e.target.value)}
                style={{ width: "100%", height: "44px", border: "1px solid #d7d3c9", borderRadius: "6px", background: "#ffffff", color: "#171a22", padding: "0 40px 0 14px", fontSize: "14px", outline: "none", boxSizing: "border-box" }}
                onFocus={(e) => e.target.style.borderColor = "#205fdc"}
                onBlur={(e) => e.target.style.borderColor = "#d7d3c9"}
                placeholder="Minimum 8 characters" />
              <button type="button" onClick={() => setShowPassword(!showPassword)}
                style={{ position: "absolute", right: "8px", top: "50%", transform: "translateY(-50%)", background: "none", border: "none", cursor: "pointer", padding: "4px", color: "#6c707a" }}>
                {showPassword ? "🙈" : "👁"}
              </button>
            </div>
          </div>

          {/* Confirm Password */}
          <div>
            <label style={{ display: "block", fontSize: "12px", fontWeight: 600, color: "#343842", marginBottom: "6px" }}>Confirm password</label>
            <input type={showPassword ? "text" : "password"} required autoComplete="new-password" value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)}
              style={{ width: "100%", height: "44px", border: "1px solid #d7d3c9", borderRadius: "6px", background: "#ffffff", color: "#171a22", padding: "0 14px", fontSize: "14px", outline: "none", boxSizing: "border-box" }}
              onFocus={(e) => e.target.style.borderColor = "#205fdc"}
              onBlur={(e) => e.target.style.borderColor = "#d7d3c9"}
              placeholder="Re-enter password" />
          </div>

          {/* Referral Code */}
          <div>
            <label style={{ display: "block", fontSize: "12px", fontWeight: 600, color: "#343842", marginBottom: "6px" }}>
              Referral Code <span style={{ fontWeight: 400, color: "#6c707a" }}>(optional)</span>
            </label>
            <input type="text" autoComplete="off" value={referralCode} onChange={(e) => setReferralCode(e.target.value)}
              style={{ width: "100%", height: "44px", border: "1px solid #d7d3c9", borderRadius: "6px", background: "#ffffff", color: "#171a22", padding: "0 14px", fontSize: "14px", outline: "none", boxSizing: "border-box" }}
              onFocus={(e) => e.target.style.borderColor = "#205fdc"}
              onBlur={(e) => e.target.style.borderColor = "#d7d3c9"}
              placeholder="PAT-XXXX..." />
          </div>

          {/* Submit button — explicit colors */}
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
            {loading ? (<><IconLoader2 size={16} className="animate-spin" /> Creating...</>) : "Create my account"}
          </button>
        </form>
      </div>

      {/* Switch to login */}
      <p style={{ textAlign: "center", fontSize: "13px", color: "#6c707a", margin: "0" }}>
        Already have an account? <Link href="/login" style={{ color: "#205fdc", fontWeight: 600, textDecoration: "none" }}>Sign in</Link>
      </p>
    </div>
  );
}

export default function RegisterPage() {
  return (
    <Suspense fallback={<div style={{ display: "flex", alignItems: "center", justifyContent: "center", minHeight: "200px" }}><IconLoader2 size={24} className="animate-spin" style={{ color: "#6c707a" }} /></div>}>
      <RegisterForm />
    </Suspense>
  );
}
