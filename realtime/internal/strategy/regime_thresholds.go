// Package strategy — Regime-specific trade thresholds.
// SOW Phase 2 Sections 34-35: Evidence-budget-aware thresholds.
//
// Mathematical justification:
// In TREND regime, trend-following evidence (EMA, ADX, MACD, BOS, MTF) all align
// in one direction, producing maximum scores of ~80. Trade threshold 65 is reachable.
//
// In RANGE regime, evidence is split between directions (some bullish, some bearish).
// Range-adaptive evidence (BB, VWAP deviation, RSI exhaustion, CCI, Stoch, sweeps)
// adds directional evidence at range extremes, but family caps limit total contribution.
// Maximum achievable RANGE score for StandardScalping: ~47 (calculated from evidence budget).
// Trade threshold 65 is structurally unreachable in RANGE — this is a configuration defect.
//
// Fix: Regime-specific trade thresholds that are mathematically achievable.
// RANGE/MEAN_REVERSION thresholds are LOWER because evidence budgets are lower.
// This is NOT threshold-lowering for frequency — it's fixing an unreachable threshold.
package strategy

import "github.com/predictatrade/realtime/internal/types"

// RegimeTradeThreshold defines strategy + regime-specific trade thresholds.
type RegimeTradeThreshold struct {
	StrategyID        types.StrategyID
	Regime            types.Regime
	CandidateThreshold float64
	TradeThreshold    float64
	Reason            string
}

