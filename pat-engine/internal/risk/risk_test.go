package risk

import (
	"testing"
	"time"

	"pat-engine/internal/broker"
)

func TestPositionSize(t *testing.T) {
	r := RiskProfile{Equity: 10000, RiskPerTradePct: 1, FreeMargin: 10000}
	e := broker.DefaultXAUUSDExecution()
	// 1.0 price stop = $100 risk = 10000 * 1% ; lots = 100 / (1.0 * 100) = 1.0
	lots := r.PositionSize(1.0, e)
	if lots < 0.99 || lots > 1.01 {
		t.Fatalf("PositionSize = %v, want ~1.0", lots)
	}
}

func TestPositionSizeZeroStop(t *testing.T) {
	r := RiskProfile{Equity: 10000, RiskPerTradePct: 1, FreeMargin: 10000}
	if v := r.PositionSize(0, broker.DefaultXAUUSDExecution()); v != 0 {
		t.Fatalf("PositionSize(0) = %v, want 0", v)
	}
}

func TestDailyLoss(t *testing.T) {
	dl := &DailyLoss{}
	dl.Update(-200.0, time.Now()) // -2%
	if dl.Breached(5.0, 10000.0) {
		t.Fatalf("2%% loss should NOT breach 5%% limit")
	}
	dl.Update(-400.0, time.Now()) // cumulative -6%
	if !dl.Breached(5.0, 10000.0) {
		t.Fatalf("6%% loss should breach 5%% limit")
	}
}
