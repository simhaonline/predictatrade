"use client";
import Link from "next/link";
import { usePathname } from "next/navigation";

const footerLinks = [
  { label: "Terms and Policies", href: "/terms" },
  { label: "Privacy Policy", href: "/privacy" },
  { label: "Complaints Procedure", href: "/complaints" },
  { label: "Sitemap", href: "/sitemap" },
  { label: "Cookie Policy", href: "/cookies" },
];

export default function Footer() {
  const pathname = usePathname();
  const isAuthPage = pathname === '/login' || pathname === '/register' || pathname === '/forgot-password' || pathname === '/reset-password' || pathname === '/verify-otp';

  return (
    <footer
      className={`flex-shrink-0 border-t border-pat-border ${isAuthPage ? 'bg-transparent' : 'bg-pat-bg-surface'}`}
      style={{ padding: "clamp(0.375rem, 1vh, 0.625rem)" }}
    >
      {/* Persistent risk disclaimer (compliance) */}
      <div className="max-w-7xl mx-auto px-4 pt-1 pb-0.5">
        <p className="text-center text-[10px] text-pat-text-muted leading-snug" data-testid="risk-disclaimer">
          Risk warning: This platform provides market monitoring and educational content only. It is not investment advice. Trading CFDs, forex and gold (XAUUSD) involves substantial risk of loss and is not suitable for all investors.
        </p>
      </div>
      <div className="max-w-7xl mx-auto px-4">
        <div className="flex flex-col sm:flex-row items-center justify-between gap-1 text-pat-text-muted"
             style={{ fontSize: "clamp(0.6rem, 1.2vh, 0.75rem)" }}>
          <div>© 2016–2026 Predict-A-Trade by Simha Online. All rights reserved.</div>
          <nav className="flex flex-wrap items-center gap-x-2 gap-y-0.5 justify-center" aria-label="Legal links">
            {footerLinks.map((link) => (
              <Link
                key={link.href}
                href={link.href}
                className="hover:text-pat-text-primary transition-colors"
              >
                {link.label}
              </Link>
            ))}
            <button
              onClick={() => window.dispatchEvent(new Event('pat:open-cookie-settings'))}
              className="hover:text-pat-text-primary transition-colors cursor-pointer"
            >
              Cookie Settings
            </button>
          </nav>
        </div>
      </div>
    </footer>
  );
}
