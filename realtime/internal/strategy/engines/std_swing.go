package engines

import (
	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/strategy"
	"github.com/predictatrade/realtime/internal/types"
)

// StdSwingEngine implements precision-tuned Standard Swing.
// Key differences from legacy:
//   - MinAbsATR = 10.0
//   - IgnoreStructure = false (keep structure for swing trading)
//   - All regimes allowed
//   - Accept A or B grade
type StdSwingEngine struct {
	cfg EngineConfig
}

func (e *StdSwingEngine) Type() EngineType    { return StdSwing }
func (e *StdSwingEngine) Config() EngineConfig { return e.cfg }

func (e *StdSwingEngine) Evaluate(legacyResult strategy.StrategyResult, state *features.MarketState) EngineResult {
	if legacyResult.Direction != types.DirectionBuy && legacyResult.Direction != types.DirectionSell {
		return EngineResult{Result: legacyResult, Fallback: true}
	}

	// Gate 1: Min ATR
	if err := checkMinATR(state, e.cfg.MinAbsATR); err != nil {
		legacyResult.Direction = types.DirectionNoTrade
		legacyResult.ReasonCodes = append(legacyResult.ReasonCodes, types.NTLowATR)
		return EngineResult{Result: legacyResult, RejectReason: err.Error()}
	}

	// Apply overrides
	modified := applyOverrides(legacyResult, state, e.cfg)
	return EngineResult{Result: modified, Applied: true}
}
