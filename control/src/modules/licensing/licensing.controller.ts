import { Body, Controller, Get, Post, Put, Param, UseGuards } from '@nestjs/common';
import { LicensingService } from './licensing.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { CurrentUser } from '../../common/decorators/current-user.decorator';

@Controller('licensing')
export class LicensingController {
  constructor(private licensingService: LicensingService) {}

  // Public endpoint — no JWT required (used by Windows Agent with license key only)
  @Post('validate')
  async validateLicense(@Body() body: { license_key?: string }) {
    if (!body.license_key) {
      return { valid: false, status: 'NO_KEY', error: 'License key is required' };
    }
    return this.licensingService.validateLicenseKey(body.license_key);
  }

  // All endpoints below require JWT authentication
  @UseGuards(JwtAuthGuard)
  @Get('licenses')
  async listLicenses(@CurrentUser('sub') userId: string) {
    return this.licensingService.listLicenses(userId);
  }

  @UseGuards(JwtAuthGuard)
  @Get('devices')
  async listDevices(@CurrentUser('sub') userId: string) {
    return this.licensingService.listDevices(userId);
  }

  @UseGuards(JwtAuthGuard)
  @Post('devices')
  async registerDevice(@CurrentUser('sub') userId: string, @Body() body: any) {
    return this.licensingService.registerDevice(userId, body);
  }

  @UseGuards(JwtAuthGuard)
  @Get('mt-accounts')
  async listMtAccounts(@CurrentUser('sub') userId: string) {
    return this.licensingService.listMtAccounts(userId);
  }

  @UseGuards(JwtAuthGuard)
  @Post('mt-accounts')
  async registerTerminal(@CurrentUser('sub') userId: string, @Body() body: any) {
    return this.licensingService.registerTerminal(userId, body);
  }

  @UseGuards(JwtAuthGuard)
  @Put('devices/:id/heartbeat')
  async heartbeat(@Param('id') deviceId: string, @Body() body: any) {
    return this.licensingService.heartbeat(deviceId, body);
  }

  @UseGuards(JwtAuthGuard)
  @Post('devices/:id/revoke')
  async revokeDevice(@CurrentUser('sub') userId: string, @Param('id') deviceId: string, @Body() body: any) {
    return this.licensingService.revokeDevice(deviceId, body.reason || 'User revoked');
  }

  @UseGuards(JwtAuthGuard)
  @Post('terminals/sync')
  async syncTerminalAccount(@CurrentUser('sub') userId: string, @Body() body: any) {
    return this.licensingService.syncTerminalAccount(userId, body);
  }
}
