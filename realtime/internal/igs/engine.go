// Package igs — Institutional Gold Signal (IGS) composite engine.
//
// Implements the institutional gold intelligence hierarchy described in
// check.md as a deterministic, bounded CONFIRMATION composite on the same
// chassis as the crossmarket engine:
//
//	IGS = CentralBankFlow + ETFFlow + COTPositioning + RealYieldRegime +
//	      USDRegime + COMEXPositioning + OptionsGamma + PhysicalDemand +
//	      InstitutionalResearchSentiment
//
// Tier semantics (check.md L440-455):
//
//	Tier S: real yields, USD regime, central banks, London OTC
//	Tier A: COT/COMEX, ETF flows, options/gamma
//	Tier B: institutional research (LLM), physical demand
//	Tier D: social sentiment (NOT part of IGS)
//
// Safety model:
//   - A component with no feed reports Quality=UNAVAILABLE and contributes 0.
//     Missing capability is EXPOSED, never fabricated (AGENTS.md market-data truth).
//   - IGS is a bounded score adjustment consumer with shadow mode default:
//     it NEVER generates trades and NEVER overrides hard gates.
//   - No synchronous I/O in the hot path: components are updated by background
//     refresh loops; Evaluate() reads cached state only.
package igs

import (
	"math"
	"sync"
	"time"
)

// Version is the IGS model version. Bump when normalization logic changes.
const Version = "1.0.0"

// WeightsVersion is the default weight-set version (mirrors igs_weight_versions).
const WeightsVersion = "1.0.0"

// ComponentName identifies an IGS input component.
type ComponentName string

const (
	ComponentETF            ComponentName = "etf_flows"
	ComponentCentralBank    ComponentName = "central_bank_flow"
	ComponentCOT            ComponentName = "cot_positioning"
	ComponentRealYield      ComponentName = "real_yield_regime"
	ComponentUSDRegime      ComponentName = "usd_regime"
	ComponentOptionsGamma   ComponentName = "options_gamma"
	ComponentAIResearch     ComponentName = "institutional_research"
	ComponentPhysicalDemand ComponentName = "physical_demand"
)

// ComponentKeys are the canonical ordered keys of the IGS composite.
// Order is stable for deterministic persistence and UI rendering.
var ComponentKeys = []ComponentName{
	ComponentUSDRegime,
	ComponentRealYield,
	ComponentETF,
	ComponentCentralBank,
	ComponentCOT,
	ComponentOptionsGamma,
	ComponentAIResearch,
	ComponentPhysicalDemand,
}

// PhysicalDemandTier marks check.md Tier-B for the physical demand component.
const PhysicalDemand = ComponentPhysicalDemand

// Classification bands (check.md L480-492).
type Classification string

const (
	ClassExtremeBull  Classification = "EXTREME_INSTITUTIONAL_BULLISH"
	ClassStrongBull   Classification = "STRONG_BULLISH"
	ClassModerateBull Classification = "MODERATE_BULLISH"
	ClassNeutral      Classification = "NEUTRAL_CONFLICT"
	ClassModerateBear Classification = "MODERATE_BEARISH"
	ClassStrongBear   Classification = "STRONG_BEARISH"
	ClassExtremeBear  Classification = "EXTREME_INSTITUTIONAL_BEARISH"
	ClassInsufficient Classification = "INSUFFICIENT_DATA"
)

// Mode controls IGS influence on production (mirrors crossmarket.Mode).
type Mode string

const (
	ModeDisabled Mode = "disabled"
	ModeShadow   Mode = "shadow"
	ModeActive   Mode = "active"
)

// Quality states (aligned with crossmarket.DataQuality semantics).
type Quality string

const (
	QualityConnected   Quality = "CONNECTED"
	QualityDegraded    Quality = "DEGRADED"
	QualityStale       Quality = "STALE"
	QualityUnavailable Quality = "UNAVAILABLE"
	QualityError       Quality = "ERROR"
)

// Direction of institutional flow influence on gold.
type Direction string

const (
	DirBullish Direction = "BULLISH"
	DirBearish Direction = "BEARISH"
	DirNeutral Direction = "NEUTRAL"
)

