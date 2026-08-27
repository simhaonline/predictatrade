import { Injectable, CanActivate, ExecutionContext, ForbiddenException, SetMetadata } from '@nestjs/common';
import { Reflector } from '@nestjs/core';

/**
 * Canonical role enum. Mirrors the `iam.roles.name` values minted by the auth
 * service (auth.service.ts generateTokens) so RBAC decisions are made against a
 * single, typed source of truth instead of scattered string literals.
 */
export enum Role {
  USER = 'USER',
  SUPPORT = 'SUPPORT',
  ANALYST = 'ANALYST',
  AUDITOR = 'AUDITOR',
  TRADING_OPERATOR = 'TRADING_OPERATOR',
  RISK_MANAGER = 'RISK_MANAGER',
  ADMIN = 'ADMIN',
  SUPER_ADMIN = 'SUPER_ADMIN',
}

export const ROLES_KEY = 'roles';
export const Roles = (...roles: Role[]) => SetMetadata(ROLES_KEY, roles);

/**
 * Resource/permission enum. Operations that are sensitive but not purely
 * role-gated (e.g. finance_mutate, payout_approve) can require an explicit
 * capability. Permissions are optional: when a route declares no permissions
 * the guard is a pass-through, so existing admin-only flows keep working.
 */
export enum Permission {
  PAYOUT_APPROVE = 'payout:approve',
  PAYOUT_RECONCILE = 'payout:reconcile',
  COMMISSION_ADJUST = 'commission:adjust',
  COMMISSION_REVERSE = 'commission:reverse',
  LICENSE_MANAGE = 'license:manage',
  BILLING_MANAGE = 'billing:manage',
  USER_MANAGE = 'user:manage',
  RISK_MANAGE = 'risk:manage',
}

export const PERMISSIONS_KEY = 'permissions';
export const RequirePermissions = (...permissions: Permission[]) =>
  SetMetadata(PERMISSIONS_KEY, permissions);

@Injectable()
export class RolesGuard implements CanActivate {
  constructor(private reflector: Reflector) {}

  canActivate(context: ExecutionContext): boolean {
    const requiredRoles = this.reflector.getAllAndOverride<Role[]>(ROLES_KEY, [
      context.getHandler(),
      context.getClass(),
    ]);
    if (!requiredRoles || requiredRoles.length === 0) {
      return true;
    }

    const req = context.switchToHttp().getRequest();
    if (!req.user) {
      throw new ForbiddenException('Authentication required');
    }

    const userRole = (req.user.role as Role) ?? Role.USER;
    if (!requiredRoles.includes(userRole)) {
      throw new ForbiddenException(`Required role: ${requiredRoles.join(' | ')}`);
    }
    return true;
  }
}

@Injectable()
export class PermissionGuard implements CanActivate {
  constructor(private reflector: Reflector) {}

  canActivate(context: ExecutionContext): boolean {
    const required = this.reflector.getAllAndOverride<Permission[]>(PERMISSIONS_KEY, [
      context.getHandler(),
      context.getClass(),
    ]);
    if (!required || required.length === 0) {
      return true;
    }

    const req = context.switchToHttp().getRequest();
    if (!req.user) {
      throw new ForbiddenException('Authentication required');
    }

    const userPerms: Permission[] = Array.isArray(req.user.permissions)
      ? (req.user.permissions as Permission[])
      : [];
    const missing = required.filter((p) => !userPerms.includes(p));
    if (missing.length > 0) {
      throw new ForbiddenException(`Missing permission: ${missing.join(' ')}`);
    }
    return true;
  }
}
