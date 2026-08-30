package features

import "testing"

func TestConfirmationScoreGate_BullishValid(t *testing.T) {
	g := NewConfirmationScoreGate()
	in := ConfirmationInput{
		EMA9: 2402, EMA21: 2400,
		SARLong:        true,
		CCINow:  25, CCIPrev: 10,
		StochRSI_K:  0.7, StochRSI_D: 0.4, StochRSI_KPrev: 0.5,
		CandleOpen: 2400, CandleClose: 2412,
		CandleHigh: 2414, CandleLow: 2398,
		PriorHigh: 2410, PriorLow: 2396,
		UpperWick: 1, LowerWick: 2,
		CandleRange: 16,
		Spread: 0.2, SLDistance: 5,
	}
	res := g.Evaluate(in, "ULTRA_SCALPING")
	if !res.Passed || res.Score < 3 { t.Fatalf("want pass score>=3, got pass=%v score=%d reason=%s", res.Passed, res.Score, res.Reason) }
}

func TestConfirmationScoreGate_LowScore(t *testing.T) {
	g := NewConfirmationScoreGate()
	in := ConfirmationInput{
		EMA9: 2402, EMA21: 2402,
		SARLong: false,
		CCINow:  -5, CCIPrev: -2,
		StochRSI_K: 0.3, StochRSI_D: 0.6, StochRSI_KPrev: 0.5,
		CandleOpen: 2400, CandleClose: 2412,
		CandleHigh: 2414, CandleLow: 2398,
		PriorHigh: 2410, PriorLow: 2396,
		UpperWick: 1, LowerWick: 2,
		CandleRange: 16,
		Spread: 0.2, SLDistance: 5,
	}
	res := g.Evaluate(in, "ULTRA_SCALPING")
	if res.Passed { t.Fatalf("should fail on low score: %+v", res) }
	if res.Reason != "confirmation_score_below_minimum" { t.Fatalf("want confirmation_score_below_minimum, got %s", res.Reason) }
}
