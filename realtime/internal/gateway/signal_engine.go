package gateway

import (
	"encoding/json"
	"net/http"
	"time"
)

// handleSignalEngine — /api/v1/admin/signal-engine (admin JWT required).
// One-stop truth for the capital-tiered signal engine: per-tier device
// classification + delivery counts, recent signals with their tier
// eligibility (from the edge-poll payload jsonb — v1.23 annotation fields
// are not persisted as trading.signals columns), and 24h pipeline stats.
// Snake_case JSON (Go API contract — the devil-liquidity PascalCase bug
// taught us this).
//
// Data sources (DB-authoritative, no in-memory guesswork):
//   - licensing.edge_signal_queue.payload → per-signal tier eligibility,
//     SuggestedLot, strategy, direction (PascalCase keys — payload is the
//     marshalled types.Signal struct)
//   - licensing.edge_device_state         → capital tier per device
//   - licensing.devices                   → exec device inventory
func (h *HTTPServer) handleSignalEngine(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h.persister == nil || h.persister.GetDB() == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]any{
			"status": "unavailable",
			"error":  "database not initialized",
		})
		return
	}
	db := h.persister.GetDB()
	ctx := r.Context()
	resp := map[string]any{"status": "ok", "generated_at": time.Now().UTC().Format(time.RFC3339)}

	// ── 1. Per-tier device distribution (exec devices only) ──────────────
	type tierRow struct {
		Tier      string  `json:"tier"`
		Devices   int     `json:"devices"`
		EquitySum float64 `json:"equity_sum"`
	}
	tiers := []tierRow{}
	if rows, err := db.QueryContext(ctx, `
		SELECT COALESCE(eds.capital_tier, '') AS tier, count(*)::int AS devices,
		       COALESCE(sum(eds.last_equity), 0)::float8 AS equity_sum
		  FROM licensing.devices d
		  LEFT JOIN licensing.edge_device_state eds ON eds.device_id = d.id
		 WHERE d.role = 'exec' AND d.revoked_at IS NULL
		 GROUP BY 1 ORDER BY 1`); err != nil {
		resp["tier_error"] = err.Error()
	} else {
		for rows.Next() {
			var t tierRow
			if err := rows.Scan(&t.Tier, &t.Devices, &t.EquitySum); err == nil {
				tiers = append(tiers, t)
			}
		}
		rows.Close()
	}
	resp["devices_by_tier"] = tiers

	// ── 2. Delivery outcomes per tier, last 24h ──────────────────────────
	type deliveryRow struct {
		Tier      string `json:"tier"`
		Delivered int    `json:"delivered"`
		Acked     int    `json:"acked"`
		Expired   int    `json:"expired"`
	}
	deliveries := []deliveryRow{}
	if rows, err := db.QueryContext(ctx, `
		SELECT COALESCE(eds.capital_tier, '') AS tier,
		       count(*)::int AS delivered,
		       count(*) FILTER (WHERE q.status = 'ACKED')::int AS acked,
		       count(*) FILTER (WHERE q.status = 'EXPIRED')::int AS expired
		  FROM licensing.edge_signal_queue q
		  JOIN licensing.devices d ON d.id = q.device_id
		  LEFT JOIN licensing.edge_device_state eds ON eds.device_id = d.id
		 WHERE q.created_at > now() - interval '24 hours'
		 GROUP BY 1 ORDER BY 1`); err != nil {
		resp["delivery_error"] = err.Error()
	} else {
		for rows.Next() {
			var d deliveryRow
			if err := rows.Scan(&d.Tier, &d.Delivered, &d.Acked, &d.Expired); err == nil {
				deliveries = append(deliveries, d)
			}
		}
		rows.Close()
	}
	resp["delivery_24h_by_tier"] = deliveries

	// ── 3. Recent signals with tier eligibility (from queue payloads) ────
	type signalRow struct {
		SignalID      string    `json:"signal_id"`
		CreatedAt     time.Time `json:"created_at"`
		StrategyID    string    `json:"strategy_id"`
		Direction     string    `json:"direction"`
		EntryPrice    float64   `json:"entry_price"`
		StopLoss      float64   `json:"stop_loss"`
		SuggestedLot  float64   `json:"suggested_lot"`
		EligibleTiers []string  `json:"eligible_tiers"`
		Delivered     int       `json:"delivered"`
		Acked         int       `json:"acked"`
	}
	signals := []signalRow{}
	if rows, err := db.QueryContext(ctx, `
		SELECT q.signal_id,
		       min(q.created_at)::timestamptz AS created_at,
		       COALESCE(min(q.payload->>'StrategyID'), '') AS strategy_id,
		       COALESCE(min(q.payload->>'Direction'), '') AS direction,
		       COALESCE(min((q.payload->>'EntryPrice')::float8), 0) AS entry_price,
		       COALESCE(min((q.payload->>'StopLoss')::float8), 0) AS stop_loss,
		       COALESCE(max(COALESCE(NULLIF(q.payload->>'SuggestedLot','')::float8, 0)), 0) AS suggested_lot,
		       COALESCE(array_agg(DISTINCT t.value) FILTER (WHERE t.value IS NOT NULL), '{}'::text[]) AS eligible_tiers,
		       count(*)::int AS delivered,
		       count(*) FILTER (WHERE q.status = 'ACKED')::int AS acked
		  FROM licensing.edge_signal_queue q
		  CROSS JOIN LATERAL jsonb_array_elements_text(
		      CASE WHEN jsonb_typeof(q.payload->'EligibleTiers') = 'array'
		           THEN q.payload->'EligibleTiers' ELSE '[]'::jsonb END) AS t(value)
		 WHERE q.payload ? 'StrategyID'
		   AND q.created_at > now() - interval '12 hours'
		 GROUP BY q.signal_id
		 ORDER BY created_at DESC
		 LIMIT 30`); err != nil {
		resp["signals_error"] = err.Error()
	} else {
		for rows.Next() {
			var s signalRow
			var tiersArr []byte
			if err := rows.Scan(&s.SignalID, &s.CreatedAt, &s.StrategyID, &s.Direction,
				&s.EntryPrice, &s.StopLoss, &s.SuggestedLot, &tiersArr, &s.Delivered, &s.Acked); err == nil {
				s.EligibleTiers = []string{}
				if len(tiersArr) > 0 {
					_ = json.Unmarshal(tiersArr, &s.EligibleTiers)
				}
				signals = append(signals, s)
			}
		}
		rows.Close()
	}
	resp["recent_signals"] = signals

	// ── 4. 24h pipeline stats ────────────────────────────────────────────
	stats := map[string]any{}
	var total24, acked24, expired24, pending24 int
	if err := db.QueryRowContext(ctx, `
		SELECT count(*)::int,
		       count(*) FILTER (WHERE status = 'ACKED')::int,
		       count(*) FILTER (WHERE status = 'EXPIRED')::int,
		       count(*) FILTER (WHERE status IN ('PENDING','IN_FLIGHT'))::int
		  FROM licensing.edge_signal_queue
		 WHERE created_at > now() - interval '24 hours'`).
		Scan(&total24, &acked24, &expired24, &pending24); err != nil {
		resp["stats_error"] = err.Error()
	}
	stats["enqueued_24h"] = total24
	stats["acked_24h"] = acked24
	stats["expired_24h"] = expired24
	stats["pending_24h"] = pending24

	// Signals restricted by capital tier in the last 24h (payload listed
	// fewer than all 3 tiers) — the tier engine's actual footprint.
	var tierRestricted int
	if err := db.QueryRowContext(ctx, `
		SELECT count(DISTINCT signal_id)::int FROM licensing.edge_signal_queue
		 WHERE created_at > now() - interval '24 hours'
		   AND payload ? 'StrategyID'
		   AND jsonb_typeof(payload->'EligibleTiers') = 'array'
		   AND jsonb_array_length(payload->'EligibleTiers') < 3`).
		Scan(&tierRestricted); err != nil {
		tierRestricted = -1
	}
	stats["tier_restricted_24h"] = tierRestricted
	resp["stats_24h"] = stats

	// ── 5. Engine liveness ───────────────────────────────────────────────
	var lastSignalAge float64
	if err := db.QueryRowContext(ctx, `
		SELECT COALESCE(EXTRACT(EPOCH FROM (now() - min(created_at))), -1)::float8
		  FROM licensing.edge_signal_queue
		 WHERE payload ? 'StrategyID'
		   AND created_at > now() - interval '48 hours'`).
		Scan(&lastSignalAge); err != nil {
		lastSignalAge = -1
	}
	resp["last_signal_age_seconds"] = lastSignalAge

	json.NewEncoder(w).Encode(resp)
}
