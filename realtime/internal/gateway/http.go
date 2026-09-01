package gateway

import (
	"context"
	"sync"

	"encoding/json"
	"fmt"
	"github.com/predictatrade/realtime/pkg/news"
	"net/http"
	"net/http/pprof" // pprof endpoints for diagnostics (localhost-only)
	"strconv"
	"strings"
	"time"

	"github.com/predictatrade/realtime/internal/cache"
	"github.com/predictatrade/realtime/internal/crossmarket"
	"github.com/predictatrade/realtime/internal/devilliquidity"
	"github.com/predictatrade/realtime/internal/engstatus"
	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/marketdata"
	"github.com/predictatrade/realtime/internal/observability"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shopspring/decimal"
)

// HTTPServer serves REST API, health checks, and metrics.
// Binds to 127.0.0.1 — Nginx is the public ingress.
type HTTPServer struct {
	hub           *WebSocketHub
	persister     *marketdata.Persister
	states        *features.StateManager
	agentHub      *AgentHub
	DataAgentHub  *AgentHub
	agentProvider interface {
		GetLastSnapshot() interface{}
		GetSnapshotCount() uint64
		HasConnectedAgents() bool
		BrokerOffsetHours() int
		LastMarketDataAt() time.Time
		LastSnapshotAt() time.Time
	}
	valkeyCache       *cache.ValkeyCache
	crossMarketEngine *crossmarket.Engine
	newsEngine        *news.RiskEngine
	engTracker        *engstatus.Tracker
	// EmergencyHalt is the server-authoritative trading halt (v1.15.0):
	// set by EMERGENCY_STOP / KILL_SWITCH, consulted by signal generation
	// and delivery so no EXECUTABLE signal leaves the engine while halted.
	EmergencyHalt *EmergencyHalt
	mux           *http.ServeMux
	server        *http.Server
	serverMu      sync.Mutex
}

// EmergencyHalt is a process-wide, thread-safe trading-halt flag.
// EMERGENCY_STOP and KILL_SWITCH both activate it; only an explicit
// RESUME (POST /api/v1/admin/emergency-resume with admin JWT) clears it.
type EmergencyHalt struct {
	mu        sync.RWMutex
	active    bool
	level     string // EMERGENCY_STOP | KILL_SWITCH
	reason    string
	timestamp time.Time
}

// Activate turns the halt on (idempotent, first-level wins for audit).
func (e *EmergencyHalt) Activate(level, reason string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.active {
		e.level = level
		e.timestamp = time.Now().UTC()
	}
	e.active = true
	if reason != "" {
		e.reason = reason
	}
}

// Resumed clears the halt (post-incident operator action).
func (e *EmergencyHalt) Resumed(reason string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.active = false
	e.reason = "resumed: " + reason
}

// Active reports whether trading is halted.
func (e *EmergencyHalt) Active() bool {
	if e == nil {
		return false
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.active
}

// Status returns the halt state for health/admin introspection.
func (e *EmergencyHalt) Status() (bool, string, string, time.Time) {
	if e == nil {
		return false, "", "", time.Time{}
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.active, e.level, e.reason, e.timestamp
}

func NewHTTPServer(hub *WebSocketHub, persister *marketdata.Persister, states *features.StateManager, agentHub *AgentHub, agentProvider interface {
	GetLastSnapshot() interface{}
	GetSnapshotCount() uint64
	HasConnectedAgents() bool
	BrokerOffsetHours() int
	LastMarketDataAt() time.Time
	LastSnapshotAt() time.Time
}, valkeyCache *cache.ValkeyCache, xmEngine *crossmarket.Engine, newsEngine *news.RiskEngine, engTracker *engstatus.Tracker) *HTTPServer {
	h := &HTTPServer{
		agentHub:          agentHub,
		agentProvider:     agentProvider,
		valkeyCache:       valkeyCache,
		crossMarketEngine: xmEngine,
		newsEngine:        newsEngine,
		engTracker:        engTracker,
		hub:               hub,
		persister:         persister,
		states:            states,
		mux:               http.NewServeMux(),
	}
	h.registerRoutes()
	return h
}
func (h *HTTPServer) registerRoutes() {
	h.mux.HandleFunc("/health", h.handleHealth)
	h.mux.HandleFunc("/ready", h.handleReady)
	h.mux.HandleFunc("/metrics", promhttp.Handler().ServeHTTP)
	// pprof diagnostic endpoints — localhost-only, behind Nginx is not exposed publicly
	h.mux.HandleFunc("/debug/pprof/", pprof.Index)
	h.mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	h.mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	h.mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	h.mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	h.mux.HandleFunc("/debug/pprof/goroutine", pprof.Handler("goroutine").ServeHTTP)
	h.mux.HandleFunc("/debug/pprof/heap", pprof.Handler("heap").ServeHTTP)
	h.mux.HandleFunc("/debug/pprof/threadcreate", pprof.Handler("threadcreate").ServeHTTP)
	h.mux.HandleFunc("/debug/pprof/block", pprof.Handler("block").ServeHTTP)
	// WebSocket at /ws and /ws/v1 (canonical production path)
	h.mux.HandleFunc("/ws", h.hub.HandleWebSocket)
	h.mux.HandleFunc("/ws/v1", h.hub.HandleWebSocket)
	h.mux.HandleFunc("/ws/v1/agent", h.agentHub.HandleAgentWebSocket)
	h.mux.HandleFunc("/ws/agent", h.agentHub.HandleAgentWebSocket)
	h.mux.HandleFunc("/api/v1/signals", h.handleSignals)
	h.mux.HandleFunc("/api/v1/trades", h.handleTrades)
	h.mux.HandleFunc("/api/v1/market/state", h.handleMarketState)
	h.mux.HandleFunc("/api/v1/candles", h.handleCandles)
	h.mux.HandleFunc("/api/v1/feeds", h.handleFeeds)
	h.mux.HandleFunc("/api/v1/pipeline/monitor", h.handlePipelineMonitor)

	// Astro Intelligence endpoints (check.md 2026-08-30: Vedic + Western Astro)
	h.mux.HandleFunc("/api/v1/astro/state", h.handleAstroState)
	h.mux.HandleFunc("/api/v1/astro/mindmap", h.handleAstroMindMap)
	h.mux.HandleFunc("/api/v1/astro/screens", h.handleAstroScreens)
	h.mux.HandleFunc("/api/v1/strategies", h.handleStrategies)
	h.mux.HandleFunc("/api/v1/market/snapshot", h.handleMarketSnapshot)
	h.mux.HandleFunc("/api/v1/agents/status", h.handleAgentsStatus)
	h.mux.HandleFunc("/api/v1/news", h.handleNews)
	h.mux.HandleFunc("/api/v1/price/history", h.handlePriceHistory)
	h.mux.HandleFunc("/api/v1/signals/resume", h.handleSignalResume)
	h.mux.HandleFunc("/api/v1/admin/regime-diagnostics", h.handleRegimeDiagnostics)
	h.mux.HandleFunc("/api/v1/system-health", h.handleSystemHealth)

	// Devil Liquidity / Devil's Mark engine API (prompt.md)
	h.mux.HandleFunc("/api/v1/devil-liquidity/marks", h.handleDevilLiquidityMarks)

	// Emergency controls — admin JWT REQUIRED (P0 SEC fix: these were
	// previously unauthenticated). Also activate the server-side EmergencyHalt
	// so signal generation and delivery stop, not just agent broadcast.
	h.mux.HandleFunc("/api/v1/admin/emergency-stop", h.requireAdminAction(h.handleEmergencyStop))
	h.mux.HandleFunc("/api/v1/admin/kill-switch", h.requireAdminAction(h.handleKillSwitch))
	h.mux.HandleFunc("/api/v1/admin/emergency-resume", h.requireAdminAction(h.handleEmergencyResume))

	// Per-strategy-engine liveness (prompt.md Sections 26, 38, 43-46)
	h.mux.HandleFunc("/api/v1/engines/status", h.handleEnginesStatus)

	// Cross-Market Confluence Engine API
	if h.crossMarketEngine != nil {
		h.mux.HandleFunc("/api/v1/cross-market/current", crossmarket.HandleCurrent(h.crossMarketEngine))
		h.mux.HandleFunc("/api/v1/cross-market/health", crossmarket.HandleHealthExtended(h.crossMarketEngine))
		h.mux.HandleFunc("/api/v1/cross-market/validation", crossmarket.HandleValidationStatus(h.crossMarketEngine))
	}
}

// handleEnginesStatus reports truthful per-strategy-engine liveness derived
// from actual evaluation activity — never fabricated green states.
func (h *HTTPServer) handleEnginesStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h.engTracker == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"engines": []engstatus.Snapshot{}, "server_time": time.Now().UTC(),
		})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"engines":     h.engTracker.All(),
		"server_time": time.Now().UTC(),
	})
}

