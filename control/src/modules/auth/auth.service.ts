import {
  Injectable, UnauthorizedException, ConflictException, NotFoundException,
  BadRequestException, Inject, Logger,
} from '@nestjs/common';
import { JwtService } from '@nestjs/jwt';
import { ConfigService } from '@nestjs/config';
import { Pool } from 'pg';
import * as bcrypt from 'bcrypt';
import * as crypto from 'crypto';
import * as otp from 'otplib';
import { RegisterDto, LoginDto, VerifyOtpDto } from './dto/auth.dto';
import { DB_POOL } from '../../common/database.module';
import { EMAIL_SERVICE } from '../../common/mail/email.service';
import type { EmailService } from '../../common/mail/email.service';

/* ─── Constants ─── */

const MAX_OTP_ATTEMPTS = 5;
const OTP_CHALLENGE_EXPIRY_MIN = 10;
const MAX_LOGIN_FAILURES = 5;
const LOCKOUT_MIN = 15;
const ACCESS_TOKEN_EXPIRY = '1h';
const REFRESH_TOKEN_BYTES = 48; // 384-bit random opaque token
const REFRESH_TOKEN_EXPIRY_DAYS = 7;
const RESET_TOKEN_EXPIRY_MIN = 30;
const REFRESH_COOKIE_NAME = 'pat_refresh_token';
const REFRESH_COOKIE_PATH = '/api/v1/auth';

/* ─── Types ─── */

export interface SessionTokens {
  accessToken: string;
  refreshToken: string; // raw — only used to set cookie, never returned in JSON
}

export interface AuthResponse {
  accessToken: string;
  user: { id: string; email: string; displayName: string };
  /** Set when a privileged account logs in without an enrolled authenticator. */
  mfaEnrollmentRequired?: boolean;
}

export interface CookieOptions {
  name: string;
  value: string;
  httpOnly: boolean;
  secure: boolean;
  sameSite: 'strict' | 'lax' | 'none';
  path: string;
  maxAge: number;
  domain?: string;
}

/** Result of a successful refresh — includes new raw refresh token for cookie. */
export interface RefreshResult {
  accessToken: string;
  refreshToken: string;
}

/** Result of a session creation — includes raw refresh token for cookie. */
export interface SessionCreationResult {
  accessToken: string;
  refreshToken: string;
}

@Injectable()
export class AuthService {
  private readonly logger = new Logger(AuthService.name);

  constructor(
    private jwtService: JwtService,
    private config: ConfigService,
    @Inject(DB_POOL) private pool: Pool,
    @Inject(EMAIL_SERVICE) private emailService: EmailService,
  ) {}

  /* ─── Registration ─── */

