"use client";
import { useState } from "react";
import Link from "next/link";
import { customInstance } from "@/lib/axios-instance";
import { useRouter } from "next/navigation";
import { getApiErrorMessage } from "@/lib/errors";
import { IconLoader2 } from "@tabler/icons-react";
import BrandLogo from "@/components/brand-logo";

export default function RegisterPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [referralCode, setReferralCode] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    if (password !== confirmPassword) { setError("Passwords do not match"); return; }
    if (password.length < 8) { setError("Password must be at least 8 characters"); return; }
    setLoading(true);
    try {
      await customInstance.post("/auth/register", {
        email,
        password,
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
    <div className="flex flex-col items-center justify-center" style={{ gap: "clamp(0.5rem, 1.5vh, 1rem)" }}>
      <div className="text-center flex flex-col items-center" style={{ gap: "0.375rem" }}>
        <BrandLogo />
        <h1 className="font-bold text-pat-text-primary" style={{ fontSize: "clamp(1.1rem, 2.5vh, 1.5rem)" }}>Create Account</h1>
        <p className="text-pat-text-secondary" style={{ fontSize: "clamp(0.7rem, 1.5vh, 0.85rem)" }}>Join Predict-A-Trade</p>
      </div>
      <div className="bg-pat-bg-surface border border-pat-card-border rounded-lg shadow-sm w-full"
           style={{ padding: "clamp(0.75rem, 2vh, 1.25rem)" }}>
        <form onSubmit={handleSubmit} style={{ display: "flex", flexDirection: "column", gap: "clamp(0.5rem, 1.2vh, 0.75rem)" }}>
          {error && <div className="text-pat-danger rounded-md px-3 py-2"
            style={{ fontSize: "0.8rem", background: "hsl(var(--pat-badge-danger-bg) / 0.1)", border: "1px solid hsl(var(--pat-badge-danger-bg) / 0.2)" }}>{error}</div>}
          <div>
            <label className="block text-pat-text-primary mb-1" style={{ fontSize: "clamp(0.7rem, 1.4vh, 0.85rem)", fontWeight: 500 }}>Email</label>
            <input type="email" required autoComplete="email" value={email} onChange={(e) => setEmail(e.target.value)}
              className="w-full rounded-md border border-pat-input-border bg-pat-input-bg text-pat-input-text focus:outline-none focus:ring-2 focus:ring-pat-primary transition-colors"
              style={{ padding: "clamp(0.4rem, 1vh, 0.55rem)", fontSize: "clamp(0.8rem, 1.5vh, 0.9rem)" }} placeholder="you@example.com" />
          </div>
          <div>
            <label className="block text-pat-text-primary mb-1" style={{ fontSize: "clamp(0.7rem, 1.4vh, 0.85rem)", fontWeight: 500 }}>Password</label>
            <input type="password" required autoComplete="new-password" value={password} onChange={(e) => setPassword(e.target.value)}
              className="w-full rounded-md border border-pat-input-border bg-pat-input-bg text-pat-input-text focus:outline-none focus:ring-2 focus:ring-pat-primary transition-colors"
              style={{ padding: "clamp(0.4rem, 1vh, 0.55rem)", fontSize: "clamp(0.8rem, 1.5vh, 0.9rem)" }} placeholder="Minimum 8 characters" />
          </div>
          <div>
            <label className="block text-pat-text-primary mb-1" style={{ fontSize: "clamp(0.7rem, 1.4vh, 0.85rem)", fontWeight: 500 }}>Confirm Password</label>
            <input type="password" required autoComplete="new-password" value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)}
              className="w-full rounded-md border border-pat-input-border bg-pat-input-bg text-pat-input-text focus:outline-none focus:ring-2 focus:ring-pat-primary transition-colors"
              style={{ padding: "clamp(0.4rem, 1vh, 0.55rem)", fontSize: "clamp(0.8rem, 1.5vh, 0.9rem)" }} placeholder="Re-enter password" />
          </div>
          <div>
            <label className="block text-pat-text-primary mb-1" style={{ fontSize: "clamp(0.7rem, 1.4vh, 0.85rem)", fontWeight: 500 }}>Referral Code <span className="text-pat-text-muted">(optional)</span></label>
            <input type="text" autoComplete="off" value={referralCode} onChange={(e) => setReferralCode(e.target.value)}
              className="w-full rounded-md border border-pat-input-border bg-pat-input-bg text-pat-input-text focus:outline-none focus:ring-2 focus:ring-pat-primary transition-colors"
              style={{ padding: "clamp(0.4rem, 1vh, 0.55rem)", fontSize: "clamp(0.8rem, 1.5vh, 0.9rem)" }} placeholder="PAT-XXXX..." />
          </div>
          <button type="submit" disabled={loading}
            className="w-full rounded-md bg-pat-primary text-pat-primary-foreground hover:bg-pat-primary-hover disabled:opacity-50 transition-colors flex items-center justify-center gap-2 font-semibold"
            style={{ padding: "clamp(0.45rem, 1.1vh, 0.6rem)", fontSize: "clamp(0.8rem, 1.5vh, 0.9rem)" }}>
            {loading ? (<><IconLoader2 size={16} className="animate-spin" /> Creating...</>) : "Create Account"}
          </button>
        </form>
      </div>
      <div className="text-center w-full text-pat-text-secondary" style={{ fontSize: "clamp(0.7rem, 1.4vh, 0.85rem)" }}>
        Already have an account? <Link href="/login" className="text-pat-primary hover:underline">Sign in</Link>
      </div>
    </div>
  );
}
