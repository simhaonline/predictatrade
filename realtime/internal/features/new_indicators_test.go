package features

import (
	"math"
	"time"
	"testing"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// === Rolling Statistics Tests (SOW Section 11) ===

func TestRollingStats_Mean(t *testing.T) {
	rs := NewRollingStats(10, 5)
	for i := 0; i < 10; i++ {
		rs.Add(float64(i + 1)) // 1..10, mean = 5.5
	}
	if !rs.Ready() {
		t.Fatal("expected ready after 10 samples with minSamples=5")
	}
	mean := rs.Mean()
	if math.Abs(mean-5.5) > 0.001 {
		t.Errorf("expected mean=5.5, got %f", mean)
	}
}

func TestRollingStats_ZScore(t *testing.T) {
	rs := NewRollingStats(20, 10)
	for i := 0; i < 20; i++ {
		rs.Add(float64(i + 1)) // 1..20
	}
	// Z-score of the mean should be ~0
	z := rs.ZScore(10.5) // 10.5 is the mean of 1..20
	if math.Abs(z) > 0.01 {
		t.Errorf("expected Z-score ~0 for mean, got %f", z)
	}
	// Z-score of max value should be positive
	z = rs.ZScore(20.0)
	if z <= 0 {
		t.Errorf("expected positive Z-score for max, got %f", z)
	}
}

func TestRollingStats_ZeroStdDev(t *testing.T) {
	rs := NewRollingStats(10, 5)
	for i := 0; i < 10; i++ {
		rs.Add(5.0) // All same value
	}
	// Zero stddev should return 0, not NaN or infinity
	z := rs.ZScore(5.0)
	if math.IsNaN(z) || math.IsInf(z, 0) {
		t.Errorf("Z-score should not be NaN/Inf for constant series, got %f", z)
	}
	if z != 0 {
		t.Errorf("expected Z-score=0 for constant series, got %f", z)
	}
}

func TestRollingStats_NotReady(t *testing.T) {
	rs := NewRollingStats(50, 20)
	rs.Add(1.0)
	rs.Add(2.0)
	if rs.Ready() {
		t.Error("should not be ready with only 2 samples (minSamples=20)")
	}
	if rs.Mean() != 0 {
		t.Error("Mean should return 0 when not ready")
	}
}

func TestRollingStats_InitFromHistory(t *testing.T) {
	rs := NewRollingStats(10, 5)
	history := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	rs.InitFromHistory(history)
	if !rs.Ready() {
		t.Fatal("expected ready after InitFromHistory with 10 values, minSamples=5")
	}
	mean := rs.Mean()
	if math.Abs(mean-5.5) > 0.001 {
		t.Errorf("expected mean=5.5, got %f", mean)
	}
}

func TestRollingStats_RollingWindow(t *testing.T) {
	rs := NewRollingStats(5, 3)
	for i := 0; i < 10; i++ {
		rs.Add(float64(i + 1))
	}
	// Window of 5: last 5 values are 6,7,8,9,10 → mean = 8
	mean := rs.Mean()
	if math.Abs(mean-8.0) > 0.001 {
		t.Errorf("expected mean=8.0 for last 5 values, got %f", mean)
	}
}

func TestRollingStats_NaNHandling(t *testing.T) {
	rs := NewRollingStats(10, 5)
	rs.Add(1.0)
	rs.Add(math.NaN()) // Should be skipped
	rs.Add(2.0)
	if rs.Count() != 2 {
		t.Errorf("expected count=2 after NaN skip, got %d", rs.Count())
	}
}

// === Parabolic SAR Tests (SOW Section 5) ===

func TestParabolicSAR_BullishTrend(t *testing.T) {
	engine := NewSAREngine(0.02, 0.20)
	// Simulate a bullish trend: rising prices
	for i := 0; i < 30; i++ {
		price := 2000.0 + float64(i)*2.0
		c := makeCandle(price-1, price+1, price-2, price)
		result := engine.Process(c)
		if i >= 2 && !result.Ready {
			t.Errorf("expected SAR ready after 2 bars, bar %d", i)
		}
	}
	// In a strong uptrend, SAR should be long
	c := makeCandle(2058, 2062, 2057, 2060)
	result := engine.Process(c)
	if !result.IsLong {
		t.Error("expected SAR to be long in uptrend")
	}
}

func TestParabolicSAR_BearishReversal(t *testing.T) {
	engine := NewSAREngine(0.02, 0.20)
	// Uptrend then sharp drop
	for i := 0; i < 20; i++ {
		price := 2000.0 + float64(i)*2.0
		engine.Process(makeCandle(price-1, price+1, price-2, price))
	}
	// Sharp drop
	engine.Process(makeCandle(2020, 2025, 2010, 2012))
	engine.Process(makeCandle(2005, 2015, 1995, 1998))
	result := engine.Process(makeCandle(1990, 2000, 1985, 1988))
	// Should have reversed to short
	if result.IsLong {
		// May not have reversed yet depending on SAR level; check next bar
		result = engine.Process(makeCandle(1985, 1995, 1970, 1975))
		if result.IsLong {
			t.Error("expected SAR to reverse to short after sharp drop")
		}
	}
}

func TestParabolicSAR_Warmup(t *testing.T) {
	engine := NewSAREngine(0.02, 0.20)
	c := makeCandle(2000, 2005, 1995, 2002)
	result := engine.Process(c)
	if result.Ready {
		t.Error("SAR should not be ready after only 1 bar")
	}
	c2 := makeCandle(2002, 2008, 2001, 2006)
	result = engine.Process(c2)
	if !result.Ready {
		t.Error("SAR should be ready after 2 bars")
	}
}

// === Ichimoku Tests (SOW Section 6) ===

func TestIchimoku_TenkanKijun(t *testing.T) {
	engine := NewIchimokuEngine(9, 26, 52, 26)
	// Generate 80 bars of data
	for i := 0; i < 80; i++ {
		price := 2000.0 + float64(i)*0.5
		c := makeCandle(price-1, price+1, price-2, price)
		engine.Process(c)
	}
	// After 80 bars, Tenkan and Kijun should be computed
	c := makeCandle(2040, 2042, 2038, 2041)
	result := engine.Process(c)
	if !result.Ready {
		t.Error("Ichimoku should be ready after 80 bars (52+26=78 required)")
	}
	if result.Tenkan.IsZero() {
		t.Error("Tenkan should not be zero after warmup")
	}
	if result.Kijun.IsZero() {
		t.Error("Kijun should not be zero after warmup")
	}
}

func TestIchimoku_NoLookAheadBias(t *testing.T) {
	engine := NewIchimokuEngine(9, 26, 52, 26)
	for i := 0; i < 78; i++ {
		price := 2000.0 + float64(i)*0.5
		c := makeCandle(price-1, price+1, price-2, price)
		result := engine.Process(c)
		// Senkou spans should not be "ready" until enough displacement data exists
		if i < 77 && result.Ready {
			t.Errorf("Ichimoku should not be ready before bar 78, ready at bar %d", i)
		}
	}
}

func TestIchimoku_CloudPosition(t *testing.T) {
	engine := NewIchimokuEngine(9, 26, 52, 26)
	// Generate enough data for readiness
	for i := 0; i < 80; i++ {
		price := 2000.0 + float64(i)*0.5
		engine.Process(makeCandle(price-1, price+1, price-2, price))
	}
	c := makeCandle(2040, 2042, 2038, 2041)
	result := engine.Process(c)
	if result.Ready {
		// Price should be above, below, or in cloud
		if !result.AboveCloud && !result.BelowCloud && !result.InCloud {
			t.Error("one of AboveCloud/BelowCloud/InCloud must be true when ready")
		}
	}
}

// === Stochastic RSI Tests (SOW Section 7) ===

func TestStochRSI_Warmup(t *testing.T) {
	engine := NewStochRSIEngine(14, 14, 3, 3)
	for i := 0; i < 30; i++ {
		price := 2000.0 + float64(i%5)
		c := makeCandle(price-1, price+1, price-2, price)
		result := engine.Process(c)
		if i < 27 && result.Ready {
			t.Errorf("StochRSI should not be ready before bar 28, ready at bar %d", i)
		}
	}
}

func TestStochRSI_Range(t *testing.T) {
	engine := NewStochRSIEngine(14, 14, 3, 3)
	for i := 0; i < 50; i++ {
		price := 2000.0 + float64(i%7)*2.0
		c := makeCandle(price-1, price+1, price-2, price)
		result := engine.Process(c)
		if result.Ready {
			// StochRSI should be between 0 and 1 (or very close)
			f, _ := result.Raw.Float64()
			if f < -0.01 || f > 1.01 {
				t.Errorf("StochRSI raw should be 0-1, got %f", f)
			}
		}
	}
}

func TestStochRSI_ZeroDenominator(t *testing.T) {
	engine := NewStochRSIEngine(14, 14, 3, 3)
	// Constant prices → RSI will be constant → zero denominator
	for i := 0; i < 50; i++ {
		c := makeCandle(2000, 2001, 1999, 2000)
		result := engine.Process(c)
		if result.Ready {
			// Should not panic or produce NaN
			f, _ := result.Raw.Float64()
			if math.IsNaN(f) {
				t.Error("StochRSI should not be NaN for constant prices")
			}
		}
	}
}

// === Fibonacci Tests (SOW Section 12) ===

func TestFibonacci_BullishSwing(t *testing.T) {
	engine := NewFibonacciEngine(nil)
	structure := StructureFeatures{
		SwingHighs:   []decimal.Decimal{decimal.NewFromFloat(2050.0)},
		SwingLows:    []decimal.Decimal{decimal.NewFromFloat(2000.0)},
		CurrentTrend: "bullish",
	}
	c := makeCandle(2030, 2035, 2025, 2032)
	result := engine.Process(c, structure)
	if !result.Ready {
		t.Fatal("Fibonacci should be ready with valid swing pair")
	}
	// 0.618 level = 2050 - 0.618 * 50 = 2050 - 30.9 = 2019.1
	level618 := result.Levels["0.618"]
	expected := decimal.NewFromFloat(2019.1)
	if !level618.Sub(expected).Abs().LessThan(decimal.NewFromFloat(0.1)) {
		v, _ := level618.Float64()
		e, _ := expected.Float64()
		t.Errorf("0.618 level: expected ~%.1f, got %.5f", e, v)
	}
}

func TestFibonacci_NoSwings(t *testing.T) {
	engine := NewFibonacciEngine(nil)
	structure := StructureFeatures{}
	c := makeCandle(2030, 2035, 2025, 2032)
	result := engine.Process(c, structure)
	if result.Ready {
		t.Error("Fibonacci should not be ready without swing highs/lows")
	}
}

func TestFibonacci_InvalidRange(t *testing.T) {
	engine := NewFibonacciEngine(nil)
	// Swing high < swing low → invalid
	structure := StructureFeatures{
		SwingHighs:   []decimal.Decimal{decimal.NewFromFloat(2000.0)},
		SwingLows:    []decimal.Decimal{decimal.NewFromFloat(2050.0)},
		CurrentTrend: "bullish",
	}
	c := makeCandle(2030, 2035, 2025, 2032)
	result := engine.Process(c, structure)
	if result.Ready {
		t.Error("Fibonacci should not be ready when swing high < swing low")
	}
}

// === Pivot Tests (SOW Section 13) ===

func TestPivots_DailyRollover(t *testing.T) {
	engine := NewPivotEngine()
	baseTime := time.Date(2024, 6, 3, 10, 0, 0, 0, time.UTC) // Monday June 3 2024

	// Feed candles for day 1
	for hour := 0; hour < 4; hour++ {
		c := &types.Candle{
			Symbol: "XAUUSD", Timeframe: "M15",
			Time: baseTime.Add(time.Duration(hour) * time.Hour),
			Open: decimal.NewFromFloat(2000 + float64(hour)),
			High: decimal.NewFromFloat(2005 + float64(hour)),
			Low:  decimal.NewFromFloat(1995 + float64(hour)),
			Close: decimal.NewFromFloat(2002 + float64(hour)),
			Volume: 100, IsClosed: true,
		}
		result := engine.Process(c)
		// Day 1 should not have daily pivots yet (no previous completed day)
		if result.Daily.Ready {
			t.Error("daily pivots should not be ready during first day")
		}
	}

	// Feed candles for day 2 — should trigger daily pivot computation
	day2 := baseTime.Add(24 * time.Hour)
	c := &types.Candle{
		Symbol: "XAUUSD", Timeframe: "M15",
		Time: day2,
		Open: decimal.NewFromFloat(2010),
		High: decimal.NewFromFloat(2015),
		Low:  decimal.NewFromFloat(2005),
		Close: decimal.NewFromFloat(2012),
		Volume: 100, IsClosed: true,
	}
	result := engine.Process(c)
	if !result.Daily.Ready {
		t.Error("daily pivots should be ready after day rollover")
	}
	// P = (H + L + C) / 3 = (2008 + 1995 + 2002) / 3 = 6005/3 ≈ 2001.67
	// Using day 1's accumulated: H=2008, L=1995, C=2002
	p, _ := result.Daily.P.Float64()
	if math.Abs(p-2002.67) > 0.5 {
		t.Errorf("expected pivot ~2002.67, got %f", p)
	}
}

func TestPivots_UsesPreviousCompletedPeriod(t *testing.T) {
	engine := NewPivotEngine()
	t1 := time.Date(2024, 6, 3, 10, 0, 0, 0, time.UTC)
	// Day 1: OHLC = 2000, 2010, 1990, 2005
	engine.Process(&types.Candle{
		Symbol: "XAUUSD", Timeframe: "M15", Time: t1,
		Open: decimal.NewFromFloat(2000), High: decimal.NewFromFloat(2010),
		Low: decimal.NewFromFloat(1990), Close: decimal.NewFromFloat(2005),
		Volume: 100, IsClosed: true,
	})
	// Day 2 candle triggers pivot from day 1
	t2 := t1.Add(24 * time.Hour)
	result := engine.Process(&types.Candle{
		Symbol: "XAUUSD", Timeframe: "M15", Time: t2,
		Open: decimal.NewFromFloat(3000), High: decimal.NewFromFloat(3000),
		Low: decimal.NewFromFloat(3000), Close: decimal.NewFromFloat(3000),
		Volume: 100, IsClosed: true,
	})
	// Pivot should be from day 1: P = (2010+1990+2005)/3 = 6005/3 ≈ 2001.67
	p, _ := result.Daily.P.Float64()
	if math.Abs(p-2001.67) > 0.5 {
		t.Errorf("pivot should use day 1 data, expected ~2001.67, got %f", p)
	}
	// NOT from day 2 data (which would be 3000)
	if p > 2500 {
		t.Error("pivot used current day data instead of previous completed day")
	}
}

func TestPivots_NotReadyBeforeRollover(t *testing.T) {
	engine := NewPivotEngine()
	t1 := time.Date(2024, 6, 3, 10, 0, 0, 0, time.UTC)
	for h := 0; h < 10; h++ {
		result := engine.Process(&types.Candle{
			Symbol: "XAUUSD", Timeframe: "M15", Time: t1.Add(time.Duration(h) * time.Hour),
			Open: decimal.NewFromFloat(2000), High: decimal.NewFromFloat(2010),
			Low: decimal.NewFromFloat(1990), Close: decimal.NewFromFloat(2005),
			Volume: 100, IsClosed: true,
		})
		if result.Daily.Ready {
			t.Error("daily pivots should not be ready before any day completes")
		}
	}
}

// === Helper ===

func makeCandle(high, low, open, close float64) *types.Candle {
	return &types.Candle{
		Symbol: "XAUUSD", Timeframe: "M15",
		Time: time.Now().UTC(),
		Open: decimal.NewFromFloat(open),
		High: decimal.NewFromFloat(high),
		Low: decimal.NewFromFloat(low),
		Close: decimal.NewFromFloat(close),
		Volume: 100, IsClosed: true,
	}
}
