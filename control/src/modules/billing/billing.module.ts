import { Module } from '@nestjs/common';
import { BillingService } from './billing.service';
import { NowPaymentsService } from './nowpayments.service';
import { StripeService } from './stripe.service';
import { BillingController } from './billing.controller';
import { StripeController } from './stripe.controller';
import { DatabaseModule } from '../../common/database.module';

@Module({
  imports: [DatabaseModule],
  controllers: [BillingController, StripeController],
  providers: [BillingService, NowPaymentsService, StripeService],
  exports: [BillingService, NowPaymentsService, StripeService],
})
export class BillingModule {}
