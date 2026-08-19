import { isAdminRole, homeRouteForRole, panelLabelForRole } from '@/lib/roles';

describe('Canonical Role Resolution', () => {
  describe('isAdminRole', () => {
    it('returns true for ADMIN', () => {
      expect(isAdminRole('ADMIN')).toBe(true);
    });
    it('returns true for SUPER_ADMIN', () => {
      expect(isAdminRole('SUPER_ADMIN')).toBe(true);
    });
    it('returns false for USER', () => {
      expect(isAdminRole('USER')).toBe(false);
    });
    it('returns false for null', () => {
      expect(isAdminRole(null)).toBe(false);
    });
    it('returns false for undefined', () => {
      expect(isAdminRole(undefined)).toBe(false);
    });
    it('returns false for unknown roles', () => {
      expect(isAdminRole('ANALYST')).toBe(false);
      expect(isAdminRole('SUPPORT')).toBe(false);
    });
  });

  describe('homeRouteForRole', () => {
    it('returns /admin/dashboard for ADMIN', () => {
      expect(homeRouteForRole('ADMIN')).toBe('/admin/dashboard');
    });
    it('returns /admin/dashboard for SUPER_ADMIN', () => {
      expect(homeRouteForRole('SUPER_ADMIN')).toBe('/admin/dashboard');
    });
    it('returns /dashboard/live for USER', () => {
      expect(homeRouteForRole('USER')).toBe('/dashboard/live');
    });
    it('returns /dashboard/live for null', () => {
      expect(homeRouteForRole(null)).toBe('/dashboard/live');
    });
  });

  describe('panelLabelForRole', () => {
    it('returns "Admin Panel" for ADMIN', () => {
      expect(panelLabelForRole('ADMIN')).toBe('Admin Panel');
    });
    it('returns "Admin Panel" for SUPER_ADMIN', () => {
      expect(panelLabelForRole('SUPER_ADMIN')).toBe('Admin Panel');
    });
    it('returns "User Panel" for USER', () => {
      expect(panelLabelForRole('USER')).toBe('User Panel');
    });
  });
});
