import { Body, Controller, Get, Post, UseGuards } from '@nestjs/common';
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
  async addMtAccount(@CurrentUser('sub') userId: string, @Body() body: any) {
    return this.licensingService.addMtAccount(userId, body);
  }
}
