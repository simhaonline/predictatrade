import { Body, Controller, Get, Post, Put, Param, UseGuards } from '@nestjs/common';
import { LicensingService } from './licensing.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { CurrentUser } from '../../common/decorators/current-user.decorator';

@Controller('licensing')
@UseGuards(JwtAuthGuard)
export class LicensingController {
  constructor(private licensingService: LicensingService) {}

  @Get('licenses')
  async listLicenses(@CurrentUser('sub') userId: string) {
    return this.licensingService.listLicenses(userId);
  }

  @Get('devices')
  async listDevices(@CurrentUser('sub') userId: string) {
    return this.licensingService.listDevices(userId);
  }

  @Post('devices')
  async registerDevice(@CurrentUser('sub') userId: string, @Body() body: any) {
    return this.licensingService.registerDevice(userId, body);
  }

  @Get('mt-accounts')
  async listMtAccounts(@CurrentUser('sub') userId: string) {
    return this.licensingService.listMtAccounts(userId);
  }

  @Post('mt-accounts')
  async registerTerminal(@CurrentUser('sub') userId: string, @Body() body: any) {
    return this.licensingService.registerTerminal(userId, body);
  }

  @Put('devices/:id/heartbeat')
  async heartbeat(@Param('id') deviceId: string, @Body() body: any) {
    return this.licensingService.heartbeat(deviceId, body);
  }

  @Post('devices/:id/revoke')
  async revokeDevice(@CurrentUser('sub') userId: string, @Param('id') deviceId: string, @Body() body: any) {
    return this.licensingService.revokeDevice(deviceId, body.reason || 'User revoked');
  }

  @Post('terminals/sync')
  async syncTerminalAccount(@CurrentUser('sub') userId: string, @Body() body: any) {
    return this.licensingService.syncTerminalAccount(userId, body);
  }
}
