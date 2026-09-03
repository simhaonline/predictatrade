import { Module } from '@nestjs/common';
import { ConnectivityWatchdogService } from './connectivity-watchdog.service';
import { MonitoringController } from './monitoring.controller';

@Module({
  controllers: [MonitoringController],
  providers: [ConnectivityWatchdogService],
  exports: [ConnectivityWatchdogService],
})
export class MonitoringModule {}