"use client";
import { useState } from "react";
import Link from "next/link";
import { customInstance } from "@/lib/axios-instance";
import { getApiErrorMessage } from "@/lib/errors";

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState("");
  const [sent, setSent] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await customInstance.post("/auth/forgot", { email });
      setSent(true);
    } catch (err: unknown) {
      setError(getApiErrorMessage(err, "Request failed"));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex flex-col items-center justify-center" style={{ gap: "clamp(0.5rem, 1.5vh, 1rem)" }}>
      <div className="text-center">
        <h1 className="font-bold text-pat-text-primary" style={{ fontSize: "clamp(1.1rem, 2.5vh, 1.5rem)" }}>Forgot Password</h1>
        <p className="text-sm text-pat-text-secondary">Enter your email to receive reset instructions</p>
      </div>
      {sent ? (
        <div className="text-center text-sm text-pat-success">If an account exists, instructions have been sent.</div>
      ) : (
        <form onSubmit={handleSubmit} className="space-y-4">
          {error && <div className="text-sm text-pat-danger">{error}</div>}
          <div>
            <label className="block text-sm font-medium mb-1">Email</label>
            <input type="email" required autoComplete="email" value={email} onChange={(e) => setEmail(e.target.value)}
              className="w-full rounded-md border border-pat-border-strong bg-pat-bg-surface px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary" />
          </div>
          <button type="submit" disabled={loading}
            className="w-full rounded-md bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/90 disabled:opacity-50">
            {loading ? "Sending..." : "Send Instructions"}
          </button>
        </form>
      )}
      <div className="text-center text-sm">
        <Link href="/login" className="hover:text-pat-text-primary">Back to sign in</Link>
      </div>
    </div>
  );
}
