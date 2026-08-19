// Package strategy — Range/regime-adaptive evidence computation.
// SOW Phase 2 Section 5, 16: Regime-specific evidence profiles.
//
// This module adds RANGE-compatible evidence to existing strategies
// WITHOUT weakening safety or changing strategy identity.
// Each strategy can now produce evidence appropriate to the detected regime.
package strategy

import (
	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// RangeEvidenceConfig defines RANGE-specific evidence weights.
// These are derived from SOW Section 12 and existing indicator capabilities.
type RangeEvidenceConfig struct {
	BBLowerTouchWeight    float64
	BBUpperTouchWeight    float64
	VWAPDeviationWeight   float64
	RSIExhaustionWeight   float64
	StochExhaustionWeight float64
	CCIExtremeWeight      float64
	RejectionCandleWeight float64
	LiquiditySweepWeight  float64
	FVGReactionWeight     float64
}

// DefaultRangeEvidenceConfig returns RANGE evidence weights appropriate for scalping.
func DefaultRangeEvidenceConfig() RangeEvidenceConfig {
	return RangeEvidenceConfig{
		BBLowerTouchWeight:    0.08,
		BBUpperTouchWeight:    0.08,
		VWAPDeviationWeight:   0.06,
		RSIExhaustionWeight:   0.07,
		StochExhaustionWeight: 0.06,
		CCIExtremeWeight:      0.05,
		RejectionCandleWeight: 0.08,
		LiquiditySweepWeight:  0.10,
		FVGReactionWeight:     0.06,
	}
}

// computeRangeEvidence adds RANGE-specific evidence to the evidence slice.
// This is called when the regime is RANGE or MEAN_REVERSION and the strategy
// supports range trading (via AcceptedRegimes).
//
// IMPORTANT: This does NOT remove or weaken existing evidence.
// It ADDS range-appropriate evidence on top of the base evidence.
func computeRangeEvidence(evidence *[]types.EvidenceContribution, state *features.MarketState, cfg RangeEvidenceConfig) {
	if state == nil {
		return
	}
	q := state.Quality

	// Bollinger Band excursion — price touching lower band = buy signal in range
	if !state.Indicators.BollLower.IsZero() && !state.Indicators.BollUpper.IsZero() {
		if state.CurrentPrice.LessThanOrEqual(state.Indicators.BollLower) {
			addEvidence(evidence, "VOLATILITY", "BB_LOWER_TOUCH_RANGE", types.DirectionBuy,
				12, cfg.BBLowerTouchWeight, q, "price at lower BB in range")
		} else if state.CurrentPrice.GreaterThanOrEqual(state.Indicators.BollUpper) {
			addEvidence(evidence, "VOLATILITY", "BB_UPPER_TOUCH_RANGE", types.DirectionSell,
				12, cfg.BBUpperTouchWeight, q, "price at upper BB in range")
		}
	}

	// VWAP deviation — price far below VWAP = buy in range
	if !state.VWAP.SessionVWAP.IsZero() {
		vwapDev := state.CurrentPrice.Sub(state.VWAP.SessionVWAP)
		atr := state.Indicators.ATR
		if !atr.IsZero() {
			normalizedDev := vwapDev.Div(atr)
			if normalizedDev.LessThan(decimal.NewFromFloat(-1.5)) {
				addEvidence(evidence, "VWAP", "VWAP_OVERSOLD_RANGE", types.DirectionBuy,
					10, cfg.VWAPDeviationWeight, q, "price >1.5 ATR below VWAP in range")
			} else if normalizedDev.GreaterThan(decimal.NewFromFloat(1.5)) {
				addEvidence(evidence, "VWAP", "VWAP_OVERBOUGHT_RANGE", types.DirectionSell,
					10, cfg.VWAPDeviationWeight, q, "price >1.5 ATR above VWAP in range")
			}
		}
	}

	// RSI exhaustion in range context
	rsi, _ := state.Indicators.RSI.Float64()
	if rsi > 0 { // Feature readiness check — don't use uninitialized RSI
		if rsi < 35 {
			addEvidence(evidence, "MOMENTUM", "RSI_OVERSOLD_RANGE", types.DirectionBuy,
				10, cfg.RSIExhaustionWeight, q, "RSI exhausted in range")
		} else if rsi > 65 {
			addEvidence(evidence, "MOMENTUM", "RSI_OVERBOUGHT_RANGE", types.DirectionSell,
				10, cfg.RSIExhaustionWeight, q, "RSI exhausted in range")
		}
	}

	// Stochastic exhaustion
	stochMain, _ := state.Indicators.StochMain.Float64()
	stochSignal, _ := state.Indicators.StochSignal.Float64()
	if stochMain > 0 && stochSignal > 0 {
		if stochMain < 20 && stochMain > stochSignal {
			addEvidence(evidence, "MOMENTUM", "STOCH_OVERSOLD_CROSS_RANGE", types.DirectionBuy,
				8, cfg.StochExhaustionWeight, q, "stochastic oversold cross in range")
		} else if stochMain > 80 && stochMain < stochSignal {
			addEvidence(evidence, "MOMENTUM", "STOCH_OVERBOUGHT_CROSS_RANGE", types.DirectionSell,
				8, cfg.StochExhaustionWeight, q, "stochastic overbought cross in range")
		}
	}

	// CCI extreme
	if !state.Indicators.CCI.IsZero() {
		if state.Indicators.CCI.LessThan(decimal.NewFromInt(-100)) {
			addEvidence(evidence, "MOMENTUM", "CCI_OVERSOLD_RANGE", types.DirectionBuy,
				8, cfg.CCIExtremeWeight, q, "CCI extreme oversold in range")
		} else if state.Indicators.CCI.GreaterThan(decimal.NewFromInt(100)) {
			addEvidence(evidence, "MOMENTUM", "CCI_OVERBOUGHT_RANGE", types.DirectionSell,
				8, cfg.CCIExtremeWeight, q, "CCI extreme overbought in range")
		}
	}

	// Rejection candle at range extremes
	if state.Candle.IsRejection {
		if state.Candle.IsBullish {
			addEvidence(evidence, "CANDLE", "BULLISH_REJECTION_RANGE", types.DirectionBuy,
				12, cfg.RejectionCandleWeight, q, "bullish rejection in range")
		} else if state.Candle.IsBearish {
			addEvidence(evidence, "CANDLE", "BEARISH_REJECTION_RANGE", types.DirectionSell,
				12, cfg.RejectionCandleWeight, q, "bearish rejection in range")
		}
	}

	// Liquidity sweep in range context (BSL/SSL sweep)
	if len(state.Liquidity.RecentSweeps) > 0 {
		sweep := state.Liquidity.RecentSweeps[len(state.Liquidity.RecentSweeps)-1]
		if sweepIsSellSide(sweep.Direction) {
			addEvidence(evidence, "LIQUIDITY", "SELL_SIDE_SWEEP_RANGE", types.DirectionBuy,
				15, cfg.LiquiditySweepWeight, q, "sell-side liquidity sweep in range")
		} else if sweepIsBuySide(sweep.Direction) {
			addEvidence(evidence, "LIQUIDITY", "BUY_SIDE_SWEEP_RANGE", types.DirectionSell,
				15, cfg.LiquiditySweepWeight, q, "buy-side liquidity sweep in range")
		}
	}

	// FVG reaction in range
	if len(state.FVG.FVGs) > 0 {
		fvg := state.FVG.FVGs[len(state.FVG.FVGs)-1]
		if fvg.Type == "BULLISH" && !fvg.Filled {
			addEvidence(evidence, "SMC", "BULLISH_FVG_RANGE", types.DirectionBuy,
				8, cfg.FVGReactionWeight, q, "bullish FVG reaction in range")
		} else if fvg.Type == "BEARISH" && !fvg.Filled {
			addEvidence(evidence, "SMC", "BEARISH_FVG_RANGE", types.DirectionSell,
				8, cfg.FVGReactionWeight, q, "bearish FVG reaction in range")
		}
	}
}

// isRegimeInRange checks if the current regime is a range-type regime.
func isRegimeInRange(regime types.Regime) bool {
	return regime == types.RegimeRange || regime == types.RegimeMeanReversion
}

// isFeatureReady checks if an indicator value is valid (not zero/uninitialized).
// This is used to prevent zero-filled features from producing false evidence.
func isFeatureReady(value decimal.Decimal) bool {
	return !value.IsZero()
}
