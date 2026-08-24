package gates

import (
	"testing"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

func TestCalculatePositionSize(t *testing.T) {
	equity := decimal.NewFromFloat(10000.0)   // $10,000
	stopDistance := decimal.NewFromFloat(2.0) // $2 stop distance
	symbol := DefaultXAUSymbolInfo()

	lots := CalculatePositionSize(equity, stopDistance, symbol)
	// risk_amount = 10000 * 0.01 = 100
	// point_value = 1 / 0.01 = 100
	// lots = 100 / (2 * 100) = 0.50
	expected := decimal.NewFromFloat(0.50)
	if !lots.Equal(expected) {
		t.Errorf("Position size = %s, want %s", lots.String(), expected.String())
	}
}

func TestCalculatePositionSizeZeroStop(t *testing.T) {
	lots := CalculatePositionSize(decimal.NewFromFloat(10000), decimal.Zero, DefaultXAUSymbolInfo())
	if !lots.IsZero() {
		t.Errorf("Position size with zero stop = %s, want 0", lots.String())
	}
}

func TestCalculatePositionSizeRoundsDown(t *testing.T) {
	equity := decimal.NewFromFloat(10000.0)
	stopDistance := decimal.NewFromFloat(1.5) // Non-round result
	symbol := DefaultXAUSymbolInfo()
	symbol.LotStep = decimal.NewFromFloat(0.1)

	lots := CalculatePositionSize(equity, stopDistance, symbol)
	// risk_amount = 100, point_value = 100, lots = 100 / (1.5 * 100) = 0.666...
	// Rounded down to 0.1 step → 0.6
	expected := decimal.NewFromFloat(0.6)
	if !lots.Equal(expected) {
		t.Errorf("Rounded position size = %s, want %s", lots.String(), expected.String())
	}
}

func TestCheckDailyLossLimit(t *testing.T) {
	// 5% loss should trigger halt
	if !CheckDailyLoss(-5.0, 5.0) {
		t.Error("Daily loss of -5% should trigger halt")
	}
	// 4% loss should NOT trigger halt
	if CheckDailyLoss(-4.0, 5.0) {
		t.Error("Daily loss of -4% should NOT trigger halt")
	}
}

func TestCheckTotalOpenRisk(t *testing.T) {
	equity := decimal.NewFromFloat(10000.0)
	// 6% open risk should exceed 5% limit
	totalRisk := decimal.NewFromFloat(600.0)
	if !CheckTotalOpenRisk(totalRisk, equity, 5.0) {
		t.Error("Total open risk of 6% should exceed 5% limit")
	}
	// 4% open risk should NOT exceed 5% limit
	totalRisk = decimal.NewFromFloat(400.0)
	if CheckTotalOpenRisk(totalRisk, equity, 5.0) {
		t.Error("Total open risk of 4% should NOT exceed 5% limit")
	}
}

func TestBuildPartialCloseSchedule(t *testing.T) {
	entry := decimal.NewFromFloat(2400.0)
	tp1 := decimal.NewFromFloat(2404.0)
	tp2 := decimal.NewFromFloat(2408.0)
	atr := decimal.NewFromFloat(2.0)

	schedule := BuildPartialCloseSchedule(entry, tp1, tp2, atr)
	if len(schedule) != 3 {
		t.Fatalf("Schedule length = %d, want 3", len(schedule))
	}

	// TP1: close 50%, SL → breakeven (entry)
	if schedule[0].ClosePercent != 50.0 {
		t.Errorf("TP1 close%% = %f, want 50", schedule[0].ClosePercent)
	}
	if !schedule[0].NewStopLoss.Equal(entry) {
		t.Errorf("TP1 SL = %s, want %s (breakeven)", schedule[0].NewStopLoss.String(), entry.String())
	}

	// TP2: close 30%, SL → TP1
	if schedule[1].ClosePercent != 30.0 {
		t.Errorf("TP2 close%% = %f, want 30", schedule[1].ClosePercent)
	}
	if !schedule[1].NewStopLoss.Equal(tp1) {
		t.Errorf("TP2 SL = %s, want %s (TP1)", schedule[1].NewStopLoss.String(), tp1.String())
	}

	// TP3: close 20%, trail 1.5*ATR
	if schedule[2].ClosePercent != 20.0 {
		t.Errorf("TP3 close%% = %f, want 20", schedule[2].ClosePercent)
	}
	if schedule[2].TrailATRMultiplier != 1.5 {
		t.Errorf("TP3 trail ATR mult = %f, want 1.5", schedule[2].TrailATRMultiplier)
	}
}

func TestCheckSwapProtectionNegativeSwap(t *testing.T) {
	entry := decimal.NewFromFloat(2400.0)
	sl := decimal.NewFromFloat(2396.0)     // 4 units stop
	tp := decimal.NewFromFloat(2408.0)     // 8 units target
	swapRate := decimal.NewFromFloat(-0.5) // negative swap
	lots := decimal.NewFromFloat(1.0)
	nights := 3
	isIntraday := false

	result := CheckSwapProtection(types.DirectionBuy, entry, sl, tp, swapRate, lots, nights, isIntraday)
	// expected_swap_cost = -0.5 * 1.0 * 3 = -1.5
	// effective_net_profit = 8 - 1.5 = 6.5
	// net_rr = 6.5 / 4 = 1.625 < 2.0 → reject
	if result.Allowed {
		t.Error("Trade with net R:R < 2.0 should be rejected due to swap")
	}
	if result.ReasonCode != "SWAP_ADJUSTED_RR_BELOW_MINIMUM" {
		t.Errorf("Reason = %s, want SWAP_ADJUSTED_RR_BELOW_MINIMUM", result.ReasonCode)
	}
}

func TestCheckSwapProtectionIntraday(t *testing.T) {
	entry := decimal.NewFromFloat(2400.0)
	sl := decimal.NewFromFloat(2396.0)
	tp := decimal.NewFromFloat(2408.0)
	swapRate := decimal.NewFromFloat(-0.5)
	lots := decimal.NewFromFloat(1.0)

	// Intraday → no swap exposure, should be allowed
	result := CheckSwapProtection(types.DirectionBuy, entry, sl, tp, swapRate, lots, 0, true)
	if !result.Allowed {
		t.Error("Intraday trade should be allowed (close before rollover)")
	}
}

func TestCheckSwapProtectionPositiveSwap(t *testing.T) {
	entry := decimal.NewFromFloat(2400.0)
	sl := decimal.NewFromFloat(2396.0)
	tp := decimal.NewFromFloat(2408.0)
	swapRate := decimal.NewFromFloat(0.5) // positive swap
	lots := decimal.NewFromFloat(1.0)

	result := CheckSwapProtection(types.DirectionBuy, entry, sl, tp, swapRate, lots, 5, false)
	if !result.Allowed {
		t.Error("Trade with positive swap should be allowed")
	}
}

func TestCheckSpreadSlippage(t *testing.T) {
	// Spread within limit
	result := CheckSpreadSlippage(20, 25)
	if !result.Allowed {
		t.Error("Spread of 20 with max 25 should be allowed")
	}

	// Spread exceeds limit
	result = CheckSpreadSlippage(30, 25)
	if result.Allowed {
		t.Error("Spread of 30 with max 25 should be rejected")
	}
	if result.ReasonCode != "SPREAD_EXCEEDS_MAXIMUM" {
		t.Errorf("Reason = %s, want SPREAD_EXCEEDS_MAXIMUM", result.ReasonCode)
	}
}

func TestDefaultCapitalProtectionConfig(t *testing.T) {
	cfg := DefaultCapitalProtectionConfig()
	if cfg.MaxDailyLossPct != 6.0 {
		t.Errorf("MaxDailyLossPct = %f, want 6.0", cfg.MaxDailyLossPct)
	}
	if cfg.MaxPerTradeRiskPct != 1.0 {
		t.Errorf("MaxPerTradeRiskPct = %f, want 1.0", cfg.MaxPerTradeRiskPct)
	}
	if cfg.MaxTotalOpenRiskPct != 5.0 {
		t.Errorf("MaxTotalOpenRiskPct = %f, want 6.0", cfg.MaxTotalOpenRiskPct)
	}
	if cfg.MinRR != 2.0 {
		t.Errorf("MinRR = %f, want 2.0", cfg.MinRR)
	}
}
