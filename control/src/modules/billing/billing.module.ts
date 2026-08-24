import { Module } from '@nestjs/common';
import { BillingService } from './billing.service';
import { NowPaymentsService } from './nowpayments.service';
import { BillingController } from './billing.controller';
import { DatabaseModule } from '../../common/database.module';

@Module({
  imports: [DatabaseModule],
  controllers: [BillingController],
  providers: [BillingService, NowPaymentsService],
  exports: [BillingService, NowPaymentsService],
})
export class BillingModule {}
