// Package health provides production health monitoring for the Predict-A-Trade engine.
package health

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	staleDataAlert = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "stale_data_alert",
		Help: "Stale data alert (1=stale, 0=fresh)",
	}, []string{"severity"})

	dataSourceHealth = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "data_source_health",
		Help: "Data source health (1=healthy, 0=unhealthy)",
	}, []string{"source"})
)

// StaleChecker monitors candle freshness and alerts when data is stale.
type StaleChecker struct {
	mu             sync.RWMutex
	lastCandleTime time.Time
	staleThreshold time.Duration
	criticalThreshold time.Duration
}

// NewStaleChecker creates a stale data checker.
// staleThreshold: warn after this duration (default 60s).
// criticalThreshold: mark unhealthy after this duration (default 120s).
func NewStaleChecker(staleThreshold, criticalThreshold time.Duration) *StaleChecker {
	if staleThreshold == 0 {
		staleThreshold = 60 * time.Second
	}
	if criticalThreshold == 0 {
		criticalThreshold = 120 * time.Second
	}
	return &StaleChecker{
		staleThreshold:    staleThreshold,
		criticalThreshold: criticalThreshold,
	}
}

// UpdateLastCandleTime records the latest candle timestamp.
func (s *StaleChecker) UpdateLastCandleTime(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastCandleTime = t
}

// Check evaluates data freshness and returns status.
func (s *StaleChecker) Check() (stale bool, critical bool, age time.Duration) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.lastCandleTime.IsZero() {
		return true, true, 0
	}

	age = time.Since(s.lastCandleTime)
	stale = age > s.staleThreshold
	critical = age > s.criticalThreshold

	if stale {
		staleDataAlert.WithLabelValues("warning").Set(1)
	} else {
		staleDataAlert.WithLabelValues("warning").Set(0)
	}

	if critical {
		dataSourceHealth.WithLabelValues("mt5").Set(0)
	} else {
		dataSourceHealth.WithLabelValues("mt5").Set(1)
	}

	return
}

// IsHealthy returns true if data source is not in critical state.
func (s *StaleChecker) IsHealthy() bool {
	_, critical, _ := s.Check()
	return !critical
}
