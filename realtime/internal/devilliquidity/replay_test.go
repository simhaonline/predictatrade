package devilliquidity

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/shopspring/decimal"
)

// TestRealCandleReplay replays real market.candles (M5) through the actual
// Engine and prints per-stage gate statistics — reproduces the production
// "0 marks despite candles flowing" symptom offline. Requires TEST_CANDLES_DSN.
func TestRealCandleReplay(t *testing.T) {
	dsn := os.Getenv("TEST_CANDLES_DSN")
	if dsn == "" {
		t.Skip("TEST_CANDLES_DSN not set — run with a read-only DB DSN to replay")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	rows, err := db.Query(`SELECT time, open::float8, high::float8, low::float8, close::float8, volume::bigint
		FROM market.candles
		WHERE symbol='XAUUSD' AND timeframe='M5' AND is_closed
		  AND time > now() - interval '5 days'
		ORDER BY time ASC`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()

	// NewEngine(DefaultConfig()) already sets enabled=true; explicit for clarity.
	e := NewEngine("", DefaultConfig())
	e.SetEnabled(true)

	n := 0
	for rows.Next() {
		var t0 sql.NullTime
		var o, h, l, c float64
		var v int64
		if err := rows.Scan(&t0, &o, &h, &l, &c, &v); err != nil {
			continue
		}
		if !t0.Valid {
			continue
		}
		ci := &CandleInput{
			Symbol: "XAUUSD", Timeframe: "M5", Time: t0.Time,
			Open: decimal.NewFromFloat(o), High: decimal.NewFromFloat(h),
			Low: decimal.NewFromFloat(l), Close: decimal.NewFromFloat(c),
			Volume: v, IsClosed: true, Spread: 0.3, Digits: 2, FeedSource: "REPLAY",
		}
		// synchronous path (bypasses Ingest goroutine)
		if err := e.ProcessCandle(ci); err != nil {
			t.Fatalf("process: %v", err)
		}
		n++
	}
	st := e.Stats()
	fmt.Printf("REPLAY: fed=%d processed=%d marks_created=%d marks_in_memory=%d\n",
		n, st.CandlesProcessed, st.MarksCreated, len(e.ActiveMarks()))
	if n > 1000 && st.MarksCreated == 0 {
		t.Errorf("BUG REPRODUCED: %d candles fed, 0 marks created", n)
	}
}
