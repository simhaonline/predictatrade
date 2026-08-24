package features

import (
	"fmt"
	"time"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// RegimeEngineVersion is the version of the regime classification logic.
const RegimeEngineVersion = "2.0.0"

// RegimeTransition records a single regime state change.
type RegimeTransition struct {
	From      types.Regime `json:"from"`
	To        types.Regime `json:"to"`
	Timestamp time.Time    `json:"timestamp"`
	Reason    string       `json:"reason"`
	// Market snapshot at transition time
	RSI         float64 `json:"rsi"`
	ADX         float64 `json:"adx"`
	ATR         float64 `json:"atr"`
	EMAAlignment string `json:"ema_alignment"`
}

// RegimeEngine classifies market regime with hysteresis and confidence decay.
// SOW Section 11: Market regime detection
//
// Phase 2 improvements:
//   - Hysteresis: minimum hold period prevents regime flickering
//   - Confidence decay: confidence decays when triggering conditions no longer hold
//   - Transition tracking: records all regime transitions with market snapshots
//   - Age tracking: tracks how long the current regime has been active
//   - Entry reason: records why the current regime was entered
//   - RSI=0 guard: uninitialized RSI no longer triggers MEAN_REVERSION
//   - Confirmation candles: new regime must be confirmed by N candles before transition
type RegimeEngine struct {
	// Current state
	currentRegime  types.Regime
	enteredAt      time.Time
	entryReason    string
	confidence     float64

	// Hysteresis configuration
	minHoldDuration  time.Duration // Minimum time before allowing transition
	confirmationCandles int        // Number of candles confirming a candidate
	maxTransitions   int           // Max transitions to keep in history

	// Candidate tracking (for confirmation before transition)
	candidateRegime  types.Regime
	candidateReason   string
	candidateCount    int
	candidateConfidence float64

	// Transition history
	transitions []RegimeTransition

	// Previous candle time (for age calculation)
	lastCandleTime time.Time

	// Confidence decay parameters
	// When the triggering condition disappears, confidence decays at this rate per candle
	confidenceDecayRate float64 // Per-candle exponential decay factor
	minConfidence       float64 // Floor for confidence before forced transition
}

// NewRegimeEngine creates a regime engine with production hysteresis parameters.
// Parameters are derived from SOW Section 11 and historical validation:
//   - minHoldDuration: 5 minutes (prevents flickering on M1 without over-committing)
//   - confirmationCandles: 3 (requires 3 consecutive candles confirming new regime)
//   - confidenceDecayRate: 0.92 (8% decay per candle when conditions no longer hold)
//   - minConfidence: 0.25 (below this, forced re-evaluation ignores hysteresis)
func NewRegimeEngine() *RegimeEngine {
	return &RegimeEngine{
		currentRegime:       types.RegimeRange,
		enteredAt:           time.Time{}, // Will be set on first Process call
		entryReason:         "INITIALIZATION",
		confidence:          0.5,
		minHoldDuration:     5 * time.Minute,
		confirmationCandles: 3,
		maxTransitions:      100,
		confidenceDecayRate: 0.92,
		minConfidence:       0.25,
	}
}

// NewRegimeEngineWithConfig creates a regime engine with custom hysteresis parameters.
// Used for replay/backtesting with different timeframe configurations.
func NewRegimeEngineWithConfig(minHold time.Duration, confirmCandles int, decayRate, minConf float64) *RegimeEngine {
	return &RegimeEngine{
		currentRegime:       types.RegimeRange,
		enteredAt:           time.Time{},
		entryReason:         "INITIALIZATION",
		confidence:          0.5,
		minHoldDuration:     minHold,
		confirmationCandles: confirmCandles,
		maxTransitions:      100,
		confidenceDecayRate: decayRate,
		minConfidence:       minConf,
	}
}

// Process classifies the market regime for a new candle.
// Returns RegimeFeatures with full diagnostic information.
func (e *RegimeEngine) Process(candle *types.Candle, ind IndicatorFeatures) RegimeFeatures {
	// Initialize on first call — accept raw regime directly without hysteresis
	isFirstCandle := e.enteredAt.IsZero() && candle != nil
	if isFirstCandle {
		e.enteredAt = candle.Time
		e.lastCandleTime = candle.Time
	}

	feat := RegimeFeatures{
		Current:             e.currentRegime,
		Confidence:          e.confidence,
		RegimeEngineVersion: RegimeEngineVersion,
	}

	if candle == nil {
		feat.Previous = e.currentRegime
		return feat
	}

	// Compute raw regime from current indicators
	rawRegime, rawConfidence, rawReason := e.classifyRaw(candle, ind)

	feat.RawRegime = rawRegime
	feat.Previous = e.currentRegime

	// Calculate age of current regime
	if !e.enteredAt.IsZero() {
		feat.EnteredAt = e.enteredAt
		feat.Age = candle.Time.Sub(e.enteredAt)
	}

	// On first candle, accept raw regime directly — no hysteresis
	if isFirstCandle {
		e.currentRegime = rawRegime
		e.entryReason = "INIT:" + rawReason
		e.confidence = rawConfidence
		feat.Current = rawRegime
		feat.Confidence = rawConfidence
		feat.EnteredAt = e.enteredAt
		feat.Age = 0
		feat.EntryReason = e.entryReason
		feat.HoldReason = "INITIALIZATION"

		// Set volatility
		if !ind.ATR.IsZero() && !candle.Close.IsZero() {
			atrPct := ind.ATR.Div(candle.Close)
			if atrPct.GreaterThan(decimal.NewFromFloat(0.002)) {
				feat.Volatility = "HIGH"
			} else {
				feat.Volatility = "NORMAL"
			}
		} else {
			feat.Volatility = "NORMAL"
		}

		e.lastCandleTime = candle.Time
		return feat
	}

	// Determine if we should transition
	shouldTransition, transitionReason, holdReason := e.evaluateTransition(rawRegime, rawConfidence, rawReason, candle.Time)

	if shouldTransition {
		// Record transition
		rsiF, _ := ind.RSI.Float64()
		adxR, _ := ind.ADX.Float64()
		atrR, _ := ind.ATR.Float64()
		emaAlign := "NEUTRAL"
		if ind.EMA9.GreaterThan(ind.EMA21) && ind.EMA21.GreaterThan(ind.EMA50) {
			emaAlign = "BULLISH"
		} else if ind.EMA9.LessThan(ind.EMA21) && ind.EMA21.LessThan(ind.EMA50) {
			emaAlign = "BEARISH"
		}

		transition := RegimeTransition{
			From:        e.currentRegime,
			To:          rawRegime,
			Timestamp:   candle.Time,
			Reason:      transitionReason,
			RSI:         rsiF,
			ADX:         adxR,
			ATR:         atrR,
			EMAAlignment: emaAlign,
		}
		e.recordTransition(transition)

		e.currentRegime = rawRegime
		e.enteredAt = candle.Time
		e.entryReason = transitionReason
		e.confidence = rawConfidence
		e.candidateRegime = types.RegimeRange
		e.candidateCount = 0
		e.candidateReason = ""
		e.candidateConfidence = 0
	} else {
		// Not transitioning — update confidence
		if rawRegime == e.currentRegime {
			// Conditions still match — restore/maintain confidence
			if rawConfidence > e.confidence {
				e.confidence = rawConfidence
			}
		} else {
			// Conditions no longer match — apply confidence decay
			e.confidence *= e.confidenceDecayRate
			if e.confidence < e.minConfidence {
				e.confidence = e.minConfidence
			}
		}

		// Track candidate for confirmation
		if rawRegime != e.currentRegime {
			if e.candidateRegime != rawRegime {
				e.candidateRegime = rawRegime
				e.candidateCount = 1
				e.candidateReason = rawReason
				e.candidateConfidence = rawConfidence
			} else {
				e.candidateCount++
			}
		} else {
			// Reset candidate when raw matches current
			e.candidateRegime = types.RegimeRange
			e.candidateCount = 0
		}
	}

	// Update features
	feat.Current = e.currentRegime
	feat.Confidence = e.confidence
	feat.EnteredAt = e.enteredAt
	feat.EntryReason = e.entryReason
	if !e.enteredAt.IsZero() {
		feat.Age = candle.Time.Sub(e.enteredAt)
	}
	feat.HoldReason = holdReason

	// Set transition candidate
	if e.candidateRegime != e.currentRegime && e.candidateRegime != types.RegimeRange {
		candidate := e.candidateRegime
		feat.TransitionCandidate = &candidate
		feat.TransitionConfidence = e.candidateConfidence
		feat.ConfirmationCount = e.candidateCount
		feat.RequiredConfirmations = e.confirmationCandles
	}

	// Volatility classification
	if !ind.ATR.IsZero() && !candle.Close.IsZero() {
		atrPct := ind.ATR.Div(candle.Close)
		if atrPct.GreaterThan(decimal.NewFromFloat(0.002)) {
			feat.Volatility = "HIGH"
		} else {
			feat.Volatility = "NORMAL"
		}
	} else {
		feat.Volatility = "NORMAL"
	}

	e.lastCandleTime = candle.Time
	return feat
}

// classifyRaw determines the regime from current indicator values without hysteresis.
// This is the "raw" classification — what the indicators say right now.
func (e *RegimeEngine) classifyRaw(candle *types.Candle, ind IndicatorFeatures) (types.Regime, float64, string) {
	// Determine regime using indicators
	bullish := ind.EMA9.GreaterThan(ind.EMA21) && ind.EMA21.GreaterThan(ind.EMA50)
	bearish := ind.EMA9.LessThan(ind.EMA21) && ind.EMA21.LessThan(ind.EMA50)

	// ADX for trend strength
	trending := ind.ADX.GreaterThan(decimal.NewFromFloat(25.0))
	ranging := ind.ADX.LessThan(decimal.NewFromFloat(20.0))

	// RSI extremes — CRITICAL FIX: guard against RSI=0 (uninitialized)
	// A zero RSI should NOT trigger oversold/MEAN_REVERSION
	rsiValid := ind.RSI.GreaterThan(decimal.Zero)
	overbought := rsiValid && ind.RSI.GreaterThan(decimal.NewFromInt(70))
	oversold := rsiValid && ind.RSI.LessThan(decimal.NewFromInt(30))

	// ATR-based volatility
	highVol := false
	if !ind.ATR.IsZero() && !candle.Close.IsZero() {
		atrPct := ind.ATR.Div(candle.Close)
		highVol = atrPct.GreaterThan(decimal.NewFromFloat(0.002)) // >0.2%
	}

	var regime types.Regime
	var confidence float64
	var reason string

	switch {
	// PRIORITY 1: Strong trend (ADX>25 + EMA alignment) — even if RSI is extreme,
	// a trending market with aligned EMAs is TRENDING, not mean-reverting.
	// RSI>70 in a trend = strong momentum, not a reversal signal.
	case trending && bullish:
		regime = types.RegimeTrendingBullish
		confidence = 0.8
		reason = "ADX>25_BULLISH_EMA_ALIGNMENT"
	case trending && bearish:
		regime = types.RegimeTrendingBearish
		confidence = 0.8
		reason = "ADX>25_BEARISH_EMA_ALIGNMENT"
	// PRIORITY 2: Mean reversion — only when NOT trending (ADX<25) and RSI is extreme.
	// This prevents RSI>70 in a strong uptrend from being misclassified as mean reversion.
	case overbought && !trending:
		regime = types.RegimeMeanReversion
		confidence = 0.7
		reason = "RSI_OVERBOUGHT_NO_TREND"
	case oversold && !trending:
		regime = types.RegimeMeanReversion
		confidence = 0.7
		reason = "RSI_OVERSOLD_NO_TREND"
	// PRIORITY 3: Range — ADX<20, no clear trend
	case ranging:
		regime = types.RegimeRange
		confidence = 0.6
		reason = "ADX<20"
	// PRIORITY 4: High volatility — ATR percentage elevated
	case highVol:
		regime = types.RegimeHighVolatility
		confidence = 0.5
		reason = "ATR_PCT>0.2%"
	// PRIORITY 5: Bullish/bearish EMA alignment without strong ADX — still directional
	case bullish:
		regime = types.RegimeTrendingBullish
		confidence = 0.55
		reason = "BULLISH_EMA_ALIGNMENT_LOW_ADX"
	case bearish:
		regime = types.RegimeTrendingBearish
		confidence = 0.55
		reason = "BEARISH_EMA_ALIGNMENT_LOW_ADX"
	default:
		regime = types.RegimeRange
		confidence = 0.4
		reason = "DEFAULT_NO_STRONG_SIGNAL"
	}

	return regime, confidence, reason
}

// evaluateTransition decides whether to transition to a new regime.
// Returns: shouldTransition, transitionReason, holdReason
func (e *RegimeEngine) evaluateTransition(rawRegime types.Regime, rawConfidence float64, rawReason string, now time.Time) (bool, string, string) {
	// Same regime — no transition needed
	if rawRegime == e.currentRegime {
		return false, "", "SAME_REGIME"
	}

	// Calculate age of current regime
	age := now.Sub(e.enteredAt)

	// Force transition if confidence has decayed below minimum
	if e.confidence <= e.minConfidence {
		return true, "CONFIDENCE_DECAYED_BELOW_MIN:" + rawReason, "confidence_below_min"
	}

	// Check minimum hold duration
	if age < e.minHoldDuration {
		return false, "", fmt.Sprintf("MIN_HOLD_NOT_MET: age=%v < min=%v", age, e.minHoldDuration)
	}

	// Check confirmation candles
	if e.candidateRegime != rawRegime {
		// New candidate — start counting
		return false, "", "NEW_CANDIDATE_DETECTED"
	}

	if e.candidateCount < e.confirmationCandles {
		return false, "", fmt.Sprintf("CONFIRMATION_PENDING: %d/%d", e.candidateCount, e.confirmationCandles)
	}

	// All conditions met — transition
	return true, "CONFIRMED_TRANSITION:" + rawReason, "transition_confirmed"
}

// recordTransition adds a transition to history, respecting max size.
func (e *RegimeEngine) recordTransition(t RegimeTransition) {
	e.transitions = append(e.transitions, t)
	if len(e.transitions) > e.maxTransitions {
		e.transitions = e.transitions[len(e.transitions)-e.maxTransitions:]
	}
}

// GetTransitions returns the transition history.
func (e *RegimeEngine) GetTransitions() []RegimeTransition {
	return e.transitions
}

// GetCurrentRegime returns the current regime without processing.
func (e *RegimeEngine) GetCurrentRegime() types.Regime {
	return e.currentRegime
}

// GetEnteredAt returns when the current regime was entered.
func (e *RegimeEngine) GetEnteredAt() time.Time {
	return e.enteredAt
}

// GetEntryReason returns the reason the current regime was entered.
func (e *RegimeEngine) GetEntryReason() string {
	return e.entryReason
}

// Reset clears all state (for replay/backtesting).
func (e *RegimeEngine) Reset() {
	e.currentRegime = types.RegimeRange
	e.enteredAt = time.Time{}
	e.entryReason = "RESET"
	e.confidence = 0.5
	e.candidateRegime = types.RegimeRange
	e.candidateCount = 0
	e.candidateReason = ""
	e.candidateConfidence = 0
	e.transitions = nil
	e.lastCandleTime = time.Time{}
}
