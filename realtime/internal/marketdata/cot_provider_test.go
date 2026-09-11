package marketdata

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Test COT provider fails safely when not configured
func TestCOTProviderNotConfigured(t *testing.T) {
	cfg := DefaultCOTConfig()
	cfg.APIKey = ""
	provider := NewCOTProvider(cfg)

	if provider.IsConfigured() {
		t.Error("Provider should not be configured without API key")
	}

	snap, err := provider.FetchReport(context.Background())
	if err != nil {
		t.Errorf("Unconfigured provider should return snapshot without error, got: %v", err)
	}
	if snap == nil {
		t.Fatal("Unconfigured provider should return a snapshot")
	}
	if snap.Status != "UNCONFIGURED" {
		t.Errorf("Expected status UNCONFIGURED, got %s", snap.Status)
	}
	if snap.ErrorMessage == "" {
		t.Error("Should have error message explaining missing API key")
	}
}

// Test COT provider handles 402 restricted endpoint (fail safe)
func TestCOTProviderHandlesRestricted(t *testing.T) {
	// FMP test server returns 402; the CFTC fallback test server returns 500.
	// Both upstreams down → provider must fail safe (UNAVAILABLE), never fabricate.
	fmp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		w.Write([]byte("Restricted Endpoint: This endpoint is not available under your current subscription"))
	}))
	defer fmp.Close()
	cftc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":true}`))
	}))
	defer cftc.Close()

	cfg := COTProviderConfig{
		APIKey:      "test_key",
		APIBase:     fmp.URL,
		CFTCAPIBase: cftc.URL,
		Symbol:      "GC",
	}
	provider := NewCOTProvider(cfg)

	snap, err := provider.FetchReport(context.Background())
	if err == nil {
		t.Error("Should return error for restricted endpoint")
	}
	if snap == nil {
		t.Fatal("Should return a snapshot even on error")
	}
	if snap.Status != "UNAVAILABLE" {
		t.Errorf("Expected status UNAVAILABLE, got %s", snap.Status)
	}
}

// Test COT provider falls back to the CFTC official public API when FMP is
// restricted (402/403). The snapshot must be AVAILABLE with Source="cftc" —
// this is the regression for the 2026-09-11 permanent COT UNAVAILABLE wall
// (FMP stable=402 restricted, v4=403 legacy endpoint killed).
func TestCOTProviderFallsBackToCFTC(t *testing.T) {
	fmp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		w.Write([]byte("Restricted Endpoint"))
	}))
	defer fmp.Close()

	// CFTC Socrata mock: newest-first rows, string-typed fields (Socrata shape).
	cftc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("cftc_contract_market_code"); got != "088691" {
			t.Errorf("expected gold contract code 088691, got %q", got)
		}
		w.Write([]byte(`[
			{"report_date_as_yyyy_mm_dd":"2026-09-08T00:00:00.000","noncomm_positions_long_all":"261007","noncomm_positions_short_all":"29047","comm_positions_long_all":"54403","comm_positions_short_all":"324677","open_interest_all":"411227"},
			{"report_date_as_yyyy_mm_dd":"2026-09-01T00:00:00.000","noncomm_positions_long_all":"260485","noncomm_positions_short_all":"32361","comm_positions_long_all":"61450","comm_positions_short_all":"326168","open_interest_all":"415196"}
		]`))
	}))
	defer cftc.Close()

	cfg := COTProviderConfig{
		APIKey:      "test_key",
		APIBase:     fmp.URL,
		CFTCAPIBase: cftc.URL,
		Symbol:      "GC",
	}
	provider := NewCOTProvider(cfg)

	snap, err := provider.FetchReport(context.Background())
	if err != nil {
		t.Fatalf("expected CFTC fallback success, got error: %v", err)
	}
	if snap.Status != "AVAILABLE" {
		t.Errorf("expected AVAILABLE via CFTC fallback, got %s (%s)", snap.Status, snap.ErrorMessage)
	}
	if snap.Source != "cftc" {
		t.Errorf("expected Source cftc, got %s", snap.Source)
	}
	// Newest row: noncomm net = 261007 - 29047 = 231960.
	if snap.NetPosition != 231960 {
		t.Errorf("expected net position 231960, got %d", snap.NetPosition)
	}
	if snap.OpenInterest != 411227 {
		t.Errorf("expected OI 411227, got %d", snap.OpenInterest)
	}
	if !snap.ReportDate.Equal(time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("expected report date 2026-09-08, got %s", snap.ReportDate.Format("2006-01-02"))
	}
}

