package livepreview

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func cfg() Config {
	return Config{
		Enabled: true, Duration: 300 * time.Second,
		CookieName: "pat_live_trial", HMACSecret: "test-secret",
		AbuseDetection: true,
	}
}

func reqWithCookie(r *http.Request, name, val string) *http.Request {
	if val != "" {
		r.AddCookie(&http.Cookie{Name: name, Value: val})
	}
	return r
}

// §79 fresh visitor → 5 minutes, cookie planted, trial recorded.
func TestFreshVisitorGetsTrial(t *testing.T) {
	s := New(cfg(), NewMemStore())
	r := httptest.NewRequest("GET", "/api/v1/market/snapshot", nil)

	d := s.Evaluate(r, true)
	if !d.Allowed || d.NewCookie == nil || d.Trial == nil {
		t.Fatalf("expected allowed trial with cookie, got %+v", d)
	}
	if rem := d.Trial.ExpiresAt.Sub(d.Trial.StartedAt); rem != 300*time.Second {
		t.Fatalf("expected 300s duration, got %v", rem)
	}
	if d.Trial.TokenHash == "" {
		t.Fatal("token hash must be stored")
	}
	// Cookie must carry the RAW token (client needs it), never the hash.
	if d.NewCookie == nil || *d.NewCookie == d.Trial.TokenHash {
		t.Fatal("cookie must contain raw token, not hash")
	}
}

// §8/§39 refresh must NOT reset the timer.
func TestRefreshDoesNotResetTimer(t *testing.T) {
	s := New(cfg(), NewMemStore())
	r := httptest.NewRequest("GET", "/", nil)
	d := s.Evaluate(r, true)
	token := *d.NewCookie
	started := d.Trial.StartedAt

	// simulate 4 minutes passing
	d.Trial.StartedAt = started.Add(-4 * time.Minute)
	d.Trial.ExpiresAt = started.Add(-4 * time.Minute).Add(300 * time.Second)
	s.mu.Lock()
	s.cache[d.Trial.TokenHash] = d.Trial
	s.mu.Unlock()

	r2 := reqWithCookie(httptest.NewRequest("GET", "/", nil), "pat_live_trial", token)
	d2 := s.Evaluate(r2, true)
	if !d2.Allowed {
		t.Fatalf("expected still active, got %+v", d2)
	}
	rem := time.Until(d2.Trial.ExpiresAt)
	if rem > time.Minute+5*time.Second {
		t.Fatalf("refresh reset the timer: %v remaining (want ~1m)", rem)
	}
	if !d2.Trial.StartedAt.Equal(d.Trial.StartedAt) {
		t.Fatal("started_at changed on resume — timer reset bug")
	}
}

// Tampered cookie → rejected, no new trial minted (§79).
func TestTamperedCookieRejected(t *testing.T) {
	s := New(cfg(), NewMemStore())
	r := reqWithCookie(httptest.NewRequest("GET", "/", nil), "pat_live_trial", "forged-token-value")
	d := s.Evaluate(r, true)
	if d.Allowed {
		t.Fatal("tampered cookie must be rejected")
	}
	if d.Reason != ReasonInvalidToken {
		t.Fatalf("want INVALID_TRIAL_TOKEN, got %s", d.Reason)
	}
	if d.NewCookie != nil {
		t.Fatal("must not mint a new trial for a tampered cookie")
	}
	if s.store.(*MemStore).Count() != 0 {
		t.Fatal("no trial row should exist for tampered cookie")
	}
}

// Expiry: after expiration access is denied with LIVE_PREVIEW_EXPIRED.
func TestExpiredTrialDenied(t *testing.T) {
	s := New(cfg(), NewMemStore())
	d := s.Evaluate(httptest.NewRequest("GET", "/", nil))
	token := *d.NewCookie
	tk := d.Trial
	tk.ExpiresAt = time.Now().UTC().Add(-time.Second)
	s.mu.Lock()
	s.cache[tk.TokenHash] = tk
	s.mu.Unlock()

	d2 := s.Evaluate(reqWithCookie(httptest.NewRequest("GET", "/", nil), "pat_live_trial", token), true)
	if d2.Allowed || d2.Reason != ReasonExpired {
		t.Fatalf("expected expired denial, got %+v", d2)
	}
	if tk.Status != StatusExpired {
		t.Fatalf("trial status should be EXPIRED, got %s", tk.Status)
	}
}

// §26/§40: cookie deleted after consuming trial → repeat visitor blocked
// only after 2+ prior coarse-signal matches (conservative scoring).
func TestRepeatVisitorAbuseFlow(t *testing.T) {
	s := New(cfg(), NewMemStore())
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.7")
	r.Header.Set("User-Agent", "Mozilla/5.0 Chrome Test")

	d := s.Evaluate(r, true)
	_ = *d.NewCookie
	// expire first trial
	d.Trial.ExpiresAt = time.Now().UTC().Add(-time.Second)
	s.expire(d.Trial, "NATURAL")

	// second visit, no cookie (deleted): 1 prior match → LOW confidence → allowed
	d2 := s.Evaluate(r, true)
	if !d2.Allowed {
		t.Fatalf("first repeat should be allowed (low confidence), got %+v", d2)
	}
	// expire second trial
	d2.Trial.ExpiresAt = time.Now().UTC().Add(-time.Second)
	s.expire(d2.Trial, "NATURAL")

	// third visit: 2 prior matches → high confidence repeat → blocked
	d3 := s.Evaluate(r, true)
	if d3.Allowed || d3.Reason != ReasonRepeatBlocked {
		t.Fatalf("expected REPEAT_TRIAL_BLOCKED on third visit, got %+v", d3)
	}
}