// Component is one normalized institutional input (-100..+100).
type Component struct {
	Name       ComponentName `json:"name"`
	RawValue   float64       `json:"raw_value,omitempty"`
	Impact     float64       `json:"impact"` // -100..+100
	Direction  Direction     `json:"direction"`
	Confidence float64       `json:"confidence"` // 0..1
	Freshness  float64       `json:"freshness"`  // 0..1 (1 = just observed)
	Quality    Quality       `json:"quality"`
	Source     string        `json:"source"`
	Reason     string        `json:"reason"`
	Timestamp  time.Time     `json:"timestamp"`
	Weight     float64       `json:"weight"`
	EffectiveW float64       `json:"effective_weight"`
}

// Config holds IGS engine configuration.
type Config struct {
	Enabled bool
	Mode    Mode

	// Which components are enabled (feed wired). A disabled component is
	// excluded from both scoring and "missing" warnings (operator decision).
	EnabledComponents map[ComponentName]bool

	// Weights — Tier hierarchy from check.md. Base weights before
	// confidence/freshness scaling. Versioned in trading.igs_weight_versions.
	Weights map[ComponentName]float64

	// Freshness TTLs in seconds per component.
	FreshnessTTL map[ComponentName]int

	// Bounded adjustment applied to the existing raw score in ACTIVE mode.
	MaxBonus   float64
	MaxPenalty float64
}

// DefaultConfig returns the safe default weight set (shadow mode).
// Weights follow check.md tier hierarchy; components without feeds are
// disabled by default and surfaced as UNAVAILABLE in health views.
func DefaultConfig() Config {
	return Config{
		Enabled: false, // opt-in via IGS_ENABLED=true
		Mode:    ModeShadow,
		EnabledComponents: map[ComponentName]bool{
			ComponentUSDRegime:      true, // fed from crossmarket DXY driver
			ComponentCOT:            true, // fed from existing COT provider
			ComponentRealYield:      true, // fed from FRED real yield
			ComponentETF:            false,
			ComponentCentralBank:    false, // no feed — surfaced UNAVAILABLE when IGS enabled
			ComponentOptionsGamma:   false,
			ComponentAIResearch:     false, // TradingAgents adapter opt-in
			ComponentPhysicalDemand: false,
		},
		Weights: map[ComponentName]float64{
			// Tier S
			ComponentUSDRegime:   20.0,
			ComponentRealYield:   20.0,
			ComponentCentralBank: 15.0,
			// Tier A
			ComponentCOT:          12.0,
			ComponentETF:          15.0,
			ComponentOptionsGamma: 8.0,
			// Tier B
			ComponentAIResearch:     6.0,
			ComponentPhysicalDemand: 4.0,
		},
		FreshnessTTL: map[ComponentName]int{
			ComponentUSDRegime:      600,     // 10 min (DXY refresh cadence)
			ComponentRealYield:      86400,   // daily
			ComponentCentralBank:    2592000, // monthly data
			ComponentCOT:            604800,  // weekly
			ComponentETF:            86400,   // daily
			ComponentOptionsGamma:   86400,
			ComponentAIResearch:     259200, // 3 days
			ComponentPhysicalDemand: 2592000,
		},
		MaxBonus:   10.0,
		MaxPenalty: -15.0,
	}
}

// Engine is the deterministic IGS composite.
type Engine struct {
	mu         sync.RWMutex
	cfg        Config
	components map[ComponentName]Component
}

// NewEngine constructs an IGS engine.
func NewEngine(cfg Config) *Engine {
	return &Engine{cfg: cfg, components: make(map[ComponentName]Component)}
}

// UpdateComponent feeds a background-refreshed component snapshot.
// Called by refresh loops / crossmarket fan-in — never the hot path.
func (e *Engine) UpdateComponent(c Component) {
	c.Timestamp = c.Timestamp.UTC()
	c.Impact = clamp100(c.Impact)
	if c.Freshness > 0 {
		c.Freshness = clamp01(c.Freshness)
	}
	c.Confidence = clamp01(c.Confidence)
	c.Direction = derivDirection(c.Impact)
	e.mu.Lock()
	e.components[c.Name] = c
	e.mu.Unlock()
}

