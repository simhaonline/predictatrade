export const STRATEGIES = [
  'STANDARD_SCALPING',
  'ULTRA_SCALPING',
  'STANDARD_SWING',
  'TREND_SWING',
  'MARNIE_FIB',
] as const;

export type Strategy = (typeof STRATEGIES)[number];
export type CommercialPlan = 'FREE' | 'STANDARD' | 'PRO' | 'ELITE';

export interface PlanPolicy {
  code: CommercialPlan;
  allowedStrategies: readonly string[];
  maxActiveStrategies: number;
}

export interface StrategyDecision {
  allowed: boolean;
  selected: Strategy[];
  reason?: string;
}

/** Authoritative backend validator. It is independent of signal generation. */
export function validateStrategySelection(
  policy: PlanPolicy,
  requested: readonly string[] | undefined,
): StrategyDecision {
  const selected = [...new Set(requested ?? [])] as string[];
  if (selected.length === 0) {
    return { allowed: false, selected: [], reason: 'AT_LEAST_ONE_STRATEGY_REQUIRED' };
  }
  if (selected.some((s) => !STRATEGIES.includes(s as Strategy))) {
    return { allowed: false, selected: [], reason: 'UNKNOWN_STRATEGY' };
  }
  if (selected.some((s) => !policy.allowedStrategies.includes(s))) {
    return { allowed: false, selected: [], reason: 'STRATEGY_NOT_ENTITLED' };
  }
  if (selected.length > policy.maxActiveStrategies) {
    return { allowed: false, selected: [], reason: 'plan_strategy_limit' };
  }
  return { allowed: true, selected: selected as Strategy[] };
}

export function planPolicyFromRow(row: {
  code: CommercialPlan;
  allowed_strategies?: unknown;
  max_active_strategy_slots?: number;
}): PlanPolicy {
  const allowed = Array.isArray(row.allowed_strategies) ? row.allowed_strategies : [];
  return {
    code: row.code,
    allowedStrategies: allowed.map(String),
    maxActiveStrategies: Number(row.max_active_strategy_slots ?? 0),
  };
}
