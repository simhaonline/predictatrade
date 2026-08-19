import type { Metadata } from "next";
import LegalLayout from "@/components/legal/legal-layout";
export const metadata: Metadata = { title: "Privacy Policy | Predict-A-Trade" };
export default function PrivacyPage() {
  return (
    <LegalLayout title="Privacy Policy" lastUpdated="August 2026">
      <section><h2>1. Information We Collect</h2><p>We collect: your email address, display name, authentication credentials (hashed), subscription and billing records, device fingerprints, MT4/MT5 account identifiers, and trading signal delivery records.</p></section>
      <section><h2>2. How We Use Your Information</h2><p>We use your information to: authenticate you, manage subscriptions and billing, deliver trading signals, manage device licensing, process referral commissions and payouts, and improve platform quality.</p></section>
      <section><h2>3. Data Storage</h2><p>Your data is stored in encrypted PostgreSQL databases. Authentication tokens are stored as HttpOnly cookies. We do not store passwords in plain text.</p></section>
      <section><h2>4. Cookies</h2><p>We use cookies for authentication (refresh tokens), theme preferences, and accessibility settings. Analytics and marketing cookies are only loaded with your consent. See our Cookie Policy for details.</p></section>
      <section><h2>5. Data Sharing</h2><p>We do not sell your personal data. We may share data with payment processors for billing purposes. Commission and payout information is shared with referred users as part of the referral program.</p></section>
      <section><h2>6. Data Retention</h2><p>We retain account data for the duration of your subscription and a reasonable period thereafter for audit and legal compliance. Trading signal data is retained for historical analysis.</p></section>
      <section><h2>7. Your Rights</h2><p>You may request access to your data, correction of inaccurate data, or deletion of your account. Contact support@predictatrade.com to exercise these rights.</p></section>
      <section><h2>8. Security</h2><p>We implement JWT-based authentication with refresh token rotation, HTTPS/TLS encryption, RBAC access controls, and audit logging. However, no system is perfectly secure.</p></section>
      <section><h2>9. Changes to This Policy</h2><p>We may update this policy periodically. We will notify users of significant changes.</p></section>
      <section><h2>10. Contact</h2><p>For privacy questions: support@predictatrade.com</p></section>
    </LegalLayout>
  );
}
