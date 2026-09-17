// indicators_raw.go — P0-3 (prompt.md): raw indicator-read surface.
//
// GET /api/v1/indicators/raw?symbol=XAUUSD[&names=rsi,adx]
//
// Returns the LIVE MarketState indicator read set via the exact same
// builder (marketdata.BuildFeatureSnapshotJSON) used for per-signal
// feature snapshots — guaranteeing endpoint == snapshot parity.
package gateway

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/marketdata"
	"github.com/predictatrade/realtime/internal/observability"
)

func (h *HTTPServer) handleIndicatorsRaw(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	symbol := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("symbol")))
	if symbol == "" {
		http.Error(w, `{"error":"symbol query param required"}`, http.StatusBadRequest)
		return
	}
	var st *features.MarketState
	if h.states != nil {
		st = h.states.Get(symbol)
	}
	if st == nil {
		http.Error(w, `{"error":"unknown symbol"}`, http.StatusNotFound)
		return
	}
	// P2: enrich with astro + crossmarket reads (same values the per-signal
	// snapshots carry). Provider injected from main.go; nil → plain reads.
	if h.enrichIndicators != nil {
		h.enrichIndicators(st)
	}
	b, err := marketdata.BuildFeatureSnapshotJSON(st)
	if err != nil {
		observability.Log.Warn().Err(err).Str("symbol", symbol).Msg("indicators/raw build failed")
		http.Error(w, `{"error":"build failed"}`, http.StatusInternalServerError)
		return
	}
	// Optional subset filter: names=rsi,adx
	if names := strings.TrimSpace(r.URL.Query().Get("names")); names != "" {
		var full map[string]any
		if json.Unmarshal(b, &full) == nil {
			sub := map[string]any{"_schema_version": full["_schema_version"], "symbol": symbol}
			for _, n := range strings.Split(names, ",") {
				n = strings.ToLower(strings.TrimSpace(n))
				if v, ok := full[n]; ok {
					sub[n] = v
				}
			}
			_ = json.NewEncoder(w).Encode(sub)
			return
		}
	}
	_, _ = w.Write(b)
}
