package backtest

import (
	"fmt"
	"sort"

	"pat-engine/internal/broker"
	"pat-engine/internal/calibrate"
	"pat-engine/internal/config"
	"pat-engine/internal/license"
	"pat-engine/internal/signal"
	"pat-engine/internal/strategy"
	"pat-engine/internal/types"
)

// LabeledOutcomes runs the full live pipeline over the snapshot series for a NAMED
// target and returns one Outcome per EXECUTABLE signal, labeled by whether the honest
// simulator realized that target before the SL. This is the training/validation label
// set for the calibrator — it uses the exact same Decide + Simulate code as live, so
// calibration transfers to production with no research drift.
//
// Target semantics (all simulator-derived, never raw score):
//   - TP1_BEFORE_SL:      TP1 (1R partial) hit before SL. Once TP1 hits the remainder
//     is moved to breakeven, so the trade is risk-free.
//   - TP2_BEFORE_SL:      TP2 (2R) hit before SL.
//   - DIRECTION_CORRECT:  price moved >= 0.5R in the signal direction before SL.
func LabeledOutcomes(states []*types.MarketState, pol *broker.BrokerPolicy, lic *license.License, target string) []calibrate.Outcome {
	cfgs := config.AllDefaults()
	strats := strategy.All()
	exec := broker.ExecutionProfile{}
	if pol != nil {
		exec = pol.Execution
	}
	if target == "" {
		target = calibrate.TargetTP1BeforeSL
	}

	var outs []calibrate.Outcome
	for _, st := range strats {
		cfg := cfgs[string(st.ID())]
		maxBars := cfg.BacktestMaxBars
		if maxBars <= 0 {
			maxBars = 50
		}
		for i, s := range states {
			if s == nil {
				continue
			}
			d := signal.Decide(s, st, cfg, pol)
			if !d.Signal.Executable {
				continue
			}
			sim := Simulate(states, i, d.Signal.Direction, d.Signal.EntryPrice, d.Signal.StopLoss,
				d.Signal.TP1, d.Signal.TP2, d.Signal.TP3, maxBars, exec)
			var win bool
			switch target {
			case calibrate.TargetTP2BeforeSL:
				win = sim.TP2Hit
			case calibrate.TargetDirectionCorrect:
				win = sim.DirCorrect
			default: // TP1_BEFORE_SL
				win = sim.TP1Hit
			}
			outs = append(outs, calibrate.Outcome{
				StrategyID: string(st.ID()),
				Regime:     s.Regime,
				RawScore:   d.Signal.RawScore,
				Win:        win,
				Target:     target,
				Features:   calibrate.FeaturesFromState(s),
			})
		}
	}
	return outs
}

// FitCalibration fits an empirical calibrator on a (training) window of states for a
// named target.
func FitCalibration(states []*types.MarketState, pol *broker.BrokerPolicy, lic *license.License, target string) *calibrate.EmpiricalModel {
	m := calibrate.NewEmpirical(10)
	m.Fit(LabeledOutcomes(states, pol, lic, target))
	return m
}

// FitCalibrationAll fits a single empirical model that carries every named target in
// targets. Because EmpiricalModel.Fit routes outcomes by target, one combined model
// serves all subscriber-facing probabilities.
func FitCalibrationAll(states []*types.MarketState, pol *broker.BrokerPolicy, lic *license.License, targets []string) *calibrate.EmpiricalModel {
	m := calibrate.NewEmpirical(10)
	for _, t := range targets {
		m.Fit(LabeledOutcomes(states, pol, lic, t))
	}
	return m
}

// reliabilityCell accumulates prediction/realization statistics for one probability band.
type reliabilityCell struct {
	predSum float64
	n       int
	wins    int
}

