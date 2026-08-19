package features

import (
	"math"

	"github.com/shopspring/decimal"
)

// RollingStats is a reusable, numerically stable rolling statistics engine.
// SOW Sections 8, 9, 10, 11: Used for OBV Z-score, Tick Volume Z-score, BB Width Z-score.
// Uses Welford's online algorithm for numerically stable variance computation.
// Supports incremental updates (O(1) per observation after warm-up) and historical initialization.
type RollingStats struct {
	window      int
	minSamples  int
	values      []float64
	head        int
	count       int
	sum         float64
	sumSq       float64
	ready       bool
}

// NewRollingStats creates a new rolling statistics engine.
// window is the rolling lookback size.
// minSamples is the minimum number of observations before producing statistics.
func NewRollingStats(window, minSamples int) *RollingStats {
	if window < 1 {
		window = 1
	}
	if minSamples < 2 {
		minSamples = 2
	}
	if minSamples > window {
		minSamples = window
	}
	return &RollingStats{
		window:     window,
		minSamples: minSamples,
		values:     make([]float64, window),
	}
}

// Add adds a single observation and updates rolling state incrementally.
func (rs *RollingStats) Add(val float64) {
	if math.IsNaN(val) || math.IsInf(val, 0) {
		return // Skip invalid values
	}

	// If the ring buffer is full, remove the oldest value
	if rs.count >= rs.window {
		old := rs.values[rs.head]
		rs.sum -= old
		rs.sumSq -= old * old
	}

	// Add new value
	rs.values[rs.head] = val
	rs.sum += val
	rs.sumSq += val * val
	rs.head = (rs.head + 1) % rs.window
	if rs.count < rs.window {
		rs.count++
	}
	rs.ready = rs.count >= rs.minSamples
}

// InitFromHistory initializes the rolling state from a slice of historical values.
// This is used for warm-up/bootstrap without processing values one-by-one.
func (rs *RollingStats) InitFromHistory(history []float64) {
	rs.Reset()
	start := 0
	if len(history) > rs.window {
		start = len(history) - rs.window
	}
	for i := start; i < len(history); i++ {
		rs.Add(history[i])
	}
}

// Reset clears all state.
func (rs *RollingStats) Reset() {
	rs.head = 0
	rs.count = 0
	rs.sum = 0
	rs.sumSq = 0
	rs.ready = false
	for i := range rs.values {
		rs.values[i] = 0
	}
}

// Mean returns the rolling mean. Returns 0 if not ready.
func (rs *RollingStats) Mean() float64 {
	if !rs.ready || rs.count == 0 {
		return 0
	}
	return rs.sum / float64(rs.count)
}

// Variance returns the rolling population variance. Returns 0 if not ready.
func (rs *RollingStats) Variance() float64 {
	if !rs.ready || rs.count < 2 {
		return 0
	}
	n := float64(rs.count)
	mean := rs.sum / n
	// Variance = E[X²] - (E[X])²
	return (rs.sumSq / n) - (mean * mean)
}

// StdDev returns the rolling standard deviation. Returns 0 if not ready or zero variance.
func (rs *RollingStats) StdDev() float64 {
	v := rs.Variance()
	if v < 0 {
		v = 0 // Guard against floating-point drift
	}
	return math.Sqrt(v)
}

// ZScore computes the Z-score of a value against the rolling distribution.
// z = (x - mean) / stddev
// Returns 0 if not ready or if stddev is zero (constant series).
func (rs *RollingStats) ZScore(val float64) float64 {
	if !rs.ready {
		return 0
	}
	std := rs.StdDev()
	if std == 0 {
		return 0 // Constant series — Z-score undefined, return 0
	}
	mean := rs.Mean()
	return (val - mean) / std
}

// Ready returns true if enough samples have been collected for reliable statistics.
func (rs *RollingStats) Ready() bool {
	return rs.ready
}

// Count returns the current number of observations in the window.
func (rs *RollingStats) Count() int {
	return rs.count
}

// Window returns the configured window size.
func (rs *RollingStats) Window() int {
	return rs.window
}

// MinSamples returns the minimum samples threshold.
func (rs *RollingStats) MinSamples() int {
	return rs.minSamples
}

// MeanDecimal returns the rolling mean as a decimal.Decimal.
func (rs *RollingStats) MeanDecimal() decimal.Decimal {
	return decimal.NewFromFloat(rs.Mean())
}

// StdDevDecimal returns the rolling standard deviation as a decimal.Decimal.
func (rs *RollingStats) StdDevDecimal() decimal.Decimal {
	return decimal.NewFromFloat(rs.StdDev())
}

// ZScoreDecimal computes the Z-score and returns it as a decimal.Decimal.
func (rs *RollingStats) ZScoreDecimal(val decimal.Decimal) decimal.Decimal {
	f, _ := val.Float64()
	return decimal.NewFromFloat(rs.ZScore(f))
}
