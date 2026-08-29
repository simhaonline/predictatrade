package calibrate

import (
	"testing"

	"pat-engine/internal/types"
)

func sample() []Outcome {
	fR := Features{Regime: "RANGE"}
	fT := Features{Regime: "TREND"}
	return []Outcome{
		{StrategyID: "ULTRA_SCALPING", Regime: "RANGE", Features: fR, RawScore: 50, Win: true},
		{StrategyID: "ULTRA_SCALPING", Regime: "RANGE", Features: fR, RawScore: 52, Win: true},
		{StrategyID: "ULTRA_SCALPING", Regime: "RANGE", Features: fR, RawScore: 55, Win: false},
		{StrategyID: "ULTRA_SCALPING", Regime: "RANGE", Features: fR, RawScore: 90, Win: true},
		{StrategyID: "ULTRA_SCALPING", Regime: "RANGE", Features: fR, RawScore: 92, Win: true},
		{StrategyID: "ULTRA_SCALPING", Regime: "TREND", Features: fT, RawScore: 80, Win: false},
	}
}

func TestEmpiricalFitPredict(t *testing.T) {
	m := NewEmpirical(5)
	m.Fit(sample())

	// Known regime + high score: high win fraction (Laplace-smoothed), exact-context hit.
	p, target, model, ok := m.Predict("ULTRA_SCALPING", "RANGE", 91, Features{Regime: "RANGE"})
	if !ok {
		t.Fatal("expected ok for seen context")
	}
	if target != TargetTP1BeforeSL {
		t.Fatalf("target = %q, want %q", target, TargetTP1BeforeSL)
	}
	if model != "empirical-direct" {
		t.Fatalf("model = %q, want empirical-direct", model)
	}
	if p <= 0.5 || p > 1.0 {
		t.Fatalf("high-score prob = %v, expected (0.5,1.0]", p)
	}

	// Low score in the same context: lower probability.
	pLow, _, _, ok2 := m.Predict("ULTRA_SCALPING", "RANGE", 51, Features{Regime: "RANGE"})
	if !ok2 {
		t.Fatal("expected ok for low score")
	}
	if pLow >= p {
		t.Fatalf("low-score prob %v should be < high-score prob %v", pLow, p)
	}
}

func TestEmpiricalUnseenStrategyIsUncalibrated(t *testing.T) {
	m := NewEmpirical(5)
	m.Fit(sample())

	// A strategy never seen by the model cannot be guessed (no strategy-level basis).
	_, target, model, ok := m.Predict("TREND_SWING", "RANGE", 70, Features{Regime: "RANGE"})
	if ok {
		t.Fatal("expected ok=false for unseen strategy")
	}
	if model != "UNCALIBRATED" {
		t.Fatalf("model = %q, want UNCALIBRATED", model)
	}
	if target != TargetTP1BeforeSL {
		t.Fatalf("target = %q", target)
	}
}

func TestEmpiricalHierarchicalFallback(t *testing.T) {
	m := NewEmpirical(5)
	m.Fit(sample())

	// Unseen exact context but seen strategy -> strategy-level fallback (low confidence,
	// not uncalibrated). This is an honest broader prior, not a fabricated probability.
	p, _, model, ok := m.Predict("ULTRA_SCALPING", "VOLATILE", 70, Features{Regime: "VOLATILE"})
	if !ok {
		t.Fatal("expected ok via strategy fallback")
	}
	if model != "empirical-strategy" {
		t.Fatalf("model = %q, want empirical-strategy", model)
	}
	if p <= 0 || p >= 1 {
		t.Fatalf("strategy prior prob = %v, want (0,1)", p)
	}
}

