/**
 * RiskEngine — the strict 10% maximum capital stop-loss allocation framework
 * for micro-trade execution planning in the Telegram assistant.
 *
 * HARD RULES (fail-closed):
 *   1. Max capital at risk per position: 10% of account equity (operator cap,
 *      never configurable above 10 from any chat surface).
 *   2. Stop distance defines lot size: lots = (equity * 10% * pctUsed) /
 *      (stopPoints * valuePerPointPerLot). Bounded by broker micro-lot
 *      (0.01) minimum — if the computed size < min lot, the trade is REFUSED
 *      (no fudging sizes up; nothing trades rather than over-risking).
 *   3. Automated hedging: a counter-position hedge of `hedgePct` of the
 *      primary risk, taken at entry, reduces directional exposure into
 *      event windows. Hedge legs inherit the same 10% envelope (combined
 *      exposure is checked, not each leg in isolation).
 *   4. ARMED mode: micro-trade execution requires
 *      TELEGRAM_TRADING_ARMED=true AND TELEGRAM_TRADING_KEY matching the
 *      operator key in the /arm command. Without arming, the bot only ever
 *      ADVISES (plans, sizes, SL levels) — it never emits execution intents.
 *   5. All intents route through the platform's existing gate flow (the
 *      realtime engine's 14-gate registry remains the execution authority;
 *      the assistant is an interface, not an execution plane).
 */

export interface AccountContext {
  equity: number;            // account currency units
  symbol: string;            // e.g. XAUUSD
  valuePerPointPerLot: number; // account currency per point per 1.0 lot (from broker spec)
  minLot: number;            // broker minimum lot (0.01 typical)
  lotStep: number;           // lot step (0.01 typical)
  maxLot: number;            // broker cap
}

export interface MicroTradePlan {
  direction: 'BUY' | 'SELL';
  entry: number;
  stopLoss: number;
  takeProfits: number[];      // TP1..TP3
  lots: number;
  capitalAtRisk: number;      // currency at risk (≤ 10% equity)
  capitalAtRiskPct: number;
  rrTp1: number;
  hedge?: {
    direction: 'BUY' | 'SELL';
    lots: number;
    rationale: string;
    combinedRiskPct: number;
  };
  warnings: string[];
  refused?: string;           // set when the plan is refused outright
}

export const MAX_CAPITAL_RISK_PCT = 10;

function roundToStep(lots: number, step: number): number {
  return Math.floor(lots / step) * step;
}

/**
 * Build a micro-trade plan with the 10% capital stop-loss framework.
 * `structureStop` = stop level derived from chart structure (assistant supplies
 * from indicators; never a raw A*R guess).
 */
