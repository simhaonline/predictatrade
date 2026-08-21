import {
  Injectable, BadRequestException, UnauthorizedException, ConflictException,
  Logger, Inject,
} from '@nestjs/common';
import { JwtService } from '@nestjs/jwt';
import { ConfigService } from '@nestjs/config';
import { Pool } from 'pg';
import * as crypto from 'crypto';
import * as bcrypt from 'bcrypt';
import {
  GuestRegisterDto, GuestOtpVerifyDto, GuestOtpResendDto,
} from './dto/guest-preview.dto';
import { DB_POOL } from '../../common/database.module';
import { EMAIL_SERVICE } from '../../common/mail/email.service';
import type { EmailService } from '../../common/mail/email.service';

/* ─── Constants ─── */

const GUEST_JWT_PURPOSE = 'guest_preview';
const GUEST_COOKIE_NAME = 'pat_guest_session';
const OTP_EXPIRY_MIN = 10;
const OTP_MAX_ATTEMPTS = 5;
const RESEND_COOLDOWN_SEC = 60;
const REFRESH_TOKEN_BYTES = 48;
const REFRESH_TOKEN_EXPIRY_DAYS = 7;
const ACCESS_TOKEN_EXPIRY = '1h';
const CONSENT_VERSION = '1.0.0';

// Exact consent text shown to the user (immutable audit copy stored per signup).
const TERMS_TEXT =
  'I accept the Terms & Conditions and Privacy Policy.';
const RISK_TEXT =
  'I understand this platform is for informational/educational purposes only and is not investment advice. ' +
  'Trading CFDs/FX/gold (XAUUSD) carries a high risk of loss.';
const MARKETING_TEXT =
  'I agree to receive marketing communications and offers via email and WhatsApp.';

export interface GuestSessionResult {
  /** Opaque signed guest token (also set as an HttpOnly cookie by the controller). */
  guestToken: string;
  /** Absolute server-side expiry timestamp (ms since epoch). */
  expiresAt: number;
  /** Preview duration in seconds (from PREVIEW_SECONDS env, default 300). */
  previewSeconds: number;
}

export interface GuestStatusResult {
  /** True when the guest preview has expired (server-authoritative). */
  locked: boolean;
  /** Absolute server-side expiry timestamp (ms since epoch), null if no session. */
  expiresAt: number | null;
  /** Remaining seconds in the preview (0 when locked). */
  remainingSeconds: number;
}

export interface GuestVerifyResult {
  accessToken: string;
  user: { id: string; email: string; displayName: string };
  /** Raw refresh token — only used to set the HttpOnly cookie by the controller. */
  _refreshToken: string;
}

@Injectable()
export class GuestPreviewService {
  private readonly logger = new Logger(GuestPreviewService.name);

  constructor(
    private jwtService: JwtService,
    private config: ConfigService,
    @Inject(DB_POOL) private pool: Pool,
    @Inject(EMAIL_SERVICE) private emailService: EmailService,
  ) {}

  /* ─── Guest preview session ─── */

  /**
   * Issue a short-lived anonymous session token with a SERVER-SIDE expiry
   * timestamp. The token is a signed JWT (purpose=guest_preview) carrying an
   * `exp` claim; the client-side countdown is display-only. The server
   * validates the expiry on every /guest/status check, so clearing cookies or
   * using incognito cannot bypass the 5-minute lock.
   */
  async issueGuestSession(): Promise<GuestSessionResult> {
    const previewSeconds = this.getPreviewSeconds();
    const expiresAt = Date.now() + previewSeconds * 1000;
    const guestToken = this.jwtService.sign(
      { purpose: GUEST_JWT_PURPOSE, jti: crypto.randomUUID() },
      { expiresIn: `${previewSeconds}s` },
    );
    return { guestToken, expiresAt, previewSeconds };
  }

