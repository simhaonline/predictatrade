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
import { ComplianceModule } from './modules/compliance/compliance.module';
import { HealthModule } from './modules/health/health.module';
import { AdminModule } from './modules/admin/admin.module';
import { DeviceAuthModule } from './modules/device-auth/device-auth.module';
import { OperationsModule } from './modules/operations/operations.module';
import { BacktestModule } from './modules/backtest/backtest.module';
import { GuestPreviewModule } from './modules/guest-preview/guest-preview.module';
import { AgentsModule } from './modules/agents/agents.module';
import { MarketProxyModule } from './modules/market-proxy/market-proxy.module';
import { FeatureFlagsModule } from './modules/feature-flags/feature-flags.module';
import { AdminExtrasModule } from './modules/admin-extras/admin-extras.module';
import { ReportsModule } from './modules/reports/reports.module';
import { APP_INTERCEPTOR } from '@nestjs/core';
import { ComplianceInterceptor } from './common/interceptors/compliance.interceptor';

@Module({
  imports: [
    ConfigModule.forRoot({ isGlobal: true }),
    // Global rate limiting: 300 requests per minute per IP (5/sec).
    // Interactive dashboards poll market/subscription data; 60/min was too
    // low and caused 429 storms on /auth/refresh and /subscriptions/entitlements.
    // Mutating/auth endpoints keep stricter per-route overrides below.
    ThrottlerModule.forRoot([
      { ttl: 60_000, limit: 300 },
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
    ComplianceModule,
    HealthModule,
    AdminModule,
    DeviceAuthModule,
    OperationsModule,
    BacktestModule,
    GuestPreviewModule,
    AgentsModule,
    MarketProxyModule,
    FeatureFlagsModule,
    AdminExtrasModule,
    ReportsModule,
  ],
  providers: [
    { provide: APP_GUARD, useClass: ThrottlerGuard },
    { provide: APP_INTERCEPTOR, useClass: ComplianceInterceptor },
  ],
})
export class AppModule {}
