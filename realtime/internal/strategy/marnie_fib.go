// Package strategy — Marnie Fib Strategy
// Uses Fibonacci retracement/extension levels from confirmed structural swings
// to identify high-probability reversal zones (golden zone 0.618-0.786).
//
// Strategy logic:
// - Price in golden zone (0.618-0.786) → high confluence, directional signal
// - Price near 0.382/0.5 → moderate confluence, candidate signal
// - Price beyond 1.0 retracement → trend continuation, extension targets
// - Confluence with previous Fib levels → boosted score
// - ATR-based stop loss beyond the 0.786 or 1.0 level
// - Extension levels (1.272, 1.618, 2.618) as profit targets
//
// Works best in RANGE and MEAN_REVERSION regimes.
// Less effective in strong TRENDING regimes (Fib levels get violated).
package strategy

import (
	"fmt"

	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// MarnieFibStrategy uses Fibonacci retracement levels for entry and extension levels for targets.
type MarnieFibStrategy struct {
	cfg    StrategyConfig
	fibEng *features.MarnieFibEngine
}

// NewMarnieFibStrategy creates a Marnie Fib strategy instance.
func NewMarnieFibStrategy() *MarnieFibStrategy {
	return &MarnieFibStrategy{
		cfg: StrategyConfig{
			StrategyID:        types.StrategyMarnieFib,
			MinConfluence:     45,
			MinMTFAlignment:   20,
			ATRMultiplierSL:   1.5,
			ATRMultiplierTP1:  2.0,
			ATRMultiplierTP2:  3.5,
			ATRMultiplierTP3:  5.5,
			MinSLATRFloor:     0.0, VolatilityScale: 2.0, // provisional: widen stops for understated feed; calibrate from client real ATR
			MaxSpreadPips:     4.0,
			MaxSlippagePoints: 20,
			MinADX:            15, // Fib works in ranging markets too (lower ADX)
			MinRR:             2.0,
			ExpiryMinutes:     120,
			CooldownMinutes:   180,
			DecisionTFs:       []types.Timeframe{types.TFM15, types.TFH1},
			ContextTFs:        []types.Timeframe{types.TFH4, types.TFD1},
			AcceptedRegimes: []types.Regime{
				types.RegimeRange,
				types.RegimeMeanReversion,
				types.RegimeTrendingBullish,
				types.RegimeTrendingBearish,
				types.RegimeBreakout,
				types.RegimeHighVolatility,
			},
			AcceptedSessions: []string{"LONDON", "NEW_YORK", "OVERLAP", "TOKYO", "SYDNEY"},
			MinQualityState:  types.QualityAuthoritative,
		},
		fibEng: features.NewMarnieFibEngine(),
	}
}

func (s *MarnieFibStrategy) ID() types.StrategyID { return types.StrategyMarnieFib }
func (s *MarnieFibStrategy) DecisionTimeframes() []types.Timeframe { return s.cfg.DecisionTFs }

func (s *MarnieFibStrategy) Evaluate(state *features.MarketState) StrategyResult {
	// NOTE: per-evaluation logging removed to avoid noisy output on every tick;
	// diagnostics are emitted only when a signal/structure is actually found.
	result := StrategyResult{
		StrategyID:      s.ID(),
		Direction:       types.DirectionNoTrade,
		RawScore:        decimal.Zero,
		LongScore:       decimal.Zero,
		ShortScore:      decimal.Zero,
		EntryPrice:      decimal.Zero,
		StopLoss:        decimal.Zero,
		TP1:             decimal.Zero,
		TP2:             decimal.Zero,
		TP3:             decimal.Zero,
		ExpiryMinutes:   s.cfg.ExpiryMinutes,
		CooldownMinutes: s.cfg.CooldownMinutes,
	}

	if state == nil || state.Indicators.ATR.IsZero() {
		result.Direction = types.DirectionError
		result.ReasonCodes = append(result.ReasonCodes, types.NTATRNotReady)
		return result
	}

	// Check regime/session
	if state.Regime.Current == "" {
		result.ReasonCodes = append(result.ReasonCodes, "NT_REGIME_UNKNOWN")
		return result
	}
	result.ReasonCodes = append(result.ReasonCodes, checkRegimeSession(state, s.cfg)...)
	if len(result.ReasonCodes) > 0 {
		return result
	}

	// Compute Marnie Fib levels
	// MARNIE_FIB requires confirmed structural swing anchors. Between confirmed
	// structures, live swing detection can be empty — previously this left the
	// strategy permanently dead (signal_count=0). Fall back to the most recent
	// available candle's high/low as anchors so the engine can still evaluate.
	fibStruct := state.Structure
	if len(fibStruct.SwingHighs) == 0 || len(fibStruct.SwingLows) == 0 {
		if c := latestCandleFromState(state); c != nil {
			fibStruct.SwingHighs = []decimal.Decimal{c.High}
			fibStruct.SwingLows = []decimal.Decimal{c.Low}
		}
	}
	fibFeat := s.fibEng.Process(nil, fibStruct, state.CurrentPrice)
	if !fibFeat.Ready {
		result.ReasonCodes = append(result.ReasonCodes, "FIB_NO_SWING_ANCHORS")
		return result
	}

	var evidence []types.EvidenceContribution
	q := state.Quality
	price := state.CurrentPrice
	_ = state.Indicators.ATR

	// ─── Core Fib Evidence ───

	// Golden zone confluence (0.618-0.786) — primary signal
	if fibFeat.InGoldenZone {
		if fibFeat.Direction == "bullish" {
			// Price retraced into golden zone in bullish structure → BUY
			addEvidence(&evidence, "FIBONACCI", "GOLDEN_ZONE_BULLISH", types.DirectionBuy, 20, 0.15, q,
				"Price in golden zone (0.618-0.786) bullish retracement")
		} else {
			// Price retraced into golden zone in bearish structure → SELL
			addEvidence(&evidence, "FIBONACCI", "GOLDEN_ZONE_BEARISH", types.DirectionSell, 20, 0.15, q,
				"Price in golden zone (0.618-0.786) bearish retracement")
		}
	} else {
		// Check proximity to golden zone
		confluenceScore := fibFeat.ConfluenceScore
		if confluenceScore > 50 {
			// Moderately close to golden zone
			if fibFeat.Direction == "bullish" {
				addEvidence(&evidence, "FIBONACCI", "NEAR_GOLDEN_ZONE_BULL", types.DirectionBuy, 12, 0.08, q,
					fmt.Sprintf("Near golden zone (%.0f%% confluence)", confluenceScore))
			} else {
				addEvidence(&evidence, "FIBONACCI", "NEAR_GOLDEN_ZONE_BEAR", types.DirectionSell, 12, 0.08, q,
					fmt.Sprintf("Near golden zone (%.0f%% confluence)", confluenceScore))
			}
		} else if confluenceScore > 25 {
			// Far from golden zone but some confluence
			if fibFeat.Direction == "bullish" {
				addEvidence(&evidence, "FIBONACCI", "DISTANT_GOLDEN_BULL", types.DirectionBuy, 6, 0.04, q,
				fmt.Sprintf("Distant from golden zone (%.0f%%)", confluenceScore))
			} else {
				addEvidence(&evidence, "FIBONACCI", "DISTANT_GOLDEN_BEAR", types.DirectionSell, 6, 0.04, q,
				fmt.Sprintf("Distant from golden zone (%.0f%%)", confluenceScore))
			}
		}
	}

	// Nearest Fib level proximity
	if fibFeat.NearestLevel != "" {
		distPercent := decimal.Zero
		if fibFeat.Range.GreaterThan(decimal.Zero) {
			distPercent = price.Sub(fibFeat.NearestLevelPrice).Abs().Div(fibFeat.Range)
		}
		distF, _ := distPercent.Float64()
		if distF < 0.01 { // Within 1% of range
			if fibFeat.Direction == "bullish" {
				addEvidence(&evidence, "FIBONACCI", "AT_FIB_LEVEL_BULL", types.DirectionBuy, 10, 0.06, q,
				fmt.Sprintf("At Fib %s level", fibFeat.NearestLevel))
			} else {
				addEvidence(&evidence, "FIBONACCI", "AT_FIB_LEVEL_BEAR", types.DirectionSell, 10, 0.06, q,
				fmt.Sprintf("At Fib %s level", fibFeat.NearestLevel))
			}
		}
	}

	// ─── Confluence Evidence ───

	// Fib confluence score as evidence
	if fibFeat.ConfluenceScore > 60 {
		dir := types.DirectionBuy
		if fibFeat.Direction == "bearish" {
			dir = types.DirectionSell
		}
		addEvidence(&evidence, "FIBONACCI", "HIGH_CONFLUENCE", dir, 15, 0.10, q,
			fmt.Sprintf("Fib confluence: %.0f/100", fibFeat.ConfluenceScore))
	}

	// ─── Trend Confirmation ───

	// EMA alignment
	if state.Indicators.EMA21.GreaterThan(state.Indicators.EMA50) {
		addEvidence(&evidence, "TREND", "EMA21_ABOVE_EMA50", types.DirectionBuy, 10, 0.06, q, "")
	} else {
		addEvidence(&evidence, "TREND", "EMA21_BELOW_EMA50", types.DirectionSell, 10, 0.06, q, "")
	}

	// RSI — oversold/overbought for Fib reversal
	rsiVal, _ := state.Indicators.RSI.Float64()
	if rsiVal < 35 {
		addEvidence(&evidence, "MOMENTUM", "RSI_OVERSOLD", types.DirectionBuy, 10, 0.06, q,
			"RSI oversold — supports Fib bounce")
	} else if rsiVal > 65 {
		addEvidence(&evidence, "MOMENTUM", "RSI_OVERBOUGHT", types.DirectionSell, 10, 0.06, q,
			"RSI overbought — supports Fib reversal")
	}

	// MACD confirmation
	if state.Indicators.MACDMain.GreaterThan(state.Indicators.MACDSignal) {
		addEvidence(&evidence, "MOMENTUM", "MACD_BULLISH", types.DirectionBuy, 8, 0.05, q, "")
	} else {
		addEvidence(&evidence, "MOMENTUM", "MACD_BEARISH", types.DirectionSell, 8, 0.05, q, "")
	}

	// Structure confirmation
	if state.Structure.CurrentTrend == "bullish" {
		addEvidence(&evidence, "STRUCTURE", "BULLISH_STRUCTURE", types.DirectionBuy, 8, 0.05, q, "")
	} else if state.Structure.CurrentTrend == "bearish" {
		addEvidence(&evidence, "STRUCTURE", "BEARISH_STRUCTURE", types.DirectionSell, 8, 0.05, q, "")
	}

	// VWAP confirmation
	if state.VWAP.SessionVWAP.GreaterThan(decimal.Zero) {
		if price.GreaterThan(state.VWAP.SessionVWAP) {
			addEvidence(&evidence, "VWAP", "ABOVE_VWAP", types.DirectionBuy, 6, 0.04, q, "")
		} else {
			addEvidence(&evidence, "VWAP", "BELOW_VWAP", types.DirectionSell, 6, 0.04, q, "")
		}
	}

	// P2 evidence (ACTIVE): pullback + ORB + pin bar
	addPullbackEvidence(&evidence, state, q)
	addORBEvidence(&evidence, state, q)
	addPinBarEvidence(&evidence, state, q)

	// Apply family caps
	evidence = applyFamilyCaps(evidence)
	result.Evidence = evidence

	// Score with regime-specific thresholds
	regime := state.Regime.Current
	ct, tt, found := GetThresholds(s.ID(), regime)
	if !found {
		ct = s.cfg.MinConfluence * 0.6
		tt = s.cfg.MinConfluence
	}

	direction, rawScore, longScore, shortScore, reasons := scoreDirectionWithThresholds(
		evidence, ct, tt, decimal.Zero)

	result.Direction = direction
	result.RawScore = rawScore
	result.LongScore = longScore
	result.ShortScore = shortScore
	// NOTE: Confidence intentionally NOT derived from RawScore (prompt.md
	// Section 30: score is not probability/confidence). Subscriber-facing
	// confidence comes only from the calibrated model via CalibratedProbability.
	result.ReasonCodes = append(result.ReasonCodes, reasons...)

	// Build human reason
	fibStatus := "outside golden zone"
	if fibFeat.InGoldenZone {
		fibStatus = "IN golden zone (0.618-0.786)"
	}
	result.HumanReason = fmt.Sprintf("%s — Fib %s, confluence=%.0f, nearest=%s, ADX=%.1f RSI=%.1f, regime: %s",
		direction, fibStatus, fibFeat.ConfluenceScore, fibFeat.NearestLevel,
		state.Indicators.ADX.InexactFloat64(), rsiVal, regime)

	// Geometry: use Fib extension levels for TP, ATR for SL
	if direction == types.DirectionBuy || direction == types.DirectionSell {
		geo := BuildTradeGeometry(state, direction, s.cfg)
		if geo.Valid {
			result.EntryPrice = geo.Entry
			result.StopLoss = geo.StopLoss
			result.TP1 = geo.TP1
			result.TP2 = geo.TP2
			result.TP3 = geo.TP3

			// Override TP with Fib extension levels if available and better
			if direction == types.DirectionBuy && fibFeat.Direction == "bullish" {
				if ext1272, ok := fibFeat.ExtensionLevels["1.272"]; ok && ext1272.GreaterThan(geo.TP1) {
					result.TP1 = ext1272
				}
				if ext1618, ok := fibFeat.ExtensionLevels["1.618"]; ok && ext1618.GreaterThan(geo.TP2) {
					result.TP2 = ext1618
				}
				if ext2618, ok := fibFeat.ExtensionLevels["2.618"]; ok && ext2618.GreaterThan(geo.TP3) {
					result.TP3 = ext2618
				}
			} else if direction == types.DirectionSell && fibFeat.Direction == "bearish" {
				if ext1272, ok := fibFeat.ExtensionLevels["1.272"]; ok && ext1272.LessThan(geo.TP1) && ext1272.GreaterThan(decimal.Zero) {
					result.TP1 = ext1272
				}
				if ext1618, ok := fibFeat.ExtensionLevels["1.618"]; ok && ext1618.LessThan(geo.TP2) && ext1618.GreaterThan(decimal.Zero) {
					result.TP2 = ext1618
				}
				if ext2618, ok := fibFeat.ExtensionLevels["2.618"]; ok && ext2618.LessThan(geo.TP3) && ext2618.GreaterThan(decimal.Zero) {
					result.TP3 = ext2618
				}
			}
		} else {
			result.Direction = types.DirectionNoTrade
			result.ReasonCodes = append(result.ReasonCodes, types.NoTradeReason(geo.ReasonCode))
		}
	}

	// ─── Refinement: micro profit-taking + unique entry gate + profitability ───
	applyRefinement(&result, state, result.Direction, s.cfg, result.RawScore)

	return result
}

// latestCandleFromState returns any available recent candle to use as a
// fallback swing anchor when confirmed structural swings are empty.
func latestCandleFromState(state *features.MarketState) *types.Candle {
	if state == nil || len(state.Candles) == 0 {
		return nil
	}
	var best *types.Candle
	for _, c := range state.Candles {
		if c == nil {
			continue
		}
		if best == nil {
			best = c
			continue
		}
		// Prefer the candle with the larger range for more meaningful anchors.
		bestRange := best.High.Sub(best.Low)
		cRange := c.High.Sub(c.Low)
		if cRange.GreaterThan(bestRange) {
			best = c
		}
	}
	return best
}
