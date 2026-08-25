import { Body, Controller, Post, Headers, UseGuards, Req, Param } from '@nestjs/common';
import { Request } from 'express';
import { RawBodyRequest } from '@nestjs/common';
import { StripeService } from './stripe.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { CurrentUser } from '../../common/decorators/current-user.decorator';

@Controller('billing')
export class StripeController {
  constructor(private stripeService: StripeService) {}

  @UseGuards(JwtAuthGuard)
  @Post('stripe/checkout')
  async createCheckout(
    @CurrentUser('sub') userId: string,
    @Body()
    body: {
      plan_id: string;
      billing_interval?: 'MONTHLY' | 'ANNUAL';
      success_url?: string;
      cancel_url?: string;
    },
  ) {
    const result = await this.stripeService.createCheckoutSession({
      planId: body?.plan_id,
      billingInterval: body?.billing_interval ?? 'MONTHLY',
      userId,
      successUrl: body?.success_url,
      cancelUrl: body?.cancel_url,
    });
    return { url: result.url, session_id: result.sessionId };
  }

  @Post('webhook/stripe')
  async stripeWebhook(@Req() req: RawBodyRequest<Request>, @Headers('stripe-signature') signature: string) {
    // Public by design: the Stripe-Signature HMAC-SHA256 verification inside
    // handleWebhook is the only authentication for gateway callbacks. Pass the
    // RAW body so the signature matches exactly what Stripe computed.
    const raw = req.rawBody ? req.rawBody.toString('utf8') : '';
    return this.stripeService.handleWebhook(raw, signature);
  }
}
