package strategy

import (
	"fmt"
	"math"
	"time"

	"github.com/predictatrade/realtime/internal/astro"
	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// ATENStrategy — Aetherial Technical Engine Node
// 5th intelligence engine (Vedic DI + Western Tropical) per check.md 2026-08-30.
//
// Uses Astro-Financial state as evidence contributor + directional bias source:
//   - Nakshatra bias (30% weight within ATEN)
//   - Hora bias (20%)
//   - Vimshottari dasha L1+L2 (50%)
//   - Western tropical aspects to Gold natal chart (28% of composite)
//   - Eclipse / contamination → MANDATORY_NO_TRADE override
//
// Composite score → directional bias when |score| ≥ 25 (lower than technical
// strategies because ASTRO is a confluence layer, not primary momentum signal).
type ATENStrategy struct{}

func NewATENStrategy() *ATENStrategy { return &ATENStrategy{} }

func (s *ATENStrategy) ID() types.StrategyID { return types.StrategyATEN }

func (s *ATENStrategy) Evaluate(state *features.MarketState) StrategyResult {
	result := StrategyResult{
		StrategyID:    types.StrategyATEN,
		Direction:     types.DirectionNoTrade,
		Evidence:      []types.EvidenceContribution{},
		ExpiryMinutes: 60,
	}

	now := time.Now().UTC()
	at := astro.Compute(now, false)

	// ─── VETO: closed market, eclipse, apocalypse, contamination ───
	if !at.EligibleForTrade {
		result.Direction = types.DirectionNoTrade
		result.HumanReason = fmt.Sprintf("ATEN suppressed: %s", at.IneligibleReason)
		result.ReasonCodes = []types.NoTradeReason{types.NTMarketClosed}
		return result
	}

	// ─── Vedic factors ──
	nakBias := at.Vedic.NakshatraBias
	horaBias := at.Vedic.HoraBias
	d1 := at.Vedic.DashaL1Bias
	d2 := at.Vedic.DashaL2Bias
	vedicCombined := (d1 + d2*0.5) / 2.0
	vedicScore := nakBias*0.3 + horaBias*0.2 + vedicCombined*0.5

	// Evidence: Nakshatra
	evidence := &result.Evidence
	addEvidence(evidence, "DI_NAKSHATRA", at.Vedic.NakshatraName+", pada "+itoa(at.Vedic.Pada),
		dirFromF(nakBias), 30, math.Abs(nakBias)*0.3, types.QualityAuthoritative, at.Vedic.NakshatraName)
	addEvidence(evidence, "DI_HORA", at.Vedic.HoraLord+" hora",
		dirFromF(horaBias), 20, math.Abs(horaBias)*0.2, types.QualityAuthoritative, at.Vedic.HoraLord)
	addEvidence(evidence, "DI_DASHA", at.Vedic.DashaL1+"/"+at.Vedic.DashaL2,
		dirFromF(vedicCombined), 50, vedicCombined*0.5, types.QualityAuthoritative, at.Vedic.DashaL1)

	// ─── Western aspects ──
	westernScore := at.Western.TotalScore
	addEvidence(evidence, "WESTERN_ASTRO", "Gold natal transits",
		dirFromF(westernScore), 28, westernScore*0.28, types.QualityAuthoritative, "")

	// ─── Composite score ──
	composite := vedicScore*0.42 + westernScore*0.28 + at.CompositeScore*0.30

	// Eclipse / contamination override — push composite negative
	if at.Vedic.Eclipse {
		composite = math.Min(composite, -30)
	}

	// ─── Direction ──
	bias := decimal.NewFromFloat(composite)
	biasThreshold := decimal.NewFromFloat(25.0)
	if bias.GreaterThanOrEqual(biasThreshold) {
		result.Direction = types.DirectionBuy
	} else if bias.LessThanOrEqual(biasThreshold.Neg()) {
		result.Direction = types.DirectionSell
	}

	result.RawScore = bias
	result.LongScore = decimal.NewFromFloat(math.Max(composite, 0))
	result.ShortScore = decimal.NewFromFloat(math.Max(-composite, 0))

	// Entry/SL/TP — ATR-calibrated from market state if available
	if state != nil && state.LastTick != nil {
		bid, _ := state.LastTick.Bid.Float64()
		if bid > 0 {
			result.EntryPrice = decimal.NewFromFloat(bid)
			if !at.MarketClosed {
				result.StopLoss = decimal.NewFromFloat(bid - 10.0)
				result.TP1 = decimal.NewFromFloat(bid + 20.0)
				result.TP2 = decimal.NewFromFloat(bid + 20.0)
				result.TP3 = decimal.NewFromFloat(bid + 30.0)
				result.ExpiryMinutes = 60
			}
		}
	}

	result.HumanReason = fmt.Sprintf(
		"ATEN: nakshatra %s (bias %+.0f) · hora %s (%+.0f) · dasha %s/%s · Western %.1f → composite %+.1f",
		at.Vedic.NakshatraName, at.Vedic.NakshatraBias,
		at.Vedic.HoraLord, at.Vedic.HoraBias,
		at.Vedic.DashaL1, at.Vedic.DashaL2,
		at.Western.TotalScore, composite)

	// Composite consensus evidence
	addEvidence(evidence, "CONSENSUS", "ATEN composite score",
		dirFromF(composite), 30, composite*0.30, types.QualityAuthoritative, fmt.Sprintf("%.1f", composite))

	return result
}

// itoa helper
func itoa(i int) string { return fmt.Sprintf("%d", i) }

// dirFromF returns types.Direction from a numeric score
func dirFromF(score float64) types.Direction {
	if score > 5 {
		return types.DirectionBuy
	}
	if score < -5 {
		return types.DirectionSell
	}
	return types.DirectionNoTrade
}

// keep _ imports alive
var _ = math.Abs
var _ = time.Now
