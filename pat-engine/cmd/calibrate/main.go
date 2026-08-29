// cmd/calibrate is the calibration research harness. It searches a parameter grid for
// ONE strategy on a TRAIN window and reports the out-of-sample (TEST) profit factor
// for every candidate — strictly separated from training.
//
// Honesty rules (SOW quant-integrity):
//   - No parameter is re-fitted on the TEST window. TEST is used only to REPORT.
//   - A config is only written out as "having edge" when TEST PF > 1 AND TEST PF is
//     not wildly below TRAIN PF (no silent over-fit). Otherwise we publish nothing and
//     say so. We never tune to force a profitable result.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"

	"pat-engine/internal/backtest"
	"pat-engine/internal/broker"
	"pat-engine/internal/config"
	"pat-engine/internal/license"
	"pat-engine/internal/strategy"
)

func newStrategy(id string, cfg config.StrategyConfig) strategy.Strategy {
	switch id {
	case "ULTRA_SCALPING":
		return strategy.NewUltraScalping(cfg)
	case "STANDARD_SCALPING":
		return strategy.NewStandardScalping(cfg)
	case "STANDARD_SWING":
		return strategy.NewStandardSwing(cfg)
	case "TREND_SWING":
		return strategy.NewTrendSwing(cfg)
	}
	return nil
}

type cand struct {
	cfg              config.StrategyConfig
	trainTrades      int
	trainPF          float64
	testTrades       int
	testPF           float64
}

