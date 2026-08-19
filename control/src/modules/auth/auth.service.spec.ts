/**
 * Auth Service Security Regression Tests
 *
 * Tests verify core security properties required for production:
 *   - Refresh token entropy (384-bit random)
 *   - Token hashing (SHA-256)
 *   - Cookie configuration (HttpOnly, Secure, SameSite, Path)
 *   - Rotation (old token invalid, new token valid)
 *   - Reuse detection (previously rotated token revokes family)
 *   - Logout (revokes session)
 *   - User status check on refresh
 *   - Password reset one-time-use
 *   - Password reset enumeration protection
 *   - Reset URL construction (uses server config, not request headers)
 */

import { AuthService } from './auth.service';
import { UnauthorizedException, BadRequestException } from '@nestjs/common';

describe('AuthService Security', () => {
  let service: AuthService;
  let jwtService: any;
  let configService: any;
  let mockEmailService: any;
  let mockPool: any;

  beforeEach(() => {
    jwtService = {
      sign: jest.fn().mockReturnValue('mock.jwt.token'),
      verify: jest.fn(),
    };

    configService = {
      get: jest.fn((key: string, def?: any) => {
        const cfg: Record<string, any> = {
          NODE_ENV: 'test',
          JWT_SECRET: 'test_secret_for_testing_only',
          APP_FRONTEND_URL: 'https://test.example.com',
          CORS_ORIGINS: 'https://test.example.com,http://localhost:3000',
        };
        return cfg[key] ?? def;
      }),
    };

    mockEmailService = {
      sendPasswordResetEmail: jest.fn().mockResolvedValue(undefined),
    };

    mockPool = {
      connect: jest.fn(),
      query: jest.fn().mockResolvedValue({ rows: [] }),
    };

    service = new AuthService(jwtService, configService, mockPool, mockEmailService);
  });

  // ─── Refresh Token Entropy ───

  describe('Refresh token generation', () => {
    it('generates 384-bit (48-byte) cryptographically random tokens', () => {
      const token1 = (service as any).generateRefreshToken();
      const token2 = (service as any).generateRefreshToken();

      // base64url of 48 bytes = 64 characters (no padding)
      expect(token1).toHaveLength(64);
      expect(token2).toHaveLength(64);
      // Must be different (random, not predictable)
      expect(token1).not.toEqual(token2);
      // Must be base64url (no +, /, or =)
      expect(token1).toMatch(/^[A-Za-z0-9_-]+$/);
    });
  });

  // ─── Token Hashing ───

  describe('Token hashing', () => {
    it('hashes tokens with SHA-256 producing 64 hex chars', () => {
      const token = 'test_token_123';
      const hash = (service as any).hashToken(token);
      expect(hash).toHaveLength(64);
      expect(hash).toMatch(/^[a-f0-9]+$/);
      // Same input → same hash (deterministic for DB lookup)
      const hash2 = (service as any).hashToken(token);
      expect(hash).toEqual(hash2);
    });

    it('different tokens produce different hashes', () => {
      const hash1 = (service as any).hashToken('token_a');
      const hash2 = (service as any).hashToken('token_b');
      expect(hash1).not.toEqual(hash2);
    });
  });

  // ─── Cookie Configuration ───

  describe('Cookie configuration', () => {
    it('returns correct cookie attributes', () => {
      const opts = service.getRefreshCookieOptions();
      expect(opts.name).toBe('pat_refresh_token');
      expect(opts.httpOnly).toBe(true);
      expect(opts.secure).toBe(false); // test mode, not production
      expect(opts.sameSite).toBe('lax');
      expect(opts.path).toBe('/api/v1/auth');
      expect(opts.maxAge).toBe(7 * 24 * 60 * 60); // 7 days
      expect(opts.domain).toBeUndefined(); // host-only cookie
    });

    it('sets Secure=true in production', () => {
      (configService.get as jest.Mock).mockImplementation((key: string, def?: any) => {
        if (key === 'NODE_ENV') return 'production';
        return def;
      });
      const opts = service.getRefreshCookieOptions();
      expect(opts.secure).toBe(true);
    });

    it('clear cookie options match set cookie path', () => {
      const setOpts = service.getRefreshCookieOptions();
      const clearOpts = service.getClearCookieOptions();
      expect(clearOpts.name).toBe(setOpts.name);
      expect(clearOpts.path).toBe(setOpts.path);
      expect(clearOpts.domain).toBe(setOpts.domain);
      expect(clearOpts.httpOnly).toBe(true);
    });
  });

  // ─── Refresh: Rotation ───

  describe('Refresh rotation', () => {
    it('rejects unknown refresh token', async () => {
      const mockClient = {
        query: jest.fn().mockImplementation(async (sql: string) => {
          if (sql.trim().toUpperCase().startsWith('BEGIN') ||
              sql.trim().toUpperCase().startsWith('COMMIT') ||
              sql.trim().toUpperCase().startsWith('ROLLBACK') ||
              sql.trim().toUpperCase().startsWith('SELECT PG_ADVISORY')) {
            return { rows: [] };
          }
          if (sql.includes('FROM iam.sessions')) {
            return { rows: [] }; // no session found
          }
          return { rows: [] };
        }),
        release: jest.fn(),
      };
      mockPool.connect.mockResolvedValue(mockClient);

      await expect(service.refresh('unknown_token')).rejects.toThrow(UnauthorizedException);
    });

    it('rejects expired session', async () => {
      const mockClient = {
        query: jest.fn().mockImplementation(async (sql: string) => {
          if (sql.trim().toUpperCase().startsWith('BEGIN') ||
              sql.trim().toUpperCase().startsWith('SELECT PG_ADVISORY')) {
            return { rows: [] };
          }
          if (sql.includes('FROM iam.sessions')) {
            return { rows: [{
              id: 's1', user_id: 'u1', token_family: 'f1',
              revoked_at: null,
              expires_at: new Date(Date.now() - 1000), // expired
              refresh_expires_at: new Date(Date.now() - 1000),
              user_status: 'ACTIVE',
            }] };
          }
          return { rows: [] };
        }),
        release: jest.fn(),
      };
      mockPool.connect.mockResolvedValue(mockClient);

      await expect(service.refresh('expired_token')).rejects.toThrow(UnauthorizedException);
    });

    it('rejects refresh for inactive (suspended) user', async () => {
      const mockClient = {
        query: jest.fn().mockImplementation(async (sql: string) => {
          if (sql.trim().toUpperCase().startsWith('BEGIN') ||
              sql.trim().toUpperCase().startsWith('SELECT PG_ADVISORY')) {
            return { rows: [] };
          }
          if (sql.includes('FROM iam.sessions')) {
            return { rows: [{
              id: 's1', user_id: 'u1', token_family: 'f1',
              revoked_at: null,
              expires_at: new Date(Date.now() + 100000),
              refresh_expires_at: new Date(Date.now() + 100000),
              user_status: 'SUSPENDED',
            }] };
          }
          return { rows: [] };
        }),
        release: jest.fn(),
      };
      mockPool.connect.mockResolvedValue(mockClient);

      await expect(service.refresh('valid_token')).rejects.toThrow(UnauthorizedException);
    });

    it('returns new access and refresh tokens on successful rotation', async () => {
      jwtService.sign.mockReturnValue('new_access_token');
      const mockClient = {
        query: jest.fn().mockImplementation(async (sql: string) => {
          if (sql.trim().toUpperCase().startsWith('BEGIN') ||
              sql.trim().toUpperCase().startsWith('SELECT PG_ADVISORY')) {
            return { rows: [] };
          }
          if (sql.includes('FROM iam.sessions')) {
            return { rows: [{
              id: 's1', user_id: 'u1', token_family: 'f1',
              revoked_at: null,
              expires_at: new Date(Date.now() + 100000),
              refresh_expires_at: new Date(Date.now() + 100000),
              user_status: 'ACTIVE',
            }] };
          }
          if (sql.includes('FROM iam.users') && sql.includes('email')) {
            return { rows: [{ email: 'test@test.com' }] };
          }
          return { rows: [] };
        }),
        release: jest.fn(),
      };
      mockPool.connect.mockResolvedValue(mockClient);

      const result = await service.refresh('valid_token');

      expect(result.accessToken).toBe('new_access_token');
      expect(result.refreshToken).toBeDefined();
      expect(result.refreshToken).toHaveLength(64); // 384-bit base64url
      expect(result.refreshToken).not.toBe(result.accessToken);
    });
  });

  // ─── Refresh: Reuse Detection ───

  describe('Refresh reuse detection', () => {
    it('detects reuse of a previously-rotated token and revokes family', async () => {
      const mockClient = {
        queries: [] as any[],
        query: jest.fn().mockImplementation(async (sql: string, params?: any[]) => {
          mockClient.queries.push({ sql, params });
          if (sql.trim().toUpperCase().startsWith('BEGIN') ||
              sql.trim().toUpperCase().startsWith('COMMIT') ||
              sql.trim().toUpperCase().startsWith('SELECT PG_ADVISORY') ||
              sql.includes('INSERT INTO iam.login_events')) {
            return { rows: [] };
          }
          if (sql.includes('FROM iam.sessions')) {
            return { rows: [{
              id: 's1', user_id: 'u1', token_family: 'family-compromised',
              revoked_at: new Date(), // already revoked — reuse!
              expires_at: new Date(Date.now() + 100000),
              refresh_expires_at: new Date(Date.now() + 100000),
              user_status: 'ACTIVE',
            }] };
          }
          return { rows: [] };
        }),
        release: jest.fn(),
      };
      mockPool.connect.mockResolvedValue(mockClient);

      await expect(service.refresh('reused_token')).rejects.toThrow(UnauthorizedException);

      // Verify family revocation was executed
      const familyRevocationQuery = mockClient.queries.find(
        (q: any) => q.sql.includes('UPDATE iam.sessions SET revoked_at') &&
             q.sql.includes('token_family')
      );
      expect(familyRevocationQuery).toBeDefined();
    });
  });

  // ─── Logout ───

  describe('Logout', () => {
    it('revokes the specific session when token provided', async () => {
      mockPool.query.mockResolvedValue({ rows: [] });
      await service.logout('user-1', 'token_to_revoke');

      const revokeCall = mockPool.query.mock.calls.find(
        (call: any[]) => typeof call[0] === 'string' &&
          call[0].includes('UPDATE iam.sessions SET revoked_at')
      );
      expect(revokeCall).toBeDefined();
      expect(revokeCall[0]).toContain('refresh_token_hash');
      expect(revokeCall[0]).toContain('revoked_at IS NULL');
    });

    it('revokes all sessions when no token provided', async () => {
      mockPool.query.mockResolvedValue({ rows: [] });
      await service.logout('user-1');

      const revokeCall = mockPool.query.mock.calls.find(
        (call: any[]) => typeof call[0] === 'string' &&
          call[0].includes('UPDATE iam.sessions SET revoked_at')
      );
      expect(revokeCall).toBeDefined();
      expect(revokeCall[0]).not.toContain('refresh_token_hash');
    });
  });

  // ─── Password Reset ───

  describe('Password reset token', () => {
    it('rejects already-used reset token', async () => {
      jwtService.verify.mockReturnValue({
        sub: 'user-1', purpose: 'password_reset', jti: 'jti-123',
      });
      const mockClient = {
        query: jest.fn().mockImplementation(async (sql: string) => {
          if (sql.trim().toUpperCase().startsWith('BEGIN') ||
              sql.trim().toUpperCase().startsWith('SELECT PG_ADVISORY')) {
            return { rows: [] };
          }
          if (sql.includes('SELECT id FROM iam.login_events')) {
            return { rows: [{ id: 'event-1' }] }; // already used
          }
          return { rows: [] };
        }),
        release: jest.fn(),
      };
      mockPool.connect.mockResolvedValue(mockClient);

      await expect(service.resetPassword('used_token', 'newpassword123'))
        .rejects.toThrow(BadRequestException);
    });

    it('rejects reset for inactive user', async () => {
      jwtService.verify.mockReturnValue({
        sub: 'user-1', purpose: 'password_reset', jti: 'jti-456',
      });
      const mockClient = {
        query: jest.fn().mockImplementation(async (sql: string) => {
          if (sql.trim().toUpperCase().startsWith('BEGIN') ||
              sql.trim().toUpperCase().startsWith('SELECT PG_ADVISORY')) {
            return { rows: [] };
          }
          if (sql.includes('SELECT id FROM iam.login_events')) {
            return { rows: [] }; // not used yet
          }
          if (sql.includes('SELECT status FROM iam.users')) {
            return { rows: [{ status: 'SUSPENDED' }] };
          }
          return { rows: [] };
        }),
        release: jest.fn(),
      };
      mockPool.connect.mockResolvedValue(mockClient);

      await expect(service.resetPassword('valid_token', 'newpassword123'))
        .rejects.toThrow(BadRequestException);
    });

    it('rejects token with wrong purpose', async () => {
      jwtService.verify.mockReturnValue({
        sub: 'user-1', purpose: 'access', jti: 'jti-789',
      });

      await expect(service.resetPassword('wrong_purpose_token', 'newpassword123'))
        .rejects.toThrow(BadRequestException);
    });

    it('rejects invalid/expired JWT', async () => {
      jwtService.verify.mockImplementation(() => {
        throw new Error('jwt expired');
      });

      await expect(service.resetPassword('expired_token', 'newpassword123'))
        .rejects.toThrow(BadRequestException);
    });
  });

  // ─── Password Reset: Email Enumeration ───

  describe('Password reset enumeration protection', () => {
    it('returns generic response for existing user', async () => {
      mockPool.query.mockResolvedValue({ rows: [{ id: 'user-1' }] });
      const result = await service.forgotPassword('existing@test.com');
      expect(result.message).toContain('If an account exists');
      expect(mockEmailService.sendPasswordResetEmail).toHaveBeenCalled();
    });

    it('returns same generic response for non-existing user', async () => {
      mockPool.query.mockResolvedValue({ rows: [] });
      const result = await service.forgotPassword('nonexistent@test.com');
      expect(result.message).toContain('If an account exists');
      expect(mockEmailService.sendPasswordResetEmail).not.toHaveBeenCalled();
    });

    it('returns generic response even when email delivery fails', async () => {
      mockPool.query.mockResolvedValue({ rows: [{ id: 'user-1' }] });
      mockEmailService.sendPasswordResetEmail.mockRejectedValue(new Error('SMTP error'));
      const result = await service.forgotPassword('existing@test.com');
      expect(result.message).toContain('If an account exists');
    });
  });

  // ─── Reset URL Construction ───

  describe('Reset URL construction', () => {
    it('uses APP_FRONTEND_URL from config, not from request headers', async () => {
      mockPool.query.mockResolvedValue({ rows: [{ id: 'user-1' }] });
      await service.forgotPassword('test@test.com');

      const emailCall = mockEmailService.sendPasswordResetEmail.mock.calls[0][0];
      expect(emailCall.resetUrl).toContain('https://test.example.com/reset-password?token=');
      // Token must be URL-encoded
      expect(emailCall.resetUrl).toContain('mock.jwt.token');
    });
  });
});
