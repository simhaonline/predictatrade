import { Controller, Get, Post, Patch, Put, Param, Query, Body, UseGuards, BadRequestException, Inject } from '@nestjs/common';
import { AdminService } from './admin.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { RolesGuard, Roles, Role, PermissionGuard, Permission, RequirePermissions } from '../../common/guards/roles.guard';
import { CurrentUser } from '../../common/decorators/current-user.decorator';
import { DB_POOL } from '../../common/database.module';
import { Pool } from 'pg';

@Controller('admin')
@UseGuards(JwtAuthGuard, RolesGuard, PermissionGuard)
@Roles(Role.ADMIN, Role.SUPER_ADMIN)
export class AdminController {
  constructor(
    private adminService: AdminService,
    @Inject(DB_POOL) private pool: Pool,
  ) {}

  @Get('overview')
  async overview() {
    return this.adminService.getOverview();
  }

  @Get('users')
  async listUsers(@Query('page') page?: string, @Query('limit') limit?: string) {
    return this.adminService.listUsers(
      page ? parseInt(page, 10) : 1,
      limit ? parseInt(limit, 10) : 20,
    );
  }

  @Patch('users/:id/status')
  @RequirePermissions(Permission.USER_MANAGE)
  async updateUserStatus(@Param('id') id: string, @Query('status') status: string, @CurrentUser('sub') actorId: string) {
    if (!status) throw new BadRequestException('Status query parameter is required');
    return this.adminService.updateUserStatus(id, status, actorId);
  }

  @Get('subscriptions')
  async listSubscriptions(@Query('page') page?: string, @Query('limit') limit?: string) {
    return this.adminService.listAllSubscriptions(
      page ? parseInt(page, 10) : 1,
      limit ? parseInt(limit, 10) : 20,
    );
  }

  @Post('subscriptions/:id/complete')
  @RequirePermissions(Permission.BILLING_MANAGE)
  async completeSubscription(
    @Param('id') id: string,
    @Body() body: { reason?: string },
    @CurrentUser('sub') actorId: string,
  ) {
    return this.adminService.completeIncompleteSubscription(id, actorId, body?.reason ?? '');
  }

  @Get('commissions')
  async listCommissions(@Query('page') page?: string, @Query('limit') limit?: string) {
    return this.adminService.listAllCommissions(
      page ? parseInt(page, 10) : 1,
      limit ? parseInt(limit, 10) : 20,
    );
  }

  @Get('commissions/summary')
  async commissionSummary() {
    return this.adminService.commissionSummary();
  }

  @Get('payouts')
  async listPayouts(@Query('page') page?: string, @Query('limit') limit?: string) {
    return this.adminService.listAllPayouts(
      page ? parseInt(page, 10) : 1,
      limit ? parseInt(limit, 10) : 20,
    );
  }

  @Get('payouts/stats')
  async payoutStats() {
    return this.adminService.payoutStats();
  }

  @Get('plans')
  async listPlans() {
    const r = await this.pool.query(
      'SELECT id, code, name, monthly_price, currency, allowed_strategies FROM control.plans WHERE code != $1 ORDER BY monthly_price ASC',
      ['BASIC'],
    );
    return r.rows;
  }

  @Get('licenses')
  async listLicenses(@Query('page') page?: string, @Query('limit') limit?: string) {
    return this.adminService.listAllLicenses(
      page ? parseInt(page, 10) : 1,
      limit ? parseInt(limit, 10) : 20,
    );
  }

  @Get('devices')
  async listDevices(@Query('page') page?: string, @Query('limit') limit?: string) {
    return this.adminService.listAllDevices(
      page ? parseInt(page, 10) : 1,
      limit ? parseInt(limit, 10) : 20,
    );
  }

  /** v1.28: EA capital-guard risk events (FLOATING_DD_BREAKER, SOFT_HALT, RECOVER...). */
  @Get('devices/risk-events')
  async listRiskEvents(@Query('device_id') deviceId?: string, @Query('limit') limit?: string) {
    const lim = Math.min(Math.max(limit ? parseInt(limit, 10) : 50, 1), 200);
    return this.adminService.listDeviceRiskEvents(deviceId || '', lim);
  }

  @Get('activations')
  async listActivations(
    @Query('page') page?: string,
    @Query('limit') limit?: string,
    @Query('scope') scope?: string,
  ) {
    return this.adminService.listAllActivations(
      page ? parseInt(page, 10) : 1,
      limit ? parseInt(limit, 10) : 20,
      scope === 'history' ? 'history' : scope === 'recent' ? 'recent' : 'live',
    );
  }

  @Get('users/:id/detail')
  async getUserDetail(@Param('id') id: string) {
    return this.adminService.getUserDetail(id);
  }

