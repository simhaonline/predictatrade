// Package strategy — Microprofit geometry for candidate signals.
//
// BUY_CANDIDATE and SELL_CANDIDATE signals have lower confluence scores than
// qualified BUY/SELL signals. They are not strong enough for the full strategy
// geometry (2×ATR stop, 2×ATR TP1 = R:R 1.0, TP2 = 4×ATR = R:R 2.0, etc.)
// but they still represent a directional edge that can be monetized.
//
// The microprofit approach uses tighter stops and closer targets:
//   SL  = 1.0 × ATR  (tighter than 1.5-2.5×ATR for qualified signals)
//   TP1 = 1.0 × ATR  (R:R = 1.0 — smaller profit, higher win probability)
//   TP2 = 2.0 × ATR  (R:R = 2.0 — captures larger move if it develops)
//   TP3 = 3.0 × ATR  (R:R = 3.0 — trailing target for remaining position)
//
// This is NOT lowering standards — it's recognizing that candidate signals
// have a real but weaker directional edge that warrants smaller position sizing
// and tighter risk management. The capital protection rules (1% per trade,
// 5% daily loss limit, partial close schedule) still apply.
//
// The microprofit geometry is used ONLY for BUY_CANDIDATE/SELL_CANDIDATE signals.
// Qualified BUY/SELL signals keep the full strategy geometry.
package strategy

