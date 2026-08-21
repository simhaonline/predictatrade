package gates

import (
	"testing"

	"github.com/shopspring/decimal"
)

// ─── I1: SL never moves backward ───

func TestMonotonicSL_BUY_RejectBackward(t *testing.T) {
	p := SLProposal{
		Direction: "BUY", EntryPrice: d("3350"), InitialSL: d("3345"),
		ConfirmedSL: d("3354"), ProposedSL: d("3353"),
		CurrentBid: d("3360"), CurrentAsk: d("3361"),
	}
	ok, reason := ValidateMonotonicSL(p)
	if ok {
		t.Errorf("Should reject backward SL for BUY: proposed 3353 < confirmed 3354")
	}
	if reason == "" {
		t.Error("Should provide rejection reason")
	}
}

func TestMonotonicSL_BUY_AcceptForward(t *testing.T) {
	p := SLProposal{
		Direction: "BUY", EntryPrice: d("3350"), InitialSL: d("3345"),
		ConfirmedSL: d("3354"), ProposedSL: d("3355"),
		CurrentBid: d("3360"), CurrentAsk: d("3361"),
	}
	ok, _ := ValidateMonotonicSL(p)
	if !ok {
		t.Error("Should accept forward SL for BUY: 3355 > 3354")
	}
}

func TestMonotonicSL_SELL_RejectBackward(t *testing.T) {
	p := SLProposal{
		Direction: "SELL", EntryPrice: d("3350"), InitialSL: d("3355"),
		ConfirmedSL: d("3346"), ProposedSL: d("3347"),
		CurrentBid: d("3343"), CurrentAsk: d("3344"),
	}
	ok, _ := ValidateMonotonicSL(p)
	if ok {
		t.Error("Should reject backward SL for SELL: proposed 3347 > confirmed 3346")
	}
}

func TestMonotonicSL_SELL_AcceptForward(t *testing.T) {
	p := SLProposal{
		Direction: "SELL", EntryPrice: d("3350"), InitialSL: d("3355"),
		ConfirmedSL: d("3346"), ProposedSL: d("3345"),
		CurrentBid: d("3343"), CurrentAsk: d("3344"),
	}
	ok, _ := ValidateMonotonicSL(p)
	if !ok {
		t.Error("Should accept forward SL for SELL: 3345 < 3346")
	}
}

func TestMonotonicSL_BUY_RejectAboveBid(t *testing.T) {
	p := SLProposal{
		Direction: "BUY", EntryPrice: d("3350"), InitialSL: d("3345"),
		ConfirmedSL: d("3354"), ProposedSL: d("3361"),
		CurrentBid: d("3360"), CurrentAsk: d("3361"),
	}
	ok, _ := ValidateMonotonicSL(p)
	if ok {
		t.Error("Should reject SL at or above bid for BUY")
	}
}

// ─── I7: Broker stop level validation ───

func TestBrokerStopLevel_BUY(t *testing.T) {
	p := SLProposal{
		Direction: "BUY", ConfirmedSL: d("3354"), ProposedSL: d("3359"),
		CurrentBid: d("3360"), CurrentAsk: d("3361"),
		BrokerStopsLevel: d("3"), // 3 points min stop distance
	}
	ok, _ := ValidateBrokerStopLevel(p)
	if ok {
		t.Error("Should reject: proposed SL 3359 is within 3 of bid 3360 (minSL=3357)")
	}
}

func TestBrokerStopLevel_BUY_Accept(t *testing.T) {
	p := SLProposal{
		Direction: "BUY", ConfirmedSL: d("3354"), ProposedSL: d("3355"),
		CurrentBid: d("3360"), CurrentAsk: d("3361"),
		BrokerStopsLevel: d("3"),
	}
	ok, _ := ValidateBrokerStopLevel(p)
	if !ok {
		t.Error("Should accept: proposed SL 3355 < minSL 3357")
	}
}

// ─── Minimum improvement hysteresis ───

func TestMinimumImprovement_RejectSmallChange(t *testing.T) {
	p := SLProposal{
		ConfirmedSL: d("3354.00"), ProposedSL: d("3354.01"),
	}
	ok, _ := ValidateMinimumImprovement(p, d("0.05"))
	if ok {
		t.Error("Should reject: 0.01 < 0.05 minimum improvement")
	}
}

func TestMinimumImprovement_AcceptLargeChange(t *testing.T) {
	p := SLProposal{
		ConfirmedSL: d("3354.00"), ProposedSL: d("3356.00"),
	}
	ok, _ := ValidateMinimumImprovement(p, d("0.05"))
	if !ok {
		t.Error("Should accept: 2.00 >= 0.05 minimum improvement")
	}
}

// ─── I8: Immutable initial R ───

func TestCalculateInitialR_BUY(t *testing.T) {
	r := CalculateInitialR(d("3350"), d("3345"))
	if !r.Equal(d("5")) {
		t.Errorf("Initial R = %s, want 5", r.String())
	}
}

func TestCalculateInitialR_SELL(t *testing.T) {
	r := CalculateInitialR(d("3350"), d("3355"))
	if !r.Equal(d("5")) {
		t.Errorf("Initial R = %s, want 5", r.String())
	}
}

// ─── Unrealized R calculation ───

func TestUnrealizedR_BUY(t *testing.T) {
	// Entry 3350, SL 3345, R=5, current bid 3355 → R = (3355-3350)/5 = 1.0
	r := CalculateUnrealizedR("BUY", d("3350"), d("5"), d("3355"), d("3356"))
	if !r.Equal(d("1")) {
		t.Errorf("Unrealized R = %s, want 1", r.String())
	}
}

