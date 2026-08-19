package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/predictatrade/realtime/internal/cache"
	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/marketdata"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// HTTPServer serves REST API, health checks, and metrics.
// Binds to 127.0.0.1 — Nginx is the public ingress.
type HTTPServer struct {
	hub       *WebSocketHub
	persister *marketdata.Persister
	states    *features.StateManager
	agentHub  *AgentHub
	agentProvider interface{ GetLastSnapshot() interface{}; GetSnapshotCount() uint64; HasConnectedAgents() bool }
	valkeyCache *cache.ValkeyCache
	mux       *http.ServeMux
	server    *http.Server
}

func NewHTTPServer(hub *WebSocketHub, persister *marketdata.Persister, states *features.StateManager, agentHub *AgentHub, agentProvider interface{ GetLastSnapshot() interface{}; GetSnapshotCount() uint64; HasConnectedAgents() bool }, valkeyCache *cache.ValkeyCache) *HTTPServer {
	h := &HTTPServer{
		agentHub: agentHub,
		agentProvider: agentProvider,
		valkeyCache: valkeyCache,
		hub:       hub,
		persister: persister,
		states:    states,
		mux:       http.NewServeMux(),
	}
	h.registerRoutes()
	return h
}

func (h *HTTPServer) registerRoutes() {
	h.mux.HandleFunc("/health", h.handleHealth)
	h.mux.HandleFunc("/ready", h.handleReady)
	h.mux.HandleFunc("/metrics", promhttp.Handler().ServeHTTP)
	// WebSocket at /ws and /ws/v1 (canonical production path)
	h.mux.HandleFunc("/ws", h.hub.HandleWebSocket)
	h.mux.HandleFunc("/ws/v1", h.hub.HandleWebSocket)
	h.mux.HandleFunc("/ws/v1/agent", h.agentHub.HandleAgentWebSocket)
	h.mux.HandleFunc("/ws/agent", h.agentHub.HandleAgentWebSocket)
	h.mux.HandleFunc("/api/v1/signals", h.handleSignals)
	h.mux.HandleFunc("/api/v1/market/state", h.handleMarketState)
	h.mux.HandleFunc("/api/v1/candles", h.handleCandles)
	h.mux.HandleFunc("/api/v1/strategies", h.handleStrategies)
	h.mux.HandleFunc("/api/v1/market/snapshot", h.handleMarketSnapshot)
	h.mux.HandleFunc("/api/v1/agents/status", h.handleAgentsStatus)
	h.mux.HandleFunc("/api/v1/price/history", h.handlePriceHistory)
	h.mux.HandleFunc("/api/v1/signals/resume", h.handleSignalResume)
	h.mux.HandleFunc("/api/v1/admin/regime-diagnostics", h.handleRegimeDiagnostics)
	h.mux.HandleFunc("/api/v1/system-health", h.handleSystemHealth)
}

func (h *HTTPServer) Start(host string, port int) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	h.server = &http.Server{
		Addr:         addr,
		Handler:      h.mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return h.server.ListenAndServe()
}

func (h *HTTPServer) Shutdown(ctx context.Context) error {
	if h.server != nil {
		return h.server.Shutdown(ctx)
	}
	return nil
}

func (h *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "ok",
		"timestamp":  time.Now().UTC(),
		"service":    "realtime-engine",
		"version":    "1.0.0",
		"ws_clients":    h.hub.ClientCount(),
		"agents":       h.agentHub.AgentCount(),
	})
}

func (h *HTTPServer) handleReady(w http.ResponseWriter, r *http.Request) {
	ready := true
	reason := ""
	if h.persister != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := h.persister.HealthCheck(ctx); err != nil {
			ready = false
			reason = "database_unavailable"
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if ready {
		json.NewEncoder(w).Encode(map[string]interface{}{"ready": true})
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{"ready": false, "reason": reason})
	}
}

func (h *HTTPServer) handleSignals(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h.persister == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"signals": []interface{}{}, "note": "no_database"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	signals, err := h.persister.GetRecentSignals(ctx, 50)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	if signals == nil {
		signals = []*types.Signal{}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"signals": signals})
}

func (h *HTTPServer) handleMarketState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Read from Valkey hot cache first (sub-ms latency)
	if h.valkeyCache != nil {
		if data, err := h.valkeyCache.GetMarketState(); err == nil && len(data) > 0 {
			w.Write(data)
			return
		}
	}
	// Fallback to in-memory state
	states := h.states.GetAll()
	if states == nil {
		states = []*features.MarketState{}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"states": states})
}

func (h *HTTPServer) handlePriceHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h.valkeyCache != nil {
		points, err := h.valkeyCache.GetPriceHistory()
		if err == nil && len(points) > 0 {
			json.NewEncoder(w).Encode(map[string]interface{}{"prices": points})
			return
		}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"prices": []interface{}{}})
}

