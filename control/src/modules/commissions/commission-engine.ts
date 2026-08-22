// Commission Engine — SOW Section 69
// Implements the critical second-payment L1-only rule.
// All monetary calculations use decimal.js for exact arithmetic (SOW Section 69.27).
import Decimal from 'decimal.js';

// SOW Section 69.9: Purchase type classification
export type PurchaseType = 'FIRST_PURCHASE' | 'SECOND_PURCHASE' | 'RECURRING_PURCHASE';
export type CommercialEventType =
  | 'FREE_REGISTRATION' | 'NEW_SUBSCRIPTION' | 'RENEWAL' | 'UPGRADE' | 'DOWNGRADE'
  | 'CANCELLATION' | 'REACTIVATION' | 'ADD_ON' | 'REFUND' | 'PARTIAL_REFUND'
  | 'CHARGEBACK' | 'PAYMENT_FAILED';

// SOW Section 69.20: Commission ledger entry
export interface CommissionEntry {
  recipientUserId: string;
  sourceUserId: string;
  sourceSubscriptionId: string;
  purchaseId: string;
  invoiceId: string;
  planId: string;
  planVersion: number;
  purchaseNumber: number;
  purchaseType: PurchaseType;
  level: number;
  baseCommissionRate: Decimal;
  purchaseMultiplier: Decimal;
  effectiveCommissionRate: Decimal;
  commissionableAmount: Decimal;
  commissionAmount: Decimal;
  currency: string;
  status: 'PENDING';
  ruleSnapshot: CommissionRuleSnapshot;
  purchaseRuleSnapshot: PurchaseRuleSnapshot;
  createdAt: Date;
}

export interface CommissionRuleSnapshot {
  planId: string;
  level: number;
  baseRate: Decimal;
  ruleVersion: number;
}

export interface PurchaseRuleSnapshot {
  purchaseType: PurchaseType;
  multiplier: Decimal;
  maxReferralLevel: number;
  ruleVersion: number;
}

export interface CommissionInput {
  planId: string;
  commissionableAmount: number;
  paymentNumber: number;
  sponsorChain: string[]; // [L1 userId, L2 userId, ...]
  sourceUserId: string;
  sourceSubscriptionId: string;
  purchaseId: string;
  invoiceId: string;
  /** Explicit event prevents FREE registration from being treated as a purchase. */
  eventType?: CommercialEventType;
  eligibleAmount?: number | string;
}

export interface CommissionResult {
  commissions: CommissionEntry[];
  purchaseType: PurchaseType;
}

interface PurchaseRule {
  multiplier: Decimal;
  maxReferralLevel: number;
  ruleVersion: number;
}

// SOW Section 69.27: Use exact decimal arithmetic. Configure rounding.
Decimal.set({ precision: 20, rounding: Decimal.ROUND_HALF_UP });

export class CommissionEngine {
  private baseRates: Map<string, Decimal[]> = new Map(); // planId -> [L1, L2, L3, L4, L5]
  private purchaseRules: Map<PurchaseType, PurchaseRule> = new Map();
  private ruleVersions: Map<string, number> = new Map();

  // SOW Section 69.8: Set base commission rates for a plan
  setBaseRates(planId: string, rates: number[]): void {
    const decimalRates = rates.map(r => new Decimal(r));
    this.baseRates.set(planId, decimalRates);
    this.ruleVersions.set(planId, 1);
  }

  // SOW Section 69.9-69.12: Set purchase commission rules
  setPurchaseRule(purchaseType: PurchaseType, rule: { multiplier: number; maxReferralLevel: number }): void {
    this.purchaseRules.set(purchaseType, {
      multiplier: new Decimal(rule.multiplier),
      maxReferralLevel: rule.maxReferralLevel,
      ruleVersion: 1,
    });
  }

