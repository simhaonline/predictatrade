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
      router.push("/login");
    } catch (err: unknown) {
      setError(getApiErrorMessage(err, "Registration failed"));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-5">
      {/* Meta header */}
      <div className="flex items-center justify-between gap-5">
        <span className="text-pat-primary font-bold uppercase tracking-widest" style={{ fontSize: "9px" }}>Create your account</span>
        <span className="flex-1 h-px bg-pat-border" />
        <span className="text-pat-primary font-bold" style={{ fontSize: "9px" }}>01</span>
      </div>

      {/* Heading */}
      <div>
        <h1 className="font-serif text-pat-text-primary" style={{ fontSize: "clamp(43px, 4.3vw, 60px)", lineHeight: "0.96", fontWeight: 600, letterSpacing: "-0.048em" }}>
          Start with a <em className="text-pat-primary italic font-medium">clearer view.</em>
        </h1>
        <p className="text-pat-text-secondary mt-4" style={{ fontSize: "14px", lineHeight: "1.7", maxWidth: "440px" }}>
          Set up your account details. It only takes a moment.
        </p>
      </div>

      {/* Auth card */}
      <div className="relative border border-pat-border rounded-lg shadow-lg p-7"
           style={{ background: "rgba(255,255,255,0.60)", backdropFilter: "blur(12px)" }}>
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
            <label className="block text-pat-text-secondary mb-2 font-bold uppercase tracking-wider" style={{ fontSize: "11px" }}>Create password</label>
            <div className="relative">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" className="absolute left-4 top-1/2 -translate-y-1/2 w-[18px] h-[18px] text-gray-400 pointer-events-none z-10">
                <rect x="4" y="10" width="16" height="11" rx="1"/><path d="M8 10V7a4 4 0 0 1 8 0v3"/>
              </svg>
              <input type={showPassword ? "text" : "password"} required autoComplete="new-password" value={password} onChange={(e) => setPassword(e.target.value)}
                className="w-full border border-pat-border bg-white/80 text-pat-text-primary focus:outline-none focus:ring-2 focus:ring-pat-primary transition-colors rounded-none"
                style={{ height: "50px", padding: "0 46px 0 45px", fontSize: "14px", fontWeight: 500 }} placeholder="Create a strong password" />
              <button type="button" onClick={() => setShowPassword(!showPassword)} className="absolute right-2 top-1/2 -translate-y-1/2 w-9 h-9 grid place-items-center text-gray-500 hover:text-pat-primary transition-colors">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" className="w-[18px] h-[18px]">
                  <path d="M2.5 12s3.5-6 9.5-6 9.5 6 9.5 6-3.5 6-9.5 6S2.5 12 2.5 12Z"/><circle cx="12" cy="12" r="2.5"/>
                </svg>
              </button>
            </div>
          </div>

          {/* Confirm Password */}
          <div>
            <label className="block text-pat-text-secondary mb-2 font-bold uppercase tracking-wider" style={{ fontSize: "11px" }}>Confirm password</label>
            <div className="relative">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" className="absolute left-4 top-1/2 -translate-y-1/2 w-[18px] h-[18px] text-gray-400 pointer-events-none z-10">
                <rect x="4" y="10" width="16" height="11" rx="1"/><path d="M8 10V7a4 4 0 0 1 8 0v3"/>
              </svg>
              <input type={showPassword ? "text" : "password"} required autoComplete="new-password" value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)}
                className="w-full border border-pat-border bg-white/80 text-pat-text-primary focus:outline-none focus:ring-2 focus:ring-pat-primary transition-colors rounded-none"
                style={{ height: "50px", padding: "0 46px 0 45px", fontSize: "14px", fontWeight: 500 }} placeholder="Repeat your password" />
            </div>
          </div>

          {/* Referral Code */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-pat-text-secondary font-bold uppercase tracking-wider" style={{ fontSize: "11px" }}>Referral Code <span className="text-pat-text-muted normal-case" style={{ fontWeight: 500 }}>(optional)</span></label>
            </div>
            <div className="relative">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" className="absolute left-4 top-1/2 -translate-y-1/2 w-[18px] h-[18px] text-gray-400 pointer-events-none z-10">
                <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
              </svg>
              <input type="text" autoComplete="off" value={referralCode} onChange={(e) => setReferralCode(e.target.value)}
                className="w-full border border-pat-border bg-white/80 text-pat-text-primary focus:outline-none focus:ring-2 focus:ring-pat-primary transition-colors rounded-none"
                style={{ height: "50px", padding: "0 46px 0 45px", fontSize: "14px", fontWeight: 500 }} placeholder="PAT-XXXX..." />
            </div>
          </div>

          {/* Submit */}
          <button type="submit" disabled={loading}
            className="w-full flex items-center justify-center gap-2 bg-pat-primary text-white font-bold uppercase tracking-widest hover:bg-pat-primary-hover disabled:opacity-70 transition-all rounded-none"
            style={{ height: "52px", fontSize: "10px", letterSpacing: "0.13em" }}>
            {loading ? (<><IconLoader2 size={15} className="animate-spin" /> Creating...</>) : (<>Create my account <span aria-hidden="true">→</span></>)}
          </button>
        </form>
      </div>

      {/* Switch to login */}
      <p className="text-center text-pat-text-secondary" style={{ fontSize: "12px" }}>
        Already have an account? <Link href="/login" className="text-pat-text-primary font-bold hover:text-pat-primary transition-colors" style={{ borderBottom: "1px solid var(--pat-primary)" }}>Sign in</Link>
      </p>

      {/* Security note */}
      <p className="flex items-center justify-center gap-2 text-pat-text-muted font-bold uppercase tracking-wider" style={{ fontSize: "9px" }}>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" className="w-[14px] h-[14px] text-pat-primary"><path d="M12 3 5 6v5c0 4.6 2.8 8 7 10 4.2-2 7-5.4 7-10V6l-7-3Z"/><path d="m9 12 2 2 4-4"/></svg>
        Your information stays private
      </p>
    </div>
  );
}

export default function RegisterPage() {
  return (
    <Suspense fallback={<div className="flex items-center justify-center min-h-screen"><IconLoader2 size={24} className="animate-spin text-pat-text-muted" /></div>}>
      <RegisterForm />
    </Suspense>
  );
}