func (h *HTTPServer) handleCandles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		symbol = "XAUUSD"
	}
	timeframe := r.URL.Query().Get("timeframe")
	if timeframe == "" {
		timeframe = "M5"
	}
	if h.persister == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"candles": []interface{}{}, "note": "no_database"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	candles, err := h.persister.GetRecentCandles(ctx, symbol, timeframe, 200)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"candles": []interface{}{}, "error": err.Error()})
		return
	}
	if candles == nil {
		candles = []*types.Candle{}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"candles": candles})
}

func (h *HTTPServer) handleStrategies(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"strategies": types.AllStrategies(),
	})
}

func (h *HTTPServer) handleMarketSnapshot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Read from Valkey hot cache first
	if h.valkeyCache != nil {
		if data, err := h.valkeyCache.GetSnapshot(); err == nil && len(data) > 0 {
			w.Write(data)
			return
		}
	}
	// Fallback to agent provider
	if h.agentProvider != nil {
		snapshot := h.agentProvider.GetLastSnapshot()
		if snapshot != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"snapshot": snapshot,
				"count":    h.agentProvider.GetSnapshotCount(),
			})
			return
		}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"snapshot": nil,
		"status":   "waiting",
		"message":  "No Master Node snapshot received yet. Ensure Master Node EA is running and connected to Windows Agent.",
		"timestamp": time.Now().UTC(),
	})
}

func (h *HTTPServer) handleAgentsStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Always use LIVE data for agent connection status.
	// Agent connection state is real-time — a stale Valkey cache entry
	// (written by another process or from before a reconnect) must NOT
	// override the authoritative in-memory agentHub count.
	// Valkey cache is still written by the broadcast loop for other
	// consumers (e.g. NestJS health aggregation), but the Go engine's
	// own endpoint must reflect the truth it holds in memory.
	agentsConnected := h.agentHub.AgentCount()
	masterNodeConnected := false
	snapshotCount := uint64(0)
	if h.agentProvider != nil {
		masterNodeConnected = h.agentProvider.HasConnectedAgents()
		snapshotCount = h.agentProvider.GetSnapshotCount()
	}
	// If the live agentHub reports a connection but the provider has not
	// yet registered it (race during initial handshake), trust the hub —
	// a connected WebSocket agent IS a connected master node.
	if agentsConnected > 0 && !masterNodeConnected {
		masterNodeConnected = true
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agents_connected":      agentsConnected,
		"master_node_connected": masterNodeConnected,
		"snapshot_count":        snapshotCount,
		"timestamp":             time.Now().UTC(),
	})
}

// handleSignalResume handles client reconnect: returns missed signals for replay.
// SOW Section 29, 47: Signal Sequence Resume Protocol.
// Client provides last_acked_sequence, server returns still-valid unacked signals.
func (h *HTTPServer) handleSignalResume(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")
	lastAckSeqStr := r.URL.Query().Get("last_acked_sequence")
	if deviceID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "device_id required"})
		return
	}
	
	// Parse last acknowledged sequence
	lastAckSeq := int64(0)
	if lastAckSeqStr != "" {
		if v, err := strconv.ParseInt(lastAckSeqStr, 10, 64); err == nil {
			lastAckSeq = v
		}
	}

	w.Header().Set("Content-Type", "application/json")
	
	// If no persister, return empty (no signals to replay)
	if h.persister == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"device_id":          deviceID,
			"last_acked_sequence": lastAckSeq,
			"missed_signals":      []interface{}{},
			"server_time":         time.Now().UTC().Format(time.RFC3339),
			"message":             "no_persistence",
		})
		return
	}

	// Query for missed signals from signal_deliveries table
	// Only return signals that are: still valid, not expired, not executed, not cancelled
	rows, err := h.persister.GetDB().QueryContext(r.Context(), `
		SELECT sd.signal_id, sd.sequence_number, s.direction, s.entry_price, s.stop_loss,
		       s.tp1, s.tp2, s.tp3, s.strategy_id, s.status, s.created_at, s.expires_at
		FROM trading.signal_deliveries sd
		JOIN trading.signals s ON s.id = sd.signal_id
		WHERE sd.device_id = $1
		  AND sd.sequence_number > $2
		  AND sd.delivery_state IN ('SENT', 'DELIVERED', 'QUEUED')
		  AND s.expires_at > now()
		  AND s.direction IN ('BUY', 'SELL')
		  AND s.status IN ('CONFIRMED', 'ACTIVE')
		ORDER BY sd.sequence_number ASC
		LIMIT 10
	`, deviceID, lastAckSeq)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "query_failed"})
		return
	}
	defer rows.Close()

	var missedSignals []map[string]interface{}
	for rows.Next() {
		var signalID, direction, entryPrice, stopLoss, tp1, tp2, tp3, strategyID, status string
		var seqNum int64
		var createdAt, expiresAt time.Time
		if err := rows.Scan(&signalID, &seqNum, &direction, &entryPrice, &stopLoss,
			&tp1, &tp2, &tp3, &strategyID, &status, &createdAt, &expiresAt); err != nil {
			continue
		}
		missedSignals = append(missedSignals, map[string]interface{}{
			"signal_id":      signalID,
			"sequence":       seqNum,
			"direction":      direction,
			"entry":          entryPrice,
			"stop_loss":      stopLoss,
			"tp1":            tp1,
			"tp2":            tp2,
			"tp3":            tp3,
			"strategy":       strategyID,
			"status":         status,
			"created_at":     createdAt,
			"expires_at":     expiresAt,
		})
	}

	if missedSignals == nil {
		missedSignals = []map[string]interface{}{}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"device_id":          deviceID,
		"last_acked_sequence": lastAckSeq,
		"missed_signals":      missedSignals,
		"server_time":         time.Now().UTC().Format(time.RFC3339),
		"count":               len(missedSignals),
	})
}

