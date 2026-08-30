// Package observability provides metrics and structured logging.
// SOW Section 149: Observability and reliability
package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Market data metrics
	TicksReceived = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pat_ticks_received_total",
		Help: "Total ticks received by provider",
	}, []string{"symbol", "source"})

	TicksRejected = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pat_ticks_rejected_total",
		Help: "Total ticks rejected as bad data",
	}, []string{"symbol", "reason"})

	CandlesGenerated = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pat_candles_generated_total",
		Help: "Total candles generated",
	}, []string{"symbol", "timeframe"})

	// Pipeline latency
	TickLatencyMs = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "pat_tick_latency_ms",
		Help:    "Tick processing latency in milliseconds",
		Buckets: []float64{0.1, 0.5, 1, 5, 10, 50, 100, 500},
	}, []string{"symbol"})

	// Signal metrics
	SignalsGenerated = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pat_signals_generated_total",
		Help: "Total signals generated",
	}, []string{"strategy_id", "direction"})

	GateVetoTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pat_gate_veto_total",
		Help: "Total gate vetoes",
	}, []string{"gate_id"})

		// Data-only Master Node (data) agent connected gauge. Monitors
	// data-collection uptime independently from execution-agent health.
	DataAgentConnected = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pat_data_agent_connected",
		Help: "Number of connected data-only Master Node agents",
	})

	// Strategy scores
	StrategyScore = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pat_strategy_score",
		Help: "Current strategy raw score",
	}, []string{"strategy_id"})

	CalibratedProbability = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pat_calibrated_probability",
		Help: "Calibrated probability",
	}, []string{"strategy_id"})

		)

// Cooldown and duplicate prevention metrics (SOW Section 17, 18, 37)
var (
	CooldownRejections = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pat_cooldown_rejections_total",
		Help: "Total signals rejected due to cooldown",
	}, []string{"strategy_id"})

	CooldownErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pat_cooldown_errors_total",
		Help: "Total cooldown check errors (Valkey failures)",
	})

	DuplicateRejections = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pat_duplicate_rejections_total",
		Help: "Total signals rejected as duplicates",
	}, []string{"strategy_id"})

	DuplicateErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pat_duplicate_errors_total",
		Help: "Total duplicate check errors (Valkey failures)",
	})

)

// ─── Advanced Risk, Adaptation, Hedging, ML/RL, Sentiment metrics ───
var (

	// P1-001: Execution eligibility denial metrics
	EntitlementDenialTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pat_entitlement_denial_total",
		Help: "Total execution denials due to unverified entitlement/license/execution state",
	})

	// Phase 2: Regime and shadow metrics (SOW Section 34)

								// StrategySignalTotal tracks BUY/SELL signals per strategy
	StrategySignalTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pat_strategy_signals_total",
		Help: "Total BUY/SELL signals per strategy",
	}, []string{"strategy_id", "direction"})

	// BE-6 reconciliation gap gauges: signals delivered but never ACKed, and
	// ACKed signals that never reported a fill, inside the monitor TTL windows.
	ReconciliationAcksTimeout = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pat_reconciliation_acks_timeout",
		Help: "Signals delivered but not acknowledged within the ACK TTL",
	})

	ReconciliationFillsTimeout = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pat_reconciliation_fills_timeout",
		Help: "Signals acknowledged but not filled within the fill TTL",
	})

	ReconciliationTracked = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pat_reconciliation_tracked_signals",
		Help: "Signals currently tracked by the reconciliation registry",
	})
)