// Shared-IP households must NOT be blocked by IP alone (§26).
func TestSharedIPDifferentUAAllowed(t *testing.T) {
	s := New(cfg(), NewMemStore())
	for i := 0; i < 3; i++ {
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("X-Forwarded-For", "198.51.100.9") // same office NAT
		r.Header.Set("User-Agent", "DifferentBrowser/"+string(rune('a'+i)))
		d := s.Evaluate(r, true)
		if !d.Allowed {
			t.Fatalf("distinct visitor on shared IP must get a trial, got %+v", d)
		}
	}
}

// §47: storage failure fails CLOSED.
func TestStoreFailureFailsClosed(t *testing.T) {
	s := New(cfg(), failingStore{})
	d := s.Evaluate(httptest.NewRequest("GET", "/", nil))
	if d.Allowed {
		t.Fatal("must fail closed when the trial store is unavailable")
	}
	if d.Reason != ReasonServiceUnavail {
		t.Fatalf("want LIVE_ACCESS_UNAVAILABLE, got %s", d.Reason)
	}
}

type failingStore struct{}

func (failingStore) GetByTokenHash(string) (*Trial, error) { return nil, errDB }
func (failingStore) Insert(*Trial) error                   { return errDB }
func (failingStore) Save(*Trial) error                     { return errDB }
func (failingStore) CountRecent(string, string, time.Time) (int, error) {
	return 0, errDB
}
func (failingStore) RecordEvent(string, string) error { return errDB }

var errDB = &testErr{"db down"}

type testErr struct{ s string }

func (e *testErr) Error() string { return e.s }

// §42: multi-tab / multi-socket — same cookie resolves the SAME trial.
func TestSameCookieSameTrial(t *testing.T) {
	s := New(cfg(), NewMemStore())
	d := s.Evaluate(httptest.NewRequest("GET", "/", nil))
	token := *d.NewCookie

	r1 := reqWithCookie(httptest.NewRequest("GET", "/", nil), "pat_live_trial", token)
	r2 := reqWithCookie(httptest.NewRequest("GET", "/", nil), "pat_live_trial", token)
	d1, d2 := s.Evaluate(r1, true), s.Evaluate(r2, true)
	if d1.Trial.TokenHash != d2.Trial.TokenHash {
		t.Fatal("same cookie must resolve the same trial (multi-tab)")
	}
	if s.store.(*MemStore).Count() != 1 {
		t.Fatal("duplicate trial rows created")
	}
}

// §13: WS revalidation hook.
func TestTokenActiveRevalidation(t *testing.T) {
	s := New(cfg(), NewMemStore())
	d := s.Evaluate(httptest.NewRequest("GET", "/", nil))
	hash := d.Trial.TokenHash
	if !s.IsTokenActive(hash) {
		t.Fatal("active trial must validate")
	}
	d.Trial.ExpiresAt = time.Now().UTC().Add(-time.Second)
	s.mu.Lock()
	s.cache[hash] = d.Trial
	s.mu.Unlock()
	if s.IsTokenActive(hash) {
		t.Fatal("expired trial must fail WS revalidation")
	}
}

// §85: status payload exposes only the allowed fields.
func TestStatusPayloadShape(t *testing.T) {
	s := New(cfg(), NewMemStore())
	d := s.Evaluate(httptest.NewRequest("GET", "/", nil))
	p := s.StatusFor(d)
	if p.Status != "ACTIVE" || p.RemainingSeconds <= 0 || p.RegistrationRequired {
		t.Fatalf("unexpected payload: %+v", p)
	}
	if p.TrialExpiresAt == "" || p.ServerTime == "" {
		t.Fatal("server time + expiry are required for countdown sync")
	}
	exp := s.StatusFor(Decision{Allowed: false, Reason: ReasonExpired})
	if !exp.RegistrationRequired || exp.Code != ReasonExpired {
		t.Fatalf("expired payload wrong: %+v", exp)
	}
}

// Data endpoints never mint trials — status is the only creator (§12).
func TestDataEndpointDoesNotCreateTrial(t *testing.T) {
	s := New(cfg(), NewMemStore())
	r := httptest.NewRequest("GET", "/api/v1/market/snapshot", nil) // no cookie
	d := s.Evaluate(r, false)
	if d.Allowed {
		t.Fatal("cookie-less data request must be denied")
	}
	if s.store.(*MemStore).Count() != 0 {
		t.Fatal("data endpoint must not create trial rows")
	}
}

// §9: disabled feature flag keeps current production behavior.
func TestDisabledFlagAllowsAll(t *testing.T) {
	c := cfg()
	c.Enabled = false
	s := New(c, NewMemStore())
	d := s.Evaluate(httptest.NewRequest("GET", "/", nil))
	if !d.Allowed {
		t.Fatal("disabled preview must not block (rollback path)")
	}
}
