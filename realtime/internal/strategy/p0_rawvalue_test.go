package strategy

import (
	"testing"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// P0-1: RawValue must carry the ACTUAL indicator read, not zero and not the
// signal direction. A TREND/EMA9 evidence row must expose the live EMA9 value.
func TestAddEvidenceRawValueCarriesIndicatorRead(t *testing.T) {
	var ev []types.EvidenceContribution
	ema9 := decimal.NewFromFloat(4321.25)
	addEvidenceRaw(&ev, "TREND", "EMA9_ABOVE_EMA21", types.DirectionBuy,
		15, 0.12, types.QualityAuthoritative, "", ema9)

	if len(ev) != 1 {
		t.Fatalf("expected 1 evidence row, got %d", len(ev))
	}
	if ev[0].RawValue.IsZero() {
		t.Fatalf("RawValue is zero — indicator read not populated (P0-1)")
	}
	if !ev[0].RawValue.Equal(ema9) {
		t.Fatalf("RawValue = %s, want %s (the live indicator read)", ev[0].RawValue, ema9)
	}
	if ev[0].RawValue.Equal(ev[0].NormalizedValue) {
		t.Fatalf("RawValue must be the indicator read, NOT the normalized contribution copy")
	}
	// legacy addEvidence (no raw read available) stays as-is — documented fallback
	addEvidence(&ev, "TREND", "BOS_CONFIRMED", types.DirectionBuy, 10, 0.1, types.QualityAuthoritative, "")
	if !ev[1].RawValue.IsZero() {
		t.Fatalf("boolean/structural evidence without a raw read should keep RawValue zero")
	}
}
