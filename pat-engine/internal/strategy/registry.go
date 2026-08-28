package strategy

import (
	"fmt"

	"pat-engine/internal/config"
	"pat-engine/internal/types"
)

// All returns every strategy product initialised from the single-source config.
// Adding a strategy = add a DefaultXxx in config + a NewXxx here.
func All() map[types.StrategyID]Strategy {
	cfgs := config.AllDefaults()
	m := map[types.StrategyID]Strategy{}
	if c, ok := cfgs["ULTRA_SCALPING"]; ok {
		m[types.StrategyUltraScalping] = NewUltraScalping(c)
	}
	if c, ok := cfgs["STANDARD_SCALPING"]; ok {
		m[types.StrategyStandardScalping] = NewStandardScalping(c)
	}
	if c, ok := cfgs["STANDARD_SWING"]; ok {
		m[types.StrategyStandardSwing] = NewStandardSwing(c)
	}
	if c, ok := cfgs["TREND_SWING"]; ok {
		m[types.StrategyTrendSwing] = NewTrendSwing(c)
	}
	return m
}

// Get returns a single strategy by ID.
func Get(id string) (Strategy, bool) {
	s, ok := All()[types.StrategyID(id)]
	return s, ok
}

// Must returns a strategy by ID or panics. For tests/fixed wiring only.
func Must(id string) Strategy {
	s, ok := Get(id)
	if !ok {
		panic(fmt.Sprintf("unknown strategy %q", id))
	}
	return s
}
