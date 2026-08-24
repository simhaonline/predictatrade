/**
 * Plan-aware feature & indicator visibility (prompt.md ADDENDUM 5, F3).
 * Visibility gating is presentation-layer only: the engine always computes the
 * full feature set for every plan and capital protection is never relaxed.
 */

export const FEATURE_ACCESS_LEVELS = ['core', 'advanced', 'smc', 'full'] as const;

export type FeatureAccessLevel = (typeof FEATURE_ACCESS_LEVELS)[number];

export const FEATURE_GROUPS = [
  'core_indicators',
  'advanced_technical',
  'structure_smc',
  'cross_market_view',
  'cross_market_full',
  'marnie_fib_evidence',
] as const;

export type FeatureGroup = (typeof FEATURE_GROUPS)[number];

export function isFeatureAccessLevel(v: unknown): v is FeatureAccessLevel {
  return typeof v === 'string' && (FEATURE_ACCESS_LEVELS as readonly string[]).includes(v);
}

/** Minimum plan level required to see a feature group. */
export const FEATURE_GROUP_MIN_LEVEL: Record<FeatureGroup, FeatureAccessLevel> = {
  core_indicators: 'core',
  advanced_technical: 'advanced',
  structure_smc: 'smc',
  cross_market_view: 'smc',
  cross_market_full: 'full',
  marnie_fib_evidence: 'full',
};

/** Feature groups visible per access level (ADDENDUM 5 table). */
export const PLAN_FEATURE_ACCESS: Record<FeatureAccessLevel, readonly FeatureGroup[]> = {
  core: ['core_indicators'],
  advanced: ['core_indicators', 'advanced_technical'],
  smc: ['core_indicators', 'advanced_technical', 'structure_smc', 'cross_market_view'],
  full: [...FEATURE_GROUPS],
};

export function hasFeatureAccess(level: unknown, group: FeatureGroup): boolean {
  if (!isFeatureAccessLevel(level)) return false;
  return PLAN_FEATURE_ACCESS[level].includes(group);
}

export function visibleFeatureGroups(level: unknown): FeatureGroup[] {
  if (!isFeatureAccessLevel(level)) return [...PLAN_FEATURE_ACCESS.core];
  return [...PLAN_FEATURE_ACCESS[level]];
}