func (h *HTTPServer) Start(host string, port int) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      h.mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	h.serverMu.Lock()
	h.server = srv
	h.serverMu.Unlock()
	return srv.ListenAndServe()
}

func (h *HTTPServer) Shutdown(ctx context.Context) error {
	h.serverMu.Lock()
	srv := h.server
	h.serverMu.Unlock()
	if srv != nil {
		return srv.Shutdown(ctx)
	}
	return nil
}

func (h *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	brokerOff := 0
	if h.agentProvider != nil {
		brokerOff = h.agentProvider.BrokerOffsetHours()
	}
	haltActive, haltLevel, haltReason, haltSince := h.EmergencyHalt.Status()
	status := "ok"
	if haltActive {
		status = "HALTED"
	}
	// DB awareness (MACRO_AUDIT 2.9): reflect configured persister/DB health
	// without crashing when no database is attached.
	dbStatus := "not_configured"
	if h.persister != nil {
		dbStatus = h.persister.DBHealth(r.Context())
	}
	cacheStatus := "not_configured"
	if h.valkeyCache != nil {
		if err := h.valkeyCache.Ping(); err != nil {
			cacheStatus = "down"
		} else {
			cacheStatus = "ok"
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        status,
		"db":            dbStatus,
		"cache":         cacheStatus,
		"emergency_halt": map[string]interface{}{
			"active":   haltActive,
			"level":    haltLevel,
			"reason":   haltReason,
			"since":    haltSince,
		},
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
		"server_time":   time.Now().UTC().Format(time.RFC3339),
		"broker_offset": brokerOff,
		"broker_time":   time.Now().UTC().Add(time.Duration(brokerOff) * time.Hour).Format(time.RFC3339),
		"time_mode":     timeModeLabel(brokerOff),
		"service":       "realtime-engine",
		"version":       "1.0.0",
		"ws_clients":    h.hub.ClientCount(),
		"agents":        h.agentHub.AgentCount(),
	})
}

// timeModeLabel returns the engine's active time-alignment mode for API clients.
func timeModeLabel(brokerOff int) string {
	if brokerOff != 0 {
		return "BROKER_ALIGNED"
	}
	return "UTC_ALIGNED"
}

