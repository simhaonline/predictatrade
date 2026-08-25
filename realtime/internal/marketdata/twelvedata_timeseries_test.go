package marketdata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

const fakeTimeSeriesJSON = `{
  "meta": {"symbol": "XAU/USD"},
  "values": [
    {"datetime": "2024-01-02 00:00:00", "open": "2000", "high": "2010", "low": "1990", "close": "2005", "volume": "100"},
    {"datetime": "2024-01-01 00:00:00", "open": "1990", "high": "2000", "low": "1980", "close": "1995", "volume": "90"}
  ],
  "status": "ok"
}`

func TestFetchTimeSeries_ParsesAndReverses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("symbol"); got != "XAU/USD" {
			t.Errorf("unexpected symbol param: %q", got)
		}
		_, _ = w.Write([]byte(fakeTimeSeriesJSON))
	}))
	defer srv.Close()

	p := NewTwelveDataProvider("test-key")
	p.apiBase = srv.URL // same-package test hook

	candles, err := p.FetchTimeSeries(context.Background(), "XAU/USD", types.TFM5, "2024-01-01", "2024-01-02")
	if err != nil {
		t.Fatalf("FetchTimeSeries: %v", err)
	}
	if len(candles) != 2 {
		t.Fatalf("expected 2 candles, got %d", len(candles))
	}
	// Must be ascending (oldest first).
	if candles[0].Time.After(candles[1].Time) {
		t.Error("candles not sorted ascending")
	}
	if candles[0].Time != time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) {
		t.Errorf("first candle time wrong: %v", candles[0].Time)
	}
	if candles[0].Symbol != "XAUUSD" {
		t.Errorf("symbol canonicalization wrong: %q", candles[0].Symbol)
	}
	if candles[0].Close.String() != "1995" {
		t.Errorf("close wrong: %s", candles[0].Close)
	}
	if candles[0].Source != "TWELVEDATA" || !candles[0].IsClosed {
		t.Errorf("source/IsClosed wrong: %q closed=%v", candles[0].Source, candles[0].IsClosed)
	}
}

func TestResolveProviderSymbol(t *testing.T) {
	p := NewTwelveDataProvider("k")
	if got := p.ResolveProviderSymbol("XAUUSD"); got != "XAU/USD" {
		t.Errorf("XAUUSD -> %q", got)
	}
	if got := p.ResolveProviderSymbol("VIX"); got != "UVXY" {
		t.Errorf("VIX -> %q", got)
	}
}

// TestPersister_SaveCandle verifies the storage path to market.candles against a
// reachable TimescaleDB. Skipped when the DB is unavailable.
func TestPersister_SaveCandle(t *testing.T) {
	dbURL := os.Getenv("PAT_DB_URL")
	if dbURL == "" {
		dbURL = "postgresql://pat_admin:pat_local_dev_only@127.0.0.1:5432/predictatrade?sslmode=disable"
	}
	p, err := NewPersister(dbURL)
	if err != nil {
		t.Skipf("TimescaleDB not reachable: %v", err)
	}
	defer p.Close()

	ctx := context.Background()
	const sym = "XAUUSDT" // test-only symbol to avoid touching real data
	// Clean slate.
	p.db.ExecContext(ctx, "DELETE FROM market.candles WHERE symbol=$1 AND source='TWELVEDATA'", sym)
	defer p.db.ExecContext(ctx, "DELETE FROM market.candles WHERE symbol=$1 AND source='TWELVEDATA'", sym)

	c := &types.Candle{
		Symbol:    sym,
		Timeframe: types.TFM5,
		Time:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Open:      decimal.NewFromInt(1990),
		High:      decimal.NewFromInt(2000),
		Low:       decimal.NewFromInt(1980),
		Close:     decimal.NewFromInt(1995),
		Volume:    90,
		Source:    "TWELVEDATA",
		Quality:   types.CandleComplete,
		IsClosed:  true,
	}
	if err := p.SaveCandle(ctx, c); err != nil {
		t.Fatalf("SaveCandle: %v", err)
	}

	var n int
	if err := p.db.QueryRowContext(ctx,
		"SELECT count(*) FROM market.candles WHERE symbol=$1 AND timeframe='M5' AND source='TWELVEDATA'", sym,
	).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 stored candle, got %d", n)
	}
}
