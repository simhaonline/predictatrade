package capitaltier

import (
	"testing"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		equity float64
		want   Tier
	}{
		{0, Unknown},
		{-5, Unknown},
		{99.99, Micro},
		{100, Micro},
		{499.99, Micro},
		{500, Standard},
		{792.60, Standard},
		{4999.99, Standard},
		{5000, Pro},
		{20000, Pro},
	}
	for _, c := range cases {
		if got := Classify(c.equity); got != c.want {
			t.Errorf("Classify(%v) = %q, want %q", c.equity, got, c.want)
		}
	}
}

func TestEvaluate_TightScalpEligibleAllTiers(t *testing.T) {
	// 0.8-point SL → min-lot risk $0.80; fits MICRO's $2 cap (100 × 2%).
	el := Evaluate(0.8, 0.51)
	if len(el.EligibleTiers) != 3 {
		t.Fatalf("expected all tiers eligible, got %v (exclusions %v)", el.EligibleTiers, el.Exclusions)
	}
}

func TestEvaluate_WideSwingStopProOnly(t *testing.T) {
	// 22-point SL → min-lot risk $22. MICRO cap = 100×2% = $2 → excluded.
	// STANDARD cap = 500×2% = $10 → excluded. PRO cap = 5000×2% = $100 → ok.
	el := Evaluate(22.0, 0.51)
	if len(el.EligibleTiers) != 1 || el.EligibleTiers[0] != Pro {
		t.Fatalf("expected PRO-only, got %v (exclusions %v)", el.EligibleTiers, el.Exclusions)
	}
	if el.Exclusions[Micro] != "min_lot_risk_exceeds_tier_cap" || el.Exclusions[Standard] != "min_lot_risk_exceeds_tier_cap" {
		t.Errorf("expected exclusion reasons on MICRO/STANDARD, got %v", el.Exclusions)
	}
}

func TestEvaluate_MidStopScalping(t *testing.T) {
	// 5-point SL → $5 min-lot risk: MICRO excluded ($2 cap), STANDARD+ ok.
	el := Evaluate(5.0, 0.51)
	if len(el.EligibleTiers) != 2 {
		t.Fatalf("expected STANDARD+PRO, got %v", el.EligibleTiers)
	}
}

func TestEvaluate_InvalidDistance(t *testing.T) {
	el := Evaluate(0, 0.51)
	if len(el.EligibleTiers) != 0 {
		t.Fatalf("expected no eligible tiers for zero SL distance, got %v", el.EligibleTiers)
	}
}
