package breakout

import (
	"testing"
	"time"

	"github.com/predictatrade/realtime/pkg/news"
	"github.com/shopspring/decimal"
)

func TestCheckEligibility_AllPass(t *testing.T) {
	input := EligibilityInput{
		NewsMode:         news.NewsModeEventBreakout,
		ProviderHealthy:  true,
		Event:            &news.NewsEvent{EventID: "e1", Currency: "USD", Country: "US", Impact: news.ImpactHigh, ScheduledAtUTC: time.Now().Add(5 * time.Minute)},
		CurrentPrice:     decimal.NewFromFloat(2500.0),
		ATR:              decimal.NewFromFloat(5.0),
		Spread:           1.0,
		MaxSpread:        3.0,
		SessionAllowed:   true,
		DailyLossClear:   true,
		DrawdownClear:    true,
		ExposureClear:    true,
		MarginSufficient: true,
		LicenseActive:    true,
		EntitlementOK:    true,
		Equity:           decimal.NewFromFloat(10000),
		TickSize:         decimal.NewFromFloat(0.01),
		TickValue:        decimal.NewFromFloat(1.0),
	}
	result := CheckEligibility(input)
	if !result.Eligible {
		t.Fatalf("expected eligible, got rejected: %s", result.RejectReason)
	}
}

func TestCheckEligibility_ModeNotEventBreakout(t *testing.T) {
	input := EligibilityInput{
		NewsMode:        news.NewsModeProtectOnly,
		ProviderHealthy: true,
		Event:           &news.NewsEvent{EventID: "e1", Currency: "USD", Country: "US", Impact: news.ImpactHigh},
	}
	result := CheckEligibility(input)
	if result.Eligible {
		t.Fatal("expected not eligible for PROTECT_ONLY mode")
	}
	if result.RejectReason != "NEWS_MODE_NOT_EVENT_BREAKOUT" {
		t.Fatalf("expected NEWS_MODE_NOT_EVENT_BREAKOUT, got %s", result.RejectReason)
	}
}

func TestCheckEligibility_ProviderUnhealthy(t *testing.T) {
	input := EligibilityInput{
		NewsMode:        news.NewsModeEventBreakout,
		ProviderHealthy: false,
		Event:           &news.NewsEvent{EventID: "e1", Currency: "USD", Country: "US", Impact: news.ImpactHigh},
	}
	result := CheckEligibility(input)
	if result.Eligible || result.RejectReason != "NEWS_PROVIDER_UNHEALTHY" {
		t.Fatalf("expected NEWS_PROVIDER_UNHEALTHY, got eligible=%v reason=%s", result.Eligible, result.RejectReason)
	}
}

func TestCheckEligibility_DailyLossBlocked(t *testing.T) {
	input := EligibilityInput{
		NewsMode:        news.NewsModeEventBreakout,
		ProviderHealthy: true,
		Event:           &news.NewsEvent{EventID: "e1", Currency: "USD", Country: "US", Impact: news.ImpactHigh},
		SessionAllowed:  true,
		DailyLossClear:  false, // daily loss gate triggered
	}
	result := CheckEligibility(input)
	if result.Eligible || result.RejectReason != "DAILY_LOSS_GATE_BLOCKED" {
		t.Fatalf("expected DAILY_LOSS_GATE_BLOCKED, got %s", result.RejectReason)
	}
}

func TestCheckEligibility_DrawdownBlocked(t *testing.T) {
	input := EligibilityInput{
		NewsMode:        news.NewsModeEventBreakout,
		ProviderHealthy: true,
		Event:           &news.NewsEvent{EventID: "e1", Currency: "USD", Country: "US", Impact: news.ImpactHigh},
		SessionAllowed:  true,
		DailyLossClear:  true,
		DrawdownClear:   false,
	}
	result := CheckEligibility(input)
	if result.Eligible || result.RejectReason != "DRAWDOWN_GATE_BLOCKED" {
		t.Fatalf("expected DRAWDOWN_GATE_BLOCKED, got %s", result.RejectReason)
	}
}

