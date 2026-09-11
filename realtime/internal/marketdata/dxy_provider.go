// Package marketdata — DXY (US Dollar Index) provider adapter.
// Fetches DXY component currencies from Twelve Data API and computes
// the ICE US Dollar Index using the official weighted geometric mean formula.
//
// SOW Section 8: DXY is macro context used in strategy confluence.
// STANDARD_SWING: macro_dxy_yield is a mandatory pillar (weight 20).
// TREND_SWING: macro_real_yield_dxy is a mandatory pillar (weight 20).
//
// Fail-safe: If the API is unavailable, rate-limited (429), or returns errors,
// the provider marks DXY as UNAVAILABLE — it NEVER fabricates data.
// The CorrelationEngine and strategy pillars return UNKNOWN/zero —
// signals degrade to NO-TRADE for mandatory DXY pillars, which is correct.
package marketdata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"
)

// DXYProviderConfig holds DXY provider configuration.
type DXYProviderConfig struct {
	APIKey       string
	APIBase      string
	RefreshMin   int    // How often to fetch (DXY doesn't change much intraday)
	TimeoutSec   int
}

// DefaultDXYConfig returns safe defaults for DXY provider configuration.
func DefaultDXYConfig() DXYProviderConfig {
	return DXYProviderConfig{
		APIKey:     "", // Must be supplied via TWELVEDATA_API_KEY environment variable
		APIBase:    "https://api.twelvedata.com",
		RefreshMin: 5,  // 5-minute refresh (6 API calls per refresh, well within 8/min rate limit)
		TimeoutSec: 15,
	}
}

// DXYSnapshot is the processed DXY data.
type DXYSnapshot struct {
	Value     float64   // Computed DXY index value
	Components map[string]float64 // Individual currency pair prices
	FetchedAt time.Time
	Source    string
	Status    string  // AVAILABLE, STALE, UNAVAILABLE, UNCONFIGURED, RATE_LIMITED
	ErrorMessage string
}

// dxyComponents maps each currency pair to its ICE DXY weight.
// The DXY formula: DXY = 50.14348112 × EURUSD^(-0.576) × USDJPY^(0.136) ×
//   GBPUSD^(-0.119) × USDCAD^(0.091) × USDSEK^(0.042) × USDCHF^(0.036)
var dxyComponents = map[string]struct {
	symbol string
	weight float64 // exponent in the geometric mean (negative for quote-currency USD)
}{
	"EUR/USD": {symbol: "EUR/USD", weight: -0.576},
	"USD/JPY": {symbol: "USD/JPY", weight: 0.136},
	"GBP/USD": {symbol: "GBP/USD", weight: -0.119},
	"USD/CAD": {symbol: "USD/CAD", weight: 0.091},
	"USD/SEK": {symbol: "USD/SEK", weight: 0.042},
	"USD/CHF": {symbol: "USD/CHF", weight: 0.036},
}

const dxyBaseFactor = 50.14348112

// DXYProvider fetches currency data from Twelve Data and computes DXY.
type DXYProvider struct {
	config            DXYProviderConfig
	mu                sync.RWMutex
	last              *DXYSnapshot
	client            *http.Client
	prevValue         float64
	snapshotCallbacks []func(value, prevValue float64, ts time.Time)
}

// NewDXYProvider creates a new DXY provider.
func NewDXYProvider(cfg DXYProviderConfig) *DXYProvider {
	if cfg.APIBase == "" {
		cfg.APIBase = "https://api.twelvedata.com"
	}
	if cfg.TimeoutSec <= 0 {
		cfg.TimeoutSec = 15
	}
	return &DXYProvider{
		config: cfg,
		client: &http.Client{Timeout: time.Duration(cfg.TimeoutSec) * time.Second},
	}
}

// IsConfigured returns true when an API key is present.
func (p *DXYProvider) IsConfigured() bool {
	return p.config.APIKey != ""
}

// GetSnapshot returns the latest cached DXY snapshot (thread-safe).
func (p *DXYProvider) GetSnapshot() *DXYSnapshot {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.last
}

