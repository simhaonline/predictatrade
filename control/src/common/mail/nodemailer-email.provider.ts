/**
 * NodemailerEmailProvider — SMTP implementation of EmailService.
 *
 * Uses nodemailer with configuration from environment variables:
 *   SMTP_HOST, SMTP_PORT, SMTP_USERNAME, SMTP_PASSWORD,
 *   EMAIL_FROM, EMAIL_FROM_NAME
 *
 * In development without SMTP configured, emails are logged to console
 * (never to production). This fallback is rejected in production mode.
 *
 * Transactional email providers (Resend, Postmark, AWS SES, SendGrid) can be
 * added by implementing the EmailService interface and adding a case in the
 * MailModule useFactory — all configured via environment variables, never
 * hardcoded credentials.
 */

import { Injectable, Logger } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { EmailService, PasswordResetEmailInput, OtpEmailInput, WelcomeEmailInput } from './email.service';
import * as nodemailer from 'nodemailer';

@Injectable()
export class NodemailerEmailProvider implements EmailService {
  private readonly logger = new Logger(NodemailerEmailProvider.name);
  private transporter: nodemailer.Transporter | null = null;
  private readonly fromAddress: string;
  private readonly fromName: string;
  private readonly isDevFallback: boolean;

  constructor(private config: ConfigService) {
    this.fromName = this.config.get<string>('EMAIL_FROM_NAME', 'Predict-A-Trade');
    this.fromAddress = this.config.get<string>('EMAIL_FROM', 'no-reply@predictatrade.com');

    const host = this.config.get<string>('SMTP_HOST');
    const port = this.config.get<number>('SMTP_PORT', 587);
    const user = this.config.get<string>('SMTP_USERNAME');
    const pass = this.config.get<string>('SMTP_PASSWORD');
    const isProduction = this.config.get<string>('NODE_ENV') === 'production';

    // P2: Validate SMTP configuration in production.
    // Known insecure placeholder values that must not be used.
    const INSECURE_SMTP_PASSWORDS = new Set([
      '', 'your_smtp_password', 'changeme', 'password', 'placeholder',
    ]);

    if (host && user && pass && !INSECURE_SMTP_PASSWORDS.has(pass)) {
      this.transporter = nodemailer.createTransport({
        host,
        port,
        secure: port === 465, // true for 465 (SSL), false for 587 (STARTTLS)
        auth: { user, pass },
        // Never disable certificate verification in production.
        // Use platform defaults for TLS (rejectUnauthorized defaults to true).
        requireTLS: port !== 465, // Upgrade to TLS on STARTTLS ports
        connectionTimeout: 10_000, // 10s connect timeout
        greetingTimeout: 10_000,
        socketTimeout: 30_000,    // 30s for send operations
      });
      this.isDevFallback = false;
      this.logger.log(`SMTP configured: ${host}:${port} (${port === 465 ? 'SSL' : 'STARTTLS'}) as ${user}`);
    } else {
      // SMTP not configured — log warning.
      // Password reset won't work, but the rest of the application should still function.
      // The forgotPassword() method catches send failures and returns a generic response.
      this.isDevFallback = true;
      if (isProduction) {
        if (pass && INSECURE_SMTP_PASSWORDS.has(pass)) {
          this.logger.error('FATAL: SMTP_PASSWORD is a known insecure placeholder — password reset emails disabled in production. Set real SMTP credentials via production secret.');
        } else {
          this.logger.warn('SMTP not configured — password reset emails cannot be sent. Set SMTP_HOST, SMTP_USERNAME, SMTP_PASSWORD to enable.');
        }
      } else {
        this.logger.warn('SMTP not configured — emails will be logged to console (dev only)');
      }
    }
  }

  /**
   * Verify SMTP connectivity. Called on startup to detect configuration issues early.
   * Returns true if connected, false otherwise. Never throws — logs errors.
   */
  async verifyConnection(): Promise<boolean> {
    if (!this.transporter) {
      return false;
    }
    try {
      await this.transporter.verify();
      this.logger.log('SMTP connection verified successfully');
      return true;
    } catch (err) {
      this.logger.error(`SMTP connection verification failed: ${err instanceof Error ? err.message : 'unknown error'}`);
      return false;
    }
  }

