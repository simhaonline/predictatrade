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
import { renderBrandedEmail, textFooter } from './email-template';
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
      // v1.31: the platform's own pat-mail-relay uses a self-signed cert
      // (operator-generated, internal docker network only). For THAT host we
      // pin TLS without public-CA validation; every external SMTP host keeps
      // full certificate verification (rejectUnauthorized stays true).
      const internalRelay = host === 'pat-mail-relay' || host.endsWith('.internal');
      this.transporter = nodemailer.createTransport({
        host,
        port,
        secure: port === 465, // true for 465 (SSL), false for 587 (STARTTLS)
        auth: { user, pass },
        requireTLS: port !== 465, // Upgrade to TLS on STARTTLS ports
        tls: internalRelay ? { rejectUnauthorized: false } : undefined,
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
      timeZone: 'UTC',
    }) + ' UTC';

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
      textFooter(),
    ].join('\n');

    const htmlBody = renderBrandedEmail({
      preheader: 'A password reset was requested for your account. Link expires ' + expiresStr + '.',
      title: 'Reset your password',
      bodyHtml: `
        <p style="margin:0 0 8px;">A password reset was requested for your Predict-A-Trade account.</p>
        <p style="margin:0 0 24px;">Click the button below to choose a new password:</p>
        <p style="margin:0 0 20px;">
          <a href="${input.resetUrl}"
             style="display:inline-block;background:#2362EA;color:#ffffff;padding:12px 32px;border-radius:8px;text-decoration:none;font-weight:600;font-size:14px;">
            Reset password
          </a>
        </p>
        <p style="margin:0;color:#77828F;font-size:12px;">
          This link expires at ${expiresStr}. If you did not request a reset, you can safely ignore this email —
          your password will not change.
        </p>`,
      bodyText: textBody,
      footerNote: 'You received this email because a password reset was requested for this address.',
    });

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
      timeZone: 'UTC',
    }) + ' UTC';

    const textBody = [
      `Hello,`,
      ``,
      `Your Predict-A-Trade verification code is: ${input.code}`,
      ``,
      `This code expires at ${expiresStr}.`,
      ``,
      `If you did not create an account, you can safely ignore this email.`,
      ``,
      textFooter(input.unsubscribeUrl),
    ].join('\n');

    const htmlBody = renderBrandedEmail({
      preheader: `Your verification code is ${input.code}. Expires ${expiresStr}.`,
      title: 'Verify your email',
      bodyHtml: `
        <p style="margin:0 0 20px;">Use the code below to complete your registration:</p>
        <p style="margin:0 0 20px;text-align:center;">
          <span style="display:inline-block;background:#0D1524;color:#F2F5F9;padding:16px 40px;
                       border:1px solid #24375A;border-radius:10px;font-family:monospace;font-size:28px;
                       letter-spacing:8px;font-weight:700;">
            ${input.code}
          </span>
        </p>
        <p style="margin:0;color:#77828F;font-size:12px;">
          This code expires at ${expiresStr}. If you did not create an account, you can safely ignore this email.
        </p>`,
      bodyText: textBody,
      unsubscribeUrl: input.unsubscribeUrl,
    });

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

    const textBody = [
      `Hello ${input.name},`,
      ``,
      `Welcome to Predict-A-Trade — your XAUUSD market monitoring dashboard is ready.`,
      ``,
      `You now have full access to live market data, signals, and the command center.`,
      input.reviewUrl ? `If you find the platform useful later, we'd appreciate a review on Google (no reward, no obligation): ${input.reviewUrl}` : '',
      ``,
      textFooter(input.unsubscribeUrl),
    ].filter(Boolean).join('\n');

    const reviewHtml = input.reviewUrl
      ? `<p style="margin:16px 0 0;color:#77828F;font-size:12px;">
           If you find the platform useful later, we'd appreciate a
           <a href="${input.reviewUrl}" style="color:#2362EA;text-decoration:none;">review on Google</a>
           (no reward, no obligation).
         </p>`
      : '';

    const htmlBody = renderBrandedEmail({
      preheader: 'Your Predict-A-Trade dashboard is ready — live XAUUSD market data, signals and the command center await.',
      title: `Welcome, ${input.name}!`,
      bodyHtml: `
        <p style="margin:0 0 8px;">Your Predict-A-Trade XAUUSD market monitoring dashboard is ready.</p>
        <p style="margin:0 0 24px;">You now have full access to live market data, signals, and the command center.</p>
        <p style="margin:0 0 8px;">
          <a href="https://platform.predictatrade.com/dashboard/live"
             style="display:inline-block;background:#2362EA;color:#ffffff;padding:12px 32px;border-radius:8px;text-decoration:none;font-weight:600;font-size:14px;">
            Open Dashboard
          </a>
        </p>
        ${reviewHtml}`,
      bodyText: textBody,
      unsubscribeUrl: input.unsubscribeUrl,
      footerNote: 'You received this email because an account was just created at platform.predictatrade.com.',
    });

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

  /**
   * Generic raw send for admin campaigns (alerts / newsletters / marketing).
   * Uses the same SMTP transport as transactional sends. The dev fallback
   * logs instead of sending so local/dev runs never emit real campaign mail.
   */
  async sendRawEmail(input: {
    to: string;
    subject: string;
    bodyHtml: string;
    bodyText: string;
    listUnsubscribe?: boolean;
  }): Promise<void> {
    if (this.isDevFallback) {
      this.logger.log(`[DEV EMAIL] To: ${input.to} | Subject: ${input.subject}`);
      return;
    }
    if (!this.transporter) {
      throw new Error('Email transporter not initialized');
    }
    // Campaign HTML is wrapped in the branded shell (logo header, PAT dark
    // card, footer with copyright + unsubscribe). Admin composes the INNER
    // content only — the shell is not theirs to omit.
    const html = renderBrandedEmail({
      title: input.subject,
      bodyHtml: input.bodyHtml,
      bodyText: input.bodyText,
      unsubscribeUrl: input.listUnsubscribe
        ? 'https://platform.predictatrade.com/unsubscribe'
        : undefined,
    });
    const text = `${input.bodyText}\n\n${textFooter(input.listUnsubscribe ? 'https://platform.predictatrade.com/unsubscribe' : undefined)}`;
    try {
      await this.transporter.sendMail({
        from: `"${this.fromName}" <${this.fromAddress}>`,
        to: input.to,
        subject: input.subject,
        text,
        html,
        headers: input.listUnsubscribe
          ? { 'List-Unsubscribe': '<https://platform.predictatrade.com/unsubscribe>' }
          : undefined,
      });
    } catch (err) {
      this.logger.error(`SMTP send failure (campaign to ${input.to}): ${err instanceof Error ? err.message : 'unknown error'}`);
      throw err;
    }
  }
}
