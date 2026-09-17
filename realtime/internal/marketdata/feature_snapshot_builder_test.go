package marketdata

import (
	"encoding/json"
	"testing"

	"github.com/predictatrade/realtime/internal/features"
	"github.com/shopspring/decimal"
)

func d(f float64) decimal.Decimal { return decimal.NewFromFloat(f) }

// P0-2: the snapshot builder must emit the full MarketState indicator read set
// with the actual values from the source state, and omit warmup (zero) fields
// rather than emitting fabricated zeros.
func TestBuildFeatureSnapshotJSON(t *testing.T) {
	st := &features.MarketState{Symbol: "XAUUSD"}
	st.Indicators.EMA9 = d(4321.25)
	st.Indicators.EMA21 = d(4320.10)
	st.Indicators.RSI = d(39.28)
	st.Indicators.ADX = d(43.82)
	st.Indicators.ATR = d(5.18)
	st.VWAP.SessionVWAP = d(4319.5)

	b, err := BuildFeatureSnapshotJSON(st)
	if err != nil {
		t.Fatalf("BuildFeatureSnapshotJSON: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m["rsi"] != 39.28 || m["adx"] != 43.82 {
		t.Fatalf("indicator reads missing: %v", m)
	}
	if m["ema9"] != 4321.25 {
		t.Fatalf("ema9 = %v, want 4321.25", m["ema9"])
	}
	if _, ok := m["ema200"]; ok {
		t.Fatalf("warmup (zero) indicators must be omitted, got ema200=%v", m["ema200"])
	}
	if _, ok := m["rsi"]; !ok {
		t.Fatal("rsi must be present")
	}
}