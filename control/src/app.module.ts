import { Module } from '@nestjs/common';
import { ConfigModule } from '@nestjs/config';
import { ThrottlerModule, ThrottlerGuard } from '@nestjs/throttler';
import { APP_GUARD } from '@nestjs/core';
import { DatabaseModule } from './common/database.module';
import { GlobalJwtModule } from './common/jwt.module';
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
import { HealthModule } from './modules/health/health.module';
import { AdminModule } from './modules/admin/admin.module';
import { DeviceAuthModule } from './modules/device-auth/device-auth.module';
import { OperationsModule } from './modules/operations/operations.module';

@Module({
  imports: [
    ConfigModule.forRoot({ isGlobal: true }),
    // Global rate limiting: 60 requests per minute per IP.
    // Auth endpoints have stricter per-route overrides below.
    ThrottlerModule.forRoot([
      { ttl: 60_000, limit: 60 },
    ]),
    DatabaseModule,
    GlobalJwtModule,
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
    HealthModule,
    AdminModule,
    DeviceAuthModule,
    OperationsModule,
  ],
  providers: [
    { provide: APP_GUARD, useClass: ThrottlerGuard },
  ],
})
export class AppModule {}
