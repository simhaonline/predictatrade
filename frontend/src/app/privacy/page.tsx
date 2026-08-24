import type { Metadata } from "next";
import LegalLayout from "@/components/legal/legal-layout";

export const metadata: Metadata = { title: "Privacy Policy | Predict-A-Trade" };

export default function PrivacyPage() {
  return (
    <LegalLayout title="Privacy Policy" lastUpdated="25 August 2026">
      <section>
        <h2>1. Introduction</h2>
        <p>Simha FinTech (&quot;we&quot;, &quot;us&quot;, or &quot;our&quot;) respects your privacy and is committed to protecting your personal data. This Privacy Policy explains how we collect, use, store, share, and protect your personal information when you use the Predict-A-Trade platform (the &quot;Service&quot;).</p>
        <p>This Privacy Policy is published in accordance with applicable data protection laws, including the UAE Personal Data Protection Law (PDPL) (Federal Decree-Law No. 45 of 2021), the EU General Data Protection Regulation (GDPR) where applicable, and other applicable privacy regulations.</p>
      </section>

      <section>
        <h2>2. Data Controller</h2>
        <p>Simha FinTech is the data controller responsible for your personal data processed through the Service. Our contact details: <strong>support@predictatrade.com</strong>.</p>
      </section>

      <section>
        <h2>3. Personal Data We Collect</h2>
        <h3>3.1 Data you provide directly</h3>
        <ul>
          <li><strong>Account data:</strong> email address, display name, password (cryptographically hashed — never stored in plaintext), and profile preferences.</li>
          <li><strong>Subscription data:</strong> billing name, payment method identifiers (tokenized — we never store full card numbers), subscription plan, billing history, and invoice records.</li>
          <li><strong>Referral data:</strong> referral codes you create and use, referral relationships, commission earnings, and payout account information.</li>
          <li><strong>Device data:</strong> device fingerprints for licensing, device name, operating system, and browser type for authorized device management.</li>
          <li><strong>MetaTrader data:</strong> MT4/MT5 account numbers, broker names, and connection status (not your trading funds or positions — we do not hold or access your brokerage account).</li>
          <li><strong>Consent records:</strong> your responses to consent checkboxes during registration, including the timestamp, exact consent text version, and your IP address and user-agent at the time of consent.</li>
          <li><strong>Marketing preferences:</strong> your opt-in/opt-out choices for email, SMS, and phone marketing communications.</li>
        </ul>
        <h3>3.2 Data collected automatically</h3>
        <ul>
          <li><strong>Usage data:</strong> pages visited, features used, session duration, and interaction events for service improvement.</li>
          <li><strong>Technical data:</strong> IP address, user-agent string, browser type, operating system, screen resolution, and timezone.</li>
          <li><strong>Authentication data:</strong> login timestamps, session tokens (HttpOnly cookies), and multi-factor authentication enrollment status.</li>
          <li><strong>Market data preferences:</strong> selected symbols, chart settings, and indicator configurations.</li>
        </ul>
        <h3>3.3 Special categories</h3>
        <p>We do <strong>not</strong> collect or process special categories of personal data (such as race, religion, health data, biometric data, or political opinions) through the Service. If you voluntarily provide such information, it will be processed only with your explicit consent and in accordance with applicable law.</p>
      </section>

      <section>
        <h2>4. Legal Basis for Processing</h2>
        <p>We process your personal data on the following legal bases:</p>
        <ul>
          <li><strong>Contract performance:</strong> processing necessary to provide the Service you requested, including account creation, authentication, subscription management, and signal delivery.</li>
          <li><strong>Legal obligation:</strong> processing required to comply with applicable laws, including tax records, anti-fraud measures, and regulatory requirements.</li>
          <li><strong>Legitimate interest:</strong> processing for service improvement, security, fraud prevention, and analytics, balanced against your privacy rights.</li>
          <li><strong>Consent:</strong> processing of marketing communications, optional analytics cookies, and any other processing where you have given explicit, informed, and freely given consent. You may withdraw consent at any time.</li>
        </ul>
      </section>

      <section>
        <h2>5. How We Use Your Personal Data</h2>
        <ul>
          <li>Authenticate you and manage your account, sessions, and device licensing.</li>
          <li>Provide and deliver trading signals, market analysis, and platform features.</li>
          <li>Process subscription payments, manage billing, and issue invoices.</li>
          <li>Manage the referral program, calculate commissions, and process payouts.</li>
          <li>Communicate with you about your account, security alerts, and service updates.</li>
          <li>Send marketing communications (email, SMS, phone) — only with your explicit opt-in consent, which you can withdraw at any time.</li>
          <li>Maintain audit logs and compliance records as required by law.</li>
          <li>Detect, prevent, and investigate fraud, abuse, and security incidents.</li>
          <li>Improve and optimize the Service through analytics and usage monitoring.</li>
        </ul>
      </section>

      <section>
        <h2>6. Data Sharing and Recipients</h2>
        <p>We do <strong>not</strong> sell your personal data to any third party. We may share your data with the following categories of recipients:</p>
        <ul>
          <li><strong>Payment processors:</strong> for processing subscription payments and commission payouts (they process payment data under their own compliance and PCI-DSS obligations).</li>
          <li><strong>Cloud infrastructure providers:</strong> for hosting, database storage, and caching (data is processed under data processing agreements).</li>
          <li><strong>Email and notification providers:</strong> for transactional emails and marketing communications (only if you opted in).</li>
          <li><strong>Market data providers:</strong> for receiving market data feeds (they receive only API credentials, not your personal data).</li>
          <li><strong>Legal and regulatory authorities:</strong> when required by law, court order, or to protect our rights and safety.</li>
        </ul>
        <p>All third-party processors are bound by data processing agreements and are required to maintain appropriate security standards.</p>
      </section>

      <section>
        <h2>7. International Data Transfers</h2>
        <p>Your personal data may be processed in jurisdictions outside your country of residence. We ensure appropriate safeguards for international transfers, including:</p>
        <ul>
          <li>Standard Contractual Clauses (SCCs) where applicable.</li>
          <li>Adequacy decisions for jurisdictions deemed to provide sufficient data protection.</li>
          <li>Binding corporate rules for intra-group transfers where applicable.</li>
        </ul>
      </section>

      <section>
        <h2>8. Data Retention</h2>
        <p>We retain your personal data only as long as necessary for the purposes described in this policy:</p>
        <ul>
          <li><strong>Active accounts:</strong> for the duration of your subscription.</li>
          <li><strong>Inactive accounts:</strong> up to 24 months after your last login, after which data is deleted or anonymized, unless legal obligations require longer retention.</li>
          <li><strong>Financial records:</strong> up to 7 years as required by tax and accounting regulations.</li>
          <li><strong>Audit and compliance logs:</strong> up to 5 years for security and regulatory compliance.</li>
          <li><strong>Consent records:</strong> for the duration of your account plus the statutory retention period.</li>
          <li><strong>Marketing data:</strong> deleted within 30 days of opt-out.</li>
        </ul>
      </section>

      <section>
        <h2>9. Your Privacy Rights</h2>
        <p>Depending on your jurisdiction, you may have the following rights:</p>
        <ul>
          <li><strong>Access:</strong> request a copy of your personal data.</li>
          <li><strong>Rectification:</strong> request correction of inaccurate or incomplete data.</li>
          <li><strong>Erasure:</strong> request deletion of your personal data (&quot;right to be forgotten&quot;).</li>
          <li><strong>Restriction:</strong> request limitation of processing under certain circumstances.</li>
          <li><strong>Portability:</strong> receive your data in a structured, machine-readable format.</li>
          <li><strong>Objection:</strong> object to processing based on legitimate interest or for direct marketing.</li>
          <li><strong>Withdraw consent:</strong> withdraw any consent you previously gave at any time.</li>
          <li><strong>Lodge a complaint:</strong> with the relevant data protection authority.</li>
        </ul>
        <p>To exercise any of these rights, contact <strong>support@predictatrade.com</strong>. We will respond within 30 days. Identity verification may be required before processing your request.</p>
      </section>

      <section>
        <h2>10. Cookies and Tracking Technologies</h2>
        <p>We use the following types of cookies:</p>
        <ul>
          <li><strong>Essential cookies:</strong> for authentication, session management, and security (HttpOnly, Secure). These are required and cannot be disabled.</li>
          <li><strong>Preference cookies:</strong> for theme selection (light/dark), accessibility settings, and display preferences.</li>
          <li><strong>Analytics cookies:</strong> for understanding usage patterns — only loaded with your consent via the cookie consent banner.</li>
          <li><strong>Marketing cookies:</strong> for measuring marketing campaign effectiveness — only loaded with your explicit opt-in consent.</li>
        </ul>
        <p>You can manage your cookie preferences at any time through the cookie consent banner or your browser settings. Disabling essential cookies will prevent you from using the Service.</p>
      </section>

      <section>
        <h2>11. Children&apos;s Privacy</h2>
        <p>The Service is not intended for individuals under 18 years of age. We do not knowingly collect personal data from children. If you believe we have collected data from a minor, please contact us immediately and we will delete it.</p>
      </section>

      <section>
        <h2>12. Marketing Communications and Consent</h2>
        <p>We offer three separate marketing communication channels, each with an independent opt-in:</p>
        <ul>
          <li><strong>Email:</strong> newsletters, product updates, and promotional offers via email.</li>
          <li><strong>SMS:</strong> short text messages for time-sensitive offers and notifications.</li>
          <li><strong>Phone call:</strong> telephone contact for promotional purposes.</li>
        </ul>
        <p>Each channel can be opted into or out of independently. You may withdraw marketing consent at any time via:</p>
        <ul>
          <li>The unsubscribe link in any marketing email.</li>
          <li>Your account settings page.</li>
          <li>By contacting support@predictatrade.com.</li>
        </ul>
        <p>Opt-outs are honored immediately and persisted. We do not send marketing communications to users who have not opted in. Registration for the Service does not require marketing opt-in — you may register with all marketing options unchecked.</p>
      </section>

      <section>
        <h2>13. Security Measures</h2>
        <p>We implement industry-standard technical and organizational security measures, including:</p>
        <ul>
          <li>JWT-based authentication with refresh token rotation and HttpOnly cookies.</li>
          <li>bcrypt password hashing (12 rounds — passwords are never stored in plaintext).</li>
          <li>HTTPS/TLS 1.2+ encryption for all data in transit.</li>
          <li>Role-Based Access Control (RBAC) with least-privilege principles.</li>
          <li>Audit logging of all security-relevant events.</li>
          <li>Rate limiting and DDoS protection.</li>
          <li>Regular security reviews and dependency vulnerability scanning.</li>
          <li>Encrypted database storage with column-level encryption for sensitive fields.</li>
          <li>Secure key management with separation of duties.</li>
        </ul>
        <p>For detailed technical and organizational security measures, please refer to our <a href="/data-processing-agreement">Data Processing and Security Agreement</a>.</p>
        <p>Despite our best efforts, no system is perfectly secure. In the event of a personal data breach, we will notify affected users and relevant authorities as required by applicable law, generally within 72 hours of becoming aware of the breach.</p>
      </section>

      <section>
        <h2>14. UAE PDPL Compliance</h2>
        <p>We comply with the UAE Personal Data Protection Law (Federal Decree-Law No. 45 of 2021). When you register, we collect explicit, separate, and informed consents for: (1) acceptance of Terms &amp; Privacy Policy (required for account creation), (2) acknowledgment of the Data Processing and Security Agreement (required), and (3) marketing opt-in preferences (optional). Each consent is logged with a timestamp, the exact consent text and version shown, the state of each checkbox, and your IP address and user-agent, for compliance and audit purposes.</p>
      </section>

      <section>
        <h2>15. Changes to This Privacy Policy</h2>
        <p>We may update this Privacy Policy periodically. Material changes will be notified to registered users via email or in-platform notification at least 30 days before taking effect. We encourage you to review this page periodically.</p>
      </section>

      <section>
        <h2>16. Contact</h2>
        <p>For privacy questions, data subject requests, or to exercise your privacy rights: <strong>support@predictatrade.com</strong></p>
      </section>
    </LegalLayout>
  );
}
