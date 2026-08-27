import { Controller, Get, Post, Put, Query, Param, Body, UseGuards } from '@nestjs/common';
import { CommissionsService } from './commissions.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { RolesGuard, Roles, Role, PermissionGuard, Permission, RequirePermissions } from '../../common/guards/roles.guard';
import { CurrentUser } from '../../common/decorators/current-user.decorator';

@Controller('commissions')
@UseGuards(JwtAuthGuard, RolesGuard)
export class CommissionsController {
  constructor(private commissionsService: CommissionsService) {}

  // Resource-scoped: callers may only see their own commissions (userId from JWT).
  @Get()
  async list(@CurrentUser('sub') userId: string) { return this.commissionsService.findByRecipient(userId); }

  @Get('summary')
  async summary(@CurrentUser('sub') userId: string) { return this.commissionsService.getSummary(userId); }

  @UseGuards(PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @Get('admin/all')
  async listAll(@Query('page') page?: string, @Query('limit') limit?: string) {
    return this.commissionsService.listAll(
      page ? parseInt(page, 10) : 1,
      limit ? parseInt(limit, 10) : 20,
    );
  }

  @UseGuards(PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @Get('admin/summary')
  async adminSummary() { return this.commissionsService.getGlobalSummary(); }

  @UseGuards(PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @Get('admin/rules')
  async listRules() { return this.commissionsService.listRules(); }

  @UseGuards(PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @RequirePermissions(Permission.COMMISSION_ADJUST)
  @Post('admin/:id/clear')
  async clear(@Param('id') id: string, @CurrentUser('sub') actorId: string) {
    return this.commissionsService.transitionCommission(id, 'CLEARED', actorId);
  }

  @UseGuards(PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @RequirePermissions(Permission.COMMISSION_ADJUST)
  @Post('admin/:id/available')
  async available(@Param('id') id: string, @CurrentUser('sub') actorId: string) {
    return this.commissionsService.transitionCommission(id, 'AVAILABLE', actorId);
  }

  @UseGuards(PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @RequirePermissions(Permission.COMMISSION_ADJUST)
  @Post('admin/:id/hold')
  async hold(@Param('id') id: string, @Body('reason') reason: string, @CurrentUser('sub') actorId: string) {
    return this.commissionsService.holdCommission(id, reason, actorId);
  }

  @UseGuards(PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @RequirePermissions(Permission.COMMISSION_ADJUST)
  @Post('admin/:id/release')
  async release(@Param('id') id: string, @Body('reason') reason: string, @CurrentUser('sub') actorId: string) {
    return this.commissionsService.releaseCommission(id, actorId);
  }

  @UseGuards(PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @RequirePermissions(Permission.COMMISSION_REVERSE)
  @Post('admin/:id/reverse')
  async reverse(
    @Param('id') id: string,
    @Body('reason') reason: string,
    @Body('amount') amount: number,
    @CurrentUser('sub') actorId: string,
  ) {
    return this.commissionsService.reverseCommission(id, reason, actorId, amount);
  }

  @UseGuards(PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @RequirePermissions(Permission.COMMISSION_ADJUST)
  @Post('admin/:id/adjust')
  async adjust(
    @Param('id') id: string,
    @Body('amount') amount: number,
    @Body('reason') reason: string,
    @CurrentUser('sub') actorId: string,
  ) {
    return this.commissionsService.adjustCommission(id, amount, reason, actorId);
  }

  @UseGuards(PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @RequirePermissions(Permission.COMMISSION_ADJUST)
  @Post('admin/clear-eligible')
  async clearEligible() {
    return this.commissionsService.clearEligible();
  }

  @UseGuards(PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @RequirePermissions(Permission.COMMISSION_ADJUST)
  @Put('admin/rules/:id')
  async updateRule(
    @Param('id') id: string,
    @Body() body: { base_rate?: number; active?: boolean; effective_until?: string },
  ) {
    return this.commissionsService.updateRule(id, body);
  }
}