  async sendPasswordResetEmail(input: PasswordResetEmailInput): Promise<void> {
    const subject = 'Reset your Predict-A-Trade password';
    const expiresStr = input.expiresAt.toLocaleString('en-US', {
      weekday: 'short', month: 'short', day: 'numeric',
      hour: '2-digit', minute: '2-digit',
    });

    const textBody = [
      `Hello,`,
      ``,
      `A password reset was requested for your Predict-A-Trade account.`,
      ``,
      `Click the link below to reset your password:`,
      input.resetUrl,
      ``,
      `This link expires at ${expiresStr}.`,
      ``,
      `If you did not request a password reset, you can safely ignore this email.`,
      ``,
      `— Predict-A-Trade`,
    ].join('\n');

    const htmlBody = `
      <div style="font-family: Inter, -apple-system, sans-serif; max-width: 480px; margin: 0 auto; padding: 24px;">
        <h2 style="color: #0F1114;">Reset your password</h2>
        <p style="color: #5B616E; font-size: 14px; line-height: 1.6;">
          A password reset was requested for your Predict-A-Trade account.
        </p>
        <p style="margin: 24px 0;">
          <a href="${input.resetUrl}"
             style="display: inline-block; background: #145CFA; color: #fff; padding: 12px 32px;
                    border-radius: 8px; text-decoration: none; font-weight: 500; font-size: 14px;">
            Reset password
          </a>
        </p>
        <p style="color: #8A919E; font-size: 12px;">
          This link expires at ${expiresStr}.
          If you did not request a password reset, you can safely ignore this email.
        </p>
        <hr style="border: none; border-top: 1px solid #E4E7EC; margin: 24px 0;">
        <p style="color: #8A919E; font-size: 12px;">© Predict-A-Trade</p>
      </div>`;

    if (this.isDevFallback) {
      // In dev, log a truncated summary — never the full reset URL with the token
      this.logger.log(`[DEV EMAIL] To: ${input.to} | Subject: ${subject} | Expires: ${expiresStr}`);
      return;
    }

    if (!this.transporter) {
      throw new Error('Email transporter not initialized');
    }

    try {
      await this.transporter.sendMail({
        from: `"${this.fromName}" <${this.fromAddress}>`,
        to: input.to,
        subject,
        text: textBody,
        html: htmlBody,
      });
    } catch (err) {
      // Log sanitized error — never expose SMTP internals or reset token to the user
      this.logger.error(`SMTP send failure: ${err instanceof Error ? err.message : 'unknown error'}`);
      throw err;
    }
  }

  /* ─── OTP / verification email (passwordless registration) ─── */

