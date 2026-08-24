package crossmarket

import (
	"encoding/json"
	"net/http"
)

// HandleCurrent serves GET /api/v1/cross-market/current
// Returns the latest confluence result with full driver breakdown.
func HandleCurrent(e *Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if e == nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "disabled",
				"message": "Cross-market confluence engine not initialized",
			})
			return
		}

		// Evaluate with neutral direction (no specific signal context)
		result := e.Evaluate(DirNeutral, EventNormal)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"score":              result.Score,
			"direction":          result.Direction,
			"confidence":         result.Confidence,
			"agreement":          result.Agreement,
			"conflict":           result.Conflict,
			"data_quality":       result.DataQuality,
			"regime":             result.Regime,
			"event_risk":         result.EventRisk,
			"correlation_regime": result.CorrelationRegime,
			"divergence":         result.DivergenceSeverity,
			"score_adjustment":   result.ScoreAdjustment,
			"mode":               result.Mode,
			"model_version":      result.ModelVersion,
			"primary_drivers":    result.PrimaryDrivers,
			"opposing_drivers":   result.OpposingDrivers,
			"missing_drivers":    result.MissingDrivers,
			"warnings":           result.Warnings,
			"drivers":            result.DriverSnapshot,
			"reason":             result.FormatReason(),
			"timestamp":          result.Timestamp,
		})
	}
}

// HandleHealth serves GET /api/v1/cross-market/health
func HandleHealth(e *Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if e == nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "disabled",
				"enabled": false,
			})
			return
		}

		e.mu.RLock()
		driverCount := len(e.drivers)
		enabledCount := 0
		for name := range e.cfg.Weights {
			if e.driverEnabled(name) {
				enabledCount++
			}
		}
		mode := e.cfg.Mode
		e.mu.RUnlock()

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":         "ok",
			"enabled":        true,
			"mode":           mode,
			"active_drivers": driverCount,
			"enabled_drivers": enabledCount,
		})
	}
}

// HandleValidationStatus serves GET /api/v1/cross-market/validation
func HandleValidationStatus(e *Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"mode":                  "shadow",
			"calendar_days":         0,
			"usable_shadow_days":    0,
			"total_candidates":      0,
			"resolved_outcomes":     0,
			"minimum_days_required": 30,
			"ablation_ready":        false,
			"walk_forward_ready":    false,
			"activation_eligible":   false,
			"message":               "Validation infrastructure ready. Shadow data collection in progress.",
		})
	}
}