// calibrationReliability measures calibration QUALITY on a held-out window: it buckets
// the predicted probability and checks the realized win rate inside each bucket is
// close to the prediction (reliability). Uncalibrated setups (ok=false) are excluded.
// It returns the mean absolute (realized - predicted) gap, the number of scored
// setups, and a plain-text table.
func calibrationReliability(m *calibrate.EmpiricalModel, states []*types.MarketState, pol *broker.BrokerPolicy, lic *license.License, target string) (gap float64, n int, text string) {
	if target == "" {
		target = calibrate.TargetTP1BeforeSL
	}
	outs := LabeledOutcomes(states, pol, lic, target)
	const nb = 5
	cells := make([]reliabilityCell, nb)
	skipped := 0
	for _, o := range outs {
		p, model, ok := m.PredictTarget(o.StrategyID, o.Regime, o.RawScore, o.Features, target)
		if !ok || model == "UNCALIBRATED" {
			skipped++
			continue
		}
		bi := int(p * float64(nb))
		if bi >= nb {
			bi = nb - 1
		}
		if bi < 0 {
			bi = 0
		}
		cells[bi].predSum += p
		cells[bi].n++
		if o.Win {
			cells[bi].wins++
		}
	}
	text = fmt.Sprintf("\n=== CALIBRATION RELIABILITY (%s, held-out) ===\n", target)
	text += fmt.Sprintf("%-12s %8s %8s %10s %12s\n", "PROB BUCKET", "N", "PRED", "REALIZED", "GAP")
	sumGap := 0.0
	sumN := 0
	for i := 0; i < nb; i++ {
		c := cells[i]
		if c.n == 0 {
			continue
		}
		pred := c.predSum / float64(c.n)
		realized := float64(c.wins) / float64(c.n)
		g := realized - pred
		sumGap += absF(g)
		sumN += c.n
		text += fmt.Sprintf("[%.1f-%.1f) %8d %8.2f %10.2f %12.2f\n",
			float64(i)/float64(nb), float64(i+1)/float64(nb), c.n, pred, realized, g)
	}
	if sumN > 0 {
		gap = sumGap / float64(sumN)
	}
	text += fmt.Sprintf("mean|gap|=%.3f  scored=%d  excluded(UNCALIBRATED)=%d  total=%d\n",
		gap, sumN, skipped, len(outs))
	return gap, sumN, text
}

// ValidateCalibration prints the reliability report for a fitted model on a held-out
// window (see calibrationReliability).
func ValidateCalibration(m *calibrate.EmpiricalModel, states []*types.MarketState, pol *broker.BrokerPolicy, lic *license.License, target string) string {
	_, _, text := calibrationReliability(m, states, pol, lic, target)
	return text
}

// FoldCalResult holds the out-of-sample calibration/edge metrics for one walk-forward fold.
type FoldCalResult struct {
	Index          int
	From, To       int
	TestTrades     int
	TestPF         float64
	ReliabilityGap float64
	Calibrated     bool
}

// StabilityReport summarizes walk-forward calibration stability across folds.
type StabilityReport struct {
	Target        string
	Folds         []FoldCalResult
	AdequateFolds int
	MedianTestPF  float64
	Stable        bool
	Notes         string
}

// WalkForwardCalibration fits the empirical model on an EXPANDING training window and
// validates it on each subsequent, strictly held-out fold. This is the honest
// stability check the SOW requires: it asks whether the calibrated probability stays
// reliable OOS across regimes AND whether a genuine edge (OOS PF>1) repeats. A model
// is only blessed STABLE when at least 3 folds carry adequate calibrated sample, the
// median OOS PF exceeds 1, and the calibration reliability gap stays within tolerance
// on every fold. Otherwise it is reported NOT_STABLE / INSUFFICIENT_DATA and must not
// be published as a production calibrator.
func WalkForwardCalibration(states []*types.MarketState, warmup, foldBars int, pol *broker.BrokerPolicy, lic *license.License, target string) StabilityReport {
	if foldBars <= 0 {
		foldBars = 40000
	}
	if warmup < 200 {
		warmup = 200
	}
	if target == "" {
		target = calibrate.TargetTP1BeforeSL
	}
	rep := StabilityReport{Target: target}
	idx := 0
	for start := warmup; start+foldBars < len(states); start += foldBars {
		end := start + foldBars
		train := states[:start] // expanding window: never touches the test fold
		test := states[start:end]

		fold := FoldCalResult{Index: idx, From: start, To: end}

		// Edge (PF) on the strictly held-out fold, aggregate of all allowed strategies.
		var trades int
		var gw, gl float64
		for _, r := range RunAll(test, pol, lic) {
			if r.ExcludedBy != "" {
				continue
			}
			trades += r.Trades
			gw += r.GrossWin
			gl += r.GrossLoss
		}
		fold.TestTrades = trades
		if gl > 0 {
			fold.TestPF = gw / gl
		}

		// Calibration reliability on the fold, using a model fit ONLY on train.
		if len(train) >= warmup {
			m := FitCalibration(train, pol, lic, target)
			gap, n, _ := calibrationReliability(m, test, pol, lic, target)
			fold.ReliabilityGap = gap
			fold.Calibrated = n > 0
		}

		rep.Folds = append(rep.Folds, fold)
		idx++
	}

	// Verdict.
	pfs := []float64{}
	for _, f := range rep.Folds {
		if f.TestTrades >= 20 && f.Calibrated {
			rep.AdequateFolds++
			pfs = append(pfs, f.TestPF)
		}
	}
	if rep.AdequateFolds < 3 {
		rep.Stable = false
		rep.Notes = "INSUFFICIENT_DATA: fewer than 3 folds with adequate trades + calibration sample"
		return rep
	}
	sort.Float64s(pfs)
	rep.MedianTestPF = pfs[len(pfs)/2]
	maxGap := 0.0
	for _, f := range rep.Folds {
		if f.Calibrated && f.ReliabilityGap > maxGap {
			maxGap = f.ReliabilityGap
		}
	}
	if rep.MedianTestPF > 1.0 && maxGap <= 0.12 {
		rep.Stable = true
		rep.Notes = "STABLE: median OOS PF>1 and calibration reliability gap within tolerance across folds"
	} else {
		rep.Stable = false
		rep.Notes = fmt.Sprintf("NOT_STABLE: median OOS PF=%.2f, max reliability gap=%.2f (need PF>1 and gap<=0.12)",
			rep.MedianTestPF, maxGap)
	}
	return rep
}

