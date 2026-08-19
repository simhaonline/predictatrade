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
			types.RegimeTrendingBullish: {types.StrategyStandardScalping, types.RegimeTrendingBullish, 40, 65, "TREND: max~80, threshold 65 reachable"},
			types.RegimeTrendingBearish: {types.StrategyStandardScalping, types.RegimeTrendingBearish, 40, 65, "TREND: max~80, threshold 65 reachable"},
			// BREAKOUT: BOS+displacement+ATR expansion → max ~75, threshold 65 reachable
			types.RegimeBreakout: {types.StrategyStandardScalping, types.RegimeBreakout, 40, 65, "BREAKOUT: max~75, threshold 65 reachable"},
			// RANGE: evidence split + family caps → max ~47, threshold 65 UNREACHABLE
			// Fixed: trade threshold 45 (mathematically achievable), candidate 30
			types.RegimeRange: {types.StrategyStandardScalping, types.RegimeRange, 30, 45, "RANGE: max~47 with family caps, old threshold 65 unreachable, fixed to 45"},
			types.RegimeMeanReversion: {types.StrategyStandardScalping, types.RegimeMeanReversion, 30, 45, "MEAN_REVERSION: max~47, threshold 45"},
			types.RegimeHighVolatility: {types.StrategyStandardScalping, types.RegimeHighVolatility, 30, 50, "HIGH_VOL: reduced evidence quality"},
		},
		types.StrategyUltraScalping: {
			types.RegimeTrendingBullish: {types.StrategyUltraScalping, types.RegimeTrendingBullish, 40, 65, "TREND: max~78"},
			types.RegimeTrendingBearish: {types.StrategyUltraScalping, types.RegimeTrendingBearish, 40, 65, "TREND: max~78"},
			types.RegimeBreakout: {types.StrategyUltraScalping, types.RegimeBreakout, 40, 65, "BREAKOUT: max~72"},
			types.RegimeMeanReversion: {types.StrategyUltraScalping, types.RegimeMeanReversion, 35, 50, "MEAN_REVERSION: max~52"},
			types.RegimeRange: {types.StrategyUltraScalping, types.RegimeRange, 35, 50, "RANGE: max~52"},
			types.RegimeHighVolatility: {types.StrategyUltraScalping, types.RegimeHighVolatility, 35, 55, "HIGH_VOL"},
		},
		types.StrategyStandardSwing: {
			types.RegimeTrendingBullish: {types.StrategyStandardSwing, types.RegimeTrendingBullish, 35, 55, "TREND: max~92"},
			types.RegimeTrendingBearish: {types.StrategyStandardSwing, types.RegimeTrendingBearish, 35, 55, "TREND: max~92"},
			types.RegimeBreakout: {types.StrategyStandardSwing, types.RegimeBreakout, 35, 55, "BREAKOUT: max~85"},
			// RANGE: already accepts RANGE, max~60 with range evidence, threshold 55 borderline
			types.RegimeRange: {types.StrategyStandardSwing, types.RegimeRange, 25, 40, "RANGE: max~60 with range evidence, old 55 borderline, fixed 40"},
			types.RegimeMeanReversion: {types.StrategyStandardSwing, types.RegimeMeanReversion, 25, 40, "MEAN_REVERSION: max~60"},
			types.RegimeHighVolatility: {types.StrategyStandardSwing, types.RegimeHighVolatility, 30, 50, "HIGH_VOL"},
		},
		types.StrategyTrendSwing: {
			// TrendSwing only accepts trending/breakout — no RANGE threshold needed
			types.RegimeTrendingBullish: {types.StrategyTrendSwing, types.RegimeTrendingBullish, 30, 50, "TREND: max~75"},
			types.RegimeTrendingBearish: {types.StrategyTrendSwing, types.RegimeTrendingBearish, 30, 50, "TREND: max~75"},
			types.RegimeBreakout: {types.StrategyTrendSwing, types.RegimeBreakout, 30, 50, "BREAKOUT: max~70"},
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
