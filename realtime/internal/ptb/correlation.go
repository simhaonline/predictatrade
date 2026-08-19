// Package ptb — Gold-specific correlation engine.
// Section 8: DXY, yields, silver, gold/silver ratio correlations.
// All external data marked UNSUPPORTED if not provided by Master Node.
package ptb

import (
	"math"
	"time"
)

// CorrelationEngine computes rolling correlations from synchronized real observations.
// Section 8: Do NOT assume DXY is always inversely correlated — measure it.
type CorrelationEngine struct {
	// Rolling buffers for price observations
	goldHistory   []float64
	dxyHistory    []float64
	silverHistory []float64
	yieldHistory  []float64

	// Timestamps for alignment
	timestamps []time.Time

	// Availability
	dxyAvailable    bool
	silverAvailable bool
	yieldAvailable  bool

	// Freshness
	lastDXYUpdate    time.Time
	lastSilverUpdate time.Time
	lastYieldUpdate  time.Time
}

// NewCorrelationEngine creates a correlation engine.
// All external feeds default to UNAVAILABLE unless explicitly populated.
func NewCorrelationEngine() *CorrelationEngine {
	return &CorrelationEngine{
		dxyAvailable:    false,
		silverAvailable: false,
		yieldAvailable:  false,
	}
}

// AddGoldObservation adds a gold price observation.
func (c *CorrelationEngine) AddGoldObservation(price float64, ts time.Time) {
	c.goldHistory = append(c.goldHistory, price)
	c.timestamps = append(c.timestamps, ts)
	if len(c.goldHistory) > 200 {
		c.goldHistory = c.goldHistory[len(c.goldHistory)-200:]
		c.timestamps = c.timestamps[len(c.timestamps)-200:]
	}
}

// AddDXYObservation adds a DXY observation. Marks DXY as available.
func (c *CorrelationEngine) AddDXYObservation(value float64, ts time.Time) {
	c.dxyHistory = append(c.dxyHistory, value)
	c.lastDXYUpdate = ts
	c.dxyAvailable = true
	if len(c.dxyHistory) > 200 {
		c.dxyHistory = c.dxyHistory[len(c.dxyHistory)-200:]
	}
}

// AddSilverObservation adds a silver price observation.
func (c *CorrelationEngine) AddSilverObservation(price float64, ts time.Time) {
	c.silverHistory = append(c.silverHistory, price)
	c.lastSilverUpdate = ts
	c.silverAvailable = true
	if len(c.silverHistory) > 200 {
		c.silverHistory = c.silverHistory[len(c.silverHistory)-200:]
	}
}

// AddYieldObservation adds a yield observation.
func (c *CorrelationEngine) AddYieldObservation(value float64, ts time.Time) {
	c.yieldHistory = append(c.yieldHistory, value)
	c.lastYieldUpdate = ts
	c.yieldAvailable = true
	if len(c.yieldHistory) > 200 {
		c.yieldHistory = c.yieldHistory[len(c.yieldHistory)-200:]
	}
}

// ComputeDXYCorrelation calculates rolling correlation between gold and DXY.
// Returns UNSUPPORTED if DXY data is not available.
// Section 8: Do NOT assume inverse correlation — measure it.
// Section 39F: Safely handle NaN, constant series, insufficient samples.
func (c *CorrelationEngine) ComputeDXYCorrelation(window int) CorrelationResult {
	if !c.dxyAvailable || len(c.goldHistory) < window || len(c.dxyHistory) < window {
		return CorrelationResult{
			Instrument:   "DXY",
			Window:       window,
			Quality:      "UNAVAILABLE",
			Availability: "NO_DXY_FEED",
		}
	}

	// Align by taking the most recent N observations from each
	goldSeries := c.goldHistory[len(c.goldHistory)-window:]
	dxySeries := c.dxyHistory[len(c.dxyHistory)-window:]

	corr := pearsonCorrelation(goldSeries, dxySeries)

	if math.IsNaN(corr) {
		return CorrelationResult{
			Instrument: "DXY",
			Window:     window,
			Quality:    "INVALID",
			Availability: "NaN_CORRELATION",
		}
	}

	direction := "NEUTRAL"
	if corr < -0.3 {
		direction = "INVERSE"
	} else if corr > 0.3 {
		direction = "DIRECT"
	}

	strength := math.Abs(corr)

	// Check freshness
	age := time.Since(c.lastDXYUpdate).Milliseconds()
	quality := "OK"
	if age > 300000 { // >5 min
		quality = "STALE"
	}

	return CorrelationResult{
		Instrument:  "DXY",
		Window:      window,
		Correlation: corr,
		Direction:   direction,
		Strength:    strength,
		SampleCount: window,
		Timestamp:   time.Now().UTC(),
		DataAge:     age,
		Quality:     quality,
		Availability: "AVAILABLE",
	}
}

