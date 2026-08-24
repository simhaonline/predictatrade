package crossmarket

import (
	"math"
	"sync"
	"time"
)

// Engine is the Cross-Market Confluence Engine.
// It collects normalized driver snapshots from providers, computes a bounded
// confluence score, and produces a score adjustment that can be applied to
// the existing signal scoring — NEVER overriding hard gates.
type Engine struct {
	mu     sync.RWMutex
	cfg    Config
	drivers map[DriverName]DriverSnapshot
	correlation *CorrelationDetector
	safeHaven   *SafeHavenDetector
	divergence  *DivergenceDetector
	modelVersion   string
	weightsVersion string
}

// NewEngine creates a cross-market confluence engine.
func NewEngine(cfg Config) *Engine {
	return &Engine{
		cfg:            cfg,
		drivers:        make(map[DriverName]DriverSnapshot),
		correlation:    NewCorrelationDetector(cfg.CorrelationWindow),
		safeHaven:      NewSafeHavenDetector(),
		divergence:     NewDivergenceDetector(),
		modelVersion:   "1.0.0",
		weightsVersion: "1.0.0",
	}
}

// UpdateDriver accepts a new normalized driver snapshot from a provider.
// This is called by background refresh loops — NOT by the signal hot path.
func (e *Engine) UpdateDriver(snap DriverSnapshot) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.drivers[snap.Name] = snap
}

// GetDriver returns a driver snapshot if available.
func (e *Engine) GetDriver(name DriverName) (DriverSnapshot, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	s, ok := e.drivers[name]
	return s, ok
}

// Evaluate computes the confluence score from current driver snapshots.
// This is the ONLY method called from the signal hot path — it reads
// cached snapshots and never performs external I/O.
func (e *Engine) Evaluate(signalDirection Direction, eventRisk EventRiskLevel) ConfluenceResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	now := time.Now().UTC()
	result := ConfluenceResult{
		Mode:           e.cfg.Mode,
		ModelVersion:   e.modelVersion,
		WeightsVersion: e.weightsVersion,
		Timestamp:      now,
		EventRisk:      eventRisk,
	}

	if e.cfg.Mode == ModeDisabled {
		result.DataQuality = QualityMissing
		return result
	}

	// Collect active driver snapshots with freshness decay
	var activeDrivers []DriverSnapshot
	var missingDrivers []string
	var warnings []string

	for name, snap := range e.drivers {
		// Check if driver is enabled
		if !e.driverEnabled(name) {
			continue
		}

		// Calculate freshness
		ttl := e.cfg.FreshnessTTL[name]
		if ttl > 0 {
			snap.Freshness = computeFreshness(snap.Timestamp, ttl, now)
		} else {
			snap.Freshness = 1.0
		}

		// Mark stale
		if snap.Freshness < 0.1 {
			snap.Quality = QualityStale
			warnings = append(warnings, string(name)+" is stale")
		}

		// Calculate effective weight
		baseWeight := e.cfg.Weights[name]
		snap.BaseWeight = baseWeight
		snap.EffectiveWeight = baseWeight * snap.Confidence * snap.Freshness

		// Store updated snapshot
		e.drivers[name] = snap
		result.DriverSnapshot = append(result.DriverSnapshot, snap)

		if snap.Quality == QualityConnected || snap.Quality == QualityDegraded {
			activeDrivers = append(activeDrivers, snap)
		} else {
			missingDrivers = append(missingDrivers, string(name))
		}
	}

	// Check for drivers that are enabled but have no snapshot at all
	for name := range e.cfg.Weights {
		if !e.driverEnabled(name) {
			continue
		}
		if _, exists := e.drivers[name]; !exists {
			missingDrivers = append(missingDrivers, string(name))
		}
	}

	result.MissingDrivers = missingDrivers
	result.Warnings = warnings

	// Calculate confluence score
	score, agreement, conflict, primary, opposing := e.computeConfluence(activeDrivers, signalDirection)
	result.Score = score
	result.Agreement = agreement
	result.Conflict = conflict
	result.PrimaryDrivers = primary
	result.OpposingDrivers = opposing

	// Determine direction
	if score > 15 {
		result.Direction = DirBullish
	} else if score < -15 {
		result.Direction = DirBearish
	} else {
		result.Direction = DirNeutral
	}

	// Confidence from agreement and data quality
	result.Confidence = e.computeConfidence(activeDrivers, agreement, missingDrivers)

	// Data quality
	result.DataQuality = e.computeDataQuality(activeDrivers, missingDrivers)

	// Safe-haven regime
	result.Regime = e.safeHaven.Classify(activeDrivers)

	// Correlation regime
	result.CorrelationRegime = e.correlation.Classify()

	// Divergence detection
	result.DivergenceSeverity = e.divergence.Detect(signalDirection, activeDrivers, result.Regime)

	// Bounded score adjustment
	result.ScoreAdjustment = e.computeAdjustment(score, result.Confidence, result.DivergenceSeverity, eventRisk)

	return result
}

