package devilliquidity

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func cndl(symbol, tf string, o, h, l, c float64, vol int64, t time.Time) *CandleInput {
	return &CandleInput{
		Symbol: symbol, Timeframe: tf, Time: t,
		Open: decimal.NewFromFloat(o), High: decimal.NewFromFloat(h),
		Low: decimal.NewFromFloat(l), Close: decimal.NewFromFloat(c),
		Volume: vol, IsClosed: true, Spread: 0.1, Digits: 2, FeedSource: "TEST",
	}
}

func run(e *Engine, cs ...*CandleInput) {
	for _, c := range cs {
		_ = e.ProcessCandle(c)
	}
}

func TestClosedCandleRule(t *testing.T) {
	e := NewEngine("", DefaultConfig())
	open := cndl("XAUUSD", "M5", 1000, 1040, 1000, 1039, 100, time.Now())
	open.IsClosed = false
	if err := e.ProcessCandle(open); err != nil {
		t.Fatal(err)
	}
	if len(e.AllMarks()) != 0 {
		t.Fatalf("non-closed candle must not create marks, got %d", len(e.AllMarks()))
	}
}

func TestBullishMarkDetectionAndReversal(t *testing.T) {
	e := NewEngine("", DefaultConfig())
	base := time.Now()
	// Warmup neutral candles (~range 7, body 5) to seed ATR/body medians.
	var warm []CandleInput
	for i := 0; i < 10; i++ {
		warm = append(warm, *cndl("XAUUSD", "M5", 1000, 1006, 999, 1005, 100, base.Add(time.Duration(i)*time.Minute)))
	}
	run(e, toPtrs(warm)...)

	// Bullish displacement candle: open=low (no lower wick), close near high.
	disp := cndl("XAUUSD", "M5", 1005, 1045, 1005, 1044, 100, base.Add(11*time.Minute))
	run(e, disp)

	marks := e.AllMarks()
	if len(marks) != 1 {
		t.Fatalf("expected 1 mark, got %d", len(marks))
	}
	m := marks[0]
	if m.Direction != DirBullish {
		t.Fatalf("expected BULLISH, got %s", m.Direction)
	}
	if m.State != StateActive {
		t.Fatalf("expected ACTIVE, got %s", m.State)
	}
	if m.MarkQuality < 40 || m.MarkQuality > 100 {
		t.Fatalf("mark quality out of range: %f", m.MarkQuality)
	}

	// Sweep below the mark (no reclaim yet).
	sweep := cndl("XAUUSD", "M5", 1005, 1006, 1000, 1001, 100, base.Add(12*time.Minute))
	run(e, sweep)
	if e.AllMarks()[0].State != StateSwept {
		t.Fatalf("expected SWEPT, got %s", e.AllMarks()[0].State)
	}

	// Reclaim + reversal confirmation.
	rev := cndl("XAUUSD", "M5", 1005, 1040, 1004, 1038, 100, base.Add(13*time.Minute))
	run(e, rev)
	final := e.AllMarks()[0].State
	if final != StateReversalConfirmed && final != StateSignalEligible {
		t.Fatalf("expected REVERSAL_CONFIRMED/SIGNAL_ELIGIBLE, got %s", final)
	}
	if e.AllMarks()[0].ReversalScore <= 0 {
		t.Fatalf("reversal score must be positive, got %f", e.AllMarks()[0].ReversalScore)
	}
}

func TestMarkExpiry(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MarkExpiryBars = 5
	e := NewEngine("", cfg)
	base := time.Now()
	var warm []CandleInput
	for i := 0; i < 8; i++ {
		warm = append(warm, *cndl("XAUUSD", "M5", 1000, 1006, 999, 1005, 100, base.Add(time.Duration(i)*time.Minute)))
	}
	run(e, toPtrs(warm)...)
	disp := cndl("XAUUSD", "M5", 1005, 1045, 1005, 1044, 100, base.Add(9*time.Minute))
	run(e, disp)
	id := e.AllMarks()[0].ID
	// Feed 7 neutral candles far from the mark (no touch/sweep).
	for i := 10; i < 17; i++ {
		run(e, cndl("XAUUSD", "M5", 1100, 1106, 1099, 1105, 100, base.Add(time.Duration(i)*time.Minute)))
	}
	var found *DevilMark
	for _, m := range e.AllMarks() {
		if m.ID == id {
			found = m
		}
	}
	if found == nil {
		t.Fatal("original mark disappeared")
	}
	if found.State != StateExpired {
		t.Fatalf("expected EXPIRED, got %s", found.State)
	}
}

func TestBullishMarkInvalidation(t *testing.T) {
	e := NewEngine("", DefaultConfig())
	base := time.Now()
	var warm []CandleInput
	for i := 0; i < 8; i++ {
		warm = append(warm, *cndl("XAUUSD", "M5", 1000, 1006, 999, 1005, 100, base.Add(time.Duration(i)*time.Minute)))
	}
	run(e, toPtrs(warm)...)
	disp := cndl("XAUUSD", "M5", 1005, 1045, 1005, 1044, 100, base.Add(9*time.Minute))
	run(e, disp)
	// Deep break below the mark beyond max sweep depth -> invalidation.
	breakC := cndl("XAUUSD", "M5", 1005, 1006, 960, 970, 100, base.Add(10*time.Minute))
	run(e, breakC)
	if e.AllMarks()[0].State != StateInvalidated {
		t.Fatalf("expected INVALIDATED, got %s", e.AllMarks()[0].State)
	}
}

func TestBearishMarkDetection(t *testing.T) {
	e := NewEngine("", DefaultConfig())
	base := time.Now()
	var warm []CandleInput
	for i := 0; i < 10; i++ {
		warm = append(warm, *cndl("XAUUSD", "M5", 1005, 1006, 999, 1000, 100, base.Add(time.Duration(i)*time.Minute)))
	}
	run(e, toPtrs(warm)...)
	// Bearish displacement: close<open, open=high (no upper wick), close near low.
	disp := cndl("XAUUSD", "M5", 1045, 1045, 1005, 1006, 100, base.Add(11*time.Minute))
	run(e, disp)
	marks := e.AllMarks()
	if len(marks) != 1 || marks[0].Direction != DirBearish {
		t.Fatalf("expected 1 BEARISH mark, got %+v", marks)
	}
}

func toPtrs(in []CandleInput) []*CandleInput {
	out := make([]*CandleInput, len(in))
	for i := range in {
		out[i] = &in[i]
	}
	return out
}
