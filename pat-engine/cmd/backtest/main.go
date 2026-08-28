package main

import (
	"fmt"
	"os"

	"pat-engine/internal/backtest"
	"pat-engine/internal/broker"
	"pat-engine/internal/license"
	"pat-engine/internal/types"
)

func main() {
	var states []*types.MarketState
	var src string

	if f := os.Getenv("BARS_CSV"); f != "" {
		bars, err := loadBars(f)
		if err != nil {
			panic(err)
		}
		// Cap to a manageable window for repeatable runs; take the most recent bars.
		if len(bars) > 150000 {
			bars = bars[len(bars)-150000:]
		}
		states = backtest.BuildSnapshots(bars)
		src = "REAL " + f
	} else {
		states = backtest.BuildSnapshots(backtest.Generate(5000, 42))
		src = "SYNTHETIC (deterministic)"
	}

	// Dev license allows every strategy so the harness behaves like an unrestricted plan.
	lic, _, _ := license.DevLicense(license.DefaultDevSecret, nil, nil)
	exec := broker.DefaultXAUUSDExecution()

	runWith(&broker.BrokerPolicy{Symbol: "XAUUSD", AllowsScalping: true, Digits: 2, MinNetRR: 1.3, Execution: exec}, lic, states, src+" | scalping ALLOWED")
	runWith(&broker.BrokerPolicy{Symbol: "XAUUSD", AllowsScalping: false, Digits: 2, MinNetRR: 1.3, Execution: exec}, lic, states, src+" | scalping FORBIDDEN (no-scalping broker)")
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
}
