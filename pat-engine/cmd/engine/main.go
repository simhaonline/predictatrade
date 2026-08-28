package main

import (
	"fmt"

	"pat-engine/internal/broker"
	"pat-engine/internal/config"
	"pat-engine/internal/marketdata"
	"pat-engine/internal/signal"
	"pat-engine/internal/strategy"
)

const samplePath = "data/sample.csv"

func runAll(pol *broker.BrokerPolicy) {
	cfgs := config.AllDefaults()
	strats := strategy.All()
	for _, st := range strats {
		cfg := cfgs[string(st.ID())]
		prov, err := marketdata.NewCSVReplay(samplePath)
		if err != nil {
			panic(err)
		}
		for {
			state, ok, err := prov.Next()
			if err != nil {
				panic(err)
			}
			if !ok {
				break
			}
			d := signal.Decide(state, st, cfg, pol)
			if d.Signal.Executable {
				rr := 0.0
				if d.Signal.Direction == "BUY" {
					rr = (d.Signal.TP1 - d.Signal.EntryPrice) / (d.Signal.EntryPrice - d.Signal.StopLoss)
				} else {
					rr = (d.Signal.EntryPrice - d.Signal.TP1) / (d.Signal.StopLoss - d.Signal.EntryPrice)
				}
				fmt.Printf("EXECUTABLE %-16s %-5s entry=%.2f sl=%.2f tp1=%.2f R:R=%.2f score=%.1f\n",
					d.Signal.StrategyID, d.Signal.Direction,
					d.Signal.EntryPrice, d.Signal.StopLoss, d.Signal.TP1, rr, d.Signal.RawScore)
			} else {
				fmt.Printf("BLOCKED   %-16s %-5s reasons=%v\n",
					d.Signal.StrategyID, d.Signal.Direction, d.Reasons)
			}
		}
		prov.Close()
	}
}

func main() {
	fmt.Println("=== Broker policy: scalping ALLOWED ===")
	runAll(&broker.BrokerPolicy{Symbol: "XAUUSD", AllowsScalping: true, Digits: 2})

	fmt.Println("\n=== Broker policy: scalping FORBIDDEN (no-scalping broker) ===")
	runAll(&broker.BrokerPolicy{Symbol: "XAUUSD", AllowsScalping: false, Digits: 2})
}
