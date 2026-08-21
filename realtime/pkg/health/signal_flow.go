// Package health — Signal flow monitor detects blockages in signal generation.
package health

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	signalFlowAlert = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "signal_flow_blockage_alert",
		Help: "Signal flow blockage alert (1=blocked, 0=normal)",
	}, []string{"severity"})

	signalsGenerated = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "signal_flow_generated_total",
		Help: "Total signals generated since startup",
	}, []string{"direction"})
)

// SignalFlowMonitor tracks signal generation rate and detects blockages.
type SignalFlowMonitor struct {
	mu                sync.Mutex
	consecutiveBlanks int
	maxBlanks         int
	lastSignalTime    time.Time
}

// NewSignalFlowMonitor creates a monitor. maxBlanks is the number of consecutive
// candles with no signal before alerting (default 5 = 25 minutes for M5).
func NewSignalFlowMonitor(maxBlanks int) *SignalFlowMonitor {
	if maxBlanks <= 0 {
		maxBlanks = 5
	}
	return &SignalFlowMonitor{maxBlanks: maxBlanks}
}

// OnSignalGenerated records that a signal was generated.
func (m *SignalFlowMonitor) OnSignalGenerated(direction string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.consecutiveBlanks = 0
	m.lastSignalTime = time.Now()
	signalsGenerated.WithLabelValues(direction).Inc()
	signalFlowAlert.WithLabelValues("high_priority").Set(0)
}

// OnCandleProcessed records that a candle was processed (with or without signal).
// Returns true if a blockage is detected.
func (m *SignalFlowMonitor) OnCandleProcessed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.consecutiveBlanks++
	if m.consecutiveBlanks >= m.maxBlanks {
		signalFlowAlert.WithLabelValues("high_priority").Set(1)
		return true
	}
	return false
}

// IsBlocked returns true if signal flow is blocked.
func (m *SignalFlowMonitor) IsBlocked() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.consecutiveBlanks >= m.maxBlanks
}

// LastSignalTime returns the timestamp of the last signal.
func (m *SignalFlowMonitor) LastSignalTime() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastSignalTime
}
