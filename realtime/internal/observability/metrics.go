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

	GateEvaluationMs = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "pat_gate_evaluation_ms",
		Help:    "Gate evaluation duration in milliseconds",
		Buckets: []float64{0.01, 0.1, 0.5, 1, 5, 10},
	}, []string{"gate_id"})

	// Signal metrics
	SignalsGenerated = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pat_signals_generated_total",
		Help: "Total signals generated",
	}, []string{"strategy_id", "direction"})

	GateVetoTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pat_gate_veto_total",
		Help: "Total gate vetoes",
	}, []string{"gate_id"})

	// WebSocket metrics
	WSConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pat_websocket_connections",
		Help: "Current WebSocket connections",
	})

	// Data-only Master Node (data) agent connected gauge. Monitors
	// data-collection uptime independently from execution-agent health.
	DataAgentConnected = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pat_data_agent_connected",
		Help: "Number of connected data-only Master Node agents",
	})

	WSMessagesSent = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pat_websocket_messages_sent_total",
		Help: "Total WebSocket messages sent",
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

	// Data quality
	DataQualityState = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pat_data_quality_state",
		Help: "Data quality state (0=AUTHORITATIVE, 7=INVALID)",
	}, []string{"symbol"})

	// Gate pass rate
	GatePassTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pat_gate_pass_total",
		Help: "Total gate passes",
	}, []string{"gate_id"})
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

	// Feature readiness metrics (SOW Section 27)
	FeatureReadiness = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pat_feature_readiness",
		Help: "Feature readiness state (1=READY, 0=not ready)",
	}, []string{"feature"})

	// History bootstrap metrics (SOW Section 3)
	HistoryBackfillTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pat_history_backfill_total",
		Help: "Total history backfill operations",
	})

	HistoryBackfillFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pat_history_backfill_failures_total",
		Help: "Total history backfill failures",
	})

	CandleHistoryCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pat_candle_history_count",
		Help: "Current candle history count per symbol",
	}, []string{"symbol"})
)

// ─── Advanced Risk, Adaptation, Hedging, ML/RL, Sentiment metrics ───
var (

	// Recovery metrics
	TradingState = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pat_trading_state",
		Help: "Trading recovery state (0=NORMAL, 1=RECOVERY, 2=HALTED, 3=DAILY_LIMIT)",
	}, []string{"account_id", "strategy_id"})

	DailyPnLPercent = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pat_daily_pnl_percent",
		Help: "Daily PnL percentage per account+strategy",
	}, []string{"account_id", "strategy_id"})

	ConsecutiveLosses = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pat_consecutive_losses",
		Help: "Current consecutive loss count per account+strategy",
	}, []string{"account_id", "strategy_id"})

	RecoveryEntriesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pat_recovery_entries_total",
		Help: "Total times recovery mode was entered",
	})

	RecoveryBlocksTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pat_recovery_blocks_total",
		Help: "Total signals blocked by recovery",
	}, []string{"reason"})

	DailyLimitHitsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pat_daily_limit_hits_total",
		Help: "Total daily loss limit hits",
	})

	// Adaptation metrics
	AdaptationPhase = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pat_adaptation_phase",
		Help: "Current adaptation phase (0-5)",
	}, []string{"strategy_id"})

	AdaptationChangesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pat_adaptation_changes_total",
		Help: "Total adaptation changes by source",
	}, []string{"source"})

	// Hedge metrics
	ActiveHedges = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pat_active_hedges",
		Help: "Current active hedge count",
	})

	HedgesOpenedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pat_hedges_opened_total",
		Help: "Total hedges opened",
	})

	HedgesClosedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pat_hedges_closed_total",
		Help: "Total hedges closed",
	})

	HedgePnL = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pat_hedge_pnl",
		Help: "Current aggregate hedge PnL",
	})

	// ML metrics
	MLModelLoaded = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pat_ml_model_loaded",
		Help: "Whether an ML model is loaded (1=yes, 0=no)",
	})

	MLPredictionConfidence = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pat_ml_prediction_confidence",
		Help: "Latest ML prediction confidence",
	})

	MLFallbackTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pat_ml_fallback_total",
		Help: "Total ML fallback occurrences",
	})

	// RL metrics
	RLMode = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pat_rl_mode",
		Help: "RL mode (0=disabled, 1=shadow, 2=filter, 3=live_approved)",
	})

	RLShadowDecisionsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pat_rl_shadow_decisions_total",
		Help: "Total RL shadow mode decisions",
	})

	RLFilterBlocksTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pat_rl_filter_blocks_total",
		Help: "Total RL filter blocks",
	})

	// Sentiment metrics
	SentimentScore = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pat_sentiment_score",
		Help: "Current overall sentiment score (-100 to +100)",
	})

	SentimentConfidence = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pat_sentiment_confidence",
		Help: "Current sentiment confidence (0 to 1)",
	})

	SentimentAgeSeconds = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pat_sentiment_age_seconds",
		Help: "Age of current sentiment snapshot in seconds",
	})

	SentimentProviderErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pat_sentiment_provider_errors_total",
		Help: "Total sentiment provider errors",
	}, []string{"provider"})

	// P1-001: Execution eligibility denial metrics
	EntitlementDenialTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pat_entitlement_denial_total",
		Help: "Total execution denials due to unverified entitlement/license/execution state",
	})

	GateStateGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pat_gate_state",
		Help: "Current gate state (0=PASS, 1=VETO, 2=DEGRADED, 3=UNKNOWN)",
	}, []string{"gate_id"})

	// Phase 2: Regime and shadow metrics (SOW Section 34)

	// RegimeTransitionTotal tracks regime transitions
	RegimeTransitionTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pat_regime_transitions_total",
		Help: "Total regime transitions",
	}, []string{"from", "to", "reason"})

	// RegimeCurrentGauge tracks the current regime as a numeric value
	RegimeCurrentGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pat_regime_current",
		Help: "Current regime (0=RANGE, 1=TRENDING_BULLISH, 2=TRENDING_BEARISH, 3=MEAN_REVERSION, 4=BREAKOUT, 5=HIGH_VOLATILITY)",
	})

	// RegimeConfidenceGauge tracks current regime confidence
	RegimeConfidenceGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pat_regime_confidence",
		Help: "Current regime confidence (0.0 to 1.0)",
	})

	// RegimeAgeSecondsGauge tracks how long the current regime has been active
	RegimeAgeSecondsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pat_regime_age_seconds",
		Help: "Age of current regime in seconds",
	})

	// ShadowCandidateTotal tracks shadow evaluation candidates
	ShadowCandidateTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pat_shadow_candidates_total",
		Help: "Total shadow evaluation candidates (regime-mismatched hypotheticals)",
	}, []string{"strategy_id", "regime"})

	// NoTradeByReasonTotal tracks NO-TRADE decisions by reason
	NoTradeByReasonTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pat_no_trade_by_reason_total",
		Help: "Total NO-TRADE decisions by reason code",
	}, []string{"strategy_id", "reason"})

	// StrategyEvaluationTotal tracks total strategy evaluations
	StrategyEvaluationTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pat_strategy_evaluations_total",
		Help: "Total strategy evaluations",
	}, []string{"strategy_id"})

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
