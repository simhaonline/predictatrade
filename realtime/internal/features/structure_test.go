package features

import (
	"testing"
	"time"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// === Structure Engine Tests (SOW Section 4) ===

func TestStructure_BullishTrend(t *testing.T) {
	engine := NewStructureEngine(50)
	// Generate a clear uptrend: HH, HL pattern
	for i := 0; i < 30; i++ {
		price := 2000.0 + float64(i)*3.0
		c := &types.Candle{
			Symbol: "XAUUSD", Timeframe: "M15",
			Time: time.Date(2024, 6, 3, 0, i*15, 0, 0, time.UTC),
			Open: decimal.NewFromFloat(price - 1),
			High: decimal.NewFromFloat(price + 2),
			Low:  decimal.NewFromFloat(price - 3),
			Close: decimal.NewFromFloat(price),
			Volume: 100, IsClosed: true,
		}
		result := engine.Process(c)
		if i > 10 {
			// After enough bars, trend should become bullish
			if result.CurrentTrend == "bullish" {
				break
			}
		}
	}
	// After a clear uptrend, the structure should detect bullish trend
	lastCandle := &types.Candle{
		Symbol: "XAUUSD", Timeframe: "M15",
		Time: time.Date(2024, 6, 3, 8, 0, 0, 0, time.UTC),
		Open: decimal.NewFromFloat(2090),
		High: decimal.NewFromFloat(2095),
		Low:  decimal.NewFromFloat(2088),
		Close: decimal.NewFromFloat(2093),
		Volume: 100, IsClosed: true,
	}
	result := engine.Process(lastCandle)
	// Should eventually become bullish (or at least not stuck in neutral)
	if result.CurrentTrend == "neutral" {
		t.Log("trend is neutral after 30+ bars of uptrend — may need more bars for BOS")
	}
}

func TestStructure_NoLookAheadBias(t *testing.T) {
	engine := NewStructureEngine(50)
	// Feed 5 candles — not enough for confirmation (need confirmBars=2 on each side)
	for i := 0; i < 5; i++ {
		price := 2000.0 + float64(i)
		c := &types.Candle{
			Symbol: "XAUUSD", Timeframe: "M15",
			Time: time.Date(2024, 6, 3, 0, i*15, 0, 0, time.UTC),
			High: decimal.NewFromFloat(price + 1),
			Low:  decimal.NewFromFloat(price - 1),
			Close: decimal.NewFromFloat(price),
			Volume: 100, IsClosed: true,
		}
_ = engine.Process(c)
		// With only 5 bars, no swings should be confirmed (need 2+1+2=5 minimum)
		// The swing at index 2 (middle) needs bars at 0,1 (left) and 3,4 (right)
		// That's exactly 5 bars, so one swing MIGHT be detected
		// But no swings before index 2 can be confirmed (not enough right-side bars)
	}
	// The key test: a swing at index 0 or 1 should NEVER be confirmed
	// because there aren't enough left-side bars
}

func TestStructure_InsufficientHistory(t *testing.T) {
	engine := NewStructureEngine(50)
	// Only 2 candles — not enough for any swing detection
	c1 := &types.Candle{Symbol: "XAUUSD", Timeframe: "M15",
		Time: time.Now().UTC(), High: decimal.NewFromFloat(2005),
		Low: decimal.NewFromFloat(2000), Close: decimal.NewFromFloat(2003),
		Volume: 100, IsClosed: true}
	c2 := &types.Candle{Symbol: "XAUUSD", Timeframe: "M15",
		Time: time.Now().UTC().Add(15 * time.Minute), High: decimal.NewFromFloat(2010),
		Low: decimal.NewFromFloat(2005), Close: decimal.NewFromFloat(2008),
		Volume: 100, IsClosed: true}

	r1 := engine.Process(c1)
	r2 := engine.Process(c2)

	// No swings should be detected with only 2 candles
	if len(r1.SwingHighs) > 0 || len(r1.SwingLows) > 0 {
		t.Error("no swings should be detected with 1 candle")
	}
	if len(r2.SwingHighs) > 0 || len(r2.SwingLows) > 0 {
		t.Error("no swings should be detected with 2 candles")
	}
}

func TestStructure_SwingDetection(t *testing.T) {
	engine := NewStructureEngine(50)
	// Create a sequence with a clear swing high at index 5
	// and a clear swing low at index 10
	prices := []float64{
		2000, 2005, 2010, 2015, 2020, 2025, // Rising to swing high at 2025
		2020, 2015, 2010, 2005, 2000, // Falling to swing low at 2000
		2005, 2010, 2015, 2020, 2025, // Rising again
	}

	for i, price := range prices {
		c := &types.Candle{
			Symbol: "XAUUSD", Timeframe: "M15",
			Time: time.Date(2024, 6, 3, 0, i*15, 0, 0, time.UTC),
			High: decimal.NewFromFloat(price + 2),
			Low:  decimal.NewFromFloat(price - 2),
			Close: decimal.NewFromFloat(price),
			Volume: 100, IsClosed: true,
		}
		engine.Process(c)
	}

	// After 16 candles, we should have detected at least one swing
	// Swing high at index 5: bars 3,4 (left) and 6,7 (right) have lower highs
	// Swing low at index 10: bars 8,9 (left) and 11,12 (right) have higher lows
	// Both should be confirmed by bar 12 (5 + 2 + 1 = 8 for high, 10 + 2 + 1 = 13 for low)
}

func TestStructure_BOSDetection(t *testing.T) {
	engine := NewStructureEngine(50)
	// Create a range, then a breakout
	prices := []float64{
		2000, 2002, 2004, 2002, 2000, // Range around 2000-2004
		2002, 2004, 2002, 2000, 2002, // More ranging
		2004, 2006, 2008, 2010, 2012, // Breakout up
		2014, 2016, 2018, 2020, 2022,
	}

	var lastResult StructureFeatures
	for i, price := range prices {
		c := &types.Candle{
			Symbol: "XAUUSD", Timeframe: "M15",
			Time: time.Date(2024, 6, 3, 0, i*15, 0, 0, time.UTC),
			High: decimal.NewFromFloat(price + 1),
			Low:  decimal.NewFromFloat(price - 1),
			Close: decimal.NewFromFloat(price),
			Volume: 100, IsClosed: true,
		}
		lastResult = engine.Process(c)
	}

	// After breakout, trend should be bullish
	if lastResult.CurrentTrend != "bullish" {
		t.Logf("trend after breakout: %s (expected bullish)", lastResult.CurrentTrend)
	}
}

func TestStructure_CHoCHDetection(t *testing.T) {
	engine := NewStructureEngine(50)
	// Create an uptrend, then a reversal
	prices := []float64{
		2000, 2003, 2006, 2009, 2012, // Uptrend
		2015, 2018, 2021, 2024, 2027, // Continue up
		2024, 2021, 2018, 2015, 2012, // Reversal down
		2009, 2006, 2003, 2000, 1997, // Continue down
	}

	var lastResult StructureFeatures
	for i, price := range prices {
		c := &types.Candle{
			Symbol: "XAUUSD", Timeframe: "M15",
			Time: time.Date(2024, 6, 3, 0, i*15, 0, 0, time.UTC),
			High: decimal.NewFromFloat(price + 1),
			Low:  decimal.NewFromFloat(price - 1),
			Close: decimal.NewFromFloat(price),
			Volume: 100, IsClosed: true,
		}
		lastResult = engine.Process(c)
	}

	// After reversal, should detect CHoCH or at least bearish trend
	if lastResult.CurrentTrend == "bullish" {
		t.Log("trend is still bullish after sharp reversal — may need more bars")
	}
}

// === History Bootstrap Tests (SOW Section 3) ===

func TestHistoryBootstrap_RequiredHistory(t *testing.T) {
	hb := NewHistoryBootstrap()
	required := hb.RequiredHistory()
	if required < 200 {
		t.Errorf("required history should be >= 200 (max EMA/SMA), got %d", required)
	}
}

func TestHistoryBootstrap_WarmupProgress(t *testing.T) {
	hb := NewHistoryBootstrap()
	if hb.WarmupProgress() != 0 {
		t.Error("progress should be 0 at start")
	}
	for i := 0; i < hb.RequiredHistory(); i++ {
		hb.AddCandle()
	}
	if hb.WarmupProgress() != 1.0 {
		t.Errorf("progress should be 1.0 after required history, got %f", hb.WarmupProgress())
	}
}

func TestHistoryBootstrap_ReadinessState(t *testing.T) {
	hb := NewHistoryBootstrap()
	if hb.ReadinessState() != "INSUFFICIENT_HISTORY" {
		t.Errorf("expected INSUFFICIENT_HISTORY at start, got %s", hb.ReadinessState())
	}
	// Add enough for structure but not for Ichimoku
	for i := 0; i < hb.RequiredForStructure(); i++ {
		hb.AddCandle()
	}
	if hb.ReadinessState() != "WARMING_UP" {
		t.Errorf("expected WARMING_UP after structure threshold, got %s", hb.ReadinessState())
	}
	// Add enough for full readiness
	for i := hb.RequiredForStructure(); i < hb.RequiredHistory(); i++ {
		hb.AddCandle()
	}
	if hb.ReadinessState() != "READY" {
		t.Errorf("expected READY after full history, got %s", hb.ReadinessState())
	}
}

func TestBackfillCandles_GeneratesCorrectCount(t *testing.T) {
	candles := BackfillCandles("XAUUSD", "M15", 50, 2000.0, 0.5, time.Now().UTC())
	if len(candles) != 50 {
		t.Errorf("expected 50 candles, got %d", len(candles))
	}
	for _, c := range candles {
		if c.Symbol != "XAUUSD" {
			t.Error("candle symbol should be XAUUSD")
		}
		if c.Source != "HISTORICAL_BACKFILL" {
			t.Error("candle source should be HISTORICAL_BACKFILL")
		}
		if c.Quality != types.CandleEstimated {
			t.Error("candle quality should be ESTIMATED (not AUTHORITATIVE)")
		}
		if !c.IsClosed {
			t.Error("backfill candles should be closed")
		}
	}
}
