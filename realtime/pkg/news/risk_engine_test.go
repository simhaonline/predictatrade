package news

import (
	"context"
	"testing"
	"time"
)

// mockProvider implements EconomicCalendarProvider for testing.
type mockProvider struct {
	events  []NewsEvent
	healthy bool
	name    string
}

func (m *mockProvider) FetchEvents(ctx context.Context, from, to time.Time) ([]NewsEvent, error) {
	return m.events, nil
}
func (m *mockProvider) Health() ProviderHealth {
	return ProviderHealth{ProviderName: m.name, Healthy: m.healthy, LastSuccessfulSync: time.Now(), StaleAfter: 15 * time.Minute}
}
func (m *mockProvider) ProviderName() string             { return m.name }
func (m *mockProvider) SupportsHistorical() bool          { return true }
func (m *mockProvider) SupportsRealtime() bool            { return false }

func TestRiskEngine_NoProvider_ReturnsDataUnavailable(t *testing.T) {
	cfg := DefaultConfig()
	engine := NewRiskEngine(cfg, nil)
	result := engine.ComputeRisk(time.Now().UTC())
	if result.Level != NewsRiskDataUnavailable {
		t.Fatalf("expected DATA_UNAVAILABLE, got %s", result.Level)
	}
	if result.ReasonCode != "NEWS_PROVIDER_NOT_CONFIGURED" {
		t.Fatalf("expected NEWS_PROVIDER_NOT_CONFIGURED, got %s", result.ReasonCode)
	}
}

func TestRiskEngine_ProviderStale_FailSafe(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "test"
	cfg.FailPolicy = FailPolicyBlockTrading
	p := &mockProvider{healthy: false, name: "test"}
	engine := NewRiskEngine(cfg, p)
	result := engine.ComputeRisk(time.Now().UTC())
	if result.Level != NewsRiskDataUnavailable {
		t.Fatalf("expected DATA_UNAVAILABLE for stale provider, got %s", result.Level)
	}
}

func TestRiskEngine_ProviderStale_FailOpen(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "test"
	cfg.FailPolicy = FailPolicyAllowTrading
	p := &mockProvider{healthy: false, name: "test"}
	engine := NewRiskEngine(cfg, p)
	result := engine.ComputeRisk(time.Now().UTC())
	if result.Level != NewsRiskNone {
		t.Fatalf("expected NONE for fail-open, got %s", result.Level)
	}
}

func TestRiskEngine_PreEventBlackout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "test"
	cfg.PreBlackoutMinutes = 15
	cfg.PostBlackoutMinutes = 15
	cfg.MinImpact = ImpactHigh

	now := time.Now().UTC()
	events := []NewsEvent{
		{
			EventID:        "test_1",
			EventName:      "FOMC Rate Decision",
			Currency:       "USD",
			Country:        "US",
			Impact:         ImpactHigh,
			ScheduledAtUTC: now.Add(10 * time.Minute), // 10 min from now — within 15 min pre-blackout
		},
	}

	p := &mockProvider{events: events, healthy: true, name: "test"}
	engine := NewRiskEngine(cfg, p)
	engine.sync(context.Background())
	result := engine.ComputeRisk(now)
	if result.Level != NewsRiskHigh {
		t.Fatalf("expected HIGH for pre-event blackout, got %s (reason: %s, evidence: %s)", result.Level, result.ReasonCode, result.Evidence)
	}
	if result.ReasonCode != "NEWS_PRE_EVENT_BLACKOUT" {
		t.Fatalf("expected NEWS_PRE_EVENT_BLACKOUT, got %s", result.ReasonCode)
	}
}

func TestRiskEngine_PostEventBlackout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "test"
	cfg.PreBlackoutMinutes = 15
	cfg.PostBlackoutMinutes = 15
	cfg.MinImpact = ImpactHigh

	now := time.Now().UTC()
	events := []NewsEvent{
		{
			EventID:        "test_2",
			EventName:      "NFP",
			Currency:       "USD",
			Country:        "US",
			Impact:         ImpactHigh,
			ScheduledAtUTC: now.Add(-5 * time.Minute), // 5 min ago — within 15 min post-blackout
		},
	}

	p := &mockProvider{events: events, healthy: true, name: "test"}
	engine := NewRiskEngine(cfg, p)
	engine.sync(context.Background())
	result := engine.ComputeRisk(now)
	if result.Level != NewsRiskHigh {
		t.Fatalf("expected HIGH for post-event blackout, got %s", result.Level)
	}
	if result.ReasonCode != "NEWS_POST_EVENT_BLACKOUT" {
		t.Fatalf("expected NEWS_POST_EVENT_BLACKOUT, got %s", result.ReasonCode)
	}
}

