// Package health — Production health manager that coordinates all health checks.
package health

import (
	"sync"

	"github.com/predictatrade/realtime/pkg/macro"
)

// Manager coordinates all health monitors and provides a single IsDegraded() check.
type Manager struct {
	mu              sync.RWMutex
	staleChecker    *StaleChecker
	signalFlow      *SignalFlowMonitor
	macroHealth     *macro.MacroHealth
	degraded        bool
	degradedReason  string
}

// NewManager creates a health manager from individual monitors.
func NewManager(stale *StaleChecker, flow *SignalFlowMonitor, macroMon *macro.MacroHealth) *Manager {
	return &Manager{
		staleChecker: stale,
		signalFlow:   flow,
		macroHealth:  macroMon,
	}
}

// IsDegraded returns true if the system is in a degraded state.
// Degraded means: stale data (critical), signal flow blocked, or macro data unavailable.
func (m *Manager) IsDegraded() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.degraded
}

// DegradedReason returns the reason for degradation.
func (m *Manager) DegradedReason() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.degradedReason
}

// Update evaluates all health monitors and updates the degraded state.
func (m *Manager) Update() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.staleChecker != nil {
		_, critical, _ := m.staleChecker.Check()
		if critical {
			m.degraded = true
			m.degradedReason = "STALE_DATA_CRITICAL"
			return
		}
	}

	if m.signalFlow != nil && m.signalFlow.IsBlocked() {
		m.degraded = true
		m.degradedReason = "SIGNAL_FLOW_BLOCKED"
		return
	}

	if m.macroHealth != nil && !m.macroHealth.IsHealthy() {
		m.degraded = true
		m.degradedReason = "MACRO_DATA_UNAVAILABLE"
		return
	}

	m.degraded = false
	m.degradedReason = ""
}

// SetDataStale marks the data source as stale (called by reconnect manager).
func (m *Manager) SetDataStale(stale bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if stale {
		m.degraded = true
		m.degradedReason = "MT5_CONNECTION_LOST"
	}
}
