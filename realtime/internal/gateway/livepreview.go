// Centralized live-dashboard access guard (prompt.md §10/§12): ONE reusable
// mechanism protecting every live.predictatrade.com data path — REST and WS.
//
// Precedence (§11): authenticated entitlement → anonymous trial → deny.
package gateway

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/predictatrade/realtime/internal/livepreview"
)

type ctxKey string

const trialTokenKey ctxKey = "lp_trial_token_hash"

// TrialGuard wraps protected live-data handlers.
type TrialGuard struct {
	svc *livepreview.Service
	db  *sql.DB

	entMu    sync.Mutex
	entCache map[string]entCacheEntry // userID → entitlement, 60s TTL (§48/§83)
}

type entCacheEntry struct {
	entitled bool
	err      error
	expires  time.Time
}

func NewTrialGuard(svc *livepreview.Service, db *sql.DB) *TrialGuard {
	return &TrialGuard{svc: svc, db: db, entCache: map[string]entCacheEntry{}}
}

func (g *TrialGuard) Enabled() bool { return g != nil && g.svc != nil && g.svc.Enabled() }

// Middleware returns the centralized guard for protected live paths.
func (g *TrialGuard) Middleware(next http.Handler) http.Handler {
	if !g.Enabled() {
		return next // feature flag off → production behavior unchanged (§67)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d := g.Resolve(r)
		if d.NewCookie != nil {
			http.SetCookie(w, g.svc.Cookie(*d.NewCookie))
		}
		if d.Allowed {
			if d.Trial != nil {
				r = r.WithContext(context.WithValue(r.Context(), trialTokenKey, d.Trial.TokenHash))
			}
			next.ServeHTTP(w, r)
			return
		}
		writeAccessDenied(w, d)
	})
}

// Resolve applies the precedence chain. Exported for the WS handshake path.
func (g *TrialGuard) Resolve(r *http.Request) livepreview.Decision {
	// 1) Authenticated entitlement (existing JWT infrastructure — §64).
	if userID, ok := userIDFromRequest(r); ok {
		entitled, err := g.entitlement(userID)
		if err != nil {
			// Fail closed on infrastructure errors (§47) but with a distinct
			// safe code; authenticated users retry when DB recovers.
			return livepreview.Decision{Allowed: false, Reason: livepreview.ReasonServiceUnavail}
		}
		if entitled {
			return livepreview.Decision{Allowed: true}
		}
		// Authenticated but no active subscription → fall through to the
		// anonymous trial they may legitimately hold (never downgrade a paid
		// user; a paid user would have been entitled above).
	}

	// 2) Anonymous server-side trial.
	return g.svc.Evaluate(r)
}

// entitlement checks the EXISTING subscription system (§64 — no new engine):
// an ACTIVE subscription row of any plan grants live-terminal access, with a
// 60s per-user cache so polling never hits PostgreSQL per request (§48).
func (g *TrialGuard) entitlement(userID string) (bool, error) {
	g.entMu.Lock()
	if e, ok := g.entCache[userID]; ok && time.Now().Before(e.expires) {
		g.entMu.Unlock()
		return e.entitled, e.err
	}
	g.entMu.Unlock()

	var entitled bool
	err := g.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM billing.subscriptions WHERE user_id = $1 AND status = 'ACTIVE'
		)`, userID).Scan(&entitled)
	if err != nil {
		entitled = false
	}
	g.entMu.Lock()
	g.entCache[userID] = entCacheEntry{entitled: entitled, err: err, expires: time.Now().Add(60 * time.Second)}
	g.entMu.Unlock()
	return entitled, err
}

func userIDFromRequest(r *http.Request) (string, bool) {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		if id, err := extractUserIDFromJWT(strings.TrimPrefix(h, "Bearer ")); err == nil && id != "" {
			return id, true
		}
	}
	if tk := r.URL.Query().Get("token"); tk != "" {
		if id, err := extractUserIDFromJWT(tk); err == nil && id != "" {
			return id, true
		}
	}
	return "", false
}

// writeAccessDenied emits the standardized machine-readable response (§14).
func writeAccessDenied(w http.ResponseWriter, d livepreview.Decision) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	switch d.Reason {
	case livepreview.ReasonServiceUnavail:
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": "LIVE_ACCESS_UNAVAILABLE", "reason": d.Reason,
			"message": "Unable to verify live access. Please refresh or sign in.",
		})
	default:
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": "REGISTRATION_REQUIRED", "reason": d.Reason,
		})
	}
}

// handleLivePreviewStatus is the anonymous countdown source of truth (§9/§85).
func (h *HTTPServer) handleLivePreviewStatus(w http.ResponseWriter, r *http.Request) {
	d := h.trialGuard.Resolve(r)
	if d.NewCookie != nil {
		http.SetCookie(w, h.trialSvc.Cookie(*d.NewCookie))
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if !d.Allowed {
		w.WriteHeader(http.StatusForbidden)
	}
	json.NewEncoder(w).Encode(h.trialSvc.StatusFor(d))
}

// handleLivePreviewEvent records funnel events once per visitor action (§30).
func (h *HTTPServer) handleLivePreviewEvent(w http.ResponseWriter, r *http.Request) {
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
	allowed := map[string]bool{
		livepreview.EventWallShown: true, livepreview.EventSignupStart: true,
	}
	if !allowed[body.Event] {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	h.trialSvc.RecordFunnelEvent(r, body.Event)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// handleLivePreviewAdminStats serves real funnel metrics from the DB view
// (§31/§32) to ADMIN-role JWTs only. No tokens, hashes or IPs are exposed.
func (h *HTTPServer) handleLivePreviewAdminStats(w http.ResponseWriter, r *http.Request) {
	if !adminFromRequest(r) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "admin role required"})
		return
	}
	rows, err := h.trialGuard.db.Query(`
		SELECT unique_preview_visitors, active_previews, preview_starts,
		       reached_1_minute, reached_3_minutes, reached_5_minutes,
		       registration_wall_reached, signup_started, signups_completed,
		       repeat_attempt_blocks, expired_naturally
		FROM live_preview.funnel_stats`)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "stats unavailable"})
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
		if err := rows.Scan(dest...); err == nil {
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func adminFromRequest(r *http.Request) bool {
	tk := ""
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		tk = strings.TrimPrefix(h, "Bearer ")
	} else {
		tk = r.URL.Query().Get("token")
	}
	if tk == "" {
		return false
	}
	_, role, err := validateJWTFull(tk)
	return err == nil && strings.EqualFold(role, "ADMIN")
}
