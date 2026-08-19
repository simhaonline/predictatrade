import { getAccessToken, setAccessToken, clearAccessToken, isAdmin } from '@/lib/auth';

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

  it('isAdmin returns true for ADMIN role', () => {
    expect(isAdmin({ id: '1', email: 'a@b.com', role: 'ADMIN' })).toBe(true);
  });

  it('isAdmin returns false for USER role', () => {
    expect(isAdmin({ id: '1', email: 'a@b.com', role: 'USER' })).toBe(false);
  });
});
