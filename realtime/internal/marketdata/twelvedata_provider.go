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
			"VIX":    "UVXY", // VIX proxy — ProShares Ultra VIX Short-Term Futures ETF
			"BTCUSD": "BTC/USD",
			"WTI":    "WTI",
			"EURUSD": "EUR/USD",
			"USDCHF": "USD/CHF",
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

// FetchAll fetches all configured symbols in a SINGLE batched Twelve Data /quote
// call (1 API credit instead of N) and caches them. It passes through the shared
// Twelve Data rate limiter so the combined call volume from both Twelve Data
// consumers stays under the free-tier 8-credits/minute cap.
func (p *TwelveDataProvider) FetchAll(ctx context.Context) map[string]*MacroAssetSnapshot {
	results := make(map[string]*MacroAssetSnapshot)

	if !p.IsConfigured() {
		for canonical := range p.symbols {
			results[canonical] = &MacroAssetSnapshot{
				CanonicalSymbol: canonical,
				Status:          "UNCONFIGURED",
				Source:          "twelvedata",
			}
		}
		return results
	}

	canonicals := make([]string, 0, len(p.symbols))
	providerSyms := make([]string, 0, len(p.symbols))
	for c, ps := range p.symbols {
		canonicals = append(canonicals, c)
		providerSyms = append(providerSyms, ps)
	}

	snaps, err := p.fetchBatchQuotes(ctx, providerSyms)
	if err != nil {
		// Batch failed entirely — mark all unavailable with the error.
		for _, c := range canonicals {
			status := "UNAVAILABLE"
			msg := err.Error()
			if isRateLimited(err) {
				status = "RATE_LIMITED"
				msg = "Twelve Data rate limit"
			}
			results[c] = &MacroAssetSnapshot{
				CanonicalSymbol: c, Status: status,
				ErrorMessage: msg, FetchedAt: time.Now().UTC(), Source: "twelvedata",
			}
		}
		return results
	}

	for i, c := range canonicals {
		snap, ok := snaps[providerSyms[i]]
		if !ok || snap.Status != "AVAILABLE" {
			results[c] = &MacroAssetSnapshot{
				CanonicalSymbol: c, Status: "UNAVAILABLE",
				ErrorMessage: "missing from batch response", FetchedAt: time.Now().UTC(), Source: "twelvedata",
			}
			continue
		}
		p.mu.Lock()
		if existing, ok := p.last[canonicals[i]]; ok && existing.Price > 0 {
			p.prev[canonicals[i]] = existing.Price
		}
		p.last[canonicals[i]] = snap
		p.mu.Unlock()
		results[canonicals[i]] = snap
	}
	return results
}

// fetchBatchQuotes fetches multiple symbols in a SINGLE Twelve Data /quote call
// (1 API credit for the whole batch) and returns a map of provider-symbol -> snapshot.
// Twelve Data's batched /quote returns a JSON OBJECT keyed by symbol
// ({"UVXY":{...},"BTC/USD":{...}}), NOT an array. Parse that form; fall back to a
// single-object/array if needed. Symbols containing '/' (e.g. BTC/USD) are
// URL-encoded so the request is well-formed.
func (p *TwelveDataProvider) fetchBatchQuotes(ctx context.Context, symbols []string) (map[string]*MacroAssetSnapshot, error) {
	// Respect the shared free-tier credit budget before issuing the call.
	if err := acquireTwelveDataCredit(ctx); err != nil {
		return nil, err
	}

	enc := make([]string, len(symbols))
	for i, s := range symbols {
		enc[i] = sanitizeSymbolForURL(s)
	}
	url := fmt.Sprintf("%s/quote?symbol=%s&apikey=%s&format=JSON",
		p.apiBase, strings.Join(enc, ","), p.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	body, rerr := io.ReadAll(resp.Body)
	resp.Body.Close()
	if rerr != nil {
		return nil, fmt.Errorf("read failed: %w", rerr)
	}

	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("rate_limited: HTTP 429")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Object-keyed form: {"UVXY":{...},"BTC/USD":{...}}.
	var objForm map[string]TwelveDataQuote
	if perr := json.Unmarshal(body, &objForm); perr != nil || len(objForm) == 0 {
		// Fallback: array form or single object.
		var quotes []TwelveDataQuote
		if aerr := json.Unmarshal(body, &quotes); aerr == nil && len(quotes) > 0 {
			objForm = make(map[string]TwelveDataQuote, len(quotes))
			for _, q := range quotes {
				key := q.Symbol
				if key == "" {
					continue
				}
				objForm[key] = q
			}
		} else {
			var single TwelveDataQuote
			if serr := json.Unmarshal(body, &single); serr != nil {
				return nil, fmt.Errorf("batch parse failed: %w", perr)
			}
			if single.Symbol != "" {
				objForm[single.Symbol] = single
			}
		}
	}

	out := make(map[string]*MacroAssetSnapshot, len(objForm))
	for sym, q := range objForm {
		// Skip error entries (e.g. {"VIX":{"status":"error",...}} has no close).
		if q.Close <= 0 {
			continue
		}
		ts := time.Now().UTC()
		if len(q.Timestamp) > 0 {
			tsStr := strings.Trim(string(q.Timestamp), "\"")
			if parsed, err := time.Parse("2006-01-02 15:04:05", tsStr); err == nil {
				ts = parsed.UTC()
			} else if parsed, err := time.Parse(time.RFC3339, tsStr); err == nil {
				ts = parsed.UTC()
			}
		}
		out[sym] = &MacroAssetSnapshot{
			CanonicalSymbol: sym,
			ProviderSymbol:  sym,
			Price:           q.Close,
			Open:            q.Open,
			High:            q.High,
			Low:             q.Low,
			Volume:          q.Volume,
			Timestamp:       ts,
			Source:          "twelvedata",
			Provider:        "twelvedata",
			Status:          "AVAILABLE",
			FetchedAt:       time.Now().UTC(),
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("batch returned no usable quotes")
	}
	return out, nil
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
