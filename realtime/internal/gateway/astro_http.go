package gateway

import (
	"encoding/json"
	"net/http"

	"github.com/predictatrade/realtime/internal/astro"
)

// Astro Intelligence HTTP handlers (check.md 2026-08-30: Vedic + Western
// Astro-Financial Intelligence Engine). All routes require authentication
// upstream; ELITE gating enforced on the frontend + delivery side.
func (h *HTTPServer) handleAstroState(w http.ResponseWriter, r *http.Request) {
	astro.HandleBuildState(w, r, func(w2 http.ResponseWriter, v any) {
		_ = json.NewEncoder(w2).Encode(v)
	})
}

func (h *HTTPServer) handleAstroMindMap(w http.ResponseWriter, r *http.Request) {
	astro.HandleMindMap(w, r)
}

func (h *HTTPServer) handleAstroScreens(w http.ResponseWriter, r *http.Request) {
	astro.HandleScreens(w, r)
}
