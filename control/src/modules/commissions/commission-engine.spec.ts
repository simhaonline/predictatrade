// Commission Engine Tests — SOW Section 69.33 Acceptance Criteria
// These tests verify the CRITICAL second-payment L1-only rule.
import { CommissionEngine, CommissionInput, CommissionResult } from './commission-engine';

describe('CommissionEngine', () => {
  let engine: CommissionEngine;

  // SOW Section 69.34 — Initial Production Configuration
  const STANDARD_PLAN_ID = 'plan-standard';
  const PRO_PLAN_ID = 'plan-pro';
  const ELITE_PLAN_ID = 'plan-elite';

  beforeEach(() => {
    engine = new CommissionEngine();

    // Seed commission rules (SOW Section 69.8)
    // STANDARD: [10%, 4%, 2%, 1%, 0.5%]
    engine.setBaseRates(STANDARD_PLAN_ID, [0.10, 0.04, 0.02, 0.01, 0.005]);
    // PRO: [15%, 5%, 3%, 2%, 1%]
    engine.setBaseRates(PRO_PLAN_ID, [0.15, 0.05, 0.03, 0.02, 0.01]);
    // ELITE: [20%, 6%, 4%, 2%, 1%]
    engine.setBaseRates(ELITE_PLAN_ID, [0.20, 0.06, 0.04, 0.02, 0.01]);

    // Purchase commission rules (SOW Section 69.9-69.12)
    engine.setPurchaseRule('FIRST_PURCHASE', { multiplier: 1.00, maxReferralLevel: 5 });
    engine.setPurchaseRule('SECOND_PURCHASE', { multiplier: 0.75, maxReferralLevel: 1 });
    engine.setPurchaseRule('RECURRING_PURCHASE', { multiplier: 0.50, maxReferralLevel: 5 });
  });

  describe('First Payment (100% multiplier, L1-L5)', () => {
    it('STANDARD first payment $99 — L1=10%, L2=4%, L3=2%, L4=1%, L5=0.5%', () => {
      const result = engine.calculate({
        planId: STANDARD_PLAN_ID,
        commissionableAmount: 99,
        paymentNumber: 1,
        sponsorChain: ['user-L1', 'user-L2', 'user-L3', 'user-L4', 'user-L5'],
        sourceUserId: 'source-user',
        sourceSubscriptionId: 'sub-1',
        purchaseId: 'pay-1',
        invoiceId: 'inv-1',
      });

      expect(result.commissions).toHaveLength(5);
      // L1: 99 * 10% * 100% = $9.90
      expect(result.commissions[0].level).toBe(1);
      expect(result.commissions[0].commissionAmount.toNumber()).toBeCloseTo(9.90, 2);
      // L2: 99 * 4% * 100% = $3.96
      expect(result.commissions[1].level).toBe(2);
      expect(result.commissions[1].commissionAmount.toNumber()).toBeCloseTo(3.96, 2);
      // L3: 99 * 2% * 100% = $1.98
      expect(result.commissions[2].commissionAmount.toNumber()).toBeCloseTo(1.98, 2);
      // L4: 99 * 1% * 100% = $0.99
      expect(result.commissions[3].commissionAmount.toNumber()).toBeCloseTo(0.99, 2);
      // L5: 99 * 0.5% * 100% = $0.495 → rounds to $0.50
      expect(result.commissions[4].commissionAmount.toNumber()).toBeCloseTo(0.50, 2);
    });

    it('PRO first payment $499 — L1=15%, L2=5%, L3=3%, L4=2%, L5=1%', () => {
      const result = engine.calculate({
        planId: PRO_PLAN_ID,
        commissionableAmount: 499,
        paymentNumber: 1,
        sponsorChain: ['L1', 'L2', 'L3', 'L4', 'L5'],
        sourceUserId: 'source',
        sourceSubscriptionId: 'sub',
        purchaseId: 'pay',
        invoiceId: 'inv',
      });

      expect(result.commissions).toHaveLength(5);
      // L1: 499 * 15% = $74.85
      expect(result.commissions[0].commissionAmount.toNumber()).toBeCloseTo(74.85, 2);
      // L2: 499 * 5% = $24.95
      expect(result.commissions[1].commissionAmount.toNumber()).toBeCloseTo(24.95, 2);
      // L3: 499 * 3% = $14.97
      expect(result.commissions[2].commissionAmount.toNumber()).toBeCloseTo(14.97, 2);
      // L4: 499 * 2% = $9.98
      expect(result.commissions[3].commissionAmount.toNumber()).toBeCloseTo(9.98, 2);
      // L5: 499 * 1% = $4.99
      expect(result.commissions[4].commissionAmount.toNumber()).toBeCloseTo(4.99, 2);
    });

    it('ELITE first payment $999 — L1=20%, L2=6%, L3=4%, L4=2%, L5=1%', () => {
      const result = engine.calculate({
        planId: ELITE_PLAN_ID,
        commissionableAmount: 999,
        paymentNumber: 1,
        sponsorChain: ['L1', 'L2', 'L3', 'L4', 'L5'],
        sourceUserId: 'source',
        sourceSubscriptionId: 'sub',
        purchaseId: 'pay',
        invoiceId: 'inv',
      });

      expect(result.commissions).toHaveLength(5);
      expect(result.commissions[0].commissionAmount.toNumber()).toBeCloseTo(199.80, 2);
      expect(result.commissions[1].commissionAmount.toNumber()).toBeCloseTo(59.94, 2);
      expect(result.commissions[2].commissionAmount.toNumber()).toBeCloseTo(39.96, 2);
      expect(result.commissions[3].commissionAmount.toNumber()).toBeCloseTo(19.98, 2);
      expect(result.commissions[4].commissionAmount.toNumber()).toBeCloseTo(9.99, 2);
    });
  });

  describe('CRITICAL: Second Payment (75% multiplier, L1 ONLY)', () => {
    it('STANDARD second payment $99 — L1 only at 7.5%, L2-L5 = ZERO', () => {
      const result = engine.calculate({
        planId: STANDARD_PLAN_ID,
        commissionableAmount: 99,
        paymentNumber: 2,
        sponsorChain: ['L1', 'L2', 'L3', 'L4', 'L5'],
        sourceUserId: 'source',
        sourceSubscriptionId: 'sub',
        purchaseId: 'pay2',
        invoiceId: 'inv2',
      });

      // CRITICAL: Only L1 gets commission on second payment
      expect(result.commissions).toHaveLength(1);

      expect(result.commissions[0].level).toBe(1);
      // L1: 99 * 10% * 75% = 99 * 0.075 = $7.425 → rounds to $7.43
      expect(result.commissions[0].commissionAmount.toNumber()).toBeCloseTo(7.43, 2);
      expect(result.commissions[0].effectiveCommissionRate.toNumber()).toBeCloseTo(0.075, 4);
    });

    it('PRO second payment $499 — L1 only at 11.25%', () => {
      const result = engine.calculate({
        planId: PRO_PLAN_ID,
        commissionableAmount: 499,
        paymentNumber: 2,
        sponsorChain: ['L1', 'L2', 'L3', 'L4', 'L5'],
        sourceUserId: 'source',
        sourceSubscriptionId: 'sub',
        purchaseId: 'pay2',
        invoiceId: 'inv2',
      });

      expect(result.commissions).toHaveLength(1);
      // L1: 499 * 15% * 75% = 499 * 0.1125 = $56.1375 → $56.14
      expect(result.commissions[0].commissionAmount.toNumber()).toBeCloseTo(56.14, 2);
    });

    it('ELITE second payment $999 — L1 only at 15%', () => {
      const result = engine.calculate({
        planId: ELITE_PLAN_ID,
        commissionableAmount: 999,
        paymentNumber: 2,
        sponsorChain: ['L1', 'L2', 'L3', 'L4', 'L5'],
        sourceUserId: 'source',
        sourceSubscriptionId: 'sub',
        purchaseId: 'pay2',
        invoiceId: 'inv2',
      });

      expect(result.commissions).toHaveLength(1);
      // L1: 999 * 20% * 75% = 999 * 0.15 = $149.85
      expect(result.commissions[0].commissionAmount.toNumber()).toBeCloseTo(149.85, 2);
    });

    it('Second payment never creates L2-L5 commission records', () => {
      const result = engine.calculate({
        planId: ELITE_PLAN_ID,
        commissionableAmount: 999,
        paymentNumber: 2,
        sponsorChain: ['L1', 'L2', 'L3', 'L4', 'L5'],
        sourceUserId: 'source',
        sourceSubscriptionId: 'sub',
        purchaseId: 'pay2',
        invoiceId: 'inv2',
      });

      // Only 1 commission record (L1) — no L2-L5 records at all
      expect(result.commissions).toHaveLength(1);
      const levels = result.commissions.map(c => c.level);
      expect(levels).not.toContain(2);
      expect(levels).not.toContain(3);
      expect(levels).not.toContain(4);
      expect(levels).not.toContain(5);
    });
  });

  describe('Third+ Payment (50% multiplier, L1-L5)', () => {
    it('STANDARD third payment $99 — L1=5%, L2=2%, L3=1%, L4=0.5%, L5=0.25%', () => {
      const result = engine.calculate({
        planId: STANDARD_PLAN_ID,
        commissionableAmount: 99,
        paymentNumber: 3,
        sponsorChain: ['L1', 'L2', 'L3', 'L4', 'L5'],
        sourceUserId: 'source',
        sourceSubscriptionId: 'sub',
        purchaseId: 'pay3',
        invoiceId: 'inv3',
      });

      expect(result.commissions).toHaveLength(5);
      // L1: 99 * 10% * 50% = 99 * 0.05 = $4.95
      expect(result.commissions[0].commissionAmount.toNumber()).toBeCloseTo(4.95, 2);
      // L2: 99 * 4% * 50% = 99 * 0.02 = $1.98
      expect(result.commissions[1].commissionAmount.toNumber()).toBeCloseTo(1.98, 2);
      // L3: 99 * 2% * 50% = 99 * 0.01 = $0.99
      expect(result.commissions[2].commissionAmount.toNumber()).toBeCloseTo(0.99, 2);
    });

    it('Payment #10 still uses recurring 50% multiplier', () => {
      const result = engine.calculate({
        planId: STANDARD_PLAN_ID,
        commissionableAmount: 99,
        paymentNumber: 10,
        sponsorChain: ['L1', 'L2', 'L3', 'L4', 'L5'],
        sourceUserId: 'source',
        sourceSubscriptionId: 'sub',
        purchaseId: 'pay10',
        invoiceId: 'inv10',
      });

      expect(result.commissions).toHaveLength(5);
      // Same as payment #3: 50% multiplier
      expect(result.commissions[0].commissionAmount.toNumber()).toBeCloseTo(4.95, 2);
    });
  });

  describe('Partial Sponsor Chain', () => {
    it('Only 3 sponsors in chain — L1-L3 get commission, L4-L5 get nothing', () => {
      const result = engine.calculate({
        planId: STANDARD_PLAN_ID,
        commissionableAmount: 99,
        paymentNumber: 1,
        sponsorChain: ['L1', 'L2', 'L3'], // Only 3 sponsors
        sourceUserId: 'source',
        sourceSubscriptionId: 'sub',
        purchaseId: 'pay',
        invoiceId: 'inv',
      });

      expect(result.commissions).toHaveLength(3);
      expect(result.commissions[2].level).toBe(3);
    });

    it('No sponsors — no commission', () => {
      const result = engine.calculate({
        planId: STANDARD_PLAN_ID,
        commissionableAmount: 99,
        paymentNumber: 1,
        sponsorChain: [], // No sponsors
        sourceUserId: 'source',
        sourceSubscriptionId: 'sub',
        purchaseId: 'pay',
        invoiceId: 'inv',
      });

      expect(result.commissions).toHaveLength(0);
    });
  });

  describe('Setup Fee Not Commissionable (SOW 69.4)', () => {
    it('Setup fee produces zero commission', () => {
      const result = engine.calculate({
        planId: STANDARD_PLAN_ID,
        commissionableAmount: 0, // Setup fee is excluded from commissionable amount
        paymentNumber: 1,
        sponsorChain: ['L1', 'L2', 'L3', 'L4', 'L5'],
        sourceUserId: 'source',
        sourceSubscriptionId: 'sub',
        purchaseId: 'pay',
        invoiceId: 'inv',
      });

      expect(result.commissions).toHaveLength(0);
    });
  });

  describe('v3 configuration and explicit commercial events', () => {
    it('supports v3 rates/depth without changing legacy engine configuration', () => {
      const v3 = new CommissionEngine();
      v3.setBaseRates('v3-standard', [0.10, 0.03, 0.01]);
      v3.setPurchaseRule('FIRST_PURCHASE', { multiplier: 1, maxReferralLevel: 3 });
      const result = v3.calculate({
        planId: 'v3-standard', commissionableAmount: 99, paymentNumber: 1,
        sponsorChain: ['L1', 'L2', 'L3', 'L4'], sourceUserId: 'source',
        sourceSubscriptionId: 'sub', purchaseId: 'payment', invoiceId: 'invoice',
        eventType: 'NEW_SUBSCRIPTION', eligibleAmount: '80.00',
      });
      expect(result.commissions.map((c) => c.level)).toEqual([1, 2, 3]);
      expect(result.commissions.map((c) => c.commissionAmount.toFixed(2))).toEqual(['8.00', '2.40', '0.80']);
    });

    it('never treats free registration as an eligible paid event', () => {
      const result = engine.calculate({
        planId: STANDARD_PLAN_ID, commissionableAmount: 0, paymentNumber: 0,
        sponsorChain: ['L1', 'L2'], sourceUserId: 'source', sourceSubscriptionId: 'sub',
        purchaseId: 'free', invoiceId: 'none', eventType: 'FREE_REGISTRATION',
      });
      expect(result.commissions).toHaveLength(0);
    });
  });

  describe('Commission Amount Snapshots (SOW 69.20)', () => {
    it('Each commission record contains rule snapshot', () => {
      const result = engine.calculate({
        planId: STANDARD_PLAN_ID,
        commissionableAmount: 99,
        paymentNumber: 1,
        sponsorChain: ['L1'],
        sourceUserId: 'source',
        sourceSubscriptionId: 'sub',
        purchaseId: 'pay',
        invoiceId: 'inv',
      });

      expect(result.commissions).toHaveLength(1);
      const c = result.commissions[0];
      expect(c.baseCommissionRate.toNumber()).toBeCloseTo(0.10, 4);
      expect(c.purchaseMultiplier.toNumber()).toBeCloseTo(1.00, 4);
      expect(c.effectiveCommissionRate.toNumber()).toBeCloseTo(0.10, 4);
      expect(c.commissionableAmount.toNumber()).toBe(99);
      expect(c.ruleSnapshot).toBeDefined();
      expect(c.purchaseType).toBe('FIRST_PURCHASE');
    });
  });

  describe('Payment Number Classification (SOW 69.9)', () => {
    it('Payment #1 = FIRST_PURCHASE', () => {
      const result = engine.calculate({
        planId: STANDARD_PLAN_ID, commissionableAmount: 99, paymentNumber: 1,
        sponsorChain: ['L1'], sourceUserId: 's', sourceSubscriptionId: 'sub', purchaseId: 'p', invoiceId: 'i',
      });
      expect(result.purchaseType).toBe('FIRST_PURCHASE');
    });

    it('Payment #2 = SECOND_PURCHASE', () => {
      const result = engine.calculate({
        planId: STANDARD_PLAN_ID, commissionableAmount: 99, paymentNumber: 2,
        sponsorChain: ['L1'], sourceUserId: 's', sourceSubscriptionId: 'sub', purchaseId: 'p', invoiceId: 'i',
      });
      expect(result.purchaseType).toBe('SECOND_PURCHASE');
    });

    it('Payment #3 = RECURRING_PURCHASE', () => {
      const result = engine.calculate({
        planId: STANDARD_PLAN_ID, commissionableAmount: 99, paymentNumber: 3,
        sponsorChain: ['L1'], sourceUserId: 's', sourceSubscriptionId: 'sub', purchaseId: 'p', invoiceId: 'i',
      });
      expect(result.purchaseType).toBe('RECURRING_PURCHASE');
    });
  });
});