  async register(dto: RegisterDto): Promise<AuthResponse & { _refreshToken: string }> {
    // ── Consent validation — required checkboxes must be true ──
    if (!dto.agreeToTerms) {
      throw new BadRequestException('You must agree to the Terms of Use and Privacy Policy');
    }
    if (!dto.acknowledgePrivacyPolicy) {
      throw new BadRequestException('You must acknowledge the Privacy Policy');
    }
    if (!dto.acknowledgeDataProcessing) {
      throw new BadRequestException('You must acknowledge the Data Processing and Security Agreement');
    }

    const existing = await this.pool.query('SELECT id FROM iam.users WHERE email = $1', [dto.email]);
    if (existing.rows.length > 0) throw new ConflictException('Email already registered');
    const passwordHash = await bcrypt.hash(dto.password, 12);
    const userId = crypto.randomUUID();
    // New users land in PENDING — an admin must approve them before they can
    // log in (login hard-blocks non-ACTIVE at auth.service.ts:198).
    await this.pool.query(
      `INSERT INTO iam.users (id, email, password_hash, full_name, status, email_verified, created_at, updated_at)
       VALUES ($1, $2, $3, $4, 'PENDING', false, now(), now())`,
      [userId, dto.email, passwordHash, dto.displayName || dto.email.split('@')[0]],
    );
    await this.pool.query(
      `INSERT INTO audit.audit_events (actor_type, actor_id, action, entity_type, entity_id, reason, new_value)
       VALUES ('system', $2, 'iam.user.registered_pending_approval', 'user', $1, 'Awaiting admin approval', '{"status":"PENDING"}'::jsonb)`,
      [userId, userId],
    );
    if (dto.referralCode) {
      const referrer = await this.pool.query(
        `SELECT rc.user_id FROM referral.referral_codes rc WHERE rc.code = $1 AND rc.active = true`,
        [dto.referralCode],
      );
      if (referrer.rows.length > 0) {
        const referrerUserId = referrer.rows[0].user_id;
        // P0: Self-referral prevention
        if (referrerUserId === userId) {
          throw new BadRequestException('Cannot use your own referral code');
        }
        // P0: Immutable attribution — only set if no existing referral
        const existing = await this.pool.query(
          `SELECT 1 FROM referral.referral_relationships WHERE child_user_id = $1 LIMIT 1`,
          [userId],
        );
        if (existing.rows.length === 0) {
          await this.pool.query(
            `INSERT INTO referral.referral_relationships (child_user_id, parent_user_id, level, created_at)
             VALUES ($1, $2, 1, now()) ON CONFLICT DO NOTHING`,
            [userId, referrerUserId],
          );
        }
      }
    }
    const refCode = 'PAT-' + userId.replace(/-/g, '').toUpperCase().substring(0, 32);
    await this.pool.query(
      `INSERT INTO referral.referral_codes (id, user_id, code, active, created_at)
       VALUES (gen_random_uuid(), $1, $2, true, now()) ON CONFLICT DO NOTHING`,
      [userId, refCode],
    );
    // ── Log consent records for compliance audit ──
    const consentVersion = '2026-08-25';
    const consents = [
      { type: 'AGREE_TO_TERMS', accepted: true, text: 'I agree to the Terms of Use and Privacy Policy' },
      { type: 'ACKNOWLEDGE_PRIVACY_POLICY', accepted: true, text: 'I confirm that I have read and acknowledge the Privacy Policy' },
      { type: 'ACKNOWLEDGE_DATA_PROCESSING', accepted: true, text: 'I confirm that I have read and acknowledge the Data Processing and Security Agreement' },
      { type: 'OPT_IN_EMAIL_MARKETING', accepted: !!dto.optInEmailMarketing, text: 'I want to receive news and promotional offers by email' },
      { type: 'OPT_IN_SMS_MARKETING', accepted: !!dto.optInSmsMarketing, text: 'I want to receive news and promotional offers by SMS' },
      { type: 'OPT_IN_PHONE_MARKETING', accepted: !!dto.optInPhoneMarketing, text: 'I want to receive news and promotional offers by phone call' },
    ];
    for (const c of consents) {
      try {
        await this.pool.query(
          `INSERT INTO audit.client_events (user_id, event_type, metadata, event_time)
           VALUES ($1, 'CONSENT_LOG', $2, now())`,
          [userId, JSON.stringify({ consentType: c.type, accepted: c.accepted, textVersion: consentVersion, consentText: c.text })],
        );
      } catch (e) {
        this.logger?.warn?.(`Failed to log consent ${c.type}: ${e}`) ;
      }
    }

    // ── Store marketing preferences and consent version on the user record ──
    try {
      await this.pool.query(
        `UPDATE iam.users SET
           marketing_email_optin = $2,
           marketing_sms_optin = $3,
           marketing_phone_optin = $4,
           consent_version = $5,
           consent_timestamp = now(),
           updated_at = now()
         WHERE id = $1`,
        [userId, !!dto.optInEmailMarketing, !!dto.optInSmsMarketing, !!dto.optInPhoneMarketing, consentVersion],
      );
    } catch (e) {
      // Marketing preference columns may not exist yet — log and continue
      this.logger?.warn?.(`Marketing preferences update failed (non-fatal): ${e}`);
    }

    const session = await this.createSession(userId, dto.email);
    return {
      accessToken: session.accessToken,

      user: { id: userId, email: dto.email, displayName: dto.displayName || dto.email.split('@')[0] },
      _refreshToken: session.refreshToken,
    };
  }

  /* ─── Login ─── */

