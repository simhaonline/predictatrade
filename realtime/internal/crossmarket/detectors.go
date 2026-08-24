package crossmarket

import (
	"math"
	"time"
)

// CorrelationDetector tracks rolling correlations between XAUUSD and external drivers.
// It does NOT assume historical correlations remain fixed.
type CorrelationDetector struct {
	window int
	// Rolling price buffers
	goldHistory  []float64
	dxyHistory   []float64
	eurusdHistory []float64
	vixHistory   []float64
	btcHistory   []float64
	// Timestamps
	lastUpdate time.Time
}

func NewCorrelationDetector(window int) *CorrelationDetector {
	if window < 10 {
		window = 50
	}
	return &CorrelationDetector{window: window}
}

func (c *CorrelationDetector) AddGoldPrice(price float64) {
	c.goldHistory = append(c.goldHistory, price)
	if len(c.goldHistory) > c.window {
		c.goldHistory = c.goldHistory[len(c.goldHistory)-c.window:]
	}
}

func (c *CorrelationDetector) AddDXY(value float64) {
	c.dxyHistory = append(c.dxyHistory, value)
	if len(c.dxyHistory) > c.window {
		c.dxyHistory = c.dxyHistory[len(c.dxyHistory)-c.window:]
	}
	c.lastUpdate = time.Now().UTC()
}

func (c *CorrelationDetector) AddEURUSD(value float64) {
	c.eurusdHistory = append(c.eurusdHistory, value)
	if len(c.eurusdHistory) > c.window {
		c.eurusdHistory = c.eurusdHistory[len(c.eurusdHistory)-c.window:]
	}
}

func (c *CorrelationDetector) AddVIX(value float64) {
	c.vixHistory = append(c.vixHistory, value)
	if len(c.vixHistory) > c.window {
		c.vixHistory = c.vixHistory[len(c.vixHistory)-c.window:]
	}
}

func (c *CorrelationDetector) AddBTC(value float64) {
	c.btcHistory = append(c.btcHistory, value)
	if len(c.btcHistory) > c.window {
		c.btcHistory = c.btcHistory[len(c.btcHistory)-c.window:]
	}
}

// Classify determines the current correlation regime.
func (c *CorrelationDetector) Classify() CorrelationRegime {
	// Need enough data for meaningful correlation
	if len(c.goldHistory) < 20 || len(c.dxyHistory) < 20 {
		return CorrInsufficient
	}

	n := min(len(c.goldHistory), len(c.dxyHistory))
	if n < 20 {
		return CorrInsufficient
	}

	// Use the last n observations
	gold := c.goldHistory[len(c.goldHistory)-n:]
	dxy := c.dxyHistory[len(c.dxyHistory)-n:]

	corr := pearsonCorrelation(gold, dxy)

	// Classify based on correlation value and stability
	if math.IsNaN(corr) {
		return CorrInsufficient
	}

	absCorr := math.Abs(corr)
	if absCorr < 0.2 {
		return CorrWeak
	}
	if corr < -0.5 {
		return CorrInverse
	}
	if absCorr < 0.3 {
		return CorrBreakdown
	}
	// Check for regime shift by comparing recent vs older correlation
	if n >= 40 {
		recentCorr := pearsonCorrelation(gold[n/2:], dxy[n/2:])
		olderCorr := pearsonCorrelation(gold[:n/2], dxy[:n/2])
		if !math.IsNaN(recentCorr) && !math.IsNaN(olderCorr) {
			if math.Abs(recentCorr-olderCorr) > 0.4 {
				return CorrRegimeShift
			}
		}
	}
	return CorrNormal
}

// SafeHavenDetector classifies the current safe-haven regime.
// This prevents incorrect "DXY up = gold must fall" logic during stress.
type SafeHavenDetector struct{}

func NewSafeHavenDetector() *SafeHavenDetector {
	return &SafeHavenDetector{}
}

// Classify determines the safe-haven regime from driver snapshots.
func (s *SafeHavenDetector) Classify(drivers []DriverSnapshot) SafeHavenRegime {
	if len(drivers) == 0 {
		return SHUnknown
	}

	// Extract key driver states
	var dxyDir Direction
	vixHigh := false
	dxyImpact := 0.0

	for _, d := range drivers {
		switch d.Name {
		case DriverDXY:
			dxyDir = d.Direction
			dxyImpact = d.ImpactScore
		case DriverVIX:
			if d.RawValue > 30 {
				vixHigh = true
			}
		}
	}

	// Detect safe-haven regimes
	if vixHigh {
		// High VIX = risk-off
		if dxyDir == DirBullish && dxyImpact > 30 {
			// USD also rising = dual safe haven
			return SHDualSafeHaven
		}
		return SHRiskOff
	}

	// Check for liquidity stress (both gold and USD falling together)
	if dxyDir == DirBearish && dxyImpact < -20 {
		return SHLiquidityStress
	}

	// Normal conditions
	if dxyDir == DirNeutral {
		return SHNORMAL
	}

	return SHMixed
}

// DivergenceDetector identifies conflicts between drivers and the signal direction.
type DivergenceDetector struct{}

func NewDivergenceDetector() *DivergenceDetector {
	return &DivergenceDetector{}
}

// Detect checks for divergences between the signal direction and macro drivers.
func (d *DivergenceDetector) Detect(signalDir Direction, drivers []DriverSnapshot, regime SafeHavenRegime) DivergenceSeverity {
	if signalDir == DirNeutral || len(drivers) == 0 {
		return DivNone
	}

	conflicts := 0
	totalDrivers := 0
	var strongConflicts []string

	for _, drv := range drivers {
		if drv.Direction == DirNeutral {
			continue
		}
		totalDrivers++
		if (signalDir == DirBullish && drv.Direction == DirBearish) ||
			(signalDir == DirBearish && drv.Direction == DirBullish) {
			conflicts++
			if math.Abs(drv.ImpactScore) > 50 {
				strongConflicts = append(strongConflicts, string(drv.Name))
			}
		}
	}

	if totalDrivers == 0 {
		return DivNone
	}

	conflictRatio := float64(conflicts) / float64(totalDrivers)

	// Safe-haven override: during dual safe haven, DXY divergence is expected
	if regime == SHDualSafeHaven || regime == SHSafeHavenGold || regime == SHSafeHavenUSD {
		// Reduce severity — divergence is expected during stress
		conflictRatio *= 0.5
	}

	switch {
	case len(strongConflicts) >= 2:
		return DivExtreme
	case conflictRatio >= 0.7:
		return DivHigh
	case conflictRatio >= 0.4:
		return DivModerate
	case conflictRatio >= 0.2:
		return DivLow
	default:
		return DivNone
	}
}

// pearsonCorrelation computes the Pearson correlation coefficient.
func pearsonCorrelation(x, y []float64) float64 {
	n := len(x)
	if n != len(y) || n < 2 {
		return math.NaN()
	}

	sumX, sumY := 0.0, 0.0
	for i := 0; i < n; i++ {
		sumX += x[i]
		sumY += y[i]
	}
	meanX := sumX / float64(n)
	meanY := sumY / float64(n)

	var num, denomX, denomY float64
	for i := 0; i < n; i++ {
		dx := x[i] - meanX
		dy := y[i] - meanY
		num += dx * dy
		denomX += dx * dx
		denyY := dy * dy
		denomY += denyY
	}

	if denomX == 0 || denomY == 0 {
		return math.NaN()
	}
	return num / math.Sqrt(denomX*denomY)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
