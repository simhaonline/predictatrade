import { AdminGuard } from './admin.guard';
import { jest } from '@jest/globals';
import { ExecutionContext, ForbiddenException } from '@nestjs/common';

describe('AdminGuard', () => {
  let guard: AdminGuard;

  beforeEach(() => {
    guard = new AdminGuard();
  });

  function createContext(user: any): ExecutionContext {
    return {
      switchToHttp: () => ({ getRequest: () => ({ user }) }),
    } as ExecutionContext;
  }

  it('SUPER_ADMIN → allowed', () => {
    const ctx = createContext({ role: 'SUPER_ADMIN', sub: 'user-1' });
    expect(guard.canActivate(ctx)).toBe(true);
  });

  it('ADMIN → allowed', () => {
    const ctx = createContext({ role: 'ADMIN', sub: 'user-2' });
    expect(guard.canActivate(ctx)).toBe(true);
  });

  it('ordinary USER role → denied', () => {
    const ctx = createContext({ role: 'USER', sub: 'user-3' });
    expect(() => guard.canActivate(ctx)).toThrow(ForbiddenException);
  });

  it('unauthenticated (no user) → denied', () => {
    const ctx = createContext(null);
    expect(() => guard.canActivate(ctx)).toThrow(ForbiddenException);
  });

  it('undefined role → denied', () => {
    const ctx = createContext({ sub: 'user-5' });
    expect(() => guard.canActivate(ctx)).toThrow(ForbiddenException);
  });
});
