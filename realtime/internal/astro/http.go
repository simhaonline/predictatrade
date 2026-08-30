package astro

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// HTTP JSON handlers — check.md: "Signal Screens + Interactive Mind Maps"
// These endpoints serve the Astro dashboard (ELITE-only enforcement done by
// the gateway/admin auth layer; callers without ELITE get the reduced public
// fields from Compute()).
type JSONHandler struct{}

// HandleState returns the full Astro state for ?ts= (default now).
// Response: composite score, Vedic section, Western section, factors,
// apocalypse trigger if any.
func HandleBuildState(w http.ResponseWriter, r *http.Request, writeJSON func(w http.ResponseWriter, v any)) {
	t := time.Now().UTC()
	if qs := r.URL.Query().Get("ts"); qs != "" {
		if parsed, err := time.Parse(time.RFC3339, qs); err == nil {
			t = parsed
		}
	}
	marketClosed := r.URL.Query().Get("market_closed") == "true"
	st := Compute(t, marketClosed)
	writeJSON(w, st)
}

// HandleMindMap returns an interactive mind-map graph: nodes = DI/Western
// factors, edges = contribution weights. Used by the Astro dashboard screen
// to render the decision tree visually.
func HandleMindMap(w http.ResponseWriter, r *http.Request) {
	t := time.Now().UTC()
	st := Compute(t, r.URL.Query().Get("market_closed") == "true")
	root := map[string]interface{}{
		"name":  "ASTRO",
		"score": st.CompositeScore,
		"children": []map[string]interface{}{
			{
				"name": "Vedic DI",
				"children": []map[string]interface{}{
					{"name": "Nakshatra", "value": st.Vedic.NakshatraName + " pada " + fmtInt(st.Vedic.Pada), "score": st.Vedic.NakshatraBias},
					{"name": "Hora", "value": st.Vedic.HoraLord, "score": st.Vedic.HoraBias},
					{"name": "Dasha L1", "value": st.Vedic.DashaL1, "score": st.Vedic.DashaL1Bias},
					{"name": "Dasha L2", "value": st.Vedic.DashaL2, "score": st.Vedic.DashaL2Bias},
				},
			},
			{
				"name": "Western Tropical",
				"children": []map[string]interface{}{
					{"name": "Total Aspects", "value": st.Western.TotalScore, "count": len(st.Western.Aspects)},
					{"name": "Mercury Rx", "value": st.Western.IsRetrograde["Mercury"]},
				},
			},
		},
	}
	writeJSON(w, root)
}

// HandleScreens returns per-signal-screen breakdown (one entry per factor
// with label, score, weight, detail) for the "Signal Screens" display.
func HandleScreens(w http.ResponseWriter, r *http.Request) {
	t := time.Now().UTC()
	st := Compute(t, r.URL.Query().Get("market_closed") == "true")
	factors := make([]FactorContribution, 0, len(st.Factors))
	for _, f := range st.Factors {
		factors = append(factors, f)
	}
	writeJSON(w, map[string]interface{}{
		"composite_score": st.CompositeScore,
		"eligible":        st.EligibleForTrade,
		"reason":          st.IneligibleReason,
		"factors":         factors,
		"vedic":           st.Vedic,
		"western":         st.Western,
		"apocalypse":      st.Apocalypse,
	})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	_ = enc.Encode(v)
}

func fmtInt(i int) string { return fmt.Sprintf("%d", i) }
