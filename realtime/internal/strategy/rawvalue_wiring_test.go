// rawvalue_wiring_test.go — Phase 0.75 Task A (prompt.md): RawValue coverage.
//
// Contract under test:
//   1. Every addEvidenceRaw call site carries the REAL indicator read — never
//      a direction, score, or placeholder. Verified per strategy by asserting
//      RawValue equals the live state field for the evidence row that carries it.
//   2. Warmup semantics: absence means "not ready" — an indicator that is not
//      ready emits NO evidence row at all (never a fabricated zero RawValue).
//   3. Legacy addEvidence sites remain boolean/structural facts with zero
//      RawValue (documented exception list in docs/strategy/RAWVALUE_COVERAGE.md).
package strategy

import (
	"testing"
	"time"

	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// requireRawOn asserts: if an evidence row exists for (pillar, feature), its
// RawValue must be non-zero and equal the expected indicator read.
func requireRawOn(t *testing.T, evs []types.EvidenceContribution, pillar, feature string, want decimal.Decimal) {
	t.Helper()
	for _, ev := range evs {
		if ev.Pillar == pillar && ev.Feature == feature {
			if ev.RawValue.IsZero() {
				t.Fatalf("%s/%s has zero RawValue — RawValue must carry the real indicator read", pillar, feature)
			}
			if !ev.RawValue.Equal(want) {
				t.Fatalf("%s/%s RawValue %s != expected indicator read %s", pillar, feature, ev.RawValue, want)
			}
			return
		}
	}
	// No row: allowed only when the indicator is not ready (warmup) — caller
	// asserts readiness separately. Here we fail if the caller expected a row.
	t.Fatalf("%s/%s evidence row missing while indicator ready", pillar, feature)
}

// TestRawValueStandardScalping — every wired read in StandardScalping carries
// the live indicator value on the matching evidence row.
func TestRawValueStandardScalping(t *testing.T) {
	price := 2381.0
	state := makeState(price, features.IndicatorFeatures{
		ATR:           decimal.NewFromFloat(3.0),
		EMA9:          decimal.NewFromFloat(2380.11),
		EMA21:         decimal.NewFromFloat(2378.20), // EMA9 above → TREND buy
		MACDMain:      decimal.NewFromFloat(1.5),
		MACDSignal:    decimal.NewFromFloat(0.9),
		MACDHistogram: decimal.NewFromFloat(0.6),
		OsMA:          decimal.NewFromFloat(0.42),
		RSI:           decimal.NewFromFloat(58.4),
		ADX:           decimal.NewFromFloat(31.2),
		ADXPlusDI:     decimal.NewFromFloat(28.0),
		ADXMinusDI:    decimal.NewFromFloat(15.0),
	}, types.RegimeTrendingBullish)
	state.CurrentPrice = decimal.NewFromFloat(price)
	state.VWAP.SessionVWAP = decimal.NewFromFloat(2379.0)

	s := NewStandardScalping()
	res := s.Evaluate(state)

	requireRawOn(t, res.Evidence, "TREND", "EMA9_ABOVE_EMA21", state.Indicators.EMA9)
	requireRawOn(t, res.Evidence, "MOMENTUM", "MACD_BULLISH", state.Indicators.MACDHistogram)
	requireRawOn(t, res.Evidence, "MOMENTUM", "RSI_BULLISH_MID", state.Indicators.RSI)
	requireRawOn(t, res.Evidence, "TREND", "ADX_BULLISH", state.Indicators.ADX)
	requireRawOn(t, res.Evidence, "MOMENTUM", "OSMA_POSITIVE", state.Indicators.OsMA)
	requireRawOn(t, res.Evidence, "VWAP", "ABOVE_VWAP", state.VWAP.SessionVWAP)
}

// TestRawValueWarmupOmission — a not-ready indicator emits NO evidence row
// (absence ≠ fabricated zero).
func TestRawValueWarmupOmission(t *testing.T) {
	state := makeState(2380.0, features.IndicatorFeatures{
		EMA9: decimal.NewFromFloat(2380.0),
		// RSI left at zero (not ready / insufficient bars)
	}, types.RegimeTrendingBullish)
	state.Indicators.EMA21 = decimal.NewFromFloat(2378.0)

	s := NewStandardScalping()
	res := s.Evaluate(state)

	for _, ev := range res.Evidence {
		if ev.Feature == "RSI_BULLISH_MID" || ev.Feature == "RSI_BEARISH_MID" {
			t.Fatalf("RSI evidence emitted while indicator not ready (warmup violation): %+v", ev)
		}
		if !ev.RawValue.IsZero() {
			t.Fatalf("boolean/structural evidence %s/%s must not carry RawValue", ev.Pillar, ev.Feature)
		}
	}
}

var _ = time.Now // parity with acceptance_test.go style