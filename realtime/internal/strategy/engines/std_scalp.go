package engines

import (
	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/strategy"
	"github.com/predictatrade/realtime/internal/types"
)

// StdScalpEngine implements precision-tuned Standard Scalping.
// Key differences from legacy:
//   - MinAbsATR = 8.0 (rejects very low volatility)
//   - IgnoreStructure = false (keep structural lows for SL)
//   - All regimes allowed
//   - Accept A or B grade
type StdScalpEngine struct {
	cfg EngineConfig
}

func (e *StdScalpEngine) Type() EngineType    { return StdScalp }
func (e *StdScalpEngine) Config() EngineConfig { return e.cfg }

func (e *StdScalpEngine) Evaluate(legacyResult strategy.StrategyResult, state *features.MarketState) EngineResult {
	if legacyResult.Direction != types.DirectionBuy && legacyResult.Direction != types.DirectionSell {
		return EngineResult{Result: legacyResult, Fallback: true}
	}

	// Gate 1: Min ATR
	if err := checkMinATR(state, e.cfg.MinAbsATR); err != nil {
		legacyResult.Direction = types.DirectionNoTrade
		legacyResult.ReasonCodes = append(legacyResult.ReasonCodes, types.NTLowATR)
		return EngineResult{Result: legacyResult, RejectReason: err.Error()}
	}

	// No regime gate — all regimes allowed
	// No grade gate — A or B accepted

	// Apply overrides (keep structure, custom TPs if different from legacy)
	modified := applyOverrides(legacyResult, state, e.cfg)
	return EngineResult{Result: modified, Applied: true}
}
