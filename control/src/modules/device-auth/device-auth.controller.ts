import { Body, Controller, Post, Get, Headers, Req, UseGuards, BadRequestException, UnauthorizedException, HttpCode, HttpStatus } from '@nestjs/common';
import { DeviceAuthService } from './device-auth.service';
import { AdminGuard } from '../../common/guards/admin.guard';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';

@Controller('devices')
export class DeviceAuthController {
  constructor(private deviceAuthService: DeviceAuthService) {}

  /**
   * POST /api/v1/devices/activate
   * Activate a device against a license key. One license = one device.
   */
  @Post('activate')
  @HttpCode(HttpStatus.OK) // EAs check status == 200 — Nest's POST default is 201
  async activate(@Body() body: any, @Req() req: any) {
    if (!body.license_key) throw new BadRequestException('license_key is required');
    if (!body.client_type) throw new BadRequestException('client_type is required');
    if (!body.fingerprint) throw new BadRequestException('fingerprint is required');
    const ip = req.headers['x-forwarded-for']?.split(',')[0]?.trim() || req.socket?.remoteAddress;
    return this.deviceAuthService.activate(body, ip);
  }

  /**
   * POST /api/v1/devices/refresh
   * Rotate refresh token and issue new access token.
   */
  @Post('refresh')
  @HttpCode(HttpStatus.OK) // EAs check status == 200 — Nest's POST default is 201
  async refresh(@Body() body: any) {
    if (!body.refresh_token) throw new BadRequestException('refresh_token is required');
    // MT4/MT5 EAs send only refresh_token; device_id is derived from the
    // token row. The explicit device_id field is accepted but not required —
    // when present it must MATCH the token's device (prevents cross-device use).
    if (body.device_id) {
      return this.deviceAuthService.refresh(body.refresh_token, body.device_id, body.role);
    }
    return this.deviceAuthService.refresh(body.refresh_token, undefined, body.role);
  }

  /**
   * POST /api/v1/devices/heartbeat
   * Renew session lease. Returns independent connection/auth/license/device/session/trading states.
   */
  @Post('heartbeat')
  @HttpCode(HttpStatus.OK) // EAs check status == 200 — Nest's POST default is 201
  async heartbeat(@Body() body: any, @Req() req: any) {
    if (!body.device_id) throw new BadRequestException('device_id is required');
    if (!body.session_id) throw new BadRequestException('session_id is required');
    const ip = req.headers['x-forwarded-for']?.split(',')[0]?.trim() || req.socket?.remoteAddress;
    return this.deviceAuthService.heartbeat(body.device_id, body.session_id, body, ip);
  }

  // ===== Admin endpoints =====

  @UseGuards(JwtAuthGuard, AdminGuard)
  @Get('sessions')
  async listSessions() {
    return this.deviceAuthService.listActiveSessions();
  }

  @UseGuards(JwtAuthGuard, AdminGuard)
  @Get('devices/:id')
  async getDevice(@Req() req: any) {
    return this.deviceAuthService.getDeviceDetails(req.params?.id || '');
  }

  @UseGuards(JwtAuthGuard, AdminGuard)
  @Post('devices/:id/revoke')
  async revokeDevice(@Req() req: any, @Body() body: any) {
    return this.deviceAuthService.revokeDevice(req.params?.id || '', body.reason || 'admin_revoke');
  }
}