// handleRegimeDiagnostics returns admin-only regime diagnostics.
// SOW Phase 2 Section 6-7: Regime transition telemetry and admin diagnostics.
func (h *HTTPServer) handleRegimeDiagnostics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	state := h.states.Get(types.SymbolXAUUSD)
	if state == nil || state.Timestamp.IsZero() {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "NO_DATA",
			"message": "No market state available",
		})
		return
	}

	rsi, _ := state.Indicators.RSI.Float64()
	adx, _ := state.Indicators.ADX.Float64()
	atr, _ := state.Indicators.ATR.Float64()

	diag := map[string]interface{}{
		"status":            "OK",
		"symbol":            state.Symbol,
		"timestamp":         state.Timestamp,
		"current_regime":    string(state.Regime.Current),
		"previous_regime":  string(state.Regime.Previous),
		"regime_age":        state.Regime.Age.String(),
		"confidence":        state.Regime.Confidence,
		"entry_reason":      state.Regime.EntryReason,
		"raw_regime":        string(state.Regime.RawRegime),
		"entered_at":        state.Regime.EnteredAt,
		"current_rsi":       rsi,
		"current_adx":       adx,
		"current_atr":       atr,
		"volatility":        state.Regime.Volatility,
		"hold_reason":       state.Regime.HoldReason,
		"regime_engine_version": state.Regime.RegimeEngineVersion,
	}

	if state.Regime.TransitionCandidate != nil {
		diag["transition_candidate"] = string(*state.Regime.TransitionCandidate)
		diag["transition_confidence"] = state.Regime.TransitionConfidence
		diag["confirmation_count"] = state.Regime.ConfirmationCount
		diag["required_confirmations"] = state.Regime.RequiredConfirmations
	}

	// EMA alignment
	emaAlignment := "NEUTRAL"
	if state.Indicators.EMA9.GreaterThan(state.Indicators.EMA21) && state.Indicators.EMA21.GreaterThan(state.Indicators.EMA50) {
		emaAlignment = "BULLISH"
	} else if state.Indicators.EMA9.LessThan(state.Indicators.EMA21) && state.Indicators.EMA21.LessThan(state.Indicators.EMA50) {
		emaAlignment = "BEARISH"
	}
	diag["ema_alignment"] = emaAlignment

	// Structure state
	diag["structure_trend"] = state.Structure.CurrentTrend
	if state.Structure.LastBOS != nil {
		diag["last_bos_direction"] = state.Structure.LastBOS.Direction
	}

	json.NewEncoder(w).Encode(diag)
}


// handleSystemHealth provides a comprehensive system health check (prompt.md Section 86)
func (h *HTTPServer) handleSystemHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	health := map[string]interface{}{
		"timestamp": time.Now().UTC(),
	}
	
	// PostgreSQL
	dbConnected := false
	dbHealthy := false
	if h.persister != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		dbConnected = h.persister.HealthCheck(ctx) == nil
		dbHealthy = dbConnected
		cancel()
	}
	health["postgresql"] = map[string]interface{}{
		"connected": dbConnected,
		"healthy":   dbHealthy,
	}
	
	// TimescaleDB
	timescaleActive := false
	if dbConnected {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		var extCount int
		_ = h.persister.GetDB().QueryRowContext(ctx, 
			"SELECT count(*) FROM pg_extension WHERE extname = 'timescaledb'").Scan(&extCount)
		timescaleActive = extCount > 0
		cancel()
	}
	health["timescaledb"] = map[string]interface{}{
		"active": timescaleActive,
	}
	
	// Valkey
	valkeyConnected := h.valkeyCache != nil
	health["valkey"] = map[string]interface{}{
		"connected": valkeyConnected,
	}
	
	// Market source
	health["market_source"] = map[string]interface{}{
		"agents_connected":     h.agentHub.AgentCount(),
		"master_node_connected": h.agentProvider.HasConnectedAgents(),
	}
	
	// Overall ready
	health["ready"] = dbConnected && dbHealthy
	health["ready_reason"] = ""
	if !dbConnected {
		health["ready_reason"] = "database_unavailable"
	}
	
	json.NewEncoder(w).Encode(health)
}