  /**
   * Check the server-authoritative status of a guest session.
   * Returns locked=true when the token is missing, invalid, or expired.
   */
  async getGuestStatus(rawToken: string | undefined): Promise<GuestStatusResult> {
    const previewSeconds = this.getPreviewSeconds();
    if (!rawToken) {
      return { locked: true, expiresAt: null, remainingSeconds: 0 };
    }
    try {
      const payload = this.jwtService.verify(rawToken) as { purpose?: string; exp?: number };
      if (payload.purpose !== GUEST_JWT_PURPOSE) {
        return { locked: true, expiresAt: null, remainingSeconds: 0 };
      }
      const expiresAt = (payload.exp ?? 0) * 1000;
      const remainingSeconds = Math.max(0, Math.floor((expiresAt - Date.now()) / 1000));
      return {
        locked: remainingSeconds <= 0,
        expiresAt,
        remainingSeconds,
      };
    } catch {
      // Expired or invalid signature → locked. A fresh session can be issued
      // but it will again expire after PREVIEW_SECONDS.
      return { locked: true, expiresAt: null, remainingSeconds: 0 };
    }
  }

  /* ─── Registration (passwordless, email-OTP) ─── */

  /**
   * Validate the registration payload, enforce the two REQUIRED consents,
   * store a hashed OTP challenge with the frozen consent snapshot, and email
   * the 6-digit code. Returns a GENERIC message that never reveals whether
   * the email is already registered.
   */
  async register(
    dto: GuestRegisterDto,
    ip: string | undefined,
    userAgent: string | undefined,
  ): Promise<{ message: string; challengeId: string }> {
    // Enforce required consents server-side (client checkboxes default unchecked).
    if (!dto.termsAccepted) {
      throw new BadRequestException('You must accept the Terms & Conditions and Privacy Policy to continue.');
    }
    if (!dto.riskAcknowledged) {
      throw new BadRequestException('You must acknowledge the risk disclosure to continue.');
    }
    // marketingOptIn is OPTIONAL — no enforcement here.

    const email = dto.email.trim().toLowerCase();
    const fullName = dto.fullName.trim();

    // Check for an existing verified account. We do NOT throw a
    // ConflictException (that would reveal the email exists). Instead we still
    // return the generic message; no OTP is sent for an already-verified email.
    const existing = await this.pool.query(
      'SELECT id FROM iam.users WHERE email = $1 AND email_verified = true', [email],
    );
    if (existing.rows.length > 0) {
      // Generic response — do not reveal the account exists.
      this.logger.log(`Registration attempt for already-verified email (suppressed)`);
      return {
        message: 'If this email is not already registered, we sent a verification code. Enter it to complete registration.',
        challengeId: '',
      };
    }

    // Rate-limit: no more than one active challenge per email per RESEND_COOLDOWN.
    const recentResult = await this.pool.query(
      `SELECT id, created_at FROM iam.registration_challenges
       WHERE email = $1 AND consumed_at IS NULL AND created_at >= now() - interval '${RESEND_COOLDOWN_SEC} seconds'
       ORDER BY created_at DESC LIMIT 1`,
      [email],
    );
    if (recentResult.rows.length > 0) {
      throw new BadRequestException(`Please wait ${RESEND_COOLDOWN_SEC} seconds before requesting another code.`);
    }

    // Generate a 6-digit OTP and store only its SHA-256 hash.
    const code = this.generateOtpCode();
    const codeHash = this.hashToken(code);
    const expiresAt = new Date(Date.now() + OTP_EXPIRY_MIN * 60_000);

    const consentSnapshot = {
      termsAccepted: dto.termsAccepted,
      riskAcknowledged: dto.riskAcknowledged,
      marketingOptIn: dto.marketingOptIn,
      termsText: TERMS_TEXT,
      riskText: RISK_TEXT,
      marketingText: MARKETING_TEXT,
      consentVersion: CONSENT_VERSION,
    };

    const insertResult = await this.pool.query(
      `INSERT INTO iam.registration_challenges
         (email, code_hash, full_name, phone, broker, consent_snapshot, max_attempts, expires_at, ip_address, user_agent)
       VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
       RETURNING id`,
      [email, codeHash, fullName, dto.phone ?? null, dto.broker ?? null,
       JSON.stringify(consentSnapshot), OTP_MAX_ATTEMPTS, expiresAt,
       ip ?? null, userAgent ?? null],
    );
    const challengeId = insertResult.rows[0].id;

    // Email the OTP. Failure to send is logged but does NOT reveal to the user
    // that delivery failed (generic message is still returned).
    const unsubscribeUrl = this.buildUnsubscribeUrl(email);
    try {
      await this.emailService.sendOtpEmail({ to: email, code, expiresAt, unsubscribeUrl });
    } catch (err) {
      this.logger.error(`Failed to send OTP email: ${err instanceof Error ? err.message : 'unknown error'}`);
    }

    return {
      message: 'If this email is not already registered, we sent a verification code. Enter it to complete registration.',
      challengeId,
    };
  }

