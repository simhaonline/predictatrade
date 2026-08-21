package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// healthEndpoint exposes a minimal local HTTP server so that external
// monitors (health-check.ps1, NSSM, or operators) can determine whether
// the agent process is alive and responsive without connecting to the
// backend or the MT4/MT5 terminal.
//
// The endpoint listens on 127.0.0.1:9000 by default. The port can be
// overridden via the PAT_HEALTH_PORT environment variable.
//
// GET /health  →  200 OK with JSON body {"status":"ok","uptime_seconds":N}
// GET /        →  200 OK (same as /health)
//
// This is a strictly additive feature — it does not interfere with the
// existing WebSocket, pipe, or heartbeat subsystems.

const defaultHealthPort = "9000"

// healthServer wraps the net/http server so we can shut it down cleanly.
type healthServer struct {
	srv      *http.Server
	startTime time.Time
}

// newHealthServer creates a health endpoint bound to localhost only.
func newHealthServer() *healthServer {
	port := os.Getenv("PAT_HEALTH_PORT")
	if port == "" {
		port = defaultHealthPort
	}

	mux := http.NewServeMux()
	startTime := time.Now()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":         "ok",
			"uptime_seconds":  int(time.Since(startTime).Seconds()),
			"agent_version":  AgentVersion,
			"timestamp":       time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Root path also returns health (convenience for curl)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":         "ok",
			"uptime_seconds":  int(time.Since(startTime).Seconds()),
			"agent_version":  AgentVersion,
			"timestamp":       time.Now().UTC().Format(time.RFC3339),
		})
	})

	return &healthServer{
		srv: &http.Server{
			Addr:         fmt.Sprintf("127.0.0.1:%s", port),
			Handler:      mux,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		},
		startTime: startTime,
	}
}

// start launches the health server in a background goroutine.
func (h *healthServer) start() {
	go func() {
		log.Printf("[health] Local health endpoint listening on http://%s/health", h.srv.Addr)
		if err := h.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[health] WARNING: health endpoint failed: %v", err)
		}
	}()
}

// stop gracefully shuts down the health server.
func (h *healthServer) stop() {
	if h != nil && h.srv != nil {
		_ = h.srv.Close()
	}
}
