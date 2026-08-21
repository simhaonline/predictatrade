// Package engines provides isolated, precision-tuned strategy engines that wrap
// (not replace) the legacy strategies.go logic. Each engine applies strategy-specific
// overrides: Min ATR filter, structural SL bypass, regime gating, grade filtering,
// and custom TP/SL multipliers. If an engine fails or the strategy is unknown, the
// factory returns nil to signal the caller to fall back to the original strategies.go.
package engines

import (
	"fmt"

	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/strategy"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// EngineType identifies which strategy engine to use.
type EngineType string

const (
	UltraScalp EngineType = "ULTRA_SCALP"
	StdScalp  EngineType = "STD_SCALP"
	StdSwing  EngineType = "STD_SWING"
	TrendSwng EngineType = "TREND_SWING"
)

// EngineConfig holds per-engine overrides from the Phase 6 configuration matrix.
type EngineConfig struct {
	Type            EngineType
	MinAbsATR       float64 // Min absolute ATR value to avoid cost erosion
	IgnoreStructure bool    // If true, SL is pure ATR (bypass structural low)
	AllowedRegimes  []string // e.g., ["TREND", "BREAKOUT"]; empty = ALL
	MinGrade        string  // e.g., "A" to filter lower quality signals
	OverrideSL      float64 // Custom SL ATR multiplier (0 = use legacy)
	OverrideTPs     [3]float64 // Custom TP ATR multipliers (0 = use legacy)
	OverrideExpiry  int  // Expiry in minutes (0 = use legacy)
}

// EngineResult is the output of a SignalEngine evaluation.
type EngineResult struct {
	// The modified strategy result (entry/SL/TP adjusted by engine overrides).
	Result strategy.StrategyResult
	// Whether the engine accepted and modified the signal.
	Applied bool
	// If non-empty, the signal was rejected by the engine with this reason.
	RejectReason string
	// Whether to fall back to legacy strategies.go logic.
	Fallback bool
}

// SignalEngine is the interface implemented by each strategy engine.
type SignalEngine interface {
	// Type returns the engine type identifier.
	Type() EngineType
	// Config returns the engine's configuration.
	Config() EngineConfig
	// Evaluate applies engine-specific overrides and gates to a strategy result.
	// It receives the legacy StrategyResult from strategies.go and the market state.
	// It returns a modified result or a rejection reason.
	Evaluate(legacyResult strategy.StrategyResult, state *features.MarketState) EngineResult
}

// applyOverrides adjusts SL/TP based on engine config, optionally bypassing structure.
func applyOverrides(res strategy.StrategyResult, state *features.MarketState, cfg EngineConfig) strategy.StrategyResult {
	if state == nil || state.Indicators.ATR.IsZero() {
		return res
	}

	atr := state.Indicators.ATR
	entry := res.EntryPrice
	if entry.IsZero() {
		entry = state.CurrentPrice
	}

	// Determine SL multiplier
	slMult := cfg.OverrideSL
	if slMult == 0 {
		slMult = 1.5 // safe default
	}

	halfSpread := state.Spread.Div(decimal.NewFromInt(2))

	// Determine TP multipliers
	tp1Mult := cfg.OverrideTPs[0]
	tp2Mult := cfg.OverrideTPs[1]
	tp3Mult := cfg.OverrideTPs[2]
	if tp1Mult == 0 {
		tp1Mult = 2.5
	}
	if tp2Mult == 0 {
		tp2Mult = 4.0
	}
	if tp3Mult == 0 {
		tp3Mult = 6.0
	}

	if res.Direction == types.DirectionBuy {
		// SL calculation
		if cfg.IgnoreStructure {
			// Pure ATR-based SL — bypass structural low to prevent stop hunt
			res.StopLoss = entry.Sub(atr.Mul(decimal.NewFromFloat(slMult))).Sub(halfSpread)
		}
		// TP calculation (always override if configured)
		if cfg.OverrideTPs[0] > 0 || cfg.OverrideSL > 0 {
			res.TP1 = entry.Add(atr.Mul(decimal.NewFromFloat(tp1Mult)))
			res.TP2 = entry.Add(atr.Mul(decimal.NewFromFloat(tp2Mult)))
			res.TP3 = entry.Add(atr.Mul(decimal.NewFromFloat(tp3Mult)))
		}
	} else if res.Direction == types.DirectionSell {
		// SL calculation
		if cfg.IgnoreStructure {
			res.StopLoss = entry.Add(atr.Mul(decimal.NewFromFloat(slMult))).Add(halfSpread)
		}
		// TP calculation
		if cfg.OverrideTPs[0] > 0 || cfg.OverrideSL > 0 {
			res.TP1 = entry.Sub(atr.Mul(decimal.NewFromFloat(tp1Mult)))
			res.TP2 = entry.Sub(atr.Mul(decimal.NewFromFloat(tp2Mult)))
			res.TP3 = entry.Sub(atr.Mul(decimal.NewFromFloat(tp3Mult)))
		}
	}

	// Override expiry if configured
	if cfg.OverrideExpiry > 0 {
		res.ExpiryMinutes = cfg.OverrideExpiry
	}

	return res
}

// checkMinATR returns an error if ATR is below the engine's minimum.
func checkMinATR(state *features.MarketState, minATR float64) error {
	if state == nil || minATR <= 0 {
		return nil
	}
	atrVal, _ := state.Indicators.ATR.Float64()
	if atrVal < minATR {
		return fmt.Errorf("ERR_LOW_VOLATILITY: ATR=%.2f < min=%.2f", atrVal, minATR)
	}
	return nil
}

// checkRegime returns an error if the current regime is not in the allowed list.
func checkRegime(state *features.MarketState, allowed []string) error {
	if state == nil || len(allowed) == 0 {
		return nil // empty = ALL regimes allowed
	}
	currentRegime := string(state.Regime.Current)
	for _, r := range allowed {
		if currentRegime == r || currentRegime == "TRENDING_BULLISH" && r == "TREND" ||
			currentRegime == "TRENDING_BEARISH" && r == "TREND" {
			return nil
		}
	}
	return fmt.Errorf("ERR_REGIME_MISMATCH: regime=%s not in %v", currentRegime, allowed)
}

// regimeString returns a human-readable regime name.
func regimeString(state *features.MarketState) string {
	if state == nil {
		return ""
	}
	return string(state.Regime.Current)
}