  /**
   * Resend the OTP for a pending registration, enforcing the 60-second cooldown
   * and rate limiting. Generic response — never reveals whether an email exists.
   */
  async resendOtp(
    dto: GuestOtpResendDto,
    ip: string | undefined,
    userAgent: string | undefined,
  ): Promise<{ message: string }> {
    const email = dto.email.trim().toLowerCase();

    // Cooldown: must have a challenge older than RESEND_COOLDOWN_SEC, or none at all.
    const recentResult = await this.pool.query(
      `SELECT id FROM iam.registration_challenges
       WHERE email = $1 AND consumed_at IS NULL AND created_at >= now() - interval '${RESEND_COOLDOWN_SEC} seconds'`,
      [email],
    );
    if (recentResult.rows.length > 0) {
      throw new BadRequestException(`Please wait ${RESEND_COOLDOWN_SEC} seconds before requesting another code.`);
    }

    // Find the most recent unconsumed, unexpired challenge to resend for.
    const challengeResult = await this.pool.query(
      `SELECT id, full_name, phone, broker, consent_snapshot FROM iam.registration_challenges
       WHERE email = $1 AND consumed_at IS NULL AND expires_at > now()
       ORDER BY created_at DESC LIMIT 1`,
      [email],
    );
    if (challengeResult.rows.length === 0) {
      // No pending challenge — generic response (do not reveal).
      return { message: 'If a pending registration exists for this email, a new code has been sent.' };
    }

    const challenge = challengeResult.rows[0];
    const code = this.generateOtpCode();
    const codeHash = this.hashToken(code);
    const expiresAt = new Date(Date.now() + OTP_EXPIRY_MIN * 60_000);

    // Create a new challenge row (supersedes older ones) with the same consent snapshot.
    await this.pool.query(
      `INSERT INTO iam.registration_challenges
         (email, code_hash, full_name, phone, broker, consent_snapshot, max_attempts, expires_at, ip_address, user_agent)
       VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
      [email, codeHash, challenge.full_name, challenge.phone, challenge.broker,
       JSON.stringify(challenge.consent_snapshot), OTP_MAX_ATTEMPTS, expiresAt,
       ip ?? null, userAgent ?? null],
    );

    const unsubscribeUrl = this.buildUnsubscribeUrl(email);
    try {
      await this.emailService.sendOtpEmail({ to: email, code, expiresAt, unsubscribeUrl });
    } catch (err) {
      this.logger.error(`Failed to resend OTP email: ${err instanceof Error ? err.message : 'unknown error'}`);
    }

    return { message: 'If a pending registration exists for this email, a new code has been sent.' };
  }

  /**
   * Verify the 6-digit OTP. On success: create the account (passwordless),
   * log the immutable consent record, send the welcome email, and create an
   * authenticated session. Enforces max-attempts and expiry.
   */
  async verifyOtp(
    dto: GuestOtpVerifyDto,
    ip: string | undefined,
    userAgent: string | undefined,
  ): Promise<GuestVerifyResult> {
    const email = dto.email.trim().toLowerCase();

    // Find the most recent unconsumed challenge for this email.
    const challengeResult = await this.pool.query(
      `SELECT id, code_hash, full_name, phone, broker, consent_snapshot, attempts, max_attempts, expires_at
       FROM iam.registration_challenges
       WHERE email = $1 AND consumed_at IS NULL
       ORDER BY created_at DESC LIMIT 1`,
      [email],
    );

    if (challengeResult.rows.length === 0) {
      // Generic error — do not reveal whether an email/challenge exists.
      throw new UnauthorizedException('Invalid or expired verification code.');
    }

    const challenge = challengeResult.rows[0];

    // Expiry check.
    if (new Date(challenge.expires_at) < new Date()) {
      throw new UnauthorizedException('Invalid or expired verification code.');
    }

    // Max-attempts check.
    const attempts = Number(challenge.attempts) || 0;
    if (attempts >= Number(challenge.max_attempts) || attempts >= OTP_MAX_ATTEMPTS) {
      // Consume the challenge to prevent further attempts.
      await this.pool.query(
        'UPDATE iam.registration_challenges SET consumed_at = now() WHERE id = $1', [challenge.id],
      );
      throw new UnauthorizedException('Too many failed attempts. Please request a new code.');
    }

    // Increment attempts BEFORE verifying (prevents brute-force racing).
    await this.pool.query(
      'UPDATE iam.registration_challenges SET attempts = attempts + 1 WHERE id = $1', [challenge.id],
    );

    const providedHash = this.hashToken(dto.code);
    if (providedHash !== challenge.code_hash) {
      const remaining = OTP_MAX_ATTEMPTS - attempts - 1;
      throw new UnauthorizedException(
        `Invalid verification code${remaining > 0 ? `. ${remaining} attempt${remaining !== 1 ? 's' : ''} remaining.` : '.'}`,
      );
    }

    // ─── Success: create the account (idempotent on email) ───
    const consent = challenge.consent_snapshot;
    const userId = crypto.randomUUID();
    // Passwordless account: set an unusable random password hash so the
    // existing password-login path safely rejects these accounts (they log in
    // via email-OTP / session), and password_hash is never null.
    const unusablePasswordHash = await bcrypt.hash(crypto.randomBytes(32).toString('hex'), 12);

    const userResult = await this.pool.query(
      `INSERT INTO iam.users (id, email, password_hash, full_name, status, email_verified, created_at, updated_at)
       VALUES ($1, $2, $3, $4, 'ACTIVE', true, now(), now())
       ON CONFLICT (email) DO UPDATE SET email_verified = true, updated_at = now()
       RETURNING id, email, full_name`,
      [userId, email, unusablePasswordHash, challenge.full_name],
    );
    const user = userResult.rows[0];

    // Log the immutable consent record (PDPL audit).
    await this.pool.query(
      `INSERT INTO iam.consent_records
         (user_id, email, stage, terms_accepted, risk_acknowledged, marketing_opt_in,
          terms_text, risk_text, marketing_text, consent_version, ip_address, user_agent)
       VALUES ($1, $2, 'REGISTRATION', $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
      [user.id, email, consent.termsAccepted, consent.riskAcknowledged, consent.marketingOptIn,
       consent.termsText, consent.riskText, consent.marketingText, consent.consentVersion,
       ip ?? null, userAgent ?? null],
    );

    // Persist phone + broker in user preferences (additive, non-breaking).
    await this.pool.query(
      `UPDATE iam.users SET preferences = preferences || $1 WHERE id = $2`,
      [JSON.stringify({ phone: challenge.phone ?? null, broker: challenge.broker ?? null }), user.id],
    );

    // Consume the challenge.
    await this.pool.query(
      'UPDATE iam.registration_challenges SET consumed_at = now() WHERE id = $1', [challenge.id],
    );

    // Send the welcome email (with unsubscribe + soft, ungated Google review ask).
    const unsubscribeUrl = this.buildUnsubscribeUrl(email);
    const reviewUrl = this.config.get<string>('GOOGLE_REVIEW_URL') || undefined;
    try {
      await this.emailService.sendWelcomeEmail({
        to: email, name: user.full_name, unsubscribeUrl, reviewUrl,
      });
    } catch (err) {
      this.logger.error(`Failed to send welcome email: ${err instanceof Error ? err.message : 'unknown error'}`);
    }

    // Create an authenticated session (access JWT + opaque refresh token cookie).
    const session = await this.createSession(user.id, user.email);

    return {
      accessToken: session.accessToken,
      user: { id: user.id, email: user.email, displayName: user.full_name },
      _refreshToken: session.refreshToken,
    };
  }

  /* ─── Unsubscribe ─── */

  /**
   * Process a marketing unsubscribe from a signed token (emailed link).
   * Honored immediately and persisted. Idempotent.
   */
  async unsubscribe(token: string, ip: string | undefined, userAgent: string | undefined): Promise<{ success: true; email: string }> {
    let payload: { email?: string; purpose?: string };
    try {
      payload = this.jwtService.verify(token) as { email?: string; purpose?: string };
    } catch {
      throw new BadRequestException('Invalid or expired unsubscribe link.');
    }
    if (payload.purpose !== 'unsubscribe' || !payload.email) {
      throw new BadRequestException('Invalid unsubscribe link.');
    }
    const email = payload.email.trim().toLowerCase();

    await this.pool.query(
      `INSERT INTO iam.marketing_unsubscribes (email, scope, ip_address, user_agent)
       VALUES ($1, 'marketing', $2, $3)
       ON CONFLICT (email) DO UPDATE SET unsubscribed_at = now(), ip_address = $2, user_agent = $3`,
      [email, ip ?? null, userAgent ?? null],
    );

    this.logger.log(`Marketing unsubscribe processed for ${email.replace(/(.{1}).*@/, '$1***@')}`);
    return { success: true as const, email };
  }

  /** Check whether an email has an active marketing unsubscribe. */
  async isUnsubscribed(email: string): Promise<boolean> {
    const result = await this.pool.query(
      'SELECT id FROM iam.marketing_unsubscribes WHERE email = $1', [email.trim().toLowerCase()],
    );
    return result.rows.length > 0;
  }

  /* ─── Cookie configuration (guest session) ─── */

  getGuestCookieOptions(): { name: string; httpOnly: boolean; secure: boolean; sameSite: 'strict' | 'lax' | 'none'; path: string; maxAge: number; domain?: string } {
    const isProduction = this.config.get<string>('NODE_ENV') === 'production';
    const sameSite = this.config.get<string>('AUTH_REFRESH_COOKIE_SAMESITE', 'lax') as 'strict' | 'lax' | 'none';
    const domain = this.config.get<string>('AUTH_REFRESH_COOKIE_DOMAIN') || undefined;
    return {
      name: this.config.get<string>('GUEST_COOKIE_NAME', GUEST_COOKIE_NAME),
      httpOnly: true,
      secure: isProduction,
      sameSite,
      path: '/',
      maxAge: this.getPreviewSeconds(),
      domain,
    };
  }

  getClearGuestCookieOptions(): { name: string; path: string; domain?: string; httpOnly: boolean } {
    const opts = this.getGuestCookieOptions();
    return { name: opts.name, path: opts.path, domain: opts.domain, httpOnly: true };
  }

  /* ─── Private helpers ─── */

  private getPreviewSeconds(): number {
    const raw = this.config.get<number>('PREVIEW_SECONDS', 300);
    const n = Number(raw);
    if (!Number.isFinite(n) || n < 30) return 300;
    return Math.min(n, 3600); // clamp 30s..1h
  }

  private generateOtpCode(): string {
    // Cryptographically random 6-digit code.
    const buf = crypto.randomBytes(4);
    const num = buf.readUInt32BE(0) % 1_000_000;
    return num.toString().padStart(6, '0');
  }

  private hashToken(token: string): string {
    return crypto.createHash('sha256').update(token).digest('hex');
  }

  private buildUnsubscribeUrl(email: string): string {
    const frontendUrl = this.config.get<string>('APP_FRONTEND_URL', 'https://platform.predictatrade.com');
    const token = this.jwtService.sign(
      { email, purpose: 'unsubscribe' },
      { expiresIn: '365d' },
    );
    return `${frontendUrl}/unsubscribe?token=${encodeURIComponent(token)}`;
  }

  private generateRefreshToken(): string {
    return crypto.randomBytes(REFRESH_TOKEN_BYTES).toString('base64url');
  }

  /** Generate access JWT + opaque refresh token. Defaults role to USER. */
  private async generateTokens(userId: string, email: string): Promise<{ accessToken: string; refreshToken: string }> {
    const accessToken = this.jwtService.sign(
      { sub: userId, email, role: 'USER', purpose: 'access' },
      { expiresIn: ACCESS_TOKEN_EXPIRY },
    );
    const refreshToken = this.generateRefreshToken();
    return { accessToken, refreshToken };
  }

  /** Create a session record with hashed refresh token. Returns raw tokens. */
  private async createSession(userId: string, email: string): Promise<{ accessToken: string; refreshToken: string }> {
    const tokens = await this.generateTokens(userId, email);
    const refreshHash = this.hashToken(tokens.refreshToken);
    const sessionId = crypto.randomUUID();
    const family = crypto.randomUUID();

    await this.pool.query(
      `INSERT INTO iam.sessions (id, user_id, token_hash, refresh_token_hash, token_family,
       expires_at, refresh_expires_at, created_at)
       VALUES ($1, $2, $3, $4, $5, now() + interval '${REFRESH_TOKEN_EXPIRY_DAYS} days',
       now() + interval '${REFRESH_TOKEN_EXPIRY_DAYS} days', now())`,
      [sessionId, userId, refreshHash, refreshHash, family],
    );

    return tokens;
  }
}