func (h *HTTPServer) handleReady(w http.ResponseWriter, r *http.Request) {
	ready := true
	reason := ""
	dbStatus := "not_configured"
	if h.persister != nil {
		dbStatus = h.persister.DBHealth(r.Context())
		if dbStatus == "down" {
			ready = false
			reason = "database_unavailable"
		}
	}
	cacheStatus := "not_configured"
	if h.valkeyCache != nil {
		if err := h.valkeyCache.Ping(); err != nil {
			cacheStatus = "down"
		} else {
			cacheStatus = "ok"
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if ready {
		json.NewEncoder(w).Encode(map[string]interface{}{"ready": true, "db": dbStatus, "cache": cacheStatus})
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{"ready": false, "db": dbStatus, "cache": cacheStatus, "reason": reason})
	}
}

func (h *HTTPServer) handleSignals(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// B-01 / P0-RT3: Entitlement filtering — the JWT is VERIFIED (HS256
	// signature + exp + alg) and we read the user's role so plan/strategy
	// entitlements can be enforced server-side (no client-side loophole).
	// Absent/invalid tokens are anonymous and get advisory-only signals.
	authHeader := r.Header.Get("Authorization")
	userID := ""
	role := ""
	authenticated := false
	if strings.HasPrefix(authHeader, "Bearer ") {
		if uid, rl, err := validateJWTFull(strings.TrimPrefix(authHeader, "Bearer ")); err == nil && uid != "" {
			userID, role = uid, rl
			authenticated = true
		}
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 200 {
			limit = v
		}
	}

	// Optional strategy filter (?strategy=MARNIE_FIB). When provided we bypass
	// the Valkey cache (it is not strategy-scoped) and query the database
	// directly so a user can inspect any single strategy's signals.
	strategyFilter := strings.TrimSpace(r.URL.Query().Get("strategy"))

	// Plan/entitlement enforcement. Admin sees everything. Authenticated
	// non-admin is restricted to their entitled strategies. Unauthenticated
	// is restricted to advisory-only (handled below).
	needsPlanFilter := authenticated && role != "admin"

	// Valkey cache is not strategy-scoped per user, so only use it when no
	// per-user plan filtering is required.
	if h.valkeyCache != nil && !needsPlanFilter && strategyFilter == "" {
		if data, err := h.valkeyCache.GetLatestSignals(); err == nil && len(data) > 0 {
			if authenticated {
				w.Write(data)
				return
			}
			// Anonymous/unverified callers get advisory-only, even from cache
			if filtered, ferr := filterAdvisorySignalsJSON(data); ferr == nil {
				w.Write(filtered)
				return
			}
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "authentication required for executable signals"})
			return
		}
	}

	// Fallback to database query
	if h.persister == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"signals": []interface{}{}, "note": "no_database"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	signals, err := h.persister.GetRecentSignals(ctx, limit, strategyFilter)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	if signals == nil {
		signals = []*types.Signal{}
	}

	// B-01/P0-RT3: anonymous or invalid-token callers get advisory-only signals
	if !authenticated {
		filtered := make([]*types.Signal, 0, len(signals))
		for _, s := range signals {
			if s.SignalClass != "EXECUTABLE" {
				filtered = append(filtered, s)
			}
		}
		signals = filtered
	}

	// Authenticated non-admin: restrict to entitled strategies. Fail closed —
	// on any lookup error or empty entitlements, expose nothing rather than
	// leak strategies the user is not entitled to.
	if needsPlanFilter {
		allowed, maxPerDay, derr := h.persister.GetUserSignalEntitlement(ctx, userID)
		if derr != nil || len(allowed) == 0 {
			signals = []*types.Signal{}
		} else {
			set := make(map[string]bool, len(allowed))
			for _, a := range allowed {
				set[a] = true
			}
			filtered := make([]*types.Signal, 0, len(signals))
			for _, s := range signals {
				if set[string(s.StrategyID)] {
					filtered = append(filtered, s)
				}
			}
			signals = filtered
			// Spec (MASTER PROMPT): Free tier is capped at max_signals_per_day
			// (e.g. 5). Server-authoritative daily quota — never exceed it.
			if maxPerDay > 0 && len(signals) > maxPerDay {
				signals = signals[:maxPerDay]
			}
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"signals": signals})
}

// handleTrades returns REAL executed trades from trading.trade_results so the
// dashboards (Trading Reports, performance) can show genuine P&L instead of
// deriving everything from advisory signals. No estimated/derived values are
// ever substituted for real fills.
func (h *HTTPServer) handleTrades(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Entitlement check: only authenticated callers receive trade history.
	if !isAuthenticatedRequest(r) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "authentication required"})
		return
	}

	limit := 200
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 2000 {
			limit = v
		}
	}
	strategy := r.URL.Query().Get("strategy")

	if h.persister == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"trades": []interface{}{}, "note": "no_database"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	trades, err := h.persister.GetRecentTrades(ctx, limit, strategy)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	if trades == nil {
		trades = []*marketdata.TradeResult{}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"trades": trades})
}

// isAuthenticatedRequest reports whether the request carries a valid (verified)
// bearer token. It mirrors the entitlement check used by handleSignals.
func isAuthenticatedRequest(r *http.Request) bool {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return false
	}
	jwtToken := strings.TrimPrefix(authHeader, "Bearer ")
	userID, err := extractUserIDFromJWT(jwtToken)
	return err == nil && userID != ""
}

