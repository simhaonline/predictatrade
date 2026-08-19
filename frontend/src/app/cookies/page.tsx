import type { Metadata } from "next";
import LegalLayout from "@/components/legal/legal-layout";
export const metadata: Metadata = { title: "Cookie Policy | Predict-A-Trade" };
export default function CookiesPage() {
  return (
    <LegalLayout title="Cookie Policy" lastUpdated="August 2026">
      <section><h2>1. What Are Cookies</h2><p>Cookies are small text files stored on your device by your browser. They are widely used to make websites work efficiently and provide information to site owners.</p></section>
      <section><h2>2. Cookies We Use</h2><h3>Strictly Necessary Cookies</h3><ul><li><strong>pat_refresh_token</strong> — HttpOnly cookie for secure session refresh. Required for authentication. Deleted on logout.</li><li><strong>pat_access_token</strong> — Short-lived access token cookie for middleware route protection. Expires in 15 minutes.</li></ul><h3>Preference Cookies</h3><ul><li><strong>pat_accessibility</strong> — Stores font scale, high contrast, reduced motion, and keyboard navigation preferences.</li><li><strong>theme</strong> — Stores your display theme preference (system, light, or dark).</li></ul><h3>Analytics Cookies</h3><p>No analytics cookies are currently in use. If analytics are added in the future, they will only be loaded after you provide consent.</p><h3>Marketing Cookies</h3><p>No marketing or advertising cookies are currently in use.</p></section>
      <section><h2>3. Managing Cookies</h2><p>You can manage your cookie preferences at any time using the Cookie Settings link in the footer, or by clearing cookies in your browser settings. Strictly necessary cookies cannot be disabled.</p></section>
      <section><h2>4. Local Storage</h2><p>We use browser local storage for: cookie consent preferences (pat_cookie_consent) and accessibility settings.</p></section>
      <section><h2>5. Third-Party Cookies</h2><p>We do not currently use third-party cookies.</p></section>
      <section><h2>6. Changes to This Policy</h2><p>If we add new cookies, we will update this policy and may request renewed consent.</p></section>
    </LegalLayout>
  );
}