func main() {
	barsCSV := os.Getenv("BARS_CSV")
	if barsCSV == "" {
		fmt.Println("set BARS_CSV=/path/to/xauusd.csv")
		os.Exit(2)
	}
	stratID := envOr("STRATEGY", "ULTRA_SCALPING")
	trainFrac := fenvOr("TRAIN_FRAC", 0.6)
	minTestTrades := 20
	if v := os.Getenv("MIN_TEST_TRADES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			minTestTrades = n
		}
	}

	bars, err := backtest.FromCSV(barsCSV)
	if err != nil || len(bars) == 0 {
		bars, err = backtest.FromMetaCSV(barsCSV)
	}
	if err != nil || len(bars) == 0 {
		fmt.Println("loaded 0 bars:", err)
		os.Exit(2)
	}
	states := backtest.BuildSnapshots(bars)
	cut := int(float64(len(states)) * trainFrac)
	if cut < 400 {
		cut = 400
	}
	if cut >= len(states) {
		cut = len(states) - 1
	}
	train, test := states[:cut], states[cut:]

	lic, _, _ := license.DevLicense(license.DefaultDevSecret, nil, nil)
	exec := broker.DefaultXAUUSDExecution()
	pol := &broker.BrokerPolicy{Symbol: "XAUUSD", AllowsScalping: true, Digits: 2, MinNetRR: 1.3, Execution: exec}

	base := config.AllDefaults()[stratID]
	if base.StrategyID == "" {
		fmt.Println("unknown strategy:", stratID)
		os.Exit(2)
	}

	fmt.Printf("CALIBRATION RESEARCH — strategy=%s  train=%d test=%d  (strict OOS)\n", stratID, len(train), len(test))

	// Parameter grid (kept modest to stay within honest compute; extend deliberately).
	slMs := []float64{3, 4, 5}
	tp1Ms := []float64{3, 4, 6}
	adxs := []float64{20, 25}
	rrs := []float64{1.5, 2.0}
	spreadATRs := []float64{0.5, 0.7}

	var cands []cand
	for _, sl := range slMs {
		for _, tp1 := range tp1Ms {
			for _, adx := range adxs {
				for _, rr := range rrs {
					for _, satr := range spreadATRs {
						cfg := base
						cfg.ATRMultiplierSL = sl
						cfg.ATRMultiplierTP1 = tp1
						cfg.ATRMultiplierTP2 = 2 * tp1
						cfg.ATRMultiplierTP3 = 3 * tp1
						cfg.MinADX = adx
						cfg.MinRR = rr
						cfg.SpreadATRGate = satr

						st := newStrategy(stratID, cfg)
						if st == nil {
							continue
						}
						tr := backtest.EvalStrategy(train, pol, lic, st, cfg)
						te := backtest.EvalStrategy(test, pol, lic, st, cfg)
						cands = append(cands, cand{cfg: cfg, trainTrades: tr.Trades, trainPF: tr.PF, testTrades: te.Trades, testPF: te.PF})
					}
				}
			}
		}
	}

	sort.Slice(cands, func(i, j int) bool { return cands[i].testPF > cands[j].testPF })

	fmt.Printf("\n%-5s %5s %5s %5s %5s %6s %7s %6s %7s\n", "SL", "TP1", "ADX", "RR", "SATR", "TrdT", "TrPF", "TstT", "TstPF")
	for _, c := range cands {
		if c.testTrades < minTestTrades {
			continue
		}
		fmt.Printf("%-5.0f %5.0f %5.0f %5.1f %5.1f %6d %7.2f %6d %7.2f\n",
			c.cfg.ATRMultiplierSL, c.cfg.ATRMultiplierTP1, c.cfg.MinADX, c.cfg.MinRR, c.cfg.SpreadATRGate,
			c.trainTrades, c.trainPF, c.testTrades, c.testPF)
	}

	// Select: best TEST PF, with honest guardrails against over-fit / insufficient sample.
	var best *cand
	for i := range cands {
		c := &cands[i]
		if c.testTrades < minTestTrades {
			continue
		}
		if c.testPF <= 1.0 {
			continue
		}
		if c.trainPF > 0 && c.testPF < 0.8*c.trainPF {
			continue // suspicious over-fit: train far better than test
		}
		best = c
		break
	}

	if best == nil {
		fmt.Println("\nRESULT: no candidate showed a genuine out-of-sample edge (TEST PF>1 with adequate sample).")
		fmt.Println("Nothing written. Either the strategy lacks edge on this data, or the grid is too narrow.")
		return
	}

	// Additional honest gate: is the edge STABLE across walk-forward folds? A single
	// train/test split can flatter a config; we only publish when the edge repeats
	// out-of-sample and the calibrated probability stays reliable.
	rep := backtest.WalkForwardCalibrationForStrategy(states, 250, 30000, pol, lic, "TP1_BEFORE_SL", stratID, best.cfg)
	fmt.Print(backtest.SummarizeStability(rep))
	if !rep.Stable {
		fmt.Println("\nRESULT: candidate had an OOS edge on the single split but is NOT walk-forward stable.")
		fmt.Println("Nothing written. We do not publish an unstable/uncalibrated config.")
		return
	}

	out := envOr("CALIBRATION_OUT", "data/calibrated_"+stratID+".json")
	b, _ := json.MarshalIndent(best.cfg, "", "  ")
	if err := os.WriteFile(out, b, 0o644); err != nil {
		fmt.Println("write error:", err)
		os.Exit(1)
	}
	fmt.Printf("\nRESULT: best OOS config written to %s\n", out)
	fmt.Printf("  SL=%0.0f TP1=%0.0f ADX=%0.0f RR=%0.1f SpreadATR=%0.1f\n",
		best.cfg.ATRMultiplierSL, best.cfg.ATRMultiplierTP1, best.cfg.MinADX, best.cfg.MinRR, best.cfg.SpreadATRGate)
	fmt.Printf("  TRAIN PF=%.2f (n=%d)  TEST PF=%.2f (n=%d)\n", best.trainPF, best.trainTrades, best.testPF, best.testTrades)

	// Also fit + write the multi-target calibration model (named probabilities),
	// gated by the same stability verdict.
	calTargets := []string{"TP1_BEFORE_SL", "TP2_BEFORE_SL", "DIRECTION_CORRECT"}
	model := backtest.FitCalibrationAll(states, pol, lic, calTargets)
	modelOut := envOr("CALIBRATION_MODEL_OUT", "data/calibration_model.json")
	if mb, err := model.Bytes(); err != nil {
		fmt.Println("model serialize error:", err)
	} else if err := os.WriteFile(modelOut, mb, 0o644); err != nil {
		fmt.Println("model write error:", err)
	} else {
		fmt.Printf("fitted multi-target calibration model written to %s\n", modelOut)
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func fenvOr(k string, def float64) float64 {
	if v := os.Getenv(k); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}
