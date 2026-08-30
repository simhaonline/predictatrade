package marketdata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// ETFProvider fetches physically backed gold ETF (GLD, IAU) daily closes from
// Twelve Data and derives a bounded flow-direction component.
//
// Capability honesty (AGENTS.md): daily NAV/close deltas are a PROXY for ETF
// flow direction, not authoritative WGC flow tonnes. The component is therefore
// capped at moderate confidence and the source is stamped "twelvedata_etf_px".
// A future WGC feed replaces the source string and raises confidence — no
// score fabrication.
type ETFProvider struct {
	apiKey  string
	apiBase string
	client  *http.Client
	mu      sync.RWMutex
	last    *ETFState
	symbols []string // e.g. {"GLD", "IAU"}
}

// ETFConfig configures the ETF flow proxy provider.
type ETFConfig struct {
	APIKey       string
	APIBase      string
	Symbols      []string
	RefreshHours int
	TimeoutSec   int
}

// DefaultETFConfig returns GLD+IAU defaults (disabled unless API key set).
func DefaultETFConfig() ETFConfig {
	return ETFConfig{
		Symbols:      []string{"GLD", "IAU"},
		RefreshHours: 24,
		TimeoutSec:   15,
	}
}

// ETFState is the derived ETF component state.
type ETFState struct {
	Status       string  // AVAILABLE, UNAVAILABLE, UNCONFIGURED
	Direction    float64 // -100..+100 gold-flow direction
	Raw          string  // human-auditable raw observation
	Timestamp    time.Time
	Source       string
	ErrorMessage string
}

// NewETFProvider creates the provider.
func NewETFProvider(cfg ETFConfig) *ETFProvider {
	if cfg.APIBase == "" {
		cfg.APIBase = "https://api.twelvedata.com"
	}
	if cfg.TimeoutSec <= 0 {
		cfg.TimeoutSec = 15
	}
	if len(cfg.Symbols) == 0 {
		cfg.Symbols = []string{"GLD", "IAU"}
	}
	return &ETFProvider{
		apiKey:  cfg.APIKey,
		apiBase: cfg.APIBase,
		symbols: cfg.Symbols,
		client:  &http.Client{Timeout: time.Duration(cfg.TimeoutSec) * time.Second},
	}
}

// IsConfigured reports whether an API key is present.
func (p *ETFProvider) IsConfigured() bool { return p.apiKey != "" }

// GetState returns the cached state.
func (p *ETFProvider) GetState() *ETFState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.last
}

// Update fetches fresh ETF closes and derives the flow-direction component.
func (p *ETFProvider) Update(ctx context.Context) error {
	if !p.IsConfigured() {
		p.setState(&ETFState{Status: "UNCONFIGURED", ErrorMessage: "TWELVEDATA_API_KEY not set", Source: "twelvedata_etf_px"})
		return nil
	}

	totalImpact := 0.0
	var errs []string
	var parts []string
	anyOK := false
	latest := time.Time{}

	for _, sym := range p.symbols {
		ret, ts, err := p.fetchDailyReturn(ctx, sym)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", sym, err))
			continue
		}
		anyOK = true
		// Daily % move mapped to bounded impact: ±2% → ±40 (capped).
		imp := ret * 20
		if imp > 40 {
			imp = 40
		}
		if imp < -40 {
			imp = -40
		}
		totalImpact += imp
		parts = append(parts, fmt.Sprintf("%s %.2f%%", sym, ret))
		if ts.After(latest) {
			latest = ts
		}
	}

	if !anyOK {
		st := &ETFState{
			Status:       "UNAVAILABLE",
			ErrorMessage: joinErrs(errs),
			Source:       "twelvedata_etf_px",
			Timestamp:    time.Now().UTC(),
		}
		p.setState(st)
		return fmt.Errorf("ETF provider unavailable: %s", st.ErrorMessage)
	}

	avg := totalImpact / float64(len(p.symbols))
	st := &ETFState{
		Status:    "AVAILABLE",
		Direction: avg,
		Raw:       "1d ETF px change: " + joinStrings(parts),
		Timestamp: latest,
		Source:    "twelvedata_etf_px",
	}
	p.setState(st)
	return nil
}

