// Package features — Pullback Feature Extraction (GitHub Reference P2-003)
// Detects pullbacks within established trends and produces structured metrics:
// pullback depth (% of trend move), ATR-normalized retracement distance,
// continuation confirmation (price resuming trend direction), and quality score.
//
// Reference inspiration: backtrader-pullback-window-xauusd (MIT) —
// pullback window detection with ATR normalization.
// Clean-room reimplementation in PAT's feature architecture.
//
// Shadow mode: features are computed and stored in MarketState but do not
// affect signal scoring until validated through ablation testing.
package features

import (
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// PullbackEngine detects pullbacks and computes structured retracement metrics.
type PullbackEngine struct {
	// trendAnchor holds the most recent swing high/low that defines the trend.
	trendAnchorPrice decimal.Decimal
	// trendDirection is BUY for uptrend (pullback = price dips below anchor).
	trendDirection   types.Direction
	// trendStart tracks when the trend was detected.
	trendExtreme     decimal.Decimal // farthest price in trend direction
	// pullbackInProgress tracks active retracement state.
	pullbackInProgress bool
}

// NewPullbackEngine creates a pullback detection engine with default state.
func NewPullbackEngine() *PullbackEngine {
	return &PullbackEngine{}
}

// Process evaluates the current market state for pullback conditions.
// It uses structure swing highs/lows to determine the trend anchor,
// then measures retracement distance from the current price.
//
// A pullback is detected when:
//   - A trend exists (swing high/low defined, direction established)
//   - Price has moved significantly away from the anchor (trend extreme defined)
//   - Price retraces toward the anchor without breaking structure (BOS) yet
func (e *PullbackEngine) Process(state *MarketState) PullbackFeatures {
	pf := PullbackFeatures{}
	if state == nil {
		return pf
	}

	// Determine trend from structure
	swingsHigh := state.Structure.SwingHighs
	swingsLow := state.Structure.SwingLows

	if len(swingsHigh) == 0 && len(swingsLow) == 0 {
		return pf
	}

	currentPrice := state.CurrentPrice
	if currentPrice.IsZero() {
		return pf
	}

	atr := state.Indicators.ATR
	currentTrend := state.Structure.CurrentTrend

	// Determine trend direction and anchor
	if currentTrend == "bullish" && len(swingsLow) > 0 {
		// Uptrend: anchor is the most recent swing low
		anchor := swingsLow[len(swingsLow)-1]
		e.trendAnchorPrice = anchor
		e.trendDirection = types.DirectionBuy
	} else if currentTrend == "bearish" && len(swingsHigh) > 0 {
		// Downtrend: anchor is the most recent swing high
		anchor := swingsHigh[len(swingsHigh)-1]
		e.trendAnchorPrice = anchor
		e.trendDirection = types.DirectionSell
	} else {
		return pf // no clear trend
	}

	// Compute trend move distance
	trendMove := currentPrice.Sub(e.trendAnchorPrice).Abs()
	if trendMove.IsZero() {
		return pf
	}

	// Track trend extreme (farthest price in trend direction)
	if e.trendExtreme.IsZero() {
		e.trendExtreme = currentPrice
	}
	if e.trendDirection == types.DirectionBuy && currentPrice.GreaterThan(e.trendExtreme) {
		e.trendExtreme = currentPrice
	} else if e.trendDirection == types.DirectionSell && currentPrice.LessThan(e.trendExtreme) {
		e.trendExtreme = currentPrice
	}

	// Detect pullback: price retraced from extreme toward the anchor
	extremeDist := e.trendExtreme.Sub(e.trendAnchorPrice).Abs()
	if extremeDist.IsZero() {
		return pf
	}

	pullbackDist := e.trendExtreme.Sub(currentPrice).Abs()
	
	// Pullback depth as % of trend move from anchor
	pf.PullbackDepthPct = pullbackDist.Div(extremeDist)
	pf.PullbackAnchor = e.trendAnchorPrice
	pf.PullbackExtreme = e.trendExtreme

	// ATR-normalized retracement
	if atr.GreaterThan(decimal.Zero) {
		pf.PullbackATRNorm = pullbackDist.Div(atr)
	}

	// A pullback is considered "active" when price has retraced > 20% but < 100%
	depthVal, _ := pf.PullbackDepthPct.Float64()
	twoTenths := decimal.NewFromFloat(0.20)
	one := decimal.NewFromInt(1)
	pf.PullbackActive = pf.PullbackDepthPct.GreaterThan(twoTenths) &&
		pf.PullbackDepthPct.LessThan(one)
	_ = depthVal // suppress unused

	// Continuation confirmation: price has resumed trend direction
	if e.trendDirection == types.DirectionBuy {
		// For uptrend pullback: confirms when price moves back above pullback low
		pf.PullbackContConfirm = e.pullbackInProgress &&
			currentPrice.GreaterThan(e.trendExtreme.Sub(pullbackDist).Add(
				atr.Mul(decimal.NewFromFloat(0.3))))
	} else {
		pf.PullbackContConfirm = e.pullbackInProgress &&
			currentPrice.LessThan(e.trendExtreme.Add(pullbackDist).Sub(
				atr.Mul(decimal.NewFromFloat(0.3))))
	}

	// Track pullback state
	e.pullbackInProgress = pf.PullbackActive

	// Quality score: optimal pullback depth (0.382-0.618) with ATR normalization
	if pf.PullbackActive {
		idealDepth := decimal.NewFromFloat(0.50)
		depthDeviation := pf.PullbackDepthPct.Sub(idealDepth).Abs()
		// Quality = 1.0 - |depth - 0.50| * 2 (so 0.50 = 1.0, edges = 0)
		depthQuality := one.Sub(
			decimal.Min(depthDeviation.Mul(decimal.NewFromInt(2)), one))
		
		// ATR quality: retracement 1-3 ATR is ideal for gold
		atrQuality := decimal.Zero
		if atr.GreaterThan(decimal.Zero) && pf.PullbackATRNorm.GreaterThan(decimal.NewFromFloat(0.5)) {
			if pf.PullbackATRNorm.LessThanOrEqual(decimal.NewFromFloat(3.0)) {
				atrQuality = one // ideal range
			} else if pf.PullbackATRNorm.LessThanOrEqual(decimal.NewFromFloat(5.0)) {
				atrQuality = decimal.NewFromFloat(0.5) // acceptable
			}
		}

		pf.PullbackQuality = depthQuality.Mul(decimal.NewFromFloat(0.6)).
			Add(atrQuality.Mul(decimal.NewFromFloat(0.4)))
	}

	return pf
}
