package features

import (
	"testing"

	"github.com/shopspring/decimal"
)

// P1 (prompt.md): Choppiness Index —
//   100 * LOG10(SUM(TR_i, n) / MaxHi(n) - MaxLo(n)) / LOG10(n)
// Values near 100 = choppy (range), near 0 = trending.
// Reference formula; warmup: < n bars → zero (omitted by snapshot builder).
func TestChoppinessIndex(t *testing.T) {
	// Constant-range series: high-low fixed, TR sums predictable.
	// 14 bars alternating up/down inside a fixed 1.0 band.
	highs := []decimal.Decimal{}
	lows := []decimal.Decimal{}
	closes := []decimal.Decimal{}
	for i := 0; i < 14; i++ {
		h := decimal.NewFromFloat(100 + float64(i%2))   // alternates 100,101
		l := h.Sub(decimal.NewFromFloat(1))              // 99,100
		c := h                                           // close at high
		highs = append(highs, h); lows = append(lows, l); closes = append(closes, c)
	}
	chop := ChoppinessIndex(highs, lows, closes, 14)
	if chop.IsZero() {
		t.Fatal("chop computed zero for a full window")
	}
	if v, _ := chop.Float64(); v < 0 || v > 100 {
		t.Fatalf("chop outside [0,100]: %v", v)
	}
	// A strongly trending monotonic series: each bar closes above the prior high,
	// so TR == full range each bar and SUM(TR) approaches range*(n) —
	// chop trends toward lower values than the alternating case.
	trendHighs, trendLows, trendCloses := []decimal.Decimal{}, []decimal.Decimal{}, []decimal.Decimal{}
	for i := 0; i < 14; i++ {
		h := decimal.NewFromFloat(100 + float64(i))     // rising 1.0/bar
		l := h.Sub(decimal.NewFromFloat(0.5))            // 0.5 range bars
		trendHighs = append(trendHighs, h); trendLows = append(trendLows, l); trendCloses = append(trendCloses, h)
	}
	chopTrend := ChoppinessIndex(trendHighs, trendLows, trendCloses, 14)
	c1, _ := chop.Float64()
	c2, _ := chopTrend.Float64()
	if chopTrend.GreaterThanOrEqual(chop) {
		t.Fatalf("trending chop (%v) should be LOWER than alternating chop (%v)", chopTrend, chop)
	}
	_ = c1; _ = c2
	// Warmup: fewer bars than n → zero
	chopWarm := ChoppinessIndex(highs[:5], lows[:5], closes[:5], 14)
	if !chopWarm.IsZero() {
		t.Fatalf("warmup must return zero, got %v", chopWarm)
	}
}

// P1: Squeeze state — Bollinger Bands inside Keltner Channels.
// squeezeOn = BB upper < KC upper AND BB lower > KC lower.
// squeezeRelease = previous squeezeOn && now off, direction by close vs BB middle.
func TestSqueezeState(t *testing.T) {
	// Narrow BB, wide KC → squeeze on
	on := SqueezeState(
		decimal.NewFromFloat(100.5), decimal.NewFromFloat(99.5),   // BB upper/lower
		decimal.NewFromFloat(101.0), decimal.NewFromFloat(99.0),   // KC upper/lower
		false)
	if !on.On {
		t.Fatal("squeeze should be ON (BB inside KC)")
	}
	// BB wider than KC → off
	off := SqueezeState(
		decimal.NewFromFloat(101.5), decimal.NewFromFloat(98.5),
		decimal.NewFromFloat(101.0), decimal.NewFromFloat(99.0),
		true) // previous bar was on → release
	if off.On {
		t.Fatal("squeeze should be OFF")
	}
	if !off.Release {
		t.Fatal("release should be true (on→off transition)")
	}
	// Warmup: zero bands → off, no release
	w := SqueezeState(decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, true)
	if w.On || w.Release {
		t.Fatal("zero bands (warmup) must produce off/no-release")
	}
}