func (p *ETFProvider) setState(s *ETFState) {
	p.mu.Lock()
	p.last = s
	p.mu.Unlock()
}

// fetchDailyReturn fetches the latest two daily closes and returns pct change.
func (p *ETFProvider) fetchDailyReturn(ctx context.Context, symbol string) (pct float64, ts time.Time, err error) {
	url := fmt.Sprintf("%s/time_series?symbol=%s&interval=1day&outputsize=2&order=ASC&apikey=%s",
		p.apiBase, symbol, p.apiKey)
	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if reqErr != nil {
		return 0, time.Time{}, reqErr
	}
	resp, doErr := p.client.Do(req)
	if doErr != nil {
		return 0, time.Time{}, doErr
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return 0, time.Time{}, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var parsed struct {
		Values []struct {
			Datetime string `json:"datetime"`
			Close    string `json:"close"`
		} `json:"values"`
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	if decErr := json.NewDecoder(resp.Body).Decode(&parsed); decErr != nil {
		return 0, time.Time{}, decErr
	}
	if parsed.Status != "ok" || len(parsed.Values) < 2 {
		return 0, time.Time{}, fmt.Errorf("insufficient data (%s)", parsed.Message)
	}
	prev, prevErr := parseClose(parsed.Values[0].Close)
	cur, curErr := parseClose(parsed.Values[1].Close)
	if prevErr != nil || curErr != nil || prev <= 0 {
		return 0, time.Time{}, fmt.Errorf("bad close values")
	}
	ts, tsErr := time.Parse("2006-01-02", parsed.Values[1].Datetime)
	if tsErr != nil {
		ts = time.Now().UTC()
	}
	return (cur - prev) / prev * 100, ts.UTC(), nil
}

func parseClose(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

func joinErrs(errs []string) string {
	out := ""
	for i, e := range errs {
		if i > 0 {
			out += "; "
		}
		out += e
	}
	return out
}

func joinStrings(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}

// StartRefreshLoop periodically refreshes the ETF component.
func (p *ETFProvider) StartRefreshLoop(ctx context.Context, logFn func(msg string, err error)) {
	if !p.IsConfigured() {
		if logFn != nil {
			logFn("ETF provider not configured (TWELVEDATA_API_KEY missing) — etf_flows stays UNAVAILABLE", nil)
		}
		return
	}
	hours := 24
	if p.client != nil {
		if err := p.Update(ctx); err != nil && logFn != nil {
			logFn("ETF initial fetch failed", err)
		} else if st := p.GetState(); st != nil && st.Status == "AVAILABLE" && logFn != nil {
			logFn("ETF flow snapshot: "+st.Raw, nil)
		}
	}
	_ = hours
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := p.Update(ctx); err != nil && logFn != nil {
				logFn("ETF refresh failed", err)
			}
		}
	}
}

// ETPComponent is the marketdata-side ETF observation. The igs package
// consumes this via its fan-in adapter (no import cycle).
type ETPComponent struct {
	Name       string
	RawValue   float64
	Impact     float64
	Confidence float64
	Quality    string
	Source     string
	Reason     string
	Timestamp  time.Time
	Available  bool
}

// ETFComponent builds the marketdata observation from current state.
// Proxy daily-close data: capped at 40 impact and 0.4 confidence until a
// WGC authoritative flow feed replaces it.
func (p *ETFProvider) ETFComponent() ETPComponent {
	st := p.GetState()
	if st == nil || st.Status != "AVAILABLE" {
		return ETPComponent{Name: "etf_flows", Available: false, Source: "twelvedata_etf_px"}
	}
	return ETPComponent{
		Name:       "etf_flows",
		RawValue:   st.Direction,
		Impact:     clampDailyImp(st.Direction),
		Confidence: 0.4,
		Quality:    "CONNECTED",
		Source:     st.Source,
		Reason:     st.Raw,
		Timestamp:  st.Timestamp,
		Available:  true,
	}
}

func clampDailyImp(v float64) float64 {
	if v > 40 {
		return 40
	}
	if v < -40 {
		return -40
	}
	return v
}