// Composite is the scored IGS output.
type Composite struct {
	Score               float64        `json:"score"`
	Classification      Classification `json:"classification"`
	Direction           Direction      `json:"direction"`
	Confidence          float64        `json:"confidence"`
	Agreement           float64        `json:"agreement"`
	Conflict            float64        `json:"conflict"`
	DataQuality         Quality        `json:"data_quality"`
	ComponentsAvailable int            `json:"components_available"`
	ComponentsTotal     int            `json:"components_total"`
	MissingComponents   []string       `json:"missing_components"`
	Warnings            []string       `json:"warnings"`
	ScoreAdjustment     float64        `json:"score_adjustment"`
	Mode                Mode           `json:"mode"`
	ModelVersion        string         `json:"model_version"`
	WeightsVersion      string         `json:"weights_version"`
	Components          []Component    `json:"components"`
	Timestamp           time.Time      `json:"timestamp"`
}

// Evaluate computes the IGS from cached components. Read-only hot path:
// no I/O, no allocations beyond the result. signalDirection allows
// agreement/conflict attribution like crossmarket.
func (e *Engine) Evaluate(signalDirection Direction) Composite {
	e.mu.RLock()
	defer e.mu.RUnlock()

	now := time.Now().UTC()
	if !e.cfg.Enabled || e.cfg.Mode == ModeDisabled {
		return Composite{Mode: e.cfg.Mode, ModelVersion: Version, WeightsVersion: WeightsVersion, Timestamp: now}
	}

	weighted := 0.0
	totalW := 0.0
	agree := 0.0
	conflict := 0.0
	available := 0
	var warnings []string
	var missing []string
	out := make([]Component, 0, len(ComponentKeys))

	for _, key := range ComponentKeys {
		if !e.cfg.EnabledComponents[key] {
			continue
		}
		c, ok := e.components[key]
		if !ok || c.Quality == QualityUnavailable || c.Quality == QualityError {
			if c.Quality == QualityUnavailable || c.Quality == QualityError || !ok {
				missing = append(missing, string(key))
			}
			continue
		}

		ttl := e.cfg.FreshnessTTL[key]
		fresh := c.Freshness
		if fresh <= 0 {
			fresh = computeFreshness(c.Timestamp, ttl, now)
		}
		if fresh < 0.1 {
			c.Freshness = fresh
			c.Quality = QualityStale
			warnings = append(warnings, string(key)+" is stale")
		}
		if fresh <= 0.0 {
			missing = append(missing, string(key))
			continue
		}

		w := e.cfg.Weights[key]
		c.Weight = w
		c.EffectiveW = w * c.Confidence * fresh
		// Freshness decay applied to impact so stale-but-recent data still
		// contributes proportionally rather than being dropped entirely.
		weighted += c.Impact * c.EffectiveWeight()
		totalW += c.EffectiveWeight()
		available++

		if signalDirection != DirNeutral {
			if (c.Direction == DirBullish && signalDirection == DirBullish) ||
				(c.Direction == DirBearish && signalDirection == DirBearish) {
				agree += c.EffectiveWeight()
			} else if c.Direction != DirNeutral {
				conflict += c.EffectiveWeight()
			}
		}

		out = append(out, c)
	}

	score := 0.0
	if totalW > 0 {
		weighted = weighted / totalW
		score = clamp100(weighted)
	}
	dir := DirNeutral
	switch {
	case score > 19:
		dir = DirBullish
	case score < -19:
		dir = DirBearish
	}

	agreePct, conflictPct := 0.0, 0.0
	if totalW > 0 && signalDirection != DirNeutral {
		agreePct = clamp01(agree / totalW)
		conflictPct = clamp01(conflict / totalW)
	}

	comp := Composite{
		Score:               score,
		Classification:      classify(score, available),
		Direction:           dir,
		Agreement:           agreePct,
		Conflict:            conflictPct,
		Confidence:          computeConfidence(available, agreePct),
		DataQuality:         dataQuality(available),
		ComponentsAvailable: available,
		ComponentsTotal:     len(ComponentKeys),
		MissingComponents:   missing,
		Warnings:            warnings,
		Mode:                e.cfg.Mode,
		ModelVersion:        Version,
		WeightsVersion:      WeightsVersion,
		Components:          out,
		Timestamp:           now,
	}
	comp.ScoreAdjustment = boundedAdjustment(score, comp.Confidence, e.cfg)
	return comp
}

