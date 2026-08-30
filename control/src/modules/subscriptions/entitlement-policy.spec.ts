import { planPolicyFromRow, validateStrategySelection } from './entitlement-policy';
import { jest } from '@jest/globals';

const FOUR = ['STANDARD_SCALPING', 'ULTRA_SCALPING', 'STANDARD_SWING', 'TREND_SWING'];
const ALL = [...FOUR, 'MARNIE_FIB'];

const policy = (code: any, allowed: string[], max: number) =>
  planPolicyFromRow({ code, allowed_strategies: allowed, max_active_strategy_slots: max });

// Bug 7 canonical plan policies as seeded by migration 067.
describe('commercial strategy entitlement policy', () => {
  it('allows only standard swing for Free with max_signals_per_day handled at delivery', () => {
    const p = policy('FREE', ['STANDARD_SWING'], 1);
    expect(validateStrategySelection(p, ['STANDARD_SWING']).allowed).toBe(true);
    expect(validateStrategySelection(p, ['STANDARD_SCALPING']).reason).toBe('STRATEGY_NOT_ENTITLED');
  });

  it('rejects a Standard user selecting 2 strategies (plan_strategy_limit)', () => {
    const p = policy('STANDARD', FOUR, 1);
    expect(validateStrategySelection(p, ['TREND_SWING']).allowed).toBe(true);
    expect(validateStrategySelection(p, ['STANDARD_SCALPING', 'STANDARD_SWING']).reason).toBe('plan_strategy_limit');
  });

  it('allows at most two Pro strategies and all five Elite strategies', () => {
    expect(validateStrategySelection(policy('PRO', FOUR, 2), FOUR.slice(0, 2)).allowed).toBe(true);
    expect(validateStrategySelection(policy('PRO', FOUR, 2), FOUR.slice(0, 3)).reason).toBe('plan_strategy_limit');
    expect(validateStrategySelection(policy('ELITE', ALL, 5), ALL).allowed).toBe(true);
  });

  it('deduplicates selections but rejects empty and unknown input', () => {
    const p = policy('PRO', ['STANDARD_SCALPING', 'ULTRA_SCALPING'], 2);
    expect(validateStrategySelection(p, ['STANDARD_SCALPING', 'STANDARD_SCALPING']).selected).toEqual(['STANDARD_SCALPING']);
    expect(validateStrategySelection(p, []).reason).toBe('AT_LEAST_ONE_STRATEGY_REQUIRED');
    expect(validateStrategySelection(p, ['NOT_A_STRATEGY']).reason).toBe('UNKNOWN_STRATEGY');
  });
});