func TestMultiTarget(t *testing.T) {
	m := NewEmpirical(5)
	// TP1 wins on high scores; TP2 (harder) wins less often.
	outs := []Outcome{}
	for _, o := range sample() {
		o.Target = TargetTP1BeforeSL
		outs = append(outs, o)
		o2 := o
		o2.Target = TargetTP2BeforeSL
		// Make TP2 strictly harder: only the 92 scorer wins it (within decile 4).
		o2.Win = o.RawScore >= 92
		outs = append(outs, o2)
	}
	m.Fit(outs)

	p1, _, _ := m.PredictTarget("ULTRA_SCALPING", "RANGE", 91, Features{Regime: "RANGE"}, TargetTP1BeforeSL)
	p2, _, ok2 := m.PredictTarget("ULTRA_SCALPING", "RANGE", 91, Features{Regime: "RANGE"}, TargetTP2BeforeSL)
	if !ok2 {
		t.Fatal("expected ok for TP2")
	}
	if p2 >= p1 {
		t.Fatalf("TP2 prob %v should be < TP1 prob %v (harder target)", p2, p1)
	}
}

func TestFeaturesFromStateBands(t *testing.T) {
	s := &types.MarketState{
		Regime:    "TRENDING_BULLISH",
		UTCHour:   9,
		SlowATR:   1.0,
		ATR:       1.5, // HIGH vol regime
		HTFBias:   types.Bullish,
		Indicators: types.Indicators{ADX: 35, RSI: 75},
		Session:   types.Session{CurrentSession: "LONDON"},
	}
	f := FeaturesFromState(s)
	if f.Regime != "TRENDING_BULLISH" || f.HTFBias != "BULLISH" || f.VolRegime != "HIGH" ||
		f.ADXBand != "STRONG" || f.RSIBand != "OB" || f.PrimeWin != "IN" || f.Session != "LONDON" {
		t.Fatalf("unexpected features: %+v", f)
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
	p1, _, _, _ := m.Predict("ULTRA_SCALPING", "RANGE", 91, Features{Regime: "RANGE"})
	p2, _, _, _ := m2.Predict("ULTRA_SCALPING", "RANGE", 91, Features{Regime: "RANGE"})
	if p1 != p2 {
		t.Fatalf("round-trip mismatch: %v vs %v", p1, p2)
	}
	// Secondary target must survive serialization too.
	m.Fit([]Outcome{{StrategyID: "ULTRA_SCALPING", Regime: "RANGE", Features: Features{Regime: "RANGE"}, RawScore: 91, Win: true, Target: TargetTP2BeforeSL}})
	b2, _ := m.Bytes()
	m3, _ := LoadModel(b2)
	if _, _, ok := m3.PredictTarget("ULTRA_SCALPING", "RANGE", 91, Features{Regime: "RANGE"}, TargetTP2BeforeSL); !ok {
		t.Fatal("secondary target lost after round-trip")
	}
}

func TestAttachMarksUncalibratedWithoutModel(t *testing.T) {
	sig := &types.Signal{StrategyID: types.StrategyUltraScalping, RawScore: 80, Regime: "RANGE"}
	Attach(sig, nil, Features{Regime: "RANGE"})
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

func TestAttachMultiTarget(t *testing.T) {
	m := NewEmpirical(5)
	m.Fit(sample())
	m.Fit([]Outcome{
		{StrategyID: "ULTRA_SCALPING", Regime: "RANGE", Features: Features{Regime: "RANGE"}, RawScore: 91, Win: true, Target: TargetTP2BeforeSL},
		{StrategyID: "ULTRA_SCALPING", Regime: "RANGE", Features: Features{Regime: "RANGE"}, RawScore: 91, Win: true, Target: TargetDirectionCorrect},
	})
	sig := &types.Signal{StrategyID: types.StrategyUltraScalping, RawScore: 91, Regime: "RANGE"}
	Attach(sig, m, Features{Regime: "RANGE"})
	if sig.ProbabilityModel == "UNCALIBRATED" {
		t.Fatal("expected calibrated primary")
	}
	if len(sig.Calibrated) != 2 {
		t.Fatalf("expected 2 secondary targets, got %d", len(sig.Calibrated))
	}
	seen := map[string]bool{}
	for _, c := range sig.Calibrated {
		seen[c.Target] = true
	}
	if !seen[TargetTP2BeforeSL] || !seen[TargetDirectionCorrect] {
		t.Fatalf("missing secondary target: %+v", sig.Calibrated)
	}
}