  async login(dto: LoginDto): Promise<(AuthResponse & { _refreshToken: string }) | { mfaRequired: true; challengeId: string; method: string }> {
    const result = await this.pool.query(
      'SELECT id, email, password_hash, full_name, status, failed_login_count, locked_until FROM iam.users WHERE email = $1',
      [dto.email],
    );

    if (result.rows.length === 0) {
      throw new UnauthorizedException('Invalid credentials');
    }

    const user = result.rows[0];

    if (user.locked_until && new Date(user.locked_until) > new Date()) {
      throw new UnauthorizedException('Account temporarily locked. Try again later.');
    }

    if (user.status !== 'ACTIVE') {
      throw new UnauthorizedException('Account is not active. Contact support.');
    }

    const valid = await bcrypt.compare(dto.password, user.password_hash);
    if (!valid) {
      const newFailCount = (user.failed_login_count || 0) + 1;
      const shouldLock = newFailCount >= MAX_LOGIN_FAILURES;
      await this.pool.query(
        `UPDATE iam.users SET failed_login_count = $1, locked_until = $2 WHERE id = $3`,
        [newFailCount, shouldLock ? new Date(Date.now() + LOCKOUT_MIN * 60_000) : null, user.id],
      );
      await this.logLoginEvent(user.id, 'LOGIN_FAILED', { reason: 'invalid_password' });
      throw new UnauthorizedException(shouldLock ? 'Account locked due to too many failed attempts.' : 'Invalid credentials');
    }

    // Check MFA
    const mfaResult = await this.pool.query(
      `SELECT id, secret FROM iam.mfa_methods WHERE user_id = $1 AND method_type = 'TOTP' AND is_enabled = true`,
      [user.id],
    );

    // AUTH-1: MFA is mandatory for privileged roles. Resolve the user's role
    // (same lookup used when minting the JWT). If an ADMIN / SUPER_ADMIN / OPERATOR
    // has not yet enrolled an authenticator, we still let the login succeed so the
    // operator can reach the enrollment screen, but flag it so the UI forces MFA
    // enrollment before any privileged action. This closes the "MFA opt-in" gap
    // WITHOUT locking operators out of the enrollment flow (chicken-and-egg).
    const roleResult = await this.pool.query(
      `SELECT r.name AS role_name FROM iam.memberships m
       JOIN iam.roles r ON m.role_id = r.id
       WHERE m.user_id = $1`,
      [user.id],
    );
    const userRole = roleResult.rows.length > 0 ? roleResult.rows[0].role_name : 'USER';
    const PRIVILEGED_ROLES = new Set(['ADMIN', 'SUPER_ADMIN', 'OPERATOR']);
    const requiresMfaEnrollment = PRIVILEGED_ROLES.has(userRole) && mfaResult.rows.length === 0;
    if (requiresMfaEnrollment) {
      await this.logLoginEvent(user.id, 'LOGIN_MFA_ENROLLMENT_REQUIRED', { role: userRole });
    }

    if (mfaResult.rows.length > 0 && !dto.mfaCode) {
      const challengeId = crypto.randomUUID();
      await this.pool.query(
        `INSERT INTO iam.login_events (user_id, event_type, metadata)
         VALUES ($1, 'MFA_CHALLENGE', $2)`,
        [user.id, JSON.stringify({ challengeId, method: 'TOTP' })],
      );
      return { mfaRequired: true, challengeId, method: 'TOTP' };
    }

    if (mfaResult.rows.length > 0 && dto.mfaCode) {
      const validOtp = otp.authenticator.verify({ token: dto.mfaCode, secret: mfaResult.rows[0].secret });
      if (!validOtp) {
        await this.logLoginEvent(user.id, 'MFA_FAILED', { reason: 'invalid_inline_code' });
        throw new UnauthorizedException('Invalid MFA code');
      }
      await this.logLoginEvent(user.id, 'MFA_SUCCESS', { inline: true });
    }

    await this.pool.query(
      'UPDATE iam.users SET failed_login_count = 0, locked_until = null, last_login_at = now() WHERE id = $1',
      [user.id],
    );
    await this.logLoginEvent(user.id, 'LOGIN_SUCCESS', {});

    const session = await this.createSession(user.id, user.email);
    return {
      accessToken: session.accessToken,
      mfaEnrollmentRequired: requiresMfaEnrollment,
      user: { id: user.id, email: user.email, displayName: user.full_name },
      _refreshToken: session.refreshToken,
    };
  }

  /* ─── OTP verification ─── */

