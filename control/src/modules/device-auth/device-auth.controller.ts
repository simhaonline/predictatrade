import { Body, Controller, Post, Get, Headers, Req, UseGuards, BadRequestException, UnauthorizedException } from '@nestjs/common';
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
  async refresh(@Body() body: any) {
    if (!body.refresh_token) throw new BadRequestException('refresh_token is required');
    if (!body.device_id) throw new BadRequestException('device_id is required');
    return this.deviceAuthService.refresh(body.refresh_token, body.device_id);
  }

  /**
   * POST /api/v1/devices/heartbeat
   * Renew session lease. Returns independent connection/auth/license/device/session/trading states.
   */
  @Post('heartbeat')
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