  // Main calculation method
  calculate(input: CommissionInput): CommissionResult {
    if (input.eventType === 'FREE_REGISTRATION' || input.eventType === 'REFUND' ||
        input.eventType === 'PARTIAL_REFUND' || input.eventType === 'CHARGEBACK' ||
        input.eventType === 'PAYMENT_FAILED' || input.eventType === 'DOWNGRADE' ||
        input.eventType === 'CANCELLATION') {
      return { commissions: [], purchaseType: this.classifyPurchaseType(input.paymentNumber) };
    }
    const commissionableAmount = new Decimal(input.eligibleAmount ?? input.commissionableAmount);

    // SOW Section 69.6: No commission if commissionable amount is zero
    if (commissionableAmount.isZero() || commissionableAmount.isNegative()) {
      return { commissions: [], purchaseType: this.classifyPurchaseType(input.paymentNumber) };
    }

    // SOW Section 69.9: Classify the purchase type
    const purchaseType = this.classifyPurchaseType(input.paymentNumber);
    const purchaseRule = this.purchaseRules.get(purchaseType);
    if (!purchaseRule) {
      return { commissions: [], purchaseType };
    }

    // Get base rates for the plan
    const baseRates = this.baseRates.get(input.planId);
    if (!baseRates) {
      return { commissions: [], purchaseType };
    }

    const commissions: CommissionEntry[] = [];
    const now = new Date();
    const planVersion = this.ruleVersions.get(input.planId) || 1;

    // SOW Section 69.10: Second payment pays L1 ONLY
    // SOW Section 69.11: Third+ pays L1-L5 at 50% multiplier
    const maxLevel = Math.min(purchaseRule.maxReferralLevel, input.sponsorChain.length, 5);

    for (let level = 1; level <= maxLevel; level++) {
      const recipientUserId = input.sponsorChain[level - 1];
      if (!recipientUserId) continue;

      const baseRate = baseRates[level - 1];
      if (!baseRate || baseRate.isZero()) continue;

      // SOW Section 69.12: effective_commission_rate = base_rate × purchase_multiplier
      const effectiveRate = baseRate.mul(purchaseRule.multiplier);

      // SOW Section 69.12: commission_amount = commissionable_amount × effective_rate
      const commissionAmount = commissionableAmount.mul(effectiveRate)
        .toDecimalPlaces(2); // Round to 2 decimal places (cents)

      if (commissionAmount.isZero() || commissionAmount.isNegative()) continue;

      const ruleSnapshot: CommissionRuleSnapshot = {
        planId: input.planId,
        level,
        baseRate: baseRate,
        ruleVersion: planVersion,
      };

      const purchaseRuleSnapshot: PurchaseRuleSnapshot = {
        purchaseType,
        multiplier: purchaseRule.multiplier,
        maxReferralLevel: purchaseRule.maxReferralLevel,
        ruleVersion: purchaseRule.ruleVersion,
      };

      commissions.push({
        recipientUserId,
        sourceUserId: input.sourceUserId,
        sourceSubscriptionId: input.sourceSubscriptionId,
        purchaseId: input.purchaseId,
        invoiceId: input.invoiceId,
        planId: input.planId,
        planVersion,
        purchaseNumber: input.paymentNumber,
        purchaseType,
        level,
        baseCommissionRate: baseRate,
        purchaseMultiplier: purchaseRule.multiplier,
        effectiveCommissionRate: effectiveRate,
        commissionableAmount,
        commissionAmount,
        currency: 'USD',
        status: 'PENDING',
        ruleSnapshot,
        purchaseRuleSnapshot,
        createdAt: now,
      });
    }

    return { commissions, purchaseType };
  }

  // SOW Section 69.9: Classify payment number to purchase type
  private classifyPurchaseType(paymentNumber: number): PurchaseType {
    if (paymentNumber === 1) return 'FIRST_PURCHASE';
    if (paymentNumber === 2) return 'SECOND_PURCHASE';
    return 'RECURRING_PURCHASE';
  }
}
