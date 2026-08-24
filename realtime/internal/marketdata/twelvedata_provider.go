// Package marketdata — Twelve Data multi-symbol provider for macro assets.
// Fetches VIX, BTCUSD, WTI Oil, and EURUSD from Twelve Data API.
// Reuses the same API key as the DXY provider — no duplicate credentials.
package marketdata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// TwelveDataQuote is a single symbol quote from Twelve Data API.
type TwelveDataQuote struct {
	Symbol      string  `json:"symbol"`
	Close       float64 `json:"close,string"`
	Open        float64 `json:"open,string"`
	High        float64 `json:"high,string"`
	Low         float64 `json:"low,string"`
	Volume      float64 `json:"volume,string"`
	Timestamp   json.RawMessage `json:"timestamp"`
	IsMarketOpen bool   `json:"is_market_open"`
}

// MacroAssetSnapshot is a normalized observation of a macro asset.
type MacroAssetSnapshot struct {
	CanonicalSymbol string    `json:"canonical_symbol"`
	ProviderSymbol  string    `json:"provider_symbol"`
	Price           float64   `json:"price"`
	Open            float64   `json:"open"`
	High            float64   `json:"high"`
	Low             float64   `json:"low"`
	Volume          float64   `json:"volume"`
	Timestamp       time.Time `json:"timestamp"`
	Source          string    `json:"source"`
	Provider        string    `json:"provider"`
	Status          string    `json:"status"` // AVAILABLE, STALE, UNAVAILABLE, UNCONFIGURED
	ErrorMessage    string    `json:"error_message,omitempty"`
	FetchedAt       time.Time `json:"fetched_at"`
}

// TwelveDataProvider fetches macro asset quotes from Twelve Data API.
// It uses the same API key as the DXY provider and supports configurable symbols.
type TwelveDataProvider struct {
	mu       sync.RWMutex
	apiKey   string
	apiBase  string
	client   *http.Client
	symbols  map[string]string // canonical -> provider symbol
	last     map[string]*MacroAssetSnapshot
	prev     map[string]float64 // previous price for change calculation
}

// NewTwelveDataProvider creates a provider for macro assets.
func NewTwelveDataProvider(apiKey string) *TwelveDataProvider {
	if apiKey == "" {
		return &TwelveDataProvider{symbols: map[string]string{}, last: map[string]*MacroAssetSnapshot{}, prev: map[string]float64{}}
	}
	return &TwelveDataProvider{
		apiKey:  apiKey,
		apiBase: "https://api.twelvedata.com",
		client:  &http.Client{Timeout: 15 * time.Second},
		symbols: map[string]string{
			"VIX":    "VIX",
			"BTCUSD": "BTC/USD",
			"WTI":    "WTI",
			"EURUSD": "EUR/USD",
		},
		last: map[string]*MacroAssetSnapshot{},
		prev: map[string]float64{},
	}
}

// IsConfigured returns true when an API key is present.
func (p *TwelveDataProvider) IsConfigured() bool {
	return p.apiKey != ""
}

// GetSnapshot returns the latest cached snapshot for a canonical symbol.
func (p *TwelveDataProvider) GetSnapshot(canonical string) *MacroAssetSnapshot {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.last[canonical]
}

// GetPrevPrice returns the previous price for change calculation.
func (p *TwelveDataProvider) GetPrevPrice(canonical string) float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.prev[canonical]
}

// FetchSymbol fetches a single symbol quote from Twelve Data.
func (p *TwelveDataProvider) FetchSymbol(ctx context.Context, canonical string) (*MacroAssetSnapshot, error) {
	if !p.IsConfigured() {
		return &MacroAssetSnapshot{
			CanonicalSymbol: canonical,
			Status:          "UNCONFIGURED",
			Source:          "twelvedata",
		}, nil
	}

	providerSymbol, ok := p.symbols[canonical]
	if !ok {
		providerSymbol = canonical
	}

	url := fmt.Sprintf("%s/quote?symbol=%s&apikey=%s&format=JSON",
		p.apiBase, providerSymbol, p.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return &MacroAssetSnapshot{CanonicalSymbol: canonical, Status: "UNAVAILABLE", ErrorMessage: err.Error(), FetchedAt: time.Now().UTC()}, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return &MacroAssetSnapshot{CanonicalSymbol: canonical, Status: "UNAVAILABLE", ErrorMessage: err.Error(), FetchedAt: time.Now().UTC()}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return &MacroAssetSnapshot{CanonicalSymbol: canonical, Status: "RATE_LIMITED", ErrorMessage: "Twelve Data rate limit", FetchedAt: time.Now().UTC()}, fmt.Errorf("rate limited")
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return &MacroAssetSnapshot{CanonicalSymbol: canonical, Status: "UNAVAILABLE", ErrorMessage: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)), FetchedAt: time.Now().UTC()}, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var quote TwelveDataQuote
	if err := json.NewDecoder(resp.Body).Decode(&quote); err != nil {
		return &MacroAssetSnapshot{CanonicalSymbol: canonical, Status: "UNAVAILABLE", ErrorMessage: err.Error(), FetchedAt: time.Now().UTC()}, err
	}

	// Parse timestamp
	ts := time.Now().UTC()
	if len(quote.Timestamp) > 0 {
		tsStr := string(quote.Timestamp)
		tsStr = strings.Trim(tsStr, "\"")
		if parsed, err := time.Parse("2006-01-02 15:04:05", tsStr); err == nil {
			ts = parsed.UTC()
		} else if parsed, err := time.Parse(time.RFC3339, tsStr); err == nil {
			ts = parsed.UTC()
		}
	}

	snap := &MacroAssetSnapshot{
		CanonicalSymbol: canonical,
		ProviderSymbol:  providerSymbol,
		Price:           quote.Close,
		Open:            quote.Open,
		High:            quote.High,
		Low:             quote.Low,
		Volume:          quote.Volume,
		Timestamp:       ts,
		Source:          "twelvedata",
		Provider:        "twelvedata",
		Status:          "AVAILABLE",
		FetchedAt:       time.Now().UTC(),
	}

	// Validate price
	if snap.Price <= 0 {
		snap.Status = "UNAVAILABLE"
		snap.ErrorMessage = "zero or negative price"
		return snap, fmt.Errorf("invalid price for %s", canonical)
	}

	return snap, nil
}