// FetchDXY fetches all 6 component currencies and computes DXY.
//
// It fetches every component in a SINGLE batched Twelve Data /price call
// (symbol=EUR/USD,USD/JPY,...) — 1 API credit instead of 6 — and passes through
// the shared Twelve Data rate limiter so the combined call volume from both
// Twelve Data consumers stays under the free-tier 8-credits/minute cap.
func (p *DXYProvider) FetchDXY(ctx context.Context) (*DXYSnapshot, error) {
	if !p.IsConfigured() {
		return &DXYSnapshot{
			Status:       "UNCONFIGURED",
			ErrorMessage: "TWELVEDATA_API_KEY not set — DXY provider not configured",
			Source:       "twelvedata",
		}, nil
	}

	// Build the batched symbol list from the DXY component map.
	symbols := make([]string, 0, len(dxyComponents))
	for _, comp := range dxyComponents {
		symbols = append(symbols, comp.symbol)
	}

	prices, rateLimited, err := p.fetchBatchPrices(ctx, symbols)
	if err != nil && len(prices) == 0 {
		status := "UNAVAILABLE"
		msg := err.Error()
		if rateLimited {
			status = "RATE_LIMITED"
			msg = "Twelve Data API rate limit reached — DXY temporarily unavailable"
		}
		return &DXYSnapshot{
			Status:       status,
			ErrorMessage: msg,
			Source:       "twelvedata",
			FetchedAt:    time.Now().UTC(),
		}, err
	}

	// All 6 components are required to compute DXY.
	if len(prices) < len(dxyComponents) {
		status := "UNAVAILABLE"
		msg := fmt.Sprintf("incomplete DXY components: got %d/%d", len(prices), len(dxyComponents))
		if rateLimited {
			status = "RATE_LIMITED"
			msg = "Twelve Data API rate limit reached — DXY temporarily unavailable"
		}
		return &DXYSnapshot{
			Status:       status,
			ErrorMessage: msg,
			Source:       "twelvedata",
			FetchedAt:    time.Now().UTC(),
		}, fmt.Errorf("%s", msg)
	}

	// Map provider symbol back to DXY pair key and compute DXY.
	components := make(map[string]float64, len(dxyComponents))
	for pair, comp := range dxyComponents {
		price, ok := prices[comp.symbol]
		if !ok || price <= 0 {
			return &DXYSnapshot{
				Status:       "UNAVAILABLE",
				ErrorMessage: fmt.Sprintf("invalid/missing price for %s", pair),
				Source:       "twelvedata",
				FetchedAt:    time.Now().UTC(),
			}, fmt.Errorf("invalid price for %s", pair)
		}
		components[pair] = price
	}

	// All 6 components are required to compute DXY
	if len(components) < len(dxyComponents) {
		status := "UNAVAILABLE"
		msg := fmt.Sprintf("incomplete DXY components: got %d/%d", len(components), len(dxyComponents))
		if rateLimited {
			status = "RATE_LIMITED"
			msg = "Twelve Data API rate limit reached — DXY temporarily unavailable"
		}
		return &DXYSnapshot{
			Status:       status,
			ErrorMessage: msg,
			Source:       "twelvedata",
			FetchedAt:    time.Now().UTC(),
		}, fmt.Errorf("%s", msg)
	}

	// Compute DXY using ICE formula:
	// DXY = 50.14348112 × EURUSD^(-0.576) × USDJPY^(0.136) × GBPUSD^(-0.119) ×
	//       USDCAD^(0.091) × USDSEK^(0.042) × USDCHF^(0.036)
	dxy := dxyBaseFactor
	for pair, comp := range dxyComponents {
		price := components[pair]
		if price <= 0 {
			return &DXYSnapshot{
				Status:       "UNAVAILABLE",
				ErrorMessage: fmt.Sprintf("invalid price for %s: %f", pair, price),
				Source:       "twelvedata",
				FetchedAt:    time.Now().UTC(),
			}, fmt.Errorf("invalid price for %s", pair)
		}
		dxy *= powFloat(price, comp.weight)
	}

	return &DXYSnapshot{
		Value:      dxy,
		Components: components,
		FetchedAt:  time.Now().UTC(),
		Source:     "twelvedata",
		Status:     "AVAILABLE",
	}, nil
}

