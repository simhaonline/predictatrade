package main

import (
	"fmt"

	"pat-engine/internal/backtest"
	"pat-engine/internal/broker"
	"pat-engine/internal/license"
	"pat-engine/internal/types"
)

func main() {
	states := backtest.BuildSnapshots(backtest.Generate(5000, 42))
	// Dev license allows every strategy so the harness behaves like an unrestricted plan.
	lic, _, _ := license.DevLicense(license.DefaultDevSecret, nil, nil)

	runWith(&broker.BrokerPolicy{Symbol: "XAUUSD", AllowsScalping: true, Digits: 2}, lic, states, "scalping ALLOWED")
	runWith(&broker.BrokerPolicy{Symbol: "XAUUSD", AllowsScalping: false, Digits: 2}, lic, states, "scalping FORBIDDEN (no-scalping broker)")
}

func runWith(pol *broker.BrokerPolicy, lic *license.License, states []*types.MarketState, label string) {
	fmt.Printf("\n=== Broker: %s ===\n", label)
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
