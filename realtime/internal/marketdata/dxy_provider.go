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
	config DXYProviderConfig
	mu     sync.RWMutex
	last   *DXYSnapshot
	client *http.Client
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
func (p *DXYProvider) FetchDXY(ctx context.Context) (*DXYSnapshot, error) {
	if !p.IsConfigured() {
		return &DXYSnapshot{
			Status:       "UNCONFIGURED",
			ErrorMessage: "TWELVEDATA_API_KEY not set — DXY provider not configured",
			Source:       "twelvedata",
		}, nil
	}

	components := make(map[string]float64)
	rateLimited := false

	for pair, comp := range dxyComponents {
		price, err := p.fetchPrice(ctx, comp.symbol)
		if err != nil {
			// Check for rate limiting
			if isRateLimited(err) {
				rateLimited = true
			}
			// Continue with remaining pairs — partial data is not useful for DXY
			// (all 6 components are required)
			continue
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

// fetchPrice fetches a single currency pair price from Twelve Data /price endpoint.
func (p *DXYProvider) fetchPrice(ctx context.Context, symbol string) (float64, error) {
	url := fmt.Sprintf("%s/price?symbol=%s&apikey=%s",
		p.config.APIBase, symbol, p.config.APIKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("DXY fetch %s failed: %w", symbol, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("DXY read %s failed: %w", symbol, err)
	}

	// Check for API errors
	var apiErr struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Status == "error" {
		if apiErr.Code == 429 {
			return 0, fmt.Errorf("rate_limited: %s", apiErr.Message)
		}
		return 0, fmt.Errorf("API error %d: %s", apiErr.Code, apiErr.Message)
	}

	// Parse price response
	var priceResp struct {
		Price string `json:"price"`
	}
	if err := json.Unmarshal(body, &priceResp); err != nil {
		return 0, fmt.Errorf("DXY parse %s failed: %w", symbol, err)
	}

	var price float64
	if _, err := fmt.Sscanf(priceResp.Price, "%f", &price); err != nil {
		return 0, fmt.Errorf("DXY parse price %s: %w", symbol, err)
	}

	return price, nil
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
