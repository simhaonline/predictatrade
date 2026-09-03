import { Module } from '@nestjs/common';
import { ConnectivityWatchdogService } from './connectivity-watchdog.service';
import { DeliveryReconciliationService } from './delivery-reconciliation.service';
import { DeliveryCanaryService } from './delivery-canary.service';
import { MonitoringController } from './monitoring.controller';

@Module({
  controllers: [MonitoringController],
  providers: [ConnectivityWatchdogService, DeliveryReconciliationService, DeliveryCanaryService],
  exports: [ConnectivityWatchdogService, DeliveryReconciliationService, DeliveryCanaryService],
})
export class MonitoringModule {}