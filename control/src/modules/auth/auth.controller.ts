import {
  Body, Controller, Post, UseGuards, Get, Req, Res, BadRequestException,
} from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { Throttle } from '@nestjs/throttler';
import { Request, Response } from 'express';
import { AuthService } from './auth.service';
import {
  RegisterDto, LoginDto, MfaSetupDto, VerifyOtpDto, ForgotPasswordDto, ResetPasswordDto,
} from './dto/auth.dto';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { CurrentUser } from '../../common/decorators/current-user.decorator';

@Controller('auth')
export class AuthController {
  constructor(
    private authService: AuthService,
    private config: ConfigService,
  ) {}

  /** Set the HttpOnly refresh-token cookie on the response. */
  private setRefreshCookie(res: Response, refreshToken: string): void {
    const opts = this.authService.getRefreshCookieOptions();
    res.cookie(opts.name, refreshToken, {
      httpOnly: opts.httpOnly,
      secure: opts.secure,
      sameSite: opts.sameSite,
      path: opts.path,
      maxAge: opts.maxAge,
      ...(opts.domain ? { domain: opts.domain } : {}),
    });
  }

  /** Clear the HttpOnly refresh-token cookie. */
  private clearRefreshCookie(res: Response): void {
    const opts = this.authService.getClearCookieOptions();
    res.clearCookie(opts.name, {
      path: opts.path,
      httpOnly: opts.httpOnly,
      ...(opts.domain ? { domain: opts.domain } : {}),
    });
  }

  /**
   * Validate the Origin/Referer header for cookie-authenticated state-changing
   * endpoints (refresh, logout). This provides CSRF defense in combination
   * with SameSite=Lax. Returns true if the request is from an allowed origin.
   */
  private validateOrigin(req: Request): boolean {
    const corsOrigins = (this.config.get<string>('CORS_ORIGINS') || 'http://localhost:3000,http://localhost:4600')
      .split(',')
      .map(s => s.trim().toLowerCase());
    const origin = (req.headers.origin || '').toLowerCase();
    const referer = (req.headers.referer || '').toLowerCase();

    // If Origin header is present, it must match an allowed origin
    if (origin) {
      return corsOrigins.some(allowed => origin === allowed.toLowerCase());
    }

    // If no Origin but Referer is present, extract the origin and validate
    if (referer) {
      try {
        const refererOrigin = new URL(referer).origin.toLowerCase();
        return corsOrigins.some(allowed => refererOrigin === allowed.toLowerCase());
      } catch {
        return false;
      }
    }

    // Same-site requests (no Origin/Referer) are allowed.
    // SameSite=Lax cookie attribute already prevents most cross-site requests.
    return true;
  }

  /** Set no-store cache headers on auth responses containing sensitive data. */
  private setNoStoreHeaders(res: Response): void {
    res.setHeader('Cache-Control', 'no-store, no-cache, must-revalidate, private');
    res.setHeader('Pragma', 'no-cache');
    res.setHeader('Expires', '0');
  }

  @Throttle({ default: { limit: 5, ttl: 60_000 } }) // 5 register attempts per minute per IP
  @Post('register')
  async register(@Body() dto: RegisterDto, @Res({ passthrough: true }) res: Response) {
    this.setNoStoreHeaders(res);
    const result = await this.authService.register(dto);
    this.setRefreshCookie(res, result._refreshToken);
    // Strip internal _refreshToken from the JSON response
    const { _refreshToken: _rt, ...publicResult } = result;
    void _rt;
    return publicResult;
  }

  @Throttle({ default: { limit: 10, ttl: 60_000 } }) // 10 login attempts per minute per IP
  @Post('login')
  async login(@Body() dto: LoginDto, @Res({ passthrough: true }) res: Response) {
    this.setNoStoreHeaders(res);
    const result = await this.authService.login(dto);
    // If MFA required, don't set cookie yet
    if ('mfaRequired' in result) {
      return result;
    }
    this.setRefreshCookie(res, result._refreshToken);
    const { _refreshToken: _rt, ...publicResult } = result;
    void _rt;
    return publicResult;
  }