// WalkForwardCalibrationForStrategy is like WalkForwardCalibration but restricts the
// edge (PF) measurement to a single strategy/config under investigation. This is what
// the research harness uses to decide whether a candidate's edge is stable enough to
// publish, rather than being flattered by unrelated strategies in the aggregate.
func WalkForwardCalibrationForStrategy(states []*types.MarketState, warmup, foldBars int, pol *broker.BrokerPolicy, lic *license.License, target, stratID string, cfg config.StrategyConfig) StabilityReport {
	if foldBars <= 0 {
		foldBars = 40000
	}
	if warmup < 200 {
		warmup = 200
	}
	if target == "" {
		target = calibrate.TargetTP1BeforeSL
	}
	rep := StabilityReport{Target: target}
	var st strategy.Strategy
	for _, s := range strategy.All() {
		if string(s.ID()) == stratID {
			st = s
			break
		}
	}
	idx := 0
	for start := warmup; start+foldBars < len(states); start += foldBars {
		end := start + foldBars
		train := states[:start]
		test := states[start:end]
		fold := FoldCalResult{Index: idx, From: start, To: end}
		if st != nil {
			r := EvalStrategy(test, pol, lic, st, cfg)
			fold.TestTrades = r.Trades
			fold.TestPF = r.PF
		}
		if len(train) >= warmup {
			m := FitCalibration(train, pol, lic, target)
			gap, n, _ := calibrationReliability(m, test, pol, lic, target)
			fold.ReliabilityGap = gap
			fold.Calibrated = n > 0
		}
		rep.Folds = append(rep.Folds, fold)
		idx++
	}

	pfs := []float64{}
	for _, f := range rep.Folds {
		if f.TestTrades >= 20 && f.Calibrated {
			rep.AdequateFolds++
			pfs = append(pfs, f.TestPF)
		}
	}
	if rep.AdequateFolds < 3 {
		rep.Stable = false
		rep.Notes = "INSUFFICIENT_DATA: fewer than 3 folds with adequate trades + calibration sample"
		return rep
	}
	sort.Float64s(pfs)
	rep.MedianTestPF = pfs[len(pfs)/2]
	maxGap := 0.0
	for _, f := range rep.Folds {
		if f.Calibrated && f.ReliabilityGap > maxGap {
			maxGap = f.ReliabilityGap
		}
	}
	if rep.MedianTestPF > 1.0 && maxGap <= 0.12 {
		rep.Stable = true
		rep.Notes = "STABLE: median OOS PF>1 and calibration reliability gap within tolerance across folds"
	} else {
		rep.Stable = false
		rep.Notes = fmt.Sprintf("NOT_STABLE: median OOS PF=%.2f, max reliability gap=%.2f (need PF>1 and gap<=0.12)",
			rep.MedianTestPF, maxGap)
	}
	return rep
}

// SummarizeStability renders a StabilityReport as plain text (no smoothing).
func SummarizeStability(rep StabilityReport) string {
	out := fmt.Sprintf("\n=== CALIBRATION STABILITY WALK-FORWARD (%s) ===\n", rep.Target)
	out += fmt.Sprintf("%-5s %10s %8s %8s %10s %12s\n", "FOLD", "BARS", "TRD", "PF", "GAP", "CAL?")
	for _, f := range rep.Folds {
		out += fmt.Sprintf("%-5d %6d-%-5d %8d %8.2f %10.3f %12v\n",
			f.Index, f.From, f.To, f.TestTrades, f.TestPF, f.ReliabilityGap, f.Calibrated)
	}
	out += fmt.Sprintf("\nadequate folds : %d\n", rep.AdequateFolds)
	out += fmt.Sprintf("median OOS PF  : %.3f\n", rep.MedianTestPF)
	out += fmt.Sprintf("VERDICT        : %s\n", verdict(rep.Stable))
	out += fmt.Sprintf("NOTES          : %s\n", rep.Notes)
	return out
}

func verdict(stable bool) string {
	if stable {
		return "STABLE"
	}
	return "NOT_STABLE"
}

func absF(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
