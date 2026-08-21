"use client";
import { Suspense, useState, useEffect } from "react";
import { useSearchParams } from "next/navigation";
import { customInstance } from "@/lib/axios-instance";
import BrandLogo from "@/components/brand-logo";
import { IconLoader2, IconCircleCheck, IconAlertTriangle } from "@tabler/icons-react";

/**
 * Unsubscribe page — reached via the unsubscribe link in every email.
 * Validates the signed token server-side and persists the opt-out immediately.
 * Honored across all marketing/email communications (idempotent).
 *
 * useSearchParams requires a Suspense boundary (Next.js build requirement).
 */
export default function UnsubscribePage() {
  return (
    <Suspense fallback={<UnsubscribeLoading />}>
      <UnsubscribeContent />
    </Suspense>
  );
}

function UnsubscribeLoading() {
  return (
    <div className="flex min-h-dvh flex-col items-center justify-center bg-pat-bg-page p-4">
      <div className="w-full max-w-md bg-pat-bg-surface border border-pat-card-border rounded-xl shadow-sm p-6 text-center">
        <div className="flex justify-center mb-4"><BrandLogo /></div>
        <IconLoader2 size={24} className="animate-spin text-pat-primary mx-auto" />
      </div>
    </div>
  );
}

function UnsubscribeContent() {
  const params = useSearchParams();
  const token = params.get("token") || "";
  // Initialize from token presence so no setState-in-effect is needed for the
  // missing-token case.
  const [state, setState] = useState<"loading" | "done" | "error">(token ? "loading" : "error");
  const [email, setEmail] = useState<string | null>(null);

  useEffect(() => {
    if (!token) return;
    let cancelled = false;
    (async () => {
      try {
        const res = await customInstance.post<{ success: boolean; email: string }>("/guest/unsubscribe", { token });
        if (cancelled) return;
        if (res.data?.success) {
          setEmail(res.data.email);
          setState("done");
        } else {
          setState("error");
        }
      } catch {
        if (!cancelled) setState("error");
      }
    })();
    return () => { cancelled = true; };
  }, [token]);

  return (
    <div className="flex min-h-dvh flex-col items-center justify-center bg-pat-bg-page p-4">
      <div className="w-full max-w-md bg-pat-bg-surface border border-pat-card-border rounded-xl shadow-sm p-6 text-center space-y-4">
        <div className="flex justify-center"><BrandLogo /></div>
        {state === "loading" && (
          <div className="flex flex-col items-center gap-2 text-pat-text-secondary">
            <IconLoader2 size={24} className="animate-spin text-pat-primary" />
            <p className="text-sm">Processing your unsubscribe…</p>
          </div>
        )}
        {state === "done" && (
          <div className="flex flex-col items-center gap-2">
            <IconCircleCheck size={32} className="text-pat-success" />
            <h1 className="text-base font-semibold text-pat-text-primary">You&apos;re unsubscribed</h1>
            <p className="text-xs text-pat-text-secondary">
              {email ? `We won't send marketing emails to ${email}. ` : ""}You will still receive essential account and security emails. This change is effective immediately.
            </p>
          </div>
        )}
        {state === "error" && (
          <div className="flex flex-col items-center gap-2">
            <IconAlertTriangle size={32} className="text-pat-danger" />
            <h1 className="text-base font-semibold text-pat-text-primary">Invalid or expired link</h1>
            <p className="text-xs text-pat-text-secondary">
              This unsubscribe link is invalid or has expired. If you believe this is an error, contact support@predictatrade.com.
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
