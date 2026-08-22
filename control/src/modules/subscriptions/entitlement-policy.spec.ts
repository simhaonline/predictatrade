import { planPolicyFromRow, validateStrategySelection } from './entitlement-policy';

const policy = (code: any, allowed: string[], max: number) =>
  planPolicyFromRow({ code, allowed_strategies: allowed, max_active_strategy_slots: max });

describe('commercial strategy entitlement policy', () => {
  it('allows only standard scalping for Free', () => {
    expect(validateStrategySelection(policy('FREE', ['STANDARD_SCALPING'], 1), ['STANDARD_SCALPING']).allowed).toBe(true);
    expect(validateStrategySelection(policy('FREE', ['STANDARD_SCALPING'], 1), ['STANDARD_SWING']).reason).toBe('STRATEGY_NOT_ENTITLED');
  });

  it('requires exactly one Standard strategy', () => {
    const p = policy('STANDARD', ['STANDARD_SCALPING', 'STANDARD_SWING'], 1);
    expect(validateStrategySelection(p, ['STANDARD_SCALPING']).allowed).toBe(true);
    expect(validateStrategySelection(p, ['STANDARD_SCALPING', 'STANDARD_SWING']).reason).toBe('STRATEGY_LIMIT_EXCEEDED');
    expect(validateStrategySelection(p, ['TREND_SWING']).allowed).toBe(false);
  });

  it('allows at most two Pro strategies and all four Elite strategies', () => {
    const all = ['STANDARD_SCALPING', 'ULTRA_SCALPING', 'STANDARD_SWING', 'TREND_SWING'];
    expect(validateStrategySelection(policy('PRO', all, 2), all.slice(0, 2)).allowed).toBe(true);
    expect(validateStrategySelection(policy('PRO', all, 2), all.slice(0, 3)).reason).toBe('STRATEGY_LIMIT_EXCEEDED');
    expect(validateStrategySelection(policy('ELITE', all, 4), all).allowed).toBe(true);
  });

  it('deduplicates selections but rejects empty and unknown input', () => {
    const p = policy('PRO', ['STANDARD_SCALPING', 'ULTRA_SCALPING'], 2);
    expect(validateStrategySelection(p, ['STANDARD_SCALPING', 'STANDARD_SCALPING']).selected).toEqual(['STANDARD_SCALPING']);
    expect(validateStrategySelection(p, []).reason).toBe('AT_LEAST_ONE_STRATEGY_REQUIRED');
    expect(validateStrategySelection(p, ['NOT_A_STRATEGY']).reason).toBe('UNKNOWN_STRATEGY');
  });
});
