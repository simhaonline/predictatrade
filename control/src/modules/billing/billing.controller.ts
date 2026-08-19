import { Body, Controller, Get, Post, Headers, UseGuards } from '@nestjs/common';
import { BillingService } from './billing.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { CurrentUser } from '../../common/decorators/current-user.decorator';

@Controller('billing')
export class BillingController {
  constructor(private billingService: BillingService) {}

  @UseGuards(JwtAuthGuard)
  @Get('invoices')
  async listInvoices(@CurrentUser('sub') userId: string) {
    return this.billingService.listInvoices(userId);
  }

  @Post('webhook')
  async webhook(@Body() body: any, @Headers() headers: any) {
    return this.billingService.handleWebhook(body, headers);
  }
}
