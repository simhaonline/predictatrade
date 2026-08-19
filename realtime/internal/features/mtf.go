package features

import (
	"github.com/predictatrade/realtime/internal/types"
	"sort"
)

// MTFEngine computes multi-timeframe alignment.
// SOW Section 134.10: mtf_alignment_score
type MTFEngine struct {
	states map[types.Timeframe]int // -1 bearish, 0 neutral, +1 bullish
}

func NewMTFEngine() *MTFEngine {
	return &MTFEngine{states: make(map[types.Timeframe]int)}
}

func (e *MTFEngine) Process(candles map[types.Timeframe]*types.Candle) MTFFeatures {
	feat := MTFFeatures{States: make(map[types.Timeframe]int)}

	weights := map[types.Timeframe]float64{
		types.TFM1: 0.5,
		types.TFM5: 1.0,
		types.TFM15: 1.5,
		types.TFM30: 1.0,
		types.TFH1: 2.0,
		types.TFH4: 1.5,
	}

	for tf, candle := range candles {
		if candle == nil {
			e.states[tf] = 0
			continue
		}
		// Simple: close > open = bullish, close < open = bearish
		if candle.Close.GreaterThan(candle.Open) {
			e.states[tf] = 1
		} else if candle.Close.LessThan(candle.Open) {
			e.states[tf] = -1
		} else {
			e.states[tf] = 0
		}
	}

	// Compute alignment score
	var tfList []types.Timeframe
	for tf := range weights {
		tfList = append(tfList, tf)
	}
	sort.Slice(tfList, func(i, j int) bool { return tfList[i] < tfList[j] })

	var w []float64
	var s []int
	for _, tf := range tfList {
		w = append(w, weights[tf])
		s = append(s, e.states[tf])
		feat.States[tf] = e.states[tf]
	}

	// Use simple weighted score
	wSum := 0.0
	weightedState := 0.0
	for i := range w {
		wSum += w[i]
		weightedState += w[i] * float64(s[i])
	}
	if wSum > 0 {
		feat.Score = 100.0 * weightedState / wSum
	}

	return feat
}