// FetchAll fetches all configured symbols and caches them.
func (p *TwelveDataProvider) FetchAll(ctx context.Context) map[string]*MacroAssetSnapshot {
	results := make(map[string]*MacroAssetSnapshot)
	for canonical := range p.symbols {
		snap, err := p.FetchSymbol(ctx, canonical)
		if err != nil {
			results[canonical] = snap
			continue
		}
		p.mu.Lock()
		// Store previous price before updating
		if existing, ok := p.last[canonical]; ok && existing.Price > 0 {
			p.prev[canonical] = existing.Price
		}
		p.last[canonical] = snap
		p.mu.Unlock()
		results[canonical] = snap
	}
	return results
}

// StartRefreshLoop runs a background goroutine that periodically fetches all symbols.
func (p *TwelveDataProvider) StartRefreshLoop(ctx context.Context, intervalMin int, onUpdate func(canonical string, snap *MacroAssetSnapshot), logFn func(msg string, err error)) {
	if !p.IsConfigured() {
		if logFn != nil {
			logFn("TwelveDataProvider not configured — TWELVEDATA_API_KEY not set (VIX/BTC/Oil/EURUSD remain UNAVAILABLE)", nil)
		}
		return
	}

	if intervalMin <= 0 {
		intervalMin = 5
	}

	// Initial fetch
	results := p.FetchAll(ctx)
	for canonical, snap := range results {
		if snap.Status == "AVAILABLE" {
			if logFn != nil {
				logFn(fmt.Sprintf("%s data fetched: price=%.2f status=%s", canonical, snap.Price, snap.Status), nil)
			}
			if onUpdate != nil {
				onUpdate(canonical, snap)
			}
		} else if logFn != nil {
			logFn(fmt.Sprintf("%s fetch returned %s: %s", canonical, snap.Status, snap.ErrorMessage), nil)
		}
	}

	ticker := time.NewTicker(time.Duration(intervalMin) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			results := p.FetchAll(ctx)
			for canonical, snap := range results {
				if snap.Status == "AVAILABLE" {
					if onUpdate != nil {
						onUpdate(canonical, snap)
					}
				}
			}
		}
	}
}

// ExtractEURUSDFromDXY extracts the EURUSD price from a DXY snapshot's components.
// This avoids a duplicate API call — EUR/USD is already fetched as part of DXY calculation.
func ExtractEURUSDFromDXY(dxySnap *DXYSnapshot) *MacroAssetSnapshot {
	if dxySnap == nil || dxySnap.Components == nil {
		return &MacroAssetSnapshot{
			CanonicalSymbol: "EURUSD",
			Status:          "UNAVAILABLE",
			Source:          "dxy_component",
		}
	}

	eurPrice, ok := dxySnap.Components["EUR/USD"]
	if !ok || eurPrice <= 0 {
		return &MacroAssetSnapshot{
			CanonicalSymbol: "EURUSD",
			Status:          "UNAVAILABLE",
			Source:          "dxy_component",
		}
	}

	return &MacroAssetSnapshot{
		CanonicalSymbol: "EURUSD",
		ProviderSymbol:  "EUR/USD",
		Price:           eurPrice,
		Timestamp:       dxySnap.FetchedAt,
		Source:          "dxy_component",
		Provider:        "twelvedata",
		Status:          "AVAILABLE",
		FetchedAt:       dxySnap.FetchedAt,
	}
}

// sanitizeSymbolForURL ensures the symbol is safe for URL construction.
func sanitizeSymbolForURL(s string) string {
	return strings.ReplaceAll(s, "/", "%2F")
}
