package features

// ConfirmationScoreGate implements the playbook 4-point checklist:
// Bullish: EMA9>EMA21 (+1), SAR bullish (+1), CCI>0 rising (+1), StochRSI K>D rising (+1)
// Bearish: EMA9<EMA21 (+1), SAR bearish (+1), CCI<0 falling (+1), StochRSI K<D falling (+1)
//
// Per-strategy minimums (playbook §9.3):
//
//	ULTRA_SCALPING, STANDARD_SCALPING: 3/4
//	STANDARD_SWING, TREND_SWING: 2/4
//
// This is the "last-mile" playbook filter that complements the composite
// scoring system. A signal must pass BOTH the composite score gate and the
// playbook confirmation score before being delivered as EXECUTABLE.
type ConfirmationScoreGate struct {
	MinimumScores map[string]int
}

func NewConfirmationScoreGate() *ConfirmationScoreGate {
	return &ConfirmationScoreGate{
		MinimumScores: map[string]int{
			"ULTRA_SCALPING":    3,
			"STANDARD_SCALPING": 3,
			"STANDARD_SWING":    2,
			"TREND_SWING":       2,
			"MARNIE_FIB":        2,
		},
	}
}

type ConfirmationInput struct {
	// EMA9/EMA21 comparison (direction bias)
	EMA9, EMA21           float64
	EMADirPrev, EMADirNow int // -1 below, +1 above; used to detect flips (noise filter)

	// SAR
	SARLong bool // true = bullish
	SARPrev bool

	// CCI(20)
	CCINow  float64
	CCIPrev float64

	// StochRSI (K/D, with previous)
	StochRSI_K     float64
	StochRSI_D     float64
	StochRSI_KPrev float64
	StochRSI_DPrev float64

	// Trigger candle context
	CandleOpen, CandleClose float64
	CandleHigh, CandleLow   float64
	PriorHigh, PriorLow     float64
	UpperWick, LowerWick    float64
	CandleRange             float64
	Spread                  float64
	SLDistance              float64
}

type ConfirmationResult struct {
	Score        int
	Direction    string // "BUY" | "SELL"
	Passed       bool
	Reason       string // human-readable reason for rejection
	TriggerValid bool   // playbook §9.2 trigger candle definition
}

func (g *ConfirmationScoreGate) Evaluate(in ConfirmationInput, strategyID string) ConfirmationResult {
	out := ConfirmationResult{}

	// ─── Trigger candle (§9.2) ───
	if in.CandleRange <= 0 {
		out.Reason = "zero_range_candle"
		return out
	}

	bull := in.CandleClose > in.CandleOpen
	body := in.CandleClose - in.CandleOpen
	if body < 0 {
		body = -body
	}
	if body < 0.5*in.CandleRange {
		out.Reason = "candle_body_below_50pct_of_range"
		return out
	}

	if bull {
		if !(in.CandleClose > in.EMA9 && in.CandleClose > in.EMA21 && in.CandleClose > in.PriorHigh) {
			out.Reason = "bull_trigger_candle_invalid_must_close_above_ema9_21_and_prior_high"
			return out
		}
		out.Direction = "BUY"
	} else {
		if !(in.CandleClose < in.EMA9 && in.CandleClose < in.EMA21 && in.CandleClose < in.PriorLow) {
			out.Reason = "bear_trigger_candle_invalid_must_close_below_ema9_21_and_prior_low"
			return out
		}
		out.Direction = "SELL"
	}
	out.TriggerValid = true

	// ─── Wick quality checks ───
	if bull && in.UpperWick > 0.6*in.CandleRange {
		out.Reason = "bull_upper_wick_too_long_distribution"
		return out
	}
	if !bull && in.LowerWick > 0.6*in.CandleRange {
		out.Reason = "bear_lower_wick_too_long_accumulation"
		return out
	}

	// ─── Confirmation score ──
	bullish := body > 0
	score := 0
	if bullish {
		if in.EMA9 > in.EMA21 {
			score++
		}
		if in.SARLong {
			score++
		}
		if in.CCINow > 0 && in.CCINow >= in.CCIPrev {
			score++
		}
		if in.StochRSI_K > in.StochRSI_D && in.StochRSI_K >= in.StochRSI_KPrev {
			score++
		}
	} else {
		if in.EMA9 < in.EMA21 {
			score++
		}
		if !in.SARLong {
			score++
		}
		if in.CCINow < 0 && in.CCINow <= in.CCIPrev {
			score++
		}
		if in.StochRSI_K < in.StochRSI_D && in.StochRSI_K <= in.StochRSI_KPrev {
			score++
		}
	}

	min := g.MinimumScores[strategyID]
	if min == 0 {
		min = 3
	}
	if score < min {
		out.Score = score
		out.Reason = "confirmation_score_below_minimum"
		return out
	}

	// ─── Spread filters (§8) ──
	// SL Distance < 5 x Spread → reject
	if in.SLDistance > 0 && in.Spread > 0 && in.SLDistance < 5*in.Spread {
		out.Score = score
		out.Reason = "sl_distance_less_than_5x_spread"
		return out
	}
	// Trigger Candle Range < 3 x Spread → reject
	if in.CandleRange > 0 && in.Spread > 0 && in.CandleRange < 3*in.Spread {
		out.Score = score
		out.Reason = "trigger_candle_range_less_than_3x_spread"
		return out
	}

	out.Score = score
	out.Passed = true
	return out
}