// EffectiveWeight returns weight*confidence*freshness (helper kept on Component
// so callers can audit effective influence without recomputing).
func (c Component) EffectiveWeight() float64 {
	fresh := c.Freshness
	if fresh <= 0 {
		fresh = 1.0 // timestamp-derived freshness is applied before this call
	}
	return clamp01(c.Confidence) * c.Weight * fresh
}

// Classify maps a score to check.md bands; below min components → INSUFFICIENT.
func classify(score float64, available int) Classification {
	if available < 2 {
		return ClassInsufficient
	}
	switch {
	case score >= 80:
		return ClassExtremeBull
	case score >= 50:
		return ClassStrongBull
	case score >= 20:
		return ClassModerateBull
	case score > -20:
		return ClassNeutral
	case score > -50:
		return ClassModerateBear
	case score > -80:
		return ClassStrongBear
	default:
		return ClassExtremeBear
	}
}

// boundedAdjustment mirrors crossmarket: 0 unless active mode.
func boundedAdjustment(score, confidence float64, cfg Config) float64 {
	if cfg.Mode != ModeActive {
		return 0
	}
	adj := score * confidence * 0.05
	return math.Max(cfg.MaxPenalty, math.Min(cfg.MaxBonus, adj))
}

func computeConfidence(available int, agree float64) float64 {
	if available == 0 {
		return 0
	}
	// Completeness-dominant confidence: with only partial Tier coverage IGS
	// must not present high conviction.
	completeness := float64(available) / float64(len(ComponentKeys))
	return clamp01(completeness*0.6 + agree*0.4)
}

func dataQuality(available int) Quality {
	switch {
	case available == 0:
		return QualityUnavailable
	case available < 3:
		return QualityDegraded
	default:
		return QualityConnected
	}
}

func computeFreshness(ts time.Time, ttlSec int, now time.Time) float64 {
	if ttlSec <= 0 {
		return 1.0
	}
	age := now.Sub(ts).Seconds()
	if age <= 0 {
		return 1.0
	}
	if age >= float64(ttlSec) {
		return 0.0
	}
	return 1.0 - age/float64(ttlSec)
}

func derivDirection(impact float64) Direction {
	switch {
	case impact > 19:
		return DirBullish
	case impact < -19:
		return DirBearish
	default:
		return DirNeutral
	}
}

func clamp100(v float64) float64 { return math.Max(-100, math.Min(100, v)) }
func clamp01(v float64) float64  { return math.Max(0, math.Min(1, v)) }

// FromCrossMarket builds IGS component snapshots from existing crossmarket
// drivers — this is the fan-in that keeps IGS zero-new-I/O.
func FromCrossMarket(snap CrossMarketDriver) Component {
	switch snap.Name {
	case "dxy":
		impact := snap.ImpactScore
		if impact > 100 {
			impact = 100
		}
		if impact < -100 {
			impact = -100
		}
		// Weak USD → bullish gold: crossmarket DXY impact is already signed
		// in gold-impact direction, so reuse directly.
		return Component{
			Name:       ComponentUSDRegime,
			RawValue:   snap.RawValue,
			Impact:     impact,
			Confidence: snap.Confidence,
			Quality:    QualityFromCrossQuality(snap.Quality),
			Source:     snap.Source,
			Reason:     "USD regime via DXY — " + snap.Reason,
			Timestamp:  snap.Timestamp,
		}
	case "real_yields":
		return Component{
			Name:       ComponentRealYield,
			RawValue:   snap.RawValue,
			Impact:     clamp100(snap.ImpactScore),
			Confidence: snap.Confidence,
			Quality:    QualityFromCrossQuality(snap.Quality),
			Source:     snap.Source,
			Reason:     "Real-yield regime — " + snap.Reason,
			Timestamp:  snap.Timestamp,
		}
	case "cot":
		// COT percentile is inverted for impact (crowding = contrarian); reuse
		// the crossmarket normalization which already applies that transform.
		return Component{
			Name:       ComponentCOT,
			RawValue:   snap.RawValue,
			Impact:     clamp100(snap.ImpactScore),
			Confidence: snap.Confidence,
			Quality:    QualityFromCrossQuality(snap.Quality),
			Source:     snap.Source,
			Reason:     "Managed-money positioning — " + snap.Reason,
			Timestamp:  snap.Timestamp,
		}
	default:
		return Component{Quality: QualityUnavailable}
	}
}
