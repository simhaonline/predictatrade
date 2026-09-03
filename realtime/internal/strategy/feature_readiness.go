// Package strategy — Feature readiness guards.
// SOW Phase 2 Section 8: Explicit feature readiness.
//
// Prevents zero-filled/uninitialized indicators from producing false evidence.
// A feature that is NOT READY must not contribute bullish or bearish evidence.
package strategy

import (
	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// FeatureReadiness tracks which features are ready for evidence computation.
type FeatureReadiness struct {
	EMA9      bool
	EMA21     bool
	EMA50     bool
	EMA100    bool
	EMA200    bool
	SMA200    bool
	RSI       bool
	ADX       bool
	ATR       bool
	MACD      bool
	Stoch     bool
	CCI       bool
	Bollinger bool
	VWAP      bool
	OBV       bool
	Structure bool
	MTF       bool
	Liquidity bool
	FVG       bool
	Candle    bool
}

// CheckFeatureReadiness evaluates which features have valid (non-zero) values.
func CheckFeatureReadiness(state *features.MarketState) FeatureReadiness {
	r := FeatureReadiness{}
	if state == nil {
		return r
	}
	ind := state.Indicators

	r.EMA9 = !ind.EMA9.IsZero()
	r.EMA21 = !ind.EMA21.IsZero()
	r.EMA50 = !ind.EMA50.IsZero()
	r.EMA100 = !ind.EMA100.IsZero()
	r.EMA200 = !ind.EMA200.IsZero()
	r.SMA200 = !ind.SMA200.IsZero()
	r.RSI = !ind.RSI.IsZero()
	r.ADX = !ind.ADX.IsZero()
	r.ATR = !ind.ATR.IsZero()
	r.MACD = !ind.MACDMain.IsZero() || !ind.MACDSignal.IsZero()
	r.Stoch = !ind.StochMain.IsZero()
	r.CCI = !ind.CCI.IsZero()
	r.Bollinger = !ind.BollUpper.IsZero() && !ind.BollLower.IsZero()
	r.VWAP = !state.VWAP.SessionVWAP.IsZero()
	r.OBV = !ind.OBV.IsZero()

	// Structure readiness
	r.Structure = len(state.Structure.SwingHighs) > 0 && len(state.Structure.SwingLows) > 0

	// MTF readiness — at least 2 timeframes have state
	r.MTF = len(state.MTF.States) >= 2

	// Liquidity readiness
	r.Liquidity = len(state.Liquidity.RecentSweeps) > 0

	// FVG readiness
	r.FVG = len(state.FVG.FVGs) > 0

	// Candle readiness
	r.Candle = !state.Candle.Range.IsZero()

	return r
}

// ReadinessPercent returns the percentage of ready features (0-100).
func (r FeatureReadiness) ReadinessPercent() float64 {
	total := 20
	ready := 0
	if r.EMA9 {
		ready++
	}
	if r.EMA21 {
		ready++
	}
	if r.EMA50 {
		ready++
	}
	if r.EMA100 {
		ready++
	}
	if r.EMA200 {
		ready++
	}
	if r.SMA200 {
		ready++
	}
	if r.RSI {
		ready++
	}
	if r.ADX {
		ready++
	}
	if r.ATR {
		ready++
	}
	if r.MACD {
		ready++
	}
	if r.Stoch {
		ready++
	}
	if r.CCI {
		ready++
	}
	if r.Bollinger {
		ready++
	}
	if r.VWAP {
		ready++
	}
	if r.OBV {
		ready++
	}
	if r.Structure {
		ready++
	}
	if r.MTF {
		ready++
	}
	if r.Liquidity {
		ready++
	}
	if r.FVG {
		ready++
	}
	if r.Candle {
		ready++
	}
	return float64(ready) / float64(total) * 100.0
}

// MissingFeatures returns a list of feature names that are not ready.
func (r FeatureReadiness) MissingFeatures() []string {
	var missing []string
	if !r.EMA9 {
		missing = append(missing, "EMA9")
	}
	if !r.EMA21 {
		missing = append(missing, "EMA21")
	}
	if !r.EMA50 {
		missing = append(missing, "EMA50")
	}
	if !r.EMA100 {
		missing = append(missing, "EMA100")
	}
	if !r.EMA200 {
		missing = append(missing, "EMA200")
	}
	if !r.SMA200 {
		missing = append(missing, "SMA200")
	}
	if !r.RSI {
		missing = append(missing, "RSI")
	}
	if !r.ADX {
		missing = append(missing, "ADX")
	}
	if !r.ATR {
		missing = append(missing, "ATR")
	}
	if !r.MACD {
		missing = append(missing, "MACD")
	}
	if !r.Stoch {
		missing = append(missing, "Stochastic")
	}
	if !r.CCI {
		missing = append(missing, "CCI")
	}
	if !r.Bollinger {
		missing = append(missing, "Bollinger")
	}
	if !r.VWAP {
		missing = append(missing, "VWAP")
	}
	if !r.OBV {
		missing = append(missing, "OBV")
	}
	if !r.Structure {
		missing = append(missing, "Structure")
	}
	if !r.MTF {
		missing = append(missing, "MTF")
	}
	if !r.Liquidity {
		missing = append(missing, "Liquidity")
	}
	if !r.FVG {
		missing = append(missing, "FVG")
	}
	if !r.Candle {
		missing = append(missing, "Candle")
	}
	return missing
}

// ScoreContributionTrace records how each feature contributed to the final score.
// This is used for diagnostics to explain why a score is zero.
type ScoreContributionTrace struct {
	StrategyID         types.StrategyID
	LongContributions  map[string]float64
	ShortContributions map[string]float64
	ConflictPenalty    float64
	FinalLongScore     float64
	FinalShortScore    float64
	Direction          types.Direction
	PrimaryReason      string
	MissingFeatures    []string
	ReadinessPercent   float64
}

// BuildContributionTrace creates a diagnostic trace from evidence and result.
func BuildContributionTrace(strategyID types.StrategyID, evidence []types.EvidenceContribution, result StrategyResult, readiness FeatureReadiness) ScoreContributionTrace {
	trace := ScoreContributionTrace{
		StrategyID:         strategyID,
		LongContributions:  make(map[string]float64),
		ShortContributions: make(map[string]float64),
		Direction:          result.Direction,
		ReadinessPercent:   readiness.ReadinessPercent(),
		MissingFeatures:    readiness.MissingFeatures(),
	}

	for _, e := range evidence {
		featName := e.Pillar + ":" + e.Feature
		contrib, _ := e.Contribution.Float64()
		if e.Direction == types.DirectionBuy {
			trace.LongContributions[featName] += contrib * 100 // Scale to 0-100
		} else if e.Direction == types.DirectionSell {
			trace.ShortContributions[featName] += contrib * 100
		}
	}

	if !result.ConflictPenalty.IsZero() {
		trace.ConflictPenalty, _ = result.ConflictPenalty.Float64()
	}

	trace.FinalLongScore, _ = result.LongScore.Float64()
	trace.FinalShortScore, _ = result.ShortScore.Float64()

	// Determine primary reason
	if len(result.ReasonCodes) > 0 {
		trace.PrimaryReason = string(result.ReasonCodes[0])
	} else if result.Direction == types.DirectionBuy || result.Direction == types.DirectionSell {
		trace.PrimaryReason = "QUALIFIED"
	} else {
		trace.PrimaryReason = "NT_NO_DIRECTION"
	}

	return trace
}

// safeIndicatorValue returns the indicator value only if ready, otherwise zero.
// This prevents zero-filled indicators from producing false directional evidence.
func safeIndicatorValue(value decimal.Decimal, ready bool) decimal.Decimal {
	if !ready {
		return decimal.Zero
	}
	return value
}
