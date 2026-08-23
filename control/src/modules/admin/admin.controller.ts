import { Controller, Get, Post, Patch, Put, Param, Query, Body, UseGuards, BadRequestException } from '@nestjs/common';
import { AdminService } from './admin.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { AdminGuard } from '../../common/guards/admin.guard';
import { CurrentUser } from '../../common/decorators/current-user.decorator';

@Controller('admin')
@UseGuards(JwtAuthGuard, AdminGuard)
export class AdminController {
  constructor(private adminService: AdminService) {}

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

  @Get('activations')
  async listActivations(@Query('page') page?: string, @Query('limit') limit?: string) {
    return this.adminService.listAllActivations(
      page ? parseInt(page, 10) : 1,
      limit ? parseInt(limit, 10) : 20,
    );
  }

  @Get('users/:id/detail')
  async getUserDetail(@Param('id') id: string) {
    return this.adminService.getUserDetail(id);
  }

  @Post('users/:id/assign-license')
  async assignLicense(
    @Param('id') id: string,
    @Body() body: { planId: string; licenseKey?: string },
    @CurrentUser('sub') actorId: string,
  ) {
    return this.adminService.assignLicense(id, body.planId, actorId, body.licenseKey);
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
}
