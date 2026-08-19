package hedging

import (
	"testing"

	"github.com/shopspring/decimal"
)

func req() HedgeRequest {
	return HedgeRequest{
		AccountID:           "acc1",
		StrategyID:          "STANDARD_SCALPING",
		Symbol:               "XAUUSD",
		OriginalTradeID:      "trade1",
		OriginalDirection:   "BUY",
		OriginalSize:         decimal.NewFromFloat(1.0),
		OriginalEntry:       decimal.NewFromFloat(2400),
		CurrentPrice:         decimal.NewFromFloat(2390),
		UnrealizedLossPct:    1.0,
		BrokerSupportsHedge:  true,
		AccountIsNetting:     false,
		LicensePermitsTrade:  true,
		ManipulationIndex:    20,
		Volatility:           0.002,
		Spread:               0.3,
		MarketDataFresh:      true,
	}
}

func TestDisabledByDefault(t *testing.T) {
	m := NewManager(DefaultConfig())
	result := m.EvaluateHedge(req())
	if result.Allowed {
		t.Fatal("hedging should be disabled by default")
	}
}

func TestNoHedgeBelowThreshold(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	m := NewManager(cfg)
	r := req()
	r.UnrealizedLossPct = 0.1 // below 0.5% minimum
	result := m.EvaluateHedge(r)
	if result.Allowed {
		t.Fatal("should not hedge below threshold")
	}
}

func TestNoHedgeBeyondSafeMaximum(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	m := NewManager(cfg)
	r := req()
	r.UnrealizedLossPct = 5.0 // above 3% max
	result := m.EvaluateHedge(r)
	if result.Allowed {
		t.Fatal("should not hedge beyond safe maximum")
	}
}

func TestNoDuplicateHedge(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	m := NewManager(cfg)
	h := &HedgePosition{
		OriginalTradeID: "trade1",
		HedgeTradeID:   "trade1_HEDGE",
		Status:         "OPEN",
	}
	m.OpenHedge(h)
	r := req()
	result := m.EvaluateHedge(r)
	if result.Allowed {
		t.Fatal("should not create duplicate hedge")
	}
}

func TestBrokerCapabilityCheck(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	m := NewManager(cfg)
	r := req()
	r.BrokerSupportsHedge = false
	result := m.EvaluateHedge(r)
	if result.Allowed {
		t.Fatal("should not hedge when broker doesn't support it")
	}
}

func TestNettingAccountBehavior(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	m := NewManager(cfg)
	r := req()
	r.AccountIsNetting = true
	result := m.EvaluateHedge(r)
	if result.Allowed {
		t.Fatal("should not hedge on netting account")
	}
}

func TestPartialSizeCalculation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	m := NewManager(cfg)
	result := m.EvaluateHedge(req())
	if !result.Allowed {
		t.Fatalf("hedge should be allowed: %s", result.Reason)
	}
	origSize, _ := result.Hedge.OriginalSize.Float64()
	hedgeSize, _ := result.Hedge.HedgeSize.Float64()
	if hedgeSize > origSize {
		t.Fatalf("hedge size should not exceed original: %f > %f", hedgeSize, origSize)
	}
	if hedgeSize > origSize*cfg.HedgeSizeCap {
		t.Fatalf("hedge size exceeds cap: %f > %f", hedgeSize, origSize*cfg.HedgeSizeCap)
	}
}

func TestExposureCap(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.MaxSimultaneousHedges = 1
	m := NewManager(cfg)
	h := &HedgePosition{OriginalTradeID: "trade0", HedgeTradeID: "h0", Status: "OPEN", HedgeSize: decimal.NewFromFloat(2.0)}
	m.OpenHedge(h)
	r := req()
	r.OriginalTradeID = "trade1"
	result := m.EvaluateHedge(r)
	if result.Allowed {
		t.Fatal("should not exceed max simultaneous hedges")
	}
}

func TestGridFeatureDisabledByDefault(t *testing.T) {
	m := NewManager(DefaultConfig())
	if m.IsGridEnabled() {
		t.Fatal("grid hedging should be disabled by default")
	}
}

func TestNoMartingaleEscalation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	m := NewManager(cfg)
	result := m.EvaluateHedge(req())
	if !result.Allowed {
		t.Fatalf("hedge should be allowed: %s", result.Reason)
	}
	// Hedge size must be <= original size (no escalation)
	origSize, _ := result.Hedge.OriginalSize.Float64()
	hedgeSize, _ := result.Hedge.HedgeSize.Float64()
	if hedgeSize > origSize {
		t.Fatalf("no martingale escalation: hedge %f > original %f", hedgeSize, origSize)
	}
}

func TestAutoCloseOnExpiry(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.MaxHedgeDurationMin = 0
	m := NewManager(cfg)
	h := &HedgePosition{
		OriginalTradeID: "trade1",
		HedgeTradeID:   "h1",
		Status:         "OPEN",
		HedgeSize:      decimal.NewFromFloat(0.5),
	}
	// Set expiry in the past
	h.ExpiresAt = h.ExpiresAt.Add(-1 * 60 * 60 * 1e9) // 1 hour ago
	m.OpenHedge(h)
	expired := m.CheckExpiredHedges()
	if len(expired) != 1 {
		t.Fatalf("expected 1 expired hedge, got %d", len(expired))
	}
}
