import { planPolicyFromRow, validateStrategySelection } from './entitlement-policy';
import { jest } from '@jest/globals';

const FOUR = ['STANDARD_SCALPING', 'ULTRA_SCALPING', 'STANDARD_SWING', 'TREND_SWING'];
const SIX = [...FOUR, 'MARNIE_FIB', 'ATEN'];

const policy = (code: any, allowed: string[], max: number) =>
  planPolicyFromRow({ code, allowed_strategies: allowed, max_active_strategy_slots: max });

// Canonical plan policies per migration 110 (MASTER PROMPT spec):
//   FREE 1 slot (STANDARD_SCALPING only), STANDARD 2, PRO 4, ELITE 6.
// max_signals_per_day (FREE = 5) is enforced at delivery, not here.
describe('commercial strategy entitlement policy', () => {
  it('allows only STANDARD_SCALPING for Free (migration 110)', () => {
    const p = policy('FREE', ['STANDARD_SCALPING'], 1);
    expect(validateStrategySelection(p, ['STANDARD_SCALPING']).allowed).toBe(true);
    expect(validateStrategySelection(p, ['STANDARD_SWING']).reason).toBe('STRATEGY_NOT_ENTITLED');
  });

  it('allows two Standard strategies (2 slots) and rejects a third (plan_strategy_limit)', () => {
    const p = policy('STANDARD', ['STANDARD_SCALPING', 'STANDARD_SWING'], 2);
    expect(validateStrategySelection(p, ['STANDARD_SCALPING', 'STANDARD_SWING']).allowed).toBe(true);
    expect(validateStrategySelection(p, ['STANDARD_SCALPING', 'ULTRA_SCALPING']).reason).toBe('STRATEGY_NOT_ENTITLED');
    // All entitled strategies beyond the 2-slot cap → plan_strategy_limit.
    const p3 = policy('STANDARD', FOUR, 2);
    expect(validateStrategySelection(p3, FOUR).reason).toBe('plan_strategy_limit');
  });

  it('allows up to four Pro strategies (4 slots) and rejects non-entitled strategies', () => {
    const p = policy('PRO', FOUR, 4);
    expect(validateStrategySelection(p, FOUR).allowed).toBe(true);
    expect(validateStrategySelection(p, [...FOUR, 'MARNIE_FIB']).reason).toBe('STRATEGY_NOT_ENTITLED');
  });

  it('allows all six Elite strategies (6 slots)', () => {
    expect(validateStrategySelection(policy('ELITE', SIX, 6), SIX).allowed).toBe(true);
  });

  it('deduplicates selections but rejects empty and unknown input', () => {
    const p = policy('PRO', ['STANDARD_SCALPING', 'ULTRA_SCALPING'], 2);
    expect(validateStrategySelection(p, ['STANDARD_SCALPING', 'STANDARD_SCALPING']).selected).toEqual(['STANDARD_SCALPING']);
    expect(validateStrategySelection(p, []).reason).toBe('AT_LEAST_ONE_STRATEGY_REQUIRED');
    expect(validateStrategySelection(p, ['NOT_A_STRATEGY']).reason).toBe('UNKNOWN_STRATEGY');
  });
});