  async verifyOtp(dto: VerifyOtpDto): Promise<AuthResponse & { licenseValid: boolean; deviceRegistered: boolean; _refreshToken: string }> {
    const challengeResult = await this.pool.query(
      `SELECT id, user_id, metadata, created_at FROM iam.login_events
       WHERE event_type = 'MFA_CHALLENGE' AND metadata->>'challengeId' = $1
       ORDER BY created_at DESC LIMIT 1`,
      [dto.challengeId],
    );

    if (challengeResult.rows.length === 0) {
      throw new UnauthorizedException('Invalid or expired challenge');
    }

    const challenge = challengeResult.rows[0];
    const challengeAge = Date.now() - new Date(challenge.created_at).getTime();
    if (challengeAge > OTP_CHALLENGE_EXPIRY_MIN * 60_000) {
      throw new UnauthorizedException('Challenge has expired. Please sign in again.');
    }

    const mfaResult = await this.pool.query(
      `SELECT secret FROM iam.mfa_methods WHERE user_id = $1 AND method_type = 'TOTP' AND is_enabled = true`,
      [challenge.user_id],
    );

    if (mfaResult.rows.length === 0) {
      throw new UnauthorizedException('MFA not configured for this account');
    }

    const attemptsResult = await this.pool.query(
      `SELECT count(*) as cnt FROM iam.login_events
       WHERE user_id = $1 AND event_type = 'MFA_FAILED' AND created_at >= $2`,
      [challenge.user_id, challenge.created_at],
    );
    const attempts = parseInt(attemptsResult.rows[0].cnt, 10);
    if (attempts >= MAX_OTP_ATTEMPTS) {
      throw new BadRequestException('Too many failed attempts. Challenge invalidated.');
    }

    const valid = otp.authenticator.verify({ token: dto.code, secret: mfaResult.rows[0].secret });
    if (!valid) {
      await this.logLoginEvent(challenge.user_id, 'MFA_FAILED', { challengeId: dto.challengeId });
      const remaining = MAX_OTP_ATTEMPTS - attempts - 1;
      throw new UnauthorizedException(`Invalid verification code${remaining > 0 ? `. ${remaining} attempt${remaining !== 1 ? 's' : ''} remaining.` : '.'}`);
    }

    await this.logLoginEvent(challenge.user_id, 'MFA_SUCCESS', { challengeId: dto.challengeId });
    await this.pool.query(
      'UPDATE iam.users SET failed_login_count = 0, locked_until = null, last_login_at = now() WHERE id = $1',
      [challenge.user_id],
    );

    const userResult = await this.pool.query('SELECT email, full_name FROM iam.users WHERE id = $1', [challenge.user_id]);
    const user = userResult.rows[0];
    const session = await this.createSession(challenge.user_id, user.email, dto.trustDevice);

    // Check license
    const licenseResult = await this.pool.query(
      `SELECT status FROM licensing.licenses WHERE user_id = $1 AND status = 'ACTIVE' LIMIT 1`,
      [challenge.user_id],
    );

    return {
      accessToken: session.accessToken,

      user: { id: challenge.user_id, email: user.email, displayName: user.full_name },
      licenseValid: licenseResult.rows.length > 0,
      deviceRegistered: Boolean(dto.trustDevice),
      _refreshToken: session.refreshToken,
    };
  }

  /* ─── Refresh (cookie-based, with rotation + reuse detection) ─── */

