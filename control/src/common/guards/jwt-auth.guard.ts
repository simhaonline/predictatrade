import { Injectable, CanActivate, ExecutionContext, UnauthorizedException, Logger } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import * as jwt from 'jsonwebtoken';

@Injectable()
export class JwtAuthGuard implements CanActivate {
  private readonly logger = new Logger(JwtAuthGuard.name);

  constructor(private readonly config: ConfigService) {}

  canActivate(context: ExecutionContext): boolean {
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
      const payload = jwt.verify(token, secret) as { sub: string; email?: string; purpose?: string };

      // Reject tokens with a non-access purpose (e.g. password_reset)
      if (payload.purpose && payload.purpose !== 'access') {
        throw new UnauthorizedException('Invalid token type');
      }

      req.user = payload;
      return true;
    } catch (err) {
      if (err instanceof UnauthorizedException) throw err;
      throw new UnauthorizedException('Invalid token');
    }
  }
}
