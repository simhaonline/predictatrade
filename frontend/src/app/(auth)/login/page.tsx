"use client";
import { useState } from "react";
import Link from "next/link";
import { customInstance } from "@/lib/axios-instance";
import { useRouter } from "next/navigation";
import { getApiErrorMessage } from "@/lib/errors";
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
          localStorage.setItem("pat_access_token", accessToken);
          window.dispatchEvent(new Event("pat:auth-changed"));
        }
        if (user?.role === "ADMIN") router.push("/admin/dashboard");
        else router.push("/dashboard/live");
      }
    } catch (err: unknown) {
      setError(getApiErrorMessage(err, "Invalid credentials"));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-5">
      {/* Meta header */}
      <div className="flex items-center justify-between gap-5">
        <span className="text-pat-primary font-bold uppercase tracking-widest" style={{ fontSize: "9px" }}>Secure platform access</span>
        <span className="flex-1 h-px bg-pat-border" />
        <span className="text-pat-primary font-bold" style={{ fontSize: "9px" }}>01</span>
      </div>

      {/* Heading */}
      <div>
        <h1 className="font-serif text-pat-text-primary" style={{ fontSize: "clamp(43px, 4.3vw, 60px)", lineHeight: "0.96", fontWeight: 600, letterSpacing: "-0.048em" }}>
          Welcome <em className="text-pat-primary italic font-medium">back.</em>
        </h1>
        <p className="text-pat-text-secondary mt-4" style={{ fontSize: "14px", lineHeight: "1.7", maxWidth: "440px" }}>
          Sign in to continue to your market-intelligence dashboard.
        </p>
      </div>

      {/* Auth card */}
      <div className="relative border border-pat-border rounded-lg shadow-lg p-7"
           style={{ background: "rgba(255,255,255,0.60)", backdropFilter: "blur(12px)" }}>
        {/* Blue top accent */}
        <div className="absolute top-0 left-0 w-[74px] h-[3px] bg-pat-primary rounded-tl-lg" />

        <form onSubmit={handleSubmit} className="space-y-5">
          {error && (
            <div className="text-pat-danger rounded-md px-3 py-2" style={{ fontSize: "0.8rem", background: "hsl(var(--pat-badge-danger-bg) / 0.1)", border: "1px solid hsl(var(--pat-badge-danger-bg) / 0.2)" }}>
              {error}
            </div>
          )}

          {/* Email */}
          <div>
            <label className="block text-pat-text-secondary mb-2 font-bold uppercase tracking-wider" style={{ fontSize: "11px" }}>Email address</label>
            <div className="relative">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" className="absolute left-4 top-1/2 -translate-y-1/2 w-[18px] h-[18px] text-gray-400 pointer-events-none z-10">
                <rect x="3" y="5" width="18" height="14" rx="1"/><path d="m4 7 8 6 8-6"/>
              </svg>
              <input type="email" required autoComplete="email" value={email} onChange={(e) => setEmail(e.target.value)}
                className="w-full border border-pat-border bg-white/80 text-pat-text-primary focus:outline-none focus:ring-2 focus:ring-pat-primary transition-colors rounded-none"
                style={{ height: "50px", padding: "0 46px 0 45px", fontSize: "14px", fontWeight: 500 }} placeholder="you@example.com" />
            </div>
          </div>

          {/* Password */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-pat-text-secondary font-bold uppercase tracking-wider" style={{ fontSize: "11px" }}>Password</label>
              <Link href="/forgot-password" className="text-pat-primary hover:underline" style={{ fontSize: "10px", fontWeight: 700 }}>Forgot password?</Link>
            </div>
            <div className="relative">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" className="absolute left-4 top-1/2 -translate-y-1/2 w-[18px] h-[18px] text-gray-400 pointer-events-none z-10">
                <rect x="4" y="10" width="16" height="11" rx="1"/><path d="M8 10V7a4 4 0 0 1 8 0v3"/>
              </svg>
              <input type={showPassword ? "text" : "password"} required autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)}
                className="w-full border border-pat-border bg-white/80 text-pat-text-primary focus:outline-none focus:ring-2 focus:ring-pat-primary transition-colors rounded-none"
                style={{ height: "50px", padding: "0 46px 0 45px", fontSize: "14px", fontWeight: 500 }} placeholder="Enter your password" />
              <button type="button" onClick={() => setShowPassword(!showPassword)} className="absolute right-2 top-1/2 -translate-y-1/2 w-9 h-9 grid place-items-center text-gray-500 hover:text-pat-primary transition-colors">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" className="w-[18px] h-[18px]">
                  <path d="M2.5 12s3.5-6 9.5-6 9.5 6 9.5 6-3.5 6-9.5 6S2.5 12 2.5 12Z"/><circle cx="12" cy="12" r="2.5"/>
                </svg>
              </button>
            </div>
          </div>

          {/* Submit */}
          <button type="submit" disabled={loading}
            className="w-full flex items-center justify-center gap-2 bg-pat-primary text-white font-bold uppercase tracking-widest hover:bg-pat-primary-hover disabled:opacity-70 transition-all rounded-none"
            style={{ height: "52px", fontSize: "10px", letterSpacing: "0.13em" }}>
            {loading ? (<><IconLoader2 size={15} className="animate-spin" /> Signing in...</>) : (<>Sign in securely <span aria-hidden="true">→</span></>)}
          </button>
        </form>
      </div>

      {/* Switch to register */}
      <p className="text-center text-pat-text-secondary" style={{ fontSize: "12px" }}>
        New to Predict-A-Trade? <Link href="/register" className="text-pat-text-primary font-bold hover:text-pat-primary transition-colors" style={{ borderBottom: "1px solid var(--pat-primary)" }}>Create an account</Link>
      </p>

      {/* Security note */}
      <p className="flex items-center justify-center gap-2 text-pat-text-muted font-bold uppercase tracking-wider" style={{ fontSize: "9px" }}>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" className="w-[14px] h-[14px] text-pat-primary"><path d="M12 3 5 6v5c0 4.6 2.8 8 7 10 4.2-2 7-5.4 7-10V6l-7-3Z"/><path d="m9 12 2 2 4-4"/></svg>
        Protected account access
      </p>
    </div>
  );
}
