import {
  Body, Controller, Post, Get, Req, Res,
} from '@nestjs/common';
import { Throttle } from '@nestjs/throttler';
import { Request, Response } from 'express';
import { GuestPreviewService } from './guest-preview.service';
import {
  GuestRegisterDto, GuestOtpVerifyDto, GuestOtpResendDto, GuestUnsubscribeDto,
} from './dto/guest-preview.dto';

@Controller('guest')
export class GuestPreviewController {
  constructor(private guestService: GuestPreviewService) {}

  /** Extract client IP (trust the first hop behind Nginx) and user-agent. */
  private clientMeta(req: Request): { ip: string | undefined; ua: string | undefined } {
    const ip = (req.headers['x-forwarded-for'] as string | undefined)?.split(',')[0]?.trim() || req.ip || undefined;
    const ua = req.headers['user-agent'] || undefined;
    return { ip, ua };
  }

  private setGuestCookie(res: Response, guestToken: string): void {
    const opts = this.guestService.getGuestCookieOptions();
    res.cookie(opts.name, guestToken, {
      httpOnly: opts.httpOnly,
      secure: opts.secure,
      sameSite: opts.sameSite,
      path: opts.path,
      maxAge: opts.maxAge,
      ...(opts.domain ? { domain: opts.domain } : {}),
    });
  }

  private clearGuestCookie(res: Response): void {
    const opts = this.guestService.getClearGuestCookieOptions();
    res.clearCookie(opts.name, { path: opts.path, httpOnly: opts.httpOnly, ...(opts.domain ? { domain: opts.domain } : {}) });
  }

  private setRefreshCookie(res: Response, refreshToken: string): void {
    // Reuse the same secure cookie attributes as the main auth refresh cookie.
    res.cookie('pat_refresh_token', refreshToken, {
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production',
      sameSite: (process.env.AUTH_REFRESH_COOKIE_SAMESITE as 'strict' | 'lax' | 'none') || 'lax',
      path: process.env.AUTH_REFRESH_COOKIE_PATH || '/api/v1/auth',
      maxAge: 7 * 24 * 60 * 60,
      ...(process.env.AUTH_REFRESH_COOKIE_DOMAIN ? { domain: process.env.AUTH_REFRESH_COOKIE_DOMAIN } : {}),
    });
  }

  private setNoStoreHeaders(res: Response): void {
    res.setHeader('Cache-Control', 'no-store, no-cache, must-revalidate, private');
    res.setHeader('Pragma', 'no-cache');
    res.setHeader('Expires', '0');
  }

  /**
   * Issue a short-lived anonymous guest session token (server-side expiry).
   * The token is set as an HttpOnly cookie; the JSON body is for the client
   * countdown display only (never the source of truth).
   */
  @Throttle({ default: { limit: 10, ttl: 60_000 } })
  @Post('session')
  async issueSession(@Res({ passthrough: true }) res: Response) {
    this.setNoStoreHeaders(res);
    const result = await this.guestService.issueGuestSession();
    this.setGuestCookie(res, result.guestToken);
    return {
      guestToken: result.guestToken,
      expiresAt: result.expiresAt,
      previewSeconds: result.previewSeconds,
    };
  }

  /**
   * Server-authoritative guest status. Returns locked=true when the
   * server-side session has expired (cookie cleared / incognito → no token →
   * locked, and a fresh session again expires after PREVIEW_SECONDS).
   */
  @Get('status')
  async getStatus(@Req() req: Request) {
    const cookieName = process.env.GUEST_COOKIE_NAME || 'pat_guest_session';
    const rawToken = req.cookies?.[cookieName] as string | undefined;
    return this.guestService.getGuestStatus(rawToken);
  }

  /**
   * Register a new guest (passwordless). Validates inputs + required consents,
   * stores a hashed OTP challenge, and emails the 6-digit code.
   * Generic response — never reveals whether an email already exists.
   */
  @Throttle({ default: { limit: 5, ttl: 60_000 } })
  @Post('register')
  async register(@Body() dto: GuestRegisterDto, @Req() req: Request, @Res({ passthrough: true }) res: Response) {
    this.setNoStoreHeaders(res);
    const { ip, ua } = this.clientMeta(req);
    return this.guestService.register(dto, ip, ua);
  }

  /**
   * Resend the OTP with a 60-second cooldown + rate limiting.
   */
  @Throttle({ default: { limit: 3, ttl: 60_000 } })
  @Post('otp/resend')
  async resendOtp(@Body() dto: GuestOtpResendDto, @Req() req: Request, @Res({ passthrough: true }) res: Response) {
    this.setNoStoreHeaders(res);
    const { ip, ua } = this.clientMeta(req);
    return this.guestService.resendOtp(dto, ip, ua);
  }

  /**
   * Verify the 6-digit OTP. On success: create the account, set an
   * authenticated session (httpOnly + Secure + SameSite refresh cookie +
   * access JWT), clear the guest cookie, and unlock the dashboard.
   */
  @Throttle({ default: { limit: 10, ttl: 60_000 } })
  @Post('otp/verify')
  async verifyOtp(@Body() dto: GuestOtpVerifyDto, @Req() req: Request, @Res({ passthrough: true }) res: Response) {
    this.setNoStoreHeaders(res);
    const { ip, ua } = this.clientMeta(req);
    const result = await this.guestService.verifyOtp(dto, ip, ua);
    this.setRefreshCookie(res, result._refreshToken);
    this.clearGuestCookie(res);
    const { _refreshToken: _rt, ...publicResult } = result;
    void _rt;
    return publicResult;
  }

  /**
   * Process a marketing unsubscribe from a signed token (emailed link).
   * Honored immediately and persisted. Idempotent.
   */
  @Throttle({ default: { limit: 10, ttl: 60_000 } })
  @Post('unsubscribe')
  async unsubscribe(@Body() dto: GuestUnsubscribeDto, @Req() req: Request) {
    const { ip, ua } = this.clientMeta(req);
    return this.guestService.unsubscribe(dto.token, ip, ua);
  }

  /** Check unsubscribe status for a given email (used by the /unsubscribe page). */
  @Get('unsubscribe-status')
  async unsubscribeStatus(@Req() req: Request) {
    const token = (req.query.token as string | undefined) || '';
    if (!token) return { unsubscribed: false };
    try {
      const result = await this.guestService.unsubscribe(token, undefined, undefined);
      return { unsubscribed: true, email: result.email };
    } catch {
      return { unsubscribed: false };
    }
  }
}
