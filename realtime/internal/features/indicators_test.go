package features

import (
	"math"
	"testing"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// === Indicator Reference Tests ===
// Deterministic tests verifying mathematical correctness of indicator calculations.
// Each test uses controlled candle sequences with independently calculated expected values.

func makeCandles(prices []float64) []*types.Candle {
	candles := make([]*types.Candle, len(prices))
	for i, p := range prices {
		candles[i] = &types.Candle{
			Symbol:   "XAUUSD",
			Open:     decimal.NewFromFloat(p),
			High:     decimal.NewFromFloat(p + 1.0),
			Low:      decimal.NewFromFloat(p - 1.0),
			Close:    decimal.NewFromFloat(p),
			Volume:   100,
			Timeframe: types.TFM1,
		}
	}
	return candles
}

// SMA tests
func TestSMA50(t *testing.T) {
	prices := make([]float64, 60)
	for i := range prices {
		prices[i] = 4400.0 + float64(i)*0.5
	}
	candles := makeCandles(prices)
	engine := NewIndicatorEngine(200)
	var feat IndicatorFeatures
	for _, c := range candles {
		feat = engine.Process(c)
	}
	// SMA50 of last 50 values: avg(4400+0.5*10 ... 4400+0.5*59)
	// = 4400 + 0.5 * avg(10..59) = 4400 + 0.5 * 34.5 = 4417.25
	expected := 4400.0 + 0.5*(10+59)/2.0
	actual, _ := feat.SMA50.Float64()
	if math.Abs(actual-expected) > 0.01 {
		t.Errorf("SMA50: expected %.2f, got %.2f", expected, actual)
	}
}

func TestSMA200_Warmup(t *testing.T) {
	// With fewer than 200 candles, SMA200 should be zero
	prices := make([]float64, 100)
	candles := makeCandles(prices)
	engine := NewIndicatorEngine(200)
	var feat IndicatorFeatures
	for _, c := range candles {
		feat = engine.Process(c)
	}
	if !feat.SMA200.IsZero() {
		t.Error("SMA200 should be zero with insufficient warmup")
	}
}

// EMA tests
func TestEMA9(t *testing.T) {
	prices := []float64{100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110}
	candles := makeCandles(prices)
	engine := NewIndicatorEngine(200)
	var feat IndicatorFeatures
	for _, c := range candles {
		feat = engine.Process(c)
	}
	// EMA9 should be between min and max of close prices
	val, _ := feat.EMA9.Float64()
	if val < 100 || val > 110 {
		t.Errorf("EMA9 should be within range [100, 110], got %f", val)
	}
}

// RSI property tests
func TestRSI_Bounded(t *testing.T) {
	prices := make([]float64, 30)
	for i := range prices {
		prices[i] = 4400 + float64(i%7-3)*2 // Oscillating
	}
	candles := makeCandles(prices)
	engine := NewIndicatorEngine(200)
	var feat IndicatorFeatures
	for _, c := range candles {
		feat = engine.Process(c)
	}
	rsi, _ := feat.RSI.Float64()
	if rsi < 0 || rsi > 100 {
		t.Errorf("RSI must be in [0, 100], got %f", rsi)
	}
}

func TestRSI_AllBullish(t *testing.T) {
	prices := make([]float64, 30)
	for i := range prices {
		prices[i] = 4400 + float64(i) // Always rising
	}
	candles := makeCandles(prices)
	engine := NewIndicatorEngine(200)
	var feat IndicatorFeatures
	for _, c := range candles {
		feat = engine.Process(c)
	}
	rsi, _ := feat.RSI.Float64()
	if rsi < 70 {
		t.Errorf("RSI should be high (>70) for all-bullish, got %f", rsi)
	}
}

func TestRSI_AllBearish(t *testing.T) {
	prices := make([]float64, 30)
	for i := range prices {
		prices[i] = 4400 - float64(i) // Always falling
	}
	candles := makeCandles(prices)
	engine := NewIndicatorEngine(200)
	var feat IndicatorFeatures
	for _, c := range candles {
		feat = engine.Process(c)
	}
	rsi, _ := feat.RSI.Float64()
	if rsi > 30 {
		t.Errorf("RSI should be low (<30) for all-bearish, got %f", rsi)
	}
}

// ATR property tests
func TestATR_NonNegative(t *testing.T) {
	prices := make([]float64, 20)
	for i := range prices {
		prices[i] = 4400 + float64(i%5)*2
	}
	candles := makeCandles(prices)
	engine := NewIndicatorEngine(200)
	var feat IndicatorFeatures
	for _, c := range candles {
		feat = engine.Process(c)
	}
	atr, _ := feat.ATR.Float64()
	if atr < 0 {
		t.Errorf("ATR must be >= 0, got %f", atr)
	}
}

// Bollinger Bands property test
func TestBollingerBands_Ordering(t *testing.T) {
	prices := make([]float64, 25)
	for i := range prices {
		prices[i] = 4400 + float64(i%10)*0.5
	}
	candles := makeCandles(prices)
	engine := NewIndicatorEngine(200)
	var feat IndicatorFeatures
	for _, c := range candles {
		feat = engine.Process(c)
	}
	if !feat.BollUpper.GreaterThan(feat.BollMiddle) {
		t.Error("BB Upper must be > BB Middle")
	}
	if !feat.BollMiddle.GreaterThan(feat.BollLower) {
		t.Error("BB Middle must be > BB Lower")
	}
}

// Bollinger Band Width
func TestBBWidth_NonNegative(t *testing.T) {
	prices := make([]float64, 25)
	for i := range prices {
		prices[i] = 4400 + float64(i%10)
	}
	candles := makeCandles(prices)
	engine := NewIndicatorEngine(200)
	var feat IndicatorFeatures
	for _, c := range candles {
		feat = engine.Process(c)
	}
	bw, _ := feat.BollWidth.Float64()
	if bw < 0 {
		t.Errorf("BB Width must be >= 0, got %f", bw)
	}
}

// ADX property test
func TestADX_NonNegative(t *testing.T) {
	prices := make([]float64, 30)
	for i := range prices {
		prices[i] = 4400 + float64(i) // Strong trend
	}
	candles := makeCandles(prices)
	engine := NewIndicatorEngine(200)
	var feat IndicatorFeatures
	for _, c := range candles {
		feat = engine.Process(c)
	}
	adx, _ := feat.ADX.Float64()
	if adx < 0 {
		t.Errorf("ADX must be >= 0, got %f", adx)
	}
}

// Stochastic property test
func TestStochastic_Bounded(t *testing.T) {
	prices := make([]float64, 20)
	for i := range prices {
		prices[i] = 4400 + float64(i%5)*3
	}
	candles := makeCandles(prices)
	engine := NewIndicatorEngine(200)
	var feat IndicatorFeatures
	for _, c := range candles {
		feat = engine.Process(c)
	}
	stoch, _ := feat.StochMain.Float64()
	if stoch < 0 || stoch > 100 {
		t.Errorf("Stochastic must be in [0, 100], got %f", stoch)
	}
}

// OBV test
func TestOBV_RisingPrices(t *testing.T) {
	prices := []float64{100, 101, 102, 103, 104}
	candles := makeCandles(prices)
	engine := NewIndicatorEngine(200)
	var feat IndicatorFeatures
	for _, c := range candles {
		feat = engine.Process(c)
	}
	obv, _ := feat.OBV.Float64()
	// 4 rising candles * 100 volume each = 400
	if obv != 400 {
		t.Errorf("OBV for all-rising should be 400, got %f", obv)
	}
}

// CCI test
func TestCCI_StrongTrend(t *testing.T) {
	prices := make([]float64, 25)
	for i := range prices {
		prices[i] = 4400 + float64(i)*2 // Strong uptrend
	}
	candles := makeCandles(prices)
	engine := NewIndicatorEngine(200)
	var feat IndicatorFeatures
	for _, c := range candles {
		feat = engine.Process(c)
	}
	cci, _ := feat.CCI.Float64()
	// Strong uptrend should produce positive CCI
	if cci <= 0 {
		t.Errorf("CCI should be positive for strong uptrend, got %f", cci)
	}
}

// Numerical safety tests
func TestInvalidCandle_Rejected(t *testing.T) {
	// High < Low — invalid candle
	candle := &types.Candle{
		Open:  decimal.NewFromFloat(100),
		High:  decimal.NewFromFloat(99), // Invalid: High < Open
		Low:   decimal.NewFromFloat(101),
		Close: decimal.NewFromFloat(100),
	}
	engine := NewIndicatorEngine(200)
	feat := engine.Process(candle)
	// Should return zero features (UNAVAILABLE, not neutral)
	if !feat.ATR.IsZero() {
		t.Error("Invalid candle should produce zero ATR")
	}
}

func TestZeroVolume_NotFabricated(t *testing.T) {
	candles := make([]*types.Candle, 5)
	for i := range candles {
		candles[i] = &types.Candle{
			Open:  decimal.NewFromFloat(100 + float64(i)),
			High:  decimal.NewFromFloat(101 + float64(i)),
			Low:   decimal.NewFromFloat(99 + float64(i)),
			Close: decimal.NewFromFloat(100 + float64(i)),
			Volume: 0, // Zero volume
		}
	}
	engine := NewIndicatorEngine(200)
	var feat IndicatorFeatures
	for _, c := range candles {
		feat = engine.Process(c)
	}
	// OBV with zero volume should be zero
	if !feat.OBV.IsZero() {
		t.Error("OBV should be zero with zero volume")
	}
}

// EMA Cross detection test
func TestEMACross_NotAlignment(t *testing.T) {
	// EMA9 > EMA21 from the start — should NOT be a cross event
	prices := make([]float64, 30)
	for i := range prices {
		prices[i] = 4400 + float64(i) * 0.5 // Steady uptrend
	}
	candles := makeCandles(prices)
	engine := NewIndicatorEngine(200)
	var feat IndicatorFeatures
	for _, c := range candles {
		feat = engine.Process(c)
	}
	// EMACross921 should be false — no actual crossover, just alignment
	if feat.EMACross921 {
		t.Error("EMACross921 should be false when EMA9 has been above EMA21 throughout (no cross event)")
	}
}

// Provenance tests
func TestCVD_UnavailableByDefault(t *testing.T) {
	liq := LiquidityFeatures{}
	if liq.CVDAvailable {
		t.Error("CVD should be UNAVAILABLE by default")
	}
}

func TestDOM_UnavailableByDefault(t *testing.T) {
	liq := LiquidityFeatures{}
	if liq.DOMAvailable {
		t.Error("DOM should be UNAVAILABLE by default")
	}
}

// Metamorphic test: scaling all prices by constant should not change RSI significantly
func TestRSI_ScaleInvariance(t *testing.T) {
	prices1 := []float64{100, 101, 99, 102, 98, 103, 97, 104, 96, 105, 95, 106, 94, 107, 93}
	prices2 := make([]float64, len(prices1))
	for i, p := range prices1 {
		prices2[i] = p * 100 // Scale by 100
	}
	c1 := makeCandles(prices1)
	c2 := makeCandles(prices2)
	e1 := NewIndicatorEngine(200)
	e2 := NewIndicatorEngine(200)
	var f1, f2 IndicatorFeatures
	for i := range c1 {
		f1 = e1.Process(c1[i])
		f2 = e2.Process(c2[i])
	}
	r1, _ := f1.RSI.Float64()
	r2, _ := f2.RSI.Float64()
	if math.Abs(r1-r2) > 0.01 {
		t.Errorf("RSI should be scale-invariant: got %f vs %f", r1, r2)
	}
}
