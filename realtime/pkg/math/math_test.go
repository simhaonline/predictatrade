package patmath

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestGrossRR(t *testing.T) {
	entry := decimal.NewFromFloat(2430.00)
	sl := decimal.NewFromFloat(2426.00)
	tp1 := decimal.NewFromFloat(2435.00)

	rr := GrossRR(entry, sl, tp1)
	// (2435-2430) / (2430-2426) = 5/4 = 1.25
	if !rr.Equal(decimal.NewFromFloat(1.25)) {
		t.Errorf("GrossRR = %s, want 1.25", rr.String())
	}
}

func TestGrossRR_ZeroStop(t *testing.T) {
	entry := decimal.NewFromFloat(2430.00)
	sl := decimal.NewFromFloat(2430.00)
	tp1 := decimal.NewFromFloat(2435.00)

	rr := GrossRR(entry, sl, tp1)
	if !rr.IsZero() {
		t.Errorf("GrossRR with zero stop = %s, want 0", rr.String())
	}
}

func TestNetRR(t *testing.T) {
	entry := decimal.NewFromFloat(2430.00)
	sl := decimal.NewFromFloat(2426.00)
	tp1 := decimal.NewFromFloat(2435.00)
	cost := decimal.NewFromFloat(0.50)

	rr := NetRR(entry, sl, tp1, cost)
	// (5 - 0.5) / (4 + 0.5) = 4.5 / 4.5 = 1.0
	if !rr.Equal(decimal.NewFromFloat(1.0)) {
		t.Errorf("NetRR = %s, want 1.0", rr.String())
	}
}

func TestExpectancy(t *testing.T) {
	pWin := decimal.NewFromFloat(0.55)
	avgWinR := decimal.NewFromFloat(1.5)
	avgLossR := decimal.NewFromFloat(1.0)

	e := Expectancy(pWin, avgWinR, avgLossR)
	// 0.55*1.5 - 0.45*1.0 = 0.825 - 0.45 = 0.375
	expected := decimal.NewFromFloat(0.375)
	if !e.Equal(expected) {
		t.Errorf("Expectancy = %s, want %s", e.String(), expected.String())
	}
}

func TestWilsonInterval(t *testing.T) {
	// 98/100 with z=1.96
	lower := WilsonLower(98, 100, 1.96)
	upper := WilsonUpper(98, 100, 1.96)

	// The Wilson interval should be wider than [0.97, 0.99]
	if lower > 0.97 {
		t.Errorf("WilsonLower(98,100) = %f, should be < 0.97", lower)
	}
	if upper < 0.99 {
		t.Errorf("WilsonUpper(98,100) = %f, should be > 0.99", upper)
	}
}

func TestWilsonInterval_ZeroSuccess(t *testing.T) {
	lower := WilsonLower(0, 100, 1.96)
	if lower != 0 {
		t.Errorf("WilsonLower(0,100) = %f, want 0", lower)
	}
}

func TestWilsonInterval_AllSuccess(t *testing.T) {
	upper := WilsonUpper(100, 100, 1.96)
	if upper < 0.999 || upper > 1.001 {
		t.Errorf("WilsonUpper(100,100) = %f, want 1", upper)
	}
}

func TestBrierScore(t *testing.T) {
	probs := []float64{0.9, 0.1, 0.8, 0.3}
	outcomes := []bool{true, false, true, false}

	brier := BrierScore(probs, outcomes)
	// (0.1^2 + 0.1^2 + 0.2^2 + 0.3^2) / 4 = (0.01+0.01+0.04+0.09)/4 = 0.0375
	if brier < 0.03 || brier > 0.04 {
		t.Errorf("BrierScore = %f, want ~0.0375", brier)
	}
}

func TestECE(t *testing.T) {
	binCounts := []int{10, 10, 10, 10}
	binMeanForecasts := []float64{0.1, 0.3, 0.5, 0.7}
	binObservedFreqs := []float64{0.1, 0.3, 0.5, 0.7}

	ece := ECE(binCounts, binMeanForecasts, binObservedFreqs)
	// Perfectly calibrated → ECE = 0
	if ece > 0.001 {
		t.Errorf("ECE (perfect) = %f, want 0", ece)
	}
}

func TestMTFAlignmentScore(t *testing.T) {
	weights := []float64{0.3, 0.3, 0.2, 0.2}
	states := []int{1, 1, 1, -1}

	score := MTFAlignmentScore(weights, states)
	// (0.3*1 + 0.3*1 + 0.2*1 + 0.2*(-1)) / (0.3+0.3+0.2+0.2) * 100
	// = (0.3+0.3+0.2-0.2) / 1.0 * 100 = 0.6 * 100 = 60
	if score < 59 || score > 61 {
		t.Errorf("MTFAlignmentScore = %f, want ~60", score)
	}
}

func TestATR(t *testing.T) {
	highs := []decimal.Decimal{
		decimal.NewFromFloat(2430), decimal.NewFromFloat(2432),
		decimal.NewFromFloat(2435), decimal.NewFromFloat(2431),
		decimal.NewFromFloat(2433),
	}
	lows := []decimal.Decimal{
		decimal.NewFromFloat(2428), decimal.NewFromFloat(2429),
		decimal.NewFromFloat(2430), decimal.NewFromFloat(2427),
		decimal.NewFromFloat(2430),
	}
	closes := []decimal.Decimal{
		decimal.NewFromFloat(2429), decimal.NewFromFloat(2431),
		decimal.NewFromFloat(2432), decimal.NewFromFloat(2428),
		decimal.NewFromFloat(2431),
	}

	atr := ATR(highs, lows, closes, 4)
	if atr.IsZero() {
		t.Error("ATR should not be zero")
	}
}

func TestRSI(t *testing.T) {
	closes := make([]decimal.Decimal, 20)
	for i := 0; i < 20; i++ {
		closes[i] = decimal.NewFromFloat(2430 + float64(i)*0.5)
	}

	rsi := RSI(closes, 14)
	// All closes rising → RSI should be near 100
	if rsi.LessThan(decimal.NewFromInt(90)) {
		t.Errorf("RSI for rising series = %s, want > 90", rsi.String())
	}
}

func TestCostToTarget(t *testing.T) {
	entry := decimal.NewFromFloat(2430.00)
	tp1 := decimal.NewFromFloat(2435.00)
	cost := decimal.NewFromFloat(1.00)

	ratio := CostToTarget(entry, tp1, cost)
	// 1.0 / 5.0 = 0.20
	if !ratio.Equal(decimal.NewFromFloat(0.20)) {
		t.Errorf("CostToTarget = %s, want 0.20", ratio.String())
	}
}
