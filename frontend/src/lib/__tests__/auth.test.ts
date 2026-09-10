import { getAccessToken, setAccessToken, clearAccessToken } from '@/lib/auth';
import { isAdminRole } from '@/lib/roles';

describe('auth helpers', () => {
  beforeEach(() => {
    clearAccessToken();
  });

  it('getAccessToken returns null initially', () => {
    expect(getAccessToken()).toBeNull();
  });

  it('setAccessToken stores token', () => {
    setAccessToken('test-token');
    expect(getAccessToken()).toBe('test-token');
  });

  it('clearAccessToken removes token', () => {
    setAccessToken('test-token');
    clearAccessToken();
    expect(getAccessToken()).toBeNull();
  });

  // Role checks live in lib/roles (canonical); auth.ts no longer carries a
  // duplicate deprecated helper.
  it('isAdminRole returns true for ADMIN role', () => {
    expect(isAdminRole('ADMIN')).toBe(true);
    expect(isAdminRole('SUPER_ADMIN')).toBe(true);
  });

  it('isAdminRole returns false for USER role', () => {
    expect(isAdminRole('USER')).toBe(false);
  });
});