// computeConfluence calculates the weighted confluence score.
// Implements anti-double-counting for DXY/EURUSD collinearity.
func (e *Engine) computeConfluence(drivers []DriverSnapshot, signalDir Direction) (score, agreement, conflict float64, primary, opposing []string) {
	if len(drivers) == 0 {
		return 0, 0, 0, nil, nil
	}

	totalWeight := 0.0
	weightedScore := 0.0
	agreeCount := 0
	conflictCount := 0

	// Anti-double-counting: DXY is primary, EURUSD is confirmation
	// EURUSD effective weight is reduced when DXY is also present
	dxyPresent := false
	for _, d := range drivers {
		if d.Name == DriverDXY {
			dxyPresent = true
		}
		if d.Name == DriverEURUSD {
		}
	}

	for _, d := range drivers {
		effWeight := d.EffectiveWeight

		// Collinearity control: reduce EURUSD weight when DXY is present
		if d.Name == DriverEURUSD && dxyPresent {
			effWeight *= 0.4 // EURUSD contributes 40% of its weight as confirmation only
		}

		weightedScore += d.ImpactScore * effWeight
		totalWeight += effWeight

		// Check agreement with signal direction
		if signalDir != DirNeutral {
			if (d.Direction == DirBullish && signalDir == DirBullish) ||
				(d.Direction == DirBearish && signalDir == DirBearish) {
				agreeCount++
				primary = append(primary, string(d.Name))
			} else if (d.Direction == DirBullish && signalDir == DirBearish) ||
				(d.Direction == DirBearish && signalDir == DirBullish) {
				conflictCount++
				opposing = append(opposing, string(d.Name))
			}
		}
	}

	if totalWeight > 0 {
		score = weightedScore / totalWeight
	}

	totalDrivers := len(drivers)
	if totalDrivers > 0 {
		agreement = float64(agreeCount) / float64(totalDrivers)
		conflict = float64(conflictCount) / float64(totalDrivers)
	}

	// Clamp score to [-100, 100]
	score = math.Max(-100, math.Min(100, score))

	return score, agreement, conflict, primary, opposing
}

// computeConfidence derives confidence from agreement ratio and data completeness.
func (e *Engine) computeConfidence(drivers []DriverSnapshot, agreement float64, missing []string) float64 {
	if len(drivers) == 0 {
		return 0
	}
	// Base confidence from agreement
	conf := agreement * 0.6

	// Penalize for missing drivers
	totalExpected := len(e.cfg.Weights)
	available := len(drivers)
	if totalExpected > 0 {
		completeness := float64(available) / float64(totalExpected)
		conf *= 0.5 + 0.5*completeness // 50% base + up to 50% from completeness
	}

	return math.Max(0, math.Min(1, conf))
}

// computeDataQuality determines overall data quality state.
func (e *Engine) computeDataQuality(drivers []DriverSnapshot, missing []string) DataQuality {
	if len(drivers) == 0 {
		return QualityMissing
	}
	staleCount := 0
	for _, d := range drivers {
		if d.Quality == QualityStale || d.Quality == QualityDegraded {
			staleCount++
		}
	}
	if staleCount == len(drivers) {
		return QualityStale
	}
	if staleCount > 0 || len(missing) > len(drivers)/2 {
		return QualityDegraded
	}
	return QualityConnected
}

