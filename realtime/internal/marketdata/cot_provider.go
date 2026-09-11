// Package marketdata — COT (Commitment of Traders) provider adapter.
// Fetches COT reports from Financial Modeling Prep (FMP) API.
//
// SOW Section 8: COT is macro/positioning context, NOT execution-critical.
// COT weight = 0 for all strategies by default. It is an optional pillar.
//
// Fail-safe: If the API is unavailable, restricted (402), or returns errors,
// the provider marks COT as UNAVAILABLE — it NEVER fabricates data.
// Signal generation continues; the cot_etf_flow pillar simply contributes 0.
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

// COTProviderConfig holds COT provider configuration.
type COTProviderConfig struct {
	APIKey       string
	APIBase      string // FMP base URL
	CFTCAPIBase  string // CFTC Socrata base URL (fallback; empty → official endpoint)
	Symbol       string // CFTC futures contract code, e.g. "GC" for gold
	RefreshHours int    // How often to fetch (COT is weekly data)
	TimeoutSec   int
}

// DefaultCOTConfig returns safe defaults for COT provider configuration.
func DefaultCOTConfig() COTProviderConfig {
	return COTProviderConfig{
		APIKey:       "", // Must be supplied via FMP_API_KEY environment variable
		APIBase:      "https://financialmodelingprep.com",
		Symbol:       "GC", // Gold futures
		RefreshHours: 6,    // Check every 6 hours (COT is published weekly)
		TimeoutSec:   30,
	}
}

// COTReport represents a single COT report row from the FMP API.
type COTReport struct {
	Date                     string  `json:"date"`
	Symbol                   string  `json:"symbol"`
	Name                     string  `json:"name"`
	Sector                   string  `json:"sector"`
	OpenInterestAll          int64   `json:"openInterestAll"`
	NoncommPositionsLongAll  int64   `json:"noncommPositionsLongAll"`
	NoncommPositionsShortAll int64   `json:"noncommPositionsShortAll"`
	CommPositionsLongAll     int64   `json:"commPositionsLongAll"`
	CommPositionsShortAll    int64   `json:"commPositionsShortAll"`
	PctOfOiNoncommLongAll    float64 `json:"pctOfOiNoncommLongAll"`
	PctOfOiNoncommShortAll   float64 `json:"pctOfOiNoncommShortAll"`
	TradersNoncommLongAll    int64   `json:"tradersNoncommLongAll"`
	TradersNoncommShortAll   int64   `json:"tradersNoncommShortAll"`
}

// COTSnapshot is the processed COT data for use in signal scoring.
type COTSnapshot struct {
	ReportDate    time.Time
	NetPosition   int64   // non-commercial long - short
	NetPercentile float64 // 0-1 percentile of recent net positioning
	NetZScore     float64 // z-score of net positioning
	OpenInterest  int64
	CommercialNet int64 // commercial long - short (hedger positioning)
	FetchedAt     time.Time
	Source        string
	Status        string // AVAILABLE, STALE, UNAVAILABLE, UNCONFIGURED
	ErrorMessage  string
}

// COTProvider fetches and processes COT data from FMP API (with the official
// CFTC public API as fallback).
type COTProvider struct {
	config            COTProviderConfig
	mu                sync.RWMutex
	lastSnap          *COTSnapshot
	client            *http.Client
	snapshotCallbacks []func(netPosition float64, percentile float64, ts time.Time)

	// lastSourceOverride records the upstream that served the most recent
	// report batch ("cftc" when the CFTC fallback succeeded, "" → FMP).
	lastSourceOverride string
}

// NewCOTProvider creates a new COT provider.
func NewCOTProvider(cfg COTProviderConfig) *COTProvider {
	if cfg.APIBase == "" {
		cfg.APIBase = "https://financialmodelingprep.com"
	}
	if cfg.TimeoutSec <= 0 {
		cfg.TimeoutSec = 30
	}
	return &COTProvider{
		config: cfg,
		client: &http.Client{Timeout: time.Duration(cfg.TimeoutSec) * time.Second},
	}
}

// IsConfigured returns true when an API key is present.
func (p *COTProvider) IsConfigured() bool {
	return p.config.APIKey != ""
}