func TestUnrealizedR_SELL(t *testing.T) {
	// Entry 3350, SL 3355, R=5, current ask 3345 → R = (3350-3345)/5 = 1.0
	r := CalculateUnrealizedR("SELL", d("3350"), d("5"), d("3344"), d("3345"))
	if !r.Equal(d("1")) {
		t.Errorf("Unrealized R = %s, want 1", r.String())
	}
}

func TestUnrealizedR_ZeroRiskDistance(t *testing.T) {
	r := CalculateUnrealizedR("BUY", d("3350"), d("0"), d("3360"), d("3361"))
	if !r.IsZero() {
		t.Error("Should return 0 for zero risk distance")
	}
}

func TestUnrealizedR_NegativeRiskDistance(t *testing.T) {
	r := CalculateUnrealizedR("BUY", d("3350"), d("-1"), d("3360"), d("3361"))
	if !r.IsZero() {
		t.Error("Should return 0 for negative risk distance")
	}
}

func TestUnrealizedR_InvalidDirection(t *testing.T) {
	r := CalculateUnrealizedR("INVALID", d("3350"), d("5"), d("3355"), d("3356"))
	if !r.IsZero() {
		t.Error("Should return 0 for invalid direction")
	}
}

// ─── Management stage determination ───

func TestManagementStage_OpenInitialRisk(t *testing.T) {
	stage := DetermineManagementStage(d("-1"), cfg("STANDARD_SCALPING"))
	if stage != StageOpenInitialRisk {
		t.Errorf("Stage = %s, want OPEN_INITIAL_RISK", stage)
	}
}

func TestManagementStage_ProfitDeveloping(t *testing.T) {
	stage := DetermineManagementStage(d("0.3"), cfg("STANDARD_SCALPING"))
	if stage != StageProfitDeveloping {
		t.Errorf("Stage = %s, want PROFIT_DEVELOPING", stage)
	}
}

func TestManagementStage_BreakEvenProtected(t *testing.T) {
	stage := DetermineManagementStage(d("1.0"), cfg("STANDARD_SCALPING"))
	if stage != StageBreakEvenProtected {
		t.Errorf("Stage = %s, want BREAK_EVEN_PROTECTED", stage)
	}
}

func TestManagementStage_ProfitLocked(t *testing.T) {
	stage := DetermineManagementStage(d("1.5"), cfg("STANDARD_SCALPING"))
	if stage != StageProfitLocked {
		t.Errorf("Stage = %s, want PROFIT_LOCKED", stage)
	}
}

func TestManagementStage_TrailingActive(t *testing.T) {
	stage := DetermineManagementStage(d("2.5"), cfg("STANDARD_SCALPING"))
	if stage != StageTrailingActive {
		t.Errorf("Stage = %s, want TRAILING_ACTIVE", stage)
	}
}

// ─── Full validation ───

func TestValidateSLProposal_AllPass(t *testing.T) {
	p := SLProposal{
		Direction: "BUY", EntryPrice: d("3350"), InitialSL: d("3345"),
		ConfirmedSL: d("3354"), ProposedSL: d("3356"),
		CurrentBid: d("3360"), CurrentAsk: d("3361"),
		BrokerStopsLevel: d("3"), TickSize: d("0.01"),
	}
	ok, reasons := ValidateSLProposal(p, d("0.05"))
	if !ok {
		t.Errorf("Should pass all validations, got reasons: %v", reasons)
	}
}

func TestValidateSLProposal_MonotonicFail(t *testing.T) {
	p := SLProposal{
		Direction: "BUY", ConfirmedSL: d("3354"), ProposedSL: d("3353"),
		CurrentBid: d("3360"), CurrentAsk: d("3361"),
	}
	ok, reasons := ValidateSLProposal(p, d("0.05"))
	if ok {
		t.Error("Should fail monotonic check")
	}
	if len(reasons) == 0 {
		t.Error("Should provide rejection reasons")
	}
}

// ─── Strategy-specific profiles ───

func TestStrategyProfiles_Distinct(t *testing.T) {
	configs := DefaultTradeManagementConfigs()
	if configs["ULTRA_SCALPING"].TrailingActivationR == configs["TREND_SWING"].TrailingActivationR {
		t.Error("Ultra Scalping and Trend Swing should have different break-even triggers")
	}
	if configs["ULTRA_SCALPING"].TrailingActivationR >= configs["STANDARD_SWING"].TrailingActivationR {
		t.Error("Ultra Scalping should activate trailing earlier than Standard Swing")
	}
}

func TestStrategyProfiles_AllPresent(t *testing.T) {
	configs := DefaultTradeManagementConfigs()
	for _, id := range []string{"STANDARD_SCALPING", "ULTRA_SCALPING", "STANDARD_SWING", "TREND_SWING"} {
		if _, ok := configs[id]; !ok {
			t.Errorf("Missing strategy profile: %s", id)
		}
	}
}

// ─── Normalize SL price ───

func TestNormalizeSLPrice(t *testing.T) {
	// Tick size 0.01, SL 3354.567 → should round to 3354.57
	normalized := NormalizeSLPrice(d("3354.567"), d("0.01"), 2)
	expected := d("3354.57")
	if !normalized.Equal(expected) {
		t.Errorf("Normalized = %s, want %s", normalized.String(), expected.String())
	}
}

func TestNormalizeSLPrice_ZeroTickSize(t *testing.T) {
	normalized := NormalizeSLPrice(d("3354.567"), d("0"), 2)
	if !normalized.Equal(d("3354.567")) {
		t.Error("Should return unchanged for zero tick size")
	}
}

// ─── Helpers ───

func d(s string) decimal.Decimal {
	v, _ := decimal.NewFromString(s)
	return v
}

func cfg(strategyID string) TradeManagementConfig {
	configs := DefaultTradeManagementConfigs()
	return configs[strategyID]
}
