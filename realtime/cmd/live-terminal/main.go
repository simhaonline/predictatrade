// live-terminal — public edge service for live.predictatrade.com.
//
// Purpose: isolate ALL anonymous visitor traffic (REST polling, WebSockets,
// preview trials) from the trading-critical realtime engine. This service
// serves the static terminal, enforces the server-side 5-minute preview,
// and proxies allowed data paths to the engine over the internal network.
//
// Access precedence: authenticated entitlement (control-plane JWT cookie /
// bearer) → anonymous ACTIVE trial → 403 REGISTRATION_REQUIRED.
// Fail-closed on infrastructure errors.
package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/gorilla/websocket"

	"github.com/predictatrade/realtime/internal/livepreview"
)

type config struct {
	port        string
	upstream    string
	staticDir   string
	jwtSecret   string
	dbURL       string
	lp          livepreview.Config
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func main() {
	cfg := config{
		port:      envOr("LIVE_TERMINAL_PORT", "13090"),
		upstream:  envOr("UPSTREAM_REALTIME_URL", "http://realtime:13081"),
		staticDir: envOr("LIVE_TERMINAL_STATIC_DIR", "/app/public"),
		jwtSecret: os.Getenv("JWT_SECRET"),
		dbURL:     os.Getenv("DATABASE_URL"),
		lp: livepreview.Config{
			Enabled:        os.Getenv("LIVE_PREVIEW_ENABLED") == "true",
			Duration:       300 * time.Second,
			CookieName:     envOr("LIVE_PREVIEW_COOKIE_NAME", "pat_live_trial"),
			HMACSecret:     os.Getenv("LIVE_PREVIEW_HMAC_SECRET"),
			AbuseDetection: os.Getenv("LIVE_PREVIEW_ABUSE_DETECTION_ENABLED") != "false",
		},
	}
	if v := os.Getenv("LIVE_PREVIEW_ABUSE_THRESHOLD"); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			cfg.lp.AbuseThreshold = n
		}
	}
	if d := os.Getenv("LIVE_PREVIEW_DURATION_SECONDS"); d != "" {
		var n int
		if _, err := fmt.Sscanf(d, "%d", &n); err == nil && n > 0 && n <= 3600 {
			cfg.lp.Duration = time.Duration(n) * time.Second
		}
	}

	var db *sql.DB
	if cfg.dbURL != "" {
		if d, err := sql.Open("pgx", cfg.dbURL); err == nil {
			db = d
		}
	}

	svc := livepreview.New(cfg.lp, &livepreview.DBStore{DB: db})
	up, err := url.Parse(cfg.upstream)
	if err != nil {
		panic(err)
	}

	lt := &liveTerminal{
		cfg: cfg, svc: svc, db: db, upstream: up,
		proxy: httputil.NewSingleHostReverseProxy(up),
		ent:   map[string]entEntry{},
	}

	mux := http.NewServeMux()
	// Trial endpoints (local — the ONLY place trials are minted)
	mux.HandleFunc("/api/v1/live-preview/status", lt.handleStatus)
	mux.HandleFunc("/api/v1/live-preview/event", lt.handleEvent)
	mux.HandleFunc("/api/v1/admin/live-preview/stats", lt.handleAdminStats)
	// Health (local)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "service": "live-terminal", "time": time.Now().UTC()})
	})
	// Protected live data → proxy with guard
	mux.Handle("/api/v1/", lt.guard(lt.proxy))
	// Browser WebSocket → guarded proxy with mid-connection sweep
	mux.HandleFunc("/ws", lt.handleWS)
	mux.HandleFunc("/ws/v1", lt.handleWS)
	// Static terminal
	mux.Handle("/", lt.staticHandler())

	// Mid-connection entitlement sweep (§13): authorization at second zero
	// is not enough for long-lived connections.
	if svc.Enabled() {
		go func() {
			for range time.Tick(15 * time.Second) {
				lt.sweep()
			}
		}()
	}

	addr := ":" + cfg.port
	fmt.Printf("live-terminal listening on %s → upstream %s (preview=%v)\n", addr, cfg.upstream, svc.Enabled())
	if err := http.ListenAndServe(addr, mux); err != nil {
		panic(err)
	}
}

type entEntry struct {
	entitled bool
	err      error
	expires  time.Time
}

type liveTerminal struct {
	cfg      config
	svc      *livepreview.Service
	db       *sql.DB
	upstream *url.URL
	proxy    *httputil.ReverseProxy

	mu      sync.Mutex
	ent     map[string]entEntry
	wsConns map[string]map[*websocket.Conn]bool // trialTokenHash → conns
}

// ── authentication (existing control-plane JWT — no new auth stack) ──

