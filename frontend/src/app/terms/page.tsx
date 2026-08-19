import type { Metadata } from "next";
import LegalLayout from "@/components/legal/legal-layout";
export const metadata: Metadata = { title: "Terms and Policies | Predict-A-Trade" };
export default function TermsPage() {
  return (
    <LegalLayout title="Terms and Policies" lastUpdated="August 2026">
      <section><h2>1. Acceptance of Terms</h2><p>By accessing and using the Predict-A-Trade platform (the &quot;Service&quot;), you agree to be bound by these Terms and Policies. If you do not agree, please do not use the Service.</p></section>
      <section><h2>2. Description of Service</h2><p>Predict-A-Trade is a XAUUSD signal generation and trading analysis platform that provides market analysis, signal generation, and related tools. The Service is provided by Simha Online.</p></section>
      <section><h2>3. Eligibility</h2><p>You must be at least 18 years old and legally capable of entering into contracts to use this Service. By registering, you represent that you meet these requirements.</p></section>
      <section><h2>4. Accounts and Licenses</h2><p>You are responsible for maintaining the security of your account credentials. Each license is tied to a single device. Unauthorized sharing of accounts or licenses may result in termination.</p></section>
      <section><h2>5. Trading Disclaimer</h2><p>The Service provides analysis and signals for informational purposes only. Nothing on this platform constitutes financial advice. Trading in financial markets, including XAUUSD, carries significant risk of loss. Past performance does not guarantee future results. You are solely responsible for your trading decisions.</p></section>
      <section><h2>6. Subscription and Billing</h2><p>Subscription fees are billed according to your selected plan. Refunds are handled on a case-by-case basis. Subscription cancellation stops future billing but does not automatically refund the current period.</p></section>
      <section><h2>7. Referral Program</h2><p>Commissions are earned through the referral program according to the published commission structure. Commission reversals may occur for refunded or fraudulent referrals.</p></section>
      <section><h2>8. Prohibited Conduct</h2><p>You agree not to: reverse engineer the platform, attempt to circumvent licensing or device authorization, use the Service for illegal activities, or redistribute signals without authorization.</p></section>
      <section><h2>9. Intellectual Property</h2><p>All content, software, indicators, strategies, and designs on this platform are the property of Simha Online. You may not copy, modify, or distribute platform content without written permission.</p></section>
      <section><h2>10. Limitation of Liability</h2><p>The Service is provided &quot;as is&quot; without warranties of any kind. Simha Online is not liable for trading losses, technical failures, data inaccuracies, or any indirect damages arising from use of the Service.</p></section>
      <section><h2>11. Termination</h2><p>We may terminate or suspend your account at any time for violation of these terms. You may cancel your subscription at any time.</p></section>
      <section><h2>12. Changes to Terms</h2><p>We may update these terms periodically. Continued use after changes constitutes acceptance of the updated terms.</p></section>
      <section><h2>13. Contact</h2><p>For questions about these terms, contact: support@predictatrade.com</p></section>
    </LegalLayout>
  );
}
