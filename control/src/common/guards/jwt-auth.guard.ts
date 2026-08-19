import { Injectable, CanActivate, ExecutionContext, UnauthorizedException, Logger } from '@nestjs/common';
import * as jwt from 'jsonwebtoken';

// DEV_JWT_SECRET removed — production must set JWT_SECRET env var

@Injectable()
export class JwtAuthGuard implements CanActivate {
  private readonly logger = new Logger(JwtAuthGuard.name);

  canActivate(context: ExecutionContext): boolean {
    const req = context.switchToHttp().getRequest();
    const auth = req.headers.authorization;
    if (!auth || !auth.startsWith('Bearer ')) {
      throw new UnauthorizedException('Missing bearer token');
    }
    const token = auth.substring(7);
    try {
      const secret = process.env.JWT_SECRET;
    if (!secret) {
      throw new UnauthorizedException('JWT_SECRET not configured');
    }
      const payload = jwt.verify(token, secret) as { sub: string; email?: string; purpose?: string };

      // Reject tokens with a password_reset purpose — those are not access tokens
      if (payload.purpose && payload.purpose !== undefined && payload.purpose !== 'access') {
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
