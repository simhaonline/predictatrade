package features

import (
	"testing"

	"github.com/shopspring/decimal"
)

// P1 (prompt.md): linear regression over the last n closes.
//   slope  = Σ((x−x̄)(y−ȳ)) / Σ((x−x̄)²)          (x = bar index)
//   R²     = 1 − SS_res / SS_tot   (0..1; 0 = no linear fit)
//   LinSlopeATR = slope / ATR (ATR-normalized, per-bar drift in ATR units)
// R² ≥ 0.45 gate lives with the caller; warmup: len < n or zero ATR → zero.
func TestLinRegSlopeR2(t *testing.T) {
	// Perfect linear rise 100,101,...: slope = 1.0, R² = 1
	closes := make([]decimal.Decimal, 10)
	for i := range closes {
		closes[i] = decimal.NewFromFloat(100 + float64(i))
	}
	slope, r2 := LinRegSlopeR2(closes, 10)
	s, _ := slope.Float64()
	r, _ := r2.Float64()
	if s < 0.999 || s > 1.001 {
		t.Fatalf("perfect linear slope should be 1.0, got %v", s)
	}
	if r < 0.999 {
		t.Fatalf("perfect fit R² should be ~1, got %v", r)
	}

	// Pure noise around a mean: slope ≈ 0, R² ≈ low
	noise := []decimal.Decimal{
		decimal.NewFromInt(5), decimal.NewFromInt(4), decimal.NewFromInt(6),
		decimal.NewFromInt(4), decimal.NewFromInt(6), decimal.NewFromInt(5),
		decimal.NewFromInt(4), decimal.NewFromInt(6), decimal.NewFromInt(5), decimal.NewFromInt(4),
	}
	_, r2n := LinRegSlopeR2(noise, 10)
	rn, _ := r2n.Float64()
	if rn > 0.3 {
		t.Fatalf("noise R² should be low, got %v", rn)
	}

	// Warmup: fewer bars than n → zeros
	s2, r22 := LinRegSlopeR2(closes[:4], 10)
	if !s2.IsZero() || !r22.IsZero() {
		t.Fatal("warmup must return zeros")
	}
}

func TestLinSlopeATR(t *testing.T) {
	slope := decimal.NewFromFloat(2.0)
	atr := decimal.NewFromFloat(4.0)
	if v, _ := LinSlopeATR(slope, atr).Float64(); v != 0.5 {
		t.Fatalf("slope/ATR = 0.5, got %v", v)
	}
	// zero ATR (warmup) → zero, never inf
	if !LinSlopeATR(slope, decimal.Zero).IsZero() {
		t.Fatal("zero ATR → zero slope")
	}
}

// P1: Asian sweep — a wick pierces the Asian range boundary then closes back
// inside. Uses the SessionORB Asian high/low (existing engine — no overlap:
// reuses SessionORBFeatures.AsianHigh/Low rather than new state).
func TestAsianSweep(t *testing.T) {
	asiaHi := decimal.NewFromFloat(100)
	asiaLo := decimal.NewFromFloat(98)

	// sweep low: low < asiaLo, close back above asiaLo
	if !AsianSweep(decimal.NewFromFloat(100.5), decimal.NewFromFloat(97.5),
		decimal.NewFromFloat(99), asiaHi, asiaLo, true) {
		t.Fatal("bull: low swept asian low, close back inside → true")
	}
	// close stayed below → no sweep (breakout, not sweep)
	if AsianSweep(decimal.NewFromFloat(100.5), decimal.NewFromFloat(97.5),
		decimal.NewFromFloat(97), asiaHi, asiaLo, true) {
		t.Fatal("close below asian low = breakdown, not sweep")
	}
	// sweep high: high > asiaHi, close back below
	if !AsianSweep(decimal.NewFromFloat(100.5), decimal.NewFromFloat(98.5),
		decimal.NewFromFloat(99.5), asiaHi, asiaLo, false) {
		t.Fatal("bear: high swept asian high, close back inside → true")
	}
	// warmup: zero asian levels → false
	if AsianSweep(decimal.NewFromInt(100), decimal.NewFromInt(99), decimal.NewFromInt(99), decimal.Zero, decimal.Zero, true) {
		t.Fatal("no asian range (warmup) → false")
	}
}
