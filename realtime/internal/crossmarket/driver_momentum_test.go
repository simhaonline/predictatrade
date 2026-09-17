package crossmarket

import (
	"testing"
	"time"
)

// P2 (prompt.md): MomPct per driver over N bars.
// MomPct = (latest − first) / first × 100 over the retained history window.
func TestDriverMomPct(t *testing.T) {
	h := NewDriverHistory(24)
	now := time.Now()
	for i := 0; i < 24; i++ {
		h.Push(DriverSnapshot{Name: DriverDXY, RawValue: 100.0 + float64(i), Timestamp: now.Add(time.Duration(i) * time.Hour)})
	}
	// history 100..123 → MomPct = 23/100 × 100 = 23%
	if v := h.MomPct(DriverDXY, 24); v < 22.9 || v > 23.1 {
		t.Fatalf("MomPct = %v, want ~23.0", v)
	}
	// warmup: fewer samples than requested window → 0
	if v := h.MomPct(DriverDXY, 100); v != 0 {
		t.Fatalf("insufficient history → 0, got %v", v)
	}
	// flat prices → 0
	h2 := NewDriverHistory(5)
	for i := 0; i < 5; i++ {
		h2.Push(DriverSnapshot{Name: DriverVIX, RawValue: 20, Timestamp: now.Add(time.Duration(i) * time.Minute)})
	}
	if v := h2.MomPct(DriverVIX, 5); v != 0 {
		t.Fatalf("flat → 0, got %v", v)
	}
}

// P2: correlation matrix across drivers (uses existing pearsonCorrelation).
func TestCorrelationMatrix(t *testing.T) {
	h := NewDriverHistory(10)
	now := time.Now()
	// DXY up 0→23, VIX up 0→46 (perfectly correlated)
	for i := 0; i < 24; i++ {
		h.Push(DriverSnapshot{Name: DriverDXY, RawValue: 100 + float64(i), Timestamp: now.Add(time.Duration(i) * time.Hour)})
		h.Push(DriverSnapshot{Name: DriverVIX, RawValue: 50 + float64(2*i), Timestamp: now.Add(time.Duration(i) * time.Minute)})
	}
	m := h.CorrelationMatrix(24, 10)
	c := m["dxy:vix"]
	if c2, ok := m["vix:dxy"]; ok {
		c = c2
	}
	if c < 0.99 {
		t.Fatalf("perfectly correlated series → ~1.0, got %v (keys: %v)", c, m)
	}
	if _, ok := m["DXY:DXY"]; ok {
		t.Fatal("self-correlation excluded")
	}
}

// P2: manual overlays — operator-set Real10y/FedCtx/CotNetChg biases added
// to the macro score, overriding missing external data (fail-soft by design:
// an operator-set overlay is authoritative for its own channel).
func TestManualOverlays(t *testing.T) {
	o := ManualOverlays{}
	o.SetReal10y(2.5)
	o.SetFedCtx(-1.0)
	o.SetCotNetChg(5.0)
	if o.Real10y != 2.5 || o.FedCtx != -1.0 || o.CotNetChg != 5.0 {
		t.Fatalf("overlay round-trip: real10y=%v fedCtx=%v cotNetChg=%v", o.Real10y, o.FedCtx, o.CotNetChg)
	}
	// Bias contribution: real yields up = gold negative (opportunity cost),
	// FedCtx hawkish = negative, COT net long increasing = positive.
	b := o.MacroBias()
	if b >= 0 {
		t.Fatalf("real10y↑ + fedCtx hawkish should dominate negative, got %v", b)
	}
	// unset overlays contribute 0
	var empty ManualOverlays
	if v := empty.MacroBias(); v != 0 {
		t.Fatalf("unset overlays → 0 bias, got %v", v)
	}
}
