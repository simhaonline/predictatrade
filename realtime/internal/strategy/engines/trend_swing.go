package engines

import (
	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/strategy"
	"github.com/predictatrade/realtime/internal/types"
)

// TrendSwingEngine implements precision-tuned Trend Swing.
// Key differences from legacy:
//   - MinAbsATR = 12.0
//   - IgnoreStructure = true (W1/D1 structure too wide, pure ATR SL better)
//   - Regime handled by scoring thresholds (engine-level gate removed; legacy
//     strategy layer still enforces TrendSwing/range separation)
//   - MinGrade = A only
type TrendSwingEngine struct {
	cfg EngineConfig
}

func (e *TrendSwingEngine) Type() EngineType    { return TrendSwng }
func (e *TrendSwingEngine) Config() EngineConfig { return e.cfg }

func (e *TrendSwingEngine) Evaluate(legacyResult strategy.StrategyResult, state *features.MarketState) EngineResult {
	if legacyResult.Direction != types.DirectionBuy && legacyResult.Direction != types.DirectionSell {
		return EngineResult{Result: legacyResult, Fallback: true}
	}

	// Gate 1: Min ATR (lowered threshold)
	if err := checkMinATR(state, e.cfg.MinAbsATR); err != nil {
		legacyResult.Direction = types.DirectionNoTrade
		legacyResult.ReasonCodes = append(legacyResult.ReasonCodes, types.NTLowATR)
		return EngineResult{Result: legacyResult, RejectReason: err.Error()}
	}

	// Regime gate removed — scoring system handles regime filtering via thresholds.
	// Hard-blocking on regime prevents valid pullback entries in mixed conditions.

	// Apply overrides (bypass structure, pure ATR SL)
	modified := applyOverrides(legacyResult, state, e.cfg)
	return EngineResult{Result: modified, Applied: true}
}
