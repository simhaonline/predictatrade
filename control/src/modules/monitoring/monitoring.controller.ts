import { Controller, Get, UseGuards } from '@nestjs/common';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { AdminGuard } from '../../common/guards/admin.guard';
import { ConnectivityWatchdogService } from './connectivity-watchdog.service';
import { DeliveryReconciliationService } from './delivery-reconciliation.service';

/**
 * Connectivity monitoring (2026-09-04) — server-side assurance that MT
 * clients stay connected and signals keep flowing. Admin-only snapshot;
 * the frontend polls this for the connectivity banner.
 *
 * 2026-09-04: added /monitoring/delivery — end-to-end signal delivery
 * reconciliation (ACKs are NOT proof of delivery; see the 2026-09-03
 * silent-drop incident). Shows per-device dispatch health, 24h
 * delivered/dropped counts, and enqueue backlog aging.
 */
@Controller('monitoring')
@UseGuards(JwtAuthGuard, AdminGuard)
export class MonitoringController {
  constructor(
    private readonly watchdog: ConnectivityWatchdogService,
    private readonly delivery: DeliveryReconciliationService,
  ) {}

  @Get('connectivity')
  async getConnectivity() {
    return this.watchdog.getConnectivitySnapshot();
  }

  @Get('delivery')
  async getDelivery() {
    // Run one reconciliation pass so the snapshot is always fresh, then read.
    await this.delivery.reconcile();
    return this.delivery.getDeliverySnapshot();
  }
}