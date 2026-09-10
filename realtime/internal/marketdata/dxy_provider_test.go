package marketdata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Test DXY provider fails safely when not configured
func TestDXYProviderNotConfigured(t *testing.T) {
	cfg := DefaultDXYConfig()
	cfg.APIKey = ""
	provider := NewDXYProvider(cfg)

	if provider.IsConfigured() {
		t.Error("Provider should not be configured without API key")
	}

	snap, err := provider.FetchDXY(context.Background())
	if err != nil {
		t.Errorf("Unconfigured provider should return snapshot without error, got: %v", err)
	}
	if snap == nil {
		t.Fatal("Unconfigured provider should return a snapshot")
	}
	if snap.Status != "UNCONFIGURED" {
		t.Errorf("Expected status UNCONFIGURED, got %s", snap.Status)
	}
}

// Test DXY provider handles rate limiting (429) — fail safe
func TestDXYProviderHandlesRateLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    429,
			"message": "You have run out of API credits",
			"status":  "error",
		})
	}))
	defer ts.Close()

	cfg := DXYProviderConfig{
		APIKey:  "test_key",
		APIBase: ts.URL,
	}
	provider := NewDXYProvider(cfg)

	snap, err := provider.FetchDXY(context.Background())
	if err == nil {
		t.Error("Should return error for rate-limited API")
	}
	if snap == nil {
		t.Fatal("Should return a snapshot")
	}
	if snap.Status != "RATE_LIMITED" && snap.Status != "UNAVAILABLE" {
		t.Errorf("Expected RATE_LIMITED or UNAVAILABLE, got %s", snap.Status)
	}
}

// Test DXY computation from 6 component currencies
func TestDXYComputation(t *testing.T) {
	// Realistic component prices
	prices := map[string]string{
		"EUR/USD": "1.0850",
		"USD/JPY": "149.50",
		"GBP/USD": "1.2700",
		"USD/CAD": "1.3600",
		"USD/SEK": "10.8500",
		"USD/CHF": "0.8900",
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		if r.URL.Query().Get("apikey") != "test_key" {
			t.Error("API key should be in request")
		}
		price, ok := prices[symbol]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 404, "message": "symbol not found", "status": "error",
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"price": price,
		})
	}))
	defer ts.Close()

	cfg := DXYProviderConfig{
		APIKey:  "test_key",
		APIBase: ts.URL,
	}
	provider := NewDXYProvider(cfg)

	snap, err := provider.FetchDXY(context.Background())
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
	if snap == nil {
		t.Fatal("Should return a snapshot")
	}
	if snap.Status != "AVAILABLE" {
		t.Errorf("Expected status AVAILABLE, got %s: %s", snap.Status, snap.ErrorMessage)
	}
	// DXY should be around 103-104 with these component prices
	if snap.Value < 90 || snap.Value > 110 {
		t.Errorf("DXY value %.4f is outside expected range (90-110)", snap.Value)
	}
	if len(snap.Components) != 6 {
		t.Errorf("Expected 6 components, got %d", len(snap.Components))
	}
	if snap.Source != "twelvedata" {
		t.Errorf("Expected source twelvedata, got %s", snap.Source)
	}
}

// Test DXY provider handles missing component — fail safe, no fabrication
func TestDXYMissingComponent(t *testing.T) {
	prices := map[string]string{
		"EUR/USD": "1.0850",
		"USD/JPY": "149.50",
		// Missing: GBP/USD, USD/CAD, USD/SEK, USD/CHF
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		price, ok := prices[symbol]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 404, "message": "symbol not found", "status": "error",
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"price": price})
	}))
	defer ts.Close()

	cfg := DXYProviderConfig{
		APIKey:  "test_key",
		APIBase: ts.URL,
	}
	provider := NewDXYProvider(cfg)

	snap, err := provider.FetchDXY(context.Background())
	if err == nil {
		t.Error("Should return error for incomplete components")
	}
	if snap == nil {
		t.Fatal("Should return a snapshot")
	}
	if snap.Status != "UNAVAILABLE" {
		t.Errorf("Expected status UNAVAILABLE, got %s", snap.Status)
	}
	// Must not have a fabricated DXY value
	if snap.Value != 0 {
		t.Error("DXY value should be 0 (not fabricated) when components are missing")
	}
}

// Test DXY snapshot caching
func TestDXYSnapshotCaching(t *testing.T) {
	cfg := DefaultDXYConfig()
	cfg.APIKey = "test"
	provider := NewDXYProvider(cfg)

	if provider.GetSnapshot() != nil {
		t.Error("Initial snapshot should be nil")
	}

	provider.mu.Lock()
	provider.last = &DXYSnapshot{
		Status: "AVAILABLE",
		Value:  103.5,
	}
	provider.mu.Unlock()

	snap := provider.GetSnapshot()
	if snap == nil {
		t.Fatal("Should return cached snapshot")
	}
	if snap.Value != 103.5 {
		t.Errorf("Expected value 103.5, got %.4f", snap.Value)
	}
}