// fetchBatchPrices fetches multiple symbols in a SINGLE Twelve Data /price call
// (1 API credit for the whole batch) and returns a map of symbol -> price.
// It passes through the shared Twelve Data rate limiter so the combined call
// volume from both Twelve Data consumers stays under the free-tier cap.
//
// Twelve Data returns an ARRAY of {symbol, price} objects when multiple symbols
// are requested. We retry on HTTP 429 (rate limit) with exponential backoff so a
// transient burst throttle self-heals instead of permanently failing the cycle.
func (p *DXYProvider) fetchBatchPrices(ctx context.Context, symbols []string) (map[string]float64, bool, error) {
	const maxRetries = 3
	prices := make(map[string]float64)
	var lastErr error
	var rateLimited bool

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second // 1s, 2s, 4s
			select {
			case <-ctx.Done():
				return prices, rateLimited, ctx.Err()
			case <-time.After(backoff):
			}
		}

		// Respect the shared free-tier credit budget before issuing the call.
		if err := acquireTwelveDataCredit(ctx); err != nil {
			return prices, rateLimited, err
		}

		url := fmt.Sprintf("%s/price?symbol=%s&apikey=%s",
			p.config.APIBase, strings.Join(symbols, ","), p.config.APIKey)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return prices, rateLimited, err
		}
		resp, err := p.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("DXY batch fetch failed: %w", err)
			continue
		}
		body, rerr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if rerr != nil {
			lastErr = fmt.Errorf("DXY batch read failed: %w", rerr)
			continue
		}

		// Twelve Data signals rate limiting via HTTP 429 or an error-status body.
		if resp.StatusCode == 429 {
			rateLimited = true
			lastErr = fmt.Errorf("rate_limited: HTTP 429")
			continue
		}
		var apiErr struct {
			Status  string `json:"status"`
			Code    int    `json:"code"`
			Message string `json:"message"`
		}
		if jerr := json.Unmarshal(body, &apiErr); jerr == nil && apiErr.Status == "error" {
			if apiErr.Code == 429 {
				rateLimited = true
			}
			lastErr = fmt.Errorf("API error %d: %s", apiErr.Code, apiErr.Message)
			if rateLimited {
				continue
			}
			return prices, rateLimited, lastErr
		}

		// Successful response. Twelve Data's batched /price returns a JSON OBJECT
		// keyed by symbol: {"EUR/USD":{"price":"1.16"},"USD/JPY":{"price":"153.5"},...}
		// (not an array). Parse that form; fall back to an array if needed.
		var objForm map[string]struct {
			Price string `json:"price"`
		}
		if perr := json.Unmarshal(body, &objForm); perr == nil && len(objForm) > 0 {
			got := 0
			for sym, v := range objForm {
				var price float64
				if _, serr := fmt.Sscanf(v.Price, "%f", &price); serr == nil && price > 0 {
					prices[sym] = price
					got++
				}
			}
			if got == 0 {
				lastErr = fmt.Errorf("DXY batch returned no usable prices")
				return prices, rateLimited, lastErr
			}
			return prices, rateLimited, nil
		}

		// Fallback: some endpoints return an array of {symbol, price}.
		var quotes []struct {
			Symbol string `json:"symbol"`
			Price  string `json:"price"`
		}
		if perr := json.Unmarshal(body, &quotes); perr != nil {
			lastErr = fmt.Errorf("DXY batch parse failed: %w", perr)
			return prices, rateLimited, lastErr
		}
		got := 0
		for _, q := range quotes {
			var price float64
			if _, serr := fmt.Sscanf(q.Price, "%f", &price); serr == nil && price > 0 {
				prices[q.Symbol] = price
				got++
			}
		}
		if got == 0 {
			lastErr = fmt.Errorf("DXY batch returned no usable prices")
			return prices, rateLimited, lastErr
		}
		return prices, rateLimited, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("DXY batch fetch failed after retries")
	}
	return prices, rateLimited, lastErr
}

// Update fetches fresh DXY data and updates the cached snapshot.
func (p *DXYProvider) Update(ctx context.Context) error {
	snap, err := p.FetchDXY(ctx)
	if err != nil {
		p.mu.Lock()
		p.last = snap
		p.mu.Unlock()
		return err
	}

	p.mu.Lock()
	p.last = snap
	p.mu.Unlock()
	return nil
}

