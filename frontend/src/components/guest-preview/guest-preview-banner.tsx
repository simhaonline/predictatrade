"use client";
import { IconClock, IconLock } from "@tabler/icons-react";

/**
 * Fixed countdown banner for the guest preview. The countdown is DISPLAY-ONLY
 * (derived from the server-authoritative remainingSeconds). Text:
 * "Free preview — 04:12 remaining. Register to keep access."
 */
export function GuestPreviewBanner({ remainingSeconds }: { remainingSeconds: number }) {
  const mm = Math.floor(remainingSeconds / 60).toString().padStart(2, "0");
  const ss = Math.floor(remainingSeconds % 60).toString().padStart(2, "0");
  const low = remainingSeconds <= 60;

  return (
    <div
      className={`fixed top-0 left-0 right-0 z-[150] flex items-center justify-center gap-2 px-4 py-1.5 text-xs font-medium text-center transition-colors ${
        low
          ? "bg-pat-danger/15 text-pat-danger border-b border-pat-danger/30"
          : "bg-pat-primary/10 text-pat-primary border-b border-pat-primary/20"
      }`}
      role="status"
      aria-live="polite"
      data-testid="guest-preview-banner"
    >
      {low ? <IconLock size={14} /> : <IconClock size={14} />}
      <span>
        Free preview — <span data-testid="guest-preview-countdown">{mm}:{ss}</span> remaining. Register to keep access.
      </span>
    </div>
  );
}
