import { Controller, Get, UseGuards } from '@nestjs/common';
import { ReferralsService } from './referrals.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { CurrentUser } from '../../common/decorators/current-user.decorator';

@Controller('referrals')
@UseGuards(JwtAuthGuard)
export class ReferralsController {
  constructor(private referralsService: ReferralsService) {}

  @Get('network')
  async getNetwork(@CurrentUser('sub') userId: string) {
    return this.referralsService.getReferralNetwork(userId);
  }

  @Get('commissions')
  async getCommissions(@CurrentUser('sub') userId: string) {
    return this.referralsService.getCommissions(userId);
  }
}
