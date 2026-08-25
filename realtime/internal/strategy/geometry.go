// Package strategy — Trade geometry builder for candidate and executable signals.
// SOW Phase 2 Sections 4-12: Canonical Entry/SL/TP computation.
//
// This function is called AFTER direction is determined but BEFORE the signal
// is returned — whether it's a candidate or executable signal.
// It uses the same structural+ATR logic as the strategy's own computation.
package strategy

import (
	"fmt"

	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// TradeGeometry holds computed entry/SL/TP levels with validation.
type TradeGeometry struct {
	Entry        decimal.Decimal
	StopLoss     decimal.Decimal
	TP1          decimal.Decimal
	TP2          decimal.Decimal
	TP3          decimal.Decimal
	RiskDistance decimal.Decimal
	GrossRR1     decimal.Decimal
	GrossRR2     decimal.Decimal
	GrossRR3     decimal.Decimal
	NetRR1       decimal.Decimal
	NetRR2       decimal.Decimal
	NetRR3       decimal.Decimal
	Valid        bool
	ReasonCode   string
	EntryType    string // MARKET, LIMIT, STOP
}

// BuildTradeGeometry computes Entry/SL/TP from market state and strategy config.
// This is the CANONICAL geometry function — used for both candidates and executable signals.
// It does NOT duplicate strategy logic — it calls the same computeStructuralSLTP function.
func BuildTradeGeometry(state *features.MarketState, direction types.Direction, cfg StrategyConfig) TradeGeometry {
	geo := TradeGeometry{EntryType: "MARKET"}

	if state == nil {
		geo.ReasonCode = "NO_MARKET_STATE"
		return geo
	}

	if direction != types.DirectionBuy && direction != types.DirectionSell {
		geo.ReasonCode = "NO_DIRECTION"
		return geo
	}

	// ATR must be ready for geometry — use strategy-specific timeframe ATR
	atr := getStrategyATR(state, cfg.StrategyID)
	if atr.IsZero() {
		// Fallback to indicator ATR
		atr = state.Indicators.ATR
	}
	if atr.IsZero() {
		geo.ReasonCode = "ATR_NOT_READY"
		return geo
	}

	// Entry price: use Ask for BUY, Bid for SELL (broker-realistic)
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

	// Get structural levels for SL
	structLow := getStructuralLow(state)
	structHigh := getStructuralHigh(state)

	// Compute SL/TP using the canonical strategy function
	// Override ATR with strategy-specific timeframe ATR for correct SL/TP distances
	originalATR := state.Indicators.ATR
	state.Indicators.ATR = atr
	// We use our Ask/Bid entry, and take only SL/TP from computeStructuralSLTP
	_, sl, tp1, tp2, tp3 := computeStructuralSLTP(state, direction, cfg, structLow, structHigh)
	// Restore original ATR
	state.Indicators.ATR = originalATR

	// If structural SL failed, check exit profile FIRST, then ATR fallback
	if sl.IsZero() {
		exitProfile := LoadExitProfile(string(cfg.StrategyID))
		if exitProfile != nil && exitProfile.CalculationMode == "PERCENTAGE" {
			pSL, pTP1, pTP2, pTP3 := computePercentageSLTP(geo.Entry, direction, atr, exitProfile)
			if !pSL.IsZero() {
				geo.StopLoss = pSL
				geo.TP1 = pTP1
				geo.TP2 = pTP2
				geo.TP3 = pTP3
				return geo
			}
		}
		// Final fallback: ATR multiplier mode
		if direction == types.DirectionBuy {
			sl = geo.Entry.Sub(atr.Mul(decimal.NewFromFloat(cfg.ATRMultiplierSL)))
			tp1 = geo.Entry.Add(atr.Mul(decimal.NewFromFloat(cfg.ATRMultiplierTP1)))
			tp2 = geo.Entry.Add(atr.Mul(decimal.NewFromFloat(cfg.ATRMultiplierTP2)))
			tp3 = geo.Entry.Add(atr.Mul(decimal.NewFromFloat(cfg.ATRMultiplierTP3)))
		} else {
			sl = geo.Entry.Add(atr.Mul(decimal.NewFromFloat(cfg.ATRMultiplierSL)))
			tp1 = geo.Entry.Sub(atr.Mul(decimal.NewFromFloat(cfg.ATRMultiplierTP1)))
			tp2 = geo.Entry.Sub(atr.Mul(decimal.NewFromFloat(cfg.ATRMultiplierTP2)))
			tp3 = geo.Entry.Sub(atr.Mul(decimal.NewFromFloat(cfg.ATRMultiplierTP3)))
		}
	}

	geo.StopLoss = sl
	geo.TP1 = tp1
	geo.TP2 = tp2
	geo.TP3 = tp3

	// Validate geometry
	if sl.IsZero() || tp1.IsZero() {
		geo.ReasonCode = "GEOMETRY_INVALID"
		return geo
	}

	// Risk distance
	geo.RiskDistance = geo.Entry.Sub(sl).Abs()

	if geo.RiskDistance.IsZero() {
		geo.ReasonCode = "ZERO_RISK"
		return geo
	}

	// Gross RR
	risk := geo.RiskDistance
	geo.GrossRR1 = geo.TP1.Sub(geo.Entry).Abs().Div(risk)
	geo.GrossRR2 = geo.TP2.Sub(geo.Entry).Abs().Div(risk)
	geo.GrossRR3 = geo.TP3.Sub(geo.Entry).Abs().Div(risk)

	// Net RR (account for spread cost)
	spread := state.Spread
	if !spread.IsZero() && !risk.IsZero() {
		netRisk := risk.Add(spread)
		geo.NetRR1 = geo.TP1.Sub(geo.Entry).Abs().Sub(spread).Div(netRisk)
		geo.NetRR2 = geo.TP2.Sub(geo.Entry).Abs().Sub(spread).Div(netRisk)
		geo.NetRR3 = geo.TP3.Sub(geo.Entry).Abs().Sub(spread).Div(netRisk)
	} else {
		geo.NetRR1 = geo.GrossRR1
		geo.NetRR2 = geo.GrossRR2
		geo.NetRR3 = geo.GrossRR3
	}

	// Validate ordering: BUY: SL < Entry < TP1 <= TP2 <= TP3
	if direction == types.DirectionBuy {
		if !geo.StopLoss.LessThan(geo.Entry) {
			geo.ReasonCode = fmt.Sprintf("BUY_GEOMETRY_INVALID: SL(%s) >= Entry(%s)", geo.StopLoss.String(), geo.Entry.String())
			return geo
		}
		if !geo.TP1.GreaterThan(geo.Entry) {
			geo.ReasonCode = fmt.Sprintf("BUY_GEOMETRY_INVALID: TP1(%s) <= Entry(%s)", geo.TP1.String(), geo.Entry.String())
			return geo
		}
	}
	// SELL: TP3 <= TP2 <= TP1 < Entry < SL
	if direction == types.DirectionSell {
		if !geo.StopLoss.GreaterThan(geo.Entry) {
			geo.ReasonCode = fmt.Sprintf("SELL_GEOMETRY_INVALID: SL(%s) <= Entry(%s)", geo.StopLoss.String(), geo.Entry.String())
			return geo
		}
		if !geo.TP1.LessThan(geo.Entry) {
			geo.ReasonCode = fmt.Sprintf("SELL_GEOMETRY_INVALID: TP1(%s) >= Entry(%s)", geo.TP1.String(), geo.Entry.String())
			return geo
		}
	}

	geo.Valid = true
	return geo
}

// GetStrategyConfig returns the config for a given strategy instance.
func GetStrategyConfig(s Strategy) StrategyConfig {
	return *getStrategyConfig(s)
}

// MinDominanceMargin returns the minimum required score difference between long and short
// to produce a directional candidate. Prevents flip-flopping on near-equal scores.
const MinDominanceMargin = 5.0

// CheckDirectionDominance verifies that long/short scores have sufficient separation.
func CheckDirectionDominance(longScore, shortScore decimal.Decimal) (types.Direction, bool) {
	diff := longScore.Sub(shortScore).Abs()
	minMargin := decimal.NewFromFloat(MinDominanceMargin)

	if diff.LessThan(minMargin) {
		return types.DirectionNoTrade, false
	}

	if longScore.GreaterThan(shortScore) {
		return types.DirectionBuy, true
	}
	return types.DirectionSell, true
}

// getStrategyATR returns the ATR from the timeframe appropriate for each strategy.
// Scalping strategies use M5 ATR (tight), swing strategies use M15/H1 ATR (medium),
// trend strategies use H1/H4 ATR (wide).
//
// This prevents the problem where H1 ATR ($24) is used for scalping strategies,
// resulting in SL/TP distances 10-20x too far.
func getStrategyATR(state *features.MarketState, strategyID types.StrategyID) decimal.Decimal {
	// Determine which timeframe ATR to use based on strategy
	var preferredTFs []types.Timeframe
	switch strategyID {
	case types.StrategyUltraScalping:
		// Ultra scalping: tightest — M1 first, M5 fallback
		preferredTFs = []types.Timeframe{types.TFM1, types.TFM5}
	case types.StrategyStandardScalping:
		// Standard scalping: M5
		preferredTFs = []types.Timeframe{types.TFM5, types.TFM1}
	case types.StrategyStandardSwing:
		// Standard swing: M15 or H1
		preferredTFs = []types.Timeframe{types.TFM15, types.TFH1}
	case types.StrategyTrendSwing:
		// Trend swing: H1 or H4
		preferredTFs = []types.Timeframe{types.TFH1, types.TFH4}
	default:
		preferredTFs = []types.Timeframe{types.TFM5}
	}

	// Try to get ATR from the preferred timeframe's candle
	for _, tf := range preferredTFs {
		if candle, ok := state.Candles[tf]; ok && candle != nil {
			// We need to compute ATR from this candle's range
			// The IndicatorEngine computes ATR from the primary candle,
			// but we need it from the strategy's preferred timeframe.
			// For now, use the candle's range as an ATR proxy if the
			// indicator ATR doesn't match this timeframe.
			candleRange := candle.High.Sub(candle.Low)
			if candleRange.IsPositive() {
				// ATR(14) ≈ average of last 14 candle ranges
				// As a quick approximation, use the candle range directly
				// (conservative — actual ATR is usually similar but smoother)
				return candleRange
			}
		}
	}

	// Fallback: use the state's indicator ATR (whatever timeframe it was computed on)
	return state.Indicators.ATR
}
