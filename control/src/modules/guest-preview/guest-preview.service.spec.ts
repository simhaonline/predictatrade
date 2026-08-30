/**
 * GuestPreviewService unit tests.
 *
 * Verifies the core security/compliance properties of the guest-preview gate:
 *   - Required consents enforced server-side (terms + risk); marketing optional.
 *   - OTP stored hashed (SHA-256), 6 digits.
 *   - Max-attempts limit (5) on verify; challenge consumed on exhaustion.
 *   - Expired challenge rejected.
 *   - Wrong code rejected with remaining-attempts message.
 *   - Email enumeration protection (generic response for already-verified email).
 *   - Resend cooldown (60s) enforced.
 *   - Guest session JWT carries a server-side expiry (exp claim).
 *   - Unsubscribe token validation (purpose check, invalid token rejected).
 *   - Preview duration configurable via PREVIEW_SECONDS.
 */

import * as crypto from 'crypto';
import { jest } from '@jest/globals';
import { GuestPreviewService } from './guest-preview.service';
import { BadRequestException, UnauthorizedException } from '@nestjs/common';

describe('GuestPreviewService', () => {
  let service: GuestPreviewService;
  let jwtService: any;
  let configService: any;
  let mockEmailService: any;
  let mockPool: any;

  beforeEach(() => {
    jwtService = {
      sign: jest.fn((payload: any, opts?: any) => {
        // Produce a deterministic fake JWT whose payload encodes exp for status checks.
        const exp = opts?.expiresIn
          ? Math.floor(Date.now() / 1000) + (opts.expiresIn.endsWith('s') ? parseInt(opts.expiresIn) : 300)
          : undefined;
        const body = Buffer.from(JSON.stringify({ ...payload, exp })).toString('base64url');
        return `header.${body}.sig`;
      }),
      verify: jest.fn(),
    };

    configService = {
      get: jest.fn((key: string, def?: any) => {
        const cfg: Record<string, any> = {
          NODE_ENV: 'test',
          PREVIEW_SECONDS: 300,
          APP_FRONTEND_URL: 'https://test.example.com',
          GUEST_COOKIE_NAME: 'pat_guest_session',
        };
        return cfg[key] ?? def;
      }),
    };

    mockEmailService = {
      sendPasswordResetEmail: jest.fn().mockResolvedValue(undefined),
      sendOtpEmail: jest.fn().mockResolvedValue(undefined),
      sendWelcomeEmail: jest.fn().mockResolvedValue(undefined),
    };

    mockPool = { query: jest.fn().mockResolvedValue({ rows: [] }) };

    service = new GuestPreviewService(jwtService, configService, mockPool, mockEmailService);
  });

  // ─── Guest session ───

  describe('Guest session issuance', () => {
    it('issues a signed token with a server-side expiry of PREVIEW_SECONDS', async () => {
      const result = await service.issueGuestSession();
      expect(result.guestToken).toBeDefined();
      expect(result.previewSeconds).toBe(300);
      expect(result.expiresAt).toBeGreaterThan(Date.now());
      // JWT sign called with purpose=guest_preview and expiresIn in seconds
      expect(jwtService.sign).toHaveBeenCalledWith(
        expect.objectContaining({ purpose: 'guest_preview' }),
        expect.objectContaining({ expiresIn: '300s' }),
      );
    });

    it('clamps PREVIEW_SECONDS to a safe range', async () => {
      (configService.get as jest.Mock).mockImplementation((key: string, def?: any) => {
        if (key === 'PREVIEW_SECONDS') return 5; // too low
        return def;
      });
      const result = await service.issueGuestSession();
      expect(result.previewSeconds).toBe(300); // falls back to default
    });

    it('returns locked=true when no token is provided', async () => {
      const status = await service.getGuestStatus(undefined);
      expect(status.locked).toBe(true);
      expect(status.expiresAt).toBeNull();
    });

    it('returns locked=true when the token is expired/invalid', async () => {
      jwtService.verify.mockImplementation(() => { throw new Error('jwt expired'); });
      const status = await service.getGuestStatus('expired.token.here');
      expect(status.locked).toBe(true);
    });

    it('returns remaining seconds for a valid token', async () => {
      jwtService.verify.mockReturnValue({ purpose: 'guest_preview', exp: Math.floor(Date.now() / 1000) + 120 });
      const status = await service.getGuestStatus('valid.token.here');
      expect(status.locked).toBe(false);
      expect(status.remainingSeconds).toBeGreaterThan(110);
    });

    it('rejects a token with the wrong purpose', async () => {
      jwtService.verify.mockReturnValue({ purpose: 'access', exp: Math.floor(Date.now() / 1000) + 120 });
      const status = await service.getGuestStatus('wrong.purpose');
      expect(status.locked).toBe(true);
    });
  });

  // ─── Registration consent enforcement ───

  describe('Registration consent enforcement', () => {
    const baseDto = {
      fullName: 'Jane Doe', email: 'jane@test.com', phone: '+971500000000', broker: 'Exness',
      termsAccepted: true, riskAcknowledged: true, marketingOptIn: false,
    };

    it('rejects when terms not accepted', async () => {
      await expect(service.register({ ...baseDto, termsAccepted: false }, undefined, undefined))
        .rejects.toThrow(BadRequestException);
    });

    it('rejects when risk not acknowledged', async () => {
      await expect(service.register({ ...baseDto, riskAcknowledged: false }, undefined, undefined))
        .rejects.toThrow(BadRequestException);
    });

    it('allows registration WITHOUT marketing opt-in (optional)', async () => {
      mockPool.query
        .mockResolvedValueOnce({ rows: [] }) // existing verified check
        .mockResolvedValueOnce({ rows: [] }) // recent challenge cooldown check
        .mockResolvedValueOnce({ rows: [{ id: 'ch-new' }] }); // INSERT returning id
      const result = await service.register({ ...baseDto, marketingOptIn: false }, undefined, undefined);
      expect(result.message).toContain('verification code');
      expect(mockEmailService.sendOtpEmail).toHaveBeenCalled();
    });

    it('lowercases and trims the email', async () => {
      mockPool.query
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [] })
        .mockResolvedValueOnce({ rows: [{ id: 'ch-new' }] });
      await service.register({ ...baseDto, email: '  Jane@Test.COM  ' }, undefined, undefined);
      const insertCall = mockPool.query.mock.calls.find(
        (c: any[]) => typeof c[0] === 'string' && c[0].includes('INSERT INTO iam.registration_challenges'),
      );
      expect(insertCall[1][0]).toBe('jane@test.com');
    });

    it('returns a generic message for an already-verified email (no enumeration)', async () => {
      mockPool.query.mockResolvedValueOnce({ rows: [{ id: 'u1' }] }); // existing verified
      const result = await service.register(baseDto, undefined, undefined);
      expect(result.message).toContain('If this email is not already registered');
      expect(mockEmailService.sendOtpEmail).not.toHaveBeenCalled();
    });

    it('enforces the 60-second resend cooldown', async () => {
      mockPool.query
        .mockResolvedValueOnce({ rows: [] }) // existing check
        .mockResolvedValueOnce({ rows: [{ id: 'c1', created_at: new Date() }] }); // recent challenge
      await expect(service.register(baseDto, undefined, undefined))
        .rejects.toThrow(BadRequestException);
    });
  });

  // ─── OTP verify ───

  describe('OTP verification', () => {
    const validCode = '123456';
    const validHash = crypto.createHash('sha256').update(validCode).digest('hex');

    function mockChallenge(overrides: any = {}) {
      return {
        id: 'ch1', code_hash: validHash, full_name: 'Jane', phone: null, broker: 'Exness',
        consent_snapshot: {
          termsAccepted: true, riskAcknowledged: true, marketingOptIn: false,
          termsText: 't', riskText: 'r', marketingText: 'm', consentVersion: '1.0.0',
        },
        attempts: 0, max_attempts: 5, expires_at: new Date(Date.now() + 60_000),
        ...overrides,
      };
    }

    it('rejects when no challenge exists (generic error)', async () => {
      mockPool.query.mockResolvedValue({ rows: [] });
      await expect(service.verifyOtp({ email: 'jane@test.com', code: validCode }, undefined, undefined))
        .rejects.toThrow(UnauthorizedException);
    });

    it('rejects an expired challenge', async () => {
      mockPool.query.mockResolvedValue({ rows: [mockChallenge({ expires_at: new Date(Date.now() - 1000) })] });
      await expect(service.verifyOtp({ email: 'jane@test.com', code: validCode }, undefined, undefined))
        .rejects.toThrow(UnauthorizedException);
    });

    it('rejects a wrong code and reports remaining attempts', async () => {
      mockPool.query.mockResolvedValue({ rows: [mockChallenge({ attempts: 2 })] });
      await expect(service.verifyOtp({ email: 'jane@test.com', code: '000000' }, undefined, undefined))
        .rejects.toThrow(/remaining/);
    });

    it('consumes the challenge after max attempts exceeded', async () => {
      mockPool.query.mockResolvedValue({ rows: [mockChallenge({ attempts: 5 })] });
      await expect(service.verifyOtp({ email: 'jane@test.com', code: validCode }, undefined, undefined))
        .rejects.toThrow(/Too many failed attempts/);
      const consumeCall = mockPool.query.mock.calls.find(
        (c: any[]) => typeof c[0] === 'string' && c[0].includes('SET consumed_at = now()'),
      );
      expect(consumeCall).toBeDefined();
    });

    it('creates the account, logs consent, and returns a session on success', async () => {
      // challenge lookup → increment attempts → user upsert → consent insert → preferences update → consume → session insert
      mockPool.query
        .mockResolvedValueOnce({ rows: [mockChallenge()] }) // challenge
        .mockResolvedValueOnce({ rows: [] }) // increment attempts
        .mockResolvedValueOnce({ rows: [{ id: 'u1', email: 'jane@test.com', full_name: 'Jane' }] }) // user upsert
        .mockResolvedValueOnce({ rows: [] }) // consent insert
        .mockResolvedValueOnce({ rows: [] }) // preferences update
        .mockResolvedValueOnce({ rows: [] }) // consume challenge
        .mockResolvedValueOnce({ rows: [] }); // session insert
      const result = await service.verifyOtp({ email: 'jane@test.com', code: validCode }, undefined, undefined);
      expect(result.accessToken).toBeDefined();
      expect(result.user.email).toBe('jane@test.com');
      expect(result._refreshToken).toBeDefined();
      expect(mockEmailService.sendWelcomeEmail).toHaveBeenCalled();
      // Consent record insert happened
      const consentCall = mockPool.query.mock.calls.find(
        (c: any[]) => typeof c[0] === 'string' && c[0].includes('INSERT INTO iam.consent_records'),
      );
      expect(consentCall).toBeDefined();
    });
  });

  // ─── OTP code generation ───

  describe('OTP code generation', () => {
    it('generates a 6-digit zero-padded code', () => {
      const c1 = (service as any).generateOtpCode();
      const c2 = (service as any).generateOtpCode();
      expect(c1).toMatch(/^\d{6}$/);
      expect(c2).toMatch(/^\d{6}$/);
    });

    it('hashes the code with SHA-256 (never stored plaintext)', () => {
      const hash = (service as any).hashToken('123456');
      expect(hash).toHaveLength(64);
      expect(hash).toMatch(/^[a-f0-9]+$/);
    });
  });

  // ─── Unsubscribe ───

  describe('Unsubscribe', () => {
    it('processes a valid unsubscribe token and persists the opt-out', async () => {
      jwtService.verify.mockReturnValue({ email: 'user@test.com', purpose: 'unsubscribe' });
      mockPool.query.mockResolvedValue({ rows: [] });
      const result = await service.unsubscribe('valid.token', undefined, undefined);
      expect(result.success).toBe(true);
      expect(result.email).toBe('user@test.com');
      const insertCall = mockPool.query.mock.calls.find(
        (c: any[]) => typeof c[0] === 'string' && c[0].includes('INSERT INTO iam.marketing_unsubscribes'),
      );
      expect(insertCall).toBeDefined();
    });

    it('rejects an invalid unsubscribe token', async () => {
      jwtService.verify.mockImplementation(() => { throw new Error('invalid'); });
      await expect(service.unsubscribe('bad.token', undefined, undefined))
        .rejects.toThrow(BadRequestException);
    });

    it('rejects a token with the wrong purpose', async () => {
      jwtService.verify.mockReturnValue({ email: 'user@test.com', purpose: 'access' });
      await expect(service.unsubscribe('wrong.purpose', undefined, undefined))
        .rejects.toThrow(BadRequestException);
    });

    it('isUnsubscribed returns true when a record exists', async () => {
      mockPool.query.mockResolvedValue({ rows: [{ id: 'x' }] });
      expect(await service.isUnsubscribed('user@test.com')).toBe(true);
    });
  });

  // ─── Cookie configuration ───

  describe('Guest cookie configuration', () => {
    it('returns HttpOnly, SameSite=lax cookie with preview-duration maxAge', () => {
      const opts = service.getGuestCookieOptions();
      expect(opts.name).toBe('pat_guest_session');
      expect(opts.httpOnly).toBe(true);
      expect(opts.sameSite).toBe('lax');
      expect(opts.path).toBe('/');
      expect(opts.maxAge).toBe(300);
    });

    it('sets Secure=true in production', () => {
      (configService.get as jest.Mock).mockImplementation((key: string, def?: any) => {
        if (key === 'NODE_ENV') return 'production';
        return def;
      });
      const opts = service.getGuestCookieOptions();
      expect(opts.secure).toBe(true);
    });
  });
});