  /**
   * Validate a raw refresh token, rotate it, and return the new tokens.
   * The entire operation is wrapped in a database transaction with FOR UPDATE
   * row locking to guarantee atomicity and prevent concurrent-use issues.
   *
   * The new raw refresh token is returned directly (not stashed on the instance)
   * to eliminate race conditions in concurrent scenarios.
   */
  async refresh(rawRefreshToken: string): Promise<RefreshResult> {
    const tokenHash = this.hashToken(rawRefreshToken);

    const client = await this.pool.connect();
    try {
      await client.query('BEGIN');

      // Find the session by refresh token hash — lock the row
      const sessionResult = await client.query(
        `SELECT s.id, s.user_id, s.token_family, s.revoked_at, s.expires_at, s.refresh_expires_at,
                u.status as user_status
         FROM iam.sessions s
         JOIN iam.users u ON u.id = s.user_id
         WHERE s.refresh_token_hash = $1
         FOR UPDATE OF s`,
        [tokenHash],
      );

      if (sessionResult.rows.length === 0) {
        await client.query('ROLLBACK');
        throw new UnauthorizedException('Invalid refresh token');
      }

      const session = sessionResult.rows[0];

      // Check if already revoked → reuse detection
      if (session.revoked_at) {
        // REUSE DETECTED: an already-rotated token was submitted again.
        // Revoke the entire token family to protect the user.
        if (session.token_family) {
          await client.query(
            `UPDATE iam.sessions SET revoked_at = now()
             WHERE token_family = $1 AND revoked_at IS NULL`,
            [session.token_family],
          );
        }
        await this.logLoginEvent(session.user_id, 'REFRESH_TOKEN_REUSE', {
          sessionId: session.id,
          tokenFamily: session.token_family,
        });
        await client.query('COMMIT');
        this.logger.warn(`Refresh token reuse detected for user ${session.user_id}, family ${session.token_family}`);
        throw new UnauthorizedException('Session invalidated due to token reuse');
      }

      // Check expiry
      if (new Date(session.expires_at) < new Date()) {
        await client.query(
          'UPDATE iam.sessions SET revoked_at = now() WHERE id = $1',
          [session.id],
        );
        await client.query('COMMIT');
        throw new UnauthorizedException('Session expired');
      }

      // Check user account status — disabled/deleted users must not refresh
      if (session.user_status !== 'ACTIVE') {
        await client.query(
          'UPDATE iam.sessions SET revoked_at = now() WHERE id = $1',
          [session.id],
        );
        await this.logLoginEvent(session.user_id, 'REFRESH_REJECTED', { reason: 'account_inactive', status: session.user_status });
        await client.query('COMMIT');
        throw new UnauthorizedException('Account is not active');
      }

      // Rotate: revoke old session, create new one in the same family
      await client.query(
        'UPDATE iam.sessions SET revoked_at = now() WHERE id = $1',
        [session.id],
      );

      const userResult = await client.query('SELECT email FROM iam.users WHERE id = $1', [session.user_id]);
      const newTokens = await this.generateTokens(session.user_id, userResult.rows[0].email);

      const newRefreshHash = this.hashToken(newTokens.refreshToken);
      const family = session.token_family || crypto.randomUUID();

      await client.query(
        `INSERT INTO iam.sessions (id, user_id, token_hash, refresh_token_hash, token_family,
         expires_at, refresh_expires_at, created_at)
         VALUES ($1, $2, $3, $4, $5, now() + interval '${REFRESH_TOKEN_EXPIRY_DAYS} days',
         now() + interval '${REFRESH_TOKEN_EXPIRY_DAYS} days', now())`,
        [crypto.randomUUID(), session.user_id, newRefreshHash, newRefreshHash, family],
      );

      await this.logLoginEvent(session.user_id, 'REFRESH_SUCCESS', {});
      await client.query('COMMIT');

      return { accessToken: newTokens.accessToken, refreshToken: newTokens.refreshToken };
    } catch (err) {
      try { await client.query('ROLLBACK'); } catch { /* already rolled back or committed */ }
      throw err;
    } finally {
      client.release();
    }
  }

  /* ─── Logout ─── */

  async logout(userId: string, rawRefreshToken?: string): Promise<void> {
    if (rawRefreshToken) {
      // Revoke the specific session
      const tokenHash = this.hashToken(rawRefreshToken);
      await this.pool.query(
        'UPDATE iam.sessions SET revoked_at = now() WHERE user_id = $1 AND refresh_token_hash = $2 AND revoked_at IS NULL',
        [userId, tokenHash],
      );
    } else {
      // Fallback: revoke all sessions for the user
      await this.pool.query(
        'UPDATE iam.sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL',
        [userId],
      );
    }
    await this.logLoginEvent(userId, 'LOGOUT', {});
  }

  /* ─── Get profile ─── */

  async getProfile(userId: string) {
    const result = await this.pool.query(
      'SELECT id, email, full_name, status, created_at FROM iam.users WHERE id = $1', [userId],
    );
    if (result.rows.length === 0) throw new NotFoundException('User not found');
    return result.rows[0];
  }

  /* ─── MFA setup/verify ─── */

