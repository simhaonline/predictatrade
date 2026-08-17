import { Module } from '@nestjs/common';
import { ReferralsService } from './referrals.service';
import { CommissionsModule } from '../commissions/commissions.module';

@Module({
  imports: [CommissionsModule],
  providers: [ReferralsService],
  exports: [ReferralsService],
})
export class ReferralsModule {}
