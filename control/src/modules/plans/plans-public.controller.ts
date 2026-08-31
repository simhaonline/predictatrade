import { Controller, Get } from '@nestjs/common';
import { PlansService } from './plans.service';

/**
 * Public, unauthenticated plans endpoint used by the marketing / pricing page.
 * Returns only safe, non-sensitive plan fields. Entitlement and billing logic
 * remains server-authoritative elsewhere; this is read-only marketing data.
 */
@Controller('public/plans')
export class PlansPublicController {
  constructor(private plansService: PlansService) {}

  @Get()
  async list() {
    const plans = await this.plansService.listActive();
    return plans.map((p) => ({
      code: p.code,
      name: p.name,
      description: p.description,
      monthly_price: p.monthly_price,
      annual_price: p.annual_price,
      max_active_strategy_slots: p.max_active_strategy_slots,
      allowed_strategies: p.allowed_strategies,
      max_signals_per_day: p.max_signals_per_day,
      annual_savings_percent: p.annual_savings_percent,
      referral_eligible: p.referral_eligible,
    }));
  }
}
