import { Body, Controller, Get, Post, Param, Res, Headers, UseGuards, Req } from '@nestjs/common';
import { Request, Response } from 'express';
import { RawBodyRequest } from '@nestjs/common';
import { BillingService } from './billing.service';
import { NowPaymentsService } from './nowpayments.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { RolesGuard, Roles, Role, PermissionGuard, Permission, RequirePermissions } from '../../common/guards/roles.guard';
import { CurrentUser } from '../../common/decorators/current-user.decorator';

@Controller('billing')
export class BillingController {
  constructor(
    private billingService: BillingService,
    private nowPaymentsService: NowPaymentsService,
  ) {}

  @UseGuards(JwtAuthGuard)
  @Get('invoices')
  async listInvoices(@CurrentUser('sub') userId: string) {
    return this.billingService.listInvoices(userId);
  }

  /**
   * USDT payment status for the user's dashboard (subscriber-visible).
   * Returns the latest payment per subscription with the NOWPayments hosted
   * URL to resume an abandoned checkout, plus the exact amount and gateway
   * status — driven from DB truth (billing.payments), never client-side.
   */
  @UseGuards(JwtAuthGuard)
  @Get('payments')
  async listPayments(@CurrentUser('sub') userId: string) {
    return this.billingService.listPaymentsForUser(userId);
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

  @UseGuards(JwtAuthGuard, RolesGuard, PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @RequirePermissions(Permission.BILLING_MANAGE)
  @Post('invoices/:id/mark-paid')
  async markPaid(
    @CurrentUser('sub') userId: string,
    @Param('id') id: string,
    @Body() body: { payment_id?: string },
  ) {
    return this.billingService.markInvoicePaid(id, body?.payment_id);
  }

  @Post('webhook')
  async webhook(@Req() req: RawBodyRequest<Request>, @Headers() headers: any) {
    // P0-CP1 fix: HMAC signature verification + event-id idempotency.
    return this.billingService.handleWebhook(req.body, headers, req.rawBody);
  }

  @UseGuards(JwtAuthGuard)
  @Post('nowpayments/create-invoice')
  async createNowPaymentsInvoice(
    @CurrentUser('sub') userId: string,
    @Body() body: { plan_id: string; billing_interval?: 'MONTHLY' | 'ANNUAL' },
  ) {
    const result = await this.nowPaymentsService.createInvoice(
      userId,
      body?.plan_id,
      body?.billing_interval ?? 'MONTHLY',
    );
    return { payment_url: result.payment_url, invoice_id: result.invoice_id };
  }

  @Post('webhook/nowpayments')
  async nowpaymentsWebhook(@Req() req: RawBodyRequest<Request>, @Headers() headers: any) {
    // Public by design: the x-nowpayments-sig HMAC-SHA512 verification inside
    // handleIPN is the only authentication for gateway callbacks. Pass the RAW
    // body so the signature matches exactly what NOWPayments computed.
    const raw = req.rawBody ? req.rawBody.toString('utf8') : '';
    return this.nowPaymentsService.handleIPN(raw, headers?.['x-nowpayments-sig']);
  }
}
