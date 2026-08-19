// Package strategy — Ultra Scalping RANGE/microstructure evidence.
// SOW Phase 2 Section 17-20: Dedicated Ultra range profile.
//
// Ultra Scalping is a DISTINCT strategy from StandardScalping.
// Its range evidence emphasizes FASTER microstructure information:
// micro liquidity sweeps, VWAP deviation, BB excursion, Stoch RSI,
// short-term RSI, rejection candles, CCI, OsMA momentum.
package strategy

import (
	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// UltraRangeConfig defines Ultra Scalping range evidence weights.
type UltraRangeConfig struct {
	LiquiditySweepWeight float64
	VWAPDeviationWeight  float64
	BBExcursionWeight    float64
	StochRSIWeight       float64
	RSIWeight            float64
	RejectionWeight      float64
	CCIWeight            float64
	OsMAMomentumWeight   float64
	SpreadQualityWeight  float64
}

// DefaultUltraRangeConfig returns weights appropriate for ultra-short range trading.
// Total budget: ~0.65 → max score ~65 (reachable with trade threshold 50)
func DefaultUltraRangeConfig() UltraRangeConfig {
	return UltraRangeConfig{
		LiquiditySweepWeight: 0.12,
		VWAPDeviationWeight:  0.08,
		BBExcursionWeight:    0.08,
		StochRSIWeight:       0.07,
		RSIWeight:            0.06,
		RejectionWeight:      0.10,
		CCIWeight:            0.05,
		OsMAMomentumWeight:   0.05,
		SpreadQualityWeight:  0.04,
	}
}

// computeUltraRangeEvidence adds RANGE-specific microstructure evidence for Ultra Scalping.
// This is SEPARATE from the StandardScalping range evidence — different weights, different emphasis.
func computeUltraRangeEvidence(evidence *[]types.EvidenceContribution, state *features.MarketState, cfg UltraRangeConfig) {
	if state == nil {
		return
	}
	q := state.Quality

	// Micro liquidity sweep — highest weight for ultra
	if len(state.Liquidity.RecentSweeps) > 0 {
		sweep := state.Liquidity.RecentSweeps[len(state.Liquidity.RecentSweeps)-1]
		if sweepIsSellSide(sweep.Direction) {
			addEvidence(evidence, "LIQUIDITY", "ULTRA_SELL_SIDE_SWEEP", types.DirectionBuy,
				18, cfg.LiquiditySweepWeight, q, "micro sell-side sweep")
		} else if sweepIsBuySide(sweep.Direction) {
			addEvidence(evidence, "LIQUIDITY", "ULTRA_BUY_SIDE_SWEEP", types.DirectionSell,
				18, cfg.LiquiditySweepWeight, q, "micro buy-side sweep")
		}
	}

	// VWAP deviation — ultra cares about short-term VWAP distance
	if !state.VWAP.SessionVWAP.IsZero() && !state.Indicators.ATR.IsZero() {
		dev := state.CurrentPrice.Sub(state.VWAP.SessionVWAP)
		normalizedDev := dev.Div(state.Indicators.ATR)
		if normalizedDev.LessThan(decimal.NewFromFloat(-1.0)) {
			addEvidence(evidence, "VWAP", "ULTRA_VWAP_OVERSOLD", types.DirectionBuy,
				12, cfg.VWAPDeviationWeight, q, "price >1 ATR below VWAP")
		} else if normalizedDev.GreaterThan(decimal.NewFromFloat(1.0)) {
			addEvidence(evidence, "VWAP", "ULTRA_VWAP_OVERBOUGHT", types.DirectionSell,
				12, cfg.VWAPDeviationWeight, q, "price >1 ATR above VWAP")
		}
	}

	// BB excursion — ultra reacts to band touches
	if !state.Indicators.BollLower.IsZero() && !state.Indicators.BollUpper.IsZero() {
		if state.CurrentPrice.LessThanOrEqual(state.Indicators.BollLower) {
			addEvidence(evidence, "VOLATILITY", "ULTRA_BB_LOWER", types.DirectionBuy,
				12, cfg.BBExcursionWeight, q, "price at lower BB")
		} else if state.CurrentPrice.GreaterThanOrEqual(state.Indicators.BollUpper) {
			addEvidence(evidence, "VOLATILITY", "ULTRA_BB_UPPER", types.DirectionSell,
				12, cfg.BBExcursionWeight, q, "price at upper BB")
		}
	}

	// Stoch RSI — fast oscillator
	if !state.Indicators.StochRSI.IsZero() {
		stochRSI, _ := state.Indicators.StochRSI.Float64()
		if stochRSI < 0.2 {
			addEvidence(evidence, "MOMENTUM", "ULTRA_STOCHRSI_OVERSOLD", types.DirectionBuy,
				10, cfg.StochRSIWeight, q, "StochRSI oversold")
		} else if stochRSI > 0.8 {
			addEvidence(evidence, "MOMENTUM", "ULTRA_STOCHRSI_OVERBOUGHT", types.DirectionSell,
				10, cfg.StochRSIWeight, q, "StochRSI overbought")
		}
	}

	// RSI — short-term exhaustion
	rsi, _ := state.Indicators.RSI.Float64()
	if rsi > 0 && rsi < 30 {
		addEvidence(evidence, "MOMENTUM", "ULTRA_RSI_OVERSOLD", types.DirectionBuy,
			10, cfg.RSIWeight, q, "RSI oversold")
	} else if rsi > 70 {
		addEvidence(evidence, "MOMENTUM", "ULTRA_RSI_OVERBOUGHT", types.DirectionSell,
			10, cfg.RSIWeight, q, "RSI overbought")
	}

	// Rejection candle — key for ultra
	if state.Candle.IsRejection {
		if state.Candle.IsBullish {
			addEvidence(evidence, "CANDLE", "ULTRA_BULLISH_REJECTION", types.DirectionBuy,
				15, cfg.RejectionWeight, q, "bullish rejection candle")
		} else if state.Candle.IsBearish {
			addEvidence(evidence, "CANDLE", "ULTRA_BEARISH_REJECTION", types.DirectionSell,
				15, cfg.RejectionWeight, q, "bearish rejection candle")
		}
	}

	// CCI — momentum extreme
	if !state.Indicators.CCI.IsZero() {
		if state.Indicators.CCI.LessThan(decimal.NewFromInt(-100)) {
			addEvidence(evidence, "MOMENTUM", "ULTRA_CCI_OVERSOLD", types.DirectionBuy,
				8, cfg.CCIWeight, q, "CCI oversold")
		} else if state.Indicators.CCI.GreaterThan(decimal.NewFromInt(100)) {
			addEvidence(evidence, "MOMENTUM", "ULTRA_CCI_OVERBOUGHT", types.DirectionSell,
				8, cfg.CCIWeight, q, "CCI overbought")
		}
	}

	// OsMA momentum
	if state.Indicators.OsMA.GreaterThan(decimal.Zero) {
		addEvidence(evidence, "MOMENTUM", "ULTRA_OSMA_BULLISH", types.DirectionBuy,
			8, cfg.OsMAMomentumWeight, q, "OsMA positive")
	} else if state.Indicators.OsMA.LessThan(decimal.Zero) {
		addEvidence(evidence, "MOMENTUM", "ULTRA_OSMA_BEARISH", types.DirectionSell,
			8, cfg.OsMAMomentumWeight, q, "OsMA negative")
	}

	// Spread quality — ultra is spread-sensitive
	if !state.Spread.IsZero() && !state.Indicators.ATR.IsZero() {
		spreadToATR := state.Spread.Div(state.Indicators.ATR)
		if spreadToATR.LessThan(decimal.NewFromFloat(0.3)) {
			addEvidence(evidence, "VOLATILITY", "ULTRA_GOOD_SPREAD", types.DirectionBuy,
				6, cfg.SpreadQualityWeight, q, "good spread for ultra")
			addEvidence(evidence, "VOLATILITY", "ULTRA_GOOD_SPREAD_S", types.DirectionSell,
				6, cfg.SpreadQualityWeight, q, "good spread for ultra")
		}
	}
}