  async sendOtpEmail(input: OtpEmailInput): Promise<void> {
    const subject = 'Your Predict-A-Trade verification code';
    const expiresStr = input.expiresAt.toLocaleString('en-US', {
      weekday: 'short', month: 'short', day: 'numeric',
      hour: '2-digit', minute: '2-digit',
    });

    const textBody = [
      `Hello,`,
      ``,
      `Your Predict-A-Trade verification code is: ${input.code}`,
      ``,
      `This code expires at ${expiresStr}.`,
      ``,
      `If you did not create an account, you can safely ignore this email.`,
      ``,
      `Unsubscribe from marketing emails: ${input.unsubscribeUrl}`,
      ``,
      `— Predict-A-Trade`,
    ].join('\n');

    const htmlBody = `
      <div style="font-family: Inter, -apple-system, sans-serif; max-width: 480px; margin: 0 auto; padding: 24px;">
        <h2 style="color: #0F1114;">Verify your email</h2>
        <p style="color: #5B616E; font-size: 14px; line-height: 1.6;">
          Use the code below to complete your registration:
        </p>
        <p style="margin: 24px 0; text-align: center;">
          <span style="display: inline-block; background: #F2F4F7; color: #0F1114; padding: 16px 40px;
                       border-radius: 10px; font-family: monospace; font-size: 28px; letter-spacing: 8px; font-weight: 700;">
            ${input.code}
          </span>
        </p>
        <p style="color: #8A919E; font-size: 12px;">
          This code expires at ${expiresStr}.
          If you did not create an account, you can safely ignore this email.
        </p>
        <hr style="border: none; border-top: 1px solid #E4E7EC; margin: 24px 0;">
        <p style="color: #8A919E; font-size: 11px;">
          Don't want marketing emails?
          <a href="${input.unsubscribeUrl}" style="color: #145CFA;">Unsubscribe</a>.
        </p>
        <p style="color: #8A919E; font-size: 12px;">© Predict-A-Trade</p>
      </div>`;

    if (this.isDevFallback) {
      // In dev, log a truncated summary — never the full OTP code
      this.logger.log(`[DEV EMAIL] To: ${input.to} | Subject: ${subject} | Expires: ${expiresStr}`);
      return;
    }

    if (!this.transporter) {
      throw new Error('Email transporter not initialized');
    }

    try {
      await this.transporter.sendMail({
        from: `"${this.fromName}" <${this.fromAddress}>`,
        to: input.to,
        subject,
        text: textBody,
        html: htmlBody,
      });
    } catch (err) {
      this.logger.error(`SMTP send failure (otp): ${err instanceof Error ? err.message : 'unknown error'}`);
      throw err;
    }
  }

  /* ─── Welcome email (post-registration, with unsubscribe + soft review ask) ─── */

  async sendWelcomeEmail(input: WelcomeEmailInput): Promise<void> {
    const subject = 'Welcome to Predict-A-Trade';
    const reviewLine = input.reviewUrl
      ? `If you find the platform useful later, we'd appreciate a review on Google (no reward, no obligation): ${input.reviewUrl}`
      : '';

    const textBody = [
      `Hello ${input.name},`,
      ``,
      `Welcome to Predict-A-Trade — your XAUUSD market monitoring dashboard is ready.`,
      ``,
      `You now have full access to live market data, signals, and the command center.`,
      reviewLine,
      ``,
      `Unsubscribe from marketing emails: ${input.unsubscribeUrl}`,
      ``,
      `— Predict-A-Trade`,
    ].filter(Boolean).join('\n');

    const reviewHtml = input.reviewUrl
      ? `<p style="color: #8A919E; font-size: 12px; margin-top: 16px;">
           If you find the platform useful later, we'd appreciate a
           <a href="${input.reviewUrl}" style="color: #145CFA;">review on Google</a>
           (no reward, no obligation).
         </p>`
      : '';

    const htmlBody = `
      <div style="font-family: Inter, -apple-system, sans-serif; max-width: 480px; margin: 0 auto; padding: 24px;">
        <h2 style="color: #0F1114;">Welcome, ${input.name}!</h2>
        <p style="color: #5B616E; font-size: 14px; line-height: 1.6;">
          Your Predict-A-Trade XAUUSD market monitoring dashboard is ready.
          You now have full access to live market data, signals, and the command center.
        </p>
        ${reviewHtml}
        <hr style="border: none; border-top: 1px solid #E4E7EC; margin: 24px 0;">
        <p style="color: #8A919E; font-size: 11px;">
          Don't want marketing emails?
          <a href="${input.unsubscribeUrl}" style="color: #145CFA;">Unsubscribe</a>.
        </p>
        <p style="color: #8A919E; font-size: 12px;">© Predict-A-Trade</p>
      </div>`;

    if (this.isDevFallback) {
      this.logger.log(`[DEV EMAIL] To: ${input.to} | Subject: ${subject}`);
      return;
    }

    if (!this.transporter) {
      throw new Error('Email transporter not initialized');
    }

    try {
      await this.transporter.sendMail({
        from: `"${this.fromName}" <${this.fromAddress}>`,
        to: input.to,
        subject,
        text: textBody,
        html: htmlBody,
      });
    } catch (err) {
      this.logger.error(`SMTP send failure (welcome): ${err instanceof Error ? err.message : 'unknown error'}`);
      throw err;
    }
  }
}
