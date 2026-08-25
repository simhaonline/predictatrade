// Package livepreview implements the server-enforced anonymous 5-minute
// live dashboard preview for live.predictatrade.com (prompt.md funnel).
//
// Security model:
//   - The backend timer is the ONLY authority. The browser may render a
//     countdown, but every access decision is made from
//     trial_expires_at stored server-side.
//   - The cookie holds a random token; the database stores only
//     HMAC-SHA256(secret, token) so a database leak cannot resurrect trials.
//   - IP/User-Agent are stored as HMAC digests (never raw) because raw IP
//     space is enumerable and unsalted hashes are reversible by lookup.
//   - IP alone can never identify a visitor (carrier NAT, offices, hotels);
//     abuse detection is a conservative scored signal, never a sole blocker.
package livepreview

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	StatusActive   = "ACTIVE"
	StatusExpired  = "EXPIRED"
	StatusConverted = "CONVERTED"
	StatusBlocked  = "BLOCKED"
	StatusRevoked  = "REVOKED"
)

// Reason codes for access decisions (prompt.md §88).
const (
	ReasonTrialActive    = "ANONYMOUS_PREVIEW_ACTIVE"
	ReasonExpired        = "LIVE_PREVIEW_EXPIRED"
	ReasonInvalidToken   = "INVALID_TRIAL_TOKEN"
	ReasonRepeatBlocked  = "REPEAT_TRIAL_BLOCKED"
	ReasonServiceUnavail = "LIVE_ACCESS_UNAVAILABLE"
)

// Funnel event names (prompt.md §30).
const (
	EventStarted     = "LIVE_PREVIEW_STARTED"
	EventResumed     = "LIVE_PREVIEW_RESUMED"
	EventExpired     = "LIVE_PREVIEW_EXPIRED"
	EventWallShown   = "REGISTRATION_WALL_SHOWN"
	EventSignupStart = "SIGNUP_STARTED"
	EventBlocked     = "REPEAT_TRIAL_BLOCKED"
)

// Trial is the durable anonymous-preview record.
type Trial struct {
	TokenHash       string     `json:"-"`
	StartedAt       time.Time  `json:"-"`
	ExpiresAt       time.Time  `json:"-"`
	Status          string     `json:"-"`
	IPHash          string     `json:"-"`
	UAHash          string     `json:"-"`
	BrowserFamily   string     `json:"-"`
	DeviceClass     string     `json:"-"`
	WallSeenAt      *time.Time `json:"-"`
	SignupStartedAt *time.Time `json:"-"`
	AbuseScore      int        `json:"-"`
	ExpirationReasonDB string `json:"-"`

	lastSeenMu      sync.Mutex // throttles DB last_seen writes (§48)
	lastSeenSynced  time.Time
}

// Decision is the outcome of an anonymous access evaluation.
type Decision struct {
	Allowed bool
	Reason  string // one of the Reason* constants ("" when allowed via auth)
	Trial   *Trial // non-nil when a trial applies
	NewCookie *string // set when a fresh trial token must be planted
}

// Store persists trials. DB is the source of record; the service keeps an
// in-process read cache because the dashboard polls every ~2s and the engine
// is a single instance — PostgreSQL is never hit per request for hot checks.
type Store interface {
	GetByTokenHash(hash string) (*Trial, error)
	Insert(t *Trial) error
	Save(t *Trial) error
	CountRecent(ipHash, uaHash string, since time.Time) (int, error)
	RecordEvent(tokenHash, event string) error
}

// Config comes from environment (prompt.md §27) — no magic constants.
type Config struct {
	Enabled          bool
	Duration         time.Duration
	CookieName       string
	HMACSecret       string
	AbuseDetection   bool
	RegistrationHost string // e.g. platform.predictatrade.com (for wall links)
}

// Service is the anonymous trial registry.
type Service struct {
	cfg   Config
	store Store
	hmacKey []byte

	mu    sync.RWMutex
	cache map[string]*Trial // tokenHash → trial (hot path)
}

func New(cfg Config, store Store) *Service {
	if cfg.Duration <= 0 {
		cfg.Duration = 300 * time.Second
	}
	if cfg.CookieName == "" {
		cfg.CookieName = "pat_live_trial"
	}
	return &Service{cfg: cfg, store: store, hmacKey: []byte(cfg.HMACSecret), cache: map[string]*Trial{}}
}

func (s *Service) Enabled() bool { return s.cfg.Enabled }

// ── hashing helpers ──

func (s *Service) hash(v string) string {
	mac := hmac.New(sha256.New, s.hmacKey)
	mac.Write([]byte(v))
	return hex.EncodeToString(mac.Sum(nil))
}

func newToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// NormalizeIP extracts the client IP from proxy headers set by nginx.
func NormalizeIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xr := r.Header.Get("X-Real-IP"); xr != "" {
		return strings.TrimSpace(xr)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func classifyUA(ua string) (browser, device string) {
	u := strings.ToLower(ua)
	switch {
	case strings.Contains(u, "edg/"):
		browser = "edge"
	case strings.Contains(u, "chrome"):
		browser = "chrome"
	case strings.Contains(u, "firefox"):
		browser = "firefox"
	case strings.Contains(u, "safari"):
		browser = "safari"
	default:
		browser = "other"
	}
	switch {
	case strings.Contains(u, "ipad"), strings.Contains(u, "tablet"):
		device = "tablet"
	case strings.Contains(u, "mobile"), strings.Contains(u, "android"), strings.Contains(u, "iphone"):
		device = "mobile"
	default:
		device = "desktop"
	}
	return
}

// ── trial lifecycle ──

// Evaluate resolves anonymous access for the request. It never panics and
// fails CLOSED on storage errors (prompt.md §47) — an unavailable trial
// store must not expose unlimited protected live data.
func (s *Service) Evaluate(r *http.Request) Decision {
	if !s.cfg.Enabled {
		return Decision{Allowed: true}
	}

	cookie, err := r.Cookie(s.cfg.CookieName)
	if err == nil && cookie.Value != "" {
		hash := s.hash(cookie.Value)
		if t, ok := s.fromCacheOrStore(hash); ok {
			return s.evaluateTrial(t, hash)
		}
		// Cookie present but unknown/tampered → reject; do NOT silently mint
		// a fresh trial (prevents cookie-churn reset attacks).
		return Decision{Allowed: false, Reason: ReasonInvalidToken}
	}

	// No cookie: privacy-conscious repeat-visitor check (§24–26). Coarse
	// HMAC signals only; IP alone is never sufficient to block.
	if s.cfg.AbuseDetection {
		ipHash := s.hash("ip:" + NormalizeIP(r))
		uaHash := s.hash("ua:" + r.UserAgent())
		if n, err := s.store.CountRecent(ipHash, uaHash, time.Now().Add(-24*time.Hour)); err == nil && n >= 2 {
			return Decision{Allowed: false, Reason: ReasonRepeatBlocked}
		}
	}

	t, token, err := s.createTrial(r)
	if err != nil {
		return Decision{Allowed: false, Reason: ReasonServiceUnavail}
	}
	_ = s.store.RecordEvent(t.TokenHash, EventStarted)
	return Decision{Allowed: true, Reason: ReasonTrialActive, Trial: t, NewCookie: &token}
}

func (s *Service) evaluateTrial(t *Trial, hash string) Decision {
	now := time.Now().UTC()
	switch {
	case t.Status == StatusBlocked || t.Status == StatusRevoked:
		return Decision{Allowed: false, Reason: ReasonRepeatBlocked}
	case t.Status == StatusConverted:
		// Converted trials are managed by the authenticated entitlement path;
		// if we reach here the user is not authenticated → wall.
		return Decision{Allowed: false, Reason: ReasonExpired}
	case now.After(t.ExpiresAt) || t.Status == StatusExpired:
		if t.Status == StatusActive {
			s.expire(t, "NATURAL")
		}
		return Decision{Allowed: false, Reason: ReasonExpired}
	default:
		// Refresh must NOT reset the timer: started_at is immutable (§8).
		s.touch(t, hash, now)
		return Decision{Allowed: true, Reason: ReasonTrialActive, Trial: t}
	}
}

func (s *Service) createTrial(r *http.Request) (*Trial, string, error) {
	token := newToken()
	now := time.Now().UTC()
	browser, device := classifyUA(r.UserAgent())
	abuse := 0
	if s.cfg.AbuseDetection {
		if n, err := s.store.CountRecent(s.hash("ip:"+NormalizeIP(r)), s.hash("ua:"+r.UserAgent()), now.Add(-24*time.Hour)); err == nil && n > 0 {
			abuse = n // informational score, not a blocker by itself
		}
	}
	t := &Trial{
		TokenHash:     s.hash(token),
		StartedAt:     now,
		ExpiresAt:     now.Add(s.cfg.Duration),
		Status:        StatusActive,
		IPHash:        s.hash("ip:" + NormalizeIP(r)),
		UAHash:        s.hash("ua:" + r.UserAgent()),
		BrowserFamily: browser,
		DeviceClass:   device,
		AbuseScore:    abuse,
	}
	if err := s.store.Insert(t); err != nil {
		return nil, "", err
	}
	t.lastSeenSynced = now
	s.mu.Lock()
	s.cache[t.TokenHash] = t
	s.mu.Unlock()
	return t, token, nil
}

func (s *Service) expire(t *Trial, reason string) {
	t.Status = StatusExpired
	t.ExpirationReasonDB = reason
	s.mu.Lock()
	s.cache[t.TokenHash] = t
	s.mu.Unlock()
	_ = s.store.Save(t)
	_ = s.store.RecordEvent(t.TokenHash, EventExpired)
}

// touch refreshes last_seen at most once per minute per trial (§48: avoid
// hitting PostgreSQL per request).
func (s *Service) touch(t *Trial, hash string, now time.Time) {
	t.lastSeenMu.Lock()
	due := now.Sub(t.lastSeenSynced) >= time.Minute
	t.lastSeenMu.Unlock()
	if due {
		t.lastSeenSynced = now
		_ = s.store.Save(t)
	}
}

func (s *Service) fromCacheOrStore(hash string) (*Trial, bool) {
	s.mu.RLock()
	t, ok := s.cache[hash]
	s.mu.RUnlock()
	if ok {
		return t, true
	}
	t, err := s.store.GetByTokenHash(hash)
	if err != nil || t == nil {
		return nil, false
	}
	s.mu.Lock()
	s.cache[hash] = t
	s.mu.Unlock()
	return t, true
}

// RecordFunnelEvent stores a wall/signup event (idempotent per event type is
// enforced by the frontend sending it once; DB keeps the raw history).
func (s *Service) RecordFunnelEvent(r *http.Request, event string) {
	if !s.cfg.Enabled {
		return
	}
	if cookie, err := r.Cookie(s.cfg.CookieName); err == nil && cookie.Value != "" {
		_ = s.store.RecordEvent(s.hash(cookie.Value), event)
		if event == EventWallShown {
			if t, ok := s.fromCacheOrStore(s.hash(cookie.Value)); ok && t.WallSeenAt == nil {
				n := time.Now().UTC()
				t.WallSeenAt = &n
				_ = s.store.Save(t)
			}
		}
	}
}

// IsTokenActive is the WebSocket mid-connection revalidation hook (§13): a
// long-lived connection must not outlive its entitlement.
func (s *Service) IsTokenActive(tokenHash string) bool {
	if !s.cfg.Enabled {
		return true
	}
	t, ok := s.fromCacheOrStore(tokenHash)
	if !ok {
		return false
	}
	if t.Status != StatusActive {
		return false
	}
	if time.Now().UTC().After(t.ExpiresAt) {
		s.expire(t, "NATURAL")
		return false
	}
	return true
}

// HashToken exposes token hashing for the WS guard (it sees the raw cookie).
func (s *Service) HashToken(raw string) string { return s.hash(raw) }

// Cookie builds the trial cookie (§7: HttpOnly, Secure, SameSite=Lax).
func (s *Service) Cookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:     s.cfg.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((s.cfg.Duration + 24*time.Hour).Seconds()),
	}
}

