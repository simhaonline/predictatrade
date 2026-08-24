package types

import (
	"testing"
	"time"
)

// prompt.md Sections 13-15: bar close time must be open + period, never a
// copy of the open time.
func TestTimeframeDuration(t *testing.T) {
	cases := map[Timeframe]time.Duration{
		TFM1:  time.Minute,
		TFM5:  5 * time.Minute,
		TFM15: 15 * time.Minute,
		TFM30: 30 * time.Minute,
		TFH1:  time.Hour,
		TFH4:  4 * time.Hour,
		TFD1:  24 * time.Hour,
		TFW1:  7 * 24 * time.Hour,
	}
	for tf, want := range cases {
		if got := tf.Duration(); got != want {
			t.Errorf("%s.Duration() = %v, want %v", tf, got, want)
		}
	}
	if Timeframe("BOGUS").Duration() != 0 {
		t.Error("unknown timeframe must return 0")
	}
}

func TestBarCloseTimeIsOpenPlusPeriod(t *testing.T) {
	open := time.Date(2026, 8, 24, 13, 0, 0, 0, time.UTC)
	close := open.Add(TFH1.Duration())
	if !close.Equal(open.Add(time.Hour)) {
		t.Fatalf("H1 close = %v, want open+1h", close)
	}
	if close.Equal(open) {
		t.Fatal("close must not equal open (regression: identical timestamps)")
	}
}