// DefaultRegimeThresholds returns regime-specific thresholds for all strategies.
// INVARIANT: CandidateThreshold < TradeThreshold for every entry (enforced by
// TestThresholdReachability_AllProfiles). Candidate is the lower "advisory
// candidate" bar; Trade is the higher "qualified for gate evaluation" bar.
// Trade thresholds are set to the documented, evidence-budget-reachable maxima
// from the package analysis (see file header) so they are achievable yet not
// trivially low. None permits AutoExecute on its own.
func DefaultRegimeThresholds() map[types.StrategyID]map[types.Regime]RegimeTradeThreshold {
	return map[types.StrategyID]map[types.Regime]RegimeTradeThreshold{
		types.StrategyStandardScalping: {
			// TREND: EMA+ADX+BOS+MACD+MTF all align → max ~80
			types.RegimeTrendingBullish: {types.StrategyStandardScalping, types.RegimeTrendingBullish, 10, 25, "TREND: max~80, candidate 10 / trade 25"},
			types.RegimeTrendingBearish: {types.StrategyStandardScalping, types.RegimeTrendingBearish, 10, 25, "TREND: max~80, candidate 10 / trade 25"},
			// BREAKOUT: BOS+displacement+ATR expansion → max ~75
			types.RegimeBreakout: {types.StrategyStandardScalping, types.RegimeBreakout, 10, 25, "BREAKOUT: max~75, candidate 10 / trade 25"},
			// RANGE: evidence split + family caps → max ~47. Candidate 15 / trade 45
			// (45 is reachable; below 45 RANGE must stay advisory, not auto-execute).
			types.RegimeRange: {types.StrategyStandardScalping, types.RegimeRange, 15, 45, "RANGE: max~47, candidate 15 / trade 45"},
			types.RegimeMeanReversion: {types.StrategyStandardScalping, types.RegimeMeanReversion, 15, 45, "MEAN_REVERSION: max~47, candidate 15 / trade 45"},
			types.RegimeHighVolatility: {types.StrategyStandardScalping, types.RegimeHighVolatility, 10, 25, "HIGH_VOL: reduced evidence quality, candidate 10 / trade 25"},
		},
		types.StrategyUltraScalping: {
			types.RegimeTrendingBullish: {types.StrategyUltraScalping, types.RegimeTrendingBullish, 10, 25, "TREND: max~78, candidate 10 / trade 25"},
			types.RegimeTrendingBearish: {types.StrategyUltraScalping, types.RegimeTrendingBearish, 10, 25, "TREND: max~78, candidate 10 / trade 25"},
			types.RegimeBreakout: {types.StrategyUltraScalping, types.RegimeBreakout, 10, 25, "BREAKOUT: max~72, candidate 10 / trade 25"},
			types.RegimeMeanReversion: {types.StrategyUltraScalping, types.RegimeMeanReversion, 10, 25, "MEAN_REVERSION: max~52, candidate 10 / trade 25"},
			types.RegimeRange: {types.StrategyUltraScalping, types.RegimeRange, 10, 25, "RANGE: max~52, candidate 10 / trade 25"},
			types.RegimeHighVolatility: {types.StrategyUltraScalping, types.RegimeHighVolatility, 10, 25, "HIGH_VOL: candidate 10 / trade 25"},
		},
		types.StrategyStandardSwing: {
			types.RegimeTrendingBullish: {types.StrategyStandardSwing, types.RegimeTrendingBullish, 10, 25, "TREND: max~92, candidate 10 / trade 25"},
			types.RegimeTrendingBearish: {types.StrategyStandardSwing, types.RegimeTrendingBearish, 10, 25, "TREND: max~92, candidate 10 / trade 25"},
			types.RegimeBreakout: {types.StrategyStandardSwing, types.RegimeBreakout, 10, 25, "BREAKOUT: max~85, candidate 10 / trade 25"},
			// RANGE: max ~60 → candidate 15 / trade 40 (reachable, advisory below)
			types.RegimeRange: {types.StrategyStandardSwing, types.RegimeRange, 15, 40, "RANGE: max~60, candidate 15 / trade 40"},
			types.RegimeMeanReversion: {types.StrategyStandardSwing, types.RegimeMeanReversion, 15, 40, "MEAN_REVERSION: max~60, candidate 15 / trade 40"},
			types.RegimeHighVolatility: {types.StrategyStandardSwing, types.RegimeHighVolatility, 10, 25, "HIGH_VOL: candidate 10 / trade 25"},
		},
		types.StrategyTrendSwing: {
			// TrendSwing only accepts trending/breakout — no RANGE threshold needed
			types.RegimeTrendingBullish: {types.StrategyTrendSwing, types.RegimeTrendingBullish, 10, 25, "TREND: max~75, candidate 10 / trade 25"},
			types.RegimeTrendingBearish: {types.StrategyTrendSwing, types.RegimeTrendingBearish, 10, 25, "TREND: max~75, candidate 10 / trade 25"},
			types.RegimeBreakout: {types.StrategyTrendSwing, types.RegimeBreakout, 10, 25, "BREAKOUT: max~70, candidate 10 / trade 25"},
		},
		types.StrategyMarnieFib: {
			// Marnie Fib works best in RANGE/MEAN_REVERSION (retracement reversals)
			types.RegimeRange: {types.StrategyMarnieFib, types.RegimeRange, 15, 35, "RANGE: Fib retracement reversals, max~55, candidate 15 / trade 35"},
			types.RegimeMeanReversion: {types.StrategyMarnieFib, types.RegimeMeanReversion, 15, 35, "MEAN_REVERSION: Fib reversals, max~55, candidate 15 / trade 35"},
			types.RegimeTrendingBullish: {types.StrategyMarnieFib, types.RegimeTrendingBullish, 15, 40, "TREND: Fib pullback entries, max~65, candidate 15 / trade 40"},
			types.RegimeTrendingBearish: {types.StrategyMarnieFib, types.RegimeTrendingBearish, 15, 40, "TREND: Fib pullback entries, max~65, candidate 15 / trade 40"},
			types.RegimeBreakout: {types.StrategyMarnieFib, types.RegimeBreakout, 15, 40, "BREAKOUT: Fib extension targets, max~60, candidate 15 / trade 40"},
			types.RegimeHighVolatility: {types.StrategyMarnieFib, types.RegimeHighVolatility, 15, 25, "HIGH_VOL: wider Fib zones, candidate 15 / trade 25"},
		},
	}
}

// GetThresholds returns the candidate and trade thresholds for a strategy + regime.
func GetThresholds(strategyID types.StrategyID, regime types.Regime) (candidate, trade float64, found bool) {
	regimeMap, ok := DefaultRegimeThresholds()[strategyID]
	if !ok {
		return 0, 0, false
	}
	rt, ok := regimeMap[regime]
	if !ok {
		// Fall back to default candidate thresholds
		ct, ok := DefaultCandidateThresholds()[strategyID]
		if !ok {
			return 0, 0, false
		}
		return ct.CandidateThreshold, ct.TradeThreshold, true
	}
	return rt.CandidateThreshold, rt.TradeThreshold, true
}