// FetchReport fetches COT data from the FMP API.
// Tries the stable endpoint first, then the legacy v4 endpoint, then the
// CFTC official public Socrata API (free, no key) as final fallback.
func (p *COTProvider) FetchReport(ctx context.Context) (*COTSnapshot, error) {
	if !p.IsConfigured() {
		return &COTSnapshot{
			Status:       "UNCONFIGURED",
			ErrorMessage: "FMP_API_KEY not set — COT provider not configured",
			Source:       "fmp",
		}, nil
	}

	// Try stable endpoint first
	reports, err := p.fetchStable(ctx)
	if err != nil {
		// Try legacy v4 endpoint as fallback
		fmpErr := err
		reports, err = p.fetchLegacyV4(ctx)
		if err != nil {
			// FMP is unavailable on this subscription/endpoint — fall back to the
			// CFTC official public API so the pillar stays alive without FMP.
			reports, err = p.fetchCFTCSocrata(ctx)
			if err != nil {
				p.mu.Lock()
				p.lastSourceOverride = ""
				p.mu.Unlock()
				return &COTSnapshot{
					Status:       "UNAVAILABLE",
					ErrorMessage: fmt.Sprintf("fmp: %v; cftc: %v", fmpErr, err),
					Source:       "fmp",
					FetchedAt:    time.Now().UTC(),
				}, err
			}
			p.mu.Lock()
			p.lastSourceOverride = "cftc"
			p.mu.Unlock()
		} else {
			p.mu.Lock()
			p.lastSourceOverride = ""
			p.mu.Unlock()
		}
	} else {
		p.mu.Lock()
		p.lastSourceOverride = ""
		p.mu.Unlock()
	}

	if len(reports) == 0 {
		return &COTSnapshot{
			Status:       "UNAVAILABLE",
			ErrorMessage: "no COT data returned for symbol " + p.config.Symbol,
			Source:       "fmp",
			FetchedAt:    time.Now().UTC(),
		}, fmt.Errorf("no COT data returned")
	}

	// Use the most recent report (first in the list). Provenance: the CFTC
	// fallback sets p.lastSourceOverride so the snapshot records the real
	// upstream instead of pretending it came from FMP.
	latest := reports[0]
	p.mu.RLock()
	source := p.lastSourceOverride
	p.mu.RUnlock()
	if source == "" {
		source = "fmp"
	}
	snap := p.processReport(latest, reports, source)
	return snap, nil
}

