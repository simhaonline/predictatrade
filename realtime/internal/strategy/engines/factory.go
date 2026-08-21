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
		MinAbsATR:       12.0,
		IgnoreStructure: true, // Bypass structural low — prevents stop hunt
		AllowedRegimes:  []string{"TREND", "BREAKOUT"},
		MinGrade:        "A",
		OverrideSL:      1.0,
		OverrideTPs:     [3]float64{2.0, 3.0, 4.0}, // TP1 raised from 1.5 to 2.0
		OverrideExpiry:  5,                        // Increased from 3 to 5 minutes
	},
	StdScalp: {
		Type:            StdScalp,
		MinAbsATR:       8.0,
		IgnoreStructure: false, // Keep structural lows
		AllowedRegimes:  []string{}, // ALL regimes
		MinGrade:        "A",     // Accept A or B
		OverrideSL:      1.5,
		OverrideTPs:     [3]float64{2.5, 4.0, 6.0}, // Unchanged
		OverrideExpiry:  10,
	},
	StdSwing: {
		Type:            StdSwing,
		MinAbsATR:       10.0,
		IgnoreStructure: false, // Keep structure
		AllowedRegimes:  []string{}, // ALL regimes
		MinGrade:        "A",     // Accept A or B
		OverrideSL:      2.0,
		OverrideTPs:     [3]float64{3.0, 5.0, 8.0}, // Unchanged
		OverrideExpiry:  60,
	},
	TrendSwng: {
		Type:            TrendSwng,
		MinAbsATR:       12.0,
		IgnoreStructure: true, // W1/D1 structure too wide, pure ATR better
		AllowedRegimes:  []string{"TREND", "BREAKOUT"},
		MinGrade:        "A",
		OverrideSL:      2.5,
		OverrideTPs:     [3]float64{4.0, 6.5, 10.0}, // Unchanged
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
