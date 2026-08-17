import { Module } from '@nestjs/common';
import { CommissionEngine } from './commission-engine';

@Module({
  providers: [CommissionEngine],
  exports: [CommissionEngine],
})
export class CommissionsModule {}
