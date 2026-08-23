import { Controller, Get, UseGuards } from '@nestjs/common';
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
}