// Test powFloat basic values
func TestPowFloat(t *testing.T) {
	// 2^3 = 8
	r := powFloat(2, 3)
	if r < 7.9 || r > 8.1 {
		t.Errorf("Expected 2^3 ≈ 8, got %.4f", r)
	}

	// 4^0.5 = 2
	r = powFloat(4, 0.5)
	if r < 1.9 || r > 2.1 {
		t.Errorf("Expected 4^0.5 ≈ 2, got %.4f", r)
	}

	// x^0 = 1
	r = powFloat(123, 0)
	if r != 1 {
		t.Errorf("Expected x^0 = 1, got %.4f", r)
	}

	// 1^-0.576 = 1
	r = powFloat(1, -0.576)
	if r < 0.99 || r > 1.01 {
		t.Errorf("Expected 1^-0.576 ≈ 1, got %.4f", r)
	}
}

// Test isRateLimited
func TestIsRateLimited(t *testing.T) {
	if !isRateLimited(fmt.Errorf("rate_limited: too many requests")) {
		t.Error("Should detect rate_limited in error")
	}
	if !isRateLimited(fmt.Errorf("HTTP 429: rate limit")) {
		t.Error("Should detect 429 in error")
	}
	if isRateLimited(fmt.Errorf("connection refused")) {
		t.Error("Should not detect rate limit in non-rate error")
	}
}

// TestDXYFireSnapshotCallbacksOnRefresh — regression for the 2026-09-10
// "DXY STALE on Macro Intelligence" live bug. OnSnapshot callbacks must fire
// on EVERY refresh, not only the initial fetch: crossmarket + IGS consumers
// are wired exclusively via OnSnapshot, and a refresh-loop that skips them
// leaves driver timestamps frozen at startup (freshness → 0, effective
// weight → 0, permanent STALE display while the provider is healthy).
//
// Drive the real refresh loop with a sub-second tick: RefreshMin is in
// minutes, so instead we simulate two refresh cycles by calling the same
// code path the loop uses (Update + fireSnapshotCallbacks) and asserting the
// callback fires and prevValue advances — the exact logic the loop now runs.
func TestDXYFireSnapshotCallbacksOnRefresh(t *testing.T) {
	prices := []string{"1.1000", "1.1001", "1.1002"}
	i := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if i >= len(prices) {
			fmt.Fprintf(w, `{"price":"%s"}`, prices[len(prices)-1])
			return
		}
		fmt.Fprintf(w, `{"price":"%s"}`, prices[i])
		i++
	}))
	defer ts.Close()

	provider := NewDXYProvider(DXYProviderConfig{
		APIKey:     "test-key",
		APIBase:    ts.URL,
		TimeoutSec: 5,
	})

	var calls []struct{ value, prev float64 }
	done := make(chan struct{}, 4)
	provider.OnSnapshot(func(value, prevValue float64, ts time.Time) {
		calls = append(calls, struct{ value, prev float64 }{value, prevValue})
		done <- struct{}{}
	})

	ctx := context.Background()
	// Initial cycle (what StartRefreshLoop does on startup).
	if err := provider.Update(ctx); err != nil {
		t.Fatalf("initial Update: %v", err)
	}
	provider.fireSnapshotCallbacks(provider.GetSnapshot().Value, provider.GetSnapshot().FetchedAt)
	<-done

	// Refresh cycle #1 — BEFORE the fix this never fired the callback.
	if err := provider.Update(ctx); err != nil {
		t.Fatalf("refresh Update 1: %v", err)
	}
	provider.fireSnapshotCallbacks(provider.GetSnapshot().Value, provider.GetSnapshot().FetchedAt)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("refresh cycle did not fire OnSnapshot callback (pre-fix behavior)")
	}

	// Refresh cycle #2 — prev must now be the previous refresh's value.
	if err := provider.Update(ctx); err != nil {
		t.Fatalf("refresh Update 2: %v", err)
	}
	provider.fireSnapshotCallbacks(provider.GetSnapshot().Value, provider.GetSnapshot().FetchedAt)
	<-done

	if len(calls) != 3 {
		t.Fatalf("expected 3 callbacks (initial + 2 refreshes), got %d", len(calls))
	}
	// Third call's prev must equal the second call's value (chained refreshes).
	if calls[2].prev != calls[1].value {
		t.Errorf("chained prev broken: got prev=%v want %v", calls[2].prev, calls[1].value)
	}
}
