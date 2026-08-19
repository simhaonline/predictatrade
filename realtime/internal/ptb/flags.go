// Package ptb implements the Professional Trader Brain shared intelligence layer.
// Stage 4: Advanced Market Intelligence Integration.
//
// Key principle: PTB provides CONTEXT and EVIDENCE to the existing four strategy
// engines. It does NOT create a second signal engine. All new modules start in
// SHADOW mode with ZERO production score impact until validated.
package ptb

import (
	"sync"

	"github.com/predictatrade/realtime/internal/types"
)

// FlagRegistry manages feature flags for all advanced intelligence modules.
// Stage 4 Section 30: Each module supports OFF / SHADOW / ACTIVE.
type FlagRegistry struct {
	mu    sync.RWMutex
	flags map[string]types.ModuleMode
}

// NewFlagRegistry creates a flag registry with all modules defaulting to SHADOW.
// Stage 4 Section 29: New modules start in SHADOW mode — calculate + persist +
// observe, but do NOT alter BUY/SELL/WAIT/NO_TRADE/BLOCKED/ERROR, score,
// Entry, SL, TP, R:R, or position size.
func NewFlagRegistry() *FlagRegistry {
	r := &FlagRegistry{flags: make(map[string]types.ModuleMode)}
	for _, m := range AllModuleNames() {
		r.flags[m] = types.ModuleShadow
	}
	// Institutional Footprint is UNSUPPORTED — broker tick data cannot provide it
	r.flags[ModuleInstitutionalFootprint] = types.ModuleUnsupported
	return r
}

// GetMode returns the current mode for a module.
func (r *FlagRegistry) GetMode(module string) types.ModuleMode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if m, ok := r.flags[module]; ok {
		return m
	}
	return types.ModuleOff
}

// SetMode sets the mode for a module. Production score impact only when ACTIVE.
func (r *FlagRegistry) SetMode(module string, mode types.ModuleMode) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.flags[module] = mode
}

// IsShadow returns true if the module is in SHADOW mode (calculates but zero score impact).
func (r *FlagRegistry) IsShadow(module string) bool {
	return r.GetMode(module) == types.ModuleShadow
}

// IsActive returns true if the module is ACTIVE (may contribute to scoring).
func (r *FlagRegistry) IsActive(module string) bool {
	return r.GetMode(module) == types.ModuleActive
}

// IsUnsupported returns true if the module is UNSUPPORTED by current data source.
func (r *FlagRegistry) IsUnsupported(module string) bool {
	return r.GetMode(module) == types.ModuleUnsupported
}

// AllModes returns a snapshot of all module modes for observability.
func (r *FlagRegistry) AllModes() map[string]types.ModuleMode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]types.ModuleMode, len(r.flags))
	for k, v := range r.flags {
		result[k] = v
	}
	return result
}

// Module name constants.
const (
	ModuleLiquidityVoid        = "liquidity_void"
	ModuleWickFill             = "wick_fill"
	ModuleSessionImbalance     = "session_imbalance"
	ModuleCandleRangeProjector = "candle_range_projector"
	ModuleTimeAtMode           = "time_at_mode"
	ModuleEngineeredLiquidity  = "engineered_liquidity_proxy"
	ModuleMarketPhase          = "market_phase"
	ModuleRelativeVolumeFlow   = "relative_tick_volume_flow"
	ModulePriceDelivery        = "price_delivery"
	ModuleStopHuntProxy        = "stop_hunt_proxy"
	ModuleInstitutionalFootprint = "institutional_footprint"
	ModuleTimeCycle            = "time_cycle_analytics"
	ModuleAlgoActivity         = "algo_activity_proxy"
	ModuleCompleteLiquidityMap = "complete_liquidity_map"
	ModuleManipulationProxy    = "manipulation_proxy"
	ModuleMTFBias              = "mtf_bias_engine"
	ModuleVolatilityRegime     = "volatility_regime_engine"
	ModuleSRQuality            = "sr_quality_engine"
	ModuleMicrostructure       = "microstructure_engine"
	ModuleStatisticalPerf      = "statistical_performance_engine"
	ModuleDataQuality          = "data_quality_engine"
)

// AllModuleNames returns all advanced module names.
func AllModuleNames() []string {
	return []string{
		ModuleLiquidityVoid,
		ModuleWickFill,
		ModuleSessionImbalance,
		ModuleCandleRangeProjector,
		ModuleTimeAtMode,
		ModuleEngineeredLiquidity,
		ModuleMarketPhase,
		ModuleRelativeVolumeFlow,
		ModulePriceDelivery,
		ModuleStopHuntProxy,
		ModuleInstitutionalFootprint,
		ModuleTimeCycle,
		ModuleAlgoActivity,
		ModuleCompleteLiquidityMap,
		ModuleManipulationProxy,
		ModuleMTFBias,
		ModuleVolatilityRegime,
		ModuleSRQuality,
		ModuleMicrostructure,
		ModuleStatisticalPerf,
		ModuleDataQuality,
	}
}
