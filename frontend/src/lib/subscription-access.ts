// Subscription-aware access control for signals & analytics visibility.
// Keeps the UI consistent with the user's plan: lower tiers see a limited,
// clearly-labeled subset of strategy analytics rather than the full grid.

export type PlanTier = "FREE" | "BASIC" | "PRO" | "ELITE" | "ENTERPRISE" | string;

export interface SubscriptionContext {
  planName?: string | null;
  planId?: string | null;
  status?: string | null; // license status
}

/** Normalize a plan name into a comparable tier rank. */
export function planRank(plan?: string | null): number {
  if (!plan) return 0;
  const p = plan.toUpperCase();
  if (p.includes("ENTERPRISE")) return 4;
  if (p.includes("ELITE")) return 3;
  if (p.includes("PRO")) return 2;
  if (p.includes("BASIC") || p.includes("STARTER")) return 1;
  return 0; // FREE / unknown
}

export function isActiveSubscription(ctx?: SubscriptionContext): boolean {
  return !!ctx && (ctx.status === "ACTIVE" || ctx.status === "TRIAL") && planRank(ctx.planName) >= 1;
}

/**
 * Given the full ranking, return the subset visible to the user's plan.
 * - FREE: only the overall top strategy (teaser), rest locked.
 * - BASIC: top 5 strategies.
 * - PRO+: all strategies, plus full analytics.
 */
export function visibleStrategies<T>(rows: T[], ctx?: SubscriptionContext): {
  visible: T[];
  lockedCount: number;
  tier: number;
} {
  const tier = planRank(ctx?.planName);
  let limit = rows.length;
  if (tier <= 0) limit = Math.min(1, rows.length);      // FREE teaser
  else if (tier === 1) limit = Math.min(5, rows.length); // BASIC
  // PRO (2), ELITE (3), ENTERPRISE (4) -> all
  return {
    visible: rows.slice(0, limit),
    lockedCount: Math.max(0, rows.length - limit),
    tier,
  };
}

export function analyticsUnlocked(ctx?: SubscriptionContext): boolean {
  return planRank(ctx?.planName) >= 2; // PRO and above
}
