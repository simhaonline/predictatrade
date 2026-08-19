package sentiment

import (
	"context"
	"errors"
	"testing"
	"time"
)

// MockProvider for testing
type mockProvider struct {
	name     string
	items    []SentimentItem
	err      error
	delay    time.Duration
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) Fetch(ctx context.Context) ([]SentimentItem, error) {
	if m.delay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(m.delay):
		}
	}
	return m.items, m.err
}

func TestProviderSuccess(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	p := &mockProvider{name: "gdelt", items: []SentimentItem{
		{Source: "gdelt", Score: 50, Confidence: 0.8, Category: "BULLISH"},
	}}
	e := NewEngine(cfg, []Provider{p})
	e.refresh()
	snap := e.GetSnapshot()
	if snap.ItemCount != 1 {
		t.Fatalf("expected 1 item, got %d", snap.ItemCount)
	}
	if snap.OverallScore <= 0 {
		t.Fatalf("expected positive score, got %f", snap.OverallScore)
	}
}

func TestProviderTimeout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.TimeoutSec = 1
	p := &mockProvider{name: "slow", delay: 5 * time.Second}
	e := NewEngine(cfg, []Provider{p})
	e.refresh()
	snap := e.GetSnapshot()
	// Should have neutral fallback (no data from timed-out provider)
	if snap.OverallScore != 0 {
		t.Fatalf("timeout should result in neutral fallback, got %f", snap.OverallScore)
	}
}

func TestRateLimit(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.MaxRetries = 2
	p := &mockProvider{name: "limited", err: errors.New("rate limit exceeded")}
	e := NewEngine(cfg, []Provider{p})
	e.refresh()
	snap := e.GetSnapshot()
	if snap.OverallScore != 0 {
		t.Fatal("failed provider should result in neutral fallback")
	}
	health := snap.ProviderHealth["limited"]
	if health.Status != "ERROR" {
		t.Fatalf("provider should be ERROR, got %s", health.Status)
	}
}

func TestPartialProviderFailure(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	p1 := &mockProvider{name: "good", items: []SentimentItem{
		{Source: "good", Score: 30, Confidence: 0.7, Category: "BULLISH"},
	}}
	p2 := &mockProvider{name: "bad", err: errors.New("failed")}
	e := NewEngine(cfg, []Provider{p1, p2})
	e.refresh()
	snap := e.GetSnapshot()
	if snap.ItemCount != 1 {
		t.Fatalf("should have 1 item from good provider, got %d", snap.ItemCount)
	}
	if snap.OverallScore <= 0 {
		t.Fatal("should have positive score from good provider")
	}
}

func TestNoProviderAvailable(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	e := NewEngine(cfg, []Provider{})
	e.refresh()
	snap := e.GetSnapshot()
	if snap.OverallScore != 0 {
		t.Fatal("no providers should give neutral")
	}
}

func TestStaleSentiment(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.StaleThresholdSec = 1
	p := &mockProvider{name: "test", items: []SentimentItem{
		{Source: "test", Score: 50, Confidence: 0.9, Category: "BULLISH"},
	}}
	e := NewEngine(cfg, []Provider{p})
	e.refresh()
	// Wait for staleness
	time.Sleep(2 * time.Second)
	if !e.IsStale() {
		t.Fatal("snapshot should be stale")
	}
	influence := e.GetInfluence()
	if influence != 0 {
		t.Fatalf("stale sentiment should give 0 influence, got %f", influence)
	}
}

func TestNeutralFallback(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = false // disabled
	e := NewEngine(cfg, nil)
	influence := e.GetInfluence()
	if influence != 0 {
		t.Fatal("disabled sentiment should give 0 influence")
	}
}

func TestSourceWeighting(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.ProviderWeights = map[string]float64{
		"reuters": 0.5,
		"reddit":  0.1,
	}
	p1 := &mockProvider{name: "reuters", items: []SentimentItem{
		{Source: "reuters", Score: 80, Confidence: 0.9, Category: "BULLISH"},
	}}
	p2 := &mockProvider{name: "reddit", items: []SentimentItem{
		{Source: "reddit", Score: -40, Confidence: 0.5, Category: "BEARISH"},
	}}
	e := NewEngine(cfg, []Provider{p1, p2})
	e.refresh()
	snap := e.GetSnapshot()
	// Reuters has higher weight and positive score — overall should be positive
	if snap.OverallScore <= 0 {
		t.Fatalf("reuters (higher weight, positive) should dominate, got %f", snap.OverallScore)
	}
}

func TestConfidenceThreshold(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.MinConfidenceThreshold = 0.8
	p := &mockProvider{name: "test", items: []SentimentItem{
		{Source: "test", Score: 50, Confidence: 0.3, Category: "BULLISH"},
	}}
	e := NewEngine(cfg, []Provider{p})
	e.refresh()
	influence := e.GetInfluence()
	if influence != 0 {
		t.Fatal("low confidence should give 0 influence")
	}
}

func TestNoBlockingExternalRequestInHotPath(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	// Create engine with slow provider but don't start refresh
	p := &mockProvider{name: "slow", delay: 5 * time.Second}
	e := NewEngine(cfg, []Provider{p})
	// GetSnapshot should return immediately (not block)
	start := time.Now()
	_ = e.GetSnapshot()
	elapsed := time.Since(start)
	if elapsed > 100*time.Millisecond {
		t.Fatalf("GetSnapshot should not block, took %v", elapsed)
	}
}
