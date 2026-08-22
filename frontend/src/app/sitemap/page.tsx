import type { Metadata } from "next";
import LegalLayout from "@/components/legal/legal-layout";
import Link from "next/link";
export const metadata: Metadata = { title: "Sitemap | Predict-A-Trade" };
export default function SitemapPage() {
  const publicPages = [
    { label: "Login", href: "/login" }, { label: "Register", href: "/register" },
    { label: "Forgot Password", href: "/forgot-password" }, { label: "Reset Password", href: "/reset-password" },
    { label: "Terms of Service", href: "/terms" }, { label: "Privacy Policy", href: "/privacy" },
    { label: "Complaints Procedure", href: "/complaints" }, { label: "Cookie Policy", href: "/cookies" },
    { label: "Sitemap", href: "/sitemap" },
  ];
  return (
    <LegalLayout title="Sitemap" lastUpdated="August 2026">
      <section><h2>Public Pages</h2><ul>{publicPages.map((p) => <li key={p.href}><Link href={p.href} className="text-pat-primary hover:underline">{p.label}</Link></li>)}</ul></section>
      <section><h2>Authenticated Pages</h2><p className="text-pat-text-muted">The following pages require authentication.</p><ul className="mt-3"><li><Link href="/dashboard/live" className="text-pat-primary hover:underline">User Dashboard</Link></li><li><Link href="/dashboard/signals" className="text-pat-primary hover:underline">Signals</Link></li><li><Link href="/dashboard/billing" className="text-pat-primary hover:underline">Billing &amp; Subscription</Link></li><li><Link href="/dashboard/settings" className="text-pat-primary hover:underline">Settings</Link></li></ul></section>
    </LegalLayout>
  );
}
