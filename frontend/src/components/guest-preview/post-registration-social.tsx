"use client";
import { IconBrandWhatsapp, IconBrandTelegram, IconBrandInstagram, IconBrandYoutube, IconBrandX, IconBrandTiktok, IconBrandFacebook, IconArrowRight } from "@tabler/icons-react";

/**
 * Post-registration social follow step — shown AFTER successful registration.
 *
 * COMPLIANCE: Nothing is gated behind a like/follow/review. These are optional
 * "Follow us for the same alerts" buttons. Clicking is never required; there is
 * always a "Skip / Continue to dashboard" action. No reward, discount, or perk
 * is offered in exchange for a follow or review (Google/Meta policy compliant).
 *
 * The Google review ask happens LATER (welcome email + dismissible toast after
 * ~3 days or a winning signal), never as a gate.
 */
const SOCIALS = [
  { label: "WhatsApp", href: "https://wa.me/", icon: IconBrandWhatsapp },
  { label: "Telegram", href: "https://t.me/", icon: IconBrandTelegram },
  { label: "Instagram", href: "https://instagram.com/", icon: IconBrandInstagram },
  { label: "YouTube", href: "https://youtube.com/", icon: IconBrandYoutube },
  { label: "X", href: "https://x.com/", icon: IconBrandX },
  { label: "TikTok", href: "https://tiktok.com/", icon: IconBrandTiktok },
  { label: "Facebook", href: "https://facebook.com/", icon: IconBrandFacebook },
];

export function PostRegistrationSocial({ onContinue }: { onContinue: () => void }) {
  return (
    <div className="fixed inset-0 z-[200] flex items-center justify-center bg-black/70 backdrop-blur-sm" role="dialog" aria-modal="true" aria-label="Follow us (optional)">
      <div className="bg-pat-bg-surface border border-pat-card-border rounded-xl shadow-2xl w-full max-w-md mx-4">
        <div className="p-5 text-center space-y-4">
          <h2 className="text-base font-semibold text-pat-text-primary">You&apos;re all set! 🎉</h2>
          <p className="text-xs text-pat-text-secondary">
            Follow us on your favourite channel for the same XAUUSD alerts and updates. Completely optional.
          </p>
          <div className="grid grid-cols-4 gap-2">
            {SOCIALS.map((s) => (
              <a key={s.label} href={s.href} target="_blank" rel="noopener noreferrer"
                className="flex flex-col items-center gap-1 p-2 rounded-lg border border-pat-border hover:bg-pat-bg-surface-secondary transition-colors"
                aria-label={`Follow on ${s.label}`}>
                <s.icon size={20} className="text-pat-text-secondary" />
                <span className="text-[10px] text-pat-text-muted">{s.label}</span>
              </a>
            ))}
          </div>
          <button onClick={onContinue}
            className="w-full rounded-md bg-pat-primary text-pat-primary-foreground hover:bg-pat-primary-hover transition-colors flex items-center justify-center gap-2 font-semibold py-2 text-sm">
            Skip / Continue to dashboard <IconArrowRight size={16} />
          </button>
          <p className="text-[10px] text-pat-text-muted">
            No reward is offered for following. We never gate content behind a review, like, or follow.
          </p>
        </div>
      </div>
    </div>
  );
}