// Test COT provider parses valid API response
func TestCOTProviderParsesValidResponse(t *testing.T) {
	reports := []COTReport{
		{
			Date:                     "2026-08-12",
			Symbol:                   "GC",
			Name:                     "GOLD - COMEX",
			OpenInterestAll:          500000,
			NoncommPositionsLongAll:  200000,
			NoncommPositionsShortAll: 100000,
			CommPositionsLongAll:     150000,
			CommPositionsShortAll:    180000,
		},
		{
			Date:                     "2026-08-05",
			Symbol:                   "GC",
			OpenInterestAll:          490000,
			NoncommPositionsLongAll:  190000,
			NoncommPositionsShortAll: 110000,
			CommPositionsLongAll:     140000,
			CommPositionsShortAll:    170000,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify API key is in the request
		if r.URL.Query().Get("apikey") != "test_key" {
			t.Error("API key should be in request")
		}
		// Verify symbol is in the request
		if r.URL.Query().Get("symbol") != "GC" {
			t.Error("Symbol should be in request")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(reports)
	}))
	defer ts.Close()

	cfg := COTProviderConfig{
		APIKey:  "test_key",
		APIBase: ts.URL,
		Symbol:  "GC",
	}
	provider := NewCOTProvider(cfg)

	snap, err := provider.FetchReport(context.Background())
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
	if snap == nil {
		t.Fatal("Should return a snapshot")
	}
	if snap.Status != "AVAILABLE" {
		t.Errorf("Expected status AVAILABLE, got %s", snap.Status)
	}
	if snap.NetPosition != 100000 {
		t.Errorf("Expected net position 100000 (200000-100000), got %d", snap.NetPosition)
	}
	if snap.CommercialNet != -30000 {
		t.Errorf("Expected commercial net -30000 (150000-180000), got %d", snap.CommercialNet)
	}
	if snap.OpenInterest != 500000 {
		t.Errorf("Expected open interest 500000, got %d", snap.OpenInterest)
	}
	if snap.Source != "fmp" {
		t.Errorf("Expected source fmp, got %s", snap.Source)
	}
}

// Test COT provider handles empty response (fail safe, no fabrication)
func TestCOTProviderEmptyResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]COTReport{})
	}))
	defer ts.Close()

	cfg := COTProviderConfig{
		APIKey:  "test_key",
		APIBase: ts.URL,
		Symbol:  "GC",
	}
	provider := NewCOTProvider(cfg)

	snap, err := provider.FetchReport(context.Background())
	if err == nil {
		t.Error("Should return error for empty response")
	}
	if snap == nil {
		t.Fatal("Should return a snapshot")
	}
	if snap.Status != "UNAVAILABLE" {
		t.Errorf("Expected status UNAVAILABLE, got %s", snap.Status)
	}
}

// Test COT percentile computation
func TestComputePercentile(t *testing.T) {
	series := []int64{10, 20, 30, 40, 50}

	p := computePercentile(30, series)
	if p < 0.39 || p > 0.41 {
		t.Errorf("Expected percentile ~0.4 for value 30, got %.2f", p)
	}

	p = computePercentile(5, series)
	if p != 0 {
		t.Errorf("Expected percentile 0 for value below all, got %.2f", p)
	}

	p = computePercentile(60, series)
	if p != 1 {
		t.Errorf("Expected percentile 1 for value above all, got %.2f", p)
	}

	// Empty series → neutral 0.5
	p = computePercentile(100, []int64{})
	if p != 0.5 {
		t.Errorf("Expected 0.5 for empty series, got %.2f", p)
	}
}

// Test COT z-score computation
func TestComputeZScore(t *testing.T) {
	series := []int64{10, 20, 30, 40, 50}

	z := computeZScore(30, series)
	// mean = 30, so z-score should be 0
	if z > 0.01 || z < -0.01 {
		t.Errorf("Expected z-score ~0 for mean value, got %.4f", z)
	}

	z = computeZScore(50, series)
	// 50 is above mean, z should be positive
	if z <= 0 {
		t.Errorf("Expected positive z-score for above-mean value, got %.4f", z)
	}

	// Too few values
	z = computeZScore(100, []int64{50})
	if z != 0 {
		t.Errorf("Expected 0 for single-value series, got %.4f", z)
	}
}

// Test COT snapshot caching
func TestCOTSnapshotCaching(t *testing.T) {
	cfg := DefaultCOTConfig()
	cfg.APIKey = "test"
	provider := NewCOTProvider(cfg)

	// Initially nil
	if provider.GetSnapshot() != nil {
		t.Error("Initial snapshot should be nil")
	}

	// Set a snapshot manually
	provider.mu.Lock()
	provider.lastSnap = &COTSnapshot{
		Status:      "AVAILABLE",
		NetPosition: 50000,
	}
	provider.mu.Unlock()

	snap := provider.GetSnapshot()
	if snap == nil {
		t.Fatal("Should return cached snapshot")
	}
	if snap.NetPosition != 50000 {
		t.Errorf("Expected net position 50000, got %d", snap.NetPosition)
	}
}

// Test redactAPIKey
func TestRedactAPIKey(t *testing.T) {
	original := "https://financialmodelingprep.com/api?apikey=SECRET_KEY_123&symbol=GC"
	redacted := redactAPIKey(original, "SECRET_KEY_123")
	if redacted == original {
		t.Error("API key should be redacted")
	}
	if containsStr(redacted, "SECRET_KEY_123") {
		t.Error("Redacted string should not contain the API key")
	}
}