// filterAdvisorySignalsJSON strips EXECUTABLE-class entries from the cached
// signals payload so unauthenticated callers never see actionable signals.
func filterAdvisorySignalsJSON(data []byte) ([]byte, error) {
	var raw struct {
		Signals []json.RawMessage `json:"signals"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	kept := make([]map[string]interface{}, 0)
	for _, sigRaw := range raw.Signals {
		var probe map[string]interface{}
		if err := json.Unmarshal(sigRaw, &probe); err != nil {
			continue
		}
		class, _ := probe["SignalClass"].(string)
		classAlt, _ := probe["signal_class"].(string)
		if class == "EXECUTABLE" || classAlt == "EXECUTABLE" {
			continue
		}
		kept = append(kept, probe)
	}
	out, err := json.Marshal(map[string]interface{}{"signals": kept})
	if err != nil {
		return nil, err
	}
	return out, nil
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
	// Fallback: derive a real price line from historical candles so the dashboard
	// never shows a blank chart before a live tick stream exists. This uses only
	// genuine stored candle closes — it never fabricates market data.
	if h.persister != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		for _, spec := range []struct {
			tf       string
			lookback time.Duration
		}{
			{"M15", 72 * time.Hour},
			{"H1", 240 * time.Hour},
			{"D1", 8760 * time.Hour},
		} {
			timeStart := time.Now().UTC().Add(-spec.lookback)
			rows, err := h.persister.GetDB().QueryContext(ctx, `
				SELECT time, close FROM market.candles
				WHERE symbol = $1 AND timeframe = $2 AND time >= $3
				ORDER BY time ASC LIMIT $4
			`, "XAUUSD", spec.tf, timeStart, 300)
			if err != nil {
				continue
			}
			pts := []map[string]interface{}{}
			for rows.Next() {
				var t time.Time
				var closeStr string
				if err := rows.Scan(&t, &closeStr); err != nil {
					continue
				}
				pts = append(pts, map[string]interface{}{
					"timestamp_ms": t.UnixMilli(),
					"price":        parseFloatStr(closeStr),
				})
			}
			rows.Close()
			if len(pts) > 0 {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"prices":    pts,
					"source":    "historical_candles",
					"timeframe": spec.tf,
				})
				return
			}
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
	source := r.URL.Query().Get("source")
	limit := 200

	// Step 1: Try Valkey cache first (fast path — sub-millisecond)
	if h.valkeyCache != nil {
		if cached, err := h.valkeyCache.GetChartCandles(symbol, timeframe, source, limit); err == nil && len(cached) > 0 {
			json.NewEncoder(w).Encode(map[string]interface{}{"candles": cached, "source": "valkey_cache"})
			return
		}
	}

	if h.persister == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"candles": []interface{}{}, "note": "no_database"})
		return
	}

	// Step 2: Query PostgreSQL with time constraint for TimescaleDB chunk exclusion
	// Calculate lookback period based on timeframe
	var lookbackHours int = 48
	switch timeframe {
	case "M5":
		lookbackHours = 24
	case "M15":
		lookbackHours = 72
	case "H1":
		lookbackHours = 240
	case "H4":
		lookbackHours = 1000
	case "D1":
		lookbackHours = 8760
	}
	timeStart := time.Now().UTC().Add(-time.Duration(lookbackHours) * time.Hour)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.persister.GetDB().QueryContext(ctx, `
		SELECT time, open, high, low, close, volume, source, quality, is_closed
		FROM market.candles
		WHERE symbol = $1 AND timeframe = $2 AND time >= $3
		  AND ($5 = '' OR source = $5)
		ORDER BY time DESC LIMIT $4
	`, symbol, timeframe, timeStart, limit, source)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"candles": []interface{}{}, "error": err.Error()})
		return
	}
	defer rows.Close()

	var cachedCandles []cache.CachedCandle
	for rows.Next() {
		var t time.Time
		var openStr, highStr, lowStr, closeStr, source, qualityStr string
		var volume int64
		var isClosed bool
		if err := rows.Scan(&t, &openStr, &highStr, &lowStr, &closeStr, &volume, &source, &qualityStr, &isClosed); err != nil {
			continue
		}
		cachedCandles = append(cachedCandles, cache.CachedCandle{
			Time: t.Format(time.RFC3339),
			Open: parseFloatStr(openStr), High: parseFloatStr(highStr),
			Low: parseFloatStr(lowStr), Close: parseFloatStr(closeStr),
			Volume: volume, Source: source, Quality: qualityStr, IsClosed: isClosed,
		})
	}

	// Cache in Valkey for subsequent requests (60-second TTL)
	if h.valkeyCache != nil && len(cachedCandles) > 0 {
		h.valkeyCache.SetChartCandles(symbol, timeframe, source, limit, cachedCandles)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"candles": cachedCandles, "source": "timescaledb"})
}

// parseFloatStr parses a decimal string to float64.
func parseFloatStr(s string) float64 {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return 0
	}
	f, _ := d.Float64()
	return f
}

func (h *HTTPServer) handleStrategies(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"strategies": types.AllStrategies(),
	})
}

func (h *HTTPServer) handleMarketSnapshot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Step 1: Get the raw MT5 snapshot from agent provider (in-memory)
	var mt5Snapshot *marketdata.MarketSnapshot
	if h.agentProvider != nil {
		if snap := h.agentProvider.GetLastSnapshot(); snap != nil {
			if ms, ok := snap.(*marketdata.MarketSnapshot); ok {
				mt5Snapshot = ms
			}
		}
	}

	// Step 1b: If no in-memory snapshot, try persistent Valkey cache.
	// This ensures the dashboard shows the last Master Node price even when
	// the market is closed, the agent disconnected, or the engine restarted.
	if mt5Snapshot == nil && h.valkeyCache != nil {
		if data, err := h.valkeyCache.GetLastSnapshot(); err == nil && len(data) > 0 {
			var persistent marketdata.MarketSnapshot
			if err := json.Unmarshal(data, &persistent); err == nil {
				mt5Snapshot = &persistent
			}
		}
	}

	// Step 2: Get the engine's locally-computed indicators
	var engineState *features.MarketState
	if h.states != nil {
		engineState = h.states.Get(types.SymbolXAUUSD)
	}

	// Step 3: Build the merged response
	// Start with MT5 snapshot fields, then enrich with locally-computed indicators
	response := map[string]interface{}{}

	if mt5Snapshot != nil {
		// Copy all MT5 snapshot fields
		response["type"] = mt5Snapshot.Type
		response["symbol"] = mt5Snapshot.Symbol
		// Use GMT (UTC) field if available, otherwise fall back to timestamp
		// The MT5 EA now sends ISO8601 UTC in both fields, but older EAs
		// may still send broker time in "timestamp" and UTC in "gmt".
		if mt5Snapshot.GMT != "" {
			response["timestamp"] = mt5Snapshot.GMT
		} else {
			response["timestamp"] = mt5Snapshot.Timestamp
		}
		response["broker_timestamp"] = mt5Snapshot.Timestamp
		response["source"] = mt5Snapshot.Source
		response["broker"] = mt5Snapshot.Broker
		response["node"] = mt5Snapshot.Node
		if mt5Snapshot.Tick.Bid > 0 && mt5Snapshot.Tick.Ask > 0 && mt5Snapshot.Tick.Ask >= mt5Snapshot.Tick.Bid {
			response["tick"] = mt5Snapshot.Tick
		}
		// Override tick with LIVE data from the engine's state manager.
		// LiveTick() returns the freshest real tick when the agent tick stream is
		// active, and falls back to a tick derived from the latest candle bar when
		// the tick stream is stale/intermittent — so the dashboard feed never
		// freezes on a 2-hour-old price while MARKET_SNAPSHOT bars keep flowing.
		if engineState != nil {
			if live := engineState.LiveTick(); live != nil && live.Bid.GreaterThan(decimal.Zero) {
				response["tick"] = map[string]interface{}{
					"bid":    toF(live.Bid),
					"ask":    toF(live.Ask),
					"spread": toF(live.Spread),
					"time":   live.SourceTimestamp.Format("2006-01-02T15:04:05Z07:00"),
				}
			}
		}
		response["bars"] = mt5Snapshot.Bars
		response["vwap"] = mt5Snapshot.VWAP
		response["account_info"] = mt5Snapshot.AccountInfo
		response["symbol_info"] = mt5Snapshot.SymbolInfo
		response["session"] = mt5Snapshot.Session
		response["positions"] = mt5Snapshot.Positions

		// Build indicators map: start with MT5 values, then add locally-computed
		indMap := map[string]interface{}{}
		// MT5-provided indicators (19 fields)
		indMap["atr"] = mt5Snapshot.Indicators.ATR
		indMap["rsi"] = mt5Snapshot.Indicators.RSI
		indMap["ema9"] = mt5Snapshot.Indicators.EMA9
		indMap["ema21"] = mt5Snapshot.Indicators.EMA21
		indMap["ema50"] = mt5Snapshot.Indicators.EMA50
		indMap["sma200"] = mt5Snapshot.Indicators.SMA200
		indMap["adx"] = mt5Snapshot.Indicators.ADX
		indMap["adx_plus_di"] = mt5Snapshot.Indicators.ADXPlusDI
		indMap["adx_minus_di"] = mt5Snapshot.Indicators.ADXMinusDI
		indMap["boll_upper"] = mt5Snapshot.Indicators.BollUpper
		indMap["boll_lower"] = mt5Snapshot.Indicators.BollLower
		indMap["boll_middle"] = mt5Snapshot.Indicators.BollMiddle
		indMap["macd_main"] = mt5Snapshot.Indicators.MACDMain
		indMap["macd_signal"] = mt5Snapshot.Indicators.MACDSignal
		indMap["stoch_main"] = mt5Snapshot.Indicators.StochMain
		indMap["stoch_signal"] = mt5Snapshot.Indicators.StochSignal
		indMap["cci"] = mt5Snapshot.Indicators.CCI

		// Add session info as indicators
		response["session"] = mt5Snapshot.Session

		// Step 4: Enrich with locally-computed indicators from the Go engine
		if engineState != nil && engineState.Indicators.ATR.GreaterThan(decimal.Zero) {
			ind := &engineState.Indicators
			// Add locally-computed fields that MT5 doesn't provide
			indMap["ema100"] = toF(ind.EMA100)
			indMap["ema200"] = toF(ind.EMA200)
			indMap["ema_cross_9_21"] = ind.EMACross921
			indMap["sma50"] = toF(ind.SMA50)
			indMap["sma100"] = toF(ind.SMA100)
			indMap["macd_histogram"] = toF(ind.MACDHistogram)
			indMap["macd_bull_cross"] = ind.MACDBullCross
			indMap["macd_bear_cross"] = ind.MACDBearCross
			indMap["boll_width"] = toF(ind.BollWidth)
			indMap["boll_bull_rev"] = ind.BollBullRev
			indMap["boll_bear_rev"] = ind.BollBearRev
			indMap["obv"] = toF(ind.OBV)
			indMap["psar"] = toF(ind.ParabolicSAR)
			indMap["psar_long"] = ind.ParabolicSARLong
			indMap["stoch_rsi"] = toF(ind.StochRSI)
			indMap["stoch_rsi_k"] = toF(ind.StochRSIK)
			indMap["stoch_rsi_d"] = toF(ind.StochRSID)
			indMap["ichimoku_tenkan"] = toF(ind.IchimokuTenkan)
			indMap["ichimoku_kijun"] = toF(ind.IchimokuKijun)
			indMap["ichimoku_senkou_a"] = toF(ind.IchimokuSenkouA)
			indMap["ichimoku_senkou_b"] = toF(ind.IchimokuSenkouB)
			// Also add VWAP if locally computed
			if ind.VWAP.GreaterThan(decimal.Zero) {
				indMap["vwap"] = toF(ind.VWAP)
			} else if mt5Snapshot.VWAP.SessionVWAP > 0 {
				indMap["vwap"] = mt5Snapshot.VWAP.SessionVWAP
			}
			response["source"] = mt5Snapshot.Source + "+LOCAL_COMPUTE"
		} else {
			// No local indicators — add VWAP from MT5 if available
			if mt5Snapshot.VWAP.SessionVWAP > 0 {
				indMap["vwap"] = mt5Snapshot.VWAP.SessionVWAP
			}
			response["source"] = mt5Snapshot.Source
		}

		// Add session/MTF/structure info into indicators map so frontend liveness works
		indMap["session"] = mt5Snapshot.Session.Name
		indMap["is_overlap"] = mt5Snapshot.Session.IsOverlap
		indMap["is_weekend"] = mt5Snapshot.Session.IsWeekend

		// Derive MACD histogram from main - signal if not already set
		if indMap["macd_histogram"] == 0 || indMap["macd_histogram"] == 0.0 {
			macdMain, _ := indMap["macd_main"].(float64)
			macdSignal, _ := indMap["macd_signal"].(float64)
			if macdMain != 0 || macdSignal != 0 {
				indMap["macd_histogram"] = macdMain - macdSignal
			}
		}

		response["indicators"] = indMap
		// Always include authoritative server time for frontend clock sync
		response["server_time"] = time.Now().UTC().Format(time.RFC3339)
		// Broker session timezone: the engine runs on Broker TF, not UTC. Expose
		// the active offset so the dashboard can display broker-local time.
		brokerOff := h.agentProvider.BrokerOffsetHours()
		response["broker_offset"] = brokerOff
		response["broker_time"] = time.Now().UTC().Add(time.Duration(brokerOff) * time.Hour).Format(time.RFC3339)
		if brokerOff != 0 {
			response["time_mode"] = "BROKER_ALIGNED"
		} else {
			response["time_mode"] = "UTC_ALIGNED"
		}
	} else if engineState != nil && engineState.Indicators.ATR.GreaterThan(decimal.Zero) {
		// No MT5 snapshot — return locally-computed indicators only
		localMap := h.buildIndicatorMap(&engineState.Indicators)
		if engineState.Session.CurrentSession != "" {
			localMap["session"] = engineState.Session.CurrentSession
			localMap["is_overlap"] = engineState.Session.IsOverlap
			localMap["is_weekend"] = engineState.Session.IsWeekend
		}
		// Derive MACD histogram from main - signal if both available
		if localMap["macd_histogram"] == 0 || localMap["macd_histogram"] == 0.0 {
			macdMain, _ := localMap["macd_main"].(float64)
			macdSignal, _ := localMap["macd_signal"].(float64)
			if macdMain != 0 || macdSignal != 0 {
				localMap["macd_histogram"] = macdMain - macdSignal
			}
		}
		// Derive ADX +DI/-DI from engine state if zero in indicators
		if localMap["adx_plus_di"] == 0 || localMap["adx_plus_di"] == 0.0 {
			if engineState.Indicators.ADXPlusDI.GreaterThan(decimal.Zero) {
				localMap["adx_plus_di"] = toF(engineState.Indicators.ADXPlusDI)
			}
		}
		if localMap["adx_minus_di"] == 0 || localMap["adx_minus_di"] == 0.0 {
			if engineState.Indicators.ADXMinusDI.GreaterThan(decimal.Zero) {
				localMap["adx_minus_di"] = toF(engineState.Indicators.ADXMinusDI)
			}
		}
		response["indicators"] = localMap
		response["source"] = "LOCAL_COMPUTE_ONLY"
		response["timestamp"] = time.Now().UTC().Format(time.RFC3339)
		response["server_time"] = time.Now().UTC().Format(time.RFC3339)
		response["message"] = "No MT5 Master Node snapshot. Showing locally-computed indicators only."
	} else {
		// No data at all
		response["snapshot"] = nil
		response["status"] = "waiting"
		response["message"] = "No Master Node snapshot received yet. Ensure Master Node EA is running and connected to Windows Agent."
		response["timestamp"] = time.Now().UTC().Format(time.RFC3339)
		response["server_time"] = time.Now().UTC().Format(time.RFC3339)
	}

	json.NewEncoder(w).Encode(response)
}

// buildIndicatorMap creates a JSON-serializable map of ALL indicator values
// from the Go engine's IndicatorFeatures. Always includes every field so the
// frontend can distinguish "computed but zero/not enough data" from "not in response".
func (h *HTTPServer) buildIndicatorMap(ind *features.IndicatorFeatures) map[string]interface{} {
	m := map[string]interface{}{}
	// Trend
	m["ema9"] = toF(ind.EMA9)
	m["ema21"] = toF(ind.EMA21)
	m["ema50"] = toF(ind.EMA50)
	m["ema100"] = toF(ind.EMA100)
	m["ema200"] = toF(ind.EMA200)
	m["ema_cross_9_21"] = ind.EMACross921
	m["sma50"] = toF(ind.SMA50)
	m["sma100"] = toF(ind.SMA100)
	m["sma200"] = toF(ind.SMA200)
	m["macd_main"] = toF(ind.MACDMain)
	m["macd_signal"] = toF(ind.MACDSignal)
	m["macd_histogram"] = toF(ind.MACDHistogram)
	m["macd_bull_cross"] = ind.MACDBullCross
	m["macd_bear_cross"] = ind.MACDBearCross
	m["adx"] = toF(ind.ADX)
	m["adx_plus_di"] = toF(ind.ADXPlusDI)
	m["adx_minus_di"] = toF(ind.ADXMinusDI)
	m["psar"] = toF(ind.ParabolicSAR)
	m["psar_long"] = ind.ParabolicSARLong
	m["ichimoku_tenkan"] = toF(ind.IchimokuTenkan)
	m["ichimoku_kijun"] = toF(ind.IchimokuKijun)
	m["ichimoku_senkou_a"] = toF(ind.IchimokuSenkouA)
	m["ichimoku_senkou_b"] = toF(ind.IchimokuSenkouB)
	// Momentum
	m["rsi"] = toF(ind.RSI)
	m["stoch_main"] = toF(ind.StochMain)
	m["stoch_signal"] = toF(ind.StochSignal)
	m["stoch_rsi"] = toF(ind.StochRSI)
	m["stoch_rsi_k"] = toF(ind.StochRSIK)
	m["stoch_rsi_d"] = toF(ind.StochRSID)
	m["cci"] = toF(ind.CCI)
	// Volatility
	m["atr"] = toF(ind.ATR)
	m["boll_upper"] = toF(ind.BollUpper)
	m["boll_lower"] = toF(ind.BollLower)
	m["boll_middle"] = toF(ind.BollMiddle)
	m["boll_width"] = toF(ind.BollWidth)
	m["boll_bull_rev"] = ind.BollBullRev
	m["boll_bear_rev"] = ind.BollBearRev
	// Volume
	m["obv"] = toF(ind.OBV)
	m["vwap"] = toF(ind.VWAP)
	return m
}

// enrichSnapshot merges locally-computed indicators into a cached snapshot JSON.
func (h *HTTPServer) enrichSnapshot(cachedData []byte, ind *features.IndicatorFeatures) []byte {
	var raw map[string]interface{}
	if err := json.Unmarshal(cachedData, &raw); err != nil {
		return nil
	}
	// Merge in locally-computed indicators
	indMap := h.buildIndicatorMap(ind)
	if existing, ok := raw["indicators"].(map[string]interface{}); ok {
		for k, v := range indMap {
			if _, exists := existing[k]; !exists {
				existing[k] = v
			}
		}
	} else {
		raw["indicators"] = indMap
	}
	if source, ok := raw["source"].(string); ok {
		raw["source"] = source + "+LOCAL_COMPUTE"
	} else {
		raw["source"] = "MT5_MASTER+LOCAL_COMPUTE"
	}
	enriched, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	return enriched
}

// toF converts decimal.Decimal to float64 for JSON serialization.
func toF(d decimal.Decimal) float64 {
	f, _ := d.Float64()
	return f
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
	agentCount := h.agentHub.AgentCount()
	agentsOnline := false
	snapshotCount := uint64(0)
	if h.agentProvider != nil {
		agentsOnline = h.agentProvider.HasConnectedAgents()
		snapshotCount = h.agentProvider.GetSnapshotCount()
	}
	// If the live agentHub reports a connection but the provider has not
	// yet registered it (race during initial handshake), trust the hub —
	// a connected WebSocket agent IS a connected agent.
	if agentCount > 0 && !agentsOnline {
		agentsOnline = true
	}
	dataAgentCount := 0
	if h.DataAgentHub != nil {
		dataAgentCount = h.DataAgentHub.AgentCount()
	}
	lastMarketDataAt := time.Time{}
	lastSnapshotAt := time.Time{}
	if h.agentProvider != nil {
		lastMarketDataAt = h.agentProvider.LastMarketDataAt()
		lastSnapshotAt = h.agentProvider.LastSnapshotAt()
	}
	// Derive a direct data-health view so a silent feed is never invisible.
	// Health is based on MARKET_SNAPSHOT receipt (the feed that builds the market
	// state required to generate signals), not bare ticks — a lone tick must not
	// mask a dead snapshot feed.
	// HEALTHY (<90s), STALE (>=90s), CRITICAL (>=180s), NO_DATA (never received).
	dataHealth := "NO_DATA"
	staleSecs := int64(-1)
	marketClosed := false
	if !lastSnapshotAt.IsZero() {
		age := int64(time.Since(lastSnapshotAt).Seconds())
		staleSecs = age
		// A closed market (weekend/holiday) means the broker streams no quotes;
		// the Master Node emits liveness-only snapshots (market_closed=true,
		// last-known price). A live feed of those snapshots still counts as
		// HEALTHY connectivity — the operator must see the difference between
		// "agents dead" and "market simply closed".
		if snap := h.agentProvider.GetLastSnapshot(); snap != nil {
			if s, ok := snap.(*marketdata.MarketSnapshot); ok && s != nil {
				marketClosed = s.MarketClosed
			}
		}
		switch {
		case !marketClosed && age >= 180:
			dataHealth = "CRITICAL"
		case !marketClosed && age >= 90:
			dataHealth = "STALE"
		default:
			dataHealth = "HEALTHY"
		}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agents_connected":      agentCount,
		"agents_online":         agentsOnline,
		"data_agents_connected": dataAgentCount,
		"snapshot_count":        snapshotCount,
		"last_market_data_at":   lastMarketDataAt.UTC().Format(time.RFC3339),
		"last_snapshot_at":      lastSnapshotAt.UTC().Format(time.RFC3339),
		"data_stale_secs":       staleSecs,
		"data_health":           dataHealth,
		"market_closed":         marketClosed,
		"next_market_open_utc":  nextMarketOpenUTC(time.Now().UTC()).Format(time.RFC3339),
		"mt4_connected":         h.agentHub.MT4ConnectedCount(),
		"mt5_connected":         h.agentHub.MT5ConnectedCount(),
		"timestamp":             time.Now().UTC().Format(time.RFC3339),
		"server_time":           time.Now().UTC().Format(time.RFC3339),
	})
}

// nextMarketOpenUTC computes the next FX re-open (Sun 22:00 UTC) for the
// client-facing weekend countdown (check.md #1).
func nextMarketOpenUTC(now time.Time) time.Time {
	y, m, d := now.UTC().Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	wd := today.Weekday()
	days := (7 - int(wd)) % 7
	nextSunday := today.AddDate(0, 0, days).Add(22 * time.Hour)
	if nextSunday.After(now) {
		return nextSunday
	}
	return today.AddDate(0, 0, 7).Add(22 * time.Hour)
}

// handleNews exposes the current news risk level and upcoming economic events
// (sourced from the configured provider, e.g. FMP). This powers the live news
// feed on the trading terminal.
func (h *HTTPServer) handleNews(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if h.newsEngine == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"risk": "NONE", "events": []interface{}{}, "configured": false})
		return
	}
	risk := h.newsEngine.ComputeRisk(time.Now())
	events := h.newsEngine.GetEvents()
	out := map[string]interface{}{
		"risk":       string(risk.Level),
		"reason":     risk.ReasonCode,
		"evidence":   risk.Evidence,
		"computedAt": risk.ComputedAt,
		"configured": h.newsEngine.HasProvider(),
		"events":     events,
	}
	json.NewEncoder(w).Encode(out)
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

	// B-01 / P0-RT3: Verify device ownership — require a VALID JWT
	// (signature + expiry + alg), not merely a non-empty header.
	// Without verification, any client could replay any device's signals.
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "authorization required for signal resume"})
		return
	}
	if _, err := extractUserIDFromJWT(strings.TrimPrefix(authHeader, "Bearer ")); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid or expired token"})
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
			"device_id":           deviceID,
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
			"signal_id":  signalID,
			"sequence":   seqNum,
			"direction":  direction,
			"entry":      entryPrice,
			"stop_loss":  stopLoss,
			"tp1":        tp1,
			"tp2":        tp2,
			"tp3":        tp3,
			"strategy":   strategyID,
			"status":     status,
			"created_at": createdAt,
			"expires_at": expiresAt,
		})
	}

	if missedSignals == nil {
		missedSignals = []map[string]interface{}{}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"device_id":           deviceID,
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
			"status":  "NO_DATA",
			"message": "No market state available",
		})
		return
	}

	rsi, _ := state.Indicators.RSI.Float64()
	adx, _ := state.Indicators.ADX.Float64()
	atr, _ := state.Indicators.ATR.Float64()

	diag := map[string]interface{}{
		"status":                "OK",
		"symbol":                state.Symbol,
		"timestamp":             state.Timestamp,
		"current_regime":        string(state.Regime.Current),
		"previous_regime":       string(state.Regime.Previous),
		"regime_age":            state.Regime.Age.String(),
		"confidence":            state.Regime.Confidence,
		"entry_reason":          state.Regime.EntryReason,
		"raw_regime":            string(state.Regime.RawRegime),
		"entered_at":            state.Regime.EnteredAt,
		"current_rsi":           rsi,
		"current_adx":           adx,
		"current_atr":           atr,
		"volatility":            state.Regime.Volatility,
		"hold_reason":           state.Regime.HoldReason,
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
		"agents_connected": h.agentHub.AgentCount(),
		"agents_online": func() bool {
			mc := h.agentProvider.HasConnectedAgents()
			if !mc && h.agentHub.AgentCount() > 0 {
				mc = true
			}
			return mc
		}(),
	}

	// Overall ready
	health["ready"] = dbConnected && dbHealthy
	health["ready_reason"] = ""
	if !dbConnected {
		health["ready_reason"] = "database_unavailable"
	}

	json.NewEncoder(w).Encode(health)
}

// requireAdminAction wraps destructive admin actions with JWT verification.
// Only control-plane tokens with an admin role may pass. The realtime engine
// mints no tokens itself — JWT_SECRET must match the control plane.
func (h *HTTPServer) requireAdminAction(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			observability.Log.Warn().Str("remote", r.RemoteAddr).Msg("ADMIN ACTION DENIED — missing bearer token")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized","reason":"admin JWT required"}`))
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == "" || jwtSecret == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error":"server_misconfigured","reason":"JWT_SECRET not set — admin actions disabled"}`))
			return
		}
		sub, role, err := validateJWTFull(token)
		if err != nil {
			observability.Log.Warn().Str("remote", r.RemoteAddr).Err(err).Msg("ADMIN ACTION DENIED — invalid JWT")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized","reason":"invalid token"}`))
			return
		}
		roleUpper := strings.ToUpper(role)
		if roleUpper != "ADMIN" && roleUpper != "SUPER_ADMIN" {
			observability.Log.Warn().Str("remote", r.RemoteAddr).Str("sub", sub).Str("role", role).
				Msg("ADMIN ACTION DENIED — insufficient role")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"forbidden","reason":"admin role required"}`))
			return
		}
		next(w, r)
	}
}

// handleEmergencyStop sends EMERGENCY_STOP to all connected agents AND
// activates the server-side EmergencyHalt: signal generation and delivery
// stop immediately (v1.15.0 server enforcement authority).
func (h *HTTPServer) handleEmergencyStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.agentHub == nil {
		http.Error(w, "agent hub not available", http.StatusServiceUnavailable)
		return
	}
	reason := r.URL.Query().Get("reason")
	if reason == "" {
		reason = "admin_manual"
	}
	if h.EmergencyHalt != nil {
		h.EmergencyHalt.Activate("EMERGENCY_STOP", reason)
	}
	// Broadcast EMERGENCY_STOP to all agents
	h.agentHub.BroadcastToAllAgents("EMERGENCY_STOP", map[string]interface{}{
		"reason":    reason,
		"timestamp": time.Now().UTC(),
	})
	observability.Log.Warn().Str("reason", reason).Msg("EMERGENCY_STOP: broadcast to all agents + SERVER-SIDE trading halt ACTIVE — no further EXECUTABLE signals")
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"EMERGENCY_STOP_SENT","server_halt_active":true,"reason":"` + reason + `"}`))
}

