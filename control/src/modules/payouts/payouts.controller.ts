import { Body, Controller, Get, Post, Param, Query, UseGuards } from '@nestjs/common';
import { PayoutsService } from './payouts.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { AdminGuard } from '../../common/guards/admin.guard';
import { CurrentUser } from '../../common/decorators/current-user.decorator';
import { RequestPayoutDto } from './dto/request-payout.dto';

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
}
