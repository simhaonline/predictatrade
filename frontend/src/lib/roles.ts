/**
 * Canonical role model for Predict-A-Trade.
 *
 * Backend roles come from the JWT `role` claim, populated from `iam.roles` table.
 * Known roles: SUPER_ADMIN, ADMIN, RISK_MANAGER, TRADING_OPERATOR, SUPPORT, ANALYST, AUDITOR, USER.
 *
 * AdminGuard (backend) accepts: ADMIN, SUPER_ADMIN.
 * The frontend must match this exactly.
 */

export type Role = string;

/** All roles the backend AdminGuard considers privileged. */
const ADMIN_ROLES: Role[] = ['ADMIN', 'SUPER_ADMIN'];

/** Returns true if the role should see the Admin Panel. */
export function isAdminRole(role: Role | null | undefined): boolean {
  if (!role) return false;
  return ADMIN_ROLES.includes(role);
}

/** Returns true if the role is a normal user (non-admin). */
export function isUserRole(role: Role | null | undefined): boolean {
  if (!role) return false;
  return !isAdminRole(role);
}

/** Canonical home route for a given role. */
export function homeRouteForRole(role: Role | null | undefined): string {
  if (isAdminRole(role)) return '/admin/dashboard';
  return '/dashboard/live';
}

/** Panel label for sidebar footer. */
export function panelLabelForRole(role: Role | null | undefined): string {
  return ''; // Panel label removed — no UserPanel/AdminPanel text
}
