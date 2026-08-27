import { Body, Controller, Get, Post, Param, Query, UseGuards } from '@nestjs/common';
import { PayoutsService } from './payouts.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { RolesGuard, Roles, Role, PermissionGuard, Permission, RequirePermissions } from '../../common/guards/roles.guard';
import { CurrentUser } from '../../common/decorators/current-user.decorator';
import { RequestPayoutDto } from './dto/request-payout.dto';
import { RejectPayoutDto } from './dto/reject-payout.dto';
import { CancelPayoutDto } from './dto/reject-payout.dto';
import { ReconcilePayoutDto } from './dto/reconcile-payout.dto';

@Controller('payouts')
@UseGuards(JwtAuthGuard, RolesGuard)
export class PayoutsController {
  constructor(private payoutsService: PayoutsService) {}

  // Resource-scoped: callers may only see their own payouts (userId from JWT).
  @Get()
  async list(@CurrentUser('sub') userId: string) { return this.payoutsService.findByUser(userId); }

  @Post('request')
  async request(@CurrentUser('sub') userId: string, @Body() dto: RequestPayoutDto) {
    return this.payoutsService.requestPayout(userId, dto);
  }

  @UseGuards(PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @RequirePermissions(Permission.PAYOUT_APPROVE)
  @Get('admin/all')
  async listAll(@Query('page') page?: string, @Query('limit') limit?: string) {
    return this.payoutsService.listAll(
      page ? parseInt(page, 10) : 1,
      limit ? parseInt(limit, 10) : 20,
    );
  }

  @UseGuards(PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @RequirePermissions(Permission.PAYOUT_APPROVE)
  @Get('admin/stats')
  async stats() { return this.payoutsService.getStats(); }

  @UseGuards(PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @RequirePermissions(Permission.PAYOUT_APPROVE)
  @Post(':id/approve')
  async approve(@Param('id') id: string, @CurrentUser('sub') actorId: string) {
    return this.payoutsService.approvePayout(id, actorId);
  }

  @UseGuards(PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @RequirePermissions(Permission.PAYOUT_APPROVE)
  @Post(':id/reject')
  async reject(@Param('id') id: string, @Body() dto: RejectPayoutDto) {
    return this.payoutsService.rejectPayout(id, dto.reason);
  }

  @UseGuards(PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @RequirePermissions(Permission.PAYOUT_APPROVE)
  @Post(':id/process')
  async process(@Param('id') id: string) { return this.payoutsService.processPayout(id); }

  @UseGuards(PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @RequirePermissions(Permission.PAYOUT_RECONCILE)
  @Post(':id/reconcile')
  async reconcile(@Param('id') id: string, @Body() dto: ReconcilePayoutDto) {
    return this.payoutsService.reconcilePayout(id, dto);
  }

  @UseGuards(PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @RequirePermissions(Permission.PAYOUT_APPROVE)
  @Post(':id/retry')
  async retry(@Param('id') id: string) { return this.payoutsService.retryPayout(id); }

  @UseGuards(PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @RequirePermissions(Permission.PAYOUT_APPROVE)
  @Post(':id/cancel')
  async cancel(@Param('id') id: string, @Body() dto: CancelPayoutDto) {
    return this.payoutsService.cancelPayout(id, dto.reason);
  }
}
