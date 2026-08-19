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

	// ATR must be ready for geometry
	if state.Indicators.ATR.IsZero() {
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
	// We use our Ask/Bid entry, and take only SL/TP from computeStructuralSLTP
	_, sl, tp1, tp2, tp3 := computeStructuralSLTP(state, direction, cfg, structLow, structHigh)

	// If structural SL failed, fall back to ATR-based SL using our entry
	if sl.IsZero() {
		atr := state.Indicators.ATR
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