func TestRiskEngine_NoEvents_ReturnsNone(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "test"
	p := &mockProvider{events: nil, healthy: true, name: "test"}
	engine := NewRiskEngine(cfg, p)
	engine.sync(context.Background())
	result := engine.ComputeRisk(time.Now().UTC())
	if result.Level != NewsRiskNone {
		t.Fatalf("expected NONE with no events, got %s", result.Level)
	}
}

func TestRiskEngine_NonUSDEvent_Ignored(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "test"
	cfg.MinImpact = ImpactLow
	now := time.Now().UTC()
	events := []NewsEvent{
		{
			EventID:        "test_eu",
			EventName:      "ECB Rate Decision",
			Currency:       "EUR",
			Country:        "DE",
			Impact:         ImpactHigh,
			ScheduledAtUTC: now.Add(5 * time.Minute),
		},
	}
	p := &mockProvider{events: events, healthy: true, name: "test"}
	engine := NewRiskEngine(cfg, p)
	engine.sync(context.Background())
	result := engine.ComputeRisk(now)
	if result.Level != NewsRiskNone {
		t.Fatalf("expected NONE for non-USD event, got %s", result.Level)
	}
}

func TestRiskEngine_LowImpactOutsideBlackout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "test"
	cfg.PreBlackoutMinutes = 15
	cfg.MinImpact = ImpactMedium
	now := time.Now().UTC()
	events := []NewsEvent{
		{
			EventID:        "test_low",
			EventName:      "Minor Data",
			Currency:       "USD",
			Country:        "US",
			Impact:         ImpactLow,
			ScheduledAtUTC: now.Add(2 * time.Hour), // far away
		},
	}
	p := &mockProvider{events: events, healthy: true, name: "test"}
	engine := NewRiskEngine(cfg, p)
	engine.sync(context.Background())
	result := engine.ComputeRisk(now)
	if result.Level != NewsRiskNone {
		t.Fatalf("expected NONE for low impact far event, got %s", result.Level)
	}
}

func TestRiskEngine_ShouldBlock(t *testing.T) {
	tests := []struct {
		level    NewsRiskLevel
		expected bool
	}{
		{NewsRiskNone, false},
		{NewsRiskLow, false},
		{NewsRiskMedium, false},
		{NewsRiskHigh, true},
		{NewsRiskExtreme, true},
		{NewsRiskDataUnavailable, true},
	}
	for _, tt := range tests {
		if tt.level.ShouldBlock() != tt.expected {
			t.Errorf("ShouldBlock(%s) = %v, want %v", tt.level, tt.level.ShouldBlock(), tt.expected)
		}
	}
}

func TestFMPProvider_EmptyKey_Fails(t *testing.T) {
	p := NewFMPProvider("")
	events, err := p.FetchEvents(context.Background(), time.Now(), time.Now().Add(time.Hour))
	if err == nil {
		t.Fatal("expected error with empty API key")
	}
	if events != nil {
		t.Fatal("expected nil events with empty API key")
	}
}

func TestFMPProvider_NormalizeImpact(t *testing.T) {
	p := NewFMPProvider("test")
	tests := []struct {
		input    string
		expected ImpactLevel
	}{
		{"high", ImpactHigh},
		{"High", ImpactHigh},
		{"HIGH", ImpactHigh},
		{"medium", ImpactMedium},
		{"low", ImpactLow},
		{"", ImpactNone},
		{"unknown", ImpactNone},
	}
	for _, tt := range tests {
		result := p.normalizeImpact(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeImpact(%q) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestFMPProvider_CategorizeEvent(t *testing.T) {
	p := NewFMPProvider("test")
	tests := []struct {
		name     string
		expected string
	}{
		{"FOMC Rate Decision", "RATE_DECISION"},
		{"Nonfarm Payrolls", "EMPLOYMENT"},
		{"CPI m/m", "INFLATION"},
		{"Core PCE Price Index", "INFLATION"},
		{"GDP Growth Rate", "GDP"},
		{"Unemployment Rate", "UNEMPLOYMENT"},
		{"Jerome Powell Speech", "SPEECH"},
		{"Retail Sales", "OTHER"},
	}
	for _, tt := range tests {
		result := p.categorizeEvent(tt.name)
		if result != tt.expected {
			t.Errorf("categorizeEvent(%q) = %s, want %s", tt.name, result, tt.expected)
		}
	}
}