import (
	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// CandidateGeometryConfig defines microprofit ATR multipliers for candidate signals.
// These are TIGHTER than the full strategy config to capture microprofit.
type CandidateGeometryConfig struct {
	ATRMultiplierSL  float64
	ATRMultiplierTP1 float64
	ATRMultiplierTP2 float64
	ATRMultiplierTP3 float64
	MinRR            float64
}

// DefaultCandidateGeometry returns microprofit geometry configs per strategy.
// Each strategy gets its own candidate geometry based on its timeframe and risk profile.
func DefaultCandidateGeometry() map[types.StrategyID]CandidateGeometryConfig {
	return map[types.StrategyID]CandidateGeometryConfig{
		// Ultra Scalping: M1, tightest stops for microprofit
		// Full config: SL=1.5, TP1=1.5, TP2=3.0, TP3=6.0
		// Candidate: SL=1.0, TP1=1.0, TP2=2.0, TP3=3.0 (R:R = 1:1, 1:2, 1:3)
		types.StrategyUltraScalping: {
			ATRMultiplierSL:  1.0,
			ATRMultiplierTP1: 1.5,
			ATRMultiplierTP2: 2.5,
			ATRMultiplierTP3: 4.0,
			MinRR:            1.5, // R:R1 must be >= 1.5 for profitability
		},
		// Standard Scalping: M5, slightly wider for microprofit
		// Full config: SL=2.0, TP1=2.0, TP2=4.0, TP3=8.0
		// Candidate: SL=1.0, TP1=1.5, TP2=2.5, TP3=4.0
		types.StrategyStandardScalping: {
			ATRMultiplierSL:  1.0,
			ATRMultiplierTP1: 1.5,
			ATRMultiplierTP2: 2.5,
			ATRMultiplierTP3: 4.0,
			MinRR:            1.5,
		},
		// Standard Swing: H1, medium-term microprofit
		// Full config: SL=2.0, TP1=2.0, TP2=4.0, TP3=8.0
		// Candidate: SL=1.5, TP1=1.5, TP2=3.0, TP3=5.0
		types.StrategyStandardSwing: {
			ATRMultiplierSL:  1.5,
			ATRMultiplierTP1: 2.5,
			ATRMultiplierTP2: 4.0,
			ATRMultiplierTP3: 6.0,
			MinRR:            1.5,
		},
		// Trend Swing: H4, widest for trend-following microprofit
		// Full config: SL=2.5, TP1=2.5, TP2=5.0, TP3=10.0
		// Candidate: SL=2.0, TP1=2.0, TP2=3.5, TP3=5.0
		types.StrategyTrendSwing: {
			ATRMultiplierSL:  2.0,
			ATRMultiplierTP1: 3.0,
			ATRMultiplierTP2: 5.0,
			ATRMultiplierTP3: 8.0,
			MinRR:            1.5,
		},
	}
}

// GetCandidateGeometryConfig returns the microprofit geometry config for a strategy.
func GetCandidateGeometryConfig(strategyID types.StrategyID) CandidateGeometryConfig {
	if cfg, ok := DefaultCandidateGeometry()[strategyID]; ok {
		return cfg
	}
	// Fallback: conservative defaults
	return CandidateGeometryConfig{
		ATRMultiplierSL:  1.0,
		ATRMultiplierTP1: 1.5,
		ATRMultiplierTP2: 2.5,
		ATRMultiplierTP3: 4.0,
		MinRR:            1.5,
	}
}

// BuildCandidateTradeGeometry computes microprofit Entry/SL/TP for candidate signals.
// This is separate from BuildTradeGeometry (used for qualified signals) to ensure
// candidates get tighter, microprofit-optimized geometry.
func BuildCandidateTradeGeometry(state *features.MarketState, direction types.Direction, strategyID types.StrategyID) TradeGeometry {
	geo := TradeGeometry{EntryType: "MARKET"}

	if state == nil {
		geo.ReasonCode = "NO_MARKET_STATE"
		return geo
	}

	if direction != types.DirectionBuy && direction != types.DirectionSell {
		geo.ReasonCode = "NO_DIRECTION"
		return geo
	}

	// Get ATR — use the engine's ATR (from MT5 or locally computed)
	atr := state.Indicators.ATR
	if atr.IsZero() {
		geo.ReasonCode = "ATR_NOT_READY"
		return geo
	}

	// Get candidate-specific geometry config
	candidateCfg := GetCandidateGeometryConfig(strategyID)

	// Entry price: use Ask for BUY, Bid for SELL
	if direction == types.DirectionBuy {
		if !state.Ask.IsZero() {
			geo.Entry = state.Ask
		} else {
			geo.Entry = state.CurrentPrice
		}
	} else {
		if !state.Bid.IsZero() {
			geo.Entry = state.Bid
		} else {
			geo.Entry = state.CurrentPrice
		}
	}

	if geo.Entry.IsZero() {
		geo.ReasonCode = "NO_ENTRY_PRICE"
		return geo
	}

	// Compute microprofit SL/TP using candidate ATR multipliers
	atrSL := atr.Mul(decimal.NewFromFloat(candidateCfg.ATRMultiplierSL))
	atrTP1 := atr.Mul(decimal.NewFromFloat(candidateCfg.ATRMultiplierTP1))
	atrTP2 := atr.Mul(decimal.NewFromFloat(candidateCfg.ATRMultiplierTP2))
	atrTP3 := atr.Mul(decimal.NewFromFloat(candidateCfg.ATRMultiplierTP3))

	// Defense-in-depth: never let a corrupted ATR (e.g. ≈ price) produce an
	// impossible TP/SL (TP1 = entry - entry*2.5). Cap every distance at 5% of
	// entry — a target farther than that is not a real level and would be
	// rejected by the EA anyway. Normal gold TPs are < ~1% of price, so this
	// only triggers on malformed ATR input.
	maxDist := geo.Entry.Mul(decimal.NewFromFloat(0.05))
	if atrSL.GreaterThan(maxDist) {
		atrSL = maxDist
	}
	if atrTP1.GreaterThan(maxDist) {
		atrTP1 = maxDist
	}
	if atrTP2.GreaterThan(maxDist) {
		atrTP2 = maxDist
	}
	if atrTP3.GreaterThan(maxDist) {
		atrTP3 = maxDist
	}

	if direction == types.DirectionBuy {
		geo.StopLoss = geo.Entry.Sub(atrSL)
		geo.TP1 = geo.Entry.Add(atrTP1)
		geo.TP2 = geo.Entry.Add(atrTP2)
		geo.TP3 = geo.Entry.Add(atrTP3)
	} else {
		geo.StopLoss = geo.Entry.Add(atrSL)
		geo.TP1 = geo.Entry.Sub(atrTP1)
		geo.TP2 = geo.Entry.Sub(atrTP2)
		geo.TP3 = geo.Entry.Sub(atrTP3)
	}

	// Validate geometry
	geo.RiskDistance = geo.Entry.Sub(geo.StopLoss).Abs()
	if geo.RiskDistance.IsZero() {
		geo.ReasonCode = "ZERO_RISK_DISTANCE"
		return geo
	}

	// Compute R:R ratios
	geo.GrossRR1 = geo.TP1.Sub(geo.Entry).Abs().Div(geo.RiskDistance)
	geo.GrossRR2 = geo.TP2.Sub(geo.Entry).Abs().Div(geo.RiskDistance)
	geo.GrossRR3 = geo.TP3.Sub(geo.Entry).Abs().Div(geo.RiskDistance)

	// Net R:R (after spread/cost) — use spread as round-trip cost estimate
	roundTripCost := decimal.Zero
	if !state.Spread.IsZero() {
		roundTripCost = state.Spread
	}
	if geo.RiskDistance.Add(roundTripCost).GreaterThan(decimal.Zero) {
		geo.NetRR1 = geo.TP1.Sub(geo.Entry).Abs().Sub(roundTripCost).Div(geo.RiskDistance.Add(roundTripCost))
		geo.NetRR2 = geo.TP2.Sub(geo.Entry).Abs().Sub(roundTripCost).Div(geo.RiskDistance.Add(roundTripCost))
		geo.NetRR3 = geo.TP3.Sub(geo.Entry).Abs().Sub(roundTripCost).Div(geo.RiskDistance.Add(roundTripCost))
	}

	// Validate: SL must be below entry for BUY, above for SELL
	if direction == types.DirectionBuy {
		if !geo.StopLoss.LessThan(geo.Entry) {
			geo.ReasonCode = "INVALID_SL_FOR_BUY"
			return geo
		}
		if !geo.TP1.GreaterThan(geo.Entry) {
			geo.ReasonCode = "INVALID_TP_FOR_BUY"
			return geo
		}
	} else {
		if !geo.StopLoss.GreaterThan(geo.Entry) {
			geo.ReasonCode = "INVALID_SL_FOR_SELL"
			return geo
		}
		if !geo.TP1.LessThan(geo.Entry) {
			geo.ReasonCode = "INVALID_TP_FOR_SELL"
			return geo
		}
	}

	geo.Valid = true
	return geo
}