  @Throttle({ default: { limit: 10, ttl: 60_000 } }) // 10 OTP attempts per minute per IP
  @Post('verify-otp')
  async verifyOtp(@Body() dto: VerifyOtpDto, @Res({ passthrough: true }) res: Response) {
    this.setNoStoreHeaders(res);
    const result = await this.authService.verifyOtp(dto);
    this.setRefreshCookie(res, result._refreshToken);
    // Strip internal _refreshToken from the JSON response
    const { _refreshToken: _rt, ...publicResult } = result;
    void _rt;
    return publicResult;
  }

  @Throttle({ default: { limit: 20, ttl: 60_000 } }) // 20 refresh attempts per minute per IP
  @Post('refresh')
  async refresh(@Req() req: Request, @Res({ passthrough: true }) res: Response) {
    this.setNoStoreHeaders(res);

    // CSRF defense: validate Origin/Referer for cookie-authenticated requests
    if (!this.validateOrigin(req)) {
      throw new BadRequestException('Invalid request origin');
    }

    // Read refresh token from HttpOnly cookie — not from request body
    const cookieName = this.config.get<string>('AUTH_REFRESH_COOKIE_NAME', 'pat_refresh_token');
    const rawRefreshToken = req.cookies?.[cookieName];

    if (!rawRefreshToken) {
      throw new BadRequestException('No refresh token');
    }

    try {
      const result = await this.authService.refresh(rawRefreshToken);
      // Set the rotated refresh token as a new HttpOnly cookie
      this.setRefreshCookie(res, result.refreshToken);
      // Return only the access token — never the refresh token
      return { accessToken: result.accessToken };
    } catch (err) {
      // Clear the invalid cookie on refresh failure
      this.clearRefreshCookie(res);
      throw err;
    }
  }

  @UseGuards(JwtAuthGuard)
  @Get('me')
  async me(@CurrentUser('sub') userId: string) {
    return this.authService.getProfile(userId);
  }

  @UseGuards(JwtAuthGuard)
  @Throttle({ default: { limit: 20, ttl: 60_000 } })
  @Post('logout')
  async logout(
    @CurrentUser('sub') userId: string,
    @Req() req: Request,
    @Res({ passthrough: true }) res: Response,
  ) {
    this.setNoStoreHeaders(res);

    // CSRF defense: validate Origin/Referer for cookie-authenticated requests
    if (!this.validateOrigin(req)) {
      throw new BadRequestException('Invalid request origin');
    }

    const cookieName = this.config.get<string>('AUTH_REFRESH_COOKIE_NAME', 'pat_refresh_token');
    const rawRefreshToken = req.cookies?.[cookieName];
    await this.authService.logout(userId, rawRefreshToken);
    this.clearRefreshCookie(res);
    return { success: true };
  }

  @UseGuards(JwtAuthGuard)
  @Post('mfa/setup')
  async setupMfa(@CurrentUser('sub') userId: string) {
    return this.authService.setupMfa(userId);
  }

  @UseGuards(JwtAuthGuard)
  @Post('mfa/verify')
  async verifyMfa(@CurrentUser('sub') userId: string, @Body() dto: MfaSetupDto) {
    return this.authService.verifyMfa(userId, dto.code);
  }

  @Throttle({ default: { limit: 5, ttl: 60_000 } }) // 5 forgot-password per minute per IP
  @Post('forgot')
  async forgotPassword(@Body() dto: ForgotPasswordDto, @Res({ passthrough: true }) res: Response) {
    this.setNoStoreHeaders(res);
    return this.authService.forgotPassword(dto.email);
  }

  @Throttle({ default: { limit: 5, ttl: 60_000 } }) // 5 reset attempts per minute per IP
  @Post('reset')
  async resetPassword(@Body() dto: ResetPasswordDto, @Res({ passthrough: true }) res: Response) {
    this.setNoStoreHeaders(res);
    const result = await this.authService.resetPassword(dto.token, dto.password);
    // Clear the refresh cookie since all sessions were revoked
    this.clearRefreshCookie(res);
    return result;
  }
}