  @Post('users/:id/assign-license')
  @RequirePermissions(Permission.USER_MANAGE)
  async assignLicense(
    @Param('id') id: string,
    @Body() body: { planId: string; licenseKey?: string },
    @CurrentUser('sub') actorId: string,
  ) {
    return this.adminService.assignLicense(id, body.planId, actorId, body.licenseKey);
  }

  /**
   * Admin-initiated password reset: mints the same one-time password_reset
   * JWT the email flow uses, attempts email delivery, and ALWAYS returns the
   * reset URL so the admin can hand it over out-of-band (chat/phone) when
   * email delivery is unavailable. Audited.
   */
  @Post('users/:id/send-reset-link')
  @RequirePermissions(Permission.USER_MANAGE)
  async sendResetLink(
    @Param('id') id: string,
    @Body() body: { reason?: string },
    @CurrentUser('sub') actorId: string,
  ) {
    return this.adminService.generatePasswordResetLink(id, actorId, body?.reason ?? '');
  }

  /**
   * v1.31 unified manual approval — one admin gate covering subscriptions,
   * licenses, and user account activation. See AdminService.approveEntitlement
   * for the per-entity side effects. Audited.
   */
  @Post('approve')
  @RequirePermissions(Permission.USER_MANAGE)
  async approveEntitlement(
    @Body() body: { entityType: 'subscription' | 'license' | 'user'; entityId: string; decision: 'approve' | 'reject'; reason?: string },
    @CurrentUser('sub') actorId: string,
  ) {
    if (!body?.entityType || !body?.entityId || !['approve', 'reject'].includes(body?.decision)) {
      throw new BadRequestException('entityType (subscription|license|user), entityId and decision (approve|reject) are required');
    }
    return this.adminService.approveEntitlement(
      body.entityType, body.entityId, body.decision, actorId, body?.reason ?? '',
    );
  }

  @Get('users-without-subscription')
  async listUsersWithoutSubscription() {
    return this.adminService.listUsersWithoutSubscription();
  }

  @Post('users/:id/start-subscription')
  @RequirePermissions(Permission.USER_MANAGE)
  async startSubscription(
    @Param('id') id: string,
    @Body() body: { planId: string; billingInterval?: 'MONTHLY' | 'ANNUAL' },
    @CurrentUser('sub') actorId: string,
  ) {
    const interval = body?.billingInterval === 'ANNUAL' ? 'ANNUAL' : 'MONTHLY';
    return this.adminService.startSubscriptionForUser(id, body.planId, interval, actorId);
  }

  @Get('trading-reports')
  async tradingReports() {
    return this.adminService.getTradingReport();
  }

  @Get('regime-diagnostics')
  async regimeDiagnostics() {
    return this.adminService.getRegimeDiagnostics();
  }

  @Get('risk-config')
  async getRiskConfig() {
    return this.adminService.getRiskConfig();
  }

  @Put('risk-config')
  @RequirePermissions(Permission.RISK_MANAGE)
  async saveRiskConfig(
    @Body() body: {
      kill_switches?: Record<string, boolean>;
      limits?: Record<string, number>;
      session_blackout?: boolean;
      news_blackout?: boolean;
      blackout_reason?: string;
    },
    @CurrentUser('sub') actorId: string,
  ) {
    return this.adminService.saveRiskConfig(body, actorId);
  }

  @Get('health')
  async systemHealth() {
    return this.adminService.systemHealth();
  }

  @Get('subscriptions/payments')
  async subscriptionPayments() {
    return this.adminService.getSubscriptionPayments();
  }

  @Get('subscriptions/refunds')
  async subscriptionRefunds() {
    return this.adminService.getSubscriptionRefunds();
  }

  @Get('subscriptions/chargebacks')
  async subscriptionChargebacks() {
    return this.adminService.getSubscriptionChargebacks();
  }

  @Get('subscriptions/coupons')
  async subscriptionCoupons() {
    return this.adminService.getSubscriptionCoupons();
  }

  @Get('subscriptions/provider')
  async subscriptionProvider() {
    return this.adminService.getSubscriptionProvider();
  }

  @Get('subscriptions/invoices')
  async subscriptionInvoices() {
    return this.adminService.getSubscriptionInvoices();
  }

  @Post('subscriptions/coupons')
  async createCoupon(@Body() body: {
    code: string;
    description?: string | null;
    discountType: 'PERCENTAGE' | 'FIXED';
    discountValue: number;
    currency?: string;
    maxRedemptions?: number | null;
    validFrom?: string | null;
    validUntil?: string | null;
    active?: boolean;
  }) {
    if (!body.code || !body.discountType || body.discountValue == null) {
      throw new BadRequestException('code, discountType and discountValue are required');
    }
    return this.adminService.createCoupon(body);
  }

  @Get('signal-accuracy')
  getSignalAccuracy() {
    return this.adminService.getSignalAccuracy();
  }

}