// ComputeSilverCorrelation calculates gold/silver correlation.
func (c *CorrelationEngine) ComputeSilverCorrelation(window int) CorrelationResult {
	if !c.silverAvailable || len(c.goldHistory) < window || len(c.silverHistory) < window {
		return CorrelationResult{
			Instrument:   "SILVER",
			Window:       window,
			Quality:      "UNAVAILABLE",
			Availability: "NO_SILVER_FEED",
		}
	}

	goldSeries := c.goldHistory[len(c.goldHistory)-window:]
	silverSeries := c.silverHistory[len(c.silverHistory)-window:]

	corr := pearsonCorrelation(goldSeries, silverSeries)

	if math.IsNaN(corr) {
		return CorrelationResult{
			Instrument: "SILVER",
			Window:     window,
			Quality:    "INVALID",
			Availability: "NaN_CORRELATION",
		}
	}

	direction := "NEUTRAL"
	if corr > 0.3 {
		direction = "DIRECT"
	} else if corr < -0.3 {
		direction = "INVERSE"
	}

	return CorrelationResult{
		Instrument:  "SILVER",
		Window:      window,
		Correlation: corr,
		Direction:   direction,
		Strength:    math.Abs(corr),
		SampleCount: window,
		Timestamp:   time.Now().UTC(),
		Quality:     "OK",
		Availability: "AVAILABLE",
	}
}

// ComputeYieldCorrelation calculates gold/yield correlation.
func (c *CorrelationEngine) ComputeYieldCorrelation(window int) CorrelationResult {
	if !c.yieldAvailable || len(c.goldHistory) < window || len(c.yieldHistory) < window {
		return CorrelationResult{
			Instrument:   "US10Y",
			Window:       window,
			Quality:      "UNAVAILABLE",
			Availability: "NO_YIELD_FEED",
		}
	}

	goldSeries := c.goldHistory[len(c.goldHistory)-window:]
	yieldSeries := c.yieldHistory[len(c.yieldHistory)-window:]

	corr := pearsonCorrelation(goldSeries, yieldSeries)

	if math.IsNaN(corr) {
		return CorrelationResult{
			Instrument: "US10Y",
			Window:     window,
			Quality:    "INVALID",
			Availability: "NaN_CORRELATION",
		}
	}

	direction := "NEUTRAL"
	if corr < -0.3 {
		direction = "INVERSE"
	} else if corr > 0.3 {
		direction = "DIRECT"
	}

	return CorrelationResult{
		Instrument:  "US10Y",
		Window:      window,
		Correlation: corr,
		Direction:   direction,
		Strength:    math.Abs(corr),
		SampleCount: window,
		Timestamp:   time.Now().UTC(),
		Quality:     "OK",
		Availability: "AVAILABLE",
	}
}

// GoldSilverRatio computes the current gold/silver ratio.
func (c *CorrelationEngine) GoldSilverRatio() (float64, bool) {
	if !c.silverAvailable || len(c.goldHistory) == 0 || len(c.silverHistory) == 0 {
		return 0, false
	}
	gold := c.goldHistory[len(c.goldHistory)-1]
	silver := c.silverHistory[len(c.silverHistory)-1]
	if silver == 0 {
		return 0, false
	}
	return gold / silver, true
}

// AllCorrelations returns all available correlation results.
func (c *CorrelationEngine) AllCorrelations(shortWindow, mediumWindow, longWindow int) []CorrelationResult {
	results := make([]CorrelationResult, 0)
	results = append(results, c.ComputeDXYCorrelation(shortWindow))
	results = append(results, c.ComputeDXYCorrelation(mediumWindow))
	results = append(results, c.ComputeDXYCorrelation(longWindow))
	results = append(results, c.ComputeSilverCorrelation(mediumWindow))
	results = append(results, c.ComputeYieldCorrelation(mediumWindow))
	return results
}

// pearsonCorrelation computes the Pearson correlation coefficient.
// Section 39F: Safely handles NaN, constant series, insufficient samples.
func pearsonCorrelation(x, y []float64) float64 {
	n := len(x)
	if n != len(y) || n < 2 {
		return 0 // insufficient samples
	}

	sumX := 0.0
	sumY := 0.0
	for i := 0; i < n; i++ {
		sumX += x[i]
		sumY += y[i]
	}
	meanX := sumX / float64(n)
	meanY := sumY / float64(n)

	var numerator, denomX, denomY float64
	for i := 0; i < n; i++ {
		dx := x[i] - meanX
		dy := y[i] - meanY
		numerator += dx * dy
		denomX += dx * dx
		denomY += dy * dy
	}

	if denomX == 0 || denomY == 0 {
		return 0 // constant series — correlation undefined
	}

	return numerator / math.Sqrt(denomX*denomY)
}