// OnSnapshot registers a callback that fires when a new DXY snapshot is available.
// The callback receives the current and previous DXY values.
func (p *DXYProvider) OnSnapshot(cb func(value, prevValue float64, ts time.Time)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.snapshotCallbacks = append(p.snapshotCallbacks, cb)
}

// fireSnapshotCallbacks invokes every registered OnSnapshot callback with the
// given snapshot and advances prevValue. Shared by the initial fetch AND the
// refresh loop (stale-DXY fix, 2026-09-10: the refresh loop previously never
// fired callbacks, so crossmarket/IGS drivers wired via OnSnapshot saw a
// Timestamp frozen at startup — DXY showed permanently STALE on Macro
// Intelligence while the provider itself was healthy).
func (p *DXYProvider) fireSnapshotCallbacks(value float64, ts time.Time) {
	p.mu.RLock()
	prev := p.prevValue
	callbacks := make([]func(float64, float64, time.Time), len(p.snapshotCallbacks))
	copy(callbacks, p.snapshotCallbacks)
	p.mu.RUnlock()
	for _, cb := range callbacks {
		cb(value, prev, ts)
	}
	p.mu.Lock()
	p.prevValue = value
	p.mu.Unlock()
}

// StartRefreshLoop runs a background goroutine that periodically fetches DXY data.
// On each successful fetch, it calls the onUpdate callback with the DXY value
// and timestamp — this is used to feed the CorrelationEngine.
func (p *DXYProvider) StartRefreshLoop(ctx context.Context, onUpdate func(value float64, ts time.Time), logFn func(msg string, err error)) {
	if !p.IsConfigured() {
		if logFn != nil {
			logFn("DXY provider not configured — TWELVEDATA_API_KEY not set (DXY remains UNAVAILABLE, mandatory DXY pillars will fail closed → NO-TRADE)", nil)
		}
		return
	}

	// Initial fetch
	if err := p.Update(ctx); err != nil {
		if logFn != nil {
			logFn("DXY initial fetch failed", err)
		}
	} else {
		snap := p.GetSnapshot()
		if snap != nil && snap.Status == "AVAILABLE" {
			if logFn != nil {
				logFn(fmt.Sprintf("DXY data fetched: value=%.4f status=%s", snap.Value, snap.Status), nil)
			}
			if onUpdate != nil {
				onUpdate(snap.Value, snap.FetchedAt)
			}
			// Fire snapshot callbacks for cross-market engine
			p.fireSnapshotCallbacks(snap.Value, snap.FetchedAt)
		} else if snap != nil {
			if logFn != nil {
				logFn(fmt.Sprintf("DXY fetch returned %s: %s", snap.Status, snap.ErrorMessage), nil)
			}
		}
	}

	refreshInterval := time.Duration(p.config.RefreshMin) * time.Minute
	if refreshInterval <= 0 {
		refreshInterval = 5 * time.Minute
	}
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := p.Update(ctx); err != nil {
				if logFn != nil {
					logFn("DXY refresh failed", err)
				}
			} else {
				snap := p.GetSnapshot()
				if snap != nil && snap.Status == "AVAILABLE" {
					if logFn != nil {
						logFn(fmt.Sprintf("DXY refreshed: value=%.4f", snap.Value), nil)
					}
					if onUpdate != nil {
						onUpdate(snap.Value, snap.FetchedAt)
					}
					// Fire snapshot callbacks on REFRESH too (stale-DXY fix,
					// 2026-09-10): the crossmarket engine + IGS fan-in are
					// wired via OnSnapshot (main.go), but this loop previously
					// only fired them on the INITIAL fetch. Result: the DXY
					// driver's Timestamp froze at startup while the provider
					// kept refreshing — freshness decayed to 0 and Macro
					// Intelligence showed DXY permanently STALE with
					// effective_weight 0 (observed live 2026-09-10).
					p.fireSnapshotCallbacks(snap.Value, snap.FetchedAt)
				}
			}
		}
	}
}

// isRateLimited checks if an error is a rate limit error.
func isRateLimited(err error) bool {
	if err == nil {
		return false
	}
	return containsStr(err.Error(), "rate_limited") || containsStr(err.Error(), "429")
}

// powFloat computes x^y using the standard math library.
func powFloat(x, y float64) float64 {
	return math.Pow(x, y)
}
