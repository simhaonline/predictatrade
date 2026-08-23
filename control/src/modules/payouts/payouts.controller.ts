import { Body, Controller, Get, Post, Param, Query, UseGuards } from '@nestjs/common';
import { PayoutsService } from './payouts.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { AdminGuard } from '../../common/guards/admin.guard';
import { CurrentUser } from '../../common/decorators/current-user.decorator';
import { RequestPayoutDto } from './dto/request-payout.dto';
import { RejectPayoutDto } from './dto/reject-payout.dto';
import { CancelPayoutDto } from './dto/reject-payout.dto';
import { ReconcilePayoutDto } from './dto/reconcile-payout.dto';

@Controller('payouts')
@UseGuards(JwtAuthGuard)
export class PayoutsController {
  constructor(private payoutsService: PayoutsService) {}

  @Get()
  async list(@CurrentUser('sub') userId: string) { return this.payoutsService.findByUser(userId); }

  @Post('request')
  async request(@CurrentUser('sub') userId: string, @Body() dto: RequestPayoutDto) {
    return this.payoutsService.requestPayout(userId, dto);
  }

  @UseGuards(AdminGuard)
  @Get('admin/all')
  async listAll(@Query('page') page?: string, @Query('limit') limit?: string) {
    return this.payoutsService.listAll(
      page ? parseInt(page, 10) : 1,
      limit ? parseInt(limit, 10) : 20,
    );
  }

  @UseGuards(AdminGuard)
  @Get('admin/stats')
  async stats() { return this.payoutsService.getStats(); }

  @UseGuards(AdminGuard)
  @Post(':id/approve')
  async approve(@Param('id') id: string) { return this.payoutsService.approvePayout(id); }

  @UseGuards(AdminGuard)
  @Post(':id/reject')
  async reject(@Param('id') id: string, @Body() dto: RejectPayoutDto) {
    return this.payoutsService.rejectPayout(id, dto.reason);
  }

  @UseGuards(AdminGuard)
  @Post(':id/process')
  async process(@Param('id') id: string) { return this.payoutsService.processPayout(id); }

  @UseGuards(AdminGuard)
  @Post(':id/reconcile')
  async reconcile(@Param('id') id: string, @Body() dto: ReconcilePayoutDto) {
    return this.payoutsService.reconcilePayout(id, dto);
  }

  @UseGuards(AdminGuard)
  @Post(':id/retry')
  async retry(@Param('id') id: string) { return this.payoutsService.retryPayout(id); }

  @UseGuards(AdminGuard)
  @Post(':id/cancel')
  async cancel(@Param('id') id: string, @Body() dto: CancelPayoutDto) {
    return this.payoutsService.cancelPayout(id, dto.reason);
  }
}
