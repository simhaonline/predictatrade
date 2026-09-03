"use client";
import { useState } from "react";
import { useSearchParams } from "next/navigation";
import { Suspense } from "react";
import { customInstance } from "@/lib/axios-instance";
import { getApiErrorMessage } from "@/lib/errors";

function ResetForm() {
  const token = useSearchParams().get("token");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState("");
  const [success, setSuccess] = useState(false);
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    if (password !== confirm) { setError("Passwords do not match"); return; }
    setLoading(true);
    try {
      await customInstance.post("/auth/reset", { token, password });
      setSuccess(true);
    } catch (err: unknown) {
      setError(getApiErrorMessage(err, "Reset failed"));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex flex-col items-center justify-center" style={{ gap: "clamp(0.5rem, 1.5vh, 1rem)" }}>
      <div className="text-center">
        <h1 className="font-bold text-pat-text-primary" style={{ fontSize: "clamp(1.1rem, 2.5vh, 1.5rem)" }}>Reset Password</h1>
      </div>
      {success ? (
        <div className="text-center text-sm text-pat-success">Password updated. You may now sign in.</div>
      ) : (
        <form onSubmit={handleSubmit} className="space-y-4">
          {error && <div className="text-sm text-pat-danger">{error}</div>}
          <div>
            <label className="block text-sm font-medium mb-1">New Password</label>
            <input type="password" required autoComplete="new-password" value={password} onChange={(e) => setPassword(e.target.value)}
              className="w-full rounded-md border border-pat-border-strong bg-pat-bg-surface px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary" />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Confirm Password</label>
            <input type="password" required autoComplete="new-password" value={confirm} onChange={(e) => setConfirm(e.target.value)}
              className="w-full rounded-md border border-pat-border-strong bg-pat-bg-surface px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary" />
          </div>
          <button type="submit" disabled={loading}
            className="w-full rounded-md bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/90 disabled:opacity-50">
            {loading ? "Updating..." : "Update Password"}
          </button>
        </form>
      )}
    </div>
  );
}

export default function ResetPasswordPage() {
  return (
    <Suspense fallback={<div className="text-center text-sm text-pat-text-secondary">Loading...</div>}>
      <ResetForm />
    </Suspense>
  );
}