  async setupMfa(userId: string) {
    const secret = otp.authenticator.generateSecret();
    const user = await this.pool.query('SELECT email FROM iam.users WHERE id = $1', [userId]);
    if (user.rows.length === 0) throw new NotFoundException('User not found');
    await this.pool.query(
      `INSERT INTO iam.mfa_methods (id, user_id, method_type, secret, is_enabled, created_at)
       VALUES ($1, $2, 'TOTP', $3, false, now())
       ON CONFLICT (user_id, method_type) DO UPDATE SET secret = $3`,
      [crypto.randomUUID(), userId, secret],
    );
    const otpauth = otp.authenticator.keyuri(user.rows[0].email, 'Predict-A-Trade', secret);
    return { secret, otpauth };
  }

  async verifyMfa(userId: string, code: string) {
    const r = await this.pool.query(
      `SELECT secret FROM iam.mfa_methods WHERE user_id = $1 AND method_type = 'TOTP'`, [userId],
    );
    if (r.rows.length === 0) throw new NotFoundException('MFA not set up');
    const valid = otp.authenticator.verify({ token: code, secret: r.rows[0].secret });
    if (!valid) throw new UnauthorizedException('Invalid MFA code');
    await this.pool.query(
      `UPDATE iam.mfa_methods SET is_enabled = true WHERE user_id = $1 AND method_type = 'TOTP'`, [userId],
    );
    return { mfaEnabled: true };
  }

  /* ─── Forgot / Reset password ─── */

  async forgotPassword(email: string): Promise<{ message: string }> {
    // Always return the same generic response to prevent account enumeration
    const genericResponse = { message: 'If an account exists for this email, password reset instructions have been sent.' };

    const result = await this.pool.query(
      'SELECT id FROM iam.users WHERE email = $1 AND status = $2', [email, 'ACTIVE'],
    );

    if (result.rows.length > 0) {
      const user = result.rows[0];
      const resetToken = this.jwtService.sign(
        { sub: user.id, purpose: 'password_reset', jti: crypto.randomUUID() },
        { expiresIn: `${RESET_TOKEN_EXPIRY_MIN}m` },
      );
      const expiresAt = new Date(Date.now() + RESET_TOKEN_EXPIRY_MIN * 60_000);

      const frontendUrl = this.config.get<string>('APP_FRONTEND_URL', 'https://platform.predictatrade.com');
      const resetUrl = `${frontendUrl}/reset-password?token=${encodeURIComponent(resetToken)}`;

      try {
        await this.emailService.sendPasswordResetEmail({ to: email, resetUrl, expiresAt });
        await this.logLoginEvent(user.id, 'PASSWORD_RESET_REQUESTED', {});
      } catch (err) {
        // Log the error but do NOT reveal to the user that the email failed.
        // Never log the reset token or full reset URL.
        this.logger.error(`Failed to send password-reset email: ${err instanceof Error ? err.message : 'unknown error'}`);
      }
    }

    return genericResponse;
  }

