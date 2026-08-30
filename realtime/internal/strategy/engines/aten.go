package engines

import (
	"fmt"
	"math"
	"time"

	"github.com/predictatrade/realtime/internal/astro"
	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/strategy"
	"github.com/predictatrade/realtime/internal/types"
)

// ATEN engine type
const Aten EngineType = "ATEN"

// AttenConfig — ATEN-specific settings (check.md: Vedic + Western astro)
var AttenConfig = EngineConfig{
	Type:            Aten,
	MinAbsATR:       2.0,
	IgnoreStructure: true,
	AllowedRegimes:  []string{},
	MinGrade:        "A",
	OverrideSL:      1.0,
	OverrideTPs:     [3]float64{1.0, 2.0, 3.0},
	OverrideExpiry:  60,
}

type AttenEngine struct{ cfg EngineConfig }

func (e *AttenEngine) Type() EngineType     { return Aten }
func (e *AttenEngine) Config() EngineConfig { return e.cfg }

func (e *AttenEngine) Evaluate(legacyResult strategy.StrategyResult, state *features.MarketState) EngineResult {
	if legacyResult.Direction != types.DirectionBuy && legacyResult.Direction != types.DirectionSell {
		return EngineResult{Result: legacyResult, Fallback: true}
	}

	at := astro.Compute(time.Now().UTC(), false)

	if !at.EligibleForTrade {
		legacyResult.Direction = types.DirectionNoTrade
		legacyResult.ReasonCodes = append(legacyResult.ReasonCodes, types.NTGateDegraded)
		return EngineResult{Result: legacyResult, RejectReason: "ATEN_ASTRO_INELIGIBLE_" + at.IneligibleReason}
	}

	score := at.CompositeScore
	if score > -25 && score < 25 {
		legacyResult.Direction = types.DirectionNoTrade
		legacyResult.ReasonCodes = append(legacyResult.ReasonCodes, types.NTGateDegraded)
		return EngineResult{Result: legacyResult, RejectReason: "ATEN_composite_below_threshold"}
	}

	if score > 0 {
		legacyResult.Direction = types.DirectionBuy
	} else {
		legacyResult.Direction = types.DirectionSell
	}

	modified := applyOverrides(legacyResult, state, e.cfg)
	modified.Confidence = confidenceFromScore(score, legacyResult.Confidence)
	modified.HumanReason = fmt.Sprintf(
		"ATEN: nakshatra %s · hora %s · dasha %s/%s · Western %.1f · composite %+.1f",
		at.Vedic.NakshatraName, at.Vedic.HoraLord, at.Vedic.DashaL1,
		at.Vedic.DashaL2, at.Western.TotalScore, at.CompositeScore)

	return EngineResult{Result: modified, Applied: true}
}

func confidenceFromScore(score, fallback float64) float64 {
	astroConf := math.Abs(score) / 100.0
	blended := astroConf*0.6 + fallback*0.4
	if blended < 0.2 {
		return 0.2
	}
	if blended > 0.95 {
		return 0.95
	}
	return blended
}
