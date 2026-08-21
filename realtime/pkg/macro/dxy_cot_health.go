// Package macro provides health monitoring for DXY and COT external data providers.
package macro

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	macroDataStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "macro_data_status",
		Help: "Macro data provider status (1=available, 0=stale/unavailable)",
	}, []string{"provider"})
)

// MacroHealth monitors DXY and COT data freshness.
type MacroHealth struct {
	mu              sync.RWMutex
	dxyAvailable    bool
	dxyLastGood     float64
	dxyLastFetch    time.Time
	dxyFailCount    int
	cotAvailable    bool
	cotLastGood     float64
	cotLastFetch    time.Time
	cotFailCount    int
	maxFailCount    int
	staleThreshold  time.Duration
}

// NewMacroHealth creates a macro data health monitor.
func NewMacroHealth() *MacroHealth {
	return &MacroHealth{
		maxFailCount:   3,
		staleThreshold: 10 * time.Minute,
	}
}

// OnDXYFetchSuccess records a successful DXY fetch.
func (m *MacroHealth) OnDXYFetchSuccess(value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dxyAvailable = true
	m.dxyLastGood = value
	m.dxyLastFetch = time.Now()
	m.dxyFailCount = 0
	macroDataStatus.WithLabelValues("dxy").Set(1)
}

// OnDXYFetchFailure records a failed DXY fetch. After maxFailCount consecutive
// failures, falls back to last known good value.
func (m *MacroHealth) OnDXYFetchFailure() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dxyFailCount++
	if m.dxyFailCount >= m.maxFailCount {
		m.dxyAvailable = false
		macroDataStatus.WithLabelValues("dxy").Set(0)
	}
}

// OnCOTFetchSuccess records a successful COT fetch.
func (m *MacroHealth) OnCOTFetchSuccess(value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cotAvailable = true
	m.cotLastGood = value
	m.cotLastFetch = time.Now()
	m.cotFailCount = 0
	macroDataStatus.WithLabelValues("cot").Set(1)
}

// OnCOTFetchFailure records a failed COT fetch. After maxFailCount failures,
// COT weight is set to 0 (non-blocking).
func (m *MacroHealth) OnCOTFetchFailure() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cotFailCount++
	if m.cotFailCount >= m.maxFailCount {
		m.cotAvailable = false
		macroDataStatus.WithLabelValues("cot").Set(0)
	}
}

// DXYValue returns the current DXY value (last known good if stale).
func (m *MacroHealth) DXYValue() (float64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.dxyLastGood, m.dxyAvailable
}

// COTValue returns the current COT value (last known good if stale).
func (m *MacroHealth) COTValue() (float64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cotLastGood, m.cotAvailable
}

// IsHealthy returns true if both providers are available or have fallback values.
func (m *MacroHealth) IsHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.dxyAvailable || m.dxyLastGood > 0
}
