import { Body, Controller, Get, Post, Param, Res, Headers, UseGuards } from '@nestjs/common';
import { Response } from 'express';
import { BillingService } from './billing.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { AdminGuard } from '../../common/guards/admin.guard';
import { CurrentUser } from '../../common/decorators/current-user.decorator';

@Controller('billing')
export class BillingController {
  constructor(private billingService: BillingService) {}

  @UseGuards(JwtAuthGuard)
  @Get('invoices')
  async listInvoices(@CurrentUser('sub') userId: string) {
    return this.billingService.listInvoices(userId);
  }

  @UseGuards(JwtAuthGuard)
  @Post('invoices/generate')
  async generateInvoice(
    @CurrentUser('sub') userId: string,
    @Body() body: { subscription_id: string; billing_period_start?: string; billing_period_end?: string },
  ) {
    const id = await this.billingService.generateInvoiceForSubscription(body.subscription_id, userId, {
      billingPeriodStart: body.billing_period_start,
      billingPeriodEnd: body.billing_period_end,
    });
    return { id };
  }

  @UseGuards(JwtAuthGuard)
  @Get('invoices/:id')
  async getInvoice(@CurrentUser('sub') userId: string, @Param('id') id: string) {
    await this.billingService.assertInvoiceOwnership(userId, id);
    return this.billingService.getInvoice(id);
  }

  @UseGuards(JwtAuthGuard)
  @Get('invoices/:id/html')
  async getInvoiceHtml(
    @CurrentUser('sub') userId: string,
    @Param('id') id: string,
    @Res() res: Response,
  ) {
    await this.billingService.assertInvoiceOwnership(userId, id);
    const html = await this.billingService.renderBrandedInvoiceHtml(id);
    res.type('text/html').send(html);
  }

  @UseGuards(JwtAuthGuard, AdminGuard)
  @Post('invoices/:id/mark-paid')
  async markPaid(
    @CurrentUser('sub') userId: string,
    @Param('id') id: string,
    @Body() body: { payment_id?: string },
  ) {
    return this.billingService.markInvoicePaid(id, body?.payment_id);
  }

  @Post('webhook')
  async webhook(@Body() body: any, @Headers() headers: any) {
    return this.billingService.handleWebhook(body, headers);
  }
}