// computeAdjustment calculates the bounded score adjustment.
// In shadow mode: returns 0 (no production impact).
// In active mode: bounded by MaxBonus/MaxPenalty.
func (e *Engine) computeAdjustment(score, confidence float64, divergence DivergenceSeverity, eventRisk EventRiskLevel) float64 {
	if e.cfg.Mode != ModeActive {
		return 0
	}

	// Base adjustment from confluence score
	adjustment := score * confidence * 0.1 // scale factor

	// Divergence penalty
	switch divergence {
	case DivModerate:
		adjustment *= 0.5
	case DivHigh:
		adjustment *= 0.25
	case DivExtreme:
		adjustment *= 0.1
	}

	// Event risk penalty
	switch eventRisk {
	case EventElevated:
		adjustment *= 0.7
	case EventHigh:
		adjustment *= 0.3
	case EventExtreme:
		adjustment = 0 // block all macro adjustment during extreme events
	}

	// Apply bounds
	adjustment = math.Max(e.cfg.MaxPenalty, math.Min(e.cfg.MaxBonus, adjustment))

	return adjustment
}

func (e *Engine) driverEnabled(name DriverName) bool {
	switch name {
	case DriverDXY:
		return e.cfg.DXYEnabled
	case DriverEURUSD:
		return e.cfg.EURUSDEnabled
	case DriverRealYields:
		return e.cfg.RealYieldsEnabled
	case DriverVIX:
		return e.cfg.VIXEnabled
	case DriverCOT:
		return e.cfg.COTEnabled
	case DriverBTC:
		return e.cfg.BTCEnabled
	case DriverOil:
		return e.cfg.OilEnabled
	case DriverUSDJPY:
		return e.cfg.USDJPYEnabled
	case DriverETF:
		return e.cfg.ETFEnabled
	default:
		return true
	}
}

// computeFreshness calculates a 0-1 freshness score from age and TTL.
func computeFreshness(timestamp time.Time, ttlSec int, now time.Time) float64 {
	if ttlSec <= 0 {
		return 1.0
	}
	age := now.Sub(timestamp).Seconds()
	if age <= 0 {
		return 1.0
	}
	if age >= float64(ttlSec) {
		return 0.0
	}
	return 1.0 - (age / float64(ttlSec))
}

// FormatReason produces a human-readable explanation of the cross-market context.
func (r *ConfluenceResult) FormatReason() string {
	if r.Score == 0 && len(r.DriverSnapshot) == 0 {
		return "Cross-market data unavailable."
	}
	dir := "neutral"
	if r.Score > 15 {
		dir = "bullish"
	} else if r.Score < -15 {
		dir = "bearish"
	}
	reason := "Cross-market confluence: " + dir + " (" + formatFloat(r.Score) + "/100)"
	if r.DivergenceSeverity != DivNone {
		reason += " | Divergence: " + string(r.DivergenceSeverity)
	}
	if r.EventRisk != EventNormal {
		reason += " | Event risk: " + string(r.EventRisk)
	}
	if len(r.MissingDrivers) > 0 {
		reason += " | Missing: " + joinStrings(r.MissingDrivers)
	}
	return reason
}

func formatFloat(f float64) string {
	return time.Now().UTC().Format("") + // placeholder
		floatToString(f, 1)
}

func floatToString(f float64, precision int) string {
	// Simple float to string without fmt for production safety
	mult := 1.0
	for i := 0; i < precision; i++ {
		mult *= 10
	}
	rounded := math.Round(f*mult) / mult
	return string(floatToBytes(rounded, precision))
}

func floatToBytes(f float64, precision int) []byte {
	if f == 0 {
		return []byte("0")
	}
	negative := f < 0
	if negative {
		f = -f
	}
	intPart := int64(f)
	fracPart := f - float64(intPart)
	result := []byte{}
	if negative {
		result = append(result, '-')
	}
	result = append(result, intToBytes(intPart)...)
	if precision > 0 {
		result = append(result, '.')
		for i := 0; i < precision; i++ {
			fracPart *= 10
			d := int(fracPart)
			result = append(result, byte('0'+d))
			fracPart -= float64(d)
		}
	}
	return result
}

func intToBytes(n int64) []byte {
	if n == 0 {
		return []byte("0")
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return digits
}

func joinStrings(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for i := 1; i < len(ss); i++ {
		result += ", " + ss[i]
	}
	return result
}

// Correlation returns the correlation detector for external observation feeding.
func (e *Engine) Correlation() *CorrelationDetector {
	return e.correlation
}
