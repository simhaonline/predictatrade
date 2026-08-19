// Package ptb — Prometheus observability metrics.
// Section 32: Structured logging + Prometheus metrics for PTB.
package ptb

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PTB Prometheus metrics — follow existing pat_ naming convention.
var (
	AnalysisTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pat_ptb_analysis_total",
		Help: "Total PTB analyses performed",
	}, []string{"symbol", "action"})

	AnalysisLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "pat_ptb_analysis_latency_ms",
		Help:    "PTB analysis latency in milliseconds",
		Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500},
	}, []string{"symbol"})

	ActionTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pat_ptb_action_total",
		Help: "PTB action distribution",
	}, []string{"action"})

	SetupQualityTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pat_ptb_setup_quality_total",
		Help: "PTB setup quality distribution",
	}, []string{"quality"})

	ComponentFailureTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pat_ptb_component_failure_total",
		Help: "PTB component failures",
	}, []string{"component"})

	StaleInputTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pat_ptb_stale_input_total",
		Help: "PTB stale input occurrences",
	}, []string{"component"})

	RegimeTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pat_ptb_regime_total",
		Help: "PTB regime distribution",
	}, []string{"regime"})

	SignalConversionTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pat_ptb_signal_conversion_total",
		Help: "PTB signal conversion (PTB action → actual signal)",
	}, []string{"ptb_action", "signal_direction"})

	ConfluenceScore = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pat_ptb_confluence_score",
		Help: "PTB confluence score",
	}, []string{"symbol"})

	ManipulationIndex = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pat_ptb_manipulation_index",
		Help: "PTB manipulation index 0-100",
	}, []string{"symbol"})
)
