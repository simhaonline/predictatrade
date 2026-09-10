/**
 * RiskEngine tests — the strict 10% maximum capital stop-loss framework:
 * lot sizing from stop distance, micro-lot refusal, rounding-down guard,
 * automated hedge sizing + combined-envelope enforcement.
 */
import { planMicroTrade, validateAccount, MAX_CAPITAL_RISK_PCT, type AccountContext } from './risk-engine';

const ACCOUNT: AccountContext = {
  equity: 3400,
  symbol: 'XAUUSD',
  valuePerPointPerLot: 1,   // $1 per point per lot (XAUUSD 0.01 spec)
  minLot: 0.01,
  lotStep: 0.01,
  maxLot: 2,
};

describe('RiskEngine — 10% capital stop-loss framework', () => {
  it('exposes the hard 10% cap constant', () => {
    expect(MAX_CAPITAL_RISK_PCT).toBe(10);
  });

  it('sizes lots so capital at risk stays ≤ 10% of equity', () => {
    // equity 3400 → max risk $340 → stop 8 points → lots = 340/8 = 42.5 → capped by maxLot 2
    const acc = { ...ACCOUNT, maxLot: 2 };
    const plan = planMicroTrade(acc, 'BUY', 3400, 3392);
    expect(plan.refused).toBeUndefined();
    expect(plan.lots).toBe(2);
    expect(plan.capitalAtRisk).toBe(16);      // 2 lots × 8 pts × $1
    expect(plan.capitalAtRiskPct).toBeCloseTo(0.47, 2);
    expect(plan.capitalAtRiskPct).toBeLessThanOrEqual(MAX_CAPITAL_RISK_PCT);
  });

  it('exactly fills the 10% envelope on a small account', () => {
    const acc = { ...ACCOUNT, equity: 600, maxLot: 100 };
    // max risk $60, stop 6 points → 10 lots exactly
    const plan = planMicroTrade(acc, 'SELL', 3400, 3394, { rrTargets: [1, 2] });
    expect(plan.lots).toBeCloseTo(10, 2);
    expect(plan.capitalAtRisk).toBeCloseTo(60, 2);
    expect(plan.capitalAtRiskPct).toBeCloseTo(10, 2);
  });

  it('rounds DOWN to lot step and stays within the cap', () => {
    const acc = { ...ACCOUNT, equity: 1000, maxLot: 50 }; // max risk $100, stop 3 → 33.33 lots
    const plan = planMicroTrade(acc, 'BUY', 100, 97);
    expect(plan.lots).toBeCloseTo(33.33, 2); // step 0.01 floor
    expect(plan.capitalAtRisk).toBeLessThanOrEqual(100.0001); // ≤ 10% of 1000
    expect(plan.capitalAtRiskPct).toBeLessThanOrEqual(MAX_CAPITAL_RISK_PCT);
  });

  it('refuses when computed size is below broker min lot (never upsizes)', () => {
    const acc = { ...ACCOUNT, equity: 50, maxLot: 2 }; // max risk $5, stop 200 pts → 0.025 → 0.02 ≥ 0.01 OK
    const tiny = { ...acc, equity: 10 }; // max risk $1, stop 200 → 0.025 lots → 0.02, min 0.01 OK
    const refuse = planMicroTrade(
      { ...ACCOUNT, equity: 5, maxLot: 2 }, // max risk $0.50, stop 10 → 0.05 lots → 0.05 OK
      'BUY', 3400, 3390,
    );
    void refuse;
    // to actually hit the refusal: equity 2, stop 10 → max risk 0.2 → 0.02 lots → still ≥ 0.01
    // use stop 10 with equity 0.5 → max risk 0.05 → 0.005 lots < 0.01 → REFUSED
    const plan = planMicroTrade({ ...ACCOUNT, equity: 0.5, maxLot: 2 }, 'BUY', 3400, 3390);
    expect(plan.refused).toBeDefined();
    expect(plan.lots).toBe(0);
    expect(plan.warnings.join(' ')).toContain('below broker min');
    void tiny;
  });

  it('refuses invalid stop geometry (< 1 point)', () => {
    const plan = planMicroTrade(ACCOUNT, 'BUY', 3400, 3399.9);
    expect(plan.refused).toBe('Invalid stop geometry');
  });

  it('builds the TP ladder at 1R/2R/3R in the trade direction', () => {
    const plan = planMicroTrade(ACCOUNT, 'BUY', 3400, 3390); // 10 pt stop
    expect(plan.takeProfits[0]).toBeCloseTo(3410, 6);
    expect(plan.takeProfits[1]).toBeCloseTo(3420, 6);
    expect(plan.takeProfits[2]).toBeCloseTo(3430, 6);
    expect(plan.rrTp1).toBeCloseTo(1, 6);
  });

  it('automated hedge: envelope split — primary 70%, hedge 30%, combined ≤ 10%', () => {
    const acc = { ...ACCOUNT, equity: 600, maxLot: 50 };
    const plan = planMicroTrade(acc, 'BUY', 3400, 3390, { hedgePct: 30 });
    expect(plan.hedge).toBeDefined();
    expect(plan.hedge!.direction).toBe('SELL');
    // envelope = $60 (10% of 600); split 70/30: primary risk $42 → 4.2 lots,
    // hedge $18 → 1.8 lots; combined $60 = exactly 10%
    expect(plan.lots).toBeCloseTo(4.2, 2);
    expect(plan.hedge!.lots).toBeCloseTo(1.8, 2);
    expect(plan.hedge!.combinedRiskPct).toBeCloseTo(10, 4); // combined includes primary — ≤ 10% cap
  });

  it('automated hedge is SKIPPED when it computes below min lot (tiny equity)', () => {
    const acc = { ...ACCOUNT, equity: 3, maxLot: 50 }; // envelope $0.30, hedge 30% = $0.09 → 0.009 lots < 0.01
    const plan = planMicroTrade(acc, 'BUY', 3400, 3390, { hedgePct: 30 });
    expect(plan.hedge).toBeUndefined();
    expect(plan.warnings.join(' ')).toContain('below min lot');
  });

  it('TP ladder direction is inverted for SELL', () => {
    const plan = planMicroTrade({ ...ACCOUNT, equity: 600, maxLot: 50 }, 'SELL', 3400, 3410);
    expect(plan.takeProfits[0]).toBeCloseTo(3390, 6); // 1R below entry
  });

  it('validateAccount refuses broken broker specs', () => {
    expect(validateAccount({ ...ACCOUNT, equity: 0 })).toContain('equity');
    expect(validateAccount({ ...ACCOUNT, valuePerPointPerLot: 0 })).toContain('point value');
    expect(validateAccount({ ...ACCOUNT, lotStep: 0 })).toContain('lot spec');
    expect(validateAccount(ACCOUNT)).toBeNull();
  });
});