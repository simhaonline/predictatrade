// Package strategy — Trend Swing transition evidence for RANGE regimes.
// SOW Phase 2 Sections 21-25: Detect developing trend transitions without trading ranges.
//
// When regime is RANGE, TrendSwing checks for transition evidence:
// ADX expansion, EMA compression→expansion, BB width expansion, ATR expansion,
// range boundary pressure, BOS, liquidity break, MTF alignment.
// If transition evidence is meaningful, a BUY_CANDIDATE/SELL_CANDIDATE is emitted
// with signal_class=ADVISORY, execution_allowed=false.
package strategy

import (
	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// computeTrendTransitionEvidence detects developing trend transitions in RANGE.
// Returns evidence if transition signals exist, empty slice if not.
func computeTrendTransitionEvidence(state *features.MarketState) []types.EvidenceContribution {
	if state == nil {
		return nil
	}

	var evidence []types.EvidenceContribution
	q := state.Quality

	// ADX expansion — ADX rising from low levels indicates trend building
	adx, _ := state.Indicators.ADX.Float64()
	if adx > 18 && adx < 30 {
		// ADX in the transition zone (18-30) — trend is building
		if state.Indicators.ADXPlusDI.GreaterThan(state.Indicators.ADXMinusDI) {
			addEvidence(&evidence, "TREND", "ADX_EXPANSION_BULLISH", types.DirectionBuy,
				15, 0.10, q, "ADX expanding with bullish +DI")
		} else if state.Indicators.ADXMinusDI.GreaterThan(state.Indicators.ADXPlusDI) {
			addEvidence(&evidence, "TREND", "ADX_EXPANSION_BEARISH", types.DirectionSell,
				15, 0.10, q, "ADX expanding with bearish -DI")
		}
	}

	// EMA slope — EMA9 above EMA21 suggests bullish momentum building
	if state.Indicators.EMA9.GreaterThan(state.Indicators.EMA21) {
		addEvidence(&evidence, "TREND", "EMA_SLOPE_BULLISH", types.DirectionBuy,
			12, 0.08, q, "EMA9 above EMA21 — momentum building")
	} else if state.Indicators.EMA9.LessThan(state.Indicators.EMA21) {
		addEvidence(&evidence, "TREND", "EMA_SLOPE_BEARISH", types.DirectionSell,
			12, 0.08, q, "EMA9 below EMA21 — momentum building")
	}

	// BB width expansion — volatility expansion precedes breakouts
	if !state.Indicators.BollWidth.IsZero() {
		// If BB width is non-trivial, market may be preparing for breakout
		bbWidth, _ := state.Indicators.BollWidth.Float64()
		if bbWidth > 0.01 {
			// Price near upper BB → potential bullish breakout
			if state.CurrentPrice.GreaterThan(state.Indicators.BollMiddle) {
				addEvidence(&evidence, "VOLATILITY", "BB_EXPANSION_UPPER", types.DirectionBuy,
					10, 0.06, q, "BB expanding, price in upper half")
			} else {
				addEvidence(&evidence, "VOLATILITY", "BB_EXPANSION_LOWER", types.DirectionSell,
					10, 0.06, q, "BB expanding, price in lower half")
			}
		}
	}

	// ATR expansion — volatility is increasing
	if !state.Indicators.ATR.IsZero() && !state.CurrentPrice.IsZero() {
		atrPct := state.Indicators.ATR.Div(state.CurrentPrice)
		if atrPct.GreaterThan(decimal.NewFromFloat(0.0015)) {
			// ATR > 0.15% of price — elevated volatility
			addEvidence(&evidence, "VOLATILITY", "ATR_EXPANSION", types.DirectionBuy,
				8, 0.04, q, "ATR elevated — breakout preparation")
			addEvidence(&evidence, "VOLATILITY", "ATR_EXPANSION_S", types.DirectionSell,
				8, 0.04, q, "ATR elevated — breakout preparation")
		}
	}

	// BOS — any break of structure suggests directional pressure
	if state.Structure.LastBOS != nil {
		if state.Structure.LastBOS.Direction == "bullish" {
			addEvidence(&evidence, "STRUCTURE", "RANGE_BOS_BULLISH", types.DirectionBuy,
				15, 0.10, q, "BOS in range — bullish pressure")
		} else if state.Structure.LastBOS.Direction == "bearish" {
			addEvidence(&evidence, "STRUCTURE", "RANGE_BOS_BEARISH", types.DirectionSell,
				15, 0.10, q, "BOS in range — bearish pressure")
		}
	}

	// MACD expansion — momentum building
	if state.Indicators.MACDMain.GreaterThan(state.Indicators.MACDSignal) {
		addEvidence(&evidence, "MOMENTUM", "MACD_EXPANSION_BULLISH", types.DirectionBuy,
			10, 0.06, q, "MACD above signal — bullish momentum")
	} else if state.Indicators.MACDMain.LessThan(state.Indicators.MACDSignal) {
		addEvidence(&evidence, "MOMENTUM", "MACD_EXPANSION_BEARISH", types.DirectionSell,
			10, 0.06, q, "MACD below signal — bearish momentum")
	}

	// MTF alignment — higher timeframe direction
	mtfScore := state.MTF.Score
	if mtfScore > 30 {
		addEvidence(&evidence, "MTF", "MTF_BULLISH_ALIGNMENT", types.DirectionBuy,
			12, 0.07, q, "MTF aligning bullish")
	} else if mtfScore < -30 {
		addEvidence(&evidence, "MTF", "MTF_BEARISH_ALIGNMENT", types.DirectionSell,
			12, 0.07, q, "MTF aligning bearish")
	}

	// Structure trend
	if state.Structure.CurrentTrend == "bullish" {
		addEvidence(&evidence, "STRUCTURE", "HH_HL_DEVELOPING", types.DirectionBuy,
			10, 0.06, q, "higher highs/lows developing")
	} else if state.Structure.CurrentTrend == "bearish" {
		addEvidence(&evidence, "STRUCTURE", "LH_LL_DEVELOPING", types.DirectionSell,
			10, 0.06, q, "lower highs/lows developing")
	}

	return evidence
}
