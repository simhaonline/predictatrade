/**
 * EmailService — abstract interface for sending transactional emails.
 *
 * The auth/guest-preview services depend on this interface, not on any specific
 * vendor. The actual provider implementation (SMTP/Resend/SES/SendGrid, or a
 * console fallback for local dev) is injected via the MailModule based on the
 * EMAIL_PROVIDER environment variable. Credentials are NEVER hardcoded — they
 * come from environment variables.
 *
 * Unsubscribe checks (marketing_unsubscribes table) are performed by the
 * calling service, which owns the database pool — not by the email provider.
 */

export interface PasswordResetEmailInput {
  to: string;
  resetUrl: string;
  expiresAt: Date;
}

export interface OtpEmailInput {
  to: string;
  code: string;
  expiresAt: Date;
  /** Unsubscribe URL for marketing communications (always included). */
  unsubscribeUrl: string;
}

export interface WelcomeEmailInput {
  to: string;
  name: string;
  /** Unsubscribe URL for marketing communications (always included). */
  unsubscribeUrl: string;
  /** Optional Google review URL for a soft, ungated, non-incentivized ask. */
  reviewUrl?: string;
}

export const EMAIL_SERVICE = Symbol('EMAIL_SERVICE');

export interface EmailService {
  sendPasswordResetEmail(input: PasswordResetEmailInput): Promise<void>;
  sendOtpEmail(input: OtpEmailInput): Promise<void>;
  sendWelcomeEmail(input: WelcomeEmailInput): Promise<void>;
}
