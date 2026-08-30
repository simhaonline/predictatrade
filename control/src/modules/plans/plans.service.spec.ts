import { PlansService } from './plans.service';
import { jest } from '@jest/globals';

describe('PlansService', () => {
  it('returns configured fees and effective referral rates without rewriting plan rows', async () => {
    const pool = {
      query: jest.fn().mockResolvedValue({ rows: [{
        code: 'PRO', monthly_price: '299.00000000', annual_price: '2990.00000000',
        referral_rates: { '1': '0.15', '2': '0.04', '3': '0.02' },
      }] }),
    };
    const service = new PlansService(pool as any);
    const result = await service.listActive();
    expect(result[0]).toMatchObject({
      code: 'PRO', monthly_price: '299.00000000', annual_price: '2990.00000000',
      annual_savings_percent: 16.67, referral_eligible: true,
    });
    expect(result[0].referral_rates).toEqual({ '1': '0.15', '2': '0.04', '3': '0.02' });
    expect(pool.query).toHaveBeenCalledWith(expect.stringContaining('commission_rules'));
  });

  it('marks Free as non-commissionable when the API row has no referral rates', async () => {
    const pool = { query: jest.fn().mockResolvedValue({ rows: [{ code: 'FREE', monthly_price: '0', annual_price: null, referral_rates: {} }] }) };
    const service = new PlansService(pool as any);
    const result = await service.listActive();
    expect(result[0].referral_eligible).toBe(false);
    expect(result[0].annual_savings_percent).toBeNull();
  });
});