func TestCheckEligibility_SpreadTooHigh(t *testing.T) {
	input := EligibilityInput{
		NewsMode:        news.NewsModeEventBreakout,
		ProviderHealthy: true,
		Event:           &news.NewsEvent{EventID: "e1", Currency: "USD", Country: "US", Impact: news.ImpactHigh},
		SessionAllowed:  true,
		DailyLossClear:  true,
		DrawdownClear:   true,
		ExposureClear:   true,
		MarginSufficient: true,
		LicenseActive:   true,
		EntitlementOK:   true,
		Spread:          5.0,
		MaxSpread:       3.0,
	}
	result := CheckEligibility(input)
	if result.Eligible || result.RejectReason != "SPREAD_TOO_HIGH" {
		t.Fatalf("expected SPREAD_TOO_HIGH, got %s", result.RejectReason)
	}
}

func TestCreatePlan_ValidPlan(t *testing.T) {
	input := EligibilityInput{
		NewsMode:         news.NewsModeEventBreakout,
		ProviderHealthy:  true,
		Event:            &news.NewsEvent{EventID: "e1", Currency: "USD", Country: "US", Impact: news.ImpactHigh, ScheduledAtUTC: time.Now().Add(5 * time.Minute)},
		CurrentPrice:     decimal.NewFromFloat(2500.0),
		ATR:              decimal.NewFromFloat(5.0),
		Spread:           1.0,
		MaxSpread:        3.0,
		SessionAllowed:   true,
		DailyLossClear:   true,
		DrawdownClear:    true,
		ExposureClear:    true,
		MarginSufficient: true,
		LicenseActive:    true,
		EntitlementOK:    true,
		Equity:           decimal.NewFromFloat(10000),
		TickSize:         decimal.NewFromFloat(0.01),
		TickValue:        decimal.NewFromFloat(1.0),
	}
	cfg := DefaultConfig()
	plan, err := CreatePlan(input, cfg, "plan_1", "oco_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.PlanID != "plan_1" {
		t.Fatalf("expected plan_1, got %s", plan.PlanID)
	}
	if plan.OcoGroupID != "oco_1" {
		t.Fatalf("expected oco_1, got %s", plan.OcoGroupID)
	}
	// Buy stop should be above current price
	if !plan.BuyStopEntry.GreaterThan(input.CurrentPrice) {
		t.Fatal("buy stop entry should be above current price")
	}
	// Sell stop should be below current price
	if !plan.SellStopEntry.LessThan(input.CurrentPrice) {
		t.Fatal("sell stop entry should be below current price")
	}
	// Buy SL should be below buy entry
	if !plan.BuyStopSL.LessThan(plan.BuyStopEntry) {
		t.Fatal("buy stop SL should be below buy stop entry")
	}
	// Sell SL should be above sell entry
	if !plan.SellStopSL.GreaterThan(plan.SellStopEntry) {
		t.Fatal("sell stop SL should be above sell stop entry")
	}
	// Volume should be positive
	if !plan.Volume.GreaterThan(decimal.Zero) {
		t.Fatal("volume should be positive")
	}
	// Status should be CREATED
	if plan.Status != StatusCreated {
		t.Fatalf("expected CREATED, got %s", plan.Status)
	}
}

func TestCreatePlan_Ineligible_Fails(t *testing.T) {
	input := EligibilityInput{
		NewsMode: news.NewsModeProtectOnly, // wrong mode
	}
	cfg := DefaultConfig()
	_, err := CreatePlan(input, cfg, "plan_1", "oco_1")
	if err == nil {
		t.Fatal("expected error for ineligible plan")
	}
}

func TestEngine_RegisterAndGetPlan(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	plan := &BreakoutPlan{PlanID: "test_plan", Status: StatusArmed}
	engine.RegisterPlan(plan)
	retrieved, ok := engine.GetPlan("test_plan")
	if !ok {
		t.Fatal("plan not found")
	}
	if retrieved.PlanID != "test_plan" {
		t.Fatal("wrong plan retrieved")
	}
}

func TestEngine_ExpirePlans(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	past := time.Now().Add(-1 * time.Minute)
	plan := &BreakoutPlan{
		PlanID: "expired_plan",
		Status: StatusArmed,
		Expiry: past,
	}
	engine.RegisterPlan(plan)
	expired := engine.ExpirePlans(time.Now())
	if len(expired) != 1 || expired[0] != "expired_plan" {
		t.Fatalf("expected 1 expired plan, got %v", expired)
	}
	retrieved, _ := engine.GetPlan("expired_plan")
	if retrieved.Status != StatusExpired {
		t.Fatalf("expected EXPIRED, got %s", retrieved.Status)
	}
}

func TestDefaultConfig_DisabledByDefault(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Enabled {
		t.Fatal("breakout must be DISABLED by default")
	}
}
