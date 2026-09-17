package gateway

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/marketdata"
	"github.com/shopspring/decimal"
)

// P0-3 (prompt.md): GET /api/v1/indicators/raw returns the live indicator
// read set for a symbol — the SAME values persisted in feature snapshots
// (both paths use marketdata.BuildFeatureSnapshotJSON).

func featuresSnapshotForTest(st *features.MarketState) ([]byte, error) {
	return marketdata.BuildFeatureSnapshotJSON(st)
}

func TestHandleIndicatorsRaw_ReturnsMarketStateReads(t *testing.T) {
	states := features.NewStateManager()
	states.Update("XAUUSD", func(st *features.MarketState) {
		st.Symbol = "XAUUSD"
		st.Indicators.RSI = decimal.NewFromFloat(39.28)
		st.Indicators.ADX = decimal.NewFromFloat(43.82)
	})
	st := states.Get("XAUUSD")
	if st == nil {
		t.Fatal("state nil")
	}

	h := &HTTPServer{states: states}
	req := httptest.NewRequest("GET", "/api/v1/indicators/raw?symbol=XAUUSD", nil)
	rec := httptest.NewRecorder()
	h.handleIndicatorsRaw(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{`"rsi":39.28`, `"adx":43.82`, `"symbol":"XAUUSD"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %s — got %s", want, body[:min(len(body), 300)])
		}
	}
	b, _ := marketdata.BuildFeatureSnapshotJSON(states.Get("XAUUSD"))
	if !strings.Contains(string(b), "39.28") {
		t.Fatal("snapshot builder parity broken")
	}
}

func TestHandleIndicatorsRaw_MissingSymbol(t *testing.T) {
	h := &HTTPServer{states: features.NewStateManager()}
	req := httptest.NewRequest("GET", "/api/v1/indicators/raw", nil)
	rec := httptest.NewRecorder()
	h.handleIndicatorsRaw(rec, req)
	if rec.Code != 400 {
		t.Fatalf("missing symbol must 400, got %d", rec.Code)
	}
}
