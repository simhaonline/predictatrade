import { Injectable, CanActivate, ExecutionContext, UnauthorizedException, ForbiddenException, Logger, Inject, Optional } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import * as jwt from 'jsonwebtoken';
import { Pool } from 'pg';
import { DB_POOL } from '../database.module';

@Injectable()
export class JwtAuthGuard implements CanActivate {
  private readonly logger = new Logger(JwtAuthGuard.name);

  constructor(
    private readonly config: ConfigService,
    @Optional() @Inject(DB_POOL) private readonly db?: Pool,
  ) {}

  async canActivate(context: ExecutionContext): Promise<boolean> {
    const req = context.switchToHttp().getRequest();
    const auth = req.headers.authorization;

    // F1: accept the access token from an HttpOnly cookie as well as the
    // Authorization header, so the SPA can avoid exposing the token to JS.
    let token: string | undefined;
    if (auth && auth.startsWith('Bearer ')) {
      token = auth.substring(7);
    } else {
      const cookieName = this.config.get<string>('AUTH_ACCESS_COOKIE_NAME', 'pat_access_token');
      const cookieToken = req.cookies?.[cookieName];
      if (cookieToken && typeof cookieToken === 'string') {
        token = cookieToken;
      }
    }

    if (!token) {
      throw new UnauthorizedException('Missing bearer token');
    }
    try {
      const secret = this.config.get<string>('JWT_SECRET');
      if (!secret) {
        throw new UnauthorizedException('JWT_SECRET not configured');
      }
      const payload = jwt.verify(token, secret) as { sub: string; email?: string; role?: string; permissions?: string[]; mfaEnrollmentRequired?: boolean; purpose?: string };

      // Reject tokens with a non-access purpose (e.g. password_reset)
      if (payload.purpose && payload.purpose !== 'access') {
        throw new UnauthorizedException('Invalid token type');
      }

      req.user = payload;

      // R1 (RBAC hardening): privileged-role access tokens are re-validated
      // against the DB so suspension / demotion takes effect immediately
      // instead of at token expiry (max 1h). Only applies to ADMIN /
      // SUPER_ADMIN — a handful of accounts — so the per-request query is
      // negligible. Any DB error FAILS CLOSED for privileged tokens.
      if (payload.role === 'ADMIN' || payload.role === 'SUPER_ADMIN') {
        await this.validatePrivilegedUser(req);
      }

      // AUTH-1 (hardened): privileged roles must have MFA enabled. Login mints
      // `mfaEnrollmentRequired: true` into the access token when a privileged
      // account lacks an enabled TOTP method (auth.service generateTokens).
      // Enrollment / logout / refresh stay reachable so the operator can
      // complete enrollment; every other endpoint 403s until MFA is enabled.
      const PRIVILEGED_ROLES = new Set(['ADMIN', 'SUPER_ADMIN', 'OPERATOR', 'RISK_MANAGER', 'TRADING_OPERATOR']);
      if (payload.mfaEnrollmentRequired === true && PRIVILEGED_ROLES.has(String(payload.role))) {
        const path: string = req.originalUrl || req.url || '';
        const EXEMPT = [/^\/api\/v1\/auth\/mfa(\/|$)/, /^\/api\/v1\/auth\/logout(\/|$)/, /^\/api\/v1\/auth\/refresh(\/|$)/];
        if (!EXEMPT.some((re) => re.test(path))) {
          this.logger.warn(`MFA gate: blocked ${payload.email ?? payload.sub} at ${path}`);
          throw new ForbiddenException('MFA enrollment required before accessing privileged resources');
        }
      }
      return true;
    } catch (err) {
      if (err instanceof UnauthorizedException) throw err;
      throw new UnauthorizedException('Invalid token');
    }
  }

  /**
   * R1: privileged-role tokens are checked against iam.users.status and the
   * active role membership. Suspended / deleted / demoted accounts are
   * rejected on the spot. DB failure fails CLOSED for privileged roles.
   */
  private async validatePrivilegedUser(req: { user?: { sub?: string; role?: string; email?: string } }): Promise<void> {
    const pool = this.db;
    const user = req.user as { sub?: string; role?: string; email?: string } | undefined;
    if (!pool || !user?.sub) {
      throw new UnauthorizedException('Privileged session validation unavailable');
    }
    try {
      const res = await pool.query(
        `SELECT u.status, r.name AS role_name
         FROM iam.users u
         LEFT JOIN iam.memberships m ON m.user_id = u.id
         LEFT JOIN iam.roles r ON r.id = m.role_id
         WHERE u.id = $1 AND u.deleted_at IS NULL`,
        [user.sub],
      );
      if (res.rows.length === 0) {
        throw new UnauthorizedException('Account not found');
      }
      const row = res.rows[0];
      if (row.status !== 'ACTIVE') {
        throw new UnauthorizedException('Account is not active');
      }
      const activeRole = res.rows.map((r) => r.role_name).filter(Boolean);
      if (!activeRole.includes(String(user.role))) {
        throw new ForbiddenException('Privileged role no longer held');
      }
    } catch (err) {
      if (err instanceof UnauthorizedException || err instanceof ForbiddenException) throw err;
      // Fail closed: cannot confirm privileged status right now
      throw new UnauthorizedException('Privileged session could not be validated');
    }
  }
}