  /**
   * Reset a password using a one-time-use JWT reset token.
   *
   * The entire operation (verify token unused → consume token → update password
   * → revoke sessions) is wrapped in a serializable transaction with advisory
   * locking on the reset token jti to prevent race-based double use.
   */
  async resetPassword(token: string, password: string): Promise<{ success: true }> {
    let payload: { sub: string; purpose: string; jti: string };
    try {
      payload = this.jwtService.verify(token) as { sub: string; purpose: string; jti: string };
    } catch {
      throw new BadRequestException('Invalid or expired reset token');
    }

    if (payload.purpose !== 'password_reset') {
      throw new BadRequestException('Invalid reset token');
    }

    const client = await this.pool.connect();
    try {
      await client.query('BEGIN');

      // Advisory lock on the jti hash to serialize concurrent requests using the same token.
      // Two simultaneous requests with the same jti will serialize; the second will find the
      // token already consumed and fail.
      const jtiHash = crypto.createHash('sha256').update(payload.jti).digest('hex').substring(0, 32);
      const lockKey = parseInt(jtiHash, 16) % 2147483647; // pg_try_advisory_xact_lock takes bigint
      await client.query('SELECT pg_advisory_xact_lock($1)', [lockKey]);

      // One-time use: check if this reset token (jti) has already been used
      const usedTokenResult = await client.query(
        `SELECT id FROM iam.login_events
         WHERE user_id = $1 AND event_type = 'PASSWORD_RESET_COMPLETED'
         AND metadata->>'jti' = $2
         FOR UPDATE`,
        [payload.sub, payload.jti],
      );
      if (usedTokenResult.rows.length > 0) {
        await client.query('ROLLBACK');
        throw new BadRequestException('This reset link has already been used. Please request a new one.');
      }

      // Check user is still active
      const userResult = await client.query('SELECT status FROM iam.users WHERE id = $1', [payload.sub]);
      if (userResult.rows.length === 0) {
        await client.query('ROLLBACK');
        throw new BadRequestException('Invalid reset token');
      }
      if (userResult.rows[0].status !== 'ACTIVE') {
        await client.query('ROLLBACK');
        throw new BadRequestException('Account is not active. Contact support.');
      }

      const passwordHash = await bcrypt.hash(password, 12);
      await client.query(
        'UPDATE iam.users SET password_hash = $1, updated_at = now() WHERE id = $2',
        [passwordHash, payload.sub],
      );

      // Revoke all existing sessions (security: stolen sessions must not survive password reset)
      await client.query(
        'UPDATE iam.sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL',
        [payload.sub],
      );

      // Record the completion BEFORE commit so concurrent requests see it
      await client.query(
        `INSERT INTO iam.login_events (user_id, event_type, metadata) VALUES ($1, 'PASSWORD_RESET_COMPLETED', $2)`,
        [payload.sub, JSON.stringify({ jti: payload.jti })],
      );

      await client.query('COMMIT');
      return { success: true as const };
    } catch (err) {
      try { await client.query('ROLLBACK'); } catch { /* already rolled back */ }
      throw err;
    } finally {
      client.release();
    }
  }

  /* ─── Cookie configuration ─── */

  getRefreshCookieOptions(): CookieOptions {
    const isProduction = this.config.get<string>('NODE_ENV') === 'production';
    const sameSite = this.config.get<string>('AUTH_REFRESH_COOKIE_SAMESITE', 'lax') as 'strict' | 'lax' | 'none';
    const domain = this.config.get<string>('AUTH_REFRESH_COOKIE_DOMAIN') || undefined;
    return {
      name: this.config.get<string>('AUTH_REFRESH_COOKIE_NAME', REFRESH_COOKIE_NAME),
      value: '', // set by caller
      httpOnly: true,
      secure: isProduction,
      sameSite,
      path: this.config.get<string>('AUTH_REFRESH_COOKIE_PATH', REFRESH_COOKIE_PATH),
      maxAge: REFRESH_TOKEN_EXPIRY_DAYS * 24 * 60 * 60,
      domain,
    };
  }

  getClearCookieOptions(): { name: string; path: string; domain?: string; httpOnly: boolean } {
    const opts = this.getRefreshCookieOptions();
    return { name: opts.name, path: opts.path, domain: opts.domain, httpOnly: true };
  }

  /* ─── Private helpers ─── */

  /** Generate a cryptographically secure opaque refresh token. */
  private generateRefreshToken(): string {
    return crypto.randomBytes(REFRESH_TOKEN_BYTES).toString('base64url');
  }

  /** Hash a refresh token using SHA-256. Only the hash is stored in the database. */
  private hashToken(token: string): string {
    return crypto.createHash('sha256').update(token).digest('hex');
  }

  /** Generate access JWT + opaque refresh token. Includes user role + permissions for RBAC. */
  private async generateTokens(userId: string, email: string): Promise<SessionTokens> {
    // Look up the user's role from memberships/roles
    let role = 'USER';
    const perms: string[] = [];
    try {
      const roleResult = await this.pool.query(
        `SELECT r.name as role_name FROM iam.memberships m
         JOIN iam.roles r ON m.role_id = r.id
         WHERE m.user_id = $1
         ORDER BY CASE r.name
           WHEN 'SUPER_ADMIN' THEN 1 WHEN 'ADMIN' THEN 2
           WHEN 'RISK_MANAGER' THEN 3 WHEN 'TRADING_OPERATOR' THEN 4
           WHEN 'SUPPORT' THEN 5 WHEN 'ANALYST' THEN 6 WHEN 'AUDITOR' THEN 7
           ELSE 8 END
         LIMIT 1`, [userId],
      );
      if (roleResult.rows.length > 0) {
        role = roleResult.rows[0].role_name;
      }
    } catch {
      // Role lookup failure is non-fatal — default to USER
    }
    // Load the user's permission names (role_permissions ⋈ permissions) into
    // the token so @RequirePermissions guards can evaluate without a DB hit.
    try {
      const permResult = await this.pool.query(
        `SELECT DISTINCT p.name FROM iam.role_permissions rp
         JOIN iam.roles r ON rp.role_id = r.id
         JOIN iam.memberships m ON m.role_id = r.id
         JOIN iam.permissions p ON p.id = rp.permission_id
         WHERE m.user_id = $1`,
        [userId],
      );
      for (const row of permResult.rows) perms.push(row.name);
    } catch {
      // Permission lookup failure is non-fatal — role-based checks still apply
    }
    const accessToken = this.jwtService.sign({ sub: userId, email, role, permissions: perms, purpose: 'access' }, { expiresIn: ACCESS_TOKEN_EXPIRY });
    const refreshToken = this.generateRefreshToken();
    return { accessToken, refreshToken };
  }

