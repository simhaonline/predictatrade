package devilliquidity

import (
	"testing"
	"time"
)

// Regression (BE-6 sprint discovery): the candle right after a displacement
// (e.g. the reversal leg) could itself qualify as a NEW displacement mark — the
// median body shifted after the first mark — registering a second mark with a
// different level but the same intent. This nondeterministically broke
// TestBullishMarkDetectionAndReversal (AllMarks() order is map-random) and,
// more importantly, double-charged the same liquidity level in production.
func TestNoDuplicateMarkWithinRecencyWindow(t *testing.T) {
	for iter := 0; iter < 100; iter++ {
		e := NewEngine("", DefaultConfig())
		base := time.Now()
		var warm []CandleInput
		for i := 0; i < 10; i++ {
			warm = append(warm, *cndl("XAUUSD", "M5", 1000, 1006, 999, 1005, 100, base.Add(time.Duration(i)*time.Minute)))
		}
		run(e, toPtrs(warm)...)
		run(e, cndl("XAUUSD", "M5", 1005, 1045, 1005, 1044, 100, base.Add(11*time.Minute)))
		run(e, cndl("XAUUSD", "M5", 1005, 1006, 1000, 1001, 100, base.Add(12*time.Minute)))
		run(e, cndl("XAUUSD", "M5", 1005, 1040, 1004, 1038, 100, base.Add(13*time.Minute)))

		marks := e.AllMarks()
		if len(marks) != 1 {
			t.Fatalf("iter %d: expected exactly 1 mark, got %d", iter, len(marks))
		}
		if m := marks[0]; m.State != StateReversalConfirmed && m.State != StateSignalEligible {
			t.Fatalf("iter %d: expected REVERSAL_CONFIRMED/SIGNAL_ELIGIBLE, got %s", iter, m.State)
		}
	}
}

// A NEW mark far beyond the recency window (different liquidity level, older
// mark resolved) must still be registrable — guard must not over-suppress.
func TestDuplicateGuardAllowsLegitimateNewMark(t *testing.T) {
	e := NewEngine("", DefaultConfig())
	base := time.Now()
	var warm []CandleInput
	for i := 0; i < 10; i++ {
		warm = append(warm, *cndl("XAUUSD", "M5", 1000, 1006, 999, 1005, 100, base.Add(time.Duration(i)*time.Minute)))
	}
	run(e, toPtrs(warm)...)
	// First displacement low (bullish support mark at 1005).
	run(e, cndl("XAUUSD", "M5", 1005, 1045, 1005, 1044, 100, base.Add(11*time.Minute)))
	// Drive the first mark to a terminal state (deep invalidation sweep).
	for i := 0; i < 12; i++ {
		run(e, cndl("XAUUSD", "M5", 1010, 1012, 984, 985, 100, base.Add((12+time.Duration(i))*time.Minute)))
	}
	if m := e.AllMarks(); len(m) > 0 && m[0].State == StateInvalidated || len(m) == 0 {
		// terminal — a brand-new bullish displacement far above, well past the
		// recency window, must create a fresh mark.
		var warm2 []CandleInput
		base2 := base.Add(30 * time.Minute)
		for i := 0; i < 12; i++ {
			warm2 = append(warm2, *cndl("XAUUSD", "M5", 1050, 1056, 1049, 1055, 100, base2.Add(time.Duration(i)*time.Minute)))
		}
		run(e, toPtrs(warm2)...)
		run(e, cndl("XAUUSD", "M5", 1055, 1095, 1055, 1094, 100, base2.Add(11*time.Minute)))
		if got := len(e.AllMarks()); got != 1 {
			t.Fatalf("legitimate new mark blocked: got %d marks", got)
		}
	}
}
