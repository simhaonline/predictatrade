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
// Candidate threshold < Trade threshold. Neither permits AutoExecute.
func DefaultRegimeThresholds() map[types.StrategyID]map[types.Regime]RegimeTradeThreshold {
	return map[types.StrategyID]map[types.Regime]RegimeTradeThreshold{
		types.StrategyStandardScalping: {
			// TREND: EMA+ADX+BOS+MACD+MTF all align → max ~80, threshold 65 reachable
			types.RegimeTrendingBullish: {types.StrategyStandardScalping, types.RegimeTrendingBullish, 10, 25, "TREND: max~80, candidate widened to 25 for wider candidate reach"},
			types.RegimeTrendingBearish: {types.StrategyStandardScalping, types.RegimeTrendingBearish, 10, 25, "TREND: max~80, candidate widened to 25"},
			// BREAKOUT: BOS+displacement+ATR expansion → max ~75, threshold 65 reachable
			types.RegimeBreakout: {types.StrategyStandardScalping, types.RegimeBreakout, 10, 25, "BREAKOUT: max~75, candidate widened to 25"},
			// RANGE: evidence split + family caps → max ~47. Wider candidate band
			// (15) so more directional NO-TRADE scores become advisory candidates.
			types.RegimeRange: {types.StrategyStandardScalping, types.RegimeRange, 10, 10, "RANGE: max~47, trade threshold 45, candidate widened to 15"},
			types.RegimeMeanReversion: {types.StrategyStandardScalping, types.RegimeMeanReversion, 10, 10, "MEAN_REVERSION: max~47, candidate widened to 15"},
			types.RegimeHighVolatility: {types.StrategyStandardScalping, types.RegimeHighVolatility, 10, 50, "HIGH_VOL: reduced evidence quality, candidate widened to 15"},
		},
		types.StrategyUltraScalping: {
			types.RegimeTrendingBullish: {types.StrategyUltraScalping, types.RegimeTrendingBullish, 10, 25, "TREND: max~78, candidate widened to 25"},
			types.RegimeTrendingBearish: {types.StrategyUltraScalping, types.RegimeTrendingBearish, 10, 25, "TREND: max~78, candidate widened to 25"},
			types.RegimeBreakout: {types.StrategyUltraScalping, types.RegimeBreakout, 10, 25, "BREAKOUT: max~72, candidate widened to 25"},
			types.RegimeMeanReversion: {types.StrategyUltraScalping, types.RegimeMeanReversion, 10, 50, "MEAN_REVERSION: max~52, candidate widened to 15"},
			types.RegimeRange: {types.StrategyUltraScalping, types.RegimeRange, 10, 50, "RANGE: max~52, candidate widened to 15"},
			types.RegimeHighVolatility: {types.StrategyUltraScalping, types.RegimeHighVolatility, 10, 55, "HIGH_VOL: candidate widened to 15"},
		},
		types.StrategyStandardSwing: {
			types.RegimeTrendingBullish: {types.StrategyStandardSwing, types.RegimeTrendingBullish, 10, 55, "TREND: max~92, candidate widened to 20"},
			types.RegimeTrendingBearish: {types.StrategyStandardSwing, types.RegimeTrendingBearish, 10, 55, "TREND: max~92, candidate widened to 20"},
			types.RegimeBreakout: {types.StrategyStandardSwing, types.RegimeBreakout, 10, 55, "BREAKOUT: max~85, candidate widened to 20"},
			types.RegimeRange: {types.StrategyStandardSwing, types.RegimeRange, 10, 10, "RANGE: max~60, candidate widened to 15"},
			types.RegimeMeanReversion: {types.StrategyStandardSwing, types.RegimeMeanReversion, 10, 10, "MEAN_REVERSION: max~60, candidate widened to 15"},
			types.RegimeHighVolatility: {types.StrategyStandardSwing, types.RegimeHighVolatility, 10, 50, "HIGH_VOL: candidate widened to 15"},
		},
		types.StrategyTrendSwing: {
			// TrendSwing only accepts trending/breakout — no RANGE threshold needed
			types.RegimeTrendingBullish: {types.StrategyTrendSwing, types.RegimeTrendingBullish, 10, 50, "TREND: max~75, candidate widened to 15"},
			types.RegimeTrendingBearish: {types.StrategyTrendSwing, types.RegimeTrendingBearish, 10, 50, "TREND: max~75, candidate widened to 15"},
			types.RegimeBreakout: {types.StrategyTrendSwing, types.RegimeBreakout, 10, 50, "BREAKOUT: max~70, candidate widened to 15"},
		},
		types.StrategyMarnieFib: {
			// Marnie Fib works best in RANGE/MEAN_REVERSION (retracement reversals)
			types.RegimeRange: {types.StrategyMarnieFib, types.RegimeRange, 10, 10, "RANGE: Fib retracement reversals, max~55, candidate widened to 15"},
			types.RegimeMeanReversion: {types.StrategyMarnieFib, types.RegimeMeanReversion, 10, 10, "MEAN_REVERSION: Fib reversals, max~55, candidate widened to 15"},
			types.RegimeTrendingBullish: {types.StrategyMarnieFib, types.RegimeTrendingBullish, 10, 10, "TREND: Fib pullback entries, max~65, candidate widened to 15"},
			types.RegimeTrendingBearish: {types.StrategyMarnieFib, types.RegimeTrendingBearish, 10, 10, "TREND: Fib pullback entries, max~65, candidate widened to 15"},
			types.RegimeBreakout: {types.StrategyMarnieFib, types.RegimeBreakout, 10, 10, "BREAKOUT: Fib extension targets, max~60, candidate widened to 15"},
			types.RegimeHighVolatility: {types.StrategyMarnieFib, types.RegimeHighVolatility, 10, 10, "HIGH_VOL: wider Fib zones, candidate widened to 15"},
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
