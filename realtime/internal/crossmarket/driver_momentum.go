// driver_momentum.go — P2 (prompt.md): crossmarket driver momentum, the
// correlation matrix, and operator manual overlays.
//
// NON-OVERLAP: the existing crossmarket Engine (engine.go) holds one
// DriverSnapshot per driver and computes confluence/weights/anti-double-
// counting. This file adds a SEPARATE bounded price history per driver
// (DriverHistory) so per-driver momentum (MomPct) and the correlation matrix
// become computable — neither existed. Engine.UpdateDriver is untouched;
// callers push snapshots to both structures (or a future adapter does).
//
// Formulas:
//   MomPct = (latest − first) / first × 100 over the requested window;
//            warmup (fewer samples) → 0; flat prices → 0.
//   CorrelationMatrix: pairwise Pearson (existing pearsonCorrelation) over
//   the aligned last-n values; diagonal excluded; NaN → skipped.
//
// Manual overlays (Real10y / FedCtx / CotNetChg): operator-set macro biases.
// Directional model (documented):
//   Real10y up      → gold opportunity cost up → negative for gold (−)
//   FedCtx hawkish  → negative for gold (−)
//   COT net change  → position-flow confirmation, positive when positive (+)
// MacroBias = −Real10y×0.5 − FedCtx×0.3 + CotNetChg×0.1 (weights documented).
package crossmarket

import "sync"

// DriverHistory keeps a bounded time-series of driver values for momentum
// and correlation analysis. Thread-safe.
type DriverHistory struct {
	mu       sync.RWMutex
	max      int
	byDriver map[DriverName][]DriverSnapshot
}

// NewDriverHistory creates a history store retaining max samples per driver.
func NewDriverHistory(maxPerDriver int) *DriverHistory {
	return &DriverHistory{max: maxPerDriver, byDriver: make(map[DriverName][]DriverSnapshot)}
}

// Push appends a snapshot, trimming to the retention bound.
func (h *DriverHistory) Push(s DriverSnapshot) {
	h.mu.Lock()
	defer h.mu.Unlock()
	series := append(h.byDriver[s.Name], s)
	if h.max > 0 && len(series) > h.max {
		series = series[len(series)-h.max:]
	}
	h.byDriver[s.Name] = series
}

// MomPct returns the percentage momentum of a driver over the last n samples.
// Warmup (fewer than n samples or flat first value) → 0.
func (h *DriverHistory) MomPct(name DriverName, n int) float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	series := h.byDriver[name]
	if len(series) < n || n < 2 {
		return 0
	}
	w := series[len(series)-n:]
	first := w[0].RawValue
	if first == 0 {
		return 0
	}
	return (w[len(w)-1].RawValue - first) / first * 100
}

// CorrelationMatrix computes pairwise Pearson correlations across all
// drivers with sufficient aligned history (minSamples). Self-pairs excluded.
func (h *DriverHistory) CorrelationMatrix(n int, minSamples int) map[string]float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := map[string]float64{}
	names := make([]DriverName, 0, len(h.byDriver))
	for name, series := range h.byDriver {
		if len(series) >= minSamples {
			names = append(names, name)
		}
	}
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			xs := lastN(h.byDriver[names[i]], n)
			ys := lastN(h.byDriver[names[j]], n)
			if len(xs) != len(ys) {
				continue
			}
			xf := floatsOf(xs)
			yf := floatsOf(ys)
			c := pearsonCorrelation(xf, yf)
			if isNaN(c) {
				continue
			}
			out[string(names[i])+":"+string(names[j])] = c
		}
	}
	return out
}

func lastN(s []DriverSnapshot, n int) []DriverSnapshot {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

func floatsOf(s []DriverSnapshot) []float64 {
	out := make([]float64, len(s))
	for i, v := range s {
		out[i] = v.RawValue
	}
	return out
}

func isNaN(f float64) bool { return f != f }

// ─── Manual overlays ────────────────────────────────────────────────────

// ManualOverlays: operator-set macro biases. Thread-safe.
type ManualOverlays struct {
	mu        sync.RWMutex
	Real10y   float64 // US 10y real yield % (FRED DFII10) — negative for gold
	FedCtx    float64 // Fed context −1 dovish … +1 hawkish — negative for gold
	CotNetChg float64 // weekly COT net-position change — confirmation (±)
}

// SetReal10y sets the 10y real yield overlay.
func (o *ManualOverlays) SetReal10y(v float64) { o.mu.Lock(); o.Real10y = v; o.mu.Unlock() }

// SetFedCtx sets the Fed context overlay (−1 dovish … +1 hawkish).
func (o *ManualOverlays) SetFedCtx(v float64) { o.mu.Lock(); o.FedCtx = v; o.mu.Unlock() }

// SetCotNetChg sets the COT net-position weekly change overlay.
func (o *ManualOverlays) SetCotNetChg(v float64) { o.mu.Lock(); o.CotNetChg = v; o.mu.Unlock() }

// MacroBias returns the overlay-driven macro bias for gold (documented
// weights: real yields −0.5×, Fed context −0.3×, COT +0.1×; clamped ±100).
// Zero-value struct (nothing set) contributes 0.
func (o *ManualOverlays) MacroBias() float64 {
	o.mu.RLock()
	defer o.mu.RUnlock()
	b := -o.Real10y*0.5 - o.FedCtx*0.3 + o.CotNetChg*0.1
	if b > 100 {
		b = 100
	}
	if b < -100 {
		b = -100
	}
	return b
}
// ─── Engine-level P2 accessors (engine.go unchanged — these live here to
// avoid touching the canonical confluence path) ─────────────────────────

// Bias returns the combined macro bias: driver confluence score is already
// computed by Engine.computeConfluence; this adds the operator-overlay bias
// and driver momentum agreement. Pure read.
//   Bias = clamp(overlayBias + Σ sign(MomPct_i)·min(|MomPct_i|,10)·0.1, −100, 100)
func (e *Engine) Bias() float64 {
	overlay := e.manual.MacroBias()
	e.mu.RLock()
	defer e.mu.RUnlock()
	mom := 0.0
	for name := range e.drivers {
		p := e.history.MomPct(name, e.cfg.DriverMomentumBars)
		if p == 0 {
			continue
		}
		mag := p
		if mag < 0 {
			mag = -mag
		}
		if mag > 10 {
			mag = 10
		}
		if p > 0 {
			mom += mag * 0.1
		} else {
			mom -= mag * 0.1
		}
	}
	total := overlay + mom
	if total > 100 {
		total = 100
	}
	if total < -100 {
		total = -100
	}
	return total
}

// BiasX6 returns the reference composite input: Bias() × 6 (prompt.md P2 —
// the macro family score consumes the ×6-scaled bias).
func (e *Engine) BiasX6() float64 { return e.Bias() * 6 }

// SetOverlays wires operator manual overlays (Real10y/FedCtx/CotNetChg).
func (e *Engine) SetOverlays(real10y, fedCtx, cotNetChg float64) {
	e.manual.SetReal10y(real10y)
	e.manual.SetFedCtx(fedCtx)
	e.manual.SetCotNetChg(cotNetChg)
}

// DriverMomPct returns one driver's momentum percentage over n samples.
func (e *Engine) DriverMomPct(name DriverName, n int) float64 {
	return e.history.MomPct(name, n)
}

// CorrelationMatrix exposes the pairwise driver correlation diagnostics.
func (e *Engine) CorrelationMatrix(n, minSamples int) map[string]float64 {
	return e.history.CorrelationMatrix(n, minSamples)
}