type jwtClaims struct {
	Sub  string `json:"sub"`
	Role string `json:"role"`
	Exp  int64  `json:"exp"`
}

// validateJWT verifies the control-plane HS256 token (shared JWT_SECRET).
func validateJWT(tokenString, secret string) (*jwtClaims, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 || secret == "" {
		return nil, fmt.Errorf("bad token")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(parts[0] + "." + parts[1]))
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(mac.Sum(nil), sig) {
		return nil, fmt.Errorf("bad signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var c jwtClaims
	if err := json.Unmarshal(payload, &c); err != nil {
		return nil, err
	}
	if c.Sub == "" || (c.Exp > 0 && time.Now().Unix() > c.Exp) {
		return nil, fmt.Errorf("expired")
	}
	return &c, nil
}

// tokenFromRequest: Authorization header, platform cookie (shared
// .predictatrade.com domain after login), or WS query token.
func (lt *liveTerminal) tokenFromRequest(r *http.Request) string {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	if ck, err := r.Cookie("pat_access_token"); err == nil && ck.Value != "" {
		return ck.Value
	}
	return r.URL.Query().Get("token")
}

func (lt *liveTerminal) entitlement(userID string) (bool, error) {
	lt.mu.Lock()
	if e, ok := lt.ent[userID]; ok && time.Now().Before(e.expires) {
		lt.mu.Unlock()
		return e.entitled, e.err
	}
	lt.mu.Unlock()
	var entitled bool
	err := fmt.Errorf("no database")
	if lt.db != nil {
		err = lt.db.QueryRow(`SELECT EXISTS (SELECT 1 FROM billing.subscriptions WHERE user_id = $1 AND status = 'ACTIVE')`, userID).Scan(&entitled)
	}
	if err != nil {
		entitled = false
	}
	lt.mu.Lock()
	lt.ent[userID] = entEntry{entitled: entitled, err: err, expires: time.Now().Add(60 * time.Second)}
	lt.mu.Unlock()
	return entitled, err
}

func (lt *liveTerminal) resolve(r *http.Request) livepreview.Decision {
	if tk := lt.tokenFromRequest(r); tk != "" {
		if claims, err := validateJWT(tk, lt.cfg.jwtSecret); err == nil && claims.Sub != "" {
			entitled, err := lt.entitlement(claims.Sub)
			if err != nil {
				return livepreview.Decision{Allowed: false, Reason: livepreview.ReasonServiceUnavail}
			}
			if entitled {
				return livepreview.Decision{Allowed: true}
			}
		}
	}
	return lt.svc.Evaluate(r, false)
}

func denyJSON(w http.ResponseWriter, d livepreview.Decision) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if d.Reason == livepreview.ReasonServiceUnavail {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"code": "LIVE_ACCESS_UNAVAILABLE", "reason": d.Reason,
			"message": "Unable to verify live access. Please refresh or sign in."})
		return
	}
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(map[string]string{"code": "REGISTRATION_REQUIRED", "reason": d.Reason})
}

// ── guard middleware (REST) ──