export function planMicroTrade(
  account: AccountContext,
  direction: 'BUY' | 'SELL',
  entry: number,
  structureStop: number,
  opts: { hedgePct?: number; rrTargets?: number[] } = {},
): MicroTradePlan {
  const warnings: string[] = [];
  const rrTargets = opts.rrTargets ?? [1, 2, 3];

  const stopPoints = Math.abs(entry - structureStop);
  if (stopPoints < 1) {
    return {
      direction, entry, stopLoss: structureStop, takeProfits: [],
      lots: 0, capitalAtRisk: 0, capitalAtRiskPct: 0, rrTp1: 0,
      warnings: ['Stop distance < 1 point — refuse: no valid structure stop.'],
      refused: 'Invalid stop geometry',
    };
  }

  // 10% framework: the ENVELOPE is 10% of equity. When a hedge is requested,
  // the envelope is SPLIT: primary takes (100 − hedgePct)% of the pool, the
  // hedge takes the remainder — combined exposure never exceeds 10%.
  const hedgePct = opts.hedgePct ?? 0;
  const primaryPct = hedgePct > 0 ? MAX_CAPITAL_RISK_PCT * (1 - hedgePct / 100) : MAX_CAPITAL_RISK_PCT;
  const maxRisk = account.equity * (primaryPct / 100);
  const rawLots = maxRisk / (stopPoints * account.valuePerPointPerLot);
  let lots = rawLots;
  if (account.lotStep > 0) lots = Math.floor(lots / account.lotStep) * account.lotStep;
  lots = Math.round(lots * 100) / 100;

  if (lots < account.minLot) {
    return {
      direction, entry, stopLoss: structureStop, takeProfits: [],
      lots: 0, capitalAtRisk: 0, capitalAtRiskPct: 0, rrTp1: 0,
      warnings: [
        `Computed size ${rawLots.toFixed(3)} lots is below broker min ${account.minLot}.`,
        '10% framework refuses the trade rather than over-risking — reduce stop distance or increase equity.',
      ],
      refused: 'Below minimum lot under 10% framework',
    };
  }
  if (lots > account.maxLot) {
    warnings.push(`Capped to broker max lot ${account.maxLot}.`);
    lots = account.maxLot;
  }

  const capitalAtRisk = lots * stopPoints * account.valuePerPointPerLot;
  const capitalAtRiskPct = (capitalAtRisk / account.equity) * 100;
  if (capitalAtRiskPct > primaryPct + 0.01) {
    // rounding-up guard: recompute down to the framework bound
    lots = Math.floor((maxRisk / (stopPoints * account.valuePerPointPerLot)) / account.lotStep) * account.lotStep;
    const reRisk = lots * stopPoints * account.valuePerPointPerLot;
    warnings.push('Resized to stay within the capital framework after rounding.');
    if (reRisk / account.equity * 100 > primaryPct + 0.01 || lots < account.minLot) {
      return {
        direction, entry, stopLoss: structureStop, takeProfits: [],
        lots: 0, capitalAtRisk: 0, capitalAtRiskPct: 0, rrTp1: 0,
        warnings: [...warnings, 'Could not fit the capital framework even at min lot — refused.'],
        refused: 'Cannot fit 10% capital framework',
      };
    }
  }

  const dirSign = direction === 'BUY' ? 1 : -1;
  const takeProfits = rrTargets.map((rr) => entry + dirSign * stopPoints * rr);

  // Automated hedging plan: the counter-leg takes the REMAINDER of the 10%
  // envelope (the envelope was already split at sizing time).
  let hedge: MicroTradePlan['hedge'] | undefined;
  if (hedgePct > 0) {
    const hedgeRiskTarget = account.equity * (MAX_CAPITAL_RISK_PCT / 100) * (hedgePct / 100);
    const hedgeStopPoints = stopPoints; // hedge stops mirrored on its own structure
    let hedgeLots = hedgeRiskTarget / (hedgeStopPoints * account.valuePerPointPerLot);
    hedgeLots = Math.floor(hedgeLots / account.lotStep) * account.lotStep;
    hedgeLots = Math.round(hedgeLots * 100) / 100;
    const combinedRisk = capitalAtRisk + hedgeLots * hedgeStopPoints * account.valuePerPointPerLot;
    const combinedRiskPct = (combinedRisk / account.equity) * 100;
    if (hedgeLots < account.minLot) {
      warnings.push(`Hedge computes below min lot at this stop distance — hedge skipped (framework refuses oversize).`);
    } else {
      hedge = {
        direction: direction === 'BUY' ? 'SELL' : 'BUY',
        lots: hedgeLots,
        rationale: `Counter-leg hedges into event/unknown windows; envelope split ${100 - hedgePct}/${hedgePct} inside the 10% cap.`,
        combinedRiskPct,
      };
    }
  }

  const rrTp1 = stopPoints > 0
    ? Math.abs(takeProfits[0] - entry) / stopPoints
    : 0;

  return {
    direction, entry, stopLoss: structureStop, takeProfits,
    lots, capitalAtRisk, capitalAtRiskPct, rrTp1, hedge, warnings,
  };
}

/** Validate an equity snapshot for sanity before planning (fail-closed). */
export function validateAccount(account: AccountContext): string | null {
  if (!Number.isFinite(account.equity) || account.equity <= 0) return 'Account equity missing or non-positive — refusing to plan.';
  if (!Number.isFinite(account.valuePerPointPerLot) || account.valuePerPointPerLot <= 0) return 'Broker point value missing — refusing to plan.';
  if (account.minLot <= 0 || account.lotStep <= 0) return 'Broker lot spec invalid — refusing to plan.';
  return null;
}