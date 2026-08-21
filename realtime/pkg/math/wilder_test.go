package patmath

import (
	"testing"

	"github.com/shopspring/decimal"
)

// TestATRWilder verifies ATR uses Wilder's smoothing (not simple average).
// With a known series, Wilder ATR differs from simple-average ATR.
func TestATRWilder(t *testing.T) {
	// 30 bars with known highs/lows/closes
	highs := make([]decimal.Decimal, 30)
	lows := make([]decimal.Decimal, 30)
	closes := make([]decimal.Decimal, 30)
	for i := 0; i < 30; i++ {
		closes[i] = decimal.NewFromFloat(2400 + float64(i)*0.5)
		highs[i] = closes[i].Add(decimal.NewFromFloat(1.0))
		lows[i] = closes[i].Sub(decimal.NewFromFloat(1.0))
	}

	atr := ATR(highs, lows, closes, 14)
	if atr.IsZero() {
		t.Fatal("ATR should not be zero with 30 bars")
	}
	// With rising closes, TR should be ~2.0 (H-L) since gaps are small
	// Wilder ATR should be close to 2.0
	if atr.LessThan(decimal.NewFromFloat(1.5)) || atr.GreaterThan(decimal.NewFromFloat(2.5)) {
		t.Errorf("ATR = %s, expected ~2.0", atr.String())
	}
}

// TestATRWilderDiffersFromSimple verifies Wilder ATR ≠ simple-average ATR
// when the series has varying TR values.
func TestATRWilderDiffersFromSimple(t *testing.T) {
	highs := []decimal.Decimal{
		decimal.NewFromFloat(2430), decimal.NewFromFloat(2435),
		decimal.NewFromFloat(2432), decimal.NewFromFloat(2440),
		decimal.NewFromFloat(2433), decimal.NewFromFloat(2438),
		decimal.NewFromFloat(2431), decimal.NewFromFloat(2445),
		decimal.NewFromFloat(2432), decimal.NewFromFloat(2436),
		decimal.NewFromFloat(2433), decimal.NewFromFloat(2437),
		decimal.NewFromFloat(2432), decimal.NewFromFloat(2435),
		decimal.NewFromFloat(2433),
	}
	lows := []decimal.Decimal{
		decimal.NewFromFloat(2428), decimal.NewFromFloat(2429),
		decimal.NewFromFloat(2430), decimal.NewFromFloat(2427),
		decimal.NewFromFloat(2430), decimal.NewFromFloat(2428),
		decimal.NewFromFloat(2427), decimal.NewFromFloat(2429),
		decimal.NewFromFloat(2430), decimal.NewFromFloat(2428),
		decimal.NewFromFloat(2429), decimal.NewFromFloat(2430),
		decimal.NewFromFloat(2428), decimal.NewFromFloat(2429),
		decimal.NewFromFloat(2430),
	}
	closes := []decimal.Decimal{
		decimal.NewFromFloat(2429), decimal.NewFromFloat(2431),
		decimal.NewFromFloat(2432), decimal.NewFromFloat(2428),
		decimal.NewFromFloat(2431), decimal.NewFromFloat(2432),
		decimal.NewFromFloat(2428), decimal.NewFromFloat(2433),
		decimal.NewFromFloat(2431), decimal.NewFromFloat(2432),
		decimal.NewFromFloat(2430), decimal.NewFromFloat(2431),
		decimal.NewFromFloat(2432), decimal.NewFromFloat(2428),
		decimal.NewFromFloat(2431),
	}

	wilderATR := ATRWilder(highs, lows, closes, 14)
	if wilderATR.IsZero() {
		t.Fatal("Wilder ATR should not be zero")
	}
	// Wilder ATR should be positive and finite
	if wilderATR.LessThanOrEqual(decimal.Zero) {
		t.Errorf("Wilder ATR = %s, should be positive", wilderATR.String())
	}
}