func (lt *liveTerminal) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !lt.svc.Enabled() {
			next.ServeHTTP(w, r)
			return
		}
		d := lt.resolve(r)
		if d.NewCookie != nil {
			http.SetCookie(w, lt.svc.Cookie(*d.NewCookie))
		}
		if !d.Allowed {
			denyJSON(w, d)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ── trial endpoints ──

func (lt *liveTerminal) handleStatus(w http.ResponseWriter, r *http.Request) {
	d := lt.svc.Evaluate(r, true) // status is the ONLY trial creator
	if d.NewCookie != nil {
		http.SetCookie(w, lt.svc.Cookie(*d.NewCookie))
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if !d.Allowed {
		w.WriteHeader(http.StatusForbidden)
	}
	json.NewEncoder(w).Encode(lt.svc.StatusFor(d))
}

func (lt *liveTerminal) handleEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Event string `json:"event"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if body.Event != livepreview.EventWallShown && body.Event != livepreview.EventSignupStart {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	lt.svc.RecordFunnelEvent(r, body.Event)
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (lt *liveTerminal) handleAdminStats(w http.ResponseWriter, r *http.Request) {
	tk := lt.tokenFromRequest(r)
	claims, err := validateJWT(tk, lt.cfg.jwtSecret)
	if err != nil || !strings.EqualFold(claims.Role, "ADMIN") {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "admin role required"})
		return
	}
	rows, err := lt.db.Query(`SELECT unique_preview_visitors, active_previews, preview_starts,
		reached_1_minute, reached_3_minutes, reached_5_minutes, registration_wall_reached,
		signup_started, signups_completed, repeat_attempt_blocks, expired_naturally
		FROM live_preview.funnel_stats`)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	defer rows.Close()
	out := map[string]int64{}
	if rows.Next() {
		var v [11]int64
		dest := make([]interface{}, 11)
		for i := range v {
			dest[i] = &v[i]
		}
		if rows.Scan(dest...) == nil {
			names := []string{"unique_preview_visitors", "active_previews", "preview_starts",
				"reached_1_minute", "reached_3_minutes", "reached_5_minutes",
				"registration_wall_reached", "signup_started", "signups_completed",
				"repeat_attempt_blocks", "expired_naturally"}
			for i, n := range names {
				out[n] = v[i]
			}
		}
	}
	out["server_time"] = time.Now().UTC().Unix()
	json.NewEncoder(w).Encode(out)
}

// ── WebSocket guarded proxy ──

var upgrader = websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024, CheckOrigin: func(*http.Request) bool { return true }}

func (lt *liveTerminal) handleWS(w http.ResponseWriter, r *http.Request) {
	if !lt.svc.Enabled() {
		lt.proxyWS(w, r, "")
		return
	}
	d := lt.resolve(r)
	if d.NewCookie != nil {
		http.SetCookie(w, lt.svc.Cookie(*d.NewCookie))
	}
	if !d.Allowed {
		denyJSON(w, d) // handshake rejected — no upgrade
		return
	}
	tokenHash := ""
	if d.Trial != nil {
		tokenHash = d.Trial.TokenHash
	}
	lt.proxyWS(w, r, tokenHash)
}

// proxyWS dials the upstream engine and pipes frames both ways. The conn is
// registered under the trial token hash so the sweep can terminate expired
// anonymous connections mid-stream (§13).
func (lt *liveTerminal) proxyWS(w http.ResponseWriter, r *http.Request, tokenHash string) {
	down, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	// Register for sweep
	if tokenHash != "" {
		lt.mu.Lock()
		if lt.wsConns == nil {
			lt.wsConns = map[string]map[*websocket.Conn]bool{}
		}
		if lt.wsConns[tokenHash] == nil {
			lt.wsConns[tokenHash] = map[*websocket.Conn]bool{}
		}
		lt.wsConns[tokenHash][down] = true
		lt.mu.Unlock()
		defer func() {
			lt.mu.Lock()
			delete(lt.wsConns[tokenHash], down)
			lt.mu.Unlock()
			down.Close()
		}()
	} else {
		defer down.Close()
	}

	upURL := "ws://" + lt.upstream.Host + r.RequestURI
	upHdr := http.Header{}
	if h := r.Header.Get("Authorization"); h != "" {
		upHdr.Set("Authorization", h)
	}
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	up, _, err := dialer.DialContext(r.Context(), upURL, upHdr)
	if err != nil {
		down.Close()
		return
	}
	defer up.Close()

	errUp := make(chan struct{})
	errDown := make(chan struct{})
	go func() {
		for {
			mt, msg, err := up.ReadMessage()
			if err != nil {
				close(errUp)
				return
			}
			if writeErr := down.WriteMessage(mt, msg); writeErr != nil {
				close(errUp)
				return
			}
		}
	}()
	go func() {
		for {
			mt, msg, err := down.ReadMessage()
			if err != nil {
				close(errDown)
				return
			}
			if writeErr := up.WriteMessage(mt, msg); writeErr != nil {
				close(errDown)
				return
			}
		}
	}()
	select {
	case <-errUp:
	case <-errDown:
	case <-r.Context().Done():
	}
}

// sweep terminates anonymous WS connections whose trial has expired.
func (lt *liveTerminal) sweep() {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	for hash, conns := range lt.wsConns {
		if lt.svc.IsTokenActive(hash) {
			continue
		}
		for c := range conns {
			ev, _ := json.Marshal(map[string]string{"code": "REGISTRATION_REQUIRED"})
			_ = c.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(4003, "REGISTRATION_REQUIRED"),
				time.Now().Add(2*time.Second))
			_ = c.WriteMessage(websocket.TextMessage, append([]byte(`{"type":"TRIAL_EXPIRED","payload":`), ev...))
			c.Close()
			delete(conns, c)
		}
	}
}

// ── static terminal ──

func (lt *liveTerminal) staticHandler() http.Handler {
	fs := http.FileServer(http.Dir(lt.cfg.staticDir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		path := lt.cfg.staticDir + r.URL.Path
		if r.URL.Path == "/" || r.URL.Path == "" {
			http.ServeFile(w, r, lt.cfg.staticDir+"/index.html")
			return
		}
		if _, err := os.Stat(path); err != nil {
			// SPA fallback
			http.ServeFile(w, r, lt.cfg.staticDir+"/index.html")
			return
		}
		fs.ServeHTTP(w, r)
	})
}

var _ = context.Background