  /** Create a session record with hashed refresh token. Returns raw tokens for cookie setting. */
  private async createSession(userId: string, email: string, _trustedDevice?: boolean): Promise<SessionCreationResult> {
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

  private async logLoginEvent(userId: string | null, eventType: string, metadata: Record<string, unknown>) {
    try {
      await this.pool.query(
        `INSERT INTO iam.login_events (user_id, event_type, metadata) VALUES ($1, $2, $3)`,
        [userId, eventType, JSON.stringify(metadata)],
      );
    } catch {
      // Don't let audit logging failures break the auth flow
    }
    // Also persist to audit.audit_events for the Admin Logs & Audit page
    try {
      const auditAction = eventType; // LOGIN_SUCCESS, LOGIN_FAILED, MFA_CHALLENGE, etc.
      const actorType = userId ? 'USER' : 'ANONYMOUS';
      await this.pool.query(
        `INSERT INTO audit.audit_events (id, event_id, actor_type, actor_id, action, entity_type, new_value, timestamp)
         VALUES (gen_random_uuid(), gen_random_uuid(), $1, $2, $3, 'auth', $4, now())`,
        [actorType, userId, auditAction, JSON.stringify(this.sanitizeMetadata(metadata))],
      );
    } catch {
      // Don't let audit logging failures break the auth flow
    }
  }

  /** Sanitize metadata to remove secrets before audit logging. */
  private sanitizeMetadata(metadata: Record<string, unknown>): Record<string, unknown> {
    const sanitized = { ...metadata };
    // Remove any sensitive fields
    for (const key of Object.keys(sanitized)) {
      if (['password', 'token', 'secret', 'hash', 'refreshToken', 'accessToken'].includes(key.toLowerCase())) {
        delete sanitized[key];
      }
    }
    return sanitized;
  }

  /** check.md 2026-08-30 #23 — admin/user Settings password change */
  async changePassword(userId: string, currentPassword: string, newPassword: string) {
    if (!newPassword || newPassword.length < 8) {
      throw new BadRequestException('Password must be at least 8 characters');
    }
    if (!currentPassword) throw new BadRequestException('Current password required');
    const r = await this.pool.query(`SELECT id, password_hash, email FROM iam.users WHERE id = $1`, [userId]);
    const user = r.rows[0];
    if (!user) throw new UnauthorizedException('user_not_found');
    const ok = await bcrypt.compare(currentPassword, user.password_hash);
    if (!ok) throw new UnauthorizedException('Current password is incorrect');
    const same = await bcrypt.compare(newPassword, user.password_hash);
    if (same) throw new BadRequestException('New password must differ from current password');
    const hash = await bcrypt.hash(newPassword, 12);
    await this.pool.query(`UPDATE iam.users SET password_hash = $2, updated_at = now() WHERE id = $1`, [userId, hash]);
    // Single session invalidation: delete refresh sessions so they must re-login
    await this.pool.query(`DELETE FROM iam.sessions WHERE user_id = $1`, [userId]);
    await this.pool.query(
      `INSERT INTO audit.audit_events (actor_type, actor_id, action, entity_type, entity_id, reason)
       VALUES ('system', $1, 'iam.password_changed', 'user', $1, 'Settings password change')`,
      [userId],
    );
    return { changed: true };
  }
}
