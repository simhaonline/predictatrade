import { Body, Controller, Get, Post, Put, UseGuards } from '@nestjs/common';
import { AdminExtrasService } from './admin-extras.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { AdminGuard } from '../../common/guards/admin.guard';

@Controller('admin')
@UseGuards(JwtAuthGuard, AdminGuard)
export class AdminExtrasController {
  constructor(private adminExtrasService: AdminExtrasService) {}

  @Get('backup-dr')
  async backupDr() {
    return this.adminExtrasService.getBackupDr();
  }

  @Get('releases')
  async releases() {
    return this.adminExtrasService.getReleases();
  }

  @Get('broker-qualification')
  async brokerQualification() {
    return this.adminExtrasService.getBrokerQualification();
  }

  @Get('macro-news')
  async macroNews() {
    return this.adminExtrasService.getMacroNews();
  }

  @Post('releases')
  async publishRelease(@Body() body: any) {
    return this.adminExtrasService.publishRelease(body);
  }

  @Post('backup-dr/restore-test')
  async triggerRestoreTest() {
    return this.adminExtrasService.triggerRestoreTest();
  }

  @Post('macro-news/blackout')
  async setBlackout(@Body() body: any) {
    return this.adminExtrasService.setBlackout(body);
  }

  @Post('broker-qualification')
  async addQualificationRun(@Body() body: any) {
    return this.adminExtrasService.addQualificationRun(body);
  }
}
