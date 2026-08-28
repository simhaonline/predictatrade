package calibrate

import (
	"testing"

	"pat-engine/internal/types"
)

func sample() []Outcome {
	return []Outcome{
		{StrategyID: "ULTRA_SCALPING", Regime: "RANGE", RawScore: 50, Win: true},
		{StrategyID: "ULTRA_SCALPING", Regime: "RANGE", RawScore: 52, Win: true},
		{StrategyID: "ULTRA_SCALPING", Regime: "RANGE", RawScore: 55, Win: false},
		{StrategyID: "ULTRA_SCALPING", Regime: "RANGE", RawScore: 90, Win: true},
		{StrategyID: "ULTRA_SCALPING", Regime: "RANGE", RawScore: 92, Win: true},
		{StrategyID: "ULTRA_SCALPING", Regime: "TREND", RawScore: 80, Win: false},
	}
}

func TestEmpiricalFitPredict(t *testing.T) {
	m := NewEmpirical(5)
	m.Fit(sample())

	// Known regime: high score should predict a high win fraction (Laplace-smoothed).
	p, target, model, ok := m.Predict("ULTRA_SCALPING", "RANGE", 91)
	if !ok {
		t.Fatal("expected ok for seen regime")
	}
	if target != TargetTP1BeforeSL {
		t.Fatalf("target = %q, want %q", target, TargetTP1BeforeSL)
	}
	if model != "empirical-region" {
		t.Fatalf("model = %q", model)
	}
	if p <= 0.5 || p > 1.0 {
		t.Fatalf("high-score prob = %v, expected (0.5,1.0]", p)
	}

	// Low score in the same regime: lower probability.
	pLow, _, _, ok2 := m.Predict("ULTRA_SCALPING", "RANGE", 51)
	if !ok2 {
		t.Fatal("expected ok for low score")
	}
	if pLow >= p {
		t.Fatalf("low-score prob %v should be < high-score prob %v", pLow, p)
	}
}

func TestEmpiricalUnseenIsUncalibrated(t *testing.T) {
	m := NewEmpirical(5)
	m.Fit(sample())

	// A regime never seen by the model cannot be guessed.
	_, target, model, ok := m.Predict("ULTRA_SCALPING", "VOLATILE", 70)
	if ok {
		t.Fatal("expected ok=false for unseen regime")
	}
	if model != "UNCALIBRATED" {
		t.Fatalf("model = %q, want UNCALIBRATED", model)
	}
	if target != TargetTP1BeforeSL {
		t.Fatalf("target = %q", target)
	}
}

func TestModelSerializeRoundTrip(t *testing.T) {
	m := NewEmpirical(5)
	m.Fit(sample())
	b, err := m.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	m2, err := LoadModel(b)
	if err != nil {
		t.Fatal(err)
	}
	p1, _, _, _ := m.Predict("ULTRA_SCALPING", "RANGE", 91)
	p2, _, _, _ := m2.Predict("ULTRA_SCALPING", "RANGE", 91)
	if p1 != p2 {
		t.Fatalf("round-trip mismatch: %v vs %v", p1, p2)
	}
}

func TestAttachMarksUncalibratedWithoutModel(t *testing.T) {
	sig := &types.Signal{StrategyID: types.StrategyUltraScalping, RawScore: 80, Regime: "RANGE"}
	Attach(sig, nil)
	if sig.ProbabilityModel != "UNCALIBRATED" {
		t.Fatalf("model = %q, want UNCALIBRATED", sig.ProbabilityModel)
	}
	if sig.CalibratedProbability != 0 {
		t.Fatalf("prob = %v, want 0", sig.CalibratedProbability)
	}
	if sig.ProbabilityTarget != TargetTP1BeforeSL {
		t.Fatalf("target = %q", sig.ProbabilityTarget)
	}
}
