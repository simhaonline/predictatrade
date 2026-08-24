package crossmarket

import (
	"sync"
	"time"
)

// DriverRegistry is the central registry for all cross-market drivers.
// Each driver exposes its health, freshness, and last observation.
type DriverRegistry struct {
	mu       sync.RWMutex
	drivers  map[DriverName]*DriverState
}

// DriverState tracks the runtime state of a single macro driver.
type DriverState struct {
	Name        DriverName
	Enabled     bool
	Configured  bool
	Provider    string
	Health      DataQuality
	LastUpdate  time.Time
	LastValue   float64
	Adapter     func() DriverSnapshot
}

// NewDriverRegistry creates a registry with default driver states.
func NewDriverRegistry(cfg Config) *DriverRegistry {
	r := &DriverRegistry{drivers: make(map[DriverName]*DriverState)}

	drivers := []struct {
		name      DriverName
		enabled   bool
		provider  string
	}{
		{DriverDXY, cfg.DXYEnabled, "twelvedata"},
		{DriverEURUSD, cfg.EURUSDEnabled, "dxy_component"},
		{DriverRealYields, cfg.RealYieldsEnabled, "fmp"},
		{DriverFedContext, false, "economic_calendar"},
		{DriverVIX, cfg.VIXEnabled, "twelvedata"},
		{DriverCOT, cfg.COTEnabled, "fmp"},
		{DriverBTC, cfg.BTCEnabled, "twelvedata"},
		{DriverOil, cfg.OilEnabled, "twelvedata"},
	}

	for _, d := range drivers {
		r.drivers[d.name] = &DriverState{
			Name:       d.name,
			Enabled:    d.enabled,
			Configured: false, // updated when provider confirms
			Provider:   d.provider,
			Health:     QualityMissing,
		}
	}

	return r
}

// UpdateDriver updates the state of a driver after receiving an observation.
func (r *DriverRegistry) UpdateDriver(name DriverName, health DataQuality, value float64, ts time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if state, ok := r.drivers[name]; ok {
		state.Health = health
		state.LastUpdate = ts
		state.LastValue = value
		if health == QualityConnected {
			state.Configured = true
		}
	}
}

// GetDriverState returns the current state of a driver.
func (r *DriverRegistry) GetDriverState(name DriverName) *DriverState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if state, ok := r.drivers[name]; ok {
		return state
	}
	return nil
}

// GetAllDrivers returns the state of all registered drivers.
func (r *DriverRegistry) GetAllDrivers() map[DriverName]*DriverState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[DriverName]*DriverState)
	for k, v := range r.drivers {
		result[k] = v
	}
	return result
}

// HealthSummary returns a summary of all driver health states.
func (r *DriverRegistry) HealthSummary() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]string)
	for name, state := range r.drivers {
		health := string(state.Health)
		if !state.Enabled {
			health = "DISABLED"
		} else if !state.Configured && state.Health == QualityMissing {
			health = "NOT_CONFIGURED"
		}
		result[string(name)] = health
	}
	return result
}
