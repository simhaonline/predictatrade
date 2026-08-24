// Package marketdata — FRED (Federal Reserve Economic Data) Real Yield provider.
// Fetches 10-Year Treasury Inflation-Indexed Security yield (real yield / TIPS).
//
// FRED series: DFII10 — 10-Year Treasury Inflation-Indexed Security, Constant Maturity
// This is the REAL yield, not nominal. Semantic type: REAL_YIELD.
//
// Fail-safe: If FRED_API_KEY is not set, provider returns UNCONFIGURED.
// The Cross-Market Engine continues safely without real yield data.
package marketdata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// FredProvider fetches real yield data from the FRED API.
type FredProvider struct {
	mu           sync.RWMutex
	apiKey       string
	seriesID     string
	apiBase      string
	client       *http.Client
	lastValue    float64
	lastDate     string
	lastFetch    time.Time
	prevValue    float64
	status       string // AVAILABLE, STALE, UNCONFIGURED, UNAVAILABLE
	errorMsg     string
}

// FredObservation is a single FRED data point.
type FredObservation struct {
	Date  string `json:"date"`
	Value string `json:"value"`
}

// FredSeriesResponse is the FRED API response.
type FredSeriesResponse struct {
	Observations []FredObservation `json:"observations"`
}

// NewFredProvider creates a FRED real yield provider.
func NewFredProvider(apiKey, seriesID string) *FredProvider {
	if seriesID == "" {
		seriesID = "DFII10" // 10-Year Treasury Inflation-Indexed Security
	}
	return &FredProvider{
		apiKey:   apiKey,
		seriesID: seriesID,
		apiBase:  "https://api.stlouisfed.org/fred",
		client:   &http.Client{Timeout: 15 * time.Second},
		status:   "UNCONFIGURED",
	}
}

// IsConfigured returns true when an API key is present.
func (p *FredProvider) IsConfigured() bool {
	return p.apiKey != ""
}

// GetStatus returns the current provider status.
func (p *FredProvider) GetStatus() (string, string) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status, p.errorMsg
}

// GetLatest returns the latest real yield value and date.
func (p *FredProvider) GetLatest() (float64, string, time.Time) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lastValue, p.lastDate, p.lastFetch
}

// GetPrevValue returns the previous real yield value for change calculation.
func (p *FredProvider) GetPrevValue() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.prevValue
}

// FetchLatest fetches the latest real yield observation from FRED.
func (p *FredProvider) FetchLatest(ctx context.Context) error {
	if !p.IsConfigured() {
		p.mu.Lock()
		p.status = "UNCONFIGURED"
		p.errorMsg = "FRED_API_KEY not set"
		p.mu.Unlock()
		return fmt.Errorf("FRED_API_KEY not set")
	}

	// FRED API: series/observations
	url := fmt.Sprintf("%s/series/observations?series_id=%s&api_key=%s&file_type=json&sort_order=desc&limit=5",
		p.apiBase, p.seriesID, p.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		p.setError("UNAVAILABLE", err.Error())
		return err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		p.setError("UNAVAILABLE", err.Error())
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("FRED API HTTP %d: %s", resp.StatusCode, string(body))
		p.setError("UNAVAILABLE", errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	var fredResp FredSeriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&fredResp); err != nil {
		p.setError("UNAVAILABLE", fmt.Sprintf("JSON decode error: %v", err))
		return err
	}

	if len(fredResp.Observations) == 0 {
		p.setError("UNAVAILABLE", "no observations returned")
		return fmt.Errorf("no observations")
	}

	// Parse the latest observation
	latest := fredResp.Observations[0]
	value, err := strconv.ParseFloat(latest.Value, 64)
	if err != nil {
		p.setError("UNAVAILABLE", fmt.Sprintf("value parse error: %v", err))
		return err
	}

	// Parse the second-latest for change calculation
	prevValue := 0.0
	if len(fredResp.Observations) > 1 {
		prevValue, _ = strconv.ParseFloat(fredResp.Observations[1].Value, 64)
	}

	p.mu.Lock()
	p.prevValue = p.lastValue
	if p.prevValue == 0 {
		p.prevValue = prevValue
	}
	p.lastValue = value
	p.lastDate = latest.Date
	p.lastFetch = time.Now().UTC()
	p.status = "AVAILABLE"
	p.errorMsg = ""
	p.mu.Unlock()

	return nil
}

// StartRefreshLoop runs a background goroutine that periodically fetches real yield data.
// Real yield is published daily on trading days, so we use a longer interval.
func (p *FredProvider) StartRefreshLoop(ctx context.Context, intervalMin int, onUpdate func(value float64, date string, ts time.Time), logFn func(msg string, err error)) {
	if !p.IsConfigured() {
		if logFn != nil {
			logFn("FRED provider not configured — FRED_API_KEY not set (Real Yield remains UNCONFIGURED, does not block signal generation)", nil)
		}
		return
	}

	if intervalMin <= 0 {
		intervalMin = 60 // Real yield is daily data — check hourly
	}

	// Initial fetch
	if err := p.FetchLatest(ctx); err != nil {
		if logFn != nil {
			logFn("FRED initial fetch failed", err)
		}
	} else {
		val, date, ts := p.GetLatest()
		if logFn != nil {
			logFn(fmt.Sprintf("FRED real yield fetched: value=%.2f%% date=%s series=%s", val, date, p.seriesID), nil)
		}
		if onUpdate != nil {
			onUpdate(val, date, ts)
		}
	}

	ticker := time.NewTicker(time.Duration(intervalMin) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := p.FetchLatest(ctx); err != nil {
				if logFn != nil {
					logFn("FRED fetch failed", err)
				}
			} else {
				val, date, ts := p.GetLatest()
				if onUpdate != nil {
					onUpdate(val, date, ts)
				}
			}
		}
	}
}

func (p *FredProvider) setError(status, msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status = status
	p.errorMsg = msg
}
