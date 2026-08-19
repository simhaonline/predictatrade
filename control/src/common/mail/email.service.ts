/**
 * EmailService — abstract interface for sending transactional emails.
 *
 * The auth service depends on this interface, not on any specific vendor.
 * The actual provider implementation (SMTP, SendGrid, SES, etc.) is injected
 * via the MailModule based on the EMAIL_PROVIDER environment variable.
 */

export interface PasswordResetEmailInput {
  to: string;
  resetUrl: string;
  expiresAt: Date;
}

export const EMAIL_SERVICE = Symbol('EMAIL_SERVICE');

export interface EmailService {
  sendPasswordResetEmail(input: PasswordResetEmailInput): Promise<void>;
}
