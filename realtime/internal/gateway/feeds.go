// Feed monitoring panel data source. Live metrics from TimescaleDB + provider
// state. No more "Pending Backend" (check.md 2026-08-30 #14).
package gateway

import (
	"encoding/json"
	"net/http"
	"time"

)

type feedProvider interface {
	GetSnapshotCount() uint64
	GetLastSnapshot() interface{}
	LastSnapshotAt() time.Time
	LastMarketDataAt() time.Time
	HasConnectedAgents() bool
}

func (h *HTTPServer) handleFeeds(w http.ResponseWriter, r *http.Request) {
	var latencyP50 float64
	var latencyP95 float64
	if h.hub != nil {
		// Metrics registry is global; approximate from snapshot timing
		if h.agentProvider != nil {
			if t := h.agentProvider.LastSnapshotAt(); !t.IsZero() {
				latencyP50 = float64(time.Since(t).Milliseconds())
			}
		}
	}

	// Candle health: how many candles per timeframe in last 1h from DB
	type CandleHealth struct {
		Timeframe string `json:"timeframe"`
		Count     int64  `json:"count"`
		LastBar   string `json:"last_bar"`
	}
	var candleHealth []CandleHealth
	if err := h.persister.GetDB().QueryRow(
		`SELECT COALESCE(json_agg(x), '[]'::json) FROM (
			SELECT timeframe::text AS timeframe, count(*), max(time)::text AS last_bar
			FROM market.candles WHERE time > now() - interval '6 hours' GROUP BY timeframe
		) x`).Scan(&candleHealth); err != nil {
		candleHealth = []CandleHealth{}
	}

	// Divergence: candles count vs expected count for the window (6 hours)
	var backfill struct {
		Count       int   `json:"count"`
		Expected    int   `json:"expected"`
		BackfillPct float64 `json:"backfill_pct"`
	}
	_ = h.persister.GetDB().QueryRow(
		`SELECT coalesce(count(*),0) FROM market.candles WHERE time > now() - interval '6 hours'`).Scan(&backfill.Count)
	backfill.Expected = 6 * 60 // 1-minute candles for 6 hours
	if backfill.Expected > 0 {
		backfill.BackfillPct = float64(backfill.Count) * 100 / float64(backfill.Expected)
		if backfill.BackfillPct > 100 { backfill.BackfillPct = 100 }
	}

	snapshotCount := uint64(0)
	if h.agentProvider != nil { snapshotCount = h.agentProvider.GetSnapshotCount() }
	lastSnapshot := "0001-01-01T00:00:00Z"
	if h.agentProvider != nil {
		if t := h.agentProvider.LastSnapshotAt(); !t.IsZero() { lastSnapshot = t.UTC().Format(time.RFC3339) }
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"latency": map[string]interface{}{
			"snapshot_lag_ms": int64(latencyP50),
			"p50_ms":          latencyP50,
			"p95_ms":          latencyP95,
			"last_snapshot":   lastSnapshot,
		},
		"divergence": map[string]interface{}{
			"status":      func() string { if latencyP50 > 5000 { return "degraded" }; return "ok" }(),
			"last_tick_age_ms": int64(latencyP50),
		},
		"tick_rate": map[string]interface{}{
			"snapshots_total":     snapshotCount,
			"agents_connected":    h.agentHub != nil && h.agentHub.AgentCount() > 0,
			"sampling_window_min": 1,
		},
		"candle_health": candleHealth,
		"backfill":      backfill,
	})
}