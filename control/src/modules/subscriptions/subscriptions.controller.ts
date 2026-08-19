import { Body, Controller, Get, Post, UseGuards } from '@nestjs/common';
import { SubscriptionsService } from './subscriptions.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { CurrentUser } from '../../common/decorators/current-user.decorator';
import { CreateSubscriptionDto } from './dto/create-subscription.dto';

@Controller('subscriptions')
@UseGuards(JwtAuthGuard)
export class SubscriptionsController {
  constructor(private subsService: SubscriptionsService) {}

  @Get()
  async list(@CurrentUser('sub') userId: string) { return this.subsService.findByUser(userId); }

  @Post()
  async create(@CurrentUser('sub') userId: string, @Body() dto: CreateSubscriptionDto) {
    return this.subsService.create(userId, dto);
  }
}