// StatusPayload is the anonymous-facing API contract (§85) — no internals.
type StatusPayload struct {
	Status              string `json:"status"`
	ServerTime          string `json:"server_time"`
	TrialStartedAt      string `json:"trial_started_at,omitempty"`
	TrialExpiresAt      string `json:"trial_expires_at,omitempty"`
	RemainingSeconds    int    `json:"remaining_seconds"`
	RegistrationRequired bool  `json:"registration_required"`
	Code                string `json:"code,omitempty"`
}

// StatusFor builds the countdown payload from server-authoritative time (§9,
// §56: the frontend derives its display from these fields, never its clock).
func (s *Service) StatusFor(d Decision) StatusPayload {
	now := time.Now().UTC()
	if !s.cfg.Enabled {
		return StatusPayload{Status: "DISABLED", ServerTime: now.Format(time.RFC3339), RemainingSeconds: -1}
	}
	switch {
	case d.Allowed && d.Trial != nil:
		rem := int(d.Trial.ExpiresAt.Sub(now).Seconds())
		if rem < 0 {
			rem = 0
		}
		return StatusPayload{
			Status: "ACTIVE", ServerTime: now.Format(time.RFC3339),
			TrialStartedAt: d.Trial.StartedAt.Format(time.RFC3339),
			TrialExpiresAt: d.Trial.ExpiresAt.Format(time.RFC3339),
			RemainingSeconds: rem,
		}
	case d.Reason == ReasonRepeatBlocked:
		return StatusPayload{Status: "BLOCKED", ServerTime: now.Format(time.RFC3339), RemainingSeconds: 0, RegistrationRequired: true, Code: ReasonRepeatBlocked}
	case d.Reason == ReasonInvalidToken:
		return StatusPayload{Status: "EXPIRED", ServerTime: now.Format(time.RFC3339), RemainingSeconds: 0, RegistrationRequired: true, Code: ReasonInvalidToken}
	case d.Reason == ReasonServiceUnavail:
		return StatusPayload{Status: "ERROR", ServerTime: now.Format(time.RFC3339), RemainingSeconds: 0, RegistrationRequired: true, Code: ReasonServiceUnavail}
	default:
		return StatusPayload{Status: "EXPIRED", ServerTime: now.Format(time.RFC3339), RemainingSeconds: 0, RegistrationRequired: true, Code: ReasonExpired}
	}
}

