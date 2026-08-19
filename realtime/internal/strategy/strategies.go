// Package strategy implements four DISTINCT strategy products.
// SOW Sections 12A-12F: Each strategy must have genuinely different logic.
// Do NOT implement all four as the same strategy with different labels.
package strategy

import (
	"fmt"
	"time"

	"strings"

	"math"

	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// Strategy is the interface for all strategy products.
type Strategy interface {
	ID() types.StrategyID
	Evaluate(state *features.MarketState) StrategyResult
}

// StrategyResult is the output of a strategy evaluation.
type StrategyResult struct {
	StrategyID    types.StrategyID
	Direction     types.Direction
	RawScore      decimal.Decimal
	LongScore     decimal.Decimal
	ShortScore    decimal.Decimal
	Evidence      []types.EvidenceContribution
	EntryPrice    decimal.Decimal
	StopLoss      decimal.Decimal
	TP1           decimal.Decimal
	TP2           decimal.Decimal
	TP3           decimal.Decimal
	ReasonCodes   []types.NoTradeReason
	HumanReason   string
	ConflictPenalty decimal.Decimal
	ExpiryMinutes  int
	CooldownMinutes int

	// Transition analysis (prompt.md Sections 6, 54)
	TransitionLongScore   decimal.Decimal
	TransitionShortScore  decimal.Decimal
	TransitionConflict     decimal.Decimal
	TransitionFinalScore  decimal.Decimal
	TransitionCandidateThreshold float64
	IsTransitionCandidate  bool

	// Dominance (prompt.md Section 23)
	Dominance   float64
}

// StrategyConfig defines strategy-specific configuration.
// SOW: Configuration should be externalized, not scattered magic constants.
type StrategyConfig struct {
	MinConfluence     float64
	MinMTFAlignment   float64
	ATRMultiplierSL   float64
	ATRMultiplierTP1  float64
	ATRMultiplierTP2  float64
	ATRMultiplierTP3  float64
	MaxSpreadPips     float64
	MinADX            float64
	MinRR             float64
	ExpiryMinutes     int
	CooldownMinutes   int
	DecisionTFs       []types.Timeframe
	ContextTFs        []types.Timeframe
	AcceptedRegimes   []types.Regime
	AcceptedSessions  []string
	MinQualityState   types.QualityState
}

// ─── Common helpers ───

func addEvidence(evidence *[]types.EvidenceContribution, pillar, feature string, dir types.Direction,
	weight, contrib float64, quality types.QualityState, reason string) {
	*evidence = append(*evidence, types.EvidenceContribution{
		Pillar: pillar, Feature: feature, Direction: dir,
		Weight: decimal.NewFromFloat(weight),
		Contribution: decimal.NewFromFloat(contrib),
		NormalizedValue: decimal.NewFromFloat(contrib), // CRITICAL: confluence engine uses this field
		Quality: quality, Source: pillar + "_engine", Version: "1.0",
		ReasonCode: reason,
	})
}

// applyFamilyCaps limits total contribution per evidence family to prevent
// double-counting of correlated indicators.
func applyFamilyCaps(evidence []types.EvidenceContribution) []types.EvidenceContribution {
	familyMax := map[string]float64{
		"TREND":      0.25,
		"MOMENTUM":   0.20,
		"VOLATILITY": 0.10,
		"VWAP":       0.10,
		"STRUCTURE":  0.20,
		"LIQUIDITY":  0.15,
		"SMC":        0.15,
		"MTF":        0.15,
		"CANDLE":     0.15,
		"REGIME":     0.10,
	}
	familySums := map[string]float64{}
	for _, e := range evidence {
		c, _ := e.Contribution.Float64()
		familySums[e.Pillar] += c
	}
	result := make([]types.EvidenceContribution, len(evidence))
	for i, e := range evidence {
		result[i] = e
		if max, ok := familyMax[e.Pillar]; ok {
			sum := familySums[e.Pillar]
			if sum > max {
				scale := max / sum
				scaled := result[i].Contribution.Mul(decimal.NewFromFloat(scale))
				result[i].Contribution = scaled
				result[i].NormalizedValue = scaled
			}
		}
	}
	return result
}

func computeEntrySLTP(state *features.MarketState, direction types.Direction, cfg StrategyConfig) (entry, sl, tp1, tp2, tp3 decimal.Decimal) {
	if state == nil || state.Indicators.ATR.IsZero() {
		return decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero
	}
	entry = state.CurrentPrice
	atr := state.Indicators.ATR
	if direction == types.DirectionBuy {
		sl = entry.Sub(atr.Mul(decimal.NewFromFloat(cfg.ATRMultiplierSL)))
		tp1 = entry.Add(atr.Mul(decimal.NewFromFloat(cfg.ATRMultiplierTP1)))
		tp2 = entry.Add(atr.Mul(decimal.NewFromFloat(cfg.ATRMultiplierTP2)))
		tp3 = entry.Add(atr.Mul(decimal.NewFromFloat(cfg.ATRMultiplierTP3)))
	} else {
		sl = entry.Add(atr.Mul(decimal.NewFromFloat(cfg.ATRMultiplierSL)))
		tp1 = entry.Sub(atr.Mul(decimal.NewFromFloat(cfg.ATRMultiplierTP1)))
		tp2 = entry.Sub(atr.Mul(decimal.NewFromFloat(cfg.ATRMultiplierTP2)))
		tp3 = entry.Sub(atr.Mul(decimal.NewFromFloat(cfg.ATRMultiplierTP3)))
	}
	return
}


// PHI applies the signal threshold function.
// PHI(x) = 0 if x < 0.65 (suppressed)
// PHI(x) = 0.65 + 0.35*(x-0.65)/0.35 if 0.65 <= x <= 1.0
// This prevents single-family dominance — requires MOST families positive.
func phi(x float64) float64 {
	if x < 0.65 {
		return 0
	}
	if x >= 1.0 {
		return 1.0
	}
	return 0.65 + 0.35*(x-0.65)/0.35
}

// computeStructuralSLTP calculates SL using structural low + ATR buffer + spread adjustment.
// SL_Long = Low_structural - lambda_SL * ATR - 0.5 * Spread
// TP_Long = Entry + max(R_min * (Entry - SL), 1.5 * ATR)
func computeStructuralSLTP(state *features.MarketState, direction types.Direction, cfg StrategyConfig, structuralLow, structuralHigh decimal.Decimal) (entry, sl, tp1, tp2, tp3 decimal.Decimal) {
	if state == nil || state.Indicators.ATR.IsZero() {
		return decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero
	}
	entry = state.CurrentPrice
	atr := state.Indicators.ATR
	spread := state.Spread
	halfSpread := spread.Div(decimal.NewFromInt(2))
	
	if direction == types.DirectionBuy {
		// SL = structural low - lambda*ATR - 0.5*spread
		slBase := structuralLow
		if slBase.IsZero() {
			slBase = entry.Sub(atr.Mul(decimal.NewFromFloat(cfg.ATRMultiplierSL)))
		} else {
			slBase = slBase.Sub(atr.Mul(decimal.NewFromFloat(cfg.ATRMultiplierSL)))
		}
		sl = slBase.Sub(halfSpread)
		// TP = Entry + max(R_min * (Entry-SL), 1.5*ATR)
		riskDist := entry.Sub(sl).Abs()
		minTP := atr.Mul(decimal.NewFromFloat(1.5))
		tp1Dist := riskDist.Mul(decimal.NewFromFloat(cfg.MinRR))
		if tp1Dist.LessThan(minTP) {
			tp1Dist = minTP
		}
		tp1 = entry.Add(tp1Dist)
		tp2 = entry.Add(tp1Dist.Mul(decimal.NewFromFloat(1.5)))
		tp3 = entry.Add(tp1Dist.Mul(decimal.NewFromFloat(2.0)))
	} else {
		slBase := structuralHigh
		if slBase.IsZero() {
			slBase = entry.Add(atr.Mul(decimal.NewFromFloat(cfg.ATRMultiplierSL)))
		} else {
			slBase = slBase.Add(atr.Mul(decimal.NewFromFloat(cfg.ATRMultiplierSL)))
		}
		sl = slBase.Add(halfSpread)
		riskDist := sl.Sub(entry).Abs()
		minTP := atr.Mul(decimal.NewFromFloat(1.5))
		tp1Dist := riskDist.Mul(decimal.NewFromFloat(cfg.MinRR))
		if tp1Dist.LessThan(minTP) {
			tp1Dist = minTP
		}
		tp1 = entry.Sub(tp1Dist)
		tp2 = entry.Sub(tp1Dist.Mul(decimal.NewFromFloat(1.5)))
		tp3 = entry.Sub(tp1Dist.Mul(decimal.NewFromFloat(2.0)))
	}
	return
}

// getStructuralLow returns the most recent swing low for SL calculation.
func getStructuralLow(state *features.MarketState) decimal.Decimal {
	if state == nil || len(state.Structure.SwingLows) == 0 {
		return decimal.Zero
	}
	return state.Structure.SwingLows[len(state.Structure.SwingLows)-1]
}

// getStructuralHigh returns the most recent swing high for SL calculation.
func getStructuralHigh(state *features.MarketState) decimal.Decimal {
	if state == nil || len(state.Structure.SwingHighs) == 0 {
		return decimal.Zero
	}
	return state.Structure.SwingHighs[len(state.Structure.SwingHighs)-1]
}

// computeFinalScore applies PHI threshold to the raw family-weighted score.
func computeFinalScore(familyScores map[string]float64, familyWeights map[string]float64) float64 {
	rawTotal := 0.0
	for family, score := range familyScores {
		weight := familyWeights[family]
		rawTotal += weight * score
	}
	return phi(rawTotal)
}


// checkConflict detects contradictory evidence and returns penalty.
func checkConflict(state *features.MarketState, direction types.Direction, decisionTF string) (decimal.Decimal, string) {
	if state == nil {
		return decimal.Zero, ""
	}
	penalty := decimal.Zero
	var conflictDesc string

	// M1 bullish but H1/H4 bearish (or vice versa)
	mtf := state.MTF.States
	if direction == types.DirectionBuy {
		if h1, ok := mtf[types.TFH1]; ok && h1 < 0 {
			penalty = penalty.Add(decimal.NewFromFloat(15))
			conflictDesc += "M1 BUY but H1 bearish; "
		}
		if h4, ok := mtf[types.TFH4]; ok && h4 < 0 {
			penalty = penalty.Add(decimal.NewFromFloat(15))
			conflictDesc += "H4 bearish; "
		}
	} else if direction == types.DirectionSell {
		if h1, ok := mtf[types.TFH1]; ok && h1 > 0 {
			penalty = penalty.Add(decimal.NewFromFloat(15))
			conflictDesc += "M1 SELL but H1 bullish; "
		}
		if h4, ok := mtf[types.TFH4]; ok && h4 > 0 {
			penalty = penalty.Add(decimal.NewFromFloat(15))
			conflictDesc += "H4 bullish; "
		}
	}

	// Trend signal while regime is RANGE
	if state.Regime.Current == types.RegimeRange {
		penalty = penalty.Add(decimal.NewFromFloat(10))
		conflictDesc += "regime is RANGE (no trend); "
	}

	// Reversal without CHoCH/MSS
	if state.Structure.LastCHoCH == nil && state.Structure.LastMSS == nil {
		if (direction == types.DirectionBuy && state.Structure.CurrentTrend == "bearish") ||
			(direction == types.DirectionSell && state.Structure.CurrentTrend == "bullish") {
			penalty = penalty.Add(decimal.NewFromFloat(20))
			conflictDesc += "countertrend without CHoCH/MSS; "
		}
	}

	// Scalping with high spread
	spread, _ := state.Spread.Float64()
	atr, _ := state.Indicators.ATR.Float64()
	if atr > 0 && spread/atr > 0.5 {
		penalty = penalty.Add(decimal.NewFromFloat(10))
		conflictDesc += "spread too wide relative to ATR; "
	}

	return penalty, conflictDesc
}

// generateHumanReason creates a deterministic technical explanation from evidence.
func generateHumanReason(direction string, state *features.MarketState, evidence []types.EvidenceContribution, conflict string) string {
	if state == nil {
		return "No market data"
	}
	reason := fmt.Sprintf("%s — ", direction)
	if state.Structure.CurrentTrend != "" {
		reason += fmt.Sprintf("structure: %s, ", state.Structure.CurrentTrend)
	}
	if state.Structure.LastBOS != nil {
		reason += fmt.Sprintf("BOS %s, ", state.Structure.LastBOS.Direction)
	}
	if state.Candle.IsDisplacement {
		reason += "displacement candle, "
	}
	if state.Candle.IsRejection {
		reason += "rejection wick, "
	}
	adx, _ := state.Indicators.ADX.Float64()
	rsi, _ := state.Indicators.RSI.Float64()
	reason += fmt.Sprintf("ADX=%.1f RSI=%.1f, ", adx, rsi)
	reason += fmt.Sprintf("regime: %s, ", state.Regime.Current)
	reason += fmt.Sprintf("session: %s", state.Session.CurrentSession)
	if conflict != "" {
		reason += fmt.Sprintf(" [CONFLICT: %s]", conflict)
	}
	return reason
}

// scoreDirection computes long/short scores from evidence.
func scoreDirection(evidence []types.EvidenceContribution, minConfluence float64, conflictPenalty decimal.Decimal) (direction types.Direction, rawScore, longScore, shortScore decimal.Decimal, reasons []types.NoTradeReason) {
	for _, e := range evidence {
		if e.Direction == types.DirectionBuy {
			longScore = longScore.Add(e.Contribution)
		} else if e.Direction == types.DirectionSell {
			shortScore = shortScore.Add(e.Contribution)
		}
	}
	// Scale to 0-100
	longScore = longScore.Mul(decimal.NewFromInt(100))
	shortScore = shortScore.Mul(decimal.NewFromInt(100))

	// Apply conflict penalty to dominant side
	if longScore.GreaterThan(shortScore) {
		longScore = longScore.Sub(conflictPenalty)
	} else if shortScore.GreaterThan(longScore) {
		shortScore = shortScore.Sub(conflictPenalty)
	}

	if longScore.GreaterThan(shortScore) {
		rawScore = longScore
		if longScore.GreaterThan(decimal.NewFromFloat(minConfluence)) {
			direction = types.DirectionBuy
		} else {
			direction = types.DirectionNoTrade
			reasons = append(reasons, types.NTInsufficientScore)
		}
	} else {
		rawScore = shortScore
		if shortScore.GreaterThan(decimal.NewFromFloat(minConfluence)) {
			direction = types.DirectionSell
		} else {
			direction = types.DirectionNoTrade
			reasons = append(reasons, types.NTInsufficientScore)
		}
	}
	return
}

// scoreDirectionWithThresholds computes long/short scores using regime-specific thresholds.
// candidateThreshold < tradeThreshold.
// score >= tradeThreshold → BUY/SELL (qualified for gate evaluation)
// candidateThreshold <= score < tradeThreshold → BUY/SELL (directional, main.go classifies as candidate)
// score < candidateThreshold → NO-TRADE
//
// IMPORTANT: This function returns BUY/SELL for any directional result so that
// geometry is ALWAYS computed. The main.go pipeline classifies candidate vs
// executable using the same thresholds.
func scoreDirectionWithThresholds(evidence []types.EvidenceContribution, candidateThreshold, tradeThreshold float64, conflictPenalty decimal.Decimal) (direction types.Direction, rawScore, longScore, shortScore decimal.Decimal, reasons []types.NoTradeReason) {
	for _, e := range evidence {
		if e.Direction == types.DirectionBuy {
			longScore = longScore.Add(e.Contribution)
		} else if e.Direction == types.DirectionSell {
			shortScore = shortScore.Add(e.Contribution)
		}
	}
	// Scale to 0-100
	longScore = longScore.Mul(decimal.NewFromInt(100))
	shortScore = shortScore.Mul(decimal.NewFromInt(100))

	// Apply conflict penalty to dominant side
	if longScore.GreaterThan(shortScore) {
		longScore = longScore.Sub(conflictPenalty)
	} else if shortScore.GreaterThan(longScore) {
		shortScore = shortScore.Sub(conflictPenalty)
	}

	// Directional dominance check (prompt.md Section 23)
	// Prevent BUY when longScore ≈ shortScore (conflicting direction)
	dominance := longScore.Sub(shortScore).Abs()
	minDominance := decimal.NewFromFloat(MinDominanceMargin)

	if longScore.GreaterThan(shortScore) {
		rawScore = longScore
		if longScore.GreaterThan(decimal.NewFromFloat(candidateThreshold)) {
			if dominance.LessThan(minDominance) {
				// Scores too close — conflicting direction
				direction = types.DirectionNoTrade
				reasons = append(reasons, types.NTConflictingDirection)
			} else {
				direction = types.DirectionBuy
			}
		} else {
			direction = types.DirectionNoTrade
			reasons = append(reasons, types.NTInsufficientScore)
		}
	} else {
		rawScore = shortScore
		if shortScore.GreaterThan(decimal.NewFromFloat(candidateThreshold)) {
			if dominance.LessThan(minDominance) {
				direction = types.DirectionNoTrade
				reasons = append(reasons, types.NTConflictingDirection)
			} else {
				direction = types.DirectionSell
			}
		} else {
			direction = types.DirectionNoTrade
			reasons = append(reasons, types.NTInsufficientScore)
		}
	}
	return
}

// getRegimeThresholds retrieves the candidate and trade thresholds for a strategy+regime.
// Falls back to the strategy's MinConfluence as trade threshold if no regime-specific config.
func getRegimeThresholds(strategyID types.StrategyID, regime types.Regime, fallbackTrade float64) (candidate, trade float64) {
	ct, tt, found := GetThresholds(strategyID, regime)
	if found {
		return ct, tt
	}
	// Fallback: candidate = trade * 0.6, trade = MinConfluence
	return fallbackTrade * 0.6, fallbackTrade
}

// checkRegimeSession validates regime and session suitability.
func checkRegimeSession(state *features.MarketState, cfg StrategyConfig) []types.NoTradeReason {
	var reasons []types.NoTradeReason
	regimeOK := len(cfg.AcceptedRegimes) == 0
	for _, r := range cfg.AcceptedRegimes {
		if state.Regime.Current == r {
			regimeOK = true
			break
		}
	}
	if !regimeOK {
		reasons = append(reasons, types.NTRegimeMismatch)
	}
	sessionOK := len(cfg.AcceptedSessions) == 0
	for _, s := range cfg.AcceptedSessions {
		if state.Session.CurrentSession == s || state.Session.IsOverlap {
			sessionOK = true
			break
		}
	}
	if !sessionOK {
		reasons = append(reasons, types.NTSessionUnsuitable)
	}
	if state.Session.NewsRisk == "HIGH" || state.Session.NewsRisk == "BLOCKED" {
		reasons = append(reasons, types.NTHighNewsRisk)
	}
	return reasons
}

// checkMTF validates multi-timeframe alignment.
func checkMTF(state *features.MarketState, direction types.Direction, minAlignment float64) []types.NoTradeReason {
	if direction == types.DirectionNoTrade {
		return nil
	}
	mtfScore := state.MTF.Score
	if direction == types.DirectionBuy && mtfScore < minAlignment {
		return []types.NoTradeReason{types.NTConflictingTimeframes}
	}
	if direction == types.DirectionSell && mtfScore > -minAlignment {
		return []types.NoTradeReason{types.NTConflictingTimeframes}
	}
	return nil
}

// ─── STANDARD_SCALPING ───
// M1/M5 scalping with M15/M30 context confirmation.
// Focus: EMA alignment, VWAP, short-term structure, candle rejection/displacement, MACD/OsMA momentum.
// Tighter SL, moderate frequency, higher quality than Ultra.

type StandardScalping struct{ cfg StrategyConfig }

func NewStandardScalping() *StandardScalping {
	return &StandardScalping{cfg: StrategyConfig{
		MinConfluence: 65, MinMTFAlignment: 40,
		ATRMultiplierSL: 1.0, ATRMultiplierTP1: 1.0, ATRMultiplierTP2: 1.5, ATRMultiplierTP3: 2.0,
		MaxSpreadPips: 2.0, MinADX: 20, MinRR: 1.2,
		ExpiryMinutes: 10, CooldownMinutes: 15,
		DecisionTFs: []types.Timeframe{types.TFM1, types.TFM5},
		ContextTFs: []types.Timeframe{types.TFM15, types.TFM30},
		AcceptedRegimes: []types.Regime{types.RegimeTrendingBullish, types.RegimeTrendingBearish, types.RegimeBreakout, types.RegimeMeanReversion, types.RegimeRange},
		AcceptedSessions: []string{"LONDON", "NEW_YORK", "OVERLAP", "TOKYO"},
		MinQualityState: types.QualityAuthoritative,
	}}
}
func (s *StandardScalping) ID() types.StrategyID { return types.StrategyStandardScalping }

func (s *StandardScalping) Evaluate(state *features.MarketState) StrategyResult {
	result := StrategyResult{StrategyID: s.ID(), Direction: types.DirectionNoTrade, ExpiryMinutes: s.cfg.ExpiryMinutes, CooldownMinutes: s.cfg.CooldownMinutes}
	if state == nil || state.Indicators.ATR.IsZero() {
		result.Direction = types.DirectionError
		result.ReasonCodes = append(result.ReasonCodes, types.NTATRNotReady)
		return result
	}

	// Regime + session checks
	result.ReasonCodes = append(result.ReasonCodes, checkRegimeSession(state, s.cfg)...)
	if len(result.ReasonCodes) > 0 {
		return result
	}

	var evidence []types.EvidenceContribution
	q := state.Quality

	// EMA 9/21 alignment — primary trend filter for scalping
	if state.Indicators.EMA9.GreaterThan(state.Indicators.EMA21) {
		addEvidence(&evidence, "TREND", "EMA9_ABOVE_EMA21", types.DirectionBuy, 15, 0.12, q, "")
	} else {
		addEvidence(&evidence, "TREND", "EMA9_BELOW_EMA21", types.DirectionSell, 15, 0.12, q, "")
	}

	// VWAP relationship
	if !state.VWAP.SessionVWAP.IsZero() {
		if state.CurrentPrice.GreaterThan(state.VWAP.SessionVWAP) {
			addEvidence(&evidence, "VWAP", "ABOVE_VWAP", types.DirectionBuy, 12, 0.08, q, "")
		} else {
			addEvidence(&evidence, "VWAP", "BELOW_VWAP", types.DirectionSell, 12, 0.08, q, "")
		}
	}

	// Short-term structure (BOS/CHoCH)
	if state.Structure.LastBOS != nil {
		dir := types.DirectionBuy
		if state.Structure.LastBOS.Direction == "bearish" {
			dir = types.DirectionSell
		}
		addEvidence(&evidence, "STRUCTURE", "BOS", dir, 18, 0.14, q, "")
	}

	// Candle evidence — displacement and rejection are key for scalping
	if state.Candle.IsDisplacement && state.Candle.IsBullish {
		addEvidence(&evidence, "CANDLE", "BULLISH_DISPLACEMENT", types.DirectionBuy, 15, 0.10, q, "")
	}
	if state.Candle.IsDisplacement && state.Candle.IsBearish {
		addEvidence(&evidence, "CANDLE", "BEARISH_DISPLACEMENT", types.DirectionSell, 15, 0.10, q, "")
	}
	if state.Candle.IsRejection && state.Candle.IsBullish {
		addEvidence(&evidence, "CANDLE", "BULLISH_REJECTION", types.DirectionBuy, 12, 0.08, q, "")
	}
	if state.Candle.IsRejection && state.Candle.IsBearish {
		addEvidence(&evidence, "CANDLE", "BEARISH_REJECTION", types.DirectionSell, 12, 0.08, q, "")
	}

	// MACD/OsMA momentum
	if state.Indicators.MACDMain.GreaterThan(state.Indicators.MACDSignal) {
		addEvidence(&evidence, "MOMENTUM", "MACD_BULLISH", types.DirectionBuy, 10, 0.06, q, "")
	} else {
		addEvidence(&evidence, "MOMENTUM", "MACD_BEARISH", types.DirectionSell, 10, 0.06, q, "")
	}
	if state.Indicators.OsMA.GreaterThan(decimal.Zero) {
		addEvidence(&evidence, "MOMENTUM", "OSMA_POSITIVE", types.DirectionBuy, 8, 0.05, q, "")
	} else {
		addEvidence(&evidence, "MOMENTUM", "OSMA_NEGATIVE", types.DirectionSell, 8, 0.05, q, "")
	}

	// RSI confirmation (not overbought/oversold extreme for scalping — mid-range trend)
	rsi, _ := state.Indicators.RSI.Float64()
	if rsi > 50 && rsi < 70 {
		addEvidence(&evidence, "MOMENTUM", "RSI_BULLISH_MID", types.DirectionBuy, 8, 0.05, q, "")
	} else if rsi < 50 && rsi > 30 {
		addEvidence(&evidence, "MOMENTUM", "RSI_BEARISH_MID", types.DirectionSell, 8, 0.05, q, "")
	}

	// ADX trend strength
	adx, _ := state.Indicators.ADX.Float64()
	if adx > s.cfg.MinADX {
		if state.Indicators.ADXPlusDI.GreaterThan(state.Indicators.ADXMinusDI) {
			addEvidence(&evidence, "TREND", "ADX_BULLISH", types.DirectionBuy, 10, 0.07, q, "")
		} else {
			addEvidence(&evidence, "TREND", "ADX_BEARISH", types.DirectionSell, 10, 0.07, q, "")
		}
	}

	// Liquidity sweep evidence
	if len(state.Liquidity.RecentSweeps) > 0 {
		sweep := state.Liquidity.RecentSweeps[len(state.Liquidity.RecentSweeps)-1]
		if sweepIsSellSide(sweep.Direction) {
			addEvidence(&evidence, "LIQUIDITY", "SELL_SIDE_SWEEP", types.DirectionBuy, 12, 0.08, q, "")
		} else if sweepIsBuySide(sweep.Direction) {
			addEvidence(&evidence, "LIQUIDITY", "BUY_SIDE_SWEEP", types.DirectionSell, 12, 0.08, q, "")
		}
	}

	// MTF alignment
	mtfScore := state.MTF.Score
	if mtfScore > 0 {
		addEvidence(&evidence, "MTF", "ALIGNMENT_BULLISH", types.DirectionBuy, 10, float64(mtfScore)/100.0*0.05, q, "")
	} else if mtfScore < 0 {
		addEvidence(&evidence, "MTF", "ALIGNMENT_BEARISH", types.DirectionSell, 10, float64(-mtfScore)/100.0*0.05, q, "")
	}

	// Regime-adaptive evidence: add RANGE-specific evidence when in range/mean-reversion
	if isRegimeInRange(state.Regime.Current) {
		computeRangeEvidence(&evidence, state, DefaultRangeEvidenceConfig())
	}

	result.Evidence = applyFamilyCaps(evidence)

	// Use regime-specific thresholds for candidate/trade classification
	candidateThresh, tradeThresh := getRegimeThresholds(s.ID(), state.Regime.Current, s.cfg.MinConfluence)
	dir, raw, long, short, reasons := scoreDirectionWithThresholds(result.Evidence, candidateThresh, tradeThresh, decimal.Zero)
	result.Direction = dir
	result.RawScore = raw
	result.LongScore = long
	result.ShortScore = short
	result.ReasonCodes = append(result.ReasonCodes, reasons...)

	// Conflict detection
	if dir != types.DirectionNoTrade {
		penalty, conflictDesc := checkConflict(state, dir, "M5")
		result.ConflictPenalty = penalty
		if penalty.GreaterThan(decimal.NewFromFloat(20)) {
			result.Direction = types.DirectionWait
			result.ReasonCodes = append(result.ReasonCodes, types.NTConflictingTimeframes)
		}
		result.HumanReason = generateHumanReason(string(result.Direction), state, result.Evidence, conflictDesc)
	}

	// MTF check
	result.ReasonCodes = append(result.ReasonCodes, checkMTF(state, result.Direction, s.cfg.MinMTFAlignment)...)

	// Compute geometry for ANY directional result (candidate or qualified) using canonical BuildTradeGeometry
	if result.Direction != types.DirectionNoTrade && result.Direction != types.DirectionWait {
		geo := BuildTradeGeometry(state, result.Direction, s.cfg)
		if geo.Valid {
			result.EntryPrice = geo.Entry
			result.StopLoss = geo.StopLoss
			result.TP1 = geo.TP1
			result.TP2 = geo.TP2
			result.TP3 = geo.TP3
		} else if !geo.Entry.IsZero() {
			result.ReasonCodes = append(result.ReasonCodes, types.NoTradeReason(geo.ReasonCode))
		}
	}

	return result
}

// ─── ULTRA_SCALPING ───
// Fastest strategy. M1 with M5 confirmation. Extremely latency/cost-sensitive.
// Focus: immediate momentum, micro-structure, tick movement, spread, VWAP, EMA9/21.
// Shorter expiry, stricter spread/cost gates, very tight SL.

type UltraScalping struct{ cfg StrategyConfig }

func NewUltraScalping() *UltraScalping {
	return &UltraScalping{cfg: StrategyConfig{
		MinConfluence: 65, MinMTFAlignment: 50,
		ATRMultiplierSL: 0.5, ATRMultiplierTP1: 0.5, ATRMultiplierTP2: 0.75, ATRMultiplierTP3: 1.0,
		MaxSpreadPips: 1.5, MinADX: 25, MinRR: 1.0,
		ExpiryMinutes: 3, CooldownMinutes: 5,
		DecisionTFs: []types.Timeframe{types.TFM1},
		ContextTFs: []types.Timeframe{types.TFM5, types.TFM15},
		AcceptedRegimes: []types.Regime{types.RegimeTrendingBullish, types.RegimeTrendingBearish, types.RegimeBreakout, types.RegimeMeanReversion, types.RegimeRange},
		AcceptedSessions: []string{"LONDON", "NEW_YORK", "OVERLAP", "TOKYO"},
		MinQualityState: types.QualityAuthoritative,
	}}
}
func (s *UltraScalping) ID() types.StrategyID { return types.StrategyUltraScalping }

func (s *UltraScalping) Evaluate(state *features.MarketState) StrategyResult {
	result := StrategyResult{StrategyID: s.ID(), Direction: types.DirectionNoTrade, ExpiryMinutes: s.cfg.ExpiryMinutes, CooldownMinutes: s.cfg.CooldownMinutes}
	if state == nil || state.Indicators.ATR.IsZero() {
		result.Direction = types.DirectionError
		result.ReasonCodes = append(result.ReasonCodes, types.NTATRNotReady)
		return result
	}

	// Regime + session checks — stricter than Standard Scalping
	result.ReasonCodes = append(result.ReasonCodes, checkRegimeSession(state, s.cfg)...)
	if len(result.ReasonCodes) > 0 {
		return result
	}

	// Spread check — Ultra Scalping is extremely cost-sensitive
	spread, _ := state.Spread.Float64()
	atr, _ := state.Indicators.ATR.Float64()
	if atr > 0 && spread/atr > 0.4 {
		result.ReasonCodes = append(result.ReasonCodes, types.NTHighSpread)
		result.HumanReason = fmt.Sprintf("NO-TRADE — spread (%.3f) too wide relative to ATR (%.3f) for Ultra Scalping", spread, atr)
		return result
	}

	// Triple EMA hierarchy: required for TREND/BREAKOUT, relaxed for RANGE/MEAN_REVERSION
	// In RANGE, EMAs are often tangled — requiring strict hierarchy would make
	// range trading impossible. Instead, use EMA9 vs EMA21 as minimal direction.
	emaHierarchyOK := false
	if isRegimeInRange(state.Regime.Current) {
		// For RANGE: only require EMA9 != EMA21 (some directional bias exists)
		emaHierarchyOK = !state.Indicators.EMA9.Equals(state.Indicators.EMA21)
	} else {
		// For TREND/BREAKOUT: require full hierarchy EMA9 > EMA21 > EMA50 or inverse
		if state.Indicators.EMA9.GreaterThan(state.Indicators.EMA21) && state.Indicators.EMA21.GreaterThan(state.Indicators.EMA50) {
			emaHierarchyOK = true // Bullish hierarchy
		} else if state.Indicators.EMA9.LessThan(state.Indicators.EMA21) && state.Indicators.EMA21.LessThan(state.Indicators.EMA50) {
			emaHierarchyOK = true // Bearish hierarchy
		}
	}
	if !emaHierarchyOK {
		result.ReasonCodes = append(result.ReasonCodes, types.NTUnclearStructure)
		if isRegimeInRange(state.Regime.Current) {
			result.HumanReason = "NO-TRADE — EMA9/EMA21 flat (no directional bias in range)"
		} else {
			result.HumanReason = "NO-TRADE — EMA hierarchy broken (requires EMA9>EMA21>EMA50 or inverse)"
		}
		return result
	}

	var evidence []types.EvidenceContribution
	q := state.Quality

	// EMA 9/21 — immediate momentum direction
	if state.Indicators.EMA9.GreaterThan(state.Indicators.EMA21) {
		addEvidence(&evidence, "TREND", "EMA9_ABOVE_EMA21", types.DirectionBuy, 20, 0.15, q, "")
	} else {
		addEvidence(&evidence, "TREND", "EMA9_BELOW_EMA21", types.DirectionSell, 20, 0.15, q, "")
	}

	// M1 candle displacement — critical for Ultra (fast momentum)
	if state.Candle.IsDisplacement && state.Candle.IsBullish {
		addEvidence(&evidence, "CANDLE", "M1_BULLISH_DISPLACEMENT", types.DirectionBuy, 20, 0.15, q, "")
	}
	if state.Candle.IsDisplacement && state.Candle.IsBearish {
		addEvidence(&evidence, "CANDLE", "M1_BEARISH_DISPLACEMENT", types.DirectionSell, 20, 0.15, q, "")
	}

	// VWAP proximity — key for ultra scalping
	if !state.VWAP.SessionVWAP.IsZero() {
		if state.CurrentPrice.GreaterThan(state.VWAP.SessionVWAP) {
			addEvidence(&evidence, "VWAP", "ABOVE_VWAP", types.DirectionBuy, 15, 0.10, q, "")
		} else {
			addEvidence(&evidence, "VWAP", "BELOW_VWAP", types.DirectionSell, 15, 0.10, q, "")
		}
	}

	// Liquidity sweep + reclaim — very important for ultra
	if len(state.Liquidity.RecentSweeps) > 0 {
		sweep := state.Liquidity.RecentSweeps[len(state.Liquidity.RecentSweeps)-1]
		if sweepIsSellSide(sweep.Direction) {
			addEvidence(&evidence, "LIQUIDITY", "SELL_SIDE_SWEEP_RECLAIM", types.DirectionBuy, 18, 0.12, q, "")
		} else if sweepIsBuySide(sweep.Direction) {
			addEvidence(&evidence, "LIQUIDITY", "BUY_SIDE_SWEEP_RECLAIM", types.DirectionSell, 18, 0.12, q, "")
		}
	}

	// OsMA — fast momentum oscillator
	if state.Indicators.OsMA.GreaterThan(decimal.Zero) {
		addEvidence(&evidence, "MOMENTUM", "OSMA_POSITIVE", types.DirectionBuy, 12, 0.08, q, "")
	} else {
		addEvidence(&evidence, "MOMENTUM", "OSMA_NEGATIVE", types.DirectionSell, 12, 0.08, q, "")
	}

	// Stochastic — fast cycle confirmation
	if state.Indicators.StochMain.GreaterThan(state.Indicators.StochSignal) {
		addEvidence(&evidence, "MOMENTUM", "STOCH_BULLISH_CROSS", types.DirectionBuy, 10, 0.06, q, "")
	} else {
		addEvidence(&evidence, "MOMENTUM", "STOCH_BEARISH_CROSS", types.DirectionSell, 10, 0.06, q, "")
	}

	// MTF alignment — requires strong alignment for Ultra
	mtfScore := state.MTF.Score
	if mtfScore > 0 {
		addEvidence(&evidence, "MTF", "ALIGNMENT_BULLISH", types.DirectionBuy, 15, float64(mtfScore)/100.0*0.08, q, "")
	} else if mtfScore < 0 {
		addEvidence(&evidence, "MTF", "ALIGNMENT_BEARISH", types.DirectionSell, 15, float64(-mtfScore)/100.0*0.08, q, "")
	}

	// ADX — needs strong trend for Ultra
	adx, _ := state.Indicators.ADX.Float64()
	if adx > s.cfg.MinADX {
		if state.Indicators.ADXPlusDI.GreaterThan(state.Indicators.ADXMinusDI) {
			addEvidence(&evidence, "TREND", "ADX_STRONG_BULLISH", types.DirectionBuy, 12, 0.08, q, "")
		} else {
			addEvidence(&evidence, "TREND", "ADX_STRONG_BEARISH", types.DirectionSell, 12, 0.08, q, "")
		}
	}

	// Bollinger — mean reversion context for ultra
	if !state.Indicators.BollUpper.IsZero() {
		if state.CurrentPrice.LessThanOrEqual(state.Indicators.BollLower) {
			addEvidence(&evidence, "VOLATILITY", "BOLL_LOWER_TOUCH", types.DirectionBuy, 8, 0.05, q, "")
		} else if state.CurrentPrice.GreaterThanOrEqual(state.Indicators.BollUpper) {
			addEvidence(&evidence, "VOLATILITY", "BOLL_UPPER_TOUCH", types.DirectionSell, 8, 0.05, q, "")
		}
	}

	// Regime-adaptive evidence: Ultra-specific RANGE/microstructure evidence
	if isRegimeInRange(state.Regime.Current) {
		computeUltraRangeEvidence(&evidence, state, DefaultUltraRangeConfig())
	}

	result.Evidence = applyFamilyCaps(evidence)

	// Use regime-specific thresholds for candidate/trade classification
	candidateThresh, tradeThresh := getRegimeThresholds(s.ID(), state.Regime.Current, s.cfg.MinConfluence)
	dir, raw, long, short, reasons := scoreDirectionWithThresholds(result.Evidence, candidateThresh, tradeThresh, decimal.Zero)
	result.Direction = dir
	result.RawScore = raw
	result.LongScore = long
	result.ShortScore = short
	result.ReasonCodes = append(result.ReasonCodes, reasons...)

	// Conflict detection — stricter for Ultra
	if dir != types.DirectionNoTrade {
		penalty, conflictDesc := checkConflict(state, dir, "M1")
		result.ConflictPenalty = penalty
		if penalty.GreaterThan(decimal.NewFromFloat(15)) {
			result.Direction = types.DirectionWait
			result.ReasonCodes = append(result.ReasonCodes, types.NTConflictingTimeframes)
		}
		result.HumanReason = generateHumanReason(string(result.Direction), state, evidence, conflictDesc)
	}

	result.ReasonCodes = append(result.ReasonCodes, checkMTF(state, result.Direction, s.cfg.MinMTFAlignment)...)

	// Compute geometry for ANY directional result (candidate or qualified) using canonical BuildTradeGeometry
	if result.Direction != types.DirectionNoTrade && result.Direction != types.DirectionWait {
		geo := BuildTradeGeometry(state, result.Direction, s.cfg)
		if geo.Valid {
			result.EntryPrice = geo.Entry
			result.StopLoss = geo.StopLoss
			result.TP1 = geo.TP1
			result.TP2 = geo.TP2
			result.TP3 = geo.TP3
		} else if !geo.Entry.IsZero() {
			// Geometry invalid but entry exists — record reason
			result.ReasonCodes = append(result.ReasonCodes, types.NoTradeReason(geo.ReasonCode))
		}
	}

	return result
}

// ─── STANDARD_SWING ───
// M15/M30/H1 with H4/D1 confirmation. Tolerates intraday noise.
// Focus: EMA21/50, SMA200, H1/H4 structure, order blocks, FVG, ADX, MACD, MTF alignment.
// Wider structurally valid stops, larger targets.

type StandardSwing struct{ cfg StrategyConfig }

func NewStandardSwing() *StandardSwing {
	return &StandardSwing{cfg: StrategyConfig{
		MinConfluence: 55, MinMTFAlignment: 30,
		ATRMultiplierSL: 1.5, ATRMultiplierTP1: 1.5, ATRMultiplierTP2: 2.5, ATRMultiplierTP3: 3.5,
		MaxSpreadPips: 3.0, MinADX: 18, MinRR: 1.8,
		ExpiryMinutes: 60, CooldownMinutes: 120,
		DecisionTFs: []types.Timeframe{types.TFM15, types.TFM30, types.TFH1},
		ContextTFs: []types.Timeframe{types.TFH4, types.TFD1},
		AcceptedRegimes: []types.Regime{types.RegimeTrendingBullish, types.RegimeTrendingBearish, types.RegimeBreakout, types.RegimeMeanReversion, types.RegimeRange},
		AcceptedSessions: []string{"LONDON", "NEW_YORK", "OVERLAP", "TOKYO"},
		MinQualityState: types.QualityAuthoritative,
	}}
}
func (s *StandardSwing) ID() types.StrategyID { return types.StrategyStandardSwing }

func (s *StandardSwing) Evaluate(state *features.MarketState) StrategyResult {
	result := StrategyResult{StrategyID: s.ID(), Direction: types.DirectionNoTrade, ExpiryMinutes: s.cfg.ExpiryMinutes, CooldownMinutes: s.cfg.CooldownMinutes}
	if state == nil || state.Indicators.ATR.IsZero() {
		result.Direction = types.DirectionError
		result.ReasonCodes = append(result.ReasonCodes, types.NTATRNotReady)
		return result
	}

	result.ReasonCodes = append(result.ReasonCodes, checkRegimeSession(state, s.cfg)...)
	if len(result.ReasonCodes) > 0 {
		return result
	}

	var evidence []types.EvidenceContribution
	q := state.Quality

	// EMA 21/50 alignment — primary trend for swing
	if state.Indicators.EMA21.GreaterThan(state.Indicators.EMA50) {
		addEvidence(&evidence, "TREND", "EMA21_ABOVE_EMA50", types.DirectionBuy, 18, 0.14, q, "")
	} else {
		addEvidence(&evidence, "TREND", "EMA21_BELOW_EMA50", types.DirectionSell, 18, 0.14, q, "")
	}

	// SMA 200 — major trend filter
	if !state.Indicators.SMA200.IsZero() {
		if state.CurrentPrice.GreaterThan(state.Indicators.SMA200) {
			addEvidence(&evidence, "TREND", "ABOVE_SMA200", types.DirectionBuy, 15, 0.10, q, "")
		} else {
			addEvidence(&evidence, "TREND", "BELOW_SMA200", types.DirectionSell, 15, 0.10, q, "")
		}
	}

	// H1/H4 market structure — BOS and CHoCH
	if state.Structure.LastBOS != nil {
		dir := types.DirectionBuy
		if state.Structure.LastBOS.Direction == "bearish" {
			dir = types.DirectionSell
		}
		addEvidence(&evidence, "STRUCTURE", "HTF_BOS", dir, 20, 0.15, q, "")
	}
	if state.Structure.LastCHoCH != nil {
		dir := types.DirectionBuy
		if state.Structure.LastCHoCH.Direction == "bearish" {
			dir = types.DirectionSell
		}
		addEvidence(&evidence, "STRUCTURE", "HTF_CHoCH", dir, 15, 0.10, q, "")
	}

	// Order block proximity
	if len(state.FVG.OrderBlocks) > 0 {
		ob := state.FVG.OrderBlocks[len(state.FVG.OrderBlocks)-1]
		if ob.Type == "BULLISH" && !ob.Mitigated {
			addEvidence(&evidence, "SMC", "BULLISH_OB", types.DirectionBuy, 12, 0.08, q, "")
		} else if ob.Type == "BEARISH" && !ob.Mitigated {
			addEvidence(&evidence, "SMC", "BEARISH_OB", types.DirectionSell, 12, 0.08, q, "")
		}
	}

	// FVG proximity
	if len(state.FVG.FVGs) > 0 {
		fvg := state.FVG.FVGs[len(state.FVG.FVGs)-1]
		if fvg.Type == "BULLISH" && !fvg.Filled {
			addEvidence(&evidence, "SMC", "BULLISH_FVG", types.DirectionBuy, 10, 0.06, q, "")
		} else if fvg.Type == "BEARISH" && !fvg.Filled {
			addEvidence(&evidence, "SMC", "BEARISH_FVG", types.DirectionSell, 10, 0.06, q, "")
		}
	}

	// ADX — trend strength
	adx, _ := state.Indicators.ADX.Float64()
	if adx > s.cfg.MinADX {
		if state.Indicators.ADXPlusDI.GreaterThan(state.Indicators.ADXMinusDI) {
			addEvidence(&evidence, "TREND", "ADX_BULLISH", types.DirectionBuy, 12, 0.08, q, "")
		} else {
			addEvidence(&evidence, "TREND", "ADX_BEARISH", types.DirectionSell, 12, 0.08, q, "")
		}
	}

	// MACD
	if state.Indicators.MACDMain.GreaterThan(state.Indicators.MACDSignal) {
		addEvidence(&evidence, "MOMENTUM", "MACD_BULLISH", types.DirectionBuy, 10, 0.06, q, "")
	} else {
		addEvidence(&evidence, "MOMENTUM", "MACD_BEARISH", types.DirectionSell, 10, 0.06, q, "")
	}

	// RSI
	rsi, _ := state.Indicators.RSI.Float64()
	if rsi > 50 {
		addEvidence(&evidence, "MOMENTUM", "RSI_ABOVE_50", types.DirectionBuy, 8, 0.05, q, "")
	} else {
		addEvidence(&evidence, "MOMENTUM", "RSI_BELOW_50", types.DirectionSell, 8, 0.05, q, "")
	}

	// Candle evidence — important closes for swing
	if state.Candle.IsBreakout && state.Candle.IsBullish {
		addEvidence(&evidence, "CANDLE", "BULLISH_BREAKOUT_CLOSE", types.DirectionBuy, 12, 0.08, q, "")
	}
	if state.Candle.IsBreakout && state.Candle.IsBearish {
		addEvidence(&evidence, "CANDLE", "BEARISH_BREAKOUT_CLOSE", types.DirectionSell, 12, 0.08, q, "")
	}

	// MTF alignment
	mtfScore := state.MTF.Score
	if mtfScore > 0 {
		addEvidence(&evidence, "MTF", "ALIGNMENT_BULLISH", types.DirectionBuy, 12, float64(mtfScore)/100.0*0.06, q, "")
	} else if mtfScore < 0 {
		addEvidence(&evidence, "MTF", "ALIGNMENT_BEARISH", types.DirectionSell, 12, float64(-mtfScore)/100.0*0.06, q, "")
	}

	// Liquidity
	if len(state.Liquidity.RecentSweeps) > 0 {
		sweep := state.Liquidity.RecentSweeps[len(state.Liquidity.RecentSweeps)-1]
		if sweepIsSellSide(sweep.Direction) {
			addEvidence(&evidence, "LIQUIDITY", "HTF_SELL_SIDE_SWEEP", types.DirectionBuy, 10, 0.06, q, "")
		} else if sweepIsBuySide(sweep.Direction) {
			addEvidence(&evidence, "LIQUIDITY", "HTF_BUY_SIDE_SWEEP", types.DirectionSell, 10, 0.06, q, "")
		}
	}

	// Regime
	if state.Regime.Current == types.RegimeTrendingBullish {
		addEvidence(&evidence, "REGIME", "TRENDING_BULLISH", types.DirectionBuy, 10, 0.07, q, "")
	} else if state.Regime.Current == types.RegimeTrendingBearish {
		addEvidence(&evidence, "REGIME", "TRENDING_BEARISH", types.DirectionSell, 10, 0.07, q, "")
	}

	// Regime-adaptive evidence: add RANGE-specific evidence when in range/mean-reversion
	if isRegimeInRange(state.Regime.Current) {
		computeRangeEvidence(&evidence, state, DefaultRangeEvidenceConfig())
	}

	result.Evidence = applyFamilyCaps(evidence)

	// Use regime-specific thresholds for candidate/trade classification
	candidateThresh, tradeThresh := getRegimeThresholds(s.ID(), state.Regime.Current, s.cfg.MinConfluence)
	dir, raw, long, short, reasons := scoreDirectionWithThresholds(result.Evidence, candidateThresh, tradeThresh, decimal.Zero)
	result.Direction = dir
	result.RawScore = raw
	result.LongScore = long
	result.ShortScore = short
	result.ReasonCodes = append(result.ReasonCodes, reasons...)

	if dir != types.DirectionNoTrade {
		penalty, conflictDesc := checkConflict(state, dir, "H1")
		result.ConflictPenalty = penalty
		if penalty.GreaterThan(decimal.NewFromFloat(25)) {
			result.Direction = types.DirectionWait
			result.ReasonCodes = append(result.ReasonCodes, types.NTConflictingTimeframes)
		}
		result.HumanReason = generateHumanReason(string(result.Direction), state, evidence, conflictDesc)
	}

	result.ReasonCodes = append(result.ReasonCodes, checkMTF(state, result.Direction, s.cfg.MinMTFAlignment)...)

	// Compute geometry for ANY directional result (candidate or qualified) using canonical BuildTradeGeometry
	if result.Direction != types.DirectionNoTrade && result.Direction != types.DirectionWait {
		geo := BuildTradeGeometry(state, result.Direction, s.cfg)
		if geo.Valid {
			result.EntryPrice = geo.Entry
			result.StopLoss = geo.StopLoss
			result.TP1 = geo.TP1
			result.TP2 = geo.TP2
			result.TP3 = geo.TP3
		} else if !geo.Entry.IsZero() {
			result.ReasonCodes = append(result.ReasonCodes, types.NoTradeReason(geo.ReasonCode))
		}
	}

	return result
}

// ─── TREND_SWING ───
// H1/H4/D1 for major continuation moves. M15/M30 for execution refinement.
// Focus: SMA200, EMA50, EMA21, ADX/+DI/-DI, confirmed HH/HL or LH/LL, BOS, pullbacks, continuation OB/FVG.
// Fewer signals, wider stops, larger targets, longer lifecycle.

type TrendSwing struct{ cfg StrategyConfig }

func NewTrendSwing() *TrendSwing {
	return &TrendSwing{cfg: StrategyConfig{
		MinConfluence: 50, MinMTFAlignment: 25,
		ATRMultiplierSL: 2.0, ATRMultiplierTP1: 2.0, ATRMultiplierTP2: 4.0, ATRMultiplierTP3: 6.0,
		MaxSpreadPips: 4.0, MinADX: 22, MinRR: 2.5,
		ExpiryMinutes: 240, CooldownMinutes: 360,
		DecisionTFs: []types.Timeframe{types.TFH1, types.TFH4},
		ContextTFs: []types.Timeframe{types.TFD1, types.TFW1},
		AcceptedRegimes: []types.Regime{types.RegimeTrendingBullish, types.RegimeTrendingBearish, types.RegimeBreakout},
		AcceptedSessions: []string{"LONDON", "NEW_YORK", "OVERLAP", "TOKYO", "SYDNEY"},
		MinQualityState: types.QualityAuthoritative,
	}}
}
func (s *TrendSwing) ID() types.StrategyID { return types.StrategyTrendSwing }

func (s *TrendSwing) Evaluate(state *features.MarketState) StrategyResult {
	result := StrategyResult{StrategyID: s.ID(), Direction: types.DirectionNoTrade, ExpiryMinutes: s.cfg.ExpiryMinutes, CooldownMinutes: s.cfg.CooldownMinutes}
	if state == nil || state.Indicators.ATR.IsZero() {
		result.Direction = types.DirectionError
		result.ReasonCodes = append(result.ReasonCodes, types.NTATRNotReady)
		return result
	}

	// prompt.md Sections 3-6: During RANGE, compute transition evidence BEFORE regime check.
	// TrendSwing's AcceptedRegimes excludes RANGE, but we must still analyze transitions.
	// Score=0 must mean "no transition evidence", NOT "strategy skipped calculation".
	if isRegimeInRange(state.Regime.Current) {
		// ALWAYS compute transition evidence — even if zero
		transitionEvidence := computeTrendTransitionEvidence(state)
		result.Evidence = applyFamilyCaps(transitionEvidence)

		// Transition candidate threshold is separate from final TradeThreshold (prompt.md Section 7)
		transitionCandidateThreshold := 20.0
		transitionTradeThreshold := 35.0

		transDir, transRaw, transLong, transShort, transReasons := scoreDirectionWithThresholds(
			result.Evidence, transitionCandidateThreshold, transitionTradeThreshold, decimal.Zero)

		// ALWAYS expose transition scores (prompt.md Section 6)
		result.TransitionLongScore = transLong
		result.TransitionShortScore = transShort
		result.TransitionConflict = transLong.Sub(transShort).Abs()
		result.TransitionFinalScore = transRaw
		result.TransitionCandidateThreshold = transitionCandidateThreshold
		result.IsTransitionCandidate = true

		// Compute dominance
		longF, _ := transLong.Float64()
		shortF, _ := transShort.Float64()
		result.Dominance = math.Abs(longF - shortF)

		if transDir != types.DirectionNoTrade {
			// Transition candidate detected — non-executable (prompt.md Sections 7-8)
			result.Direction = transDir
			result.RawScore = transRaw
			result.LongScore = transLong
			result.ShortScore = transShort
			result.ReasonCodes = append(result.ReasonCodes, transReasons...)
			result.HumanReason = fmt.Sprintf("TREND_TRANSITION — regime %s, transition score %.1f, long=%.1f short=%.1f",
				state.Regime.Current, func() float64 { f, _ := transRaw.Float64(); return f }(),
				longF, shortF)

			// Compute geometry for transition candidate (prompt.md Section 34)
			geo := BuildTradeGeometry(state, transDir, s.cfg)
			if geo.Valid {
				result.EntryPrice = geo.Entry
				result.StopLoss = geo.StopLoss
				result.TP1 = geo.TP1
				result.TP2 = geo.TP2
				result.TP3 = geo.TP3
			}
		} else {
			// Genuine NO-TRADE: no transition evidence above candidate threshold (prompt.md Section 10)
			result.Direction = types.DirectionNoTrade
			result.RawScore = transRaw
			result.LongScore = transLong
			result.ShortScore = transShort
			result.ReasonCodes = append(result.ReasonCodes, types.NTNoTrendTransition)
			result.HumanReason = fmt.Sprintf("NO-TRADE — regime %s, no trend transition evidence (long=%.1f short=%.1f)",
				state.Regime.Current, longF, shortF)
		}
		return result
	}

	// For non-RANGE regimes, do the normal regime/session check
	result.ReasonCodes = append(result.ReasonCodes, checkRegimeSession(state, s.cfg)...)
	if len(result.ReasonCodes) > 0 {
		return result
	}

	// Trend Swing requires a trending regime — no range trading
	if state.Regime.Current != types.RegimeTrendingBullish && state.Regime.Current != types.RegimeTrendingBearish && state.Regime.Current != types.RegimeBreakout {
		// Non-range, non-trending regime (e.g. HIGH_VOLATILITY, NEWS_EVENT)
		result.ReasonCodes = append(result.ReasonCodes, types.NTRegimeMismatch)
		result.HumanReason = fmt.Sprintf("NO-TRADE — regime %s not suitable for Trend Swing", state.Regime.Current)
		return result
	}
	// Mandatory macro gate: EMA100 > EMA200 on trend TF (bullish) or EMA100 < EMA200 (bearish)
	// No Trend Swing against the macro trend
	if !state.Indicators.EMA100.IsZero() && !state.Indicators.EMA200.IsZero() {
		macroBullish := state.Indicators.EMA100.GreaterThan(state.Indicators.EMA200)
		macroBearish := state.Indicators.EMA100.LessThan(state.Indicators.EMA200)
		if !macroBullish && !macroBearish {
			result.ReasonCodes = append(result.ReasonCodes, types.NTUnclearStructure)
			result.HumanReason = "NO-TRADE — EMA100/EMA200 macro trend unclear"
			return result
		}
	}

	var evidence []types.EvidenceContribution
	q := state.Quality

	// SMA 200 — major trend direction (highest weight for trend swing)
	if !state.Indicators.SMA200.IsZero() {
		if state.CurrentPrice.GreaterThan(state.Indicators.SMA200) {
			addEvidence(&evidence, "TREND", "ABOVE_SMA200", types.DirectionBuy, 25, 0.18, q, "")
		} else {
			addEvidence(&evidence, "TREND", "BELOW_SMA200", types.DirectionSell, 25, 0.18, q, "")
		}
	}

	// EMA 50 — intermediate trend
	if !state.Indicators.EMA50.IsZero() {
		if state.CurrentPrice.GreaterThan(state.Indicators.EMA50) {
			addEvidence(&evidence, "TREND", "ABOVE_EMA50", types.DirectionBuy, 15, 0.10, q, "")
		} else {
			addEvidence(&evidence, "TREND", "BELOW_EMA50", types.DirectionSell, 15, 0.10, q, "")
		}
	}

	// EMA 21 — short-term trend within larger trend
	if state.Indicators.EMA21.GreaterThan(state.Indicators.EMA50) {
		addEvidence(&evidence, "TREND", "EMA21_ABOVE_EMA50", types.DirectionBuy, 12, 0.08, q, "")
	} else {
		addEvidence(&evidence, "TREND", "EMA21_BELOW_EMA50", types.DirectionSell, 12, 0.08, q, "")
	}

	// ADX with +DI/-DI — must have strong trend
	adx, _ := state.Indicators.ADX.Float64()
	if adx > s.cfg.MinADX {
		if state.Indicators.ADXPlusDI.GreaterThan(state.Indicators.ADXMinusDI) {
			addEvidence(&evidence, "TREND", "ADX_STRONG_BULLISH", types.DirectionBuy, 18, 0.12, q, "")
		} else {
			addEvidence(&evidence, "TREND", "ADX_STRONG_BEARISH", types.DirectionSell, 18, 0.12, q, "")
		}
	} else {
		// Weak ADX = no trend = no trend swing
		result.ReasonCodes = append(result.ReasonCodes, types.NTInsufficientScore)
		result.HumanReason = fmt.Sprintf("NO-TRADE — ADX %.1f below minimum %.1f for Trend Swing", adx, s.cfg.MinADX)
		return result
	}

	// Confirmed HH/HL or LH/LL structure
	if state.Structure.CurrentTrend == "bullish" {
		addEvidence(&evidence, "STRUCTURE", "HH_HL_SEQUENCE", types.DirectionBuy, 18, 0.12, q, "")
	} else if state.Structure.CurrentTrend == "bearish" {
		addEvidence(&evidence, "STRUCTURE", "LH_LL_SEQUENCE", types.DirectionSell, 18, 0.12, q, "")
	}

	// BOS — confirmed break of structure
	if state.Structure.LastBOS != nil {
		dir := types.DirectionBuy
		if state.Structure.LastBOS.Direction == "bearish" {
			dir = types.DirectionSell
		}
		addEvidence(&evidence, "STRUCTURE", "BOS_CONFIRMED", dir, 15, 0.10, q, "")
	}

	// Continuation order blocks
	if len(state.FVG.OrderBlocks) > 0 {
		ob := state.FVG.OrderBlocks[len(state.FVG.OrderBlocks)-1]
		if ob.Type == "BULLISH" && !ob.Mitigated {
			addEvidence(&evidence, "SMC", "CONTINUATION_BULLISH_OB", types.DirectionBuy, 10, 0.06, q, "")
		} else if ob.Type == "BEARISH" && !ob.Mitigated {
			addEvidence(&evidence, "SMC", "CONTINUATION_BEARISH_OB", types.DirectionSell, 10, 0.06, q, "")
		}
	}

	// Continuation FVG
	if len(state.FVG.FVGs) > 0 {
		fvg := state.FVG.FVGs[len(state.FVG.FVGs)-1]
		if fvg.Type == "BULLISH" && !fvg.Filled {
			addEvidence(&evidence, "SMC", "CONTINUATION_BULLISH_FVG", types.DirectionBuy, 8, 0.05, q, "")
		} else if fvg.Type == "BEARISH" && !fvg.Filled {
			addEvidence(&evidence, "SMC", "CONTINUATION_BEARISH_FVG", types.DirectionSell, 8, 0.05, q, "")
		}
	}

	// MACD — trend continuation
	if state.Indicators.MACDMain.GreaterThan(state.Indicators.MACDSignal) {
		addEvidence(&evidence, "MOMENTUM", "MACD_BULLISH_CONTINUATION", types.DirectionBuy, 10, 0.06, q, "")
	} else {
		addEvidence(&evidence, "MOMENTUM", "MACD_BEARISH_CONTINUATION", types.DirectionSell, 10, 0.06, q, "")
	}

	// CCI — momentum confirmation
	if state.Indicators.CCI.GreaterThan(decimal.Zero) {
		addEvidence(&evidence, "MOMENTUM", "CCI_BULLISH", types.DirectionBuy, 8, 0.05, q, "")
	} else {
		addEvidence(&evidence, "MOMENTUM", "CCI_BEARISH", types.DirectionSell, 8, 0.05, q, "")
	}

	// Candle — pullback continuation
	if state.Candle.IsBullish && state.Candle.ConsecutiveBull >= 2 {
		addEvidence(&evidence, "CANDLE", "PULLBACK_BULLISH_CONTINUATION", types.DirectionBuy, 10, 0.06, q, "")
	}
	if state.Candle.IsBearish && state.Candle.ConsecutiveBear >= 2 {
		addEvidence(&evidence, "CANDLE", "PULLBACK_BEARISH_CONTINUATION", types.DirectionSell, 10, 0.06, q, "")
	}

	// MTF alignment — critical for trend swing
	mtfScore := state.MTF.Score
	if mtfScore > 0 {
		addEvidence(&evidence, "MTF", "ALIGNMENT_BULLISH", types.DirectionBuy, 15, float64(mtfScore)/100.0*0.07, q, "")
	} else if mtfScore < 0 {
		addEvidence(&evidence, "MTF", "ALIGNMENT_BEARISH", types.DirectionSell, 15, float64(-mtfScore)/100.0*0.07, q, "")
	}

	// VWAP context
	if !state.VWAP.SessionVWAP.IsZero() {
		if state.CurrentPrice.GreaterThan(state.VWAP.SessionVWAP) {
			addEvidence(&evidence, "VWAP", "ABOVE_VWAP", types.DirectionBuy, 8, 0.05, q, "")
		} else {
			addEvidence(&evidence, "VWAP", "BELOW_VWAP", types.DirectionSell, 8, 0.05, q, "")
		}
	}

	result.Evidence = applyFamilyCaps(evidence)

	candidateThresh, tradeThresh := getRegimeThresholds(s.ID(), state.Regime.Current, s.cfg.MinConfluence)
	dir, raw, long, short, reasons := scoreDirectionWithThresholds(result.Evidence, candidateThresh, tradeThresh, decimal.Zero)
	result.Direction = dir
	result.RawScore = raw
	result.LongScore = long
	result.ShortScore = short
	result.ReasonCodes = append(result.ReasonCodes, reasons...)

	if dir != types.DirectionNoTrade {
		penalty, conflictDesc := checkConflict(state, dir, "H4")
		result.ConflictPenalty = penalty
		// Trend Swing is more tolerant of short-term conflicts
		if penalty.GreaterThan(decimal.NewFromFloat(30)) {
			result.Direction = types.DirectionWait
			result.ReasonCodes = append(result.ReasonCodes, types.NTConflictingTimeframes)
		}
		result.HumanReason = generateHumanReason(string(result.Direction), state, evidence, conflictDesc)
	}

	result.ReasonCodes = append(result.ReasonCodes, checkMTF(state, result.Direction, s.cfg.MinMTFAlignment)...)

	// Compute geometry for ANY directional result using canonical BuildTradeGeometry
	if result.Direction != types.DirectionNoTrade && result.Direction != types.DirectionWait {
		geo := BuildTradeGeometry(state, result.Direction, s.cfg)
		if geo.Valid {
			result.EntryPrice = geo.Entry
			result.StopLoss = geo.StopLoss
			result.TP1 = geo.TP1
			result.TP2 = geo.TP2
			result.TP3 = geo.TP3
		} else if !geo.Entry.IsZero() {
			result.ReasonCodes = append(result.ReasonCodes, types.NoTradeReason(geo.ReasonCode))
		}
	}

	return result
}

// sweepIsSellSide checks if a sweep event is a sell-side liquidity sweep.
// Handles both lowercase ("sell_side") and uppercase ("SELL_SIDE_SWEEP") forms
// produced by the liquidity engine, ensuring wiring is case-robust.
func sweepIsSellSide(dir string) bool {
	return strings.EqualFold(dir, "sell_side") || strings.EqualFold(dir, "SELL_SIDE_SWEEP")
}

// sweepIsBuySide checks if a sweep event is a buy-side liquidity sweep.
func sweepIsBuySide(dir string) bool {
	return strings.EqualFold(dir, "buy_side") || strings.EqualFold(dir, "BUY_SIDE_SWEEP")
}

// AllStrategies returns instances of all four strategy products.
func AllStrategies() []Strategy {
	return []Strategy{
		NewStandardScalping(),
		NewUltraScalping(),
		NewStandardSwing(),
		NewTrendSwing(),
	}
}

// Unused import guard
var _ = time.Now