// fetchStable fetches from the stable COT endpoint.
func (p *COTProvider) fetchStable(ctx context.Context) ([]COTReport, error) {
	url := fmt.Sprintf("%s/stable/commitment-of-traders-report?symbol=%s&apikey=%s",
		p.config.APIBase, p.config.Symbol, p.config.APIKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("COT stable request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("COT stable read failed: %w", err)
	}

	if resp.StatusCode == 402 {
		return nil, fmt.Errorf("COT endpoint restricted (HTTP 402) — subscription tier does not include COT data")
	}
	if resp.StatusCode != 200 {
		bodyStr := string(body)
		if len(bodyStr) > 200 {
			bodyStr = bodyStr[:200]
		}
		return nil, fmt.Errorf("COT stable HTTP %d: %s", resp.StatusCode, bodyStr)
	}

	var reports []COTReport
	if err := json.Unmarshal(body, &reports); err != nil {
		return nil, fmt.Errorf("COT stable parse failed: %w", err)
	}

	return reports, nil
}

// fetchLegacyV4 fetches from the legacy v4 COT endpoint.
func (p *COTProvider) fetchLegacyV4(ctx context.Context) ([]COTReport, error) {
	url := fmt.Sprintf("%s/api/v4/commitment_of_traders_report/%s?apikey=%s",
		p.config.APIBase, p.config.Symbol, p.config.APIKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("COT v4 request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("COT v4 read failed: %w", err)
	}

	if resp.StatusCode == 402 {
		return nil, fmt.Errorf("COT endpoint restricted (HTTP 402) — subscription tier does not include COT data")
	}
	if resp.StatusCode != 200 {
		bodyStr := string(body)
		if len(bodyStr) > 200 {
			bodyStr = bodyStr[:200]
		}
		return nil, fmt.Errorf("COT v4 HTTP %d: %s", resp.StatusCode, bodyStr)
	}

	var reports []COTReport
	if err := json.Unmarshal(body, &reports); err != nil {
		return nil, fmt.Errorf("COT v4 parse failed: %w", err)
	}

	return reports, nil
}

// cftcContractCodes maps the provider's futures symbol to the CFTC
// cftc_contract_market_code used by the official public Socrata datasets.
// COMMODITIES (fut_hist_txt, dataset key 6dca-aqww).
var cftcContractCodes = map[string]string{
	"GC": "088691", // Gold COMEX
	"SI": "084691", // Silver COMEX
	"HG": "085692", // Copper COMEX
	"CL": "067651", // Light Crude NYMEX
	"NG": "023651", // Natural Gas NYMEX
}

// cftcSocrataBase is the official CFTC public reporting endpoint (Socrata).
const cftcSocrataBase = "https://publicreporting.cftc.gov"

// fetchCFTCSocrata fetches COT history directly from the CFTC official public
// API — free, no API key, no subscription. Used as the final fallback when FMP
// is unavailable (402 restricted / 403 legacy-killed). The legacy
// futures-only dataset (fut_hist_txt, Socrata 6dca-aqww) carries the same
// noncommercial/commercial/open-interest fields FMP's COT reports expose.
// Rows come oldest-first and are returned oldest-first, matching
// processReport's expectations (it computes percentile/z-score over the whole
// series and uses reports[0] as the latest, so we reverse to newest-first).
func (p *COTProvider) fetchCFTCSocrata(ctx context.Context) ([]COTReport, error) {
	code, ok := cftcContractCodes[p.config.Symbol]
	if !ok {
		return nil, fmt.Errorf("CFTC fallback: no contract code mapped for symbol %q", p.config.Symbol)
	}

	base := p.config.CFTCAPIBase
	if base == "" {
		base = cftcSocrataBase
	}
	url := fmt.Sprintf("%s/resource/6dca-aqww.json?$select=report_date_as_yyyy_mm_dd,noncomm_positions_long_all,noncomm_positions_short_all,comm_positions_long_all,comm_positions_short_all,open_interest_all&cftc_contract_market_code=%s&$order=report_date_as_yyyy_mm_dd%%20DESC&$limit=160",
		base, code)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("CFTC request failed: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("CFTC request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("CFTC read failed: %w", err)
	}

	if resp.StatusCode != 200 {
		bodyStr := string(body)
		if len(bodyStr) > 200 {
			bodyStr = bodyStr[:200]
		}
		return nil, fmt.Errorf("CFTC HTTP %d: %s", resp.StatusCode, bodyStr)
	}

	// Socrata returns every field as a JSON string (e.g. "261007").
	var rows []struct {
		Date     string `json:"report_date_as_yyyy_mm_dd"`
		NcLong   string `json:"noncomm_positions_long_all"`
		NcShort  string `json:"noncomm_positions_short_all"`
		CommLong string `json:"comm_positions_long_all"`
		CommShrt string `json:"comm_positions_short_all"`
		OI       string `json:"open_interest_all"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("CFTC parse failed: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("CFTC returned no rows for contract %s", code)
	}

	atoi64 := func(s string) int64 {
		var v int64
		fmt.Sscanf(strings.TrimSpace(s), "%d", &v)
		return v
	}

	// Socrata $order DESC → newest-first, same shape as the FMP stable feed.
	reports := make([]COTReport, 0, len(rows))
	for _, r := range rows {
		reports = append(reports, COTReport{
			Date:                     strings.Split(r.Date, "T")[0],
			Symbol:                   p.config.Symbol,
			OpenInterestAll:          atoi64(r.OI),
			NoncommPositionsLongAll:  atoi64(r.NcLong),
			NoncommPositionsShortAll: atoi64(r.NcShort),
			CommPositionsLongAll:     atoi64(r.CommLong),
			CommPositionsShortAll:    atoi64(r.CommShrt),
		})
	}
	return reports, nil
}

// processReport computes COT features from raw report data.
func (p *COTProvider) processReport(latest COTReport, history []COTReport, source string) *COTSnapshot {
	reportDate, _ := time.Parse("2006-01-02", latest.Date)
	if reportDate.IsZero() {
		reportDate = time.Now().UTC()
	}

	netPos := latest.NoncommPositionsLongAll - latest.NoncommPositionsShortAll
	commercialNet := latest.CommPositionsLongAll - latest.CommPositionsShortAll

	// Compute percentile and z-score from history (up to 156 weeks = 3 years)
	netSeries := make([]int64, 0, len(history))
	for _, r := range history {
		// Legacy v4 returns oldest-first; stable returns newest-first
		netSeries = append(netSeries, r.NoncommPositionsLongAll-r.NoncommPositionsShortAll)
	}

	percentile := computePercentile(netPos, netSeries)
	zScore := computeZScore(netPos, netSeries)

	return &COTSnapshot{
		ReportDate:    reportDate,
		NetPosition:   netPos,
		NetPercentile: percentile,
		NetZScore:     zScore,
		OpenInterest:  latest.OpenInterestAll,
		CommercialNet: commercialNet,
		FetchedAt:     time.Now().UTC(),
		Source:        source,
		Status:        "AVAILABLE",
	}
}

// GetSnapshot returns the latest cached COT snapshot (thread-safe).
// OnSnapshot registers a callback for cross-market engine integration.
func (p *COTProvider) OnSnapshot(cb func(netPosition float64, percentile float64, ts time.Time)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.snapshotCallbacks = append(p.snapshotCallbacks, cb)
}

func (p *COTProvider) GetSnapshot() *COTSnapshot {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lastSnap
}

// Update fetches fresh COT data and updates the cached snapshot.
func (p *COTProvider) Update(ctx context.Context) error {
	snap, err := p.FetchReport(ctx)
	if err != nil {
		p.mu.Lock()
		p.lastSnap = snap // Store error snapshot
		p.mu.Unlock()
		return err
	}

	p.mu.Lock()
	p.lastSnap = snap
	callbacks := make([]func(float64, float64, time.Time), len(p.snapshotCallbacks))
	copy(callbacks, p.snapshotCallbacks)
	p.mu.Unlock()

	// Fire snapshot callbacks for cross-market engine
	if snap.Status == "AVAILABLE" {
		for _, cb := range callbacks {
			cb(float64(snap.NetPosition), snap.NetPercentile, snap.ReportDate)
		}
	}
	return nil
}

// StartRefreshLoop runs a background goroutine that periodically fetches COT data.
func (p *COTProvider) StartRefreshLoop(ctx context.Context, logFn func(msg string, err error)) {
	if !p.IsConfigured() {
		if logFn != nil {
			logFn("COT provider not configured — FMP_API_KEY not set (COT remains UNAVAILABLE, does not block signal generation)", nil)
		}
		return
	}

	// Initial fetch
	if err := p.Update(ctx); err != nil {
		if logFn != nil {
			logFn("COT initial fetch failed", err)
		}
	} else {
		if logFn != nil {
			snap := p.GetSnapshot()
			if snap != nil {
				logFn(fmt.Sprintf("COT data fetched: report_date=%s net_position=%d percentile=%.2f status=%s",
					snap.ReportDate.Format("2006-01-02"), snap.NetPosition, snap.NetPercentile, snap.Status), nil)
			}
		}
	}

	ticker := time.NewTicker(time.Duration(p.config.RefreshHours) * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := p.Update(ctx); err != nil {
				if logFn != nil {
					logFn("COT refresh failed", err)
				}
			} else {
				if logFn != nil {
					snap := p.GetSnapshot()
					if snap != nil && snap.Status == "AVAILABLE" {
						logFn(fmt.Sprintf("COT refreshed: report_date=%s net_position=%d status=%s",
							snap.ReportDate.Format("2006-01-02"), snap.NetPosition, snap.Status), nil)
					}
				}
			}
		}
	}
}

// computePercentile computes the percentile of a value within a series (0-1).
func computePercentile(value int64, series []int64) float64 {
	if len(series) == 0 {
		return 0.5 // neutral
	}
	below := 0
	for _, v := range series {
		if v < value {
			below++
		}
	}
	return float64(below) / float64(len(series))
}

// computeZScore computes the z-score of a value within a series.
func computeZScore(value int64, series []int64) float64 {
	if len(series) < 2 {
		return 0
	}
	var sum float64
	for _, v := range series {
		sum += float64(v)
	}
	mean := sum / float64(len(series))

	var sumSqDiff float64
	for _, v := range series {
		diff := float64(v) - mean
		sumSqDiff += diff * diff
	}
	stdDev := sqrtFloat(sumSqDiff / float64(len(series)))
	if stdDev == 0 {
		return 0
	}
	return (float64(value) - mean) / stdDev
}

func sqrtFloat(x float64) float64 {
	if x <= 0 {
		return 0
	}
	// Newton's method
	z := x
	for i := 0; i < 20; i++ {
		z = (z + x/z) / 2
	}
	return z
}

// redactAPIKey removes the API key from error/URL strings for safe logging.
func redactAPIKey(s, key string) string {
	if key == "" {
		return s
	}
	return strings.ReplaceAll(s, key, "***REDACTED***")
}