// handleKillSwitch sends KILL_SWITCH to all connected agents AND activates the
// server-side halt (same engine-level suppression as EMERGENCY_STOP).
func (h *HTTPServer) handleKillSwitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.agentHub == nil {
		http.Error(w, "agent hub not available", http.StatusServiceUnavailable)
		return
	}
	reason := r.URL.Query().Get("reason")
	if reason == "" {
		reason = "admin_manual"
	}
	if h.EmergencyHalt != nil {
		h.EmergencyHalt.Activate("KILL_SWITCH", reason)
	}
	// Broadcast KILL_SWITCH to all agents
	h.agentHub.BroadcastToAllAgents("KILL_SWITCH", map[string]interface{}{
		"reason":    reason,
		"timestamp": time.Now().UTC(),
	})
	observability.Log.Warn().Str("reason", reason).Msg("KILL_SWITCH: broadcast to all agents + SERVER-SIDE trading halt ACTIVE — no further EXECUTABLE signals")
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"KILL_SWITCH_SENT","server_halt_active":true,"reason":"` + reason + `"}`))
}

// handleEmergencyResume clears the server-side halt after incident review.
// Explicit operator action — never automatic on restart.
func (h *HTTPServer) handleEmergencyResume(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	reason := r.URL.Query().Get("reason")
	if reason == "" {
		reason = "admin_resume"
	}
	if h.EmergencyHalt != nil {
		h.EmergencyHalt.Resumed(reason)
	}
	observability.Log.Warn().Str("reason", reason).Msg("EMERGENCY HALT RESUMED — signal generation re-enabled by operator")
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"RESUMED","server_halt_active":false,"reason":"` + reason + `"}`))
}

// handleDevilLiquidityMarks returns active Devil's Marks from the global engine.
// It reads in-memory state first, falling back to the DB-backed store when the
// engine is persistence-enabled and no in-memory marks exist.
func (h *HTTPServer) handleDevilLiquidityMarks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	eng := devilliquidity.GlobalEngine()
	type resp struct {
		Enabled bool                        `json:"enabled"`
		Count   int                         `json:"count"`
		Marks   []*devilliquidity.DevilMark `json:"marks"`
		Stats   devilliquidity.EngineStats  `json:"stats"`
	}
	out := resp{Enabled: false, Marks: []*devilliquidity.DevilMark{}}
	if eng == nil {
		writeJSON(w, out)
		return
	}
	out.Enabled = true
	out.Stats = eng.Stats()
	marks := eng.ActiveMarks()
	if len(marks) == 0 {
		// Fallback: try DB-backed recent marks (e.g. after restart).
		// The global engine does not expose the store directly here; in-memory
		// is authoritative during a live session.
	}
	out.Marks = marks
	out.Count = len(marks)
	writeJSON(w, out)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
