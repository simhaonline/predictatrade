"use client";
import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useGuestPreview } from "@/lib/guest-preview-api";
import { GuestPreviewBanner } from "./guest-preview-banner";
import { RegistrationModal } from "./registration-modal";
import { PostRegistrationSocial } from "./post-registration-social";

/**
 * GuestPreviewGate — wraps the live dashboard for unauthenticated (guest)
 * visitors. Enforces the SERVER-SIDE preview timer:
 *
 *   - Shows a fixed countdown banner (display-only; server is the source of truth).
 *   - Renders children (the full dashboard) during the preview.
 *   - Blurs the signal/strategy panels (Signal Grid, Edge Matrix, Strategy
 *     signals, Neural Shell) behind an "Unlock with free registration" overlay
 *     so there is a concrete reason to register.
 *   - When the server reports locked=true, shows the RegistrationModal.
 *   - After successful OTP verification, shows the optional (skippable) social
 *     follow step, then redirects to the authenticated /dashboard/live.
 *
 * A returning visitor with a verified account never reaches this gate — the
 * proxy + AuthProvider route them straight to /dashboard/live.
 */
export function GuestPreviewGate({ children }: { children: React.ReactNode }) {
  const { status, loading } = useGuestPreview();
  const router = useRouter();
  const [showSocial, setShowSocial] = useState(false);
  const [registered, setRegistered] = useState(false);
  const [dismissed, setDismissed] = useState(false);

  // Derived: show the lock modal when the server says locked and the user
  // hasn't registered or dismissed it. No setState-in-effect needed.
  const showLock = Boolean(status?.locked) && !registered && !dismissed;

  // Listen for a successful registration (dispatched by the modal on verify).
  useEffect(() => {
    const handler = () => {
      setRegistered(true);
      setShowSocial(true);
    };
    window.addEventListener("pat:guest-registered", handler);
    return () => window.removeEventListener("pat:guest-registered", handler);
  }, []);

  const continueToDashboard = useCallback(() => {
    setShowSocial(false);
    // Authenticated session is now set — go to the real dashboard.
    router.replace("/dashboard/live");
  }, [router]);

  if (loading && !status) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <div className="inline-block h-8 w-8 animate-spin rounded-full border-2 border-pat-border-strong border-t-pat-primary" />
      </div>
    );
  }

  const remaining = status?.remainingSeconds ?? 0;
  const isPreviewing = !status?.locked && remaining > 0;

  return (
    <div className="relative">
      {isPreviewing && <GuestPreviewBanner remainingSeconds={remaining} />}

      {/* Top padding so content isn't hidden behind the fixed banner */}
      <div className={isPreviewing ? "pt-8" : ""}>
        {children}
      </div>

      {/* Blurred overlay over signal/strategy panels during preview.
          The panels themselves are NOT removed — they keep rendering behind the
          blur so the live data pipeline stays active; the overlay just obscures
          them to give a concrete reason to register. */}
      {isPreviewing && <SignalOverlay />}

      {/* Lock modal when the server-side timer expires */}
      {showLock && <RegistrationModal onClose={() => setDismissed(true)} />}

      {/* Optional, skippable post-registration social follow */}
      {showSocial && <PostRegistrationSocial onContinue={continueToDashboard} />}
    </div>
  );
}

/**
 * Visual overlay that blurs the signal/strategy area. It sits as a sibling
 * overlay (pointer-events-none) so the underlying panels keep working but are
 * obscured. A CTA prompts registration.
 */
function SignalOverlay() {
  return (
    <div
      className="pointer-events-none absolute inset-0 z-[120] flex items-center justify-center"
      aria-hidden="true"
      data-testid="guest-signal-overlay"
    >
      {/* Blur backdrop over the lower (signal/strategy) portion of the dashboard */}
      <div className="absolute left-0 right-0 bottom-0 top-24 backdrop-blur-md bg-pat-bg-page/40 rounded-lg" />
      <div className="relative pointer-events-auto flex flex-col items-center gap-2 p-6 rounded-xl bg-pat-bg-surface border border-pat-card-border shadow-lg max-w-sm text-center">
        <span className="text-sm font-semibold text-pat-text-primary">Unlock with free registration</span>
        <p className="text-xs text-pat-text-secondary">
          Signal Grid, Edge Matrix, Strategy signals, and the Neural Shell decision core are available after a free registration.
        </p>
      </div>
    </div>
  );
}