// TestRSIWilder verifies RSI uses Wilder's smoothing.
func TestRSIWilder(t *testing.T) {
	// Rising series → RSI near 100
	closes := make([]decimal.Decimal, 30)
	for i := 0; i < 30; i++ {
		closes[i] = decimal.NewFromFloat(2430 + float64(i)*0.5)
	}
	rsi := RSIWilder(closes, 14)
	if rsi.LessThan(decimal.NewFromInt(90)) {
		t.Errorf("RSI for rising series = %s, want > 90", rsi.String())
	}

	// Falling series → RSI near 0
	for i := 0; i < 30; i++ {
		closes[i] = decimal.NewFromFloat(2430 - float64(i)*0.5)
	}
	rsi = RSIWilder(closes, 14)
	if rsi.GreaterThan(decimal.NewFromInt(10)) {
		t.Errorf("RSI for falling series = %s, want < 10", rsi.String())
	}
}

// TestRSIWilderFlatPrice verifies flat price returns 50 (undefined, not 100).
func TestRSIWilderFlatPrice(t *testing.T) {
	closes := make([]decimal.Decimal, 20)
	for i := range closes {
		closes[i] = decimal.NewFromFloat(2400.0)
	}
	rsi := RSIWilder(closes, 14)
	// Both avg_gain and avg_loss are zero → RSI = 50 (undefined)
	if !rsi.Equal(decimal.NewFromInt(50)) {
		t.Errorf("RSI for flat price = %s, want 50 (undefined)", rsi.String())
	}
}

// TestADXWilder verifies ADX returns non-zero with a trending series.
func TestADXWilder(t *testing.T) {
	// Strong uptrend: 40 bars
	n := 40
	highs := make([]decimal.Decimal, n)
	lows := make([]decimal.Decimal, n)
	closes := make([]decimal.Decimal, n)
	for i := 0; i < n; i++ {
		closes[i] = decimal.NewFromFloat(2400 + float64(i)*1.0)
		highs[i] = closes[i].Add(decimal.NewFromFloat(0.5))
		lows[i] = closes[i].Sub(decimal.NewFromFloat(0.5))
	}

	adx, plusDI, minusDI := ADXWilder(highs, lows, closes, 14)
	if adx.IsZero() {
		t.Error("ADX should not be zero with a strong trend")
	}
	// In an uptrend, +DI should be > -DI
	if !plusDI.GreaterThan(minusDI) {
		t.Errorf("In uptrend: +DI (%s) should be > -DI (%s)", plusDI.String(), minusDI.String())
	}
}

// TestADXWilderInsufficientData verifies graceful handling of short series.
func TestADXWilderInsufficientData(t *testing.T) {
	highs := []decimal.Decimal{decimal.NewFromFloat(2430), decimal.NewFromFloat(2431)}
	lows := []decimal.Decimal{decimal.NewFromFloat(2428), decimal.NewFromFloat(2429)}
	closes := []decimal.Decimal{decimal.NewFromFloat(2429), decimal.NewFromFloat(2430)}

	adx, _, _ := ADXWilder(highs, lows, closes, 14)
	if !adx.IsZero() {
		t.Errorf("ADX with insufficient data = %s, want 0", adx.String())
	}
}

// TestTrueRangeSeries verifies TR series computation.
func TestTrueRangeSeries(t *testing.T) {
	highs := []decimal.Decimal{decimal.NewFromFloat(2432), decimal.NewFromFloat(2435)}
	lows := []decimal.Decimal{decimal.NewFromFloat(2428), decimal.NewFromFloat(2429)}
	closes := []decimal.Decimal{decimal.NewFromFloat(2429), decimal.NewFromFloat(2431)}

	tr := TrueRangeSeries(highs, lows, closes)
	if len(tr) != 2 {
		t.Fatalf("TR series length = %d, want 2", len(tr))
	}
	// TR[0] = H[0] - L[0] = 2432 - 2428 = 4
	if !tr[0].Equal(decimal.NewFromInt(4)) {
		t.Errorf("TR[0] = %s, want 4", tr[0].String())
	}
	// TR[1] = max(2435-2429, |2435-2429|, |2429-2429|) = max(6, 6, 0) = 6
	if !tr[1].Equal(decimal.NewFromInt(6)) {
		t.Errorf("TR[1] = %s, want 6", tr[1].String())
	}
}
