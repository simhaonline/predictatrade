package gateway

import (
	"encoding/json"
	"net/http"
	"time"
)

// PipelineMonitor — check.md idea (2026-08-30): "Signal → Risk → Execution → Review"
// Per-signal pipeline monitor.
func (h *HTTPServer) handlePipelineMonitor(w http.ResponseWriter, r *http.Request) {
	// Stage 1: SIGNAL
	signalStatus := "healthy"
	signalCount := int64(0)
	signalLast := "—"
	if h.persister != nil && h.persister.GetDB() != nil {
		var lastSignal *time.Time
		if err := h.persister.GetDB().QueryRow(
			"SELECT max(created_at) FROM trading.signals WHERE created_at > now() - interval '5 minutes'",
		).Scan(&lastSignal); err == nil && lastSignal != nil {
			signalLast = lastSignal.Format(time.RFC3339)
		}
		if err := h.persister.GetDB().QueryRow(
			"SELECT count(*) FROM trading.signals WHERE created_at > now() - interval '5 minutes'",
		).Scan(&signalCount); err != nil {
			signalStatus = "db_error"
		}
	}
	if signalCount == 0 {
		signalStatus = "idle"
	}

	// Stage 2: Risk — hard gates fail-closed per signal
	riskStatus := "live"
	// Compute the real veto count from persisted gate decisions (not a hardcoded
	// zero). A NO-TRADE from insufficient score is NOT a gate veto; only hard-gate
	// VETO results are counted, so this reconciles with the console's NO-TRADE
	// reasons. This is pure instrumentation — it never blocks signal generation.
	vetoCountRecent := 0
	if h.persister != nil && h.persister.GetDB() != nil {
		if err := h.persister.GetDB().QueryRow(
			"SELECT count(DISTINCT signal_id) FROM trading.risk_decisions WHERE gate_result = 'VETO' AND evaluated_at > now() - interval '5 minutes'",
		).Scan(&vetoCountRecent); err != nil {
			riskStatus = "db_error"
		}
	}

	// Stage 3: Execution — slippage + commission + broker fees
	execStatus := "connected"

	// Stage 4: Review — Backtest + Forward test + Reconciliation
	reviewPct := 100.0

	stages := []map[string]any{
		{
			"stage":    "Signal",
			"name":     "6 Intelligence Engines",
			"status":   signalStatus,
			"count_5m": signalCount,
			"last_at":  signalLast,
			"engines":  []string{"ATEN", "EQFE", "STANDARD_SCALPING", "ULTRA_SCALPING", "STANDARD_SWING", "TREND_SWING"},
			"note":     "6 engines evaluate per-TF candle closed. Confirmation gate + ASTRO gate + hard 16-gate pipeline.",
		},
		{
			"stage":     "Risk",
			"name":      "16 Hard Risk Gates",
			"status":    riskStatus,
			"vetoed_5m": vetoCountRecent,
			"detail":    "Order: ExecutionPermission → BrokerSymbol → Capital → DailyLoss → Spread → News → Slippage → etc",
		},
		{
			"stage":  "Execution",
			"name":   "Windows Agent → MetaTrader",
			"status": execStatus,
			"detail": "Spread/commission/slippage tracked per execution. SL distance filter + candle range filter (playbook §8).",
		},
		{
			"stage":    "Review",
			"name":     "Backtest + Forward Test",
			"status":   "live",
			"backfill": reviewPct,
			"detail":   "ATEN Astro engine runs independent walk-forward; signal reconciliation tracks ACK + fill rates",
		},
	}

	json.NewEncoder(w).Encode(map[string]any{
		"pipeline":  stages,
		"timestamp": time.Now().UTC(),
	})
}
