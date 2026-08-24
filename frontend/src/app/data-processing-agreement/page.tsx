import type { Metadata } from "next";
import LegalLayout from "@/components/legal/legal-layout";

export const metadata: Metadata = {
  title: "Data Processing and Security Agreement | Predict-A-Trade",
  description: "Data Processing and Security Agreement (DPA) for Simha FinTech Predict-A-Trade platform",
};

export default function DataProcessingPage() {
  return (
    <LegalLayout title="Data Processing and Security Agreement" lastUpdated="25 August 2026">
      <section>
        <h2>1. Purpose and Scope</h2>
        <p>This Data Processing and Security Agreement (&quot;DPA&quot; or &quot;Agreement&quot;) forms part of and is incorporated into the Predict-A-Trade Terms of Service and Privacy Policy. It describes the technical and organizational measures that Simha FinTech (&quot;Data Controller&quot; or &quot;Processor&quot;, as applicable) implements to protect the confidentiality, integrity, and availability of personal data processed through the Predict-A-Trade platform (the &quot;Service&quot;).</p>
        <p>By acknowledging this Agreement during registration, you confirm that you have read and understood the data processing and security practices described herein.</p>
      </section>

      <section>
        <h2>2. Roles and Responsibilities</h2>
        <p>Simha FinTech acts as the <strong>Data Controller</strong> for personal data collected directly from users of the Service (account data, subscription data, marketing preferences, etc.).</p>
        <p>Simha FinTech may act as a <strong>Data Processor</strong> on behalf of business partners who integrate with the Service, in which case a separate controller-processor agreement governs that relationship.</p>
        <p>Third-party service providers (payment processors, cloud hosting, email delivery) act as <strong>Sub-processors</strong> and are bound by written data processing agreements.</p>
      </section>

      <section>
        <h2>3. Categories of Personal Data Processed</h2>
        <ul>
          <li><strong>Identification data:</strong> email address, display name, user ID.</li>
          <li><strong>Authentication data:</strong> hashed passwords, JWT tokens, refresh tokens, MFA secrets, session identifiers.</li>
          <li><strong>Financial data:</strong> subscription plan, billing records, commission earnings, payout details (tokenized payment identifiers — no full card numbers stored).</li>
          <li><strong>Device data:</strong> device fingerprints, device names, IP addresses, user-agent strings.</li>
          <li><strong>Usage data:</strong> feature usage, page views, session duration, interaction events.</li>
          <li><strong>Consent records:</strong> consent timestamps, consent text versions, checkbox states, IP and user-agent at consent time.</li>
          <li><strong>Marketing data:</strong> opt-in/opt-out preferences for email, SMS, and phone marketing.</li>
        </ul>
      </section>

      <section>
        <h2>4. Data Processing Purposes</h2>
        <p>Personal data is processed exclusively for the following purposes:</p>
        <ul>
          <li>User authentication, authorization, and session management.</li>
          <li>Subscription billing, invoicing, and payment processing.</li>
          <li>Service delivery, including signal generation, market analysis, and real-time data visualization.</li>
          <li>Device licensing and authorization management.</li>
          <li>Referral program management, commission calculation, and payout processing.</li>
          <li>Marketing communications (only with explicit opt-in consent).</li>
          <li>Fraud prevention, security monitoring, and incident response.</li>
          <li>Legal compliance, audit, and regulatory reporting.</li>
          <li>Service improvement, analytics, and quality assurance.</li>
        </ul>
      </section>

      <section>
        <h2>5. Technical Security Measures</h2>
        <h3>5.1 Encryption</h3>
        <ul>
          <li>All data in transit is encrypted using TLS 1.2+ (HTTPS/WSS). TLS 1.0 and 1.1 are disabled.</li>
          <li>All data at rest is encrypted using database-level encryption (PostgreSQL TDE / volume encryption).</li>
          <li>Authentication tokens (JWT access tokens) are short-lived (15-minute expiry) and signed with RS256.</li>
          <li>Refresh tokens are HttpOnly, Secure, SameSite=Lax cookies with rotation on each use.</li>
          <li>Passwords are hashed using bcrypt with a cost factor of 12. No plaintext passwords are ever stored.</li>
          <li>API keys and signing keys are stored in a dedicated secrets management system with encryption at rest and access-controlled decryption.</li>
          <li>Column-level encryption is applied to sensitive database fields (e.g., MFA secrets, payment token references).</li>
        </ul>

        <h3>5.2 Access Control</h3>
        <ul>
          <li>Role-Based Access Control (RBAC) with least-privilege principle — users and service accounts have only the permissions necessary for their function.</li>
          <li>Multi-Factor Authentication (MFA) required for all administrative accounts.</li>
          <li>Database access uses least-privilege roles; no application or service account has database superuser privileges.</li>
          <li>Internal access is logged and auditable; all privileged operations are recorded in audit logs.</li>
          <li>Secret management uses separated keys: signing keys, API keys, and database credentials are never shared across environments.</li>
          <li>Key rotation is performed on a defined schedule, with emergency rotation procedures for compromise scenarios.</li>
        </ul>

        <h3>5.3 Network Security</h3>
        <ul>
          <li>Reverse proxy (nginx) with SSL/TLS termination and security headers (HSTS, X-Frame-Options, X-Content-Type-Options, CSP).</li>
          <li>Rate limiting on all public endpoints (per-IP and per-user throttling using token bucket algorithms).</li>
          <li>Web Application Firewall (WAF) rules for common attack patterns (SQL injection, XSS, path traversal).</li>
          <li>All inter-service communication within the Docker network is isolated; no service exposes unnecessary ports externally.</li>
          <li>Database, cache (Valkey/Redis), and internal services are not directly accessible from the public internet.</li>
        </ul>

        <h3>5.4 Application Security</h3>
        <ul>
          <li>Input validation and sanitization on all API endpoints using class-validator and parameterized queries (no string concatenation in SQL).</li>
          <li>Output encoding to prevent XSS attacks.</li>
          <li>CSRF protection via SameSite=Lax cookies and Origin/Referer header validation on state-changing endpoints.</li>
          <li>Idempotency keys required for mutation and execution endpoints to prevent replay attacks.</li>
          <li>Stable, machine-readable error responses with correlation IDs for all requests.</li>
          <li>Dependency vulnerability scanning (Snyk/npm-audit) integrated into CI/CD pipeline.</li>
          <li>Static Application Security Testing (SAST) on every code change.</li>
          <li>Software Bill of Materials (SBOM) generated for each release.</li>
        </ul>
      </section>

      <section>
        <h2>6. Organizational Security Measures</h2>
        <ul>
          <li><strong>Access management:</strong> Access to production systems is restricted to authorized personnel only, granted on a need-to-know basis, and reviewed quarterly.</li>
          <li><strong>Personnel training:</strong> All personnel with access to personal data receive data protection and security awareness training.</li>
          <li><strong>Confidentiality:</strong> All personnel are bound by confidentiality agreements.</li>
          <li><strong>Incident response:</strong> A documented incident response plan defines roles, procedures, and notification timelines for security incidents.</li>
          <li><strong>Data minimization:</strong> We collect only the personal data necessary for the stated purposes.</li>
          <li><strong>Purpose limitation:</strong> Personal data is used only for the purposes described in this Agreement and our Privacy Policy.</li>
          <li><strong>Separation of duties:</strong> No single individual has unrestricted access to all systems or the ability to perform all critical operations alone.</li>
        </ul>
      </section>

      <section>
        <h2>7. Data Storage and Infrastructure</h2>
        <ul>
          <li><strong>Primary database:</strong> PostgreSQL with TimescaleDB extension for time-series market data. Data is encrypted at rest.</li>
          <li><strong>Cache layer:</strong> Valkey (Redis-compatible) for hot/session state — used only for non-durable session and cache data, never as the sole source of financial or trading truth.</li>
          <li><strong>Containerized deployment:</strong> All services run in Docker containers with isolated networks. No systemd services are used.</li>
          <li><strong>Backups:</strong> Automated daily backups with point-in-time recovery capability. Backup encryption and tested restore procedures.</li>
          <li><strong>Geographic location:</strong> Data is stored in the cloud region selected for the Service deployment. Cross-region replication is encrypted.</li>
        </ul>
      </section>

      <section>
        <h2>8. Sub-Processors</h2>
        <p>The following categories of sub-processors may process personal data on our behalf:</p>
        <ul>
          <li><strong>Cloud infrastructure:</strong> container hosting, database hosting, and CDN services.</li>
          <li><strong>Payment processing:</strong> subscription payment processing and commission payout processing.</li>
          <li><strong>Email delivery:</strong> transactional and marketing email delivery.</li>
          <li><strong>Market data:</strong> real-time and historical market data feeds (receive only API credentials, not user personal data).</li>
          <li><strong>Monitoring and observability:</strong> application performance monitoring and error tracking.</li>
        </ul>
        <p>Each sub-processor is bound by a data processing agreement that requires equivalent security standards. We conduct due diligence before engaging any new sub-processor and notify users of material changes to the sub-processor list.</p>
      </section>

      <section>
        <h2>9. Personal Data Breach Notification</h2>
        <p>In the event of a personal data breach that is likely to result in a risk to your rights and freedoms, we will:</p>
        <ul>
          <li>Notify the relevant data protection authority within 72 hours of becoming aware of the breach, as required by applicable law.</li>
          <li>Notify affected users without undue delay, describing the nature of the breach, likely consequences, and measures taken.</li>
          <li>Document the breach, its effects, and the remedial actions taken.</li>
          <li>Conduct a post-incident review to prevent recurrence.</li>
        </ul>
      </section>

      <section>
        <h2>10. Data Subject Rights Support</h2>
        <p>We provide mechanisms for data subjects to exercise their rights under applicable data protection laws, including access, rectification, erasure, restriction, portability, and objection. Requests can be submitted to <strong>support@predictatrade.com</strong> and will be processed within 30 days, with identity verification as required.</p>
      </section>

      <section>
        <h2>11. Data Return and Deletion</h2>
        <p>Upon account termination or upon request, we will delete or anonymize your personal data in accordance with our retention schedule, except where retention is required by law. Data that must be retained will be isolated and access-restricted until the retention period expires, after which it is permanently deleted.</p>
      </section>

      <section>
        <h2>12. Audit and Compliance</h2>
        <p>We maintain audit logs of all security-relevant events, including authentication, authorization changes, data access, and administrative actions. Logs are tamper-evident and retained for a minimum of 5 years for compliance and forensic purposes.</p>
        <p>We undergo periodic security assessments and may engage independent third parties to conduct penetration testing and security audits.</p>
      </section>

      <section>
        <h2>13. Changes to This Agreement</h2>
        <p>We may update this Data Processing and Security Agreement to reflect changes in our practices, technology, or legal requirements. Material changes will be notified to registered users at least 30 days before taking effect.</p>
      </section>

      <section>
        <h2>14. Contact</h2>
        <p>For questions about data processing and security: <strong>security@predictatrade.com</strong></p>
        <p>For data subject requests: <strong>support@predictatrade.com</strong></p>
      </section>
    </LegalLayout>
  );
}
