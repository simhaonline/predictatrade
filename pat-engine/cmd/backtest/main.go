package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"pat-engine/internal/backtest"
	"pat-engine/internal/broker"
	"pat-engine/internal/config"
	"pat-engine/internal/license"
	"pat-engine/internal/signal"
	"pat-engine/internal/strategy"
	"pat-engine/internal/types"
)

// maxDataAge is the staleness threshold for real historical data: if the most
// recent bar is older than this, results may not reflect the current volatility
// regime and the run is explicitly flagged STALE.
const maxDataAge = 30 * 24 * time.Hour

func main() {
	wf := os.Getenv("WALKFORWARD") == "1"

	var bars []backtest.Bar
	var src string
	var synthetic bool

	if f := os.Getenv("BARS_CSV"); f != "" {
		var err error
		bars, err = loadBars(f)
		if err != nil {
			panic(err)
		}
		if len(bars) == 0 {
			panic("loaded 0 bars from " + f)
		}
		src = "REAL " + f
		synthetic = false
	} else if os.Getenv("ALLOW_SYNTHETIC") == "1" {
		// Synthetic data is NON-REPRESENTATIVE: random walks have no real edge and
		// must never be reported as a genuine performance result.
		bars = backtest.Generate(5000, 42)
		src = "SYNTHETIC (NON-REPRESENTATIVE — do not report as real)"
		synthetic = true
	} else {
		fmt.Println("ERROR: no market data provided.")
		fmt.Println("  Set BARS_CSV=/path/to/xauusd_m1.csv for REAL data, or")
		fmt.Println("  set ALLOW_SYNTHETIC=1 to run on synthetic data (NON-REPRESENTATIVE).")
		os.Exit(2)
	}

	fmt.Print(describeBars(bars, src, synthetic))

	states := backtest.BuildSnapshots(bars)
	if len(states) == 0 {
		panic("BuildSnapshots produced 0 states (check warm-up / data)")
	}

	// Dev license allows every strategy so the harness behaves like an unrestricted plan.
	lic, _, _ := license.DevLicense(license.DefaultDevSecret, nil, nil)
	exec := broker.DefaultXAUUSDExecution()

	pol := &broker.BrokerPolicy{Symbol: "XAUUSD", AllowsScalping: true, Digits: 2, MinNetRR: 1.3, Execution: exec}

	if wf {
		foldBars := 30000
		if v := os.Getenv("WF_FOLD"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				foldBars = n
			}
		}
		fmt.Printf("\n=== WALK-FORWARD / OOS REPORT (%s) foldBars=%d ===\n", src, foldBars)
		folds := backtest.WalkForward(states, 250, foldBars, pol, lic)
		fmt.Print(backtest.SummarizeFolds(folds))
		return
	}

	runWith(pol, lic, states, src+" | scalping ALLOWED")
	runWith(&broker.BrokerPolicy{Symbol: "XAUUSD", AllowsScalping: false, Digits: 2, MinNetRR: 1.3, Execution: exec}, lic, states, src+" | scalping FORBIDDEN (no-scalping broker)")
}

// describeBars prints data provenance and a staleness check. Showing the date
// range, bar count and spread statistics up front is what keeps a backtest honest:
// you can see immediately whether the numbers are real and fresh.
func describeBars(bars []backtest.Bar, src string, synthetic bool) string {
	if len(bars) == 0 {
		return ""
	}
	first, last := bars[0].Time, bars[len(bars)-1].Time
	minSp, maxSp, sumSp := bars[0].Spread, bars[0].Spread, 0.0
	hasSpread := false
	for _, b := range bars {
		if b.Spread > 0 {
			hasSpread = true
		}
		if b.Spread < minSp {
			minSp = b.Spread
		}
		if b.Spread > maxSp {
			maxSp = b.Spread
		}
		sumSp += b.Spread
	}
	out := "\n=== DATA PROVENANCE ===\n"
	out += fmt.Sprintf("source        : %s\n", src)
	out += fmt.Sprintf("bars          : %d\n", len(bars))
	if !synthetic {
		out += fmt.Sprintf("range         : %s -> %s (UTC)\n",
			time.Unix(first, 0).UTC().Format("2006-01-02 15:04"),
			time.Unix(last, 0).UTC().Format("2006-01-02 15:04"))
		age := time.Since(time.Unix(last, 0).UTC())
		if age > maxDataAge {
			out += fmt.Sprintf("STALENESS     : ⚠ STALE — last bar is %s old (> %s). Regime may have changed.\n", age.Round(time.Hour), maxDataAge)
		} else {
			out += fmt.Sprintf("freshness     : ok (last bar %s old)\n", age.Round(time.Hour))
		}
	}
	if hasSpread {
		out += fmt.Sprintf("spread (pts)  : min %.2f / avg %.2f / max %.2f\n", minSp, sumSp/float64(len(bars)), maxSp)
	} else {
		out += "spread        : NOT PROVIDED (backtest will assume default; real spread recommended)\n"
	}
	out += "======================\n"
	return out
}

// loadBars tries the comma-delimited format first, then MetaTrader (semicolon+date).
func loadBars(f string) ([]backtest.Bar, error) {
	bars, err := backtest.FromCSV(f)
	if err == nil && len(bars) > 0 {
		return bars, nil
	}
	return backtest.FromMetaCSV(f)
}

func runWith(pol *broker.BrokerPolicy, lic *license.License, states []*types.MarketState, label string) {
	debug := os.Getenv("DEBUG_REASONS") == "1"
	reasonTally := map[string]map[string]int{} // strategy -> reason -> count

	fmt.Printf("\n=== %s ===\n", label)
	fmt.Printf("%-18s %6s %7s %7s %9s %9s\n", "STRATEGY", "TRD", "WIN%", "PF", "GROSSWIN", "GROSSLOSS")
	for _, r := range backtest.RunAll(states, pol, lic) {
		if r.ExcludedBy != "" {
			fmt.Printf("%-18s EXCLUDED (%s)\n", r.Strategy, r.ExcludedBy)
			continue
		}
		fmt.Printf("%-18s %6d %6.1f%% %7.2f %9.2f %9.2f\n",
			r.Strategy, r.Trades, r.WinRate*100, r.PF, r.GrossWin, r.GrossLoss)
	}
	if debug {
		// Re-run tallying the rejection reasons so we can see WHY trades are scarce.
		cfgs := config.AllDefaults()
		for _, st := range strategy.All() {
			cfg := cfgs[string(st.ID())]
			tally := map[string]int{}
			for _, s := range states {
				d := signal.Decide(s, st, cfg, pol)
				if !d.Signal.Executable {
					for _, rs := range d.Reasons {
						tally[rs]++
					}
				}
			}
			reasonTally[string(st.ID())] = tally
		}
		fmt.Printf("\n--- REJECTION REASONS (%s) [DEBUG_REASONS=1] ---\n", label)
		for strat, tally := range reasonTally {
			fmt.Printf("%s:\n", strat)
			for reason, n := range tally {
				fmt.Printf("    %-28s %d\n", reason, n)
			}
		}
	}
}
