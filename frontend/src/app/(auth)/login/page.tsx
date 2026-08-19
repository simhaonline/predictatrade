"use client";
import { useState } from "react";
import Link from "next/link";
import { useAuth } from "@/providers/auth-provider";
import { getApiErrorMessage } from "@/lib/errors";
import { IconEye, IconEyeOff, IconLoader2 } from "@tabler/icons-react";
import BrandLogo from "@/components/brand-logo";

export default function LoginPage() {
  const { login } = useAuth();
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
      await login(email, password);
    } catch (err: unknown) {
      setError(getApiErrorMessage(err, "Login failed"));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div
      className="flex flex-col items-center justify-center"
      style={{ gap: "clamp(0.5rem, 1.5vh, 1rem)" }}
    >
      {/* Logo + heading block */}
      <div className="text-center flex flex-col items-center" style={{ gap: "0.375rem" }}>
        <BrandLogo />
        <h1 className="font-bold text-pat-text-primary" style={{ fontSize: "clamp(1.1rem, 2.5vh, 1.5rem)" }}>Sign In</h1>
        <p className="text-pat-text-secondary" style={{ fontSize: "clamp(0.7rem, 1.5vh, 0.85rem)" }}>Secure Platform Access</p>
      </div>

      {/* Login card */}
      <div className="bg-pat-bg-surface border border-pat-card-border rounded-lg shadow-sm w-full"
           style={{ padding: "clamp(0.75rem, 2vh, 1.25rem)" }}>
        <form onSubmit={handleSubmit} suppressHydrationWarning style={{ display: "flex", flexDirection: "column", gap: "clamp(0.5rem, 1.2vh, 0.75rem)" }}>
          {error && (
            <div className="text-pat-danger rounded-md px-3 py-2"
                 style={{ fontSize: "0.8rem", background: "hsl(var(--pat-badge-danger-bg) / 0.1)", border: "1px solid hsl(var(--pat-badge-danger-bg) / 0.2)" }}>
              {error}
            </div>
          )}
          <div>
            <label htmlFor="email" className="block text-pat-text-primary mb-1"
                   style={{ fontSize: "clamp(0.7rem, 1.4vh, 0.85rem)", fontWeight: 500 }}>Email</label>
            <input
              id="email" type="email" required autoComplete="email" value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full rounded-md border border-pat-input-border bg-pat-input-bg text-pat-input-text focus:outline-none focus:ring-2 focus:ring-pat-primary focus:border-pat-primary transition-colors"
              style={{ padding: "clamp(0.4rem, 1vh, 0.55rem)", fontSize: "clamp(0.8rem, 1.5vh, 0.9rem)" }}
              placeholder="you@example.com"
            />
          </div>
          <div>
            <label htmlFor="password" className="block text-pat-text-primary mb-1"
                   style={{ fontSize: "clamp(0.7rem, 1.4vh, 0.85rem)", fontWeight: 500 }}>Password</label>
            <div className="relative">
              <input
                id="password" type={showPassword ? "text" : "password"} required autoComplete="current-password" value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full rounded-md border border-pat-input-border bg-pat-input-bg text-pat-input-text focus:outline-none focus:ring-2 focus:ring-pat-primary focus:border-pat-primary transition-colors pr-10"
                style={{ padding: "clamp(0.4rem, 1vh, 0.55rem)", fontSize: "clamp(0.8rem, 1.5vh, 0.9rem)" }}
                placeholder="••••••••"
              />
              <button
                type="button" onClick={() => setShowPassword(!showPassword)}
                className="absolute right-2 top-1/2 -translate-y-1/2 text-pat-text-muted hover:text-pat-text-primary p-1"
                aria-label={showPassword ? "Hide password" : "Show password"}
              >
                {showPassword ? <IconEyeOff size={16} /> : <IconEye size={16} />}
              </button>
            </div>
          </div>
          <button type="submit" disabled={loading}
            className="w-full rounded-md bg-pat-primary text-pat-primary-foreground hover:bg-pat-primary-hover disabled:opacity-50 transition-colors flex items-center justify-center gap-2 font-semibold"
            style={{ padding: "clamp(0.45rem, 1.1vh, 0.6rem)", fontSize: "clamp(0.8rem, 1.5vh, 0.9rem)" }}>
            {loading ? (<><IconLoader2 size={16} className="animate-spin" /> Signing in...</>) : "Sign In"}
          </button>
        </form>
      </div>

      {/* Links */}
      <div className="flex justify-between w-full text-pat-text-secondary"
           style={{ fontSize: "clamp(0.7rem, 1.4vh, 0.85rem)" }}>
        <Link href="/forgot-password" className="hover:text-pat-text-primary transition-colors">Forgot password?</Link>
        <Link href="/register" className="hover:text-pat-text-primary transition-colors">Create account</Link>
      </div>
    </div>
  );
}
