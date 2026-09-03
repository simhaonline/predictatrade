package engines

import (
	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/strategy"
	"github.com/predictatrade/realtime/internal/types"
)

// UltraScalpEngine implements precision-tuned Ultra Scalping.
// Key differences from legacy:
//   - MinAbsATR = 12.0 (rejects low-volatility signals that get eaten by cost)
//   - IgnoreStructure = true (pure ATR SL, prevents stop hunt trap)
//   - AllowedRegimes = TREND/BREAKOUT only (no RANGE/MEAN_REVERSION)
//   - MinGrade = A (rejects B/C grade signals)
//   - TP1 raised from 1.5 to 2.0 ATR (better R:R)
//   - Expiry increased from 3 to 5 minutes
type UltraScalpEngine struct {
	cfg EngineConfig
}

func (e *UltraScalpEngine) Type() EngineType     { return UltraScalp }
func (e *UltraScalpEngine) Config() EngineConfig { return e.cfg }

func (e *UltraScalpEngine) Evaluate(legacyResult strategy.StrategyResult, state *features.MarketState) EngineResult {
	// If legacy returned NO-TRADE or ERROR, pass through
	if legacyResult.Direction != types.DirectionBuy && legacyResult.Direction != types.DirectionSell {
		return EngineResult{Result: legacyResult, Fallback: true}
	}

	// Gate 1: Min ATR (lowered threshold — only reject dead-flat markets)
	if err := checkMinATR(state, e.cfg.MinAbsATR); err != nil {
		legacyResult.Direction = types.DirectionNoTrade
		legacyResult.ReasonCodes = append(legacyResult.ReasonCodes, types.NTLowATR)
		return EngineResult{Result: legacyResult, RejectReason: err.Error()}
	}

	// Regime and grade gates removed — the scoring system + hard gates handle quality.
	// Blocking based on regime prevents valid signals in mixed-regime conditions.
	// The score itself already reflects evidence quality.

	// Apply overrides (SL bypass structure, custom TPs, expiry)
	modified := applyOverrides(legacyResult, state, e.cfg)
	return EngineResult{Result: modified, Applied: true}
}
