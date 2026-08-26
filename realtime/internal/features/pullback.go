// Package features — Pullback Feature Extraction (GitHub Reference P2-003)
// Detects pullbacks within established trends and produces structured metrics:
// pullback depth (% of trend move), ATR-normalized retracement distance,
// continuation confirmation (price resuming trend direction), and quality score.
//
// Reference inspiration: backtrader-pullback-window-xauusd (MIT)
// Clean-room reimplementation in PAT's feature architecture.
// ACTIVE mode: features feed into strategy evidence scoring.
package features

import (
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// PullbackEngine detects pullbacks and computes structured retracement metrics.
type PullbackEngine struct {
	trendAnchorPrice   decimal.Decimal
	trendDirection     types.Direction
	trendExtreme       decimal.Decimal
	pullbackInProgress bool
}

func NewPullbackEngine() *PullbackEngine {
	return &PullbackEngine{}
}

// ProcessEvaluated computes pullback features from structure and price data.
func (e *PullbackEngine) ProcessEvaluated(structure StructureFeatures, currentPrice, atr decimal.Decimal) PullbackFeatures {
	pf := PullbackFeatures{}
	if currentPrice.IsZero() {
		return pf
	}

	swingsHigh := structure.SwingHighs
	swingsLow := structure.SwingLows
	if len(swingsHigh) == 0 && len(swingsLow) == 0 {
		return pf
	}

	currentTrend := structure.CurrentTrend
	if currentTrend == "bullish" && len(swingsLow) > 0 {
		e.trendAnchorPrice = swingsLow[len(swingsLow)-1]
		e.trendDirection = types.DirectionBuy
	} else if currentTrend == "bearish" && len(swingsHigh) > 0 {
		e.trendAnchorPrice = swingsHigh[len(swingsHigh)-1]
		e.trendDirection = types.DirectionSell
	} else {
		return pf
	}

	trendMove := currentPrice.Sub(e.trendAnchorPrice).Abs()
	if trendMove.IsZero() {
		return pf
	}

	if e.trendExtreme.IsZero() {
		e.trendExtreme = currentPrice
	}
	if e.trendDirection == types.DirectionBuy && currentPrice.GreaterThan(e.trendExtreme) {
		e.trendExtreme = currentPrice
	} else if e.trendDirection == types.DirectionSell && currentPrice.LessThan(e.trendExtreme) {
		e.trendExtreme = currentPrice
	}

	extremeDist := e.trendExtreme.Sub(e.trendAnchorPrice).Abs()
	if extremeDist.IsZero() {
		return pf
	}

	pullbackDist := e.trendExtreme.Sub(currentPrice).Abs()
	pf.PullbackDepthPct = pullbackDist.Div(extremeDist)
	pf.PullbackAnchor = e.trendAnchorPrice
	pf.PullbackExtreme = e.trendExtreme

	if atr.GreaterThan(decimal.Zero) {
		pf.PullbackATRNorm = pullbackDist.Div(atr)
	}

	one := decimal.NewFromInt(1)
	pf.PullbackActive = pf.PullbackDepthPct.GreaterThan(decimal.NewFromFloat(0.20)) &&
		pf.PullbackDepthPct.LessThan(one)

	// Continuation confirmation
	if e.trendDirection == types.DirectionBuy {
		recoveryLevel := e.trendExtreme.Sub(pullbackDist).Add(atr.Mul(decimal.NewFromFloat(0.3)))
		pf.PullbackContConfirm = e.pullbackInProgress && currentPrice.GreaterThan(recoveryLevel)
	} else {
		recoveryLevel := e.trendExtreme.Add(pullbackDist).Sub(atr.Mul(decimal.NewFromFloat(0.3)))
		pf.PullbackContConfirm = e.pullbackInProgress && currentPrice.LessThan(recoveryLevel)
	}
	e.pullbackInProgress = pf.PullbackActive

	// Quality score
	if pf.PullbackActive {
		idealDepth := decimal.NewFromFloat(0.50)
		depthQuality := one.Sub(decimal.Min(
			pf.PullbackDepthPct.Sub(idealDepth).Abs().Mul(decimal.NewFromInt(2)), one))
		atrQuality := decimal.Zero
		if pf.PullbackATRNorm.GreaterThan(decimal.NewFromFloat(0.5)) &&
			pf.PullbackATRNorm.LessThanOrEqual(decimal.NewFromFloat(3.0)) {
			atrQuality = one
		} else if pf.PullbackATRNorm.LessThanOrEqual(decimal.NewFromFloat(5.0)) {
			atrQuality = decimal.NewFromFloat(0.5)
		}
		pf.PullbackQuality = depthQuality.Mul(decimal.NewFromFloat(0.6)).
			Add(atrQuality.Mul(decimal.NewFromFloat(0.4)))
	}
	return pf
}
