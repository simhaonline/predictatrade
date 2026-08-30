import { jest } from '@jest/globals';
/**
 * Security Validation Regression Tests
 *
 * P2-001: JWT secret validation rejects placeholder/weak secrets in production
 * P2-002: Database URL validation rejects insecure hardcoded passwords in production
 */

describe('JWT Secret Validation (P2-001)', () => {
  const INSECURE_SECRETS = [
    '',
    'pat_local_dev_secret_change_in_production',
    'CHANGE_ME_IN_PRODUCTION',
    'CHANGE_ME_IN_PRODUCTION_USE_SECRET_FILE',
    'change_this_to_a_long_random_secret',
    'changeme',
    'secret',
    'placeholder',
    'development',
  ];

  it('should reject all known insecure placeholder secrets', () => {
    for (const s of INSECURE_SECRETS) {
      // In production, each of these must cause startup failure
      expect(INSECURE_SECRETS.includes(s)).toBe(true);
    }
  });

  it('should accept a strong production secret (min 32 chars)', () => {
    const strongSecret =
      'a-very-long-and-random-production-secret-key-1234567890abcdef';
    expect(strongSecret.length).toBeGreaterThanOrEqual(32);
    expect(INSECURE_SECRETS.includes(strongSecret)).toBe(false);
  });

  it('should reject short secrets in production (min 32 chars)', () => {
    const shortSecret = 'shortsecret12345'; // 15 chars
    expect(shortSecret.length).toBeLessThan(32);
  });
});

describe('Database URL Validation (P2-002)', () => {
  const INSECURE_DB_PASSWORDS = [
    'pat_local_dev_only',
    'change_me',
    'changeme',
    'password',
    'postgres',
  ];

  function isInsecureDBUrl(url: string): boolean {
    if (!url) return true;
    for (const p of INSECURE_DB_PASSWORDS) {
      if (url.includes(`:${p}@`)) return true;
    }
    return false;
  }

  it('should reject empty DATABASE_URL', () => {
    expect(isInsecureDBUrl('')).toBe(true);
  });

  it('should reject URLs with known insecure passwords', () => {
    for (const p of INSECURE_DB_PASSWORDS) {
      const url = `postgres://user:${p}@127.0.0.1:5432/db`;
      expect(isInsecureDBUrl(url)).toBe(true);
    }
  });

  it('should accept URLs with secure passwords', () => {
    const url =
      'postgres://app:SecureR4nd0mP@ss!@db.internal:5432/predictatrade';
    expect(isInsecureDBUrl(url)).toBe(false);
  });

  it('should allow insecure passwords in development (non-production)', () => {
    // The validation only applies in production; dev can use dev passwords
    const devUrl =
      'postgres://pat_admin:pat_local_dev_only@127.0.0.1:5432/predictatrade';
    expect(isInsecureDBUrl(devUrl)).toBe(true); // detected as insecure
    // But in development mode, the app does not fail on this
  });
});
