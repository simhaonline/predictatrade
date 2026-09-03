import { Controller, Get, UseGuards, Query } from '@nestjs/common';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { AdminGuard } from '../../common/guards/admin.guard';
import { ConnectivityWatchdogService } from './connectivity-watchdog.service';

/**
 * Connectivity monitoring (2026-09-04) — server-side assurance that MT
 * clients stay connected and signals keep flowing. Admin-only snapshot;
 * the frontend polls this for the connectivity banner.
 */
@Controller('monitoring')
@UseGuards(JwtAuthGuard, AdminGuard)
export class MonitoringController {
  constructor(private readonly watchdog: ConnectivityWatchdogService) {}

  @Get('connectivity')
  async getConnectivity() {
    return this.watchdog.getConnectivitySnapshot();
  }
}