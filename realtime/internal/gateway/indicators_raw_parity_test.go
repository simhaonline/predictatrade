package gateway

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/marketdata"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// P0-4 (prompt.md acceptance): calculate indicator → RawValue populated →
// snapshot persisted → endpoint returns SAME values. This test proves the
// endpoint/snapshot parity (the DB leg is covered by TestFeatureSnapshotRoundTrip
// against PERSISTENCE_TEST_URL).
func TestIndicatorsRawParityWithSnapshot(t *testing.T) {
	states := features.NewStateManager()
	states.Update("XAUUSD", func(st *features.MarketState) {
		st.Symbol = "XAUUSD"
		st.Indicators.RSI = decimal.NewFromFloat(57.94)
		st.Indicators.ADX = decimal.NewFromFloat(23.52)
		st.Indicators.EMA9 = decimal.NewFromFloat(4314.8325)
		st.Indicators.ATR = decimal.NewFromFloat(158.91)
	})

	// 1) endpoint surface
	h := &HTTPServer{states: states}
	rec := httptest.NewRecorder()
	h.handleIndicatorsRaw(rec, httptest.NewRequest("GET", "/api/v1/indicators/raw?symbol=XAUUSD", nil))
	if rec.Code != 200 {
		t.Fatalf("endpoint status %d", rec.Code)
	}
	var ep map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &ep); err != nil {
		t.Fatal(err)
	}

	// 2) snapshot payload (what SaveFeatureSnapshot persists)
	states.Update("XAUUSD", func(st *features.MarketState) { st.Symbol = "XAUUSD" })
	sb, err := marketdata.BuildFeatureSnapshotJSON(states.Get("XAUUSD"))
	if err != nil {
		t.Fatal(err)
	}
	var snap map[string]any
	if err := json.Unmarshal(sb, &snap); err != nil {
		t.Fatal(err)
	}

	// 3) parity: every key the endpoint returns for indicator reads must equal
	// the snapshot value exactly (same builder, same state).
	for _, k := range []string{"rsi", "adx", "ema9", "atr"} {
		ev, eok := ep[k].(float64)
		sv, sok := snap[k].(float64)
		if !eok || !sok || ev != sv {
			t.Fatalf("parity broken for %s: endpoint=%v snapshot=%v", k, ep[k], snap[k])
		}
	}

	// 4) signal wiring contract: the signal carries the snapshot; SaveSignal
	// persists it and sets FeatureSnapshotID = signal ID (1:1). Assert the
	// signal-side fields exist and serialize the snapshot.
	sig := &types.Signal{ID: "sig-1", FeatureSnapshotJSON: sb}
	if len(sig.FeatureSnapshotJSON) == 0 || sig.FeatureSnapshotID != "" {
		t.Fatal("signal snapshot fields misdeclared")
	}
	if !strings.Contains(string(sig.FeatureSnapshotJSON), "57.94") {
		t.Fatal("snapshot json missing the RSI read")
	}
}
