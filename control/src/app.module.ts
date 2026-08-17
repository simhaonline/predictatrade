import { Module } from '@nestjs/common';
import { ConfigModule } from '@nestjs/config';
import { AuthModule } from './modules/auth/auth.module';
import { UsersModule } from './modules/users/users.module';
import { PlansModule } from './modules/plans/plans.module';
import { SubscriptionsModule } from './modules/subscriptions/subscriptions.module';
import { BillingModule } from './modules/billing/billing.module';
import { ReferralsModule } from './modules/referrals/referrals.module';
import { CommissionsModule } from './modules/commissions/commissions.module';
import { PayoutsModule } from './modules/payouts/payouts.module';
import { LicensingModule } from './modules/licensing/licensing.module';
import { AuditModule } from './modules/audit/audit.module';

@Module({
  imports: [
    ConfigModule.forRoot({ isGlobal: true }),
    AuthModule,
    UsersModule,
    PlansModule,
    SubscriptionsModule,
    BillingModule,
    ReferralsModule,
    CommissionsModule,
    PayoutsModule,
    LicensingModule,
    AuditModule,
  ],
})
export class AppModule {}
