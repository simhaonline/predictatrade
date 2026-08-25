package engines

import (
	"fmt"

	"github.com/predictatrade/realtime/internal/types"
)

// defaultConfigs is the Phase 6 hardcoded configuration matrix.
// These values override the legacy strategies.go defaults when an engine is active.
var defaultConfigs = map[EngineType]EngineConfig{
	UltraScalp: {
		Type:            UltraScalp,
		MinAbsATR:       3.0,
		IgnoreStructure: true,
		AllowedRegimes:  []string{},
		MinGrade:        "A",
		OverrideSL:      0.5,              // Tight SL = 0.5 * ATR ≈ 3-4 points
		OverrideTPs:     [3]float64{0.5, 0.8, 1.2},  // Micro profit: TP1≈3-4pts, TP2≈5-6pts, TP3≈8-10pts
		OverrideExpiry:  5,
	},
	StdScalp: {
		Type:            StdScalp,
		MinAbsATR:       2.0,
		IgnoreStructure: true,
		AllowedRegimes:  []string{},
		MinGrade:        "A",
		OverrideSL:      0.8,              // Tighter SL = 0.8 * ATR ≈ 6-7 points
		OverrideTPs:     [3]float64{1.0, 1.5, 2.5},  // TP1≈7-8pts, TP2≈10-12pts, TP3≈18-20pts
		OverrideExpiry:  10,
	},
	StdSwing: {
		Type:            StdSwing,
		MinAbsATR:       2.0,
		IgnoreStructure: true,
		AllowedRegimes:  []string{},
		MinGrade:        "A",
		OverrideSL:      1.0,              // SL = 1.0 * ATR
		OverrideTPs:     [3]float64{2.0, 3.5, 5.0},  // TP1≈14pts, TP2≈25pts, TP3≈35pts
		OverrideExpiry:  60,
	},
	TrendSwng: {
		Type:            TrendSwng,
		MinAbsATR:       3.0,
		IgnoreStructure: true,
		AllowedRegimes:  []string{},
		MinGrade:        "A",
		OverrideSL:      1.5,              // SL = 1.5 * ATR
		OverrideTPs:     [3]float64{3.0, 5.0, 8.0},  // TP1≈21pts, TP2≈35pts, TP3≈56pts
		OverrideExpiry:  240,
	},
}

// strategyToEngine maps strategy IDs to engine types.
var strategyToEngine = map[types.StrategyID]EngineType{
	types.StrategyUltraScalping:      UltraScalp,
	types.StrategyStandardScalping:   StdScalp,
	types.StrategyStandardSwing:      StdSwing,
	types.StrategyTrendSwing:         TrendSwng,
}

// GetEngine returns the specialized engine for a strategy, or nil for legacy fallback.
// If the strategy name is unknown or the engine is not configured, returns nil
// to signal the caller to use the original strategies.go logic.
func GetEngine(strategyName types.StrategyID) (SignalEngine, error) {
	engineType, ok := strategyToEngine[strategyName]
	if !ok {
		return nil, nil // Unknown strategy → fall back to legacy
	}

	cfg, ok := defaultConfigs[engineType]
	if !ok {
		return nil, fmt.Errorf("no config for engine type %s", engineType)
	}

	switch engineType {
	case UltraScalp:
		return &UltraScalpEngine{cfg: cfg}, nil
	case StdScalp:
		return &StdScalpEngine{cfg: cfg}, nil
	case StdSwing:
		return &StdSwingEngine{cfg: cfg}, nil
	case TrendSwng:
		return &TrendSwingEngine{cfg: cfg}, nil
	default:
		return nil, nil
	}
}

// GetEngineConfig returns the engine config for a strategy (for gate access).
func GetEngineConfig(strategyName types.StrategyID) *EngineConfig {
	engineType, ok := strategyToEngine[strategyName]
	if !ok {
		return nil
	}
	cfg, ok := defaultConfigs[engineType]
	if !ok {
		return nil
	}
	return &cfg
}
