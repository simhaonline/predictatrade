import { getRoleFromToken, getRoleFromTokenUnchecked } from '@/lib/auth';

function createMockJwt(role: string, expOffsetSeconds: number): string {
  const header = Buffer.from(JSON.stringify({ alg: 'HS256', typ: 'JWT' })).toString('base64url');
  const payload = Buffer.from(JSON.stringify({
    sub: 'test-user',
    email: 'test@test.com',
    role,
    exp: Math.floor(Date.now() / 1000) + expOffsetSeconds,
  })).toString('base64url');
  return `${header}.${payload}.signature`;
}

describe('Token Expiry Edge Case', () => {
  it('getRoleFromToken returns ADMIN for valid token', () => {
    const token = createMockJwt('ADMIN', 3600);
    expect(getRoleFromToken(token)).toBe('ADMIN');
  });

  it('getRoleFromToken returns null for expired token', () => {
    const token = createMockJwt('ADMIN', -10);
    expect(getRoleFromToken(token)).toBeNull();
  });

  it('getRoleFromTokenUnchecked returns ADMIN for expired token', () => {
    const token = createMockJwt('ADMIN', -10);
    expect(getRoleFromTokenUnchecked(token)).toBe('ADMIN');
  });

  it('getRoleFromTokenUnchecked returns SUPER_ADMIN for expired token', () => {
    const token = createMockJwt('SUPER_ADMIN', -10);
    expect(getRoleFromTokenUnchecked(token)).toBe('SUPER_ADMIN');
  });

  it('getRoleFromTokenUnchecked returns USER for expired token', () => {
    const token = createMockJwt('USER', -10);
    expect(getRoleFromTokenUnchecked(token)).toBe('USER');
  });

  it('getRoleFromTokenUnchecked returns null for invalid token', () => {
    expect(getRoleFromTokenUnchecked('invalid')).toBeNull();
    expect(getRoleFromTokenUnchecked(null)).toBeNull();
  });

  it('getRoleFromTokenUnchecked returns USER for token without role claim', () => {
    const header = Buffer.from(JSON.stringify({ alg: 'HS256', typ: 'JWT' })).toString('base64url');
    const payload = Buffer.from(JSON.stringify({ sub: 'test', exp: 9999999999 })).toString('base64url');
    const token = `${header}.${payload}.signature`;
    // When role claim is missing, the function defaults to 'USER'
    expect(getRoleFromTokenUnchecked(token)).toBe('USER');
  });

  it('fallback chain: checked → unchecked → USER preserves admin role for expired token', () => {
    const expiredAdminToken = createMockJwt('ADMIN', -10);
    const role = getRoleFromToken(expiredAdminToken) || getRoleFromTokenUnchecked(expiredAdminToken) || 'USER';
    expect(role).toBe('ADMIN');
  });

  it('fallback chain: checked → unchecked → USER preserves super_admin role for expired token', () => {
    const expiredSuperAdminToken = createMockJwt('SUPER_ADMIN', -10);
    const role = getRoleFromToken(expiredSuperAdminToken) || getRoleFromTokenUnchecked(expiredSuperAdminToken) || 'USER';
    expect(role).toBe('SUPER_ADMIN');
  });

  it('fallback chain returns USER for completely invalid token', () => {
    const role = getRoleFromToken('totally-invalid') || getRoleFromTokenUnchecked('totally-invalid') || 'USER';
    expect(role).toBe('USER');
  });

  it('fallback chain returns USER for null token', () => {
    const role = getRoleFromToken(null) || getRoleFromTokenUnchecked(null) || 'USER';
    expect(role).toBe('USER');
  });
